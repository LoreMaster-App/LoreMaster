package openapidocs

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/content-generation/inputselection"
)

// defaultInput selects the usual names of an OpenAPI description, anywhere.
var defaultInput = []string{
	"**/openapi.yaml", "**/openapi.yml", "**/openapi.json",
	"**/swagger.yaml", "**/swagger.yml", "**/swagger.json",
	"**/*.openapi.yaml", "**/*.openapi.yml", "**/*.openapi.json",
}

// skippedFolders are never searched: dependencies and version control, never the project's
// own API descriptions.
var skippedFolders = []string{"node_modules", ".git", "vendor"}

// findDescriptions lists the workspace-relative paths ('/'-separated, sorted) of the YAML and
// JSON files the patterns select: gitignore syntax, with "!" leaving out what it matches, and
// the usual OpenAPI file names when nothing selects.
func findDescriptions(ctx context.Context, workspaceRoot string, patterns []string) ([]string, error) {
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
		switch strings.ToLower(filepath.Ext(relative)) {
		case ".yaml", ".yml", ".json":
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
