package filemoves

import (
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// ParentRef is the "parent:" line of a document's annotation.
type ParentRef struct {
	Line  int
	Text  string
	Value string
}

// Document is a Markdown file the plan may touch.
type Document struct {
	// Path is workspace-relative, '/'-separated.
	Path string
	// Parent is the annotation's parent line, nil when there is none.
	Parent *ParentRef
}

// PlanMoves works out the follow-up changes of moves over the workspace's documents.
// A moved file's dotted children (same directory, name starting with its stem and a dot)
// are renamed to keep nesting under it; every "parent:" that names a moved or renamed
// file, or sits in one, is rewritten so it still names the same document.
func PlanMoves(documents []Document, moves []Move) Plan {
	plan := Plan{Renames: []Rename{}, Edits: []LineEdit{}, Warnings: []string{}}
	sorted := slices.Clone(documents)
	slices.SortFunc(sorted, func(a, b Document) int { return strings.Compare(a.Path, b.Path) })

	existing := map[string]string{}
	for _, document := range sorted {
		existing[strings.ToLower(document.Path)] = document.Path
	}
	moved := func(p string) string {
		for _, move := range moves {
			switch {
			case p == move.From:
				return move.To
			case strings.HasPrefix(p, move.From+"/"):
				return move.To + p[len(move.From):]
			}
		}

		return p
	}

	derived := map[string]string{}
	taken := map[string]bool{}
	for _, document := range sorted {
		newPath := moved(document.Path)
		if newPath == document.Path || !isMarkdown(document.Path) || !isMarkdown(newPath) || isDirectoryIndex(document.Path) {
			continue
		}
		oldStem, newStem := stemOf(document.Path), stemOf(newPath)
		dir := path.Dir(document.Path)
		for _, child := range sorted {
			if child.Path == document.Path || path.Dir(child.Path) != dir || moved(child.Path) != child.Path {
				continue
			}
			base := path.Base(child.Path)
			if !isMarkdown(base) || len(base) <= len(oldStem)+1 || !strings.EqualFold(base[:len(oldStem)+1], oldStem+".") {
				continue
			}
			target := joinPath(path.Dir(newPath), newStem+base[len(oldStem):])
			folded := strings.ToLower(target)
			if _, clash := existing[folded]; clash && !strings.EqualFold(target, child.Path) {
				plan.Warnings = append(plan.Warnings, child.Path+" was not renamed to "+target+": that file exists")

				continue
			}
			if taken[folded] {
				plan.Warnings = append(plan.Warnings, child.Path+" was not renamed to "+target+": another file takes that name")

				continue
			}
			taken[folded] = true
			derived[child.Path] = target
		}
	}
	final := func(p string) string {
		if renamed, ok := derived[p]; ok {
			return renamed
		}

		return moved(p)
	}

	for from, to := range derived {
		plan.Renames = append(plan.Renames, Rename{From: from, To: to})
	}
	slices.SortFunc(plan.Renames, func(a, b Rename) int { return strings.Compare(a.From, b.From) })

	for _, document := range sorted {
		if edit, ok := parentEdit(document, existing, final); ok {
			plan.Edits = append(plan.Edits, edit)
		}
	}

	return plan
}

// parentEdit rewrites a document's parent line when it, or the document it names, moved.
func parentEdit(document Document, existing map[string]string, final func(string) string) (LineEdit, bool) {
	if document.Parent == nil || document.Parent.Value == "" {
		return LineEdit{}, false
	}
	value := document.Parent.Value
	absolute := strings.HasPrefix(value, "/")
	var target string
	if absolute {
		target = path.Clean(strings.TrimLeft(value, "/"))
	} else {
		target = path.Join(path.Dir(document.Path), value)
	}
	resolved, ok := existing[strings.ToLower(target)]
	if !ok {
		return LineEdit{}, false
	}
	newTarget, newDocument := final(resolved), final(document.Path)
	if newTarget == resolved && (absolute || path.Dir(newDocument) == path.Dir(document.Path)) {
		return LineEdit{}, false
	}

	var written string
	if absolute {
		written = "/" + newTarget
	} else {
		written = relativeTo(path.Dir(newDocument), newTarget)
		if strings.HasPrefix(value, "./") && !strings.HasPrefix(written, "../") {
			written = "./" + written
		}
	}
	if written == value {
		return LineEdit{}, false
	}

	text := document.Parent.Text
	colon := strings.Index(text, ":")
	if colon < 0 {
		return LineEdit{}, false
	}
	after := text[colon+1:]
	gap := after[:len(after)-len(strings.TrimLeft(after, " \t"))]

	return LineEdit{Path: document.Path, Line: document.Parent.Line, Text: text[:colon+1] + gap + written}, true
}

func isMarkdown(p string) bool {
	return strings.HasSuffix(strings.ToLower(p), ".md")
}

// isDirectoryIndex reports whether the file is a folder's README.md or index.md. Those already
// parent everything in their folder, so their dotted children stay where they are.
func isDirectoryIndex(p string) bool {
	switch strings.ToLower(path.Base(p)) {
	case "readme.md", "index.md":
		return true
	}

	return false
}

func stemOf(p string) string {
	base := path.Base(p)

	return base[:len(base)-len(path.Ext(base))]
}

func joinPath(dir, name string) string {
	if dir == "." {
		return name
	}

	return dir + "/" + name
}

func relativeTo(dir, target string) string {
	rel, err := filepath.Rel(filepath.FromSlash(dir), filepath.FromSlash(target))
	if err != nil {
		return target
	}

	return filepath.ToSlash(rel)
}
