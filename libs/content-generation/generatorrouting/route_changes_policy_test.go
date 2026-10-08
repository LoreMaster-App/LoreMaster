package generatorrouting

import (
	"slices"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

var specs = []generatedfile.Spec{
	{Type: "test-results", Output: "docs/tests"},
	{Type: "go-docs", Input: []string{"libs/", "!libs/legacy/"}, Output: "docs/api"},
	{Type: "ts-docs", Input: []string{"apps/web/"}, Output: "docs/ts"},
	{Type: "openapi-docs", Input: []string{"api/"}, Output: "docs/openapi"},
	{Type: "python-docs", Output: "docs/py"},
	{Type: "csharp-docs", Output: "docs/cs"},
	{Type: "dart-docs", Output: "docs/dart"},
}

func TestAChangeAffectsOnlyTheGeneratorsThatReadIt(t *testing.T) {
	cases := []struct {
		file string
		want []int
	}{
		{"reports/junit.xml", []int{0}},
		{"reports/other.xml", nil},
		{"libs/core/x.go", []int{1}},
		{"libs/legacy/x.go", nil},
		{"tools/x.go", nil},
		{"apps/web/src/a.ts", []int{2}},
		{"apps/api/src/a.ts", nil},
		{"api/openapi.yaml", []int{3}},
		{"api/notes.yaml", []int{3}},
		{"other/swagger.json", nil},
		{"svc/billing/app.py", []int{4}},
		{"svc/pyproject.toml", []int{4}},
		{"src/Shop/Cart.cs", []int{5}},
		{"pkg/lib/cart.dart", []int{6}},
		{"pkg/pubspec.yaml", []int{6}},
		{"docs/guide.md", nil},
		{"image.png", nil},
	}
	for _, tc := range cases {
		if got := Route([]string{tc.file}, specs).Generators; !slices.Equal(got, tc.want) {
			t.Errorf("%s: generators %v, want %v", tc.file, got, tc.want)
		}
	}
}

func TestAGeneratorsOwnOutputNeverRunsItAgain(t *testing.T) {
	own := []generatedfile.Spec{{Type: "go-docs", Output: "docs/api"}, {Type: "test-results", Output: "docs/tests"}}

	plan := Route([]string{"docs/api/store/store.md", "docs/tests/junit.xml"}, own)

	if len(plan.Generators) != 0 {
		t.Fatalf("generators %v", plan.Generators)
	}
	if !slices.Equal(plan.Markdown, []string{"docs/api/store/store.md"}) {
		t.Fatalf("markdown %v", plan.Markdown)
	}
}

func TestSeveralChangesGiveEachAffectedGeneratorOnceAndTheMarkdownSorted(t *testing.T) {
	plan := Route([]string{"libs/b.go", "docs/z.md", "libs/a.go", "reports/junit.xml", "README.md", "docs/a.MD"}, specs)

	if !slices.Equal(plan.Generators, []int{0, 1}) || plan.Everything {
		t.Fatalf("plan %+v", plan)
	}
	if !slices.Equal(plan.Markdown, []string{"README.md", "docs/a.MD", "docs/z.md"}) {
		t.Fatalf("markdown %v", plan.Markdown)
	}
}

func TestAChangedSettingsFileCallsForEverything(t *testing.T) {
	plan := Route([]string{".lore-master.yaml"}, specs)

	if !plan.Everything || len(plan.Generators) != len(specs) {
		t.Fatalf("plan %+v", plan)
	}
}

func TestNoChangesAndNoGeneratorsCallForNothing(t *testing.T) {
	if plan := Route(nil, specs); plan.Everything || len(plan.Generators) != 0 || len(plan.Markdown) != 0 {
		t.Fatalf("plan %+v", plan)
	}
	if plan := Route([]string{"a.go", "b.md"}, nil); len(plan.Generators) != 0 || !slices.Equal(plan.Markdown, []string{"b.md"}) {
		t.Fatalf("plan %+v", plan)
	}
	if plan := Route([]string{"a.go"}, []generatedfile.Spec{{Type: "future-docs", Output: "docs/x"}}); len(plan.Generators) != 0 {
		t.Fatalf("an unknown type was routed: %+v", plan)
	}
}
