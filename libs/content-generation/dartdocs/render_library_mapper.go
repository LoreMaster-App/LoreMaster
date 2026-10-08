package dartdocs

import (
	"regexp"
	"strings"

	"lore-master/libs/content-generation/markdownwriting"
)

var headingLine = regexp.MustCompile(`^#{1,6}\s+(.*)$`)

// renderLibrary writes the page for one file: its public declarations in source order, each
// with its signature and doc comment and then its members. A file with nothing public gives an
// empty string, so it has no page.
func renderLibrary(title string, declarations []declaration) string {
	var out strings.Builder
	for _, d := range declarations {
		if isPrivate(d.Name) {
			continue
		}
		renderDeclaration(&out, d)
	}
	if out.Len() == 0 {
		return ""
	}

	return "# " + title + "\n\n" + out.String()
}

func renderDeclaration(out *strings.Builder, d declaration) {
	out.WriteString("## " + heading(d) + "\n\n")
	out.WriteString(codeAndDescription(d, signature(d, nil)))

	if len(d.Values) > 0 {
		out.WriteString("**Values**\n\n")
		for _, value := range d.Values {
			out.WriteString("- `" + value.Name + "`")
			if text := firstParagraph(value.Description); text != "" {
				out.WriteString(" — " + text)
			}
			out.WriteString("\n")
		}
		out.WriteString("\n")
	}

	fieldTypes := map[string]string{}
	for _, member := range d.Members {
		if member.Kind == "field" {
			fieldTypes[member.Name] = member.Type
		}
	}
	for _, member := range d.Members {
		if isPrivate(member.Name) || strings.Contains(member.Name, "._") {
			continue
		}
		out.WriteString("### " + memberHeading(d, member) + "\n\n")
		out.WriteString(codeAndDescription(member, signature(member, fieldTypes)))
	}
}

// heading is "class Cart", "function run", "variable version".
func heading(d declaration) string {
	return d.Kind + " " + d.Name
}

func memberHeading(owner declaration, member declaration) string {
	switch member.Kind {
	case "constructor":
		if member.Name == owner.Name {
			return owner.Name + " (constructor)"
		}

		return member.Name + " (constructor)"
	case "getter":
		return member.Name + " (getter)"
	case "setter":
		return member.Name + " (setter)"
	}

	return member.Name
}

// codeAndDescription is the annotations and signature in a Dart code block, then the doc comment.
func codeAndDescription(d declaration, signatureText string) string {
	var lines []string
	for _, a := range d.Annotations {
		if a.Name == "@override" {
			continue
		}
		text := a.Name
		if len(a.Arguments) > 0 {
			text += "(" + strings.Join(a.Arguments, ", ") + ")"
		}
		lines = append(lines, text)
	}
	lines = append(lines, signatureText)

	out := markdownwriting.CodeBlock("dart", strings.Join(lines, "\n")) + "\n"
	if description := cleanDescription(d.Description); description != "" {
		out += description + "\n\n"
	}

	return out
}

// cleanDescription keeps a doc comment's Markdown but makes a heading in it bold text, so it
// cannot break the page's own outline (or become its title).
func cleanDescription(text string) string {
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n")), "\n")
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
		}
		if !inFence {
			if match := headingLine.FindStringSubmatch(line); match != nil {
				lines[i] = "**" + match[1] + "**"
			}
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func firstParagraph(text string) string {
	paragraph, _, _ := strings.Cut(strings.TrimSpace(text), "\n\n")

	return strings.Join(strings.Fields(paragraph), " ")
}

func isPrivate(name string) bool {
	return strings.HasPrefix(name, "_")
}
