package csharpdocs

import (
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
)

var (
	// anchorLine is the <a name='...'></a> DefaultDocumentation puts before members, with the
	// blank line after it.
	anchorLine = regexp.MustCompile(`(?m)^<a name='[^']*'></a>\n\n?`)
	// pageLink is a link to another generated page: the target, an optional fragment, and the
	// tooltip title DefaultDocumentation adds.
	pageLink = regexp.MustCompile(`\]\(([^\s#)']+\.md)(?:#[^\s']*)?(?: '(?:[^'\\]|\\.)*')?\)`)
	// plainName is a file stem that is a dotted C# name and nothing else.
	plainName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
)

const assemblyPage = "index.md"

// convertPages turns DefaultDocumentation's flat folder of pages (Shop.Billing.Invoice.md) into
// pages nested by namespace under prefix: a namespace is the README of its folder, a type a file
// in its namespace's folder. Breadcrumbs and anchors are dropped, the top heading becomes an H1
// that is unique (a type's title is its full name) and links between pages are rewritten to the
// new paths. The assembly page is left out: the generator writes its own index.
func convertPages(raw map[string]string, prefix string) []generatedfile.File {
	names := make([]string, 0, len(raw))
	for name := range raw {
		if name != assemblyPage {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	bodies := make(map[string]string, len(names))
	namespaces := make(map[string]bool, len(names))
	targets := make(map[string]string, len(names))
	for _, name := range names {
		body := cleaned(raw[name])
		bodies[name] = body
		namespaces[name] = strings.HasSuffix(firstHeading(body), " Namespace")
		targets[name] = pagePath(prefix, name, namespaces[name])
	}

	files := make([]generatedfile.File, 0, len(names))
	for _, name := range names {
		body := retitled(bodies[name], strings.TrimSuffix(name, ".md"), namespaces[name])
		body = rewriteLinks(body, path.Dir(targets[name]), targets)
		files = append(files, generatedfile.File{Path: targets[name], Body: []byte(body)})
	}

	return files
}

// pagePath places a page by its dotted name: Shop.Billing is the README of Shop/Billing and
// Shop.Billing.Invoice the file Shop/Billing/Invoice.md.
func pagePath(prefix string, file string, isNamespace bool) string {
	segments := strings.Split(strings.TrimSuffix(file, ".md"), ".")
	if isNamespace {
		return path.Join(append(append([]string{prefix}, segments...), "README.md")...)
	}
	segments[len(segments)-1] += ".md"

	return path.Join(append([]string{prefix}, segments...)...)
}

// cleaned drops the breadcrumb lines above the first heading and the member anchors.
func cleaned(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(body, "\n")
	start := 0
	for start < len(lines) && !strings.HasPrefix(lines[start], "## ") {
		start++
	}
	if start == len(lines) {
		start = 0
	}
	body = strings.Join(lines[start:], "\n")
	body = anchorLine.ReplaceAllString(body, "")

	return strings.TrimSpace(body) + "\n"
}

func firstHeading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			return strings.TrimPrefix(line, "## ")
		}
	}

	return ""
}

// retitled makes the first heading the page's H1. A type's title is its full name followed by
// its kind ("Shop.Cart Class"), so two types of one name in different namespaces do not clash.
func retitled(body string, stem string, isNamespace bool) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		heading := strings.TrimPrefix(line, "## ")
		if !isNamespace && plainName.MatchString(stem) {
			if at := strings.LastIndex(heading, " "); at >= 0 {
				heading = stem + heading[at:]
			}
		}
		lines[i] = "# " + heading

		break
	}

	return strings.Join(lines, "\n")
}

// rewriteLinks points links to generated pages at their new place, relative to fromDir, without
// the tooltip or fragment. Links to anything else (the .NET documentation) are left alone.
func rewriteLinks(body string, fromDir string, targets map[string]string) string {
	return pageLink.ReplaceAllStringFunc(body, func(match string) string {
		target, found := targets[pageLink.FindStringSubmatch(match)[1]]
		if !found {
			return match
		}
		relative, err := filepath.Rel(filepath.FromSlash(fromDir), filepath.FromSlash(target))
		if err != nil {
			return match
		}

		return "](" + filepath.ToSlash(relative) + ")"
	})
}
