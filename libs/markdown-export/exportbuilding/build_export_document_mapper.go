// Package exportbuilding turns a parsed Markdown document into the neutral model the export
// formats render from.
package exportbuilding

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"

	"lore-master/libs/markdown-export/exportmodel"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// builder walks one document's syntax tree; body is what its segments index into.
type builder struct {
	body    []byte
	anchors map[*ast.Heading]string
}

// BuildDocument maps a parsed document to the export model. Front matter and the lore-master
// annotation are not part of the parsed tree, so they never reach an export.
func BuildDocument(document documentparsing.MarkdownDocument) exportmodel.Document {
	anchors := map[*ast.Heading]string{}
	for _, heading := range documentparsing.Headings(document) {
		anchors[heading.Node] = heading.Slug
	}
	walk := builder{body: document.Body, anchors: anchors}

	return exportmodel.Document{Title: document.Title, Blocks: walk.blocks(document.AST)}
}

func (b builder) blocks(parent ast.Node) []exportmodel.Block {
	var blocks []exportmodel.Block
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if block, ok := b.block(child); ok {
			blocks = append(blocks, block)
		}
	}

	return blocks
}

func (b builder) block(node ast.Node) (exportmodel.Block, bool) {
	switch current := node.(type) {
	case *ast.Heading:
		return exportmodel.Block{Kind: exportmodel.HeadingBlock, Level: current.Level, Anchor: b.anchors[current], Inlines: b.inlines(current)}, true
	case *ast.Paragraph, *ast.TextBlock:
		return exportmodel.Block{Kind: exportmodel.ParagraphBlock, Inlines: b.inlines(current)}, true
	case *ast.List:
		return b.list(current), true
	case *ast.Blockquote:
		return exportmodel.Block{Kind: exportmodel.QuoteBlock, Children: b.blocks(current)}, true
	case *ast.FencedCodeBlock:
		return exportmodel.Block{Kind: exportmodel.CodeBlock, Lang: string(current.Language(b.body)), Text: b.lines(current)}, true
	case *ast.CodeBlock:
		return exportmodel.Block{Kind: exportmodel.CodeBlock, Text: b.lines(current)}, true
	case *ast.ThematicBreak:
		return exportmodel.Block{Kind: exportmodel.RuleBlock}, true
	case *ast.HTMLBlock:
		return exportmodel.Block{Kind: exportmodel.HTMLBlock, Text: b.lines(current)}, true
	case *extast.Table:
		return b.table(current), true
	}

	return exportmodel.Block{}, false
}

// lines is the verbatim text of a block made of lines, without the newline that ends it.
func (b builder) lines(node ast.Node) string {
	var text strings.Builder
	segments := node.Lines()
	for i := range segments.Len() {
		segment := segments.At(i)
		text.Write(segment.Value(b.body))
	}

	return strings.TrimRight(text.String(), "\r\n")
}

func (b builder) list(list *ast.List) exportmodel.Block {
	block := exportmodel.Block{Kind: exportmodel.ListBlock, Ordered: list.IsOrdered(), Start: list.Start}
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		entry := exportmodel.Item{Blocks: b.blocks(item)}
		if checked, isTask := taskState(item); isTask {
			entry.Checked = &checked
		}
		block.Items = append(block.Items, entry)
	}

	return block
}

// taskState reads the "[ ]" or "[x]" at the start of an item.
func taskState(item ast.Node) (bool, bool) {
	first := item.FirstChild()
	if first == nil {
		return false, false
	}
	if box, ok := first.FirstChild().(*extast.TaskCheckBox); ok {
		return box.IsChecked, true
	}

	return false, false
}

func (b builder) table(table *extast.Table) exportmodel.Block {
	block := exportmodel.Block{Kind: exportmodel.TableBlock}
	for _, alignment := range table.Alignments {
		block.Aligns = append(block.Aligns, alignOf(alignment))
	}
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		cells := make([]exportmodel.Cell, 0, len(block.Aligns))
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, b.inlines(cell))
		}
		if _, isHeader := row.(*extast.TableHeader); isHeader {
			block.Header = cells
		} else {
			block.Rows = append(block.Rows, cells)
		}
	}

	return block
}

func alignOf(alignment extast.Alignment) exportmodel.Align {
	switch alignment {
	case extast.AlignLeft:
		return exportmodel.AlignLeft
	case extast.AlignCenter:
		return exportmodel.AlignCenter
	case extast.AlignRight:
		return exportmodel.AlignRight
	default:
		return exportmodel.AlignDefault
	}
}

func (b builder) inlines(parent ast.Node) []exportmodel.Inline {
	var inlines []exportmodel.Inline
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		inlines = append(inlines, b.inline(child)...)
	}

	return inlines
}

func (b builder) inline(node ast.Node) []exportmodel.Inline {
	switch current := node.(type) {
	case *ast.Text:
		return b.text(current)
	case *ast.String:
		return []exportmodel.Inline{{Kind: exportmodel.TextInline, Text: string(current.Value)}}
	case *ast.CodeSpan:
		return []exportmodel.Inline{{Kind: exportmodel.CodeInline, Text: documentparsing.PlainText(current, b.body)}}
	case *ast.Emphasis:
		kind := exportmodel.EmphasisInline
		if current.Level >= 2 {
			kind = exportmodel.StrongInline
		}

		return []exportmodel.Inline{{Kind: kind, Children: b.inlines(current)}}
	case *extast.Strikethrough:
		return []exportmodel.Inline{{Kind: exportmodel.StrikeInline, Children: b.inlines(current)}}
	case *ast.Link:
		return []exportmodel.Inline{{Kind: exportmodel.LinkInline, URL: string(current.Destination), Title: string(current.Title), Children: b.inlines(current)}}
	case *ast.AutoLink:
		address := string(current.URL(b.body))
		label := string(current.Label(b.body))

		return []exportmodel.Inline{{Kind: exportmodel.LinkInline, URL: address, Children: []exportmodel.Inline{{Kind: exportmodel.TextInline, Text: label}}}}
	case *ast.Image:
		return []exportmodel.Inline{{Kind: exportmodel.ImageInline, URL: string(current.Destination), Title: string(current.Title), Text: documentparsing.PlainText(current, b.body)}}
	case *ast.RawHTML:
		return []exportmodel.Inline{{Kind: exportmodel.HTMLInline, Text: b.rawHTML(current)}}
	case *extast.TaskCheckBox:
		return nil
	}

	return b.inlines(node)
}

func (b builder) rawHTML(node *ast.RawHTML) string {
	var text strings.Builder
	for i := range node.Segments.Len() {
		segment := node.Segments.At(i)
		text.Write(segment.Value(b.body))
	}

	return text.String()
}

// text is a text run plus the line break that ends it, if any.
func (b builder) text(node *ast.Text) []exportmodel.Inline {
	inlines := []exportmodel.Inline{{Kind: exportmodel.TextInline, Text: string(node.Value(b.body))}}
	switch {
	case node.HardLineBreak():
		inlines = append(inlines, exportmodel.Inline{Kind: exportmodel.HardBreakInline})
	case node.SoftLineBreak():
		inlines = append(inlines, exportmodel.Inline{Kind: exportmodel.SoftBreakInline})
	}

	return inlines
}
