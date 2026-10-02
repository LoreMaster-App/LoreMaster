package documenttree

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

// MaxTitleLength is Confluence's limit on a page title, in characters.
const MaxTitleLength = 255

// PageTitles validates and returns every page's final title, "<prefix>: <title>" (just
// the title when the prefix is empty), trimmed. Confluence titles are unique per space,
// so a clash is an error, not a warning: the second page would fail mid-sync. Titles
// are compared case-insensitively, which is the safe side whichever way a Confluence
// edition compares them. Every problem is reported at once, each with its remedy.
func PageTitles(tree DocumentTree, prefix string) (map[documentdiscovery.DocumentPath]string, error) {
	prefix = strings.TrimSpace(prefix)
	titles := map[documentdiscovery.DocumentPath]string{}
	byFolded := map[string][]documentdiscovery.DocumentPath{}
	var tooLong []string

	tree.Walk(func(node *TreeNode, _ int) {
		title := strings.TrimSpace(node.Title())
		if prefix != "" {
			title = prefix + ": " + title
		}
		titles[node.Document.Path] = title
		folded := strings.ToLower(title)
		byFolded[folded] = append(byFolded[folded], node.Document.Path)
		if length := utf8.RuneCountInString(title); length > MaxTitleLength {
			tooLong = append(tooLong, fmt.Sprintf("  %s → %d characters", node.Document.Path, length))
		}
	})

	var clashes []string
	for _, paths := range byFolded {
		if len(paths) < 2 {
			continue
		}
		slices.Sort(paths)
		names := make([]string, len(paths))
		for i, documentPath := range paths {
			names[i] = string(documentPath)
		}
		clashes = append(clashes, fmt.Sprintf("  %s → %q", strings.Join(names, ", "), titles[paths[0]]))
	}
	slices.Sort(clashes)
	slices.Sort(tooLong)

	var problems []string
	if len(clashes) > 0 {
		problems = append(problems, "These documents would get the same page title, and Confluence allows a title only once per space:\n"+
			strings.Join(clashes, "\n")+
			"\nGive all but one of each a \"title:\" line in its lore-master annotation, or change its H1.")
	}
	if len(tooLong) > 0 {
		problems = append(problems, fmt.Sprintf("These page titles are longer than Confluence's %d characters:\n", MaxTitleLength)+
			strings.Join(tooLong, "\n")+
			"\nShorten the H1, set a shorter \"title:\" in the lore-master annotation, or choose a shorter title prefix.")
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(problems, "\n\n"))
	}

	return titles, nil
}
