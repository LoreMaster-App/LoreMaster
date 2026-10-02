package storageformat

import (
	"fmt"
	"strconv"
	"strings"
)

// Render writes doc as Confluence storage format. It never emits an XML comment, so the
// rule that a comment may not contain "--" cannot be broken.
func Render(doc Document) (string, error) {
	r := &renderer{}
	for _, block := range doc.Blocks {
		if err := r.block(block); err != nil {
			return "", err
		}
	}

	return r.out.String(), nil
}

type renderer struct {
	out strings.Builder
	// taskID numbers task-list items in document order, which keeps the output stable
	// between syncs (Confluence would otherwise assign ids of its own).
	taskID int
}

func (r *renderer) block(block Block) error {
	switch b := block.(type) {
	case Paragraph:
		return r.wrapBlock("<p>", b.Inlines, "</p>")
	case Heading:
		if b.Level < 1 || b.Level > 6 {
			return fmt.Errorf("heading level %d is outside 1 to 6", b.Level)
		}
		tag := "h" + strconv.Itoa(b.Level)

		return r.wrapBlock("<"+tag+">", b.Inlines, "</"+tag+">")
	case Blockquote:
		r.out.WriteString("<blockquote>")
		for _, inner := range b.Blocks {
			if err := r.block(inner); err != nil {
				return err
			}
		}
		r.out.WriteString("</blockquote>")
	case List:
		return r.list(b)
	case TaskList:
		r.out.WriteString("<ac:task-list>")
		for _, item := range b.Items {
			r.taskID++
			status := "incomplete"
			if item.Done {
				status = "complete"
			}
			open := "<ac:task><ac:task-id>" + strconv.Itoa(r.taskID) + "</ac:task-id><ac:task-status>" + status + "</ac:task-status><ac:task-body>"
			if err := r.wrapBlock(open, item.Inlines, "</ac:task-body></ac:task>"); err != nil {
				return err
			}
		}
		r.out.WriteString("</ac:task-list>")
	case Table:
		return r.table(b)
	case ThematicBreak:
		r.out.WriteString("<hr />")
	case nil:
		return fmt.Errorf("nil block")
	default:
		return fmt.Errorf("unsupported block %T", block)
	}

	return nil
}

func (r *renderer) list(list List) error {
	tag := "ul"
	open := "<ul>"
	if list.Ordered {
		tag = "ol"
		open = "<ol>"
		if list.Start > 1 {
			open = `<ol start="` + strconv.Itoa(list.Start) + `">`
		}
	}
	r.out.WriteString(open)
	for _, item := range list.Items {
		r.out.WriteString("<li>")
		if paragraph, single := singleParagraph(item); single {
			if err := r.inlines(paragraph.Inlines); err != nil {
				return err
			}
		} else {
			for _, inner := range item.Blocks {
				if err := r.block(inner); err != nil {
					return err
				}
			}
		}
		r.out.WriteString("</li>")
	}
	r.out.WriteString("</" + tag + ">")

	return nil
}

// singleParagraph is a tight list item: its one paragraph renders without <p>.
func singleParagraph(item ListItem) (Paragraph, bool) {
	if len(item.Blocks) != 1 {
		return Paragraph{}, false
	}
	paragraph, ok := item.Blocks[0].(Paragraph)

	return paragraph, ok
}

func (r *renderer) table(table Table) error {
	r.out.WriteString("<table><tbody>")
	if len(table.Header) > 0 {
		if err := r.row(table.Header, "th", table.Align); err != nil {
			return err
		}
	}
	for _, row := range table.Rows {
		if err := r.row(row, "td", table.Align); err != nil {
			return err
		}
	}
	r.out.WriteString("</tbody></table>")

	return nil
}

var alignStyles = map[Alignment]string{
	AlignLeft: ` style="text-align: left;"`, AlignCenter: ` style="text-align: center;"`, AlignRight: ` style="text-align: right;"`,
}

func (r *renderer) row(cells []TableCell, tag string, align []Alignment) error {
	r.out.WriteString("<tr>")
	for i, cell := range cells {
		style := ""
		if i < len(align) {
			style = alignStyles[align[i]]
		}
		if err := r.wrapBlock("<"+tag+style+">", cell.Inlines, "</"+tag+">"); err != nil {
			return err
		}
	}
	r.out.WriteString("</tr>")

	return nil
}

func (r *renderer) inlines(inlines []Inline) error {
	for _, inline := range inlines {
		var err error
		switch n := inline.(type) {
		case Text:
			r.out.WriteString(escapeText(n.Value))
		case Emphasis:
			err = r.wrapBlock("<em>", n.Inlines, "</em>")
		case Strong:
			err = r.wrapBlock("<strong>", n.Inlines, "</strong>")
		case Strikethrough:
			err = r.wrapBlock(`<span style="text-decoration: line-through;">`, n.Inlines, "</span>")
		case CodeSpan:
			r.out.WriteString("<code>" + escapeText(n.Value) + "</code>")
		case HardBreak:
			r.out.WriteString("<br />")
		case nil:
			err = fmt.Errorf("nil inline")
		default:
			err = fmt.Errorf("unsupported inline %T", inline)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// wrapBlock writes open, the inlines, then closing.
func (r *renderer) wrapBlock(open string, inlines []Inline, closing string) error {
	r.out.WriteString(open)
	if err := r.inlines(inlines); err != nil {
		return err
	}
	r.out.WriteString(closing)

	return nil
}
