package syncjournal

import (
	"strconv"
	"testing"
	"time"

	"lore-master/libs/documentation-sync/platformport"
)

func TestJournalsAreSavedLoadedAndListedNewestFirst(t *testing.T) {
	repository := NewRepository(t.TempDir())
	older := Journal{ID: NewID(time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)), Pages: []PageChange{{Kind: Created, PageID: "1", Title: "A", AfterVersion: 1}}}
	newer := Journal{ID: NewID(time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)), Pages: []PageChange{{Kind: Updated, PageID: "2", Title: "B", Before: &platformport.PageSnapshot{ID: "2", Title: "B0", Markup: "<p>old</p>"}, AfterVersion: 3}}}
	for _, journal := range []Journal{older, newer} {
		if err := repository.Save(journal); err != nil {
			t.Fatal(err)
		}
	}

	listed, err := repository.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 || listed[0].ID != newer.ID {
		t.Fatalf("listed = %+v", listed)
	}
	loaded, err := repository.Load(newer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Pages[0].Before.Markup != "<p>old</p>" || loaded.Summarize().Updated != 1 {
		t.Fatalf("loaded = %+v", loaded)
	}
	if err := repository.Remove(newer.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Load(newer.ID); err == nil {
		t.Fatal("a removed journal still loads")
	}
}

func TestOnlyTheLatestJournalsAreKept(t *testing.T) {
	repository := NewRepository(t.TempDir())
	for index := 0; index < keep+5; index++ {
		id := "2026101" + strconv.Itoa(index/10) + "T0000" + strconv.Itoa(index%10) + "0Z-aaaaaa"
		if err := repository.Save(Journal{ID: id}); err != nil {
			t.Fatal(err)
		}
	}

	listed, err := repository.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != keep {
		t.Fatalf("kept %d journals, want %d", len(listed), keep)
	}
}

func TestAJournalOfAnotherWorkspaceIsInAnotherFolder(t *testing.T) {
	if DirFor("/cache", "/work/a") == DirFor("/cache", "/work/b") {
		t.Fatal("two workspaces share a journal folder")
	}
	if DirFor("/cache", "/work/a") != DirFor("/cache", "/work/a") {
		t.Fatal("a workspace's folder is not stable")
	}
}
