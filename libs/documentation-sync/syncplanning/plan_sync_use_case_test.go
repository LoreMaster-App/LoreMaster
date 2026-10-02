package syncplanning

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
	"lore-master/libs/markdown-workspace/syncannotation"
)

const baseURL = "https://docs.example"

// world is a fake platform with a root page, and the output that syncs under it.
type world struct {
	platform *platformport.InMemoryPlatform
	output   workspacesettings.Output
	root     string
}

func newWorld() world {
	platform := platformport.NewInMemoryPlatform(platformport.Space{ID: "1", Key: "ENG", Name: "Engineering"})
	root := platform.SeedPage("ENG", "", "Engineering Home")

	return world{platform: platform, root: root, output: workspacesettings.Output{
		Platform: "confluence", BaseURL: baseURL, Space: "ENG", ParentPageID: root, TitlePrefix: "ENG",
		Direction: "to-platform", TitleCollision: "fail", MermaidMode: "image", LinkMode: "title",
	}}
}

// synced seeds a marked page as an earlier sync would have left it, and returns the
// file content annotated with it.
func (w world) synced(parentID string, title string, body string, sourcePath string) (string, string) {
	id := w.platform.SeedPage("ENG", parentID, title)
	_ = w.platform.MarkPage(context.Background(), id, sourcePath)

	return id, annotated(id, 1, body, "")
}

func annotated(pageID string, version int, body string, titleOverride string) string {
	block := fmt.Sprintf("<!-- lore-master\nbase-url: %s\nspace: ENG\npage-id: %s\nversion: %d\ncontent-hash: %s\n", baseURL, pageID, version, syncannotation.ContentHash([]byte(body)))
	if titleOverride != "" {
		block += "title: " + titleOverride + "\n"
	}

	return block + "-->\n" + body
}

func treeOf(t *testing.T, files map[string]string) documenttree.DocumentTree {
	t.Helper()
	var documents []documentparsing.MarkdownDocument
	for path, content := range files {
		document, err := documentparsing.ParseDocument(documentdiscovery.DocumentPath(path), []byte(content))
		if err != nil {
			t.Fatal(err)
		}
		documents = append(documents, document)
	}
	tree, err := documenttree.BuildTree(documents)
	if err != nil {
		t.Fatal(err)
	}

	return tree
}

// summary is one line per action: "kind path title parent=<path|id> changes".
func summary(plan SyncPlan) []string {
	var lines []string
	for _, action := range plan.Actions {
		parent := string(action.ParentPath)
		if parent == "" {
			parent = "root"
		}
		line := fmt.Sprintf("%s %s %q parent=%s", action.Kind, action.Path, action.Title, parent)
		if len(action.Changes) > 0 {
			line += fmt.Sprintf(" %v", action.Changes)
		}
		lines = append(lines, line)
	}

	return lines
}

func plan(t *testing.T, w world, files map[string]string, scope ...documentdiscovery.DocumentPath) SyncPlan {
	t.Helper()
	result, err := PlanSync(context.Background(), w.platform, Input{Tree: treeOf(t, files), Output: w.output, SpaceID: "1", Scope: scope})
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func expect(t *testing.T, got SyncPlan, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(summary(got), want) {
		t.Fatalf("plan\n got: %q\nwant: %q\nwarnings %q errors %q", summary(got), want, got.Warnings, got.Errors)
	}
}

func TestFirstSyncCreatesEverythingParentsFirst(t *testing.T) {
	w := newWorld()
	got := plan(t, w, map[string]string{
		"README.md": "# Home\n", "readme.guide.md": "# Guide\n", "readme.guide.deep.md": "# Deep\n",
	})
	expect(t, got,
		`create README.md "ENG: Home" parent=root`,
		`create readme.guide.md "ENG: Guide" parent=README.md`,
		`create readme.guide.deep.md "ENG: Deep" parent=readme.guide.md`,
	)
	if got.Actions[0].ParentPageID != w.root || got.Actions[1].ParentPageID != "" {
		t.Fatalf("a root-level page goes under the configured parent; a child of a new page waits for its id: %+v", got.Actions)
	}
}

// steady is a workspace synced once: every file annotated, every page in place.
func steady(t *testing.T) (world, map[string]string, map[string]string) {
	t.Helper()
	w := newWorld()
	ids := map[string]string{}
	files := map[string]string{}
	ids["README.md"], files["README.md"] = w.synced(w.root, "ENG: Home", "# Home\n", "README.md")
	ids["readme.guide.md"], files["readme.guide.md"] = w.synced(ids["README.md"], "ENG: Guide", "# Guide\n", "readme.guide.md")
	ids["readme.guide.deep.md"], files["readme.guide.deep.md"] = w.synced(ids["readme.guide.md"], "ENG: Deep", "# Deep\n", "readme.guide.deep.md")

	return w, files, ids
}

func TestAnUntouchedWorkspaceIsUnchanged(t *testing.T) {
	w, files, _ := steady(t)
	expect(t, plan(t, w, files),
		`unchanged README.md "ENG: Home" parent=root`,
		`unchanged readme.guide.md "ENG: Guide" parent=README.md`,
		`unchanged readme.guide.deep.md "ENG: Deep" parent=readme.guide.md`,
	)
}

func TestEditingOneFileUpdatesExactlyOnePage(t *testing.T) {
	w, files, ids := steady(t)
	files["readme.guide.md"] = annotated(ids["readme.guide.md"], 1, "# Guide\n", "") + "New paragraph.\n"
	got := plan(t, w, files)
	expect(t, got,
		`unchanged README.md "ENG: Home" parent=root`,
		`update readme.guide.md "ENG: Guide" parent=README.md [content]`,
		`unchanged readme.guide.deep.md "ENG: Deep" parent=readme.guide.md`,
	)
	if counts := got.Counts(); counts[Update] != 1 || counts[Unchanged] != 2 {
		t.Fatalf("counts %v", counts)
	}
	if got.Actions[1].RemoteVersion != 1 || got.Actions[1].PageID != ids["readme.guide.md"] {
		t.Fatalf("an update is based on the remote version: %+v", got.Actions[1])
	}
}

func TestMovingAFileMovesItsPage(t *testing.T) {
	w, files, ids := steady(t)
	// deep becomes a child of README instead of guide: rename the file.
	files["readme.deep.md"] = files["readme.guide.deep.md"]
	delete(files, "readme.guide.deep.md")
	got := plan(t, w, files)
	expect(t, got,
		`unchanged README.md "ENG: Home" parent=root`,
		`move readme.deep.md "ENG: Deep" parent=README.md [parent]`,
		`unchanged readme.guide.md "ENG: Guide" parent=README.md`,
	)
	if got.Actions[1].ParentPageID != ids["README.md"] {
		t.Fatalf("parent page %q", got.Actions[1].ParentPageID)
	}
}

func TestATitleOverrideRenamesWithoutAnUpdate(t *testing.T) {
	w, files, ids := steady(t)
	files["readme.guide.md"] = annotated(ids["readme.guide.md"], 1, "# Guide\n", "User guide")
	expect(t, plan(t, w, files),
		`unchanged README.md "ENG: Home" parent=root`,
		`rename_title readme.guide.md "ENG: User guide" parent=README.md [title]`,
		`unchanged readme.guide.deep.md "ENG: Deep" parent=readme.guide.md`,
	)
}

func TestARemoteEditIsAConflictNeverAnOverwrite(t *testing.T) {
	w, files, ids := steady(t)
	w.platform.EditRemotely(ids["readme.guide.md"])
	files["readme.guide.md"] = annotated(ids["readme.guide.md"], 1, "# Guide\n", "") + "Local change too.\n"
	got := plan(t, w, files)
	expect(t, got,
		`unchanged README.md "ENG: Home" parent=root`,
		`conflict readme.guide.md "ENG: Guide" parent=README.md`,
		`unchanged readme.guide.deep.md "ENG: Deep" parent=readme.guide.md`,
	)
	if !strings.Contains(got.Actions[1].Reason, "edited on the platform after the last sync (version 2, synced at 1)") {
		t.Fatalf("reason %q", got.Actions[1].Reason)
	}
}

func TestAPageDeletedRemotelyIsCreatedAgain(t *testing.T) {
	w, files, ids := steady(t)
	_ = w.platform.TrashPage(context.Background(), ids["readme.guide.deep.md"])
	got := plan(t, w, files)
	expect(t, got,
		`unchanged README.md "ENG: Home" parent=root`,
		`unchanged readme.guide.md "ENG: Guide" parent=README.md`,
		`create readme.guide.deep.md "ENG: Deep" parent=readme.guide.md`,
	)
	want := "readme.guide.deep.md: page " + ids["readme.guide.deep.md"] + " no longer exists; it will be created again"
	if len(got.Warnings) != 1 || got.Warnings[0] != want {
		t.Fatalf("warnings %q", got.Warnings)
	}
}

func TestADeletedFileLeavesAnOrphanButHandMadePagesAreIgnored(t *testing.T) {
	w, files, ids := steady(t)
	delete(files, "readme.guide.deep.md")
	w.platform.SeedPage("ENG", ids["README.md"], "Hand-made notes")
	got := plan(t, w, files)
	expect(t, got,
		`unchanged README.md "ENG: Home" parent=root`,
		`unchanged readme.guide.md "ENG: Guide" parent=README.md`,
		`orphan  "ENG: Deep" parent=root`,
	)
	if got.Actions[2].PageID != ids["readme.guide.deep.md"] {
		t.Fatalf("orphan %+v", got.Actions[2])
	}
}

func TestATakenTitleFailsOrIsAdopted(t *testing.T) {
	w := newWorld()
	existing := w.platform.SeedPage("ENG", w.root, "ENG: Home")
	files := map[string]string{"README.md": "# Home\n"}

	failed := plan(t, w, files)
	want := `README.md: a page titled "ENG: Home" already exists (https://docs.example/pages/` + existing + `) and this file has no lore-master annotation; set titleCollision: adopt to take that page over, or give the file a "title:" override`
	if len(failed.Actions) != 0 || len(failed.Errors) != 1 || failed.Errors[0] != want {
		t.Fatalf("plan %+v", failed)
	}

	w.output.TitleCollision = "adopt"
	adopted := plan(t, w, files)
	expect(t, adopted, `adopt README.md "ENG: Home" parent=root [content parent title]`)
	if adopted.Actions[0].PageID != existing || adopted.Actions[0].RemoteVersion != 1 {
		t.Fatalf("adopt %+v", adopted.Actions[0])
	}
}

func TestScopeKeepsTheFileAndTheAncestorsItNeedsCreated(t *testing.T) {
	w, files, _ := steady(t)
	files["readme.guide.deep.extra.md"] = "# Extra\n"
	files["docs/README.md"] = "# Docs\n"
	files["docs/a.md"] = "# A\n"
	files["docs/b.md"] = "# B\n"
	expect(t, plan(t, w, files, "docs/a.md", "readme.guide.deep.extra.md"),
		`create docs/README.md "ENG: Docs" parent=README.md`,
		`create docs/a.md "ENG: A" parent=docs/README.md`,
		`create readme.guide.deep.extra.md "ENG: Extra" parent=readme.guide.deep.md`,
	)
}

func TestDuplicateTitlesStopThePlanBeforeAnyRequest(t *testing.T) {
	w := newWorld()
	got := plan(t, w, map[string]string{"a/setup.md": "# Setup\n", "b/setup.md": "# Setup\n"})
	if len(got.Actions) != 0 || len(got.Errors) != 1 || !strings.Contains(got.Errors[0], `a/setup.md, b/setup.md → "ENG: Setup"`) {
		t.Fatalf("plan %+v", got)
	}
	if calls := w.platform.Calls(); len(calls) != 0 {
		t.Fatalf("no request may be made, got %q", calls)
	}
}

func TestAnAnnotationFromAnotherSpaceStartsAfresh(t *testing.T) {
	w, files, ids := steady(t)
	w.output.Space = "OPS"
	w.platform = platformport.NewInMemoryPlatform(platformport.Space{ID: "2", Key: "OPS", Name: "Operations"})
	w.output.ParentPageID = w.platform.SeedPage("OPS", "", "Operations Home")
	// The ENG page id means something else here: a stranger's page.
	if stranger := w.platform.SeedPage("OPS", w.output.ParentPageID, "Runbook"); stranger != ids["README.md"] {
		t.Fatalf("fixture: want the ids to collide, got %s and %s", stranger, ids["README.md"])
	}
	got := plan(t, w, map[string]string{"README.md": files["README.md"]})
	expect(t, got, `create README.md "ENG: Home" parent=root`)
	if got.Actions[0].PageID != "" || len(got.Warnings) != 0 {
		t.Fatalf("a page id from another space must not be looked up: %+v %q", got.Actions[0], got.Warnings)
	}
}

func TestAnEditedImageUpdatesThePageItIsOn(t *testing.T) {
	w, files, ids := steady(t)
	// The guide shows one image; the last sync recorded its hash.
	files["readme.guide.md"] = strings.Replace(annotated(ids["readme.guide.md"], 1, "# Guide\n", ""), "-->", "attachments: {\"diagram.png\":\"old\"}\n-->", 1)
	run := func(attachments map[documentdiscovery.DocumentPath]map[string]string) SyncPlan {
		result, err := PlanSync(context.Background(), w.platform, Input{Tree: treeOf(t, files), Output: w.output, SpaceID: "1", Attachments: attachments})
		if err != nil {
			t.Fatal(err)
		}

		return result
	}

	same := run(map[documentdiscovery.DocumentPath]map[string]string{"readme.guide.md": {"diagram.png": "old"}})
	edited := run(map[documentdiscovery.DocumentPath]map[string]string{"readme.guide.md": {"diagram.png": "new"}})
	removed := run(map[documentdiscovery.DocumentPath]map[string]string{})
	unknown := run(nil)
	for name, got := range map[string]SyncPlan{"same": same, "unknown": unknown} {
		if got.Actions[1].Kind != Unchanged {
			t.Errorf("%s: %s", name, summary(got))
		}
	}
	for name, got := range map[string]SyncPlan{"edited": edited, "removed": removed} {
		if got.Actions[1].Kind != Update || got.Counts()[Update] != 1 {
			t.Errorf("%s: %s", name, summary(got))
		}
	}
}

func TestAFileThatLostItsAnnotationTakesItsPageBack(t *testing.T) {
	w, files, ids := steady(t)
	files["readme.guide.md"] = "# Guide\n"
	got := plan(t, w, files)
	expect(t, got,
		`unchanged README.md "ENG: Home" parent=root`,
		`adopt readme.guide.md "ENG: Guide" parent=README.md [content parent title]`,
		`unchanged readme.guide.deep.md "ENG: Deep" parent=readme.guide.md`,
	)
	if got.Actions[1].PageID != ids["readme.guide.md"] || len(got.Errors) != 0 ||
		len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "its annotation is gone; the page is taken back") {
		t.Fatalf("titleCollision is fail, yet the file's own page is no collision: %+v", got)
	}
}

func TestAMarkedPageAnotherFileStillPointsAtIsNotTakenBack(t *testing.T) {
	w, files, ids := steady(t)
	// The guide was retitled "Manual"; a new file now uses its old title.
	files["readme.guide.md"] = annotated(ids["readme.guide.md"], 1, "# Manual\n", "")
	files["readme.newguide.md"] = "# Guide\n"
	got := plan(t, w, files)
	if len(got.Errors) != 1 || !strings.Contains(got.Errors[0], `readme.newguide.md: a page titled "ENG: Guide" already exists`) {
		t.Fatalf("errors %q", got.Errors)
	}
}

func TestTwoMarkedPagesWithTheTitleAreNotGuessedBetween(t *testing.T) {
	w := newWorld()
	for _, title := range []string{"ENG: Home", "ENG: HOME"} {
		_ = w.platform.MarkPage(context.Background(), w.platform.SeedPage("ENG", w.root, title), "README.md")
	}
	got := plan(t, w, map[string]string{"README.md": "# Home\n"})
	if counts := got.Counts(); counts[Adopt] != 0 || counts[Orphan] != 2 || len(got.Errors) != 1 || !strings.Contains(got.Errors[0], "already exists") {
		t.Fatalf("plan %+v", got)
	}
}

func TestAnAnnotationForAnotherSpaceClaimsNothingHere(t *testing.T) {
	w, files, ids := steady(t)
	// Same page id, but written by a sync to another space: it says nothing about
	// this platform's page, which this file may still take back.
	files["readme.guide.md"] = strings.Replace(annotated(ids["readme.guide.md"], 1, "# Guide\n", ""), "space: ENG", "space: OPS", 1)
	got := plan(t, w, files)
	if got.Actions[1].Kind != Adopt || got.Actions[1].PageID != ids["readme.guide.md"] {
		t.Fatalf("%s %q", summary(got), got.Errors)
	}
}
