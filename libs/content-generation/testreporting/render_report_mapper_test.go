package testreporting

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func joined(files []generatedfile.File) string {
	var out strings.Builder
	for _, file := range files {
		fmt.Fprintf(&out, "<!-- file: %s -->\n%s\n", file.Path, file.Body)
	}

	return out.String()
}

func bodyOf(t *testing.T, files []generatedfile.File, path string) string {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return string(file.Body)
		}
	}
	t.Fatalf("no page %s in %v", path, joined(files))

	return ""
}

func TestRenderMatchesTheGoldenPages(t *testing.T) {
	var suites []Suite
	for _, name := range []string{"go-junit.xml", "jest-junit.xml"} {
		parsed, err := ParseJUnit(fixture(t, name), "reports/"+name)
		if err != nil {
			t.Fatal(err)
		}
		suites = append(suites, parsed...)
	}

	got := joined(RenderReport("Test results", suites))

	const golden = "testdata/report.golden.md"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if normalised := strings.ReplaceAll(string(want), "\r\n", "\n"); got != normalised {
		t.Fatalf("rendered pages differ from %s (run with UPDATE_GOLDEN=1 to refresh it)\n got:\n%s\nwant:\n%s", golden, got, normalised)
	}
}

func TestRenderIsTheSameWhateverOrderTheSuitesArriveIn(t *testing.T) {
	a := Suite{Name: "a", Source: "x.xml", Tests: 1, Cases: []Case{{Name: "t", Status: Passed}}}
	b := Suite{Name: "b", Source: "x.xml", Tests: 1, Cases: []Case{{Name: "t", Status: Passed}}}

	if joined(RenderReport("R", []Suite{a, b})) != joined(RenderReport("R", []Suite{b, a})) {
		t.Fatal("the output depends on input order")
	}
}

func TestRenderGivesEverySuiteItsOwnFileAndTitle(t *testing.T) {
	cases := []Case{{Name: "t", Status: Passed}}
	suites := []Suite{
		{Name: "Unit tests", Source: "a/junit.xml", Tests: 1, Cases: cases},
		{Name: "Unit tests", Source: "b/junit.xml", Tests: 1, Cases: cases},
		{Name: "unit.tests", Source: "c/junit.xml", Tests: 1, Cases: cases},
		{Name: "README", Source: "d/junit.xml", Tests: 1, Cases: cases},
		{Name: "Test results", Source: "e/junit.xml", Tests: 1, Cases: cases},
		{Name: "!!!", Source: "f/junit.xml", Tests: 1, Cases: cases},
	}

	files := RenderReport("Test results", suites)

	var paths []string
	for _, file := range files {
		paths = append(paths, file.Path)
		if file.Path != "README.md" && strings.Contains(strings.TrimSuffix(file.Path, ".md"), ".") {
			t.Fatalf("%s has a dot, which would nest the page under another", file.Path)
		}
	}
	if want := "README.md,unit-tests.md,unit-tests-2.md,unit-tests-3.md,readme-2.md,test-results.md,suite.md"; strings.Join(paths, ",") != want {
		t.Fatalf("paths %v, want %s", paths, want)
	}
	titles := map[string]bool{}
	for _, file := range files[1:] {
		heading := strings.SplitN(string(file.Body), "\n", 2)[0]
		if titles[strings.ToLower(heading)] {
			t.Fatalf("duplicate page title %q", heading)
		}
		titles[strings.ToLower(heading)] = true
	}
	if !strings.Contains(bodyOf(t, files, "test-results.md"), "# Test results (e/junit.xml)") {
		t.Fatal("a suite named like the index is disambiguated by its report")
	}
}

func TestRenderSaysSoWhenThereAreNoReports(t *testing.T) {
	files := RenderReport("Test results", nil)

	if len(files) != 1 || files[0].Path != "README.md" || !strings.Contains(string(files[0].Body), "No test reports were found.") {
		t.Fatalf("pages %s", joined(files))
	}
}

func TestRenderEscapesWhatWouldBreakATableOrACodeBlock(t *testing.T) {
	suite := Suite{
		Name: "tricky | [suite]", Source: "r.xml", Tests: 1, Failures: 1,
		Cases: []Case{{Name: "a|b`c", ClassName: "pkg", Status: Failed, Message: "boom", Details: "has ````` inside"}},
	}

	files := RenderReport("R", []Suite{suite})

	index := bodyOf(t, files, "README.md")
	if !strings.Contains(index, `[tricky \| \[suite\]](tricky-suite.md)`) {
		t.Fatalf("index %s", index)
	}
	body := bodyOf(t, files, "tricky-suite.md")
	if !strings.Contains(body, "| `pkg.a\\|b'c` | failed |") {
		t.Fatalf("table cell not escaped:\n%s", body)
	}
	if !strings.Contains(body, "``````text\nboom\n\nhas ````` inside\n``````\n") {
		t.Fatalf("the fence is not longer than the backticks inside:\n%s", body)
	}
}

func TestRenderBoundsHugeSuitesAndStackTraces(t *testing.T) {
	var cases []Case
	for i := 0; i < maxCaseRows+10; i++ {
		cases = append(cases, Case{Name: fmt.Sprintf("t%04d", i), Status: Passed})
	}
	cases = append(cases, Case{Name: "broken", Status: Failed, Details: strings.Repeat("x", maxDetailRunes+500)})
	suite := Suite{Name: "big", Source: "r.xml", Tests: len(cases), Failures: 1, Cases: cases}

	body := bodyOf(t, RenderReport("R", []Suite{suite}), "big.md")

	if !strings.Contains(body, fmt.Sprintf("Showing the first %d of %d tests; every failure is listed above.", maxCaseRows, len(cases))) {
		t.Fatal("the long table is not bounded")
	}
	if strings.Contains(body, fmt.Sprintf("t%04d", maxCaseRows+1)) {
		t.Fatal("rows past the bound were written")
	}
	if !strings.Contains(body, "… (truncated)") || strings.Count(body, "x") > maxDetailRunes+10 {
		t.Fatal("the stack trace is not bounded")
	}
	if !strings.Contains(body, "### `broken`") {
		t.Fatal("a failure past the table bound is still listed")
	}
}

func TestRenderListsFailingTestsOnTheIndexUpToALimit(t *testing.T) {
	var cases []Case
	for i := 0; i < maxListedFailures+3; i++ {
		cases = append(cases, Case{Name: fmt.Sprintf("t%02d", i), Status: Failed})
	}
	suite := Suite{Name: "s", Source: "r.xml", Tests: len(cases), Failures: len(cases), Cases: cases}

	index := bodyOf(t, RenderReport("R", []Suite{suite}), "README.md")

	if !strings.Contains(index, "## Failing tests") || !strings.Contains(index, "- … and 3 more") {
		t.Fatalf("index %s", index)
	}
}

func TestRenderADescriptionOfASummarisedSuiteHasNoTestTable(t *testing.T) {
	body := bodyOf(t, RenderReport("R", []Suite{{Name: "s", Source: "r.xml", Tests: 4, Failures: 1}}), "s.md")

	if !strings.Contains(body, "only its totals") || strings.Contains(body, "| Test |") {
		t.Fatalf("body %s", body)
	}
}
