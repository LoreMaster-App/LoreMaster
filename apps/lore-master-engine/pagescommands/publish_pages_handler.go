package pagescommands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/github-pages/sitepublish"
)

// PublishPages handles pages/publish: the rendered site (see RenderSite) and the git
// publish. When the Markdown has errors nothing is published; they come back in the result so
// the editor can show them.
func PublishPages() rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.PagesPublishParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		site, err := RenderSite(ctx, params.WorkspaceRoot, params.Output)
		if err != nil {
			return nil, err
		}
		if len(site.Problems) > 0 {
			return rpcprotocol.PagesPublishResult{Warnings: site.Warnings, Errors: site.Problems}, nil
		}

		published, err := sitepublish.PublishSite(ctx, sitepublish.PublishOptions{
			WorkspaceRoot: params.WorkspaceRoot, Repo: site.Output.Repo, Branch: site.Output.Branch, Path: site.Output.Path,
			Wiki: site.Output.Platform == "github-wiki",
		}, site.Files)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodePlatformUnreachable, "%s", err.Error())
		}

		return rpcprotocol.PagesPublishResult{
			Branch: published.Branch, Remote: published.Remote, Commit: published.Commit,
			Changed: published.Changed, Files: published.Files, URL: siteURL(site.Output.Platform, published.Remote),
			Warnings: site.Warnings,
		}, nil
	}
}

// pagesOutput loads and checks the settings and picks the output, which must be a
// github-pages output.
func pagesOutput(workspaceRoot string, outputIndex int) (workspacesettings.Output, workspacesettings.DiscoveryScope, error) {
	loaded, err := workspacesettings.LoadSettings(workspaceRoot)
	if err != nil {
		return workspacesettings.Output{}, workspacesettings.DiscoveryScope{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
	}
	if err := workspacesettings.Validate(loaded.Settings); err != nil {
		return workspacesettings.Output{}, workspacesettings.DiscoveryScope{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
	}
	if outputIndex < 0 || outputIndex >= len(loaded.Settings.Outputs) {
		return workspacesettings.Output{}, workspacesettings.DiscoveryScope{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "output %d does not exist; %s has %d", outputIndex, workspacesettings.FileName, len(loaded.Settings.Outputs))
	}
	output := loaded.Settings.Outputs[outputIndex]
	if output.Platform != "github-pages" && output.Platform != "github-wiki" {
		return workspacesettings.Output{}, workspacesettings.DiscoveryScope{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "output %d is a %q output, not github-pages or github-wiki", outputIndex, output.Platform)
	}

	return output, loaded.Settings.DiscoveryScope(), nil
}

// deriveSiteTitle labels the site's sidebar: the repository name when one is configured,
// otherwise the workspace folder's name.
func deriveSiteTitle(output workspacesettings.Output, root string) string {
	if output.Repo != "" {
		return repoName(output.Repo)
	}

	return filepath.Base(root)
}

// repoName is the last path segment of a repository reference, without a .git suffix.
func repoName(repo string) string {
	name := strings.TrimSuffix(strings.TrimRight(repo, "/"), ".git")
	if cut := strings.LastIndexAny(name, "/:"); cut >= 0 {
		name = name[cut+1:]
	}

	return name
}

// siteURL is where a published output can be read: the wiki's page on github.com for a
// github-wiki output, the github.io address otherwise; empty when the remote is not github.com.
func siteURL(platform, remote string) string {
	if platform != "github-wiki" {
		return pagesURL(remote)
	}
	address := pagesURL(strings.TrimSuffix(strings.TrimSuffix(remote, ".git"), ".wiki"))
	if address == "" {
		return ""
	}
	// pagesURL gives https://<owner>.github.io/<repo>/; the wiki lives at github.com/<owner>/<repo>/wiki.
	trimmed := strings.TrimPrefix(address, "https://")
	owner := trimmed[:strings.Index(trimmed, ".github.io")]
	repo := strings.Trim(trimmed[strings.Index(trimmed, "/"):], "/")

	return fmt.Sprintf("https://github.com/%s/%s/wiki", owner, repo)
}

// pagesURL is the GitHub Pages address for a github.com remote, or empty when the remote is
// not recognisably github.com.
func pagesURL(remote string) string {
	trimmed := strings.TrimSuffix(remote, ".git")
	var repoPath string
	switch {
	case strings.Contains(trimmed, "github.com:"):
		repoPath = trimmed[strings.Index(trimmed, "github.com:")+len("github.com:"):]
	case strings.Contains(trimmed, "github.com/"):
		repoPath = trimmed[strings.Index(trimmed, "github.com/")+len("github.com/"):]
	default:
		return ""
	}
	parts := strings.Split(strings.Trim(repoPath, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}

	return fmt.Sprintf("https://%s.github.io/%s/", parts[0], parts[1])
}
