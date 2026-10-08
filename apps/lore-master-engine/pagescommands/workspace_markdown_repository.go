package pagescommands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// loadMarkdown discovers and parses the output's Markdown: every "markdown" content entry,
// each with its own roots and excludes plus the scope's ignore list, merged without repeats. A file that cannot be read
// or parsed is an error to report, not a reason to stop.
func loadMarkdown(ctx context.Context, root string, output workspacesettings.Output, scope workspacesettings.DiscoveryScope) ([]documentparsing.MarkdownDocument, []string, []string, error) {
	var paths []documentdiscovery.DocumentPath
	var warnings, errs []string
	for _, content := range output.Content {
		if content.Type != "markdown" {
			continue
		}
		found, err := documentdiscovery.DiscoverDocuments(ctx, documentdiscovery.Options{WorkspaceRoot: root, Roots: content.Roots, Excludes: scope.ExcludesFor(content), IncludeGitignored: !scope.SkipGitignored})
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

	documents := make([]documentparsing.MarkdownDocument, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(string(path))))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))

			continue
		}
		document, err := documentparsing.ParseDocument(path, content)
		if err != nil {
			errs = append(errs, err.Error())

			continue
		}
		for _, warning := range document.Warnings {
			warnings = append(warnings, fmt.Sprintf("%s: %s", path, warning))
		}
		documents = append(documents, document)
	}

	return documents, warnings, errs, nil
}
