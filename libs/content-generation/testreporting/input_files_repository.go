package testreporting

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// defaultInput selects the reports the common runners write: JUnit-style names anywhere.
var defaultInput = []string{"**/junit*.xml", "**/TEST-*.xml", "**/*junit.xml"}

// skippedFolders are never searched: dependencies and version control, never reports.
var skippedFolders = []string{"node_modules", ".git"}

// findReports lists the workspace-relative paths ('/'-separated, sorted) of the files the
// patterns select. Patterns use gitignore syntax, so "**/junit*.xml" and "reports/" work.
func findReports(ctx context.Context, workspaceRoot string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		patterns = defaultInput
	}
	matcher := ignore.CompileIgnoreLines(patterns...)

	var found []string
	err := filepath.WalkDir(workspaceRoot, func(full string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if slices.Contains(skippedFolders, entry.Name()) && full != workspaceRoot {
				return filepath.SkipDir
			}

			return nil
		}
		relative, err := filepath.Rel(workspaceRoot, full)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if strings.EqualFold(filepath.Ext(relative), ".xml") && matcher.MatchesPath(relative) {
			found = append(found, relative)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(found)

	return found, nil
}
