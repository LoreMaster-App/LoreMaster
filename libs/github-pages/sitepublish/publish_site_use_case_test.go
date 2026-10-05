package sitepublish

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"lore-master/libs/github-pages/siterender"
)

// runGit runs git in dir and fails the test on error; it is the test's own setup helper,
// independent of the code under test.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// workspaceWithRemote sets up a bare "remote" and a working repository with one commit on
// main whose origin points at the remote. It returns the working repository path.
func workspaceWithRemote(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	remote := filepath.ToSlash(t.TempDir())
	runGit(t, remote, "init", "--bare", "--initial-branch=main")

	work := t.TempDir()
	runGit(t, work, "init", "--initial-branch=main")
	runGit(t, work, "config", "user.name", "Test")
	runGit(t, work, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("# Code\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "-A")
	runGit(t, work, "commit", "-m", "initial")
	runGit(t, work, "remote", "add", "origin", remote)
	runGit(t, work, "push", "-u", "origin", "main")

	return work
}

// cloneBranch clones one branch of the work repo's origin into a fresh directory so the
// test can read what was actually pushed.
func cloneBranch(t *testing.T, work, branch string) string {
	t.Helper()
	remote, err := exec.Command("git", "-C", work, "remote", "get-url", "origin").Output()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	runGit(t, "", "clone", "--branch", branch, "--single-branch", string(remote[:len(remote)-1]), dir)

	return dir
}

func site(home string) []siterender.SiteFile {
	return []siterender.SiteFile{
		{Path: "index.html", Content: []byte(home)},
		{Path: "docs/guide.html", Content: []byte("<h1>Guide</h1>")},
		{Path: ".nojekyll", Content: []byte{}},
	}
}

func TestPublishSiteCreatesTheBranchAndPushesTheFiles(t *testing.T) {
	work := workspaceWithRemote(t)

	result, err := PublishSite(context.Background(), PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Branch != "gh-pages" || result.Commit == "" || result.Files != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}

	published := cloneBranch(t, work, "gh-pages")
	for _, want := range []string{"index.html", "docs/guide.html", ".nojekyll"} {
		if _, err := os.Stat(filepath.Join(published, filepath.FromSlash(want))); err != nil {
			t.Errorf("published branch is missing %q: %v", want, err)
		}
	}
	// The site is the whole branch: the code repo's README must not be there.
	if _, err := os.Stat(filepath.Join(published, "README.md")); !os.IsNotExist(err) {
		t.Errorf("gh-pages should not inherit the code history's files")
	}
}

func TestPublishSiteIsANoOpWhenNothingChanged(t *testing.T) {
	work := workspaceWithRemote(t)
	ctx := context.Background()

	if _, err := PublishSite(ctx, PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>")); err != nil {
		t.Fatal(err)
	}
	second, err := PublishSite(ctx, PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>"))
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || second.Commit != "" {
		t.Fatalf("re-publishing the same site should be a no-op, got %+v", second)
	}
}

func TestPublishSiteUpdatesAnExistingBranch(t *testing.T) {
	work := workspaceWithRemote(t)
	ctx := context.Background()

	if _, err := PublishSite(ctx, PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>")); err != nil {
		t.Fatal(err)
	}
	updated, err := PublishSite(ctx, PublishOptions{WorkspaceRoot: work, CommitMessage: "docs: second"}, site("<h1>Changed</h1>"))
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Changed || updated.Commit == "" {
		t.Fatalf("a changed site should produce a new commit, got %+v", updated)
	}

	published := cloneBranch(t, work, "gh-pages")
	got, err := os.ReadFile(filepath.Join(published, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "<h1>Changed</h1>" {
		t.Fatalf("the branch should hold the updated home page, got %q", got)
	}
}

func TestResolveRemoteBuildsAGitHubURLFromOwnerName(t *testing.T) {
	url, err := resolveRemote(context.Background(), newGitClient(), PublishOptions{Repo: "russoedu/LoreMaster"})
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://github.com/russoedu/LoreMaster.git" {
		t.Fatalf("got %q", url)
	}
}
