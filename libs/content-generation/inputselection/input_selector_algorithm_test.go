package inputselection

import "testing"

func TestSelectorSelectsWhatTheIncludePatternsMatch(t *testing.T) {
	selector := NewSelector([]string{"reports/", "**/junit*.xml"}, []string{"ignored-default"})

	for path, want := range map[string]bool{
		"reports/a.xml":     true,
		"ci/junit-go.xml":   true,
		"junit.xml":         true,
		"build/other.xml":   false,
		"ignored-default":   false,
		"src/reports/b.xml": true,
	} {
		if got := selector.Selects(path, false); got != want {
			t.Errorf("%s: got %v, want %v", path, got, want)
		}
	}
}

func TestSelectorExcludesWhatANegatedPatternMatchesWhateverSelectedIt(t *testing.T) {
	selector := NewSelector([]string{"reports/", "!reports/old/"}, nil)

	if !selector.Selects("reports/new.xml", false) || selector.Selects("reports/old/a.xml", false) {
		t.Fatal("the exclusion does not apply")
	}
}

func TestSelectorFallsBackToTheDefaultsWhenNothingIsIncluded(t *testing.T) {
	selector := NewSelector([]string{"!vendor/"}, []string{"**/junit*.xml"})

	if !selector.Selects("a/junit.xml", false) || selector.Selects("a/other.xml", false) || selector.Selects("vendor/junit.xml", false) {
		t.Fatal("the defaults plus the exclusion do not apply")
	}
}

func TestSelectorWithNoPatternsAndNoDefaultsSelectsEverything(t *testing.T) {
	selector := NewSelector(nil, nil)

	if !selector.Selects("anything/at/all.go", false) || !selector.Selects("libs", true) {
		t.Fatal("everything should be selected")
	}
}

func TestSelectorMatchesFoldersWithTheirTrailingSlash(t *testing.T) {
	selector := NewSelector([]string{"libs/", "!libs/internal/"}, nil)

	if !selector.Selects("libs/a", true) || selector.Selects("libs/internal", true) {
		t.Fatal("folder patterns do not match folders")
	}
}
