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
		kind    ActionKind
		changes []Change
	}{
		{"nothing changed", synced, same, Unchanged, nil},
		{"body", synced, localState{"h2", "T", "p", false}, Update, []Change{ChangeContent}},
		{"parent", synced, localState{"h", "T", "q", false}, Move, []Change{ChangeParent}},
		{"parent created in this sync", synced, localState{"h", "T", "", false}, Move, []Change{ChangeParent}},
		{"parent created in this sync, page was at the space root", remoteState{3, "h", 3, "", "T"}, localState{"h", "T", "", false}, Move, []Change{ChangeParent}},
		{"an attachment alone", synced, localState{contentHash: "h", title: "T", parentPageID: "p", attachmentsChanged: true}, Update, []Change{ChangeContent}},
		{"title", synced, localState{"h", "U", "p", false}, RenameTitle, []Change{ChangeTitle}},
		{"move and rename", synced, localState{"h", "U", "q", false}, Move, []Change{ChangeParent, ChangeTitle}},
		{"everything", synced, localState{"h2", "U", "q", false}, Update, []Change{ChangeContent, ChangeParent, ChangeTitle}},
		{"remote edit alone", edited, same, Conflict, nil},
		{"remote edit beats a local edit", edited, localState{"h2", "U", "q", false}, Conflict, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, changes := detectChange(c.remote, c.local)
			if kind != c.kind || !reflect.DeepEqual(changes, c.changes) {
				t.Fatalf("got %s %v, want %s %v", kind, changes, c.kind, c.changes)
			}
		})
	}
}
