package browserexport

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const page = "<!doctype html><html><head><meta charset=\"utf-8\"><title>T</title></head><body><h1>Hello</h1><p>Some text.</p></body></html>"

// browserOrSkip runs a test that needs a real browser only where there is one.
func browserOrSkip(t *testing.T) string {
	t.Helper()
	browser, err := LocateBrowser("")
	if err != nil {
		t.Skip("no browser installed: " + err.Error())
	}

	return browser
}

func TestHeightIsReadBackFromTheDump(t *testing.T) {
	height, found := pageHeight(`<head><meta name="lore-master-height" content="1234"></head>`)
	if !found || height != 1234 {
		t.Fatalf("got %d, %v", height, found)
	}
	if _, found := pageHeight("<head></head>"); found {
		t.Fatal("a page without the probe has a height")
	}
}

func TestPageStyleAndSizeFallBack(t *testing.T) {
	if validPageSize("letter") != "Letter" || validPageSize("nonsense") != "A4" || validPageSize("") != "A4" {
		t.Fatal("page sizes are not normalised")
	}
	if got := withHeadBefore(page, pageStyle("A4")); !strings.Contains(got, "@page{size:A4") || !strings.Contains(got, "</head>") {
		t.Fatalf("style not injected: %s", got)
	}
}

func TestPDFIsPrintedByTheBrowser(t *testing.T) {
	browser := browserOrSkip(t)

	data, err := RenderPDF(page, Options{BrowserPath: browser})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %q", data[:min(len(data), 20)])
	}
}

func TestPNGCapturesTheWholePageAtTheWidthAsked(t *testing.T) {
	browser := browserOrSkip(t)
	tall := strings.Replace(page, "<p>Some text.</p>", strings.Repeat("<p>Line of text.</p>", 200), 1)

	data, err := RenderPNG(tall, Options{BrowserPath: browser, Width: 700})
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != 700 {
		t.Errorf("width = %d, want 700", config.Width)
	}
	if config.Height < 2000 {
		t.Errorf("height = %d: only part of a tall page was captured", config.Height)
	}
}

func TestWithoutABrowserTheErrorSaysWhatToDo(t *testing.T) {
	if _, err := RenderPDF(page, Options{BrowserPath: "/no/such/browser"}); err == nil || !strings.Contains(err.Error(), "browser") {
		t.Fatalf("got %v", err)
	}
}

// bundledMermaid is the library the editor extension ships, which exists once the extension has been built.
func bundledMermaid(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "apps", "lore-master-vscode", "dist", "assets", "webview", "mermaid.min.js"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Skip("the Mermaid library is not built: " + path)
	}

	return path
}

const diagramPage = "<!doctype html><html><head><meta charset=\"utf-8\"></head><body><h1>D</h1>" +
	"<pre><code class=\"language-mermaid\">graph TD\n  A[Start] --&gt; B[Done]</code></pre>" +
	"<pre><code class=\"language-mermaid\">not a diagram</code></pre><p>After.</p></body></html>"

func TestDiagramsAreOnlyAddedWhenThereAreAnyAndALibraryIsGiven(t *testing.T) {
	if withDiagrams(diagramPage, "") != diagramPage || withDiagrams(page, "/m.js") != page {
		t.Fatal("a page was changed without a reason")
	}
	drawn := withDiagrams(diagramPage, "/lib/mermaid.min.js")
	if !strings.Contains(drawn, `src="file:///lib/mermaid.min.js"`) || strings.Index(drawn, "mermaid.min.js") > strings.LastIndex(drawn, "</body>") {
		t.Fatalf("library not placed: %s", drawn)
	}
	if got := waitForDiagrams(diagramPage, "/m.js"); len(got) != 1 || waitForDiagrams(page, "/m.js") != nil {
		t.Fatalf("waiting is wrong: %v", got)
	}
}

func TestPNGDrawsDiagramsAndTheTallerPageIsCaptured(t *testing.T) {
	browser := browserOrSkip(t)
	library := bundledMermaid(t)

	without, err := RenderPNG(diagramPage, Options{BrowserPath: browser, Width: 700})
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := RenderPNG(diagramPage, Options{BrowserPath: browser, Width: 700, MermaidPath: library})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := png.DecodeConfig(bytes.NewReader(without))
	after, _ := png.DecodeConfig(bytes.NewReader(drawn))

	if after.Height <= before.Height {
		t.Errorf("a drawn diagram should make the page taller: %d without, %d with", before.Height, after.Height)
	}
}

func TestPDFWithDiagramsIsStillAPDF(t *testing.T) {
	browser := browserOrSkip(t)
	library := bundledMermaid(t)

	data, err := RenderPDF(diagramPage, Options{BrowserPath: browser, MermaidPath: library})
	if err != nil || !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("%v", err)
	}
}
