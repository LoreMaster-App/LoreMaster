package confluenceplatform

import (
	"errors"
	"html"
	"regexp"
	"strings"
)

// The content region of a templated page is fenced by two anchor macros, which Confluence
// shows as nothing. A two-way pull reads only what is between them, so the template's own
// text is never turned into Markdown.
const (
	contentStartAnchor = "lm-content-start"
	contentEndAnchor   = "lm-content-end"
)

func anchorMacro(name string) string {
	return `<ac:structured-macro ac:name="anchor" ac:schema-version="1"><ac:parameter ac:name="">` + name + `</ac:parameter></ac:structured-macro>`
}

// tokenPattern matches a placeholder as an author may have left it: {{name}}, or the
// monospace text the Confluence editor turns "{{name}}" into.
func tokenPattern(name string) string {
	return `(?:\{\{\s*` + name + `\s*\}\}|<code>\s*` + name + `\s*</code>|<code>\s*\{\{\s*` + name + `\s*\}\}\s*</code>)`
}

var (
	// contentParagraph is a paragraph that holds only the content placeholder: replaced whole,
	// so the page's blocks do not end up inside a <p>.
	contentParagraph = regexp.MustCompile(`(?s)<p(?:\s[^>]*)?>\s*` + tokenPattern("content") + `\s*</p>`)
	contentToken     = regexp.MustCompile(tokenPattern("content"))
	titleToken       = regexp.MustCompile(tokenPattern("title"))

	startAnchorPattern = regexp.MustCompile(`(?s)<ac:structured-macro[^>]*ac:name="anchor"[^>]*>\s*<ac:parameter[^>]*>` + contentStartAnchor + `</ac:parameter>\s*</ac:structured-macro>`)
	endAnchorPattern   = regexp.MustCompile(`(?s)<ac:structured-macro[^>]*ac:name="anchor"[^>]*>\s*<ac:parameter[^>]*>` + contentEndAnchor + `</ac:parameter>\s*</ac:structured-macro>`)
	leadingParagraph   = regexp.MustCompile(`^\s*</p>`)
	trailingParagraph  = regexp.MustCompile(`<p(?:\s[^>]*)?>\s*$`)
)

// TemplateProblems says what is wrong with a template's storage format: no {{content}}.
func TemplateProblems(shell string) []string {
	if !contentToken.MatchString(shell) {
		return []string{"it has no {{content}} placeholder, so no page would show its text"}
	}

	return nil
}

// ApplyTemplate puts a page's body (storage format) into the template. {{title}} is replaced
// by the page's title. The body sits between two invisible anchors so ExtractContent can find
// it again.
func ApplyTemplate(shell string, body string, title string) (string, error) {
	if shell == "" {
		return body, nil
	}
	if problems := TemplateProblems(shell); len(problems) > 0 {
		return "", errors.New("the Confluence template is not usable: " + problems[0])
	}
	region := anchorMacro(contentStartAnchor) + body + anchorMacro(contentEndAnchor)

	placed := false
	replace := func(string) string {
		if placed {
			return ""
		}
		placed = true

		return region
	}
	filled := contentParagraph.ReplaceAllStringFunc(shell, replace)
	filled = contentToken.ReplaceAllStringFunc(filled, replace)
	filled = titleToken.ReplaceAllLiteralString(filled, html.EscapeString(title))

	return filled, nil
}

// ExtractContent is the page's own content within a templated body, the part a pull converts
// back to Markdown. A body without the anchors (a page that was never templated, or one whose
// anchors were deleted) is returned whole.
func ExtractContent(storage string) string {
	start := startAnchorPattern.FindStringIndex(storage)
	if start == nil {
		return storage
	}
	rest := storage[start[1]:]
	end := endAnchorPattern.FindStringIndex(rest)
	if end == nil {
		return storage
	}
	content := rest[:end[0]]
	// Confluence may have wrapped each macro in a paragraph of its own.
	content = leadingParagraph.ReplaceAllString(content, "")
	content = trailingParagraph.ReplaceAllString(content, "")

	return strings.TrimSpace(content)
}
