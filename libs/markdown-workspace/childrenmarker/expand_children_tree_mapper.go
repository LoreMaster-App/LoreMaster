package childrenmarker

import (
	"path"
	"strings"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documenttree"
)

// ExpandTree returns the documents whose Markdown holds a children marker, each with the
// marker lines replaced by a bulleted list of the page's child pages as Markdown links, relative
// to the page, so a site or a wiki shows what Confluence's children macro shows. Documents
// without a marker are not in the result. A marker inside a fenced code block is left alone.
func ExpandTree(tree documenttree.DocumentTree) map[documentdiscovery.DocumentPath][]byte {
	expanded := map[documentdiscovery.DocumentPath][]byte{}
	tree.Walk(func(node *documenttree.TreeNode, _ int) {
		if node.Document == nil || !strings.Contains(strings.ToLower(string(node.Document.Body)), "[children") {
			return
		}
		body, changed := expandBody(string(node.Document.Body), func(options Options) string {
			return list(node, options)
		})
		if changed {
			expanded[node.Document.Path] = []byte(body)
		}
	})

	return expanded
}

// expandBody replaces each marker line outside fenced code with replace(options).
func expandBody(body string, replace func(Options) string) (string, bool) {
	lines := strings.Split(body, "\n")
	fence := ""
	changed := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if opening := fenceOf(trimmed); opening != "" {
			switch {
			case fence == "":
				fence = opening
			case strings.HasPrefix(trimmed, fence):
				fence = ""
			}

			continue
		}
		if fence != "" {
			continue
		}
		if options, ok := Parse(trimmed); ok {
			lines[index] = replace(options)
			changed = true
		}
	}

	return strings.Join(lines, "\n"), changed
}

// fenceOf is the fence characters opening or closing a fenced block on this line, or "".
func fenceOf(trimmed string) string {
	for _, mark := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, mark) {
			count := len(trimmed) - len(strings.TrimLeft(trimmed, mark[:1]))

			return strings.Repeat(mark[:1], count)
		}
	}

	return ""
}

// list renders the page's descendants as nested bullets, to the depth the options ask for.
func list(node *documenttree.TreeNode, options Options) string {
	var lines []string
	var add func(parent *documenttree.TreeNode, level int)
	add = func(parent *documenttree.TreeNode, level int) {
		if options.Levels() != 0 && level > options.Levels() {
			return
		}
		for _, child := range parent.Children {
			if child.Document == nil {
				continue
			}
			lines = append(lines, strings.Repeat("  ", level-1)+"- ["+escapeLabel(child.Title())+"]("+relativeLink(string(node.Document.Path), string(child.Document.Path))+")")
			add(child, level+1)
		}
	}
	add(node, 1)

	return strings.Join(lines, "\n")
}

func escapeLabel(label string) string {
	return strings.NewReplacer("[", `\[`, "]", `\]`).Replace(label)
}

// relativeLink is the path of target as seen from the document at from, '/'-separated, with
// spaces encoded so it stays one Markdown link destination.
func relativeLink(from string, target string) string {
	base := path.Dir(from)
	segments := strings.Split(path.Clean(target), "/")
	prefix := strings.Split(path.Clean(base), "/")
	if base == "." {
		prefix = nil
	}
	shared := 0
	for shared < len(prefix) && shared < len(segments)-1 && prefix[shared] == segments[shared] {
		shared++
	}
	parts := make([]string, 0, len(prefix)-shared+len(segments)-shared)
	for range prefix[shared:] {
		parts = append(parts, "..")
	}
	parts = append(parts, segments[shared:]...)

	return strings.ReplaceAll(strings.Join(parts, "/"), " ", "%20")
}
