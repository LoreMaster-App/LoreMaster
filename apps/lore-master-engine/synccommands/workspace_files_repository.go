package synccommands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/documentation-sync/syncexecution"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// fileReader reads workspace files by their workspace-relative path, and never outside
// the workspace, whatever a path says.
func fileReader(root string) syncexecution.FileReader {
	return func(path documentdiscovery.DocumentPath) ([]byte, error) {
		full := filepath.Join(root, filepath.FromSlash(string(path)))
		relative, err := filepath.Rel(root, full)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("%s is outside the workspace", path)
		}

		return os.ReadFile(full)
	}
}

// loadDocuments discovers and parses the output's Markdown: every "markdown" content
// entry, each with its own roots and excludes, merged without repeats. A file that
// cannot be read or parsed is an error for the plan, not a reason to stop.
func loadDocuments(ctx context.Context, root string, output workspacesettings.Output) ([]documentparsing.MarkdownDocument, []string, []string, error) {
	var paths []documentdiscovery.DocumentPath
	var warnings, errors []string
	for _, content := range output.Content {
		if content.Type != "markdown" {
			continue
		}
		found, err := documentdiscovery.DiscoverDocuments(ctx, documentdiscovery.Options{WorkspaceRoot: root, Roots: content.Roots, Excludes: content.Excludes})
		if err != nil {
			return nil, nil, nil, err
		}
		warnings = append(warnings, found.Warnings...)
		for _, path := range found.Documents {
			if !slices.Contains(paths, path) {
				paths = append(paths, path)
			}
		}
	}

	read := fileReader(root)
	documents := make([]documentparsing.MarkdownDocument, 0, len(paths))
	for _, path := range paths {
		content, err := read(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))

			continue
		}
		document, err := documentparsing.ParseDocument(path, content)
		if err != nil {
			errors = append(errors, err.Error())

			continue
		}
		for _, warning := range document.Warnings {
			warnings = append(warnings, fmt.Sprintf("%s: %s", path, warning))
		}
		documents = append(documents, document)
	}

	return documents, warnings, errors, nil
}
