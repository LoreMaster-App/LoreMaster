package documenttree

import "testing"

// The nesting conventions must stay aligned with the Rule constants the policy uses: the
// first four entries are those rules, in precedence order. A new or renamed rule that is
// not reflected here is the drift this test exists to catch.
func TestNestingConventionsCoverTheRulesInOrder(t *testing.T) {
	conventions := NestingConventions()
	wantNesting := []Rule{RuleExplicitParent, RuleDottedName, RuleDirectoryIndex, RuleSelectedParent}
	if len(conventions) < len(wantNesting) {
		t.Fatalf("expected at least %d conventions, got %d", len(wantNesting), len(conventions))
	}
	for i, rule := range wantNesting {
		if conventions[i].Key != string(rule) {
			t.Errorf("convention %d: key %q, want %q", i, conventions[i].Key, rule)
		}
		if conventions[i].Summary == "" {
			t.Errorf("convention %q has no summary", conventions[i].Key)
		}
	}
	// The title rules follow and must be present.
	for _, key := range []string{"title", "title-prefix", "title-unique"} {
		found := false
		for _, c := range conventions {
			if c.Key == key {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("missing convention %q", key)
		}
	}
}
