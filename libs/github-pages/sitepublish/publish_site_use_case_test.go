package sitepublish

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// seedBranch puts an unrelated site (somebody's web app) on a branch of the remote, the way
// another deploy would have, and returns to main.
func seedBranch(t *testing.T, work, branch string) {
	t.Helper()
	runGit(t, work, "checkout", "--orphan", branch)
	runGit(t, work, "rm", "-rf", "--quiet", ".")
	if err := os.WriteFile(filepath.Join(work, "index.html"), []byte("the web app"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, work, "add", "-A")
	runGit(t, work, "commit", "-m", "deploy the app")
	runGit(t, work, "push", "origin", branch)
	runGit(t, work, "checkout", "main")
}

func TestPublishSiteIntoAPathLeavesTheRestOfTheBranchAlone(t *testing.T) {
	work := workspaceWithRemote(t)
	seedBranch(t, work, "gh-pages")
	ctx := context.Background()

	result, err := PublishSite(ctx, PublishOptions{WorkspaceRoot: work, Path: "docs"}, site("<h1>Docs home</h1>"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatalf("unexpected result: %+v", result)
	}

	published := cloneBranch(t, work, "gh-pages")
	if app, _ := os.ReadFile(filepath.Join(published, "index.html")); string(app) != "the web app" {
		t.Errorf("the other site's home page was replaced: %q", app)
	}
	if docs, _ := os.ReadFile(filepath.Join(published, "docs", "index.html")); string(docs) != "<h1>Docs home</h1>" {
		t.Errorf("the docs were not published under docs/: %q", docs)
	}

	// A second publish replaces only the docs folder again, and is a no-op when unchanged.
	again, err := PublishSite(ctx, PublishOptions{WorkspaceRoot: work, Path: "docs"}, site("<h1>Docs home</h1>"))
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed {
		t.Errorf("an unchanged republish should be a no-op: %+v", again)
	}
}

func TestPublishSiteRefusesABranchHoldingAnotherSite(t *testing.T) {
	work := workspaceWithRemote(t)
	seedBranch(t, work, "gh-pages")

	_, err := PublishSite(context.Background(), PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>"))
	if !IsForeignFolder(err) {
		t.Fatalf("want a foreign-folder refusal, got %v", err)
	}

	published := cloneBranch(t, work, "gh-pages")
	if app, _ := os.ReadFile(filepath.Join(published, "index.html")); string(app) != "the web app" {
		t.Errorf("the other site was modified: %q", app)
	}
}

// workspaceWithWiki is workspaceWithRemote plus the repository's wiki: a bare
// "<remote>.wiki.git" holding the Home page GitHub makes when the wiki is first created.
func workspaceWithWiki(t *testing.T) (work, wikiRemotePath string) {
	t.Helper()
	work = workspaceWithRemote(t)
	origin, err := exec.Command("git", "-C", work, "remote", "get-url", "origin").Output()
	if err != nil {
		t.Fatal(err)
	}
	wikiRemotePath = strings.TrimSpace(string(origin)) + ".wiki.git"
	if err := os.MkdirAll(wikiRemotePath, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, wikiRemotePath, "init", "--bare", "--initial-branch=master")

	seed := t.TempDir()
	runGit(t, "", "clone", wikiRemotePath, seed)
	runGit(t, seed, "config", "user.name", "Test")
	runGit(t, seed, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(seed, "Home.md"), []byte("Welcome to the wiki"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seed, "Notes.md"), []byte("written on GitHub"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, seed, "add", "-A")
	runGit(t, seed, "commit", "-m", "Initial Home page")
	runGit(t, seed, "push", "origin", "HEAD:master")

	return work, wikiRemotePath
}

func wikiFiles() []siterender.SiteFile {
	return []siterender.SiteFile{
		{Path: "Home.md", Content: []byte("# Project\n")},
		{Path: "Guide.md", Content: []byte("# Guide\n")},
		{Path: "_Sidebar.md", Content: []byte("- [Project](Home)\n")},
	}
}

func TestPublishSiteToTheWikiOverlaysPagesAndKeepsTheOnesWrittenOnGitHub(t *testing.T) {
	work, wiki := workspaceWithWiki(t)

	result, err := PublishSite(context.Background(), PublishOptions{WorkspaceRoot: work, Wiki: true}, wikiFiles())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Branch != "master" || !strings.HasSuffix(result.Remote, ".wiki.git") {
		t.Fatalf("unexpected result: %+v", result)
	}

	clone := t.TempDir()
	runGit(t, "", "clone", wiki, clone)
	for _, name := range []string{"Home.md", "Guide.md", "_Sidebar.md", "Notes.md"} {
		if _, err := os.Stat(filepath.Join(clone, name)); err != nil {
			t.Errorf("wiki is missing %s: %v", name, err)
		}
	}
	if got, _ := os.ReadFile(filepath.Join(clone, "Home.md")); strings.TrimSpace(string(got)) != "# Project" {
		t.Errorf("Home.md was not replaced: %q", got)
	}

	again, err := CheckSite(context.Background(), PublishOptions{WorkspaceRoot: work, Wiki: true}, wikiFiles())
	if err != nil {
		t.Fatal(err)
	}
	if !again.UpToDate {
		t.Errorf("a published wiki should be up to date: %+v", again.Changes)
	}
}

func TestPublishSiteToAWikiThatDoesNotExistExplainsHowToCreateIt(t *testing.T) {
	work := workspaceWithRemote(t)

	_, err := PublishSite(context.Background(), PublishOptions{WorkspaceRoot: work, Wiki: true}, wikiFiles())

	if err == nil || !strings.Contains(err.Error(), "first page is created on GitHub") {
		t.Fatalf("want guidance about creating the wiki, got %v", err)
	}
}

func TestWikiRemoteInsertsWikiBeforeTheGitSuffix(t *testing.T) {
	cases := map[string]string{
		"https://github.com/o/r.git":      "https://github.com/o/r.wiki.git",
		"https://github.com/o/r":          "https://github.com/o/r.wiki.git",
		"git@github.com:o/r.git":          "git@github.com:o/r.wiki.git",
		"https://github.com/o/r.wiki.git": "https://github.com/o/r.wiki.git",
	}
	for in, want := range cases {
		if got := wikiRemote(in); got != want {
			t.Errorf("wikiRemote(%q) = %q, want %q", in, got, want)
		}
	}
}
