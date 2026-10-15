package syncjournal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"lore-master/libs/documentation-sync/platformport"
)

var space = platformport.SpaceRef{Key: "ENG"}

func newPlatform() *platformport.InMemoryPlatform {
	return platformport.NewInMemoryPlatform(platformport.Space{Key: "ENG", Name: "Engineering"})
}

func resultsOf(outcome Outcome) map[string]string {
	results := map[string]string{}
	for _, page := range outcome.Pages {
		results[page.Title] = page.Result
	}

	return results
}

func TestRollingBackDeletesTheCreatedPagesChildrenFirst(t *testing.T) {
	ctx := context.Background()
	inner := newPlatform()
	root := inner.SeedPage("ENG", "", "Root")
	recorder := NewRecorder(inner)

	parent, err := recorder.CreatePage(ctx, platformport.NewPage{Space: space, ParentID: root, Title: "Guide"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := recorder.CreatePage(ctx, platformport.NewPage{Space: space, ParentID: parent.ID, Title: "Guide: Install"})
	if err != nil {
		t.Fatal(err)
	}

	journal := Journal{Pages: recorder.Changes()}
	outcome := Rollback(ctx, inner, "", journal, Options{})

	if got := resultsOf(outcome); got["Guide"] != ResultTrashed || got["Guide: Install"] != ResultTrashed {
		t.Fatalf("results = %v", got)
	}
	if outcome.Pages[0].PageID != child.ID {
		t.Fatalf("the child must go before its parent: %+v", outcome.Pages)
	}
	if inner.Page(parent.ID) != nil || inner.Page(child.ID) != nil {
		t.Fatal("created pages are still there")
	}
	if inner.Page(root) == nil {
		t.Fatal("the page that was there before was removed")
	}
	if !outcome.Complete() {
		t.Fatalf("not complete: %+v", outcome)
	}
}

func TestRollingBackPutsUpdatedPagesBackAsTheyWere(t *testing.T) {
	ctx := context.Background()
	inner := newPlatform()
	oldParent := inner.SeedPage("ENG", "", "Old parent")
	newParent := inner.SeedPage("ENG", "", "New parent")
	page := inner.SeedPage("ENG", oldParent, "Quotes: Intro")
	recorder := NewRecorder(inner)

	updated, err := recorder.UpdatePage(ctx, platformport.PageUpdate{ID: page, ExpectedVersion: 1, Title: "Quotes: Introduction", ParentID: newParent})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recorder.UpdatePage(ctx, platformport.PageUpdate{ID: page, ExpectedVersion: updated.Version, Title: "Quotes: Introduction", ParentID: newParent}); err != nil {
		t.Fatal(err)
	}
	changes := recorder.Changes()
	if len(changes) != 1 || changes[0].Kind != Updated || changes[0].Before.Title != "Quotes: Intro" || changes[0].AfterVersion != 3 {
		t.Fatalf("changes = %+v", changes)
	}

	outcome := Rollback(ctx, inner, "", Journal{Pages: changes}, Options{})

	restored := inner.Page(page)
	if restored.Title != "Quotes: Intro" || restored.ParentID != oldParent {
		t.Fatalf("page = %+v", restored.RemotePage)
	}
	if got := resultsOf(outcome)["Quotes: Intro"]; got != ResultRestored {
		t.Fatalf("results = %v", resultsOf(outcome))
	}
}

func TestAPageEditedAfterTheSyncIsLeftAloneUnlessForced(t *testing.T) {
	ctx := context.Background()
	inner := newPlatform()
	page := inner.SeedPage("ENG", "", "Notes")
	recorder := NewRecorder(inner)
	if _, err := recorder.UpdatePage(ctx, platformport.PageUpdate{ID: page, ExpectedVersion: 1, Title: "Notes v2"}); err != nil {
		t.Fatal(err)
	}
	created, err := recorder.CreatePage(ctx, platformport.NewPage{Space: space, Title: "Fresh"})
	if err != nil {
		t.Fatal(err)
	}
	inner.EditRemotely(page)
	inner.EditRemotely(created.ID)
	journal := Journal{Pages: recorder.Changes()}

	skipped := Rollback(ctx, inner, "", journal, Options{})
	if got := resultsOf(skipped); got["Notes"] != ResultSkipped || got["Fresh"] != ResultSkipped {
		t.Fatalf("results = %v", got)
	}
	if skipped.Complete() {
		t.Fatal("a skipped page must keep the journal")
	}
	if inner.Page(created.ID) == nil {
		t.Fatal("a skipped page was deleted")
	}

	forced := Rollback(ctx, inner, "", journal, Options{Force: true})
	if got := resultsOf(forced); got["Notes"] != ResultRestored || got["Fresh"] != ResultTrashed {
		t.Fatalf("results = %v", got)
	}
}

func TestAPageWithOthersBelowItIsNotDeleted(t *testing.T) {
	ctx := context.Background()
	inner := newPlatform()
	recorder := NewRecorder(inner)
	created, err := recorder.CreatePage(ctx, platformport.NewPage{Space: space, Title: "Hub"})
	if err != nil {
		t.Fatal(err)
	}
	inner.SeedPage("ENG", created.ID, "Someone else's page")

	outcome := Rollback(ctx, inner, "", Journal{Pages: recorder.Changes()}, Options{})

	if got := resultsOf(outcome)["Hub"]; got != ResultSkipped {
		t.Fatalf("results = %v", resultsOf(outcome))
	}
}

func TestADryRunChangesNothing(t *testing.T) {
	ctx := context.Background()
	inner := newPlatform()
	page := inner.SeedPage("ENG", "", "Notes")
	recorder := NewRecorder(inner)
	if _, err := recorder.UpdatePage(ctx, platformport.PageUpdate{ID: page, ExpectedVersion: 1, Title: "Notes v2"}); err != nil {
		t.Fatal(err)
	}
	created, err := recorder.CreatePage(ctx, platformport.NewPage{Space: space, Title: "Fresh"})
	if err != nil {
		t.Fatal(err)
	}

	outcome := Rollback(ctx, inner, "", Journal{Pages: recorder.Changes()}, Options{DryRun: true})

	if got := resultsOf(outcome); got["Notes"] != ResultWouldRestore || got["Fresh"] != ResultWouldTrash {
		t.Fatalf("results = %v", got)
	}
	if inner.Page(created.ID) == nil || inner.Page(page).Title != "Notes v2" {
		t.Fatal("a dry run changed the platform")
	}
}

func TestATrashedPageIsReportedAsIrreversible(t *testing.T) {
	ctx := context.Background()
	inner := newPlatform()
	page := inner.SeedPage("ENG", "", "Old")
	recorder := NewRecorder(inner)
	if err := recorder.TrashPage(ctx, page); err != nil {
		t.Fatal(err)
	}

	outcome := Rollback(ctx, inner, "", Journal{Pages: recorder.Changes()}, Options{})

	if got := resultsOf(outcome)["Old"]; got != ResultIrreversible {
		t.Fatalf("results = %v", resultsOf(outcome))
	}
	if outcome.Complete() {
		t.Fatal("an irreversible page must keep the journal")
	}
}

func TestFilesGetTheirOldContentBackAndCreatedFilesAreRemoved(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.md", "# A\n")
	write("edited.md", "# Edited\n")

	before := CaptureFiles(root, []string{"a.md", "edited.md", "new.md"})
	write("a.md", "<!-- lore-master page 1 -->\n# A\n")
	write("edited.md", "<!-- lore-master page 2 -->\n# Edited\n")
	write("new.md", "pulled\n")
	changes := ChangedFiles(root, before)
	if len(changes) != 3 {
		t.Fatalf("changes = %+v", changes)
	}
	write("edited.md", "# Edited by hand after the sync\n")

	outcome := Rollback(context.Background(), newPlatform(), root, Journal{Files: changes}, Options{})

	results := map[string]string{}
	for _, file := range outcome.Files {
		results[file.Path] = file.Result
	}
	if results["a.md"] != ResultRestored || results["new.md"] != ResultRestored || results["edited.md"] != ResultSkipped {
		t.Fatalf("results = %v", results)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "a.md")); string(got) != "# A\n" {
		t.Fatalf("a.md = %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "new.md")); !os.IsNotExist(err) {
		t.Fatal("the created file is still there")
	}

	forced := Rollback(context.Background(), newPlatform(), root, Journal{Files: changes}, Options{Force: true})
	if got, _ := os.ReadFile(filepath.Join(root, "edited.md")); string(got) != "# Edited\n" {
		t.Fatalf("edited.md = %q (%+v)", got, forced.Files)
	}
}

func TestUnchangedFilesAreNotJournalled(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "same.md"), []byte("# Same\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := CaptureFiles(root, []string{"same.md", "never-written.md"})

	if changes := ChangedFiles(root, before); len(changes) != 0 {
		t.Fatalf("changes = %+v", changes)
	}
}
