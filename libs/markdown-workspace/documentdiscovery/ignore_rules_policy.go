package documentdiscovery

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"slices"

	ignore "github.com/sabhiram/go-gitignore"
)

// ignoreRules decides whether a workspace-relative path is left out of the scan. Each
// .gitignore is matched against paths relative to its own directory, as git does, and
// is read once, lazily, the first time a path below it is checked.
type ignoreRules struct {
	workspaceRoot string
	excludes      *ignore.GitIgnore
	excludeSet    *patternSet
	includeSet    *patternSet
	gitignored    bool
	gitignores    map[string]*ignore.GitIgnore
}

func newIgnoreRules(workspaceRoot string, excludes []string, includes []string, honourGitignore bool) *ignoreRules {
	rules := &ignoreRules{
		workspaceRoot: workspaceRoot,
		excludes:      ignore.CompileIgnoreLines(excludes...),
		gitignored:    honourGitignore,
		gitignores:    map[string]*ignore.GitIgnore{},
	}
	if len(includes) > 0 {
		rules.excludeSet, rules.includeSet = newPatternSet(excludes), newPatternSet(includes)
	}

	return rules
}

// ignored reports whether rel ('/'-separated, relative to the workspace root) is excluded.
// Directories must be checked with isDir set: gitignore's trailing-slash patterns only
// match a directory, and go-gitignore only sees one when the path ends in '/'.
func (r *ignoreRules) ignored(rel string, isDir bool) (bool, error) {
	if isDir && slices.Contains(DefaultExcludedDirectories, path.Base(rel)) {
		return true, nil
	}
	switch {
	case r.includeSet == nil:
		if r.excludes.MatchesPath(withDirectorySlash(rel, isDir)) {
			return true, nil
		}
	case !isDir && r.leftOutByLists(rel):
		return true, nil
	}
	if !r.gitignored {
		return false, nil
	}
	for dir := path.Dir(rel); ; dir = path.Dir(dir) {
		rules, err := r.gitignoreIn(dir)
		if err != nil {
			return false, err
		}
		if rules != nil && rules.MatchesPath(withDirectorySlash(relativeTo(dir, rel), isDir)) {
			return true, nil
		}
		if dir == "." {
			return false, nil
		}
	}
}

// leftOutByLists decides a file when the settings carry includes: an exclude leaves it out
// unless an include at least as specific takes it back. A folder is never decided here, so
// the walk goes into an excluded folder to find what is included below it.
func (r *ignoreRules) leftOutByLists(rel string) bool {
	excluded := r.excludeSet.specificity(rel, false)
	if excluded == 0 {
		return false
	}

	return r.includeSet.specificity(rel, false) < excluded
}

func (r *ignoreRules) gitignoreIn(dir string) (*ignore.GitIgnore, error) {
	if rules, seen := r.gitignores[dir]; seen {
		return rules, nil
	}
	rules, err := ignore.CompileIgnoreFile(filepath.Join(r.workspaceRoot, filepath.FromSlash(dir), ".gitignore"))
	if errors.Is(err, fs.ErrNotExist) {
		rules, err = nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.gitignores[dir] = rules

	return rules, nil
}

func relativeTo(dir string, rel string) string {
	if dir == "." {
		return rel
	}

	return rel[len(dir)+1:]
}

func withDirectorySlash(rel string, isDir bool) string {
	if isDir {
		return rel + "/"
	}

	return rel
}
