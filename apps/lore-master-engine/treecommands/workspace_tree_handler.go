package treecommands

import (
	"context"
	"fmt"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/documentloading"
	"lore-master/libs/documentation-sync/syncplanning"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documenttree"
)

// WorkspaceTree handles workspace/tree. Settings that are not valid are an
// invalid-settings error naming each problem; files that cannot be read are reported in
// the result so the rest of the tree still shows.
func WorkspaceTree() rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.WorkspaceTreeParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(params.WorkspaceRoot) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "workspaceRoot must be an absolute path, got %q", params.WorkspaceRoot)
		}
		settings, err := workspacesettings.LoadSettings(params.WorkspaceRoot)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}
		outputs := settings.Settings.Outputs
		if params.Output < 0 || params.Output >= len(outputs) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "output %d does not exist; %s has %d", params.Output, workspacesettings.FileName, len(outputs))
		}
		output := outputs[params.Output]

		loaded, err := documentloading.LoadOutputDocuments(ctx, params.WorkspaceRoot, output, settings.Settings.DiscoveryScope())
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}
		result := rpcprotocol.WorkspaceTreeResult{Nodes: []rpcprotocol.TreeNode{}, Warnings: loaded.Warnings, Problems: loaded.Problems}
		leftOut, err := documentloading.ExplainOutputLeftOut(ctx, params.WorkspaceRoot, output, settings.Settings.DiscoveryScope(), loaded)
		if err != nil {
			result.Warnings = append(result.Warnings, "could not list the files left out: "+err.Error())
		}
		for _, left := range leftOut.Documents {
			result.LeftOut = append(result.LeftOut, rpcprotocol.LeftOutFile{Path: string(left.Path), Rule: string(left.Rule), Pattern: left.Pattern, Source: left.Source})
		}
		result.LeftOutTotal = leftOut.Total

		tree, err := documenttree.BuildTree(loaded.Documents)
		if err != nil {
			result.Problems = append(result.Problems, err.Error())

			return result, nil
		}
		titles, titleErr := documenttree.PageTitles(tree, output.TitlePrefix)
		if titleErr != nil {
			result.Warnings = append(result.Warnings, titleErr.Error())
		}
		result.Warnings = append(result.Warnings, tree.Warnings...)

		tracks := output.Platform == "confluence"
		tree.Walk(func(node *documenttree.TreeNode, depth int) {
			entry := rpcprotocol.TreeNode{
				Path: string(node.Document.Path), Title: node.Title(), PageTitle: titles[node.Document.Path],
				Rule: string(node.Decision.Rule), Depth: depth, Warnings: node.Decision.Warnings,
			}
			if node.Decision.Parent != nil {
				entry.Parent = string(*node.Decision.Parent)
			}
			if tracks {
				entry.Status = string(syncplanning.LocalStatusOf(*node.Document, output))
				if annotation := node.Document.Annotation; annotation != nil && entry.Status != rpcprotocol.TreeStatusNew {
					entry.PageID = annotation.PageID
				}
			}
			result.Nodes = append(result.Nodes, entry)
		})
		if len(result.Nodes) == 0 && len(result.Problems) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("no Markdown files found for output %d", params.Output))
		}

		return result, nil
	}
}
