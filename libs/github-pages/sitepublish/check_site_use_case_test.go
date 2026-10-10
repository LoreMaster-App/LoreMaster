package sitepublish

import (
	"context"
	"testing"

	"lore-master/libs/github-pages/siterender"
)

func kinds(changes []SiteChange) map[string]string {
	byPath := map[string]string{}
	for _, change := range changes {
		byPath[change.Path] = change.Kind
	}

	return byPath
}

func TestCheckSiteReportsEveryFileOfABranchThatDoesNotExistYet(t *testing.T) {
	work := workspaceWithRemote(t)

	result, err := CheckSite(context.Background(), PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>"))
	if err != nil {
		t.Fatal(err)
	}
	if result.UpToDate || result.Files != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := kinds(result.Changes); got["index.html"] != ChangeAdded || got["docs/guide.html"] != ChangeAdded {
		t.Errorf("want the site files added, got %v", got)
	}
}

func TestCheckSiteIsUpToDateAfterAPublishAndPushesNothing(t *testing.T) {
	work := workspaceWithRemote(t)
	if _, err := PublishSite(context.Background(), PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>")); err != nil {
		t.Fatal(err)
	}

	result, err := CheckSite(context.Background(), PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.UpToDate || len(result.Changes) != 0 {
		t.Fatalf("want up to date, got %+v", result)
	}
}

func TestCheckSiteSeesModifiedAddedAndRemovedFilesWithoutPublishing(t *testing.T) {
	work := workspaceWithRemote(t)
	if _, err := PublishSite(context.Background(), PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>")); err != nil {
		t.Fatal(err)
	}

	changed := []siterender.SiteFile{
		{Path: "index.html", Content: []byte("<h1>New home</h1>")},
		{Path: "docs/new.html", Content: []byte("<h1>New</h1>")},
		{Path: ".nojekyll", Content: []byte{}},
	}
	result, err := CheckSite(context.Background(), PublishOptions{WorkspaceRoot: work}, changed)
	if err != nil {
		t.Fatal(err)
	}
	got := kinds(result.Changes)
	if got["index.html"] != ChangeModified || got["docs/new.html"] != ChangeAdded || got["docs/guide.html"] != ChangeRemoved {
		t.Errorf("unexpected changes: %v", got)
	}

	// The check must not have published: the branch still has the earlier site.
	again, err := CheckSite(context.Background(), PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>"))
	if err != nil {
		t.Fatal(err)
	}
	if !again.UpToDate {
		t.Error("a check published the change")
	}
}

func TestCheckSiteRefusesABranchHoldingAnotherSite(t *testing.T) {
	work := workspaceWithRemote(t)
	seedBranch(t, work, "gh-pages")

	_, err := CheckSite(context.Background(), PublishOptions{WorkspaceRoot: work}, site("<h1>Home</h1>"))
	if !IsForeignFolder(err) {
		t.Fatalf("want a foreign-folder refusal, got %v", err)
	}
}
