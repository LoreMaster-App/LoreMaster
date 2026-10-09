package pagescommands

import (
	"context"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/libs/documentation-sync/documentloading"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/github-pages/siterender"
	"lore-master/libs/markdown-workspace/documenttree"
)

// renderedSite is one github-pages output turned into site files, before anything is
// published or written.
type renderedSite struct {
	Output   workspacesettings.Output
	Files    []siterender.SiteFile
	Warnings []string
	// Problems are errors in the Markdown; when there are any, Files is empty and nothing
	// should be published.
	Problems []string
}

// renderSite does everything the build and the publish share: settings, discovery, parsing,
// the page tree, the site's pages and the images and files they reference. The returned
// error is already an RPC error.
func renderSite(ctx context.Context, workspaceRoot string, outputIndex int) (renderedSite, error) {
	if !filepath.IsAbs(workspaceRoot) {
		return renderedSite{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "workspaceRoot must be an absolute path, got %q", workspaceRoot)
	}
	output, scope, err := pagesOutput(workspaceRoot, outputIndex)
	if err != nil {
		return renderedSite{}, err
	}

	loaded, err := documentloading.LoadOutputDocuments(ctx, workspaceRoot, output, scope)
	if err != nil {
		return renderedSite{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
	}
	documents, warnings, problems := loaded.Documents, loaded.Warnings, loaded.Problems
	tree, err := documenttree.BuildTree(documents)
	if err != nil {
		problems = append(problems, err.Error())
	}
	if len(problems) > 0 {
		return renderedSite{Output: output, Warnings: warnings, Problems: problems}, nil
	}

	files, err := siterender.GenerateSite(deriveSiteTitle(output, workspaceRoot), tree)
	if err != nil {
		return renderedSite{}, rpcprotocol.Errorf(rpcprotocol.CodeInternalError, "%s", err.Error())
	}
	// Include the images and linked files the pages reference, so their relative src/href
	// resolve on the site instead of 404ing.
	assets, assetWarnings := collectSiteAssets(workspaceRoot, documents)

	return renderedSite{
		Output:   output,
		Files:    append(files, assets...),
		Warnings: append(warnings, assetWarnings...),
	}, nil
}
