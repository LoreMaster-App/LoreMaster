package documentloading

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// Loaded is an output's Markdown as read.
type Loaded struct {
	Documents []documentparsing.MarkdownDocument
	// Warnings are discovery and parse notes that do not stop anything.
	Warnings []string
	// Problems are files that could not be read or parsed; each is one to report, not a
	// reason to stop loading the others.
	Problems []string
}

// LoadOutputDocuments discovers and parses the Markdown of every "markdown" content entry
// of output. A file that cannot be read or parsed is a Problem, not an error: the error is
// for a scan that could not run at all (a root that does not exist, a cancelled context).
func LoadOutputDocuments(ctx context.Context, workspaceRoot string, output workspacesettings.Output, scope workspacesettings.DiscoveryScope) (Loaded, error) {
	var loaded Loaded
	var paths []documentdiscovery.DocumentPath
	for _, content := range output.Content {
		if content.Type != "markdown" {
			continue
		}
		excludes, includes := scope.ScanFor(output, content)
		found, err := documentdiscovery.DiscoverDocuments(ctx, documentdiscovery.Options{
			WorkspaceRoot: workspaceRoot, Roots: content.Roots,
			Excludes: excludes, Includes: includes, IncludeGitignored: !scope.SkipGitignored,
		})
		if err != nil {
			return Loaded{}, err
		}
		loaded.Warnings = append(loaded.Warnings, found.Warnings...)
		for _, path := range found.Documents {
			if !slices.Contains(paths, path) {
				paths = append(paths, path)
			}
		}
	}

	loaded.Documents = make([]documentparsing.MarkdownDocument, 0, len(paths))
	for _, path := range paths {
		content, err := readInWorkspace(workspaceRoot, path)
		if err != nil {
			loaded.Problems = append(loaded.Problems, fmt.Sprintf("%s: %v", path, err))

			continue
		}
		document, err := documentparsing.ParseDocument(path, content)
		if err != nil {
			loaded.Problems = append(loaded.Problems, err.Error())

			continue
		}
		for _, warning := range document.Warnings {
			loaded.Warnings = append(loaded.Warnings, fmt.Sprintf("%s: %s", path, warning))
		}
		loaded.Documents = append(loaded.Documents, document)
	}

	return loaded, nil
}

// readInWorkspace reads a workspace-relative file, never reaching outside the workspace
// whatever the path says.
func readInWorkspace(workspaceRoot string, path documentdiscovery.DocumentPath) ([]byte, error) {
	full := filepath.Join(workspaceRoot, filepath.FromSlash(string(path)))
	relative, err := filepath.Rel(workspaceRoot, full)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("%s is outside the workspace", path)
	}

	return os.ReadFile(full)
}
