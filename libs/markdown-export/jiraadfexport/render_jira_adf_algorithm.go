// Package jiraadfexport renders a document as Atlassian Document Format, the JSON that Jira Cloud and
// Confluence Cloud accept for rich text.
package jiraadfexport

import (
	"encoding/json"
	"strconv"

	"lore-master/libs/markdown-export/exportmodel"
)

// node is one ADF node. Fields a node does not use stay out of the JSON.
type node struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []node         `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []mark         `json:"marks,omitempty"`
}

type mark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

type document struct {
	Version int    `json:"version"`
	Type    string `json:"type"`
	Content []node `json:"content"`
}

// converter numbers the task items it writes, which ADF needs for each to carry its own id.
type converter struct {
	tasks int
}

// Render writes the document as indented ADF JSON.
func Render(source exportmodel.Document) (string, error) {
	var walk converter
	body := document{Version: 1, Type: "doc", Content: walk.blocks(source.Blocks)}
	if body.Content == nil {
		body.Content = []node{}
	}
	encoded, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return "", err
	}

	return string(encoded) + "\n", nil
}

func (c *converter) blocks(list []exportmodel.Block) []node {
	var nodes []node
	for _, current := range list {
		nodes = append(nodes, c.block(current)...)
	}

	return nodes
}

func (c *converter) block(current exportmodel.Block) []node {
	switch current.Kind {
	case exportmodel.HeadingBlock:
		return []node{{Type: "heading", Attrs: map[string]any{"level": min(max(current.Level, 1), 6)}, Content: c.inlines(current.Inlines, nil)}}
	case exportmodel.ParagraphBlock:
		return []node{{Type: "paragraph", Content: c.inlines(current.Inlines, nil)}}
	case exportmodel.ListBlock:
		return []node{c.list(current)}
	case exportmodel.QuoteBlock:
		return []node{{Type: "blockquote", Content: c.quoted(current.Children)}}
	case exportmodel.CodeBlock:
		code := node{Type: "codeBlock"}
		if current.Lang != "" {
			code.Attrs = map[string]any{"language": current.Lang}
		}
		if current.Text != "" {
			code.Content = []node{{Type: "text", Text: current.Text}}
		}

		return []node{code}
	case exportmodel.TableBlock:
		return []node{c.table(current)}
	case exportmodel.RuleBlock:
		return []node{{Type: "rule"}}
	case exportmodel.HTMLBlock:
		return []node{{Type: "codeBlock", Content: []node{{Type: "text", Text: current.Text}}}}
	default:
		return nil
	}
}

// quoted keeps what a blockquote may hold; a nested quote or a heading becomes a paragraph.
func (c *converter) quoted(list []exportmodel.Block) []node {
	var nodes []node
	for _, current := range list {
		switch current.Kind {
		case exportmodel.QuoteBlock:
			nodes = append(nodes, c.quoted(current.Children)...)
		case exportmodel.HeadingBlock:
			nodes = append(nodes, node{Type: "paragraph", Content: c.inlines(current.Inlines, []mark{{Type: "strong"}})})
		default:
			nodes = append(nodes, c.block(current)...)
		}
	}

	return nodes
}

func (c *converter) list(current exportmodel.Block) node {
	if isTaskList(current) {
		c.tasks++
		tasks := node{Type: "taskList", Attrs: map[string]any{"localId": "tasks-" + strconv.Itoa(c.tasks)}}
		for _, item := range current.Items {
			c.tasks++
			state := "TODO"
			if *item.Checked {
				state = "DONE"
			}
			tasks.Content = append(tasks.Content, node{
				Type:    "taskItem",
				Attrs:   map[string]any{"localId": "task-" + strconv.Itoa(c.tasks), "state": state},
				Content: c.itemText(item),
			})
		}

		return tasks
	}

	kind := "bulletList"
	var attrs map[string]any
	if current.Ordered {
		kind = "orderedList"
		if current.Start > 1 {
			attrs = map[string]any{"order": current.Start}
		}
	}
	list := node{Type: kind, Attrs: attrs}
	for _, item := range current.Items {
		list.Content = append(list.Content, node{Type: "listItem", Content: c.itemBlocks(item)})
	}

	return list
}

func isTaskList(current exportmodel.Block) bool {
	if len(current.Items) == 0 {
		return false
	}
	for _, item := range current.Items {
		if item.Checked == nil {
			return false
		}
	}

	return true
}

// itemText is the inline content of a task: its first paragraph.
func (c *converter) itemText(item exportmodel.Item) []node {
	for _, inner := range item.Blocks {
		if inner.Kind == exportmodel.ParagraphBlock {
			return c.inlines(inner.Inlines, nil)
		}
	}

	return nil
}

// itemBlocks keeps what a list item may hold: paragraphs, lists and code. Anything else is
// written as a paragraph of its text, so the item is still valid ADF.
func (c *converter) itemBlocks(item exportmodel.Item) []node {
	var nodes []node
	for _, inner := range item.Blocks {
		switch inner.Kind {
		case exportmodel.ParagraphBlock, exportmodel.ListBlock, exportmodel.CodeBlock:
			nodes = append(nodes, c.block(inner)...)
		case exportmodel.HeadingBlock:
			nodes = append(nodes, node{Type: "paragraph", Content: c.inlines(inner.Inlines, []mark{{Type: "strong"}})})
		default:
			for _, converted := range c.block(inner) {
				nodes = append(nodes, node{Type: "paragraph", Content: textOf(converted)})
			}
		}
	}
	if len(nodes) == 0 || nodes[0].Type != "paragraph" {
		nodes = append([]node{{Type: "paragraph"}}, nodes...)
	}

	return nodes
}

// textOf gathers the text a converted node holds.
func textOf(current node) []node {
	var text []node
	if current.Type == "text" {
		return []node{current}
	}
	for _, child := range current.Content {
		text = append(text, textOf(child)...)
	}

	return text
}

func (c *converter) table(current exportmodel.Block) node {
	table := node{Type: "table", Attrs: map[string]any{"isNumberColumnEnabled": false, "layout": "default"}}
	if len(current.Header) > 0 {
		table.Content = append(table.Content, c.row("tableHeader", current.Header))
	}
	for _, row := range current.Rows {
		table.Content = append(table.Content, c.row("tableCell", row))
	}

	return table
}

func (c *converter) row(cellType string, cells []exportmodel.Cell) node {
	row := node{Type: "tableRow"}
	for _, cell := range cells {
		row.Content = append(row.Content, node{Type: cellType, Content: []node{{Type: "paragraph", Content: c.inlines(cell, nil)}}})
	}

	return row
}

// inlines flattens nested formatting into text nodes that each carry every mark that applies to
// them, which is how ADF writes it.
func (c *converter) inlines(list []exportmodel.Inline, marks []mark) []node {
	var nodes []node
	for _, inline := range list {
		switch inline.Kind {
		case exportmodel.TextInline:
			nodes = appendText(nodes, inline.Text, marks)
		case exportmodel.CodeInline:
			nodes = appendText(nodes, inline.Text, codeMarks(marks))
		case exportmodel.EmphasisInline:
			nodes = append(nodes, c.inlines(inline.Children, withMark(marks, mark{Type: "em"}))...)
		case exportmodel.StrongInline:
			nodes = append(nodes, c.inlines(inline.Children, withMark(marks, mark{Type: "strong"}))...)
		case exportmodel.StrikeInline:
			nodes = append(nodes, c.inlines(inline.Children, withMark(marks, mark{Type: "strike"}))...)
		case exportmodel.LinkInline:
			nodes = append(nodes, c.link(inline, marks)...)
		case exportmodel.ImageInline:
			label := inline.Text
			if label == "" {
				label = inline.URL
			}
			nodes = appendText(nodes, label, withMark(marks, mark{Type: "link", Attrs: map[string]any{"href": inline.URL}}))
		case exportmodel.SoftBreakInline:
			nodes = appendText(nodes, " ", marks)
		case exportmodel.HardBreakInline:
			nodes = append(nodes, node{Type: "hardBreak"})
		case exportmodel.HTMLInline:
		}
	}

	return nodes
}

func (c *converter) link(inline exportmodel.Inline, marks []mark) []node {
	if inline.URL == "" {
		return c.inlines(inline.Children, marks)
	}

	return c.inlines(inline.Children, withMark(marks, mark{Type: "link", Attrs: map[string]any{"href": inline.URL}}))
}

func appendText(nodes []node, text string, marks []mark) []node {
	if text == "" {
		return nodes
	}

	return append(nodes, node{Type: "text", Text: text, Marks: marks})
}

// withMark returns marks plus one more, without sharing storage with the caller's slice.
func withMark(marks []mark, added mark) []mark {
	combined := make([]mark, 0, len(marks)+1)

	return append(append(combined, marks...), added)
}

// codeMarks is what a code span keeps of the marks around it: ADF lets the code mark sit only
// beside a link.
func codeMarks(marks []mark) []mark {
	kept := []mark{{Type: "code"}}
	for _, each := range marks {
		if each.Type == "link" {
			kept = append(kept, each)
		}
	}

	return kept
}
