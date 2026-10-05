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
	"lore-master/libs/github-pages/siterender"
	"lore-master/libs/markdown-workspace/documenttree"
)

// PublishPages handles pages/publish: settings, discovery, parsing, tree, site generation
// and the git publish. When the Markdown has errors nothing is published; they come back in
// the result so the editor can show them.
func PublishPages() rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.PagesPublishParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(params.WorkspaceRoot) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "workspaceRoot must be an absolute path, got %q", params.WorkspaceRoot)
		}
		output, err := pagesOutput(params)
		if err != nil {
			return nil, err
		}

		documents, warnings, problems, err := loadMarkdown(ctx, params.WorkspaceRoot, output)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}
		tree, err := documenttree.BuildTree(documents)
		if err != nil {
			problems = append(problems, err.Error())
		}
		if len(problems) > 0 {
			return rpcprotocol.PagesPublishResult{Warnings: warnings, Errors: problems}, nil
		}

		files, err := siterender.GenerateSite(deriveSiteTitle(output, params.WorkspaceRoot), tree)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInternalError, "%s", err.Error())
		}
		published, err := sitepublish.PublishSite(ctx, sitepublish.PublishOptions{
			WorkspaceRoot: params.WorkspaceRoot, Repo: output.Repo, Branch: output.Branch,
		}, files)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodePlatformUnreachable, "%s", err.Error())
		}

		return rpcprotocol.PagesPublishResult{
			Branch: published.Branch, Remote: published.Remote, Commit: published.Commit,
			Changed: published.Changed, Files: published.Files, URL: pagesURL(published.Remote),
			Warnings: warnings,
		}, nil
	}
}

// pagesOutput loads and checks the settings and picks the output, which must be a
// github-pages output.
func pagesOutput(params rpcprotocol.PagesPublishParams) (workspacesettings.Output, error) {
	loaded, err := workspacesettings.LoadSettings(params.WorkspaceRoot)
	if err != nil {
		return workspacesettings.Output{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
	}
	if err := workspacesettings.Validate(loaded.Settings); err != nil {
		return workspacesettings.Output{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
	}
	if params.Output < 0 || params.Output >= len(loaded.Settings.Outputs) {
		return workspacesettings.Output{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "output %d does not exist; %s has %d", params.Output, workspacesettings.FileName, len(loaded.Settings.Outputs))
	}
	output := loaded.Settings.Outputs[params.Output]
	if output.Platform != "github-pages" {
		return workspacesettings.Output{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "output %d is a %q output, not github-pages", params.Output, output.Platform)
	}

	return output, nil
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
