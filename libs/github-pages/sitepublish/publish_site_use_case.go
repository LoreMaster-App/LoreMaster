package sitepublish

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lore-master/libs/github-pages/siterender"
)

const (
	defaultBranch        = "gh-pages"
	defaultCommitMessage = "docs: publish site with LoreMaster"
	// commitIdentity is used only when the repository has no configured author, so a
	// publish from a bare CI checkout still commits.
	commitAuthorName  = "LoreMaster"
	commitAuthorEmail = "lore-master@users.noreply.github.com"
)

// PublishSite writes the generated site as the entire content of the target folder (the
// branch root, or opts.Path inside it) and pushes it, shelling out to the user's git so their existing credentials and remotes are
// used. It never touches the workspace's working tree: it clones the remote into a
// temporary directory, replaces its contents with the site, commits and pushes. When the
// site already matches the branch the result is a no-op (Changed is false). A folder that
// holds other content LoreMaster did not write is refused, so a branch shared with another
// site is never wiped.
func PublishSite(ctx context.Context, opts PublishOptions, files []siterender.SiteFile) (PublishResult, error) {
	git := newGitClient()

	staged, err := stageSite(ctx, git, opts, files)
	if err != nil {
		return PublishResult{}, err
	}
	defer staged.cleanup()
	tmp, branch, remote := staged.dir, staged.branch, staged.remote

	status, err := git.run(ctx, tmp, "status", "--porcelain", "--no-renames")
	if err != nil {
		return PublishResult{}, err
	}
	if status == "" {
		return PublishResult{Branch: branch, Remote: remote, Changed: false, Files: len(files)}, nil
	}

	message := opts.CommitMessage
	if message == "" {
		message = defaultCommitMessage
	}
	if _, err := git.run(ctx, tmp, "-c", "user.name="+commitAuthorName, "-c", "user.email="+commitAuthorEmail, "commit", "-m", message); err != nil {
		return PublishResult{}, err
	}
	hash, err := git.run(ctx, tmp, "rev-parse", "--short", "HEAD")
	if err != nil {
		return PublishResult{}, err
	}
	if _, err := git.run(ctx, tmp, "push", "origin", branch); err != nil {
		return PublishResult{}, err
	}

	return PublishResult{Branch: branch, Remote: remote, Commit: hash, Changed: true, Files: len(files)}, nil
}

// resolveRemote is the clone URL to publish to: the workspace's own origin when Repo is
// empty, otherwise Repo itself (a URL) or a github.com URL built from "owner/name".
func resolveRemote(ctx context.Context, git gitClient, opts PublishOptions) (string, error) {
	if opts.Repo == "" {
		url, err := git.run(ctx, opts.WorkspaceRoot, "remote", "get-url", "origin")
		if err != nil {
			return "", fmt.Errorf("no origin remote to publish to in %s: %w", opts.WorkspaceRoot, err)
		}

		return url, nil
	}
	if strings.Contains(opts.Repo, "://") || strings.HasPrefix(opts.Repo, "git@") {
		return opts.Repo, nil
	}

	return "https://github.com/" + opts.Repo + ".git", nil
}

// checkoutBranch clones the target branch into tmp, or, when it does not exist yet, clones
// the default branch and starts the target as an orphan so the site does not inherit the
// code history.
func checkoutBranch(ctx context.Context, git gitClient, remote, branch, tmp string) error {
	if _, err := git.run(ctx, "", "clone", "--depth", "1", "--single-branch", "--branch", branch, remote, tmp); err == nil {
		return nil
	}
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	if _, err := git.run(ctx, "", "clone", "--depth", "1", remote, tmp); err != nil {
		return fmt.Errorf("clone %s: %w", remote, err)
	}
	if _, err := git.run(ctx, tmp, "switch", "--orphan", branch); err != nil {
		return fmt.Errorf("start branch %s: %w", branch, err)
	}

	return nil
}

// stagedSite is a temporary clone of the target branch with the generated site laid over the
// target folder and added to the index, ready to be inspected or committed.
type stagedSite struct {
	dir, branch, remote string
	cleanup             func()
}

// stageSite clones the target branch and replaces the target folder with the site, exactly as
// a publish would, without committing. The caller must call cleanup.
func stageSite(ctx context.Context, git gitClient, opts PublishOptions, files []siterender.SiteFile) (stagedSite, error) {
	branch := opts.Branch
	if branch == "" {
		branch = defaultBranch
	}
	remote, err := resolveRemote(ctx, git, opts)
	if err != nil {
		return stagedSite{}, err
	}

	tmp, err := os.MkdirTemp("", "lore-master-pages-*")
	if err != nil {
		return stagedSite{}, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	if err := checkoutBranch(ctx, git, remote, branch, tmp); err != nil {
		cleanup()

		return stagedSite{}, err
	}
	target := tmp
	if opts.Path != "" {
		target = filepath.Join(tmp, filepath.FromSlash(opts.Path))
	}
	if err := WriteSite(target, files); err != nil {
		cleanup()

		return stagedSite{}, err
	}
	if _, err := git.run(ctx, tmp, "add", "-A"); err != nil {
		cleanup()

		return stagedSite{}, err
	}

	return stagedSite{dir: tmp, branch: branch, remote: remote, cleanup: cleanup}, nil
}
