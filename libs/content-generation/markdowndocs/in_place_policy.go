package markdowndocs

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
)

// IsInPlace reports whether the Markdown is already where it should be: the input is empty, or it
// names the output folder itself. The command that produced the pages wrote them there, so there
// is nothing to import and the pages are left as the command wrote them.
func IsInPlace(spec generatedfile.Spec) bool {
	output := folderOf(spec.Output)
	if output == "" {
		return false
	}
	selecting := 0
	for _, pattern := range spec.Input {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" || strings.HasPrefix(trimmed, "!") {
			continue
		}
		selecting++
		if folderOf(trimmed) != output {
			return false
		}
	}

	return selecting > 0 || spec.Type == "markdown-docs"
}

// InPlaceFiles lists the workspace-relative Markdown files under the output folder, sorted.
func InPlaceFiles(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) ([]string, error) {
	output := folderOf(spec.Output)
	root := filepath.Join(workspaceRoot, filepath.FromSlash(output))
	var found []string
	err := filepath.WalkDir(root, func(full string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if full == root {
				return nil
			}

			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if full != root && slices.Contains(skippedFolders, entry.Name()) {
				return filepath.SkipDir
			}

			return nil
		}
		if extension := strings.ToLower(filepath.Ext(full)); extension == ".md" || extension == ".markdown" {
			relative, relErr := filepath.Rel(workspaceRoot, full)
			if relErr != nil {
				return relErr
			}
			found = append(found, filepath.ToSlash(relative))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(found)

	return found, nil
}

// folderOf normalises a folder or a pattern naming one: no leading ./, no trailing slash or glob.
func folderOf(value string) string {
	clean := strings.TrimSpace(filepath.ToSlash(value))
	clean = strings.TrimSuffix(clean, "/**")
	clean = strings.TrimSuffix(clean, "/*")
	clean = strings.Trim(path.Clean(clean), "/")
	if clean == "." {
		return ""
	}

	return clean
}
