package wikirender

import (
	"path"
	"regexp"
	"strconv"
	"strings"

	"lore-master/libs/github-pages/siterender"
	"lore-master/libs/markdown-workspace/documenttree"
)

const (
	homePageName = "Home"
	sidebarFile  = "_Sidebar.md"
)

// orderPrefix is a file name's leading numeric ordering prefix ("01-", "2_"), ignored when
// deciding whether a page is the wiki's home ("readme"/"index").
var orderPrefix = regexp.MustCompile(`^\d+[-_. ]`)

// inlineLink matches the target of an inline Markdown link or image: `](target)` or
// `](target "title")`. A target holding spaces or angle brackets is left alone.
var inlineLink = regexp.MustCompile(`\]\(([^)\s<>]+)((?:\s+"[^"]*")?)\)`)

// invalidNameChars are replaced in a page name: they have a meaning in a wiki address or
// a file name.
var invalidNameChars = strings.NewReplacer(
	"/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-", "#", "-", "%", "-",
	",", "", "'", "", "’", "",
)

// wikiPage is one document's place in the wiki.
type wikiPage struct {
	source string
	name   string
	title  string
	depth  int
	body   []byte
}

// GenerateWiki renders a document tree as the files of a GitHub wiki: one Markdown page per
// document named after its title (the wiki's address for it), the home page as Home.md, and a
// _Sidebar.md that mirrors the tree. Links between documents and to local images and files are
// rewritten to where the wiki serves them. It is pure: nothing is written.
func GenerateWiki(tree documenttree.DocumentTree) []siterender.SiteFile {
	pages := collectPages(tree)
	byPath := make(map[string]string, len(pages))
	for _, page := range pages {
		byPath[page.source] = page.name
	}

	files := make([]siterender.SiteFile, 0, len(pages)+1)
	for _, page := range pages {
		files = append(files, siterender.SiteFile{Path: page.name + ".md", Content: rewriteLinks(page, byPath)})
	}
	if len(pages) > 0 {
		files = append(files, siterender.SiteFile{Path: sidebarFile, Content: sidebar(pages)})
	}

	return files
}

// collectPages lists the documents parents-first and gives each its unique page name.
func collectPages(tree documenttree.DocumentTree) []wikiPage {
	var pages []wikiPage
	tree.Walk(func(node *documenttree.TreeNode, depth int) {
		if node.Document == nil {
			return
		}
		pages = append(pages, wikiPage{
			source: string(node.Document.Path), title: node.Title(), depth: depth, body: node.Document.Body,
		})
	})
	if len(pages) == 0 {
		return nil
	}

	home := homeIndex(pages)
	taken := map[string]bool{strings.ToLower(homePageName): true}
	for i := range pages {
		if i == home {
			pages[i].name = homePageName

			continue
		}
		base := pageName(pages[i].title)
		name := base
		for n := 2; taken[strings.ToLower(name)]; n++ {
			name = base + "-" + strconv.Itoa(n)
		}
		taken[strings.ToLower(name)] = true
		pages[i].name = name
	}

	return pages
}

// homeIndex is the page that becomes Home: a top-level readme or index, else the first
// top-level page.
func homeIndex(pages []wikiPage) int {
	for i, page := range pages {
		if page.depth == 0 && isHomeName(page.source) {
			return i
		}
	}
	for i, page := range pages {
		if page.depth == 0 {
			return i
		}
	}

	return 0
}

func isHomeName(source string) bool {
	base := path.Base(source)
	base = strings.TrimSuffix(base, path.Ext(base))
	base = orderPrefix.ReplaceAllString(base, "")

	return strings.EqualFold(base, "readme") || strings.EqualFold(base, "index")
}

// pageName turns a title into a wiki page name: spaces become hyphens, characters with a
// meaning in an address are replaced, and an empty result becomes "Page".
func pageName(title string) string {
	name := invalidNameChars.Replace(strings.TrimSpace(title))
	name = strings.Join(strings.Fields(name), "-")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-.")
	if name == "" {
		return "Page"
	}

	return name
}

// sidebar is the navigation list shown on every wiki page.
func sidebar(pages []wikiPage) []byte {
	escape := strings.NewReplacer("[", "\\[", "]", "\\]")
	var out strings.Builder
	for _, page := range pages {
		out.WriteString(strings.Repeat("  ", page.depth))
		out.WriteString("- [" + escape.Replace(page.title) + "](" + page.name + ")\n")
	}

	return []byte(out.String())
}

// rewriteLinks points a page's relative links at the wiki: a link to another document becomes
// that page's name (keeping its #anchor), and a link to any other local file becomes its
// workspace-relative path, which is where the build publishes it. Fenced code is untouched.
func rewriteLinks(page wikiPage, byPath map[string]string) []byte {
	dir := path.Dir(page.source)
	lines := strings.Split(string(page.body), "\n")
	fenced := false
	fence := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if marker, ok := fenceMarker(trimmed); ok {
			switch {
			case !fenced:
				fenced, fence = true, marker
			case strings.HasPrefix(trimmed, fence):
				fenced = false
			}

			continue
		}
		if fenced {
			continue
		}
		lines[i] = inlineLink.ReplaceAllStringFunc(line, func(match string) string {
			parts := inlineLink.FindStringSubmatch(match)

			return "](" + rewriteTarget(parts[1], dir, byPath) + parts[2] + ")"
		})
	}

	return []byte(strings.Join(lines, "\n"))
}

func fenceMarker(trimmed string) (string, bool) {
	for _, marker := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, marker) {
			return marker, true
		}
	}

	return "", false
}

func rewriteTarget(target, dir string, byPath map[string]string) string {
	if target == "" || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "/") || strings.Contains(target, ":") {
		return target
	}
	local, fragment := target, ""
	if cut := strings.IndexAny(target, "#?"); cut >= 0 {
		local, fragment = target[:cut], target[cut:]
	}
	resolved := path.Clean(path.Join(dir, local))
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return target
	}
	if name, ok := byPath[resolved]; ok {
		return name + fragment
	}

	return resolved + fragment
}
