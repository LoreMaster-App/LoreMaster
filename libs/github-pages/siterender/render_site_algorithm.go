package siterender

import (
	"encoding/base64"
	"html"
	"regexp"
	"strings"

	"lore-master/libs/markdown-workspace/childrenmarker"
	"lore-master/libs/markdown-workspace/documenttree"
)

// markdownExtensions are stripped from a document path to form its '.html' site path.
var markdownExtensions = []string{".markdown", ".mdown", ".mkd", ".md"}

// orderPrefix is a file name's leading numeric ordering prefix ("01-", "2_"), ignored when
// deciding whether a page is the site's home ("readme"/"index").
var orderPrefix = regexp.MustCompile(`^\d+[-_. ]`)

// page is one document's place in the generated site: its '/'-separated site path, its
// title and nesting depth (for the nav), and the raw Markdown to embed.
type page struct {
	sitePath string
	title    string
	depth    int
	body     []byte
	// documentPath is the Markdown file the page came from, workspace-relative.
	documentPath string
}

// navItem is one entry in a page's navigation sidebar, with an href relative to the page
// being rendered.
type navItem struct {
	Label  string
	Href   string
	Depth  int
	Active bool
}

// GenerateSite renders a document tree as a static HTML site: one HTML page per Markdown
// document (the raw Markdown embedded and rendered in the browser), a shared navigation
// sidebar, the default stylesheet under assets/, an index.html entry pointing at the home
// page, and a .nojekyll marker so GitHub Pages serves the files untouched. It is pure: no
// file is written and no network is touched. siteTitle labels the sidebar.
func GenerateSite(siteTitle string, tree documenttree.DocumentTree) ([]SiteFile, error) {
	return GenerateSiteWith(siteTitle, tree, nil)
}

// GenerateSiteWith is GenerateSite with a transform applied to each document's Markdown before
// it is embedded (a document's path and its body in, the body to embed out): how a diagram
// language the browser cannot draw is turned into an image at build time. nil changes nothing.
func GenerateSiteWith(siteTitle string, tree documenttree.DocumentTree, transform func(path string, body []byte) []byte) ([]SiteFile, error) {
	chosen, _ := BuiltinTemplate("default")

	return GenerateSiteTemplated(siteTitle, tree, transform, chosen)
}

// GenerateSiteTemplated is GenerateSiteWith with the pages laid out by chosen.
func GenerateSiteTemplated(siteTitle string, tree documenttree.DocumentTree, transform func(path string, body []byte) []byte, chosen Template) ([]SiteFile, error) {
	pages := collectPages(tree)
	if transform != nil {
		for i := range pages {
			pages[i].body = transform(pages[i].documentPath, pages[i].body)
		}
	}
	if chosen.Headless {
		return headlessFiles(siteTitle, pages)
	}

	files := make([]SiteFile, 0, len(pages)+3)
	for i, current := range pages {
		files = append(files, SiteFile{Path: current.sitePath, Content: renderPage(siteTitle, chosen, i, pages)})
	}

	// index.html redirects to the home page's own file rather than copying it, so the home
	// page is served where its relative links resolve (a copy at the root would break them).
	if len(pages) > 0 {
		files = append(files, SiteFile{Path: "index.html", Content: indexRedirect(pages[homeIndex(pages)].sitePath)})
	}

	files = append(files,
		SiteFile{Path: "assets/lore-master.css", Content: []byte(contentCSS + chosen.LayoutCSS)},
		SiteFile{Path: ".nojekyll", Content: []byte{}},
	)
	files = append(files, chosen.Assets...)

	return files, nil
}

// collectPages flattens the tree into pages in parents-first order, mapping each
// document's path to its '.html' site path and carrying its nesting depth.
func collectPages(tree documenttree.DocumentTree) []page {
	var pages []page
	expanded := childrenmarker.ExpandTree(tree)
	tree.Walk(func(node *documenttree.TreeNode, depth int) {
		if node.Document == nil {
			return
		}
		body := node.Document.Body
		if withChildren, ok := expanded[node.Document.Path]; ok {
			body = withChildren
		}
		pages = append(pages, page{
			sitePath: htmlPath(string(node.Document.Path)),
			title:    node.Title(),
			depth:    depth,
			body:     body,

			documentPath: string(node.Document.Path),
		})
	})

	return pages
}

// renderPage renders one document's HTML into the template's shell: its own content, and a
// nav and breadcrumbs whose hrefs are relative to this page's position in the tree.
func renderPage(siteTitle string, chosen Template, index int, pages []page) []byte {
	current := pages[index]
	prefix := relativePrefix(current.sitePath)
	nav := make([]navItem, 0, len(pages))
	for _, target := range pages {
		nav = append(nav, navItem{
			Label:  target.title,
			Href:   prefix + target.sitePath,
			Depth:  target.depth,
			Active: target.sitePath == current.sitePath,
		})
	}

	var ancestors []navItem
	for want, j := current.depth-1, index-1; want >= 0 && j >= 0; j-- {
		if pages[j].depth == want {
			ancestors = append([]navItem{nav[j]}, ancestors...)
			want--
		}
	}

	// The Markdown rides in a data attribute, not a <script>: an attribute value is
	// HTML-escaped and base64's alphabet has nothing HTML-special, so it survives byte for byte.
	content := `<article id="lm-markdown" class="markdown-body"></article>` + "\n" +
		`<div id="lm-source" data-markdown="` + base64.StdEncoding.EncodeToString(current.body) + `" hidden></div>` + "\n" +
		`<script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>` + "\n" +
		`<script src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"></script>` + "\n" +
		"<script>" + renderScript + "</script>"
	values := map[string]string{
		"content":     content,
		"title":       html.EscapeString(current.title),
		"site_title":  html.EscapeString(siteTitle),
		"nav":         navHTML(nav),
		"breadcrumbs": breadcrumbsHTML(ancestors),
		"root":        prefix,
		"style":       `<link rel="stylesheet" href="` + prefix + `assets/lore-master.css">`,
		"source":      html.EscapeString(current.documentPath),
	}

	return []byte(fillShell(chosen.Shell, values))
}

// indexRedirect is the site root: a small page that sends the browser to the home page's own
// file, so the home content is served where its relative links resolve.
func indexRedirect(homeSitePath string) []byte {
	target := html.EscapeString(homeSitePath)

	return []byte("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n" +
		"<meta http-equiv=\"refresh\" content=\"0; url=" + target + "\">\n" +
		"<link rel=\"canonical\" href=\"" + target + "\">\n<title>Redirecting…</title>\n</head>\n" +
		"<body>\n<p><a href=\"" + target + "\">Continue to the documentation</a></p>\n</body>\n</html>\n")
}

// homeIndex picks the site's entry page: the first top-level "readme" or "index", else the
// first top-level page, else the first page.
func homeIndex(pages []page) int {
	for i, p := range pages {
		if p.depth == 0 && isHomeName(p.sitePath) {
			return i
		}
	}
	for i, p := range pages {
		if p.depth == 0 {
			return i
		}
	}

	return 0
}

// isHomeName reports whether a site path's base name (minus any ordering prefix and the
// .html extension) is "readme" or "index".
func isHomeName(sitePath string) bool {
	base := sitePath
	if slash := strings.LastIndex(base, "/"); slash >= 0 {
		base = base[slash+1:]
	}
	base = strings.TrimSuffix(base, ".html")
	base = orderPrefix.ReplaceAllString(base, "")

	return strings.EqualFold(base, "readme") || strings.EqualFold(base, "index")
}

// htmlPath turns a '/'-separated document path into its site path by replacing the
// Markdown extension with .html; directories are preserved.
func htmlPath(documentPath string) string {
	lower := strings.ToLower(documentPath)
	for _, ext := range markdownExtensions {
		if strings.HasSuffix(lower, ext) {
			return documentPath[:len(documentPath)-len(ext)] + ".html"
		}
	}

	return documentPath + ".html"
}

// relativePrefix is the "../" climb from a page back to the site root, so a page in a
// sub-directory can reach root-relative assets and sibling pages.
func relativePrefix(sitePath string) string {
	return strings.Repeat("../", strings.Count(sitePath, "/"))
}
