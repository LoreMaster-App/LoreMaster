package pagescommands

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/github-pages/sitepublish"
)

// maxCheckChanges caps the file list of a check result; the total is always complete.
const maxCheckChanges = 200

// CheckPages handles pages/check: it renders the site as a publish would and reports how it
// differs from the published branch, publishing nothing.
func CheckPages() rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.PagesCheckParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		site, err := RenderSite(ctx, params.WorkspaceRoot, params.Output)
		if err != nil {
			return nil, err
		}
		if len(site.Problems) > 0 {
			return rpcprotocol.PagesCheckResult{Warnings: site.Warnings, Errors: site.Problems}, nil
		}

		checked, err := sitepublish.CheckSite(ctx, sitepublish.PublishOptions{
			WorkspaceRoot: params.WorkspaceRoot, Repo: site.Output.Repo, Branch: site.Output.Branch, Path: site.Output.Path,
		}, site.Files)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodePlatformUnreachable, "%s", err.Error())
		}

		result := rpcprotocol.PagesCheckResult{
			Branch: checked.Branch, Remote: checked.Remote, UpToDate: checked.UpToDate,
			ChangesTotal: len(checked.Changes), Files: checked.Files, Warnings: site.Warnings,
		}
		for _, change := range checked.Changes {
			if len(result.Changes) == maxCheckChanges {
				break
			}
			result.Changes = append(result.Changes, rpcprotocol.PagesChange{Path: change.Path, Kind: change.Kind})
		}

		return result, nil
	}
}
