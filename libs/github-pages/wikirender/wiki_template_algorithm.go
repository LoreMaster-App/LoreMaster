package wikirender

import (
	"regexp"
	"slices"
	"strings"
)

// WikiPlaceholders are the names a wiki template may use, written {{name}}.
var WikiPlaceholders = []string{"content", "title", "source"}

var wikiPlaceholder = regexp.MustCompile(`\{\{\s*([A-Za-z_]+)\s*\}\}`)

// ShellProblems says what is wrong with a wiki template: no {{content}}, or a placeholder
// the wiki does not know.
func ShellProblems(shell string) []string {
	var problems []string
	found := map[string]bool{}
	for _, match := range wikiPlaceholder.FindAllStringSubmatch(shell, -1) {
		found[match[1]] = true
		message := "unknown placeholder {{" + match[1] + "}}; the known ones are " + strings.Join(WikiPlaceholders, ", ")
		if !slices.Contains(WikiPlaceholders, match[1]) && !slices.Contains(problems, message) {
			problems = append(problems, message)
		}
	}
	if !found["content"] {
		problems = append([]string{"it has no {{content}} placeholder, so no page would show its text"}, problems...)
	}

	return problems
}

// wrapInShell puts a page's Markdown into the template. A template with no shell leaves the
// page as it is.
func wrapInShell(shell string, page wikiPage, content []byte) []byte {
	if shell == "" {
		return content
	}
	values := map[string]string{"content": strings.TrimRight(string(content), "\n"), "title": page.title, "source": page.source}
	filled := wikiPlaceholder.ReplaceAllStringFunc(shell, func(token string) string {
		if value, ok := values[wikiPlaceholder.FindStringSubmatch(token)[1]]; ok {
			return value
		}

		return token
	})
	if !strings.HasSuffix(filled, "\n") {
		filled += "\n"
	}

	return []byte(filled)
}
