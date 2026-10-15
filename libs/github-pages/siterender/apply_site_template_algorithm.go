package siterender

import (
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var placeholder = regexp.MustCompile(`\{\{\s*([A-Za-z_]+)\s*\}\}`)

// ShellProblems says what is wrong with a template's HTML: no {{content}}, or a placeholder
// LoreMaster does not know (a typo that would otherwise show up as literal braces).
func ShellProblems(shell string) []string {
	var problems []string
	found := map[string]bool{}
	for _, match := range placeholder.FindAllStringSubmatch(shell, -1) {
		name := match[1]
		found[name] = true
		message := "unknown placeholder {{" + name + "}}; the known ones are " + strings.Join(Placeholders, ", ")
		if !slices.Contains(Placeholders, name) && !slices.Contains(problems, message) {
			problems = append(problems, message)
		}
	}
	if !found["content"] {
		problems = append([]string{"it has no {{content}} placeholder, so no page would show its text"}, problems...)
	}

	return problems
}

// fillShell replaces every known placeholder in shell with its value. A placeholder it does
// not know is left as written.
func fillShell(shell string, values map[string]string) string {
	return placeholder.ReplaceAllStringFunc(shell, func(token string) string {
		name := placeholder.FindStringSubmatch(token)[1]
		if value, ok := values[name]; ok {
			return value
		}

		return token
	})
}

// navHTML is the page tree as a list, hrefs relative to the page being rendered.
func navHTML(items []navItem) string {
	var out strings.Builder
	out.WriteString("<ul>\n")
	for _, item := range items {
		out.WriteString(`<li class="lm-depth-` + strconv.Itoa(item.Depth))
		if item.Active {
			out.WriteString(" lm-active")
		}
		out.WriteString(`"><a href="` + html.EscapeString(item.Href) + `">` + html.EscapeString(item.Label) + "</a></li>\n")
	}
	out.WriteString("</ul>")

	return out.String()
}

// breadcrumbsHTML links the page's ancestors, nearest last; "" for a top-level page.
func breadcrumbsHTML(ancestors []navItem) string {
	if len(ancestors) == 0 {
		return ""
	}
	parts := make([]string, len(ancestors))
	for i, item := range ancestors {
		parts[i] = `<a href="` + html.EscapeString(item.Href) + `">` + html.EscapeString(item.Label) + "</a>"
	}

	return `<nav class="lm-breadcrumbs" aria-label="Breadcrumb">` + strings.Join(parts, " › ") + "</nav>"
}
