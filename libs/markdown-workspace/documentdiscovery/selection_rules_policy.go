package documentdiscovery

import (
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// patternSet is a list of gitignore-syntax patterns that can also say how specific the
// entry that matched a path is: the number of path segments the entry names, so an entry for
// "docs/adr/" (2) is more specific than one for "docs/" (1), and a literal file path is the
// most specific of all.
type patternSet struct {
	all   *ignore.GitIgnore
	lines []*ignore.GitIgnore
}

func newPatternSet(patterns []string) *patternSet {
	set := &patternSet{all: ignore.CompileIgnoreLines(patterns...)}
	for _, pattern := range patterns {
		if strings.HasPrefix(pattern, "!") || strings.TrimSpace(pattern) == "" {
			continue
		}
		set.lines = append(set.lines, ignore.CompileIgnoreLines(pattern))
	}

	return set
}

// specificity is 0 when no entry of the set covers rel (a file, or a directory when isDir is
// set), and otherwise the specificity of the most specific entry that does. A folder entry
// covers everything below it, and a negated line takes a path out of the set as git does.
func (s *patternSet) specificity(rel string, isDir bool) int {
	if !coveredBy(s.all, rel, isDir) {
		return 0
	}
	deepest := 1
	for _, line := range s.lines {
		if depth := firstCoveringDepth(line, rel, isDir); depth > deepest {
			deepest = depth
		}
	}

	return deepest
}

// coveredBy reports whether rel, or any folder above it, matches rules.
func coveredBy(rules *ignore.GitIgnore, rel string, isDir bool) bool {
	return firstCoveringDepth(rules, rel, isDir) > 0
}

// firstCoveringDepth is the number of segments of the shallowest path (rel itself, or a folder
// above it) that rules match, or 0.
func firstCoveringDepth(rules *ignore.GitIgnore, rel string, isDir bool) int {
	parts := strings.Split(rel, "/")
	for i := 1; i < len(parts); i++ {
		if matched, how := rules.MatchesPathHow(strings.Join(parts[:i], "/") + "/"); matched && how != nil && !how.Negate {
			return i
		}
	}
	if matched, how := rules.MatchesPathHow(withDirectorySlash(rel, isDir)); matched && how != nil && !how.Negate {
		return len(parts)
	}

	return 0
}
