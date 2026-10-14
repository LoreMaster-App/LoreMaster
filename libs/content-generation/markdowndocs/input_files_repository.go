package markdowndocs

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/content-generation/inputselection"
)

// skippedFolders are never searched: dependencies and version control.
var skippedFolders = []string{"node_modules", ".git"}

// hasSelectingPattern reports whether at least one pattern selects (does not start with "!").
func hasSelectingPattern(patterns []string) bool {
	return slices.ContainsFunc(patterns, func(pattern string) bool {
		trimmed := strings.TrimSpace(pattern)

		return trimmed != "" && !strings.HasPrefix(trimmed, "!")
	})
}

// findDocuments lists the workspace-relative paths ('/'-separated, sorted) of the Markdown files
// the patterns select, never looking inside the generator's own output folder.
func findDocuments(ctx context.Context, workspaceRoot string, patterns []string, output string) ([]string, error) {
	selector := inputselection.NewSelector(patterns, nil)
	output = strings.Trim(filepath.ToSlash(output), "/")

	var found []string
	err := filepath.WalkDir(workspaceRoot, func(full string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, err := filepath.Rel(workspaceRoot, full)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if full != workspaceRoot && (slices.Contains(skippedFolders, entry.Name()) || (output != "" && relative == output)) {
				return filepath.SkipDir
			}

			return nil
		}
		switch strings.ToLower(filepath.Ext(relative)) {
		case ".md", ".markdown":
			if selector.Selects(relative, false) {
				found = append(found, relative)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(found)

	return found, nil
}
