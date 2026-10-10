package documentdiscovery

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

const defaultLeftOutLimit = 500

// ExplainLeftOut walks the whole workspace and reports every Markdown file the scan does not
// read, with the first rule that leaves it out: the settings' ignore list, a content entry's
// excludes, a .gitignore, or being outside every root. Directories the scan never enters
// (node_modules, .git ...) are not walked and their files are not reported, because nobody
// expects them. Symbolic links are skipped, as the scan skips them.
func ExplainLeftOut(ctx context.Context, options LeftOutOptions) (LeftOutReport, error) {
	roots, err := normalisedRoots(options.Roots)
	if err != nil {
		return LeftOutReport{}, err
	}
	limit := options.Limit
	if limit <= 0 {
		limit = defaultLeftOutLimit
	}
	included := make(map[DocumentPath]bool, len(options.Included))
	for _, document := range options.Included {
		included[document] = true
	}

	matchers := leftOutMatchers{
		ignore:     ignore.CompileIgnoreLines(options.Ignore...),
		excludes:   ignore.CompileIgnoreLines(options.Excludes...),
		gitignores: newIgnoreRules(options.WorkspaceRoot, nil, nil, true),
		honourGit:  !options.IncludeGitignored,
	}

	var report LeftOutReport
	walkErr := filepath.WalkDir(options.WorkspaceRoot, func(osPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(options.WorkspaceRoot, osPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		switch {
		case rel == ".":
			return nil
		case entry.IsDir():
			if slices.Contains(DefaultExcludedDirectories, path.Base(rel)) {
				return filepath.SkipDir
			}

			return nil
		case entry.Type()&fs.ModeSymlink != 0, !entry.Type().IsRegular(), !isMarkdown(rel):
			return nil
		case included[DocumentPath(rel)]:
			return nil
		}

		left, ok, err := matchers.explain(rel, roots)
		if err != nil {
			return err
		}
		if ok {
			report.Total++
			if len(report.Documents) < limit {
				report.Documents = append(report.Documents, left)
			}
		}

		return nil
	})
	if walkErr != nil {
		return LeftOutReport{}, walkErr
	}
	slices.SortFunc(report.Documents, func(a, b LeftOut) int { return strings.Compare(string(a.Path), string(b.Path)) })

	return report, nil
}

type leftOutMatchers struct {
	ignore, excludes *ignore.GitIgnore
	gitignores             *ignoreRules
	honourGit              bool
}

// explain finds the first rule that leaves rel out; ok is false when none does, which means
// the file was not read for no reason this function knows (it is then not reported).
func (m leftOutMatchers) explain(rel string, roots []string) (LeftOut, bool, error) {
	if pattern, matched := matchesWithAncestors(m.ignore, rel); matched {
		return LeftOut{Path: DocumentPath(rel), Rule: RuleIgnore, Pattern: pattern}, true, nil
	}
	if pattern, matched := matchesWithAncestors(m.excludes, rel); matched {
		return LeftOut{Path: DocumentPath(rel), Rule: RuleExcludes, Pattern: pattern}, true, nil
	}
	if m.honourGit {
		left, matched, err := m.gitignored(rel)
		if err != nil || matched {
			return left, matched, err
		}
	}
	if !withinRoots(rel, roots) {
		return LeftOut{Path: DocumentPath(rel), Rule: RuleOutsideRoots}, true, nil
	}

	return LeftOut{}, false, nil
}

// gitignored checks every .gitignore from the file's directory up, as the scan does, and
// every ancestor directory of the file against it, since a rule may ignore a folder.
func (m leftOutMatchers) gitignored(rel string) (LeftOut, bool, error) {
	for dir := path.Dir(rel); ; dir = path.Dir(dir) {
		rules, err := m.gitignores.gitignoreIn(dir)
		if err != nil {
			return LeftOut{}, false, err
		}
		if rules != nil {
			if pattern, matched := matchesWithAncestors(rules, relativeTo(dir, rel)); matched {
				source := ".gitignore"
				if dir != "." {
					source = dir + "/.gitignore"
				}

				return LeftOut{Path: DocumentPath(rel), Rule: RuleGitignore, Pattern: pattern, Source: source}, true, nil
			}
		}
		if dir == "." {
			return LeftOut{}, false, nil
		}
	}
}

// matchesWithAncestors reports whether rel, or any directory above it, matches rules, and the
// pattern line that did.
func matchesWithAncestors(rules *ignore.GitIgnore, rel string) (string, bool) {
	parts := strings.Split(rel, "/")
	for i := 1; i < len(parts); i++ {
		directory := strings.Join(parts[:i], "/") + "/"
		if matched, how := rules.MatchesPathHow(directory); matched && how != nil && !how.Negate {
			return how.Line, true
		}
	}
	if matched, how := rules.MatchesPathHow(rel); matched && how != nil && !how.Negate {
		return how.Line, true
	}

	return "", false
}

func withinRoots(rel string, roots []string) bool {
	for _, root := range roots {
		if root == "." || rel == root || strings.HasPrefix(rel, root+"/") {
			return true
		}
	}

	return false
}
