package documentloading

import (
	"context"
	"slices"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
)

// ExplainOutputLeftOut says which Markdown files of the workspace output did not read and why:
// the settings' ignore list, a content entry's excludes, a .gitignore, or no root containing
// them. It is a separate walk of the whole workspace, so callers that only need the documents
// never pay for it.
func ExplainOutputLeftOut(ctx context.Context, workspaceRoot string, output workspacesettings.Output, scope workspacesettings.DiscoveryScope, loaded Loaded) (documentdiscovery.LeftOutReport, error) {
	var roots, excludes []string
	for _, content := range output.Content {
		if content.Type != "markdown" {
			continue
		}
		roots = append(roots, content.Roots...)
		for _, pattern := range content.Excludes {
			if !slices.Contains(excludes, pattern) {
				excludes = append(excludes, pattern)
			}
		}
	}

	for _, pattern := range output.Exclude {
		if !slices.Contains(excludes, pattern) {
			excludes = append(excludes, pattern)
		}
	}

	included := make([]documentdiscovery.DocumentPath, 0, len(loaded.Documents))
	for _, document := range loaded.Documents {
		included = append(included, document.Path)
	}

	return documentdiscovery.ExplainLeftOut(ctx, documentdiscovery.LeftOutOptions{
		WorkspaceRoot: workspaceRoot, Roots: roots,
		Ignore: scope.Ignore, Excludes: excludes,
		IncludeGitignored: !scope.SkipGitignored,
		Included:          included,
	})
}
