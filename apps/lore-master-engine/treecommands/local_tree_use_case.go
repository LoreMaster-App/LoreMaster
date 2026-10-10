package treecommands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
)

// localTree is every Markdown file of the workspace as the tree it forms, whatever any
// output leaves out: each file says whether git ignores it and which outputs would sync it.
// It is the editor's "Local" view, the one place a file excluded everywhere is still shown.
func localTree(ctx context.Context, root string, settings workspacesettings.Settings) (rpcprotocol.WorkspaceTreeResult, error) {
	result := rpcprotocol.WorkspaceTreeResult{Nodes: []rpcprotocol.TreeNode{}}

	all, err := documentdiscovery.DiscoverDocuments(ctx, documentdiscovery.Options{WorkspaceRoot: root, IncludeGitignored: true})
	if err != nil {
		return result, err
	}
	tracked, err := documentdiscovery.DiscoverDocuments(ctx, documentdiscovery.Options{WorkspaceRoot: root})
	if err != nil {
		return result, err
	}
	result.Warnings = append(result.Warnings, all.Warnings...)

	scope := settings.DiscoveryScope()
	syncedTo := map[documentdiscovery.DocumentPath][]int{}
	for index, output := range settings.Outputs {
		for _, content := range output.Content {
			if content.Type != "markdown" {
				continue
			}
			excludes, includes := scope.ScanFor(output, content)
			found, err := documentdiscovery.DiscoverDocuments(ctx, documentdiscovery.Options{
				WorkspaceRoot: root, Roots: content.Roots, Excludes: excludes, Includes: includes,
				IncludeGitignored: !scope.SkipGitignored,
			})
			if err != nil {
				return result, err
			}
			for _, path := range found.Documents {
				if !slices.Contains(syncedTo[path], index) {
					syncedTo[path] = append(syncedTo[path], index)
				}
			}
		}
	}

	documents := make([]documentparsing.MarkdownDocument, 0, len(all.Documents))
	for _, path := range all.Documents {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(string(path))))
		if err != nil {
			result.Problems = append(result.Problems, fmt.Sprintf("%s: %v", path, err))

			continue
		}
		document, err := documentparsing.ParseDocument(path, content)
		if err != nil {
			result.Problems = append(result.Problems, err.Error())

			continue
		}
		documents = append(documents, document)
	}

	tree, err := documenttree.BuildTree(documents)
	if err != nil {
		result.Problems = append(result.Problems, err.Error())

		return result, nil
	}
	result.Warnings = append(result.Warnings, tree.Warnings...)
	tree.Walk(func(node *documenttree.TreeNode, depth int) {
		entry := rpcprotocol.TreeNode{
			Path: string(node.Document.Path), Title: node.Title(), PageTitle: node.Title(),
			Rule: string(node.Decision.Rule), Depth: depth, Warnings: node.Decision.Warnings,
			GitIgnored: !slices.Contains(tracked.Documents, node.Document.Path),
			SyncedTo:   syncedTo[node.Document.Path],
		}
		if node.Decision.Parent != nil {
			entry.Parent = string(*node.Decision.Parent)
		}
		result.Nodes = append(result.Nodes, entry)
	})

	return result, nil
}
