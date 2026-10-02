package documentparsing

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuin/goldmark/ast"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current output")

func TestTitle(t *testing.T) {
	cases := []struct {
		name, path, content, title string
		fromHeading                bool
		order                      int
		ordered                    bool
	}{
		{"inline formatting dropped", "a.md", "# Hello *world*\n", "Hello world", true, 0, false},
		{"links, code spans, escapes and entities", "a.md", "# The `a*b` [API](x.md) \\*not\\* &amp; &#169;\n", "The a*b API *not* & ©", true, 0, false},
		{"image alt text and autolinks", "a.md", "# ![Logo](l.png) at <https://x.io>\n", "Logo at https://x.io", true, 0, false},
		{"raw HTML ignored", "a.md", "# A <span>B</span> C\n", "A B C", true, 0, false},
		{"setext H1", "a.md", "Hello\n=====\n", "Hello", true, 0, false},
		{"first H1, not the first heading", "a.md", "## Sub\n\n# Main\n\n# Second\n", "Main", true, 0, false},
		{"H1 in a block quote is content", "notes.md", "> # Quoted\n", "notes", false, 0, false},
		{"empty H1 falls through", "a.md", "#\n\n# Real\n", "Real", true, 0, false},
		{"missing H1 uses the file name", "docs/setup-guide.md", "Text only\n", "setup-guide", false, 0, false},
		{"numeric prefix with a dash", "01-intro.md", "Text\n", "intro", false, 1, true},
		{"numeric prefix with an underscore", "2_setup.md", "Text\n", "setup", false, 2, true},
		{"prefix kept as order even with an H1", "03-x.md", "# Explicit\n", "Explicit", true, 3, true},
		{"dotted name: last segment", "readme.02-setup.md", "Text\n", "setup", false, 2, true},
		{"a dot is nesting, not an order separator", "01.setup.md", "Text\n", "setup", false, 0, false},
		{"digits only are a name", "2024.md", "Text\n", "2024", false, 0, false},
		{"upper-case extension", "Guide.MD", "Text\n", "Guide", false, 0, false},
		{"front matter is not a heading", "fm.md", "---\ntitle: Not this\n---\nText\n", "fm", false, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document, err := ParseDocument(documentdiscovery.DocumentPath(tc.path), []byte(tc.content))
			if err != nil {
				t.Fatal(err)
			}
			if document.Title != tc.title || document.TitleFromHeading != tc.fromHeading {
				t.Fatalf("title %q (from heading %v), want %q (%v)", document.Title, document.TitleFromHeading, tc.title, tc.fromHeading)
			}
			if document.Order != tc.order || document.Ordered != tc.ordered {
				t.Fatalf("order %d (%v), want %d (%v)", document.Order, document.Ordered, tc.order, tc.ordered)
			}
		})
	}
}

func TestFrontMatterStaysOutOfTheTree(t *testing.T) {
	document, err := ParseDocument("fm.md", []byte("---\ntitle: X\n---\nText\n"))
	if err != nil {
		t.Fatal(err)
	}
	if first := document.AST.FirstChild(); first == nil || first.Kind() != ast.KindParagraph {
		t.Fatalf("front matter leaked into the tree as %v", first.Kind())
	}
}

func TestParseDocumentRejectsABrokenAnnotation(t *testing.T) {
	_, err := ParseDocument("docs/a.md", []byte("<!-- lore-master\npage-id: 1\n"))
	want := `docs/a.md: the lore-master annotation opened on line 1 is never closed with "-->"`
	if err == nil || err.Error() != want {
		t.Fatalf("error %v, want %q", err, want)
	}
}

// TestGolden renders a document carrying front matter, an annotation and every GFM
// extension, and compares the HTML: neither the front matter nor the annotation may
// reach the page, and each extension must be on.
func TestGolden(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "full.md"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := ParseDocument("full.md", content)
	if err != nil {
		t.Fatal(err)
	}
	if document.Title != "Getting started with lore" || document.Annotation == nil || document.Annotation.PageID != "42" {
		t.Fatalf("title %q, annotation %+v", document.Title, document.Annotation)
	}

	var html bytes.Buffer
	if err := markdown.Renderer().Render(&html, document.Body, document.AST); err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "full.golden.html")
	if *update {
		if err := os.WriteFile(golden, html.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(html.Bytes(), want) {
		t.Fatalf("rendered HTML differs from %s\n got: %s\nwant: %s", golden, html.Bytes(), want)
	}
}
