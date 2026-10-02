package documentparsing

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
)

// Heading is one heading with the anchor a "#fragment" link uses to reach it.
type Heading struct {
	Level int
	// Text is the heading's plain text.
	Text string
	// Slug is GitHub's anchor for the heading, so a link written for GitHub or any
	// editor that follows it ("#getting-started") finds the same heading here.
	Slug string
	Node *ast.Heading
}

// Headings lists a document's headings in order, nested ones included, each with
// its slug. Repeated slugs get "-1", "-2", … as on GitHub.
func Headings(document MarkdownDocument) []Heading {
	var headings []Heading
	seen := map[string]int{}
	_ = ast.Walk(document.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		heading, isHeading := node.(*ast.Heading)
		if !entering || !isHeading {
			return ast.WalkContinue, nil
		}
		text := PlainText(heading, document.Body)
		slug := githubSlug(text)
		if count, repeated := seen[slug]; repeated {
			seen[slug] = count + 1
			slug += "-" + strconv.Itoa(count+1)
		} else {
			seen[slug] = 0
		}
		headings = append(headings, Heading{Level: heading.Level, Text: text, Slug: slug, Node: heading})

		return ast.WalkSkipChildren, nil
	})

	return headings
}

// FindHeading returns the heading a fragment points at, matching GitHub's slug first
// and the fragment as written (percent-decoded, any case) second.
func FindHeading(headings []Heading, fragment string) (Heading, bool) {
	fragment = decoded(fragment)
	for _, heading := range headings {
		if heading.Slug == fragment {
			return heading, true
		}
	}
	for _, heading := range headings {
		if strings.EqualFold(heading.Slug, fragment) || strings.EqualFold(heading.Text, fragment) {
			return heading, true
		}
	}

	return Heading{}, false
}

// githubSlug lower-cases the text, drops everything but letters, marks, digits, '_'
// and '-', and turns each space into '-' (runs are not collapsed, as on GitHub).
func githubSlug(text string) string {
	var slug strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case r == ' ':
			slug.WriteRune('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsNumber(r):
			slug.WriteRune(r)
		}
	}

	return slug.String()
}
