package testreporting

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/content-generation/inputselection"
)

// defaultInput selects the reports the common runners write: JUnit-style names anywhere.
var defaultInput = []string{"**/junit*.xml", "**/TEST-*.xml", "**/*junit.xml"}

// skippedFolders are never searched: dependencies and version control, never reports.
var skippedFolders = []string{"node_modules", ".git"}

// findReports lists the workspace-relative paths ('/'-separated, sorted) of the files the
// patterns select: gitignore syntax, so "**/junit*.xml" and "reports/" work, and a pattern
// starting with "!" leaves out what it matches. With none selecting, the usual report names apply.
func findReports(ctx context.Context, workspaceRoot string, patterns []string) ([]string, error) {
	selector := inputselection.NewSelector(patterns, defaultInput)

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
		if strings.EqualFold(filepath.Ext(relative), ".xml") && selector.Selects(relative, false) {
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
