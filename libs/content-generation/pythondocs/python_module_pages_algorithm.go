package pythondocs

import (
	"path"
	"sort"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
)

// modulePage is the documentation of one module as pydoc-markdown printed it.
type modulePage struct {
	Name string
	Body string
}

// splitModules cuts pydoc-markdown's single stream into one page per module. A module starts at
// a "# name" heading outside a code fence (members are "##" and deeper). The anchors the tool
// puts before every heading are dropped: they are HTML noise in a wiki page. Pages come back
// sorted by module name so the output does not depend on the tool's order.
func splitModules(markdown string) []modulePage {
	var pages []modulePage
	var current *modulePage
	var body []string
	flush := func() {
		if current != nil {
			current.Body = strings.TrimSpace(strings.Join(body, "\n")) + "\n"
			pages = append(pages, *current)
		}
		body = nil
	}

	inFence := false
	for _, line := range strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n") {
		line = strings.TrimRight(line, " \t")
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
		}
		if !inFence && isAnchor(line) {
			continue
		}
		if !inFence && strings.HasPrefix(line, "# ") {
			flush()
			current = &modulePage{Name: strings.TrimSpace(strings.TrimPrefix(line, "# "))}
		}
		if current != nil {
			body = append(body, line)
		}
	}
	flush()
	sort.Slice(pages, func(i, j int) bool { return pages[i].Name < pages[j].Name })

	return pages
}

func isAnchor(line string) bool {
	return strings.HasPrefix(line, `<a id="`) && strings.HasSuffix(line, `"></a>`)
}

// pagePath places a module in the folders of its package: a package is the README of its folder
// (shop/billing/README.md), a plain module a file beside its siblings (shop/cart.md).
func pagePath(prefix string, module string, isPackage bool) string {
	segments := strings.Split(module, ".")
	if isPackage {
		return path.Join(append(append([]string{prefix}, segments...), "README.md")...)
	}
	segments[len(segments)-1] += ".md"

	return path.Join(append([]string{prefix}, segments...)...)
}

// toFiles turns module pages into files under prefix; isPackage tells which modules are packages.
func toFiles(pages []modulePage, prefix string, isPackage func(module string) bool) []generatedfile.File {
	files := make([]generatedfile.File, 0, len(pages))
	for _, page := range pages {
		files = append(files, generatedfile.File{Path: pagePath(prefix, page.Name, isPackage(page.Name)), Body: []byte(page.Body)})
	}

	return files
}
