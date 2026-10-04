package syncplanning

import (
	"reflect"
	"testing"
)

func TestDetectChange(t *testing.T) {
	synced := remoteState{annotationVersion: 3, annotationHash: "h", remoteVersion: 3, remoteParentID: "p", remoteTitle: "T"}
	same := localState{contentHash: "h", title: "T", parentPageID: "p"}
	edited := synced
	edited.remoteVersion = 4

	cases := []struct {
		name    string
		remote  remoteState
		local   localState
		twoWay  bool
		kind    ActionKind
		changes []Change
	}{
		{"nothing changed", synced, same, false, Unchanged, nil},
		{"body", synced, localState{"h2", "T", "p", false}, false, Update, []Change{ChangeContent}},
		{"parent", synced, localState{"h", "T", "q", false}, false, Move, []Change{ChangeParent}},
		{"parent created in this sync", synced, localState{"h", "T", "", false}, false, Move, []Change{ChangeParent}},
		{"parent created in this sync, page was at the space root", remoteState{3, "h", 3, "", "T"}, localState{"h", "T", "", false}, false, Move, []Change{ChangeParent}},
		{"what the page shows alone", synced, localState{contentHash: "h", title: "T", parentPageID: "p", renderedChanged: true}, false, Update, []Change{ChangeContent}},
		{"title", synced, localState{"h", "U", "p", false}, false, RenameTitle, []Change{ChangeTitle}},
		{"move and rename", synced, localState{"h", "U", "q", false}, false, Move, []Change{ChangeParent, ChangeTitle}},
		{"everything", synced, localState{"h2", "U", "q", false}, false, Update, []Change{ChangeContent, ChangeParent, ChangeTitle}},
		{"remote edit alone", edited, same, false, Conflict, nil},
		{"remote edit beats a local edit", edited, localState{"h2", "U", "q", false}, false, Conflict, nil},

		// Two-way sync (#95): the remote-changed case splits on whether the file changed too.
		{"two-way, remote edit, file unchanged, pulls", edited, same, true, Pull, []Change{ChangeContent}},
		{"two-way, remote edit, file body changed, conflicts", edited, localState{"h2", "T", "p", false}, true, Conflict, nil},
		{"two-way, remote edit, what the page shows changed, conflicts", edited, localState{contentHash: "h", title: "T", parentPageID: "p", renderedChanged: true}, true, Conflict, nil},
		{"two-way, remote edit, only local title changed, pulls (file body unchanged)", edited, localState{"h", "U", "p", false}, true, Pull, []Change{ChangeContent}},
		{"two-way, no remote edit, body changed, still pushes an update", synced, localState{"h2", "T", "p", false}, true, Update, []Change{ChangeContent}},
		{"two-way, no remote edit, parent changed, still pushes a move", synced, localState{"h", "T", "q", false}, true, Move, []Change{ChangeParent}},
		{"two-way, nothing changed", synced, same, true, Unchanged, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, changes := detectChange(c.remote, c.local, c.twoWay)
			if kind != c.kind || !reflect.DeepEqual(changes, c.changes) {
				t.Fatalf("got %s %v, want %s %v", kind, changes, c.kind, c.changes)
			}
		})
	}
}
