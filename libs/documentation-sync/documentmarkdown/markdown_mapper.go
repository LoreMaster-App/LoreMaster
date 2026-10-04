package documentmarkdown

import (
	"fmt"
	"strings"

	"lore-master/libs/documentation-sync/platformport"
)

// Links maps a synced page's final (prefixed) title to the Markdown target that links to its
// local file — a workspace-relative path. ok is false when no local file owns that title.
type Links func(title string) (target string, ok bool)

// ToMarkdown serialises a neutral Document as Markdown, the reverse of documentconversion.
// links resolves a page link back to the local file that owns its title; an unresolved page
// link falls back to its URL, then to its link text, and is flagged. The returned flags name
// content that could not be represented faithfully in Markdown; the caller decides whether to
// write the result or leave the file for the user. Passing a nil links treats every page link
// as unresolved.
func ToMarkdown(doc platformport.Document, links Links) (markdown string, flags []string) {
	if links == nil {
		links = func(string) (string, bool) { return "", false }
	}
	r := &renderer{links: links}
	body := r.blocks(doc.Blocks)
	return strings.TrimRight(body, "\n") + "\n", r.flags
}

type renderer struct {
	links Links
	flags []string
}

func (r *renderer) flag(format string, a ...any) {
	r.flags = append(r.flags, fmt.Sprintf(format, a...))
}

// blocks renders a run of blocks, separated by a blank line.
func (r *renderer) blocks(blocks []platformport.Block) string {
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if s := r.block(b); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n\n")
}

func (r *renderer) block(b platformport.Block) string {
	switch n := b.(type) {
	case platformport.Heading:
		return strings.Repeat("#", clampLevel(n.Level)) + " " + r.inlines(n.Inlines)
	case platformport.Paragraph:
		return r.inlines(n.Inlines)
	case platformport.ThematicBreak:
		return "---"
	case platformport.CodeBlock:
		return fence(n.Language, n.Code)
	case platformport.Diagram:
		lang := n.Language
		if lang == "" {
			lang = "mermaid"
		}
		return fence(lang, n.Source)
	case platformport.Blockquote:
		return prefixLines(r.blocks(n.Blocks), "> ", ">")
	case platformport.List:
		return r.list(n)
	case platformport.TaskList:
		return r.taskList(n)
	case platformport.Table:
		return r.table(n)
	default:
		r.flag("a %T block could not be converted to Markdown", b)
		return ""
	}
}

func (r *renderer) list(l platformport.List) string {
	lines := make([]string, 0, len(l.Items))
	for i, item := range l.Items {
		marker := "- "
		if l.Ordered {
			marker = fmt.Sprintf("%d. ", listStart(l.Start)+i)
		}
		lines = append(lines, indent(marker, r.blocks(item.Blocks)))
	}
	return strings.Join(lines, "\n")
}

func (r *renderer) taskList(t platformport.TaskList) string {
	lines := make([]string, 0, len(t.Items))
	for _, item := range t.Items {
		box := "- [ ] "
		if item.Done {
			box = "- [x] "
		}
		text := r.inlines(item.Inlines)
		if nested := r.blocks(item.Blocks); nested != "" {
			text = strings.TrimRight(text+"\n\n"+nested, "\n")
		}
		lines = append(lines, indent(box, text))
	}
	return strings.Join(lines, "\n")
}

func (r *renderer) table(t platformport.Table) string {
	columns := len(t.Header)
	for _, row := range t.Rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	if columns == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(r.row(t.Header, columns))
	b.WriteString("\n")
	b.WriteString(dividerRow(t.Align, columns))
	for _, row := range t.Rows {
		b.WriteString("\n")
		b.WriteString(r.row(row, columns))
	}
	return b.String()
}

func (r *renderer) row(cells []platformport.TableCell, columns int) string {
	out := make([]string, columns)
	for i := range out {
		if i < len(cells) {
			out[i] = escapeCell(r.inlines(cells[i].Inlines))
		}
	}
	return "| " + strings.Join(out, " | ") + " |"
}

func (r *renderer) inlines(inlines []platformport.Inline) string {
	var b strings.Builder
	for _, in := range inlines {
		b.WriteString(r.inline(in))
	}
	return b.String()
}

func (r *renderer) inline(in platformport.Inline) string {
	switch n := in.(type) {
	case platformport.Text:
		return escapeText(n.Value)
	case platformport.Emphasis:
		return "*" + r.inlines(n.Inlines) + "*"
	case platformport.Strong:
		return "**" + r.inlines(n.Inlines) + "**"
	case platformport.Strikethrough:
		return "~~" + r.inlines(n.Inlines) + "~~"
	case platformport.CodeSpan:
		return codeSpan(n.Value)
	case platformport.HardBreak:
		return "  \n"
	case platformport.Image:
		return r.image(n)
	case platformport.Link:
		return r.link(n)
	default:
		r.flag("a %T inline could not be converted to Markdown", in)
		return ""
	}
}

func (r *renderer) image(img platformport.Image) string {
	src := ""
	switch s := img.Source.(type) {
	case *platformport.AttachmentRef:
		src = s.Filename
	case *platformport.URLRef:
		src = s.URL
	default:
		r.flag("an image with an unknown source could not be converted to Markdown")
	}
	alt := escapeText(img.Alt)
	if img.Title != "" {
		return fmt.Sprintf("![%s](%s %q)", alt, src, img.Title)
	}
	return fmt.Sprintf("![%s](%s)", alt, src)
}

func (r *renderer) link(link platformport.Link) string {
	text := r.inlines(link.Inlines)
	target := ""
	switch t := link.Target.(type) {
	case platformport.PageLink:
		if resolved, ok := r.links(t.Title); ok {
			target = resolved
		} else if t.URL != "" {
			target = t.URL
			r.flag("a link to page %q fell back to its URL; no local file owns that title", t.Title)
		} else {
			r.flag("a link to page %q could not be resolved to a local file or URL", t.Title)
			return text
		}
		if t.Anchor != "" {
			target += "#" + t.Anchor
		}
	case *platformport.AttachmentRef:
		target = t.Filename
	case *platformport.URLRef:
		target = t.URL
	default:
		r.flag("a link with an unknown target could not be converted to Markdown")
		return text
	}
	return "[" + text + "](" + target + ")"
}

func clampLevel(level int) int {
	switch {
	case level < 1:
		return 1
	case level > 6:
		return 6
	default:
		return level
	}
}

func listStart(start int) int {
	if start < 1 {
		return 1
	}
	return start
}

// fence wraps code in a fence long enough to contain any backtick run inside it.
func fence(language, code string) string {
	ticks := "```"
	for strings.Contains(code, ticks) {
		ticks += "`"
	}
	return ticks + language + "\n" + strings.TrimRight(code, "\n") + "\n" + ticks
}

// codeSpan wraps inline code in enough backticks to contain any run inside it, padding with a
// space when the content starts or ends with a backtick (CommonMark rule).
func codeSpan(value string) string {
	ticks := "`"
	for strings.Contains(value, ticks) {
		ticks += "`"
	}
	if strings.HasPrefix(value, "`") || strings.HasSuffix(value, "`") {
		return ticks + " " + value + " " + ticks
	}
	return ticks + value + ticks
}

// indent prefixes the first line with marker and the rest with marker's width in spaces, so a
// multi-block list item lines up under its bullet.
func indent(marker, content string) string {
	lines := strings.Split(content, "\n")
	pad := strings.Repeat(" ", len(marker))
	for i, line := range lines {
		switch {
		case i == 0:
			lines[i] = marker + line
		case line == "":
			lines[i] = ""
		default:
			lines[i] = pad + line
		}
	}
	return strings.Join(lines, "\n")
}

// prefixLines prefixes non-empty lines with prefix and empty lines with emptyPrefix.
func prefixLines(content, prefix, emptyPrefix string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = emptyPrefix
		} else {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func dividerRow(align []platformport.Alignment, columns int) string {
	cells := make([]string, columns)
	for i := range cells {
		a := platformport.AlignDefault
		if i < len(align) {
			a = align[i]
		}
		switch a {
		case platformport.AlignLeft:
			cells[i] = ":---"
		case platformport.AlignCenter:
			cells[i] = ":---:"
		case platformport.AlignRight:
			cells[i] = "---:"
		default:
			cells[i] = "---"
		}
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

// markdownEscapes are the characters escaped in ordinary text so the serialised Markdown does
// not re-parse as a construct it did not come from.
// Pipe is not escaped here — it is only special inside a table cell, handled by escapeCell.
// Intraword underscores are not emphasis in GFM, so escaping every "_" would needlessly
// uglify snake_case; a true Markdown emphasis arrives as an Emphasis node, not as text.
var markdownEscapes = strings.NewReplacer(
	`\`, `\\`,
	"`", "\\`",
	"*", `\*`,
	"[", `\[`,
	"]", `\]`,
	"<", `\<`,
)

func escapeText(s string) string {
	return markdownEscapes.Replace(s)
}

// escapeCell escapes a cell's rendered inlines for a one-line table cell: pipes are escaped and
// hard breaks collapse to spaces, since a GFM cell cannot span lines.
func escapeCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "|", `\|`)
}
