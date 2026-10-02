package slicecheck

import (
	"path/filepath"
	"reflect"
	"testing"
)

// TestRepositoryFollowsSliceRules is the check itself: the real repository.
func TestRepositoryFollowsSliceRules(t *testing.T) {
	findings, err := Violations(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		t.Error(finding)
	}
}

// TestDetectsEveryKindOfViolation proves the check can fail: testdata/violations
// plants one instance of each rule, and testdata/clean mirrors a valid layout.
func TestDetectsEveryKindOfViolation(t *testing.T) {
	clean, err := Violations(filepath.Join("testdata", "clean"))
	if err != nil {
		t.Fatal(err)
	}
	if len(clean) != 0 {
		t.Fatalf("clean tree reported %v", clean)
	}

	got, err := Violations(filepath.Join("testdata", "violations"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`apps/engine/server.go: only doc.go, main.go, main_test.go may sit at a project root; move it into a slice package`,
		`libs/lore/Bad/x_model.go: package clause must equal the directory name "Bad"`,
		`libs/lore/Bad: slice package names are lowercase letters and digits only`,
		`libs/lore/helpers: "helpers" says nothing about an outcome; name the slice after what it delivers`,
		`libs/lore/lore.go: only doc.go may sit at a project root; move it into a slice package`,
		`libs/lore/sync/nested: a slice is flat; only testdata/ may sit inside it`,
		`libs/lore/sync/plan.go: name it <snake>_<role>.go with one role from the ADR`,
		`libs/lore/sync/plan_use_case.go: package clause must equal the directory name "sync"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("findings differ\n got: %q\nwant: %q", got, want)
	}
}
