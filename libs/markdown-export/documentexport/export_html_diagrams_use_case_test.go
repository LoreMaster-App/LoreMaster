package documentexport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/markdown-export/browserexport"
	"lore-master/libs/markdown-export/exportmodel"
)

const diagramDocument = "# D\n\n```mermaid\ngraph TD\n  A[Start] --> B[Done]\n```\n\n- item\n\n  ```mermaid\n  graph TD\n    A[Start] --> B[Done]\n  ```\n\n```mermaid\nnot a diagram\n```\n"

func bundledMermaid(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "apps", "lore-master-vscode", "dist", "assets", "webview", "mermaid.min.js"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Skip("the Mermaid library is not built: " + path)
	}
	if _, err := browserexport.LocateBrowser(""); err != nil {
		t.Skip("no browser installed: " + err.Error())
	}

	return path
}

func TestMermaidSourcesAreListedOnceWhereverTheyAre(t *testing.T) {
	blocks := []exportmodel.Block{
		{Kind: exportmodel.CodeBlock, Lang: "mermaid", Text: "graph TD"},
		{Kind: exportmodel.CodeBlock, Lang: "go", Text: "not a diagram"},
		{Kind: exportmodel.QuoteBlock, Children: []exportmodel.Block{{Kind: exportmodel.CodeBlock, Lang: "mermaid", Text: "graph LR"}}},
		{Kind: exportmodel.ListBlock, Items: []exportmodel.Item{{Blocks: []exportmodel.Block{{Kind: exportmodel.CodeBlock, Lang: "mermaid", Text: "graph TD"}}}}},
	}

	got := mermaidSources(blocks)

	if len(got) != 2 || got[0] != "graph TD" || got[1] != "graph LR" {
		t.Fatalf("got %q", got)
	}
}

func TestWithoutALibraryDiagramsStayCodeAsTheyAlwaysWere(t *testing.T) {
	file, err := Export("d.md", []byte(diagramDocument), "html", Options{})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(file.Content), "<svg") || strings.Count(string(file.Content), `class="language-mermaid"`) != 3 {
		t.Fatalf("unexpected output:\n%s", file.Content)
	}
}

func TestHTMLDrawsDiagramsWhenThereIsALibraryAndABrowserAndKeepsABadOneAsCode(t *testing.T) {
	library := bundledMermaid(t)

	file, err := Export("d.md", []byte(diagramDocument), "html", Options{MermaidPath: library})
	if err != nil {
		t.Fatal(err)
	}
	page := string(file.Content)

	if strings.Count(page, `<div class="diagram">`) != 2 || strings.Count(page, "<svg") != 2 {
		t.Errorf("both copies of the valid diagram should be drawn: %d svg", strings.Count(page, "<svg"))
	}
	if strings.Count(page, `class="language-mermaid"`) != 1 || !strings.Contains(page, "not a diagram") {
		t.Errorf("the invalid diagram should stay as code")
	}
	if strings.Contains(page, "<script") {
		t.Error("the page needs no script to show a drawn diagram")
	}
}

func TestAPreviewIsLeftToDrawItsOwnDiagrams(t *testing.T) {
	library := bundledMermaid(t)

	file, err := Export("d.md", []byte(diagramDocument), "html", Options{MermaidPath: library, Preview: true})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(file.Content), "<svg") {
		t.Error("a preview should not be given drawn diagrams")
	}
}

func TestAMissingBrowserDoesNotFailTheExport(t *testing.T) {
	library := bundledMermaid(t)

	file, err := Export("d.md", []byte(diagramDocument), "html", Options{MermaidPath: library, BrowserPath: "/no/such/browser"})
	if err != nil || strings.Contains(string(file.Content), "<svg") {
		t.Fatalf("the export should fall back to code: %v", err)
	}
}
