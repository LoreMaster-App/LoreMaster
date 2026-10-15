package confluenceplatform

import (
	"strings"
	"testing"
)

const pageBody = `<h1>Setup</h1><p>Install &amp; run.</p><ul><li>one</li></ul>`

func TestTheBodyReplacesThePlaceholderParagraph(t *testing.T) {
	shell := `<ac:structured-macro ac:name="info"><ac:rich-text-body><p>Internal</p></ac:rich-text-body></ac:structured-macro><h2>{{title}}</h2><p>{{content}}</p><p>Footer</p>`

	got, err := ApplyTemplate(shell, pageBody, "ENG: Setup & Install")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(got, "<p><ac:structured-macro ac:name=\"anchor\"") || strings.Contains(got, "{{") {
		t.Errorf("the body ended up inside the paragraph, or a placeholder is left:\n%s", got)
	}
	for _, want := range []string{"<h2>ENG: Setup &amp; Install</h2>", pageBody, "<p>Footer</p>", "Internal"} {
		if !strings.Contains(got, want) {
			t.Errorf("lacks %q:\n%s", want, got)
		}
	}
}

func TestTheEditorsMonospaceFormOfThePlaceholderIsUnderstood(t *testing.T) {
	for _, shell := range []string{
		`<p><code>content</code></p>`,
		`<p><code>{{content}}</code></p>`,
		`<table><tbody><tr><td>{{ content }}</td></tr></tbody></table>`,
	} {
		got, err := ApplyTemplate(shell, pageBody, "T")
		if err != nil || !strings.Contains(got, pageBody) {
			t.Errorf("%s -> %s, %v", shell, got, err)
		}
	}
}

func TestATemplateWithoutContentIsRefused(t *testing.T) {
	if _, err := ApplyTemplate("<p>Hello</p>", pageBody, "T"); err == nil || !strings.Contains(err.Error(), "{{content}}") {
		t.Fatalf("error: %v", err)
	}
	if got, err := ApplyTemplate("", pageBody, "T"); err != nil || got != pageBody {
		t.Fatalf("no template changed the body: %q, %v", got, err)
	}
}

func TestExtractContentGivesBackTheBodyWhatEverTheTemplateAddedAround(t *testing.T) {
	shell := `<p>Banner</p><p>{{content}}</p><p>Footer</p>`
	laidOut, _ := ApplyTemplate(shell, pageBody, "T")

	if got := ExtractContent(laidOut); got != pageBody {
		t.Fatalf("extracted:\n%s", got)
	}
}

func TestExtractContentSurvivesConfluenceWrappingTheAnchorsInParagraphs(t *testing.T) {
	// Confluence may re-save the page with each anchor macro in a paragraph of its own and a
	// macro id added.
	stored := `<p>Banner</p><p><ac:structured-macro ac:name="anchor" ac:schema-version="1" ac:macro-id="a1"><ac:parameter ac:name="">lm-content-start</ac:parameter></ac:structured-macro></p>` +
		pageBody + `<p><ac:structured-macro ac:name="anchor" ac:schema-version="1" ac:macro-id="a2"><ac:parameter ac:name="">lm-content-end</ac:parameter></ac:structured-macro></p><p>Footer</p>`

	if got := ExtractContent(stored); got != pageBody {
		t.Fatalf("extracted:\n%s", got)
	}
}

func TestAPageWithoutAnchorsIsReturnedWhole(t *testing.T) {
	if got := ExtractContent(pageBody); got != pageBody {
		t.Fatalf("got %q", got)
	}
}
