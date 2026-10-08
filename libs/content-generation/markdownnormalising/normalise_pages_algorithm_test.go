package markdownnormalising

import (
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func page(path string, body string) generatedfile.File {
	return generatedfile.File{Path: path, Body: []byte(body)}
}

func bodies(pages []generatedfile.File) map[string]string {
	out := map[string]string{}
	for _, p := range pages {
		out[p.Path] = string(p.Body)
	}

	return out
}

func TestNormaliseCleansLineEndingsFrontMatterStampsAndWhitespace(t *testing.T) {
	input := []generatedfile.File{page("README.md", "\ufeff---\r\ntitle: x\r\n---\r\n# Project  \r\n\r\n\r\n\r\nText with trailing space   \r\n\r\nDefined in: [src/a.ts:12](https://example.com/blob/abc123/src/a.ts#L12)\r\n\r\nEnd\r\n")}

	pages, warnings := NormalisePages(input, Options{DropLinePrefixes: []string{"Defined in:"}})

	if got, want := bodies(pages)["README.md"], "# Project\n\nText with trailing space\n\nEnd\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings %q", warnings)
	}
}

func TestNormaliseKeepsWhatLooksLikeAStampedLineInsideACodeBlock(t *testing.T) {
	input := []generatedfile.File{page("README.md", "# P\n\n```text\nDefined in: not a stamp\n```\n\nDefined in: a stamp\n")}

	pages, _ := NormalisePages(input, Options{DropLinePrefixes: []string{"Defined in:"}})

	if got, want := bodies(pages)["README.md"], "# P\n\n```text\nDefined in: not a stamp\n```\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormaliseAddsAHeadingToAPageThatHasNone(t *testing.T) {
	pages, warnings := NormalisePages([]generatedfile.File{page("README.md", "# Home\n"), page("classes/Widget.md", "Just text\n"), page("guide/README.md", "More text\n")}, Options{})

	got := bodies(pages)
	if got["classes/Widget.md"] != "# Widget\n\nJust text\n" || got["guide/README.md"] != "# guide\n\nMore text\n" {
		t.Fatalf("pages %v", got)
	}
	if len(warnings) != 2 || !strings.Contains(warnings[0], "classes/Widget.md had no top heading") {
		t.Fatalf("warnings %q", warnings)
	}
}

func TestNormaliseMakesRepeatedTitlesUniqueByFolderThenNumber(t *testing.T) {
	pages, warnings := NormalisePages([]generatedfile.File{
		page("README.md", "# Library\n"),
		page("a/Widget.md", "# Class: Widget\n"),
		page("b/Widget.md", "# Class: Widget\n"),
		page("c/Widget.md", "# class: widget\n"),
		page("d/Library.md", "# library\n"),
	}, Options{IndexTitle: "API"})

	got := bodies(pages)
	for path, want := range map[string]string{
		"a/Widget.md": "# Class: Widget\n",
		"b/Widget.md": "# Class: Widget (b)\n",
		"c/Widget.md": "# class: widget (c)\n",
		"d/Library.md": "# library (d)\n",
	} {
		if got[path] != want {
			t.Errorf("%s: got %q, want %q", path, got[path], want)
		}
	}
	if len(warnings) != 3 {
		t.Fatalf("warnings %q", warnings)
	}

	titles := map[string]bool{}
	for _, p := range pages {
		heading := strings.ToLower(strings.SplitN(string(p.Body), "\n", 2)[0])
		if titles[heading] {
			t.Fatalf("duplicate title %q", heading)
		}
		titles[heading] = true
	}
}

func TestNormaliseKeepsTheIndexTitleForTheIndexAndTellsAnotherPageThatUsesItApart(t *testing.T) {
	pages, _ := NormalisePages([]generatedfile.File{page("README.md", "# API reference\n"), page("guide/Intro.md", "# API reference\n")}, Options{IndexTitle: "API reference"})

	got := bodies(pages)
	if got["README.md"] != "# API reference\n" || got["guide/Intro.md"] != "# API reference (guide)\n" {
		t.Fatalf("pages %v", got)
	}
}

func TestNormaliseWritesAnIndexWhenTheToolWroteNone(t *testing.T) {
	pages, _ := NormalisePages([]generatedfile.File{page("modules/b.md", "# B [beta]\n"), page("modules/a.md", "# A\n")}, Options{IndexTitle: "Docs"})

	if pages[0].Path != "README.md" {
		t.Fatalf("pages %v", bodies(pages))
	}
	if got, want := string(pages[0].Body), "# Docs\n\n- [A](modules/a.md)\n- [B \\[beta\\]](modules/b.md)\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormaliseIsDeterministicWhateverTheOrderOfThePages(t *testing.T) {
	a := []generatedfile.File{page("x/A.md", "# T\n"), page("y/B.md", "# T\n"), page("README.md", "# R\n")}
	b := []generatedfile.File{a[2], a[1], a[0]}

	first, _ := NormalisePages(a, Options{})
	second, _ := NormalisePages(b, Options{})

	if len(first) != len(second) {
		t.Fatal("different page counts")
	}
	for i := range first {
		if first[i].Path != second[i].Path || string(first[i].Body) != string(second[i].Body) {
			t.Fatalf("page %d differs: %q vs %q", i, first[i].Body, second[i].Body)
		}
	}
}
