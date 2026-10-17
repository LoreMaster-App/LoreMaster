package pagescommands

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/github-pages/sitepublish"
)

// maxCheckChanges caps the file list of a check result; the total is always complete.
const maxCheckChanges = 200

// maxNewPagesNamed is how many new pages a warning names before saying how many more there are.
const maxNewPagesNamed = 5

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
			Wiki: site.Output.Platform == "github-wiki",
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

		result.NewPages, result.NewPagePaths = newPages(checked.Changes, site.Output.Platform == "github-wiki")
		if result.NewPages > 0 {
			result.Warnings = append(result.Warnings, newPagesWarning(result.NewPages, result.NewPagePaths))
		}

		return result, nil
	}
}

// newPages finds the pages a publish would add to the public output, as opposed to the support files (stylesheets, the
// navigation, the redirect from the home address): the HTML pages of a site and the Markdown pages of a wiki.
func newPages(changes []sitepublish.SiteChange, wiki bool) (int, []string) {
	var named []string
	count := 0
	for _, change := range changes {
		if change.Kind != sitepublish.ChangeAdded || !isPage(change.Path, wiki) {
			continue
		}
		count++
		if len(named) < maxNewPagesNamed {
			named = append(named, change.Path)
		}
	}

	return count, named
}

func isPage(file string, wiki bool) bool {
	name := path.Base(file)
	if wiki {
		return strings.HasSuffix(name, ".md") && !strings.HasPrefix(name, "_")
	}

	return strings.HasSuffix(name, ".html") && name != "index.html" && !slices.Contains(strings.Split(path.Dir(file), "/"), "assets")
}

// newPagesWarning says that publishing makes pages public that were not, which is the thing worth stopping to read in a
// public output: a file added to the folder becomes visible to everyone on the next publish.
func newPagesWarning(count int, named []string) string {
	more := ""
	if count > len(named) {
		more = fmt.Sprintf(", and %d more", count-len(named))
	}
	noun := "pages are"
	if count == 1 {
		noun = "page is"
	}

	return fmt.Sprintf("%d %s new to this public output and become visible to everyone when you publish: %s%s", count, noun, strings.Join(named, ", "), more)
}
