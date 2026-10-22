package storageformat

import (
	"strings"
	"testing"
)

const mermaidSVG = `<?xml version="1.0"?><svg id="d1" width="100%" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 800" style="max-width: 1600px;" role="graphics-document"><g/></svg>`

func TestSizedSVGCapsTheWidthAndKeepsTheProportions(t *testing.T) {
	got := sizedSVG(mermaidSVG, 900)

	for _, want := range []string{`<div style="max-width:900px;width:100%;">`, `width="100%" style="height:auto;"`, `viewBox="0 0 1600 800"`, `id="d1"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "<?xml") || strings.Contains(got, "max-width: 1600px") {
		t.Errorf("the prolog and Mermaid's own sizing must be dropped: %s", got)
	}
}

func TestSizedSVGNeverEnlargesAPictureBeyondItsDrawnSize(t *testing.T) {
	if got := sizedSVG(mermaidSVG, 3000); !strings.Contains(got, "max-width:1600px") {
		t.Errorf("a cap above the drawn width keeps the drawn width: %s", got)
	}
	if got := sizedSVG(mermaidSVG, 0); !strings.Contains(got, "max-width:1600px") {
		t.Errorf("no cap keeps the drawn width: %s", got)
	}
}

func TestSizedSVGWithoutAnySizeUsesTheCap(t *testing.T) {
	if got := sizedSVG(`<svg><g/></svg>`, 700); !strings.Contains(got, "max-width:700px") {
		t.Errorf("an SVG that does not say its size takes the cap: %s", got)
	}
	if got := sizedSVG(`not an svg`, 700); got != "not an svg" {
		t.Errorf("anything else is left alone: %s", got)
	}
}

func TestHTMLMacroModeShowsTheSVGAboveACollapsedSource(t *testing.T) {
	doc := Document{Blocks: []Block{Mermaid{Source: "graph TD; A-->B", SVG: mermaidSVG}}}
	got, err := Render(doc, Options{MermaidMode: MermaidHTMLMacro, DiagramMaxWidth: 800})
	if err != nil {
		t.Fatal(err)
	}
	html, code := strings.Index(got, `ac:name="html"`), strings.Index(got, `ac:name="code"`)
	if html < 0 || code < 0 || html > code || !strings.Contains(got, `<ac:parameter ac:name="collapse">true</ac:parameter>`) || strings.Contains(got, "ac:image") {
		t.Errorf("want an html macro, then a collapsed code macro, and no attachment: %s", got)
	}
}

func TestHTMLMacroModeFallsBackToTheCodeWhenNothingWasDrawn(t *testing.T) {
	got, err := Render(Document{Blocks: []Block{Mermaid{Source: "graph TD; A-->B"}}}, Options{MermaidMode: MermaidHTMLMacro})
	if err != nil || strings.Contains(got, `ac:name="html"`) || !strings.Contains(got, `ac:name="code"`) {
		t.Errorf("want the code macro alone, got %v %s", err, got)
	}
}
