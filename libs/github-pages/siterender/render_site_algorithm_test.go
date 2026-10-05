package siterender

import (
	"encoding/base64"
	"strings"
	"testing"

	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
)

// sampleTree is a home page with one nested child and one page in a sub-directory, which
// together exercise nav nesting, home detection and relative paths.
func sampleTree() documenttree.DocumentTree {
	home := &documenttree.TreeNode{Document: &documentparsing.MarkdownDocument{Path: "readme.md", Title: "Home", Body: []byte("# Home\n\nWelcome.")}}
	guide := &documenttree.TreeNode{Document: &documentparsing.MarkdownDocument{Path: "readme.guide.md", Title: "Guide", Body: []byte("# Guide\n")}}
	home.Children = []*documenttree.TreeNode{guide}
	setup := &documenttree.TreeNode{Document: &documentparsing.MarkdownDocument{Path: "docs/setup.md", Title: "Setup", Body: []byte("# Setup\n")}}

	return documenttree.DocumentTree{Root: &documenttree.TreeNode{Children: []*documenttree.TreeNode{home, setup}}}
}

func fileByPath(files []SiteFile, path string) (SiteFile, bool) {
	for _, file := range files {
		if file.Path == path {
			return file, true
		}
	}

	return SiteFile{}, false
}

func TestGenerateSiteEmitsOnePagePerDocumentPlusEntryAndAssets(t *testing.T) {
	files, err := GenerateSite("My Project", sampleTree())
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"readme.html", "readme.guide.html", "docs/setup.html", "index.html", "assets/lore-master.css", ".nojekyll"} {
		if _, ok := fileByPath(files, want); !ok {
			t.Errorf("missing site file %q", want)
		}
	}
	if len(files) != 6 {
		t.Fatalf("want 6 files, got %d: %v", len(files), paths(files))
	}
}

func TestGenerateSiteEmbedsTheRawMarkdownUnconverted(t *testing.T) {
	files, _ := GenerateSite("My Project", sampleTree())
	page, _ := fileByPath(files, "readme.html")

	// The base64 must sit in a data attribute, verbatim: inside a <script> html/template would
	// JS-escape it (turning '/' into '\/'), which breaks atob in the browser.
	wantB64 := base64.StdEncoding.EncodeToString([]byte("# Home\n\nWelcome."))
	if !strings.Contains(string(page.Content), `data-markdown="`+wantB64+`"`) {
		t.Fatalf("readme.html should embed the raw Markdown as a base64 data attribute; it did not")
	}
	// The home page's bytes are rendered in the browser, not server-side: the HTML itself
	// must not contain the converted heading.
	if strings.Contains(string(page.Content), "<h1>Home</h1>") {
		t.Fatalf("the Markdown must not be converted on our side")
	}
}

func TestGenerateSiteLinksAssetsAndSiblingsRelativeToEachPage(t *testing.T) {
	files, _ := GenerateSite("My Project", sampleTree())

	root, _ := fileByPath(files, "readme.html")
	if !strings.Contains(string(root.Content), `href="assets/lore-master.css"`) {
		t.Errorf("a root page should link assets with no climb")
	}

	nested, _ := fileByPath(files, "docs/setup.html")
	if !strings.Contains(string(nested.Content), `href="../assets/lore-master.css"`) {
		t.Errorf("a sub-directory page should climb once to assets")
	}
	if !strings.Contains(string(nested.Content), `href="../readme.html"`) {
		t.Errorf("a sub-directory page should link a root sibling with one climb")
	}
}

func TestGenerateSiteIndexRedirectsToTheHomePage(t *testing.T) {
	files, _ := GenerateSite("My Project", sampleTree())
	index, _ := fileByPath(files, "index.html")
	html := string(index.Content)

	// index.html redirects to the home page's own file rather than copying it, so the home
	// page's relative links resolve where the page actually lives.
	if !strings.Contains(html, `content="0; url=readme.html"`) {
		t.Errorf("index.html should redirect to the home page's file")
	}
	if !strings.Contains(html, `href="readme.html"`) {
		t.Errorf("index.html should offer a link to the home page")
	}
	homeB64 := base64.StdEncoding.EncodeToString([]byte("# Home\n\nWelcome."))
	if strings.Contains(html, homeB64) {
		t.Errorf("index.html should not duplicate the home page's content")
	}
}

func TestGenerateSiteConvertsInternalMarkdownLinks(t *testing.T) {
	files, _ := GenerateSite("My Project", sampleTree())
	page, _ := fileByPath(files, "readme.html")

	// The render script rewrites a relative .md link to .html, but leaves external links and
	// anchors alone.
	script := string(page.Content)
	if !strings.Contains(script, `.replace(/\.(md|markdown)(#.*)?$/i, '.html$2')`) {
		t.Errorf("the page should rewrite internal Markdown links to .html")
	}
}

func TestGenerateSiteCarriesTitleAndNavLabels(t *testing.T) {
	files, _ := GenerateSite("My Project", sampleTree())
	page, _ := fileByPath(files, "readme.html")
	html := string(page.Content)

	if !strings.Contains(html, "<title>Home</title>") {
		t.Errorf("the page title should be the document title")
	}
	for _, label := range []string{">Home</a>", ">Guide</a>", ">Setup</a>"} {
		if !strings.Contains(html, label) {
			t.Errorf("nav should list %q", label)
		}
	}
	if !strings.Contains(html, "My Project") {
		t.Errorf("the sidebar should show the site title")
	}
}

func TestGenerateSiteEmptyTreeStillShipsAssets(t *testing.T) {
	files, err := GenerateSite("Empty", documenttree.DocumentTree{Root: &documenttree.TreeNode{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fileByPath(files, "index.html"); ok {
		t.Errorf("an empty tree has no home, so no index.html")
	}
	if _, ok := fileByPath(files, "assets/lore-master.css"); !ok {
		t.Errorf("the stylesheet should ship even for an empty tree")
	}
}

func paths(files []SiteFile) []string {
	out := make([]string, len(files))
	for i, file := range files {
		out[i] = file.Path
	}

	return out
}
