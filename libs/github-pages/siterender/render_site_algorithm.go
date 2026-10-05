package siterender

import (
	"bytes"
	"encoding/base64"
	"html/template"
	"regexp"
	"strings"

	"lore-master/libs/markdown-workspace/documenttree"
)

var pageTemplate = template.Must(template.New("page").Parse(pageTemplateSource))

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
}

// navItem is one entry in a page's navigation sidebar, with an href relative to the page
// being rendered.
type navItem struct {
	Label  string
	Href   string
	Depth  int
	Active bool
}

// pageData is what the page template needs to render one document.
type pageData struct {
	Title          string
	SiteTitle      string
	AssetsPrefix   string
	MarkdownBase64 string
	Nav            []navItem
	RenderScript   template.JS
}

// GenerateSite renders a document tree as a static HTML site: one HTML page per Markdown
// document (the raw Markdown embedded and rendered in the browser), a shared navigation
// sidebar, the default stylesheet under assets/, an index.html entry pointing at the home
// page, and a .nojekyll marker so GitHub Pages serves the files untouched. It is pure: no
// file is written and no network is touched. siteTitle labels the sidebar.
func GenerateSite(siteTitle string, tree documenttree.DocumentTree) ([]SiteFile, error) {
	pages := collectPages(tree)

	files := make([]SiteFile, 0, len(pages)+3)
	for _, current := range pages {
		content, err := renderPage(siteTitle, current.title, current.body, current.sitePath, current.sitePath, pages)
		if err != nil {
			return nil, err
		}
		files = append(files, SiteFile{Path: current.sitePath, Content: content})
	}

	if len(pages) > 0 {
		home := pages[homeIndex(pages)]
		index, err := renderPage(siteTitle, home.title, home.body, "index.html", home.sitePath, pages)
		if err != nil {
			return nil, err
		}
		files = append(files, SiteFile{Path: "index.html", Content: index})
	}

	files = append(files,
		SiteFile{Path: "assets/lore-master.css", Content: []byte(defaultCSS)},
		SiteFile{Path: ".nojekyll", Content: []byte{}},
	)

	return files, nil
}

// collectPages flattens the tree into pages in parents-first order, mapping each
// document's path to its '.html' site path and carrying its nesting depth.
func collectPages(tree documenttree.DocumentTree) []page {
	var pages []page
	tree.Walk(func(node *documenttree.TreeNode, depth int) {
		if node.Document == nil {
			return
		}
		pages = append(pages, page{
			sitePath: htmlPath(string(node.Document.Path)),
			title:    node.Title(),
			depth:    depth,
			body:     node.Document.Body,
		})
	})

	return pages
}

// renderPage renders one document's HTML. selfSitePath is the page being written (its
// position decides the relative hrefs); activeSitePath is the page the nav should mark as
// current (the same page, except for index.html, which mirrors the home page).
func renderPage(siteTitle, title string, body []byte, selfSitePath, activeSitePath string, pages []page) ([]byte, error) {
	prefix := relativePrefix(selfSitePath)
	nav := make([]navItem, 0, len(pages))
	for _, target := range pages {
		nav = append(nav, navItem{
			Label:  target.title,
			Href:   prefix + target.sitePath,
			Depth:  target.depth,
			Active: target.sitePath == activeSitePath,
		})
	}

	data := pageData{
		Title:          title,
		SiteTitle:      siteTitle,
		AssetsPrefix:   prefix,
		MarkdownBase64: base64.StdEncoding.EncodeToString(body),
		Nav:            nav,
		RenderScript:   template.JS(renderScript),
	}

	var out bytes.Buffer
	if err := pageTemplate.Execute(&out, data); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
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
