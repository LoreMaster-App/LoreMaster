package notesdocs

import (
	"fmt"
	"slices"
	"strings"

	"lore-master/libs/code-notes/notetags"
)

// cell makes text safe inside a Markdown table cell: a pipe would end it and angle brackets would start a tag.
func cell(text string) string {
	return strings.NewReplacer("|", `\|`, "&", "&amp;", "<", "&lt;", ">", "&gt;", "\n", " ").Replace(text)
}

// tagOrder puts the default tags first, in their order, then any other tag alphabetically.
func tagOrder(notes []notetags.Note) []string {
	present := map[string]bool{}
	for _, note := range notes {
		present[note.Tag] = true
	}
	var order []string
	for _, tag := range notetags.DefaultTags {
		if present[tag] {
			order = append(order, tag)
			delete(present, tag)
		}
	}
	var others []string
	for tag := range present {
		others = append(others, tag)
	}
	slices.Sort(others)

	return append(order, others...)
}

// renderPage writes the page: a heading, a summary, and a table per tag.
func renderPage(title string, notes []notetags.Note, files int) string {
	var page strings.Builder
	page.WriteString("# " + title + "\n\n")
	if len(notes) == 0 {
		page.WriteString("No notes were found in the comments of the source files.\n")

		return page.String()
	}
	fmt.Fprintf(&page, "The tagged comments of the source files: %d in %d files.\n\n", len(notes), files)

	order := tagOrder(notes)
	page.WriteString("| Tag | Notes |\n| --- | ---: |\n")
	for _, tag := range order {
		fmt.Fprintf(&page, "| %s | %d |\n", tag, count(notes, tag))
	}
	for _, tag := range order {
		page.WriteString("\n" + renderTag(tag, notes))
	}

	return page.String()
}

func count(notes []notetags.Note, tag string) int {
	total := 0
	for _, note := range notes {
		if note.Tag == tag {
			total++
		}
	}

	return total
}

// renderTag writes one tag's table; the columns for owner, issue and date appear only when some note of the tag has one.
func renderTag(tag string, notes []notetags.Note) string {
	var rows []notetags.Note
	var owner, issue, due bool
	for _, note := range notes {
		if note.Tag != tag {
			continue
		}
		rows = append(rows, note)
		owner, issue, due = owner || note.Owner != "", issue || note.Issue != "", due || note.Due != ""
	}

	header, rule := "| Note | Where |", "| --- | --- |"
	for _, column := range []struct {
		on   bool
		name string
	}{{owner, "Owner"}, {issue, "Issue"}, {due, "Due"}} {
		if column.on {
			header, rule = header+" "+column.name+" |", rule+" --- |"
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, "## %s (%d)\n\n%s\n%s\n", tag, len(rows), header, rule)
	for _, note := range rows {
		text := cell(note.Text)
		if text == "" {
			text = "_(no text)_"
		}
		fmt.Fprintf(&out, "| %s | `%s:%d` |", text, note.Path, note.Line)
		if owner {
			out.WriteString(" " + cell(note.Owner) + " |")
		}
		if issue {
			if note.Issue != "" {
				out.WriteString(" #" + note.Issue + " |")
			} else {
				out.WriteString("  |")
			}
		}
		if due {
			out.WriteString(" " + note.Due + " |")
		}
		out.WriteString("\n")
	}

	return out.String()
}
