package documentdiscovery

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// DiscoverDocuments walks every root and returns the Markdown files a sync would read,
// sorted and de-duplicated across overlapping roots. Symbolic links are never followed
// (a link can leave the workspace or loop) and are reported as warnings instead, as are
// files whose paths differ only by case, which collide on Windows and macOS.
func DiscoverDocuments(ctx context.Context, options Options) (Discovery, error) {
	roots, err := normalisedRoots(options.Roots)
	if err != nil {
		return Discovery{}, err
	}
	rules := newIgnoreRules(options.WorkspaceRoot, options.Excludes, options.Includes, !options.IncludeGitignored)
	found := map[DocumentPath]bool{}
	var warnings []string

	for _, root := range roots {
		start := filepath.Join(options.WorkspaceRoot, filepath.FromSlash(root))
		walkErr := filepath.WalkDir(start, func(osPath string, entry fs.DirEntry, err error) error {
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
			if rel == "." {
				return nil
			}
			isDir := entry.IsDir()
			ignored, err := rules.ignored(rel, isDir)
			if err != nil {
				return err
			}
			switch {
			case ignored && isDir:
				return filepath.SkipDir
			case ignored, isDir:
				return nil
			case entry.Type()&fs.ModeSymlink != 0:
				if isMarkdown(rel) {
					warnings = append(warnings, fmt.Sprintf("%s is a symbolic link and was skipped", rel))
				}
			case entry.Type().IsRegular() && isMarkdown(rel):
				found[DocumentPath(rel)] = true
			}

			return nil
		})
		if walkErr != nil {
			return Discovery{}, fmt.Errorf("discover documents under %q: %w", root, walkErr)
		}
	}

	documents := make([]DocumentPath, 0, len(found))
	for document := range found {
		documents = append(documents, document)
	}
	slices.Sort(documents)
	warnings = append(warnings, caseCollisionWarnings(documents)...)

	return NewDiscovery(documents, warnings), nil
}

func normalisedRoots(roots []string) ([]string, error) {
	if len(roots) == 0 {
		return []string{"."}, nil
	}
	normalised := make([]string, 0, len(roots))
	for _, root := range roots {
		clean := path.Clean(filepath.ToSlash(root))
		if path.IsAbs(clean) || filepath.IsAbs(root) || clean == ".." || strings.HasPrefix(clean, "../") {
			return nil, fmt.Errorf("root %q must be a path inside the workspace", root)
		}
		normalised = append(normalised, clean)
	}

	return normalised, nil
}

func isMarkdown(rel string) bool {
	return strings.EqualFold(path.Ext(rel), ".md")
}

func caseCollisionWarnings(sorted []DocumentPath) []string {
	byFolded := map[string][]string{}
	var order []string
	for _, document := range sorted {
		folded := strings.ToLower(string(document))
		if _, seen := byFolded[folded]; !seen {
			order = append(order, folded)
		}
		byFolded[folded] = append(byFolded[folded], string(document))
	}
	var warnings []string
	for _, folded := range order {
		if paths := byFolded[folded]; len(paths) > 1 {
			warnings = append(warnings, fmt.Sprintf("%s differ only by case and collide on Windows and macOS", strings.Join(paths, ", ")))
		}
	}

	return warnings
}
