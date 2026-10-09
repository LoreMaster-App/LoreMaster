package pagescommands

import (
	"context"
	"path/filepath"
	"strings"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/github-pages/sitepublish"
)

// BuildPages handles pages/build: the rendered site (see renderSite) written into a local
// folder. When the Markdown has errors nothing is written; they come back in the result.
func BuildPages() rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.PagesBuildParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(params.OutDir) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "outDir must be an absolute path, got %q", params.OutDir)
		}
		if containsOrEquals(params.OutDir, params.WorkspaceRoot) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "outDir %q would replace the workspace; choose a folder inside it, such as dist/docs", params.OutDir)
		}

		site, err := renderSite(ctx, params.WorkspaceRoot, params.Output)
		if err != nil {
			return nil, err
		}
		if len(site.Problems) > 0 {
			return rpcprotocol.PagesBuildResult{Warnings: site.Warnings, Errors: site.Problems}, nil
		}

		if err := sitepublish.WriteSite(params.OutDir, site.Files); err != nil {
			if sitepublish.IsForeignFolder(err) {
				return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "%s", err.Error())
			}

			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInternalError, "write the site: %s", err.Error())
		}

		return rpcprotocol.PagesBuildResult{OutDir: params.OutDir, Files: len(site.Files), Warnings: site.Warnings}, nil
	}
}

// containsOrEquals reports whether folder is the same as inner or an ancestor of it, so a
// build can never be pointed at the workspace itself or at a folder above it.
func containsOrEquals(folder, inner string) bool {
	relative, err := filepath.Rel(filepath.Clean(folder), filepath.Clean(inner))
	if err != nil {
		return false
	}

	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
