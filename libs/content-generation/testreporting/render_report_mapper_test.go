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

func TestRenderGivesEveryReportItsOwnFileAndTitle(t *testing.T) {
	cases := []Case{{Name: "t", Status: Passed}}
	suites := []Suite{
		{Name: "Unit tests", Source: "a/b.xml", Tests: 1, Cases: cases},
		{Name: "Unit tests", Source: "a-b.xml", Tests: 1, Cases: cases},
		{Name: "Unit tests", Source: "README", Tests: 1, Cases: cases},
		{Name: "Unit tests", Source: "!!!", Tests: 1, Cases: cases},
	}

	files := RenderReport("Test results", suites)

	var paths []string
	for _, file := range files {
		paths = append(paths, file.Path)
		if file.Path != "README.md" && strings.Contains(strings.TrimSuffix(file.Path, ".md"), ".") {
			t.Fatalf("%s has a dot, which would nest the page under another", file.Path)
		}
	}
	if want := "README.md,report.md,readme-2.md,a-b-xml.md,a-b-xml-2.md"; strings.Join(paths, ",") != want {
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
	if !strings.Contains(bodyOf(t, files, "a-b-xml-2.md"), "# Test results: a/b.xml\n") {
		t.Fatal("a page is titled by the report it shows")
	}
}

func TestRenderWritesOnePageForOneReportHoweverManySuitesItHas(t *testing.T) {
	var suites []Suite
	for i := 0; i < 300; i++ {
		suites = append(suites, Suite{Name: fmt.Sprintf("suite %03d", i), Source: "junit.xml", Tests: 2, Cases: []Case{{Name: "a", Status: Passed}, {Name: "b", Status: Passed}}})
	}

	files := RenderReport("Test results", suites)

	if len(files) != 1 || files[0].Path != "README.md" {
		t.Fatalf("pages %v, want only README.md", pathsOf(generatedfile.Output{Files: files}))
	}
	body := string(files[0].Body)
	for _, expected := range []string{"# Test results\n", "600 tests in 300 suites: 600 passed, 0 failed, 0 errors, 0 skipped", "| suite 000 | 2 | 2 | 0 | 0 | 0 |", "| suite 299 | 2 | 2 | 0 | 0 | 0 |"} {
		if !strings.Contains(body, expected) {
			t.Errorf("the page lacks %q", expected)
		}
	}
	if strings.Contains(body, "## Failures") {
		t.Error("a run with no failure has no Failures section")
	}
}

func TestRenderWithSeveralReportsMakesAnIndexAndAPageForEach(t *testing.T) {
	passing := []Case{{Name: "t", Status: Passed}}
	files := RenderReport("R", []Suite{
		{Name: "s1", Source: "apps/a/junit.xml", Tests: 1, Cases: passing},
		{Name: "s2", Source: "apps/a/junit.xml", Tests: 1, Cases: passing},
		{Name: "s3", Source: "libs/b/junit.xml", Tests: 1, Cases: passing},
	})

	if got := strings.Join(pathsOf(generatedfile.Output{Files: files}), ","); got != "README.md,apps-a-junit-xml.md,libs-b-junit-xml.md" {
		t.Fatalf("paths %s", got)
	}
	index := bodyOf(t, files, "README.md")
	for _, expected := range []string{"3 tests in 3 suites in 2 reports: 3 passed", "| [apps/a/junit.xml](apps-a-junit-xml.md) | 2 | 2 | 2 | 0 | 0 | 0 |", "| [libs/b/junit.xml](libs-b-junit-xml.md) | 1 | 1 | 1 | 0 | 0 | 0 |"} {
		if !strings.Contains(index, expected) {
			t.Errorf("index lacks %q:\n%s", expected, index)
		}
	}
	if page := bodyOf(t, files, "apps-a-junit-xml.md"); !strings.Contains(page, "| s1 | 1 | 1 | 0 | 0 | 0 |") || strings.Contains(page, "| s3 |") {
		t.Errorf("the page of a report has the suites of that report only:\n%s", page)
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

	body := bodyOf(t, files, "README.md")
	if !strings.Contains(body, `| tricky \| \[suite\] | 1 | 0 | 1 | 0 | 0 |`) {
		t.Fatalf("table cell not escaped:\n%s", body)
	}
	if !strings.Contains(body, "#### `pkg.a\\|b'c`") {
		t.Fatalf("test name not escaped:\n%s", body)
	}
	if !strings.Contains(body, "``````text\nboom\n\nhas ````` inside\n``````\n") {
		t.Fatalf("the fence is not longer than the backticks inside:\n%s", body)
	}
}

func TestRenderCountsPassingTestsAndBoundsStackTraces(t *testing.T) {
	var cases []Case
	for i := 0; i < 600; i++ {
		cases = append(cases, Case{Name: fmt.Sprintf("t%04d", i), Status: Passed})
	}
	cases = append(cases, Case{Name: "broken", Status: Failed, Details: strings.Repeat("x", maxDetailRunes+500)})
	suite := Suite{Name: "big", Source: "r.xml", Tests: len(cases), Failures: 1, Cases: cases}

	body := bodyOf(t, RenderReport("R", []Suite{suite}), "README.md")

	if !strings.Contains(body, "| big | 601 | 600 | 1 | 0 | 0 |") {
		t.Fatalf("the suite row counts the passing tests:\n%s", body)
	}
	if strings.Contains(body, "t0001") {
		t.Fatal("a test that passed was listed")
	}
	if !strings.Contains(body, "… (truncated)") || strings.Count(body, "x") > maxDetailRunes+10 {
		t.Fatal("the stack trace is not bounded")
	}
	if !strings.Contains(body, "#### `broken`") {
		t.Fatal("a failure is always listed")
	}
}

func TestRenderListsFailingSuitesOnTheIndexUpToALimit(t *testing.T) {
	var suites []Suite
	for _, source := range []string{"a.xml", "b.xml"} {
		for i := 0; i < maxListedFailures/2+5; i++ {
			suites = append(suites, Suite{Name: fmt.Sprintf("s%02d", i), Source: source, Tests: 1, Failures: 1, Cases: []Case{{Name: "t", Status: Failed}}})
		}
	}

	index := bodyOf(t, RenderReport("R", suites), "README.md")

	if !strings.Contains(index, "## Failing suites") || !strings.Contains(index, "- … and 10 more") {
		t.Fatalf("index %s", index)
	}
}

func TestRenderASummarisedSuiteHasItsTotalsAndNoFailureSection(t *testing.T) {
	body := bodyOf(t, RenderReport("R", []Suite{{Name: "s", Source: "r.xml", Tests: 4, Failures: 1}}), "README.md")

	if !strings.Contains(body, "| s | 4 | 3 | 1 | 0 | 0 |") || strings.Contains(body, "## Failures") {
		t.Fatalf("body %s", body)
	}
}
