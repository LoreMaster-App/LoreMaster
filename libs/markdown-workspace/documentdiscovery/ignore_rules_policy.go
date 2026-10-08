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
	gitignored    bool
	gitignores    map[string]*ignore.GitIgnore
}

func newIgnoreRules(workspaceRoot string, excludes []string, honourGitignore bool) *ignoreRules {
	return &ignoreRules{
		workspaceRoot: workspaceRoot,
		excludes:      ignore.CompileIgnoreLines(excludes...),
		gitignored:    honourGitignore,
		gitignores:    map[string]*ignore.GitIgnore{},
	}
}

// ignored reports whether rel ('/'-separated, relative to the workspace root) is excluded.
// Directories must be checked with isDir set: gitignore's trailing-slash patterns only
// match a directory, and go-gitignore only sees one when the path ends in '/'.
func (r *ignoreRules) ignored(rel string, isDir bool) (bool, error) {
	if isDir && slices.Contains(DefaultExcludedDirectories, path.Base(rel)) {
		return true, nil
	}
	if r.excludes.MatchesPath(withDirectorySlash(rel, isDir)) {
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
