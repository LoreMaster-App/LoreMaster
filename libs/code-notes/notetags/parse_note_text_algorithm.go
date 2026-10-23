package notetags

import (
	"regexp"
	"slices"
	"strings"
)

var (
	issueForm = regexp.MustCompile(`^#(\d+)$`)
	dateForm  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Matcher finds a note at the start of a piece of comment text.
type Matcher struct {
	pattern *regexp.Regexp
}

// NewMatcher looks for any of tags (upper case words), each at the start of the text, optionally
// followed by brackets and a colon: "TODO: x", "FIXME(alice): x", "NOTE x".
func NewMatcher(tags []string) Matcher {
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		if word := strings.TrimSpace(tag); word != "" {
			cleaned = append(cleaned, regexp.QuoteMeta(word))
		}
	}
	if len(cleaned) == 0 {
		cleaned = []string{"TODO"}
	}
	// Longer tags first so FIXME is never read as a shorter tag that starts it.
	slices.SortStableFunc(cleaned, func(a, b string) int { return len(b) - len(a) })

	return Matcher{pattern: regexp.MustCompile(`^(` + strings.Join(cleaned, "|") + `)\b(?:\(([^)]*)\))?[ \t]*:?[ \t]*(.*)$`)}
}

// Parse reads a note from text (a comment's text with its marker and leading blanks taken off). ok is
// false when the text does not start with a tag. Path, Line and Column are left for the caller.
func (m Matcher) Parse(text string) (Note, bool) {
	match := m.pattern.FindStringSubmatch(text)
	if match == nil {
		return Note{}, false
	}
	note := Note{Tag: match[1], Text: strings.TrimSpace(match[3])}
	for _, part := range strings.Split(match[2], ",") {
		switch part = strings.TrimSpace(part); {
		case part == "":
		case issueForm.MatchString(part):
			note.Issue = issueForm.FindStringSubmatch(part)[1]
		case dateForm.MatchString(part):
			note.Due = part
		default:
			note.Owner = part
		}
	}

	return note, true
}
