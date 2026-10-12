package filemoves

import (
	"reflect"
	"testing"
)

func docs(paths ...string) []Document {
	out := make([]Document, len(paths))
	for i, p := range paths {
		out[i] = Document{Path: p}
	}

	return out
}

func withParent(document Document, line int, text, value string) Document {
	document.Parent = &ParentRef{Line: line, Text: text, Value: value}

	return document
}

func TestDottedChildrenFollowARenamedParent(t *testing.T) {
	plan := PlanMoves(
		docs("docs/readme.api.md", "docs/readme.api.config.md", "docs/readme.api.config.env.md", "docs/readme.apix.md", "docs/other/readme.api.md"),
		[]Move{{From: "docs/readme.api.md", To: "docs/readme.rest.md"}},
	)
	want := []Rename{
		{From: "docs/readme.api.config.env.md", To: "docs/readme.rest.config.env.md"},
		{From: "docs/readme.api.config.md", To: "docs/readme.rest.config.md"},
	}
	if !reflect.DeepEqual(plan.Renames, want) {
		t.Fatalf("renames = %+v, want %+v", plan.Renames, want)
	}
}

func TestDottedChildrenMoveWithAFileToAnotherFolder(t *testing.T) {
	plan := PlanMoves(docs("a.md", "a.b.md"), []Move{{From: "a.md", To: "guides/a.md"}})
	want := []Rename{{From: "a.b.md", To: "guides/a.b.md"}}
	if !reflect.DeepEqual(plan.Renames, want) {
		t.Fatalf("renames = %+v, want %+v", plan.Renames, want)
	}
}

func TestAFolderMoveRenamesNothingExtra(t *testing.T) {
	plan := PlanMoves(docs("docs/a.md", "docs/a.b.md"), []Move{{From: "docs", To: "guides"}})
	if len(plan.Renames) != 0 {
		t.Fatalf("renames = %+v, want none", plan.Renames)
	}
}

func TestARenameThatWouldClashIsSkippedWithAWarning(t *testing.T) {
	plan := PlanMoves(docs("a.md", "a.b.md", "c.b.md"), []Move{{From: "a.md", To: "c.md"}})
	if len(plan.Renames) != 0 || len(plan.Warnings) != 1 {
		t.Fatalf("plan = %+v, want no renames and one warning", plan)
	}
}

func TestParentLinesFollowMovesAndRenames(t *testing.T) {
	plan := PlanMoves([]Document{
		{Path: "docs/guide.md"},
		withParent(Document{Path: "docs/setup.md"}, 2, "parent: ./guide.md", "./guide.md"),
		withParent(Document{Path: "docs/deep/x.md"}, 1, "parent:   ../guide.md", "../guide.md"),
		withParent(Document{Path: "top.md"}, 3, "parent: /docs/guide.md", "/docs/guide.md"),
		withParent(Document{Path: "docs/loose.md"}, 1, "parent: nowhere.md", "nowhere.md"),
	}, []Move{{From: "docs/guide.md", To: "docs/manual/guide.md"}})
	want := []LineEdit{
		{Path: "docs/deep/x.md", Line: 1, Text: "parent:   ../manual/guide.md"},
		{Path: "docs/setup.md", Line: 2, Text: "parent: ./manual/guide.md"},
		{Path: "top.md", Line: 3, Text: "parent: /docs/manual/guide.md"},
	}
	if !reflect.DeepEqual(plan.Edits, want) {
		t.Fatalf("edits = %+v, want %+v", plan.Edits, want)
	}
}

func TestAParentLineInAMovedFileIsRewrittenRelativeToItsNewHome(t *testing.T) {
	plan := PlanMoves([]Document{
		{Path: "docs/guide.md"},
		withParent(Document{Path: "docs/setup.md"}, 1, "parent: guide.md", "guide.md"),
	}, []Move{{From: "docs/setup.md", To: "docs/steps/setup.md"}})
	want := []LineEdit{{Path: "docs/setup.md", Line: 1, Text: "parent: ../guide.md"}}
	if !reflect.DeepEqual(plan.Edits, want) {
		t.Fatalf("edits = %+v, want %+v", plan.Edits, want)
	}
}

func TestNoEditWhenBothMoveTogether(t *testing.T) {
	plan := PlanMoves([]Document{
		{Path: "docs/guide.md"},
		withParent(Document{Path: "docs/setup.md"}, 1, "parent: guide.md", "guide.md"),
	}, []Move{{From: "docs", To: "manual"}})
	if len(plan.Edits) != 0 {
		t.Fatalf("edits = %+v, want none", plan.Edits)
	}
}
