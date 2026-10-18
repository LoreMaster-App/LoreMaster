// Package outlineexport renders a document as a JSON outline of its headings, links and tasks,
// for tooling that wants the structure and not the prose.
package outlineexport

import (
	"encoding/json"

	"lore-master/libs/markdown-export/exportmodel"
)

type heading struct {
	Level  int    `json:"level"`
	Text   string `json:"text"`
	Anchor string `json:"anchor,omitempty"`
}

type link struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type task struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type outline struct {
	Title    string    `json:"title"`
	Headings []heading `json:"headings"`
	Links    []link    `json:"links"`
	Tasks    []task    `json:"tasks"`
}

// Render writes the outline as indented JSON.
func Render(document exportmodel.Document) (string, error) {
	result := outline{Title: document.Title, Headings: []heading{}, Links: []link{}, Tasks: []task{}}
	walk(document.Blocks, &result)
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(encoded) + "\n", nil
}

func walk(list []exportmodel.Block, result *outline) {
	for _, current := range list {
		switch current.Kind {
		case exportmodel.HeadingBlock:
			result.Headings = append(result.Headings, heading{Level: current.Level, Text: exportmodel.PlainText(current.Inlines), Anchor: current.Anchor})
			links(current.Inlines, result)
		case exportmodel.ParagraphBlock:
			links(current.Inlines, result)
		case exportmodel.QuoteBlock:
			walk(current.Children, result)
		case exportmodel.ListBlock:
			for _, item := range current.Items {
				if item.Checked != nil && len(item.Blocks) > 0 {
					result.Tasks = append(result.Tasks, task{Text: exportmodel.PlainText(item.Blocks[0].Inlines), Done: *item.Checked})
				}
				walk(item.Blocks, result)
			}
		case exportmodel.TableBlock:
			for _, cell := range current.Header {
				links(cell, result)
			}
			for _, row := range current.Rows {
				for _, cell := range row {
					links(cell, result)
				}
			}
		}
	}
}

func links(inlines []exportmodel.Inline, result *outline) {
	for _, inline := range inlines {
		if inline.Kind == exportmodel.LinkInline {
			result.Links = append(result.Links, link{Text: exportmodel.PlainText(inline.Children), URL: inline.URL})
		}
		links(inline.Children, result)
	}
}
