package inputselection

import (
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// Selector says whether a workspace-relative path is one a generator should read.
type Selector struct {
	include *ignore.GitIgnore
	exclude *ignore.GitIgnore
}

// NewSelector builds a selector from gitignore-style patterns. A pattern selects what it
// matches; a pattern starting with "!" leaves out what it matches, whatever selected it. When
// no pattern selects anything — there are none, or only "!" patterns — defaults select
// instead; an empty defaults means every path.
func NewSelector(patterns []string, defaults []string) Selector {
	var include, exclude []string
	for _, pattern := range patterns {
		if rest, negated := strings.CutPrefix(pattern, "!"); negated {
			exclude = append(exclude, rest)
		} else {
			include = append(include, pattern)
		}
	}
	if len(include) == 0 {
		include = defaults
	}

	selector := Selector{}
	if len(include) > 0 {
		selector.include = ignore.CompileIgnoreLines(include...)
	}
	if len(exclude) > 0 {
		selector.exclude = ignore.CompileIgnoreLines(exclude...)
	}

	return selector
}

// Selects reports whether the path is read. path is '/'-separated and relative to the
// workspace; isDir says it names a folder, which matters to patterns ending in "/".
func (s Selector) Selects(path string, isDir bool) bool {
	if isDir {
		path += "/"
	}

	return (s.include == nil || s.include.MatchesPath(path)) && (s.exclude == nil || !s.exclude.MatchesPath(path))
}
