package platformport

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestInMemoryPlatformKeepsThePlatformsRules(t *testing.T) {
	ctx := context.Background()
	platform := NewInMemoryPlatform(Space{ID: "1", Key: "ENG", Name: "Engineering"})
	root := platform.SeedPage("ENG", "", "Home")
	eng := SpaceRef{ID: "1", Key: "ENG"}

	child, err := platform.CreatePage(ctx, NewPage{Space: eng, ParentID: root, Title: "ENG: Setup"})
	if err != nil || child.Version != 1 || child.ParentID != root {
		t.Fatalf("create %+v, %v", child, err)
	}
	var taken *TitleTakenError
	if _, err := platform.CreatePage(ctx, NewPage{Space: eng, ParentID: root, Title: "eng: setup"}); !errors.As(err, &taken) {
		t.Fatalf("a title differing only in case must clash, got %v", err)
	}

	platform.EditRemotely(child.ID)
	var conflict *VersionConflictError
	if _, err := platform.UpdatePage(ctx, PageUpdate{ID: child.ID, ExpectedVersion: 1, Title: "ENG: Setup"}); !errors.As(err, &conflict) {
		t.Fatalf("a stale version must conflict, got %v", err)
	}
	updated, err := platform.UpdatePage(ctx, PageUpdate{ID: child.ID, ExpectedVersion: 2, Title: "ENG: Install"})
	if err != nil || updated.Version != 3 || updated.Title != "ENG: Install" {
		t.Fatalf("update %+v, %v", updated, err)
	}

	grandchild, _ := platform.CreatePage(ctx, NewPage{Space: eng, ParentID: child.ID, Title: "ENG: Deep"})
	_ = platform.MarkPage(ctx, grandchild.ID, "docs/deep.md")
	platform.SeedPage("ENG", child.ID, "Hand-made")
	marked, _ := platform.ListMarkedDescendants(ctx, root)
	if len(marked) != 1 || marked[0].ID != grandchild.ID {
		t.Fatalf("marked %+v", marked)
	}

	first, _ := platform.UploadFile(ctx, child.ID, File{Name: "a.svg", Content: []byte("<svg/>")})
	second, _ := platform.UploadFile(ctx, child.ID, File{Name: "a.svg", Content: []byte("<svg/>")})
	if first.Skipped || !second.Skipped || first.ID != second.ID {
		t.Fatalf("upload %+v then %+v", first, second)
	}

	_ = platform.TrashPage(ctx, grandchild.ID)
	var missing *PageNotFoundError
	if _, err := platform.GetPage(ctx, grandchild.ID); !errors.As(err, &missing) {
		t.Fatalf("a trashed page is gone, got %v", err)
	}
	if want := []string{
		"CreatePage ENG: Setup", "CreatePage eng: setup", "UpdatePage p2 v1", "UpdatePage p2 v2", "CreatePage ENG: Deep",
		"MarkPage p3 docs/deep.md", "ListMarkedDescendants p1", "UploadFile p2 a.svg", "UploadFile p2 a.svg", "TrashPage p3", "GetPage p3",
	}; !reflect.DeepEqual(platform.Calls(), want) {
		t.Fatalf("calls\n got: %q\nwant: %q", platform.Calls(), want)
	}
}
