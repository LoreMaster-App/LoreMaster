package testreporting

import (
	"os"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}

	return []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
}

func TestParseReadsGoStyleReportsWithFailuresErrorsAndSkips(t *testing.T) {
	suites, err := ParseJUnit(fixture(t, "go-junit.xml"), "reports/go-junit.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(suites) != 2 {
		t.Fatalf("got %d suites", len(suites))
	}

	parsing := suites[0]
	if parsing.Name != "lore-master/libs/markdown-workspace/documentparsing" || parsing.Source != "reports/go-junit.xml" || parsing.Timestamp != "2026-10-08T21:00:00+01:00" {
		t.Fatalf("suite %+v", parsing)
	}
	if parsing.Tests != 3 || parsing.Passes() != 1 || parsing.Failures != 1 || parsing.Errors != 0 || parsing.SkippedCount != 1 {
		t.Fatalf("totals %+v", parsing)
	}
	golden := parsing.Cases[1]
	if golden.Status != Failed || golden.Message != "Failed" || !strings.Contains(golden.Details, "rendered HTML differs") {
		t.Fatalf("failed case %+v", golden)
	}
	if skipped := parsing.Cases[2]; skipped.Status != Skipped || skipped.Message != "needs a symlink" {
		t.Fatalf("skipped case %+v", skipped)
	}

	storage := suites[1]
	if storage.Errors != 1 || storage.Cases[1].Status != Errored || !strings.Contains(storage.Cases[1].Details, "storage_test.go:42") {
		t.Fatalf("errored suite %+v", storage)
	}
}

func TestParseReadsAJestStyleReportAndSumsDurationsFromTheCases(t *testing.T) {
	suites, err := ParseJUnit(fixture(t, "jest-junit.xml"), "junit.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(suites) != 2 || suites[0].Name != "Pages view" || suites[0].Tests != 2 || suites[0].Failures != 0 {
		t.Fatalf("suites %+v", suites)
	}
	if got := suites[0].Seconds; got < 0.0069 || got > 0.0071 {
		t.Fatalf("seconds %v", got)
	}
}

func TestParseAcceptsASingleSuiteRoot(t *testing.T) {
	suites, err := ParseJUnit([]byte(`<testsuite name="solo" tests="1"><testcase name="a" classname="c"/></testsuite>`), "r.xml")
	if err != nil || len(suites) != 1 || suites[0].Name != "solo" || suites[0].Cases[0].ClassName != "c" {
		t.Fatalf("suites %+v, err %v", suites, err)
	}
}

func TestParseFlattensNestedSuitesAndDropsEmptyOnes(t *testing.T) {
	data := `<testsuites>
	  <testsuite name="outer"><testcase name="a"/>
	    <testsuite name="inner"><testcase name="b"><failure message="x"/></testcase></testsuite>
	  </testsuite>
	  <testsuite name="empty"/>
	  <testsuite><testcase name="n"/></testsuite>
	</testsuites>`
	suites, err := ParseJUnit([]byte(data), "r.xml")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, suite := range suites {
		names = append(names, suite.Name)
	}
	if want := "outer,outer / inner,(unnamed suite)"; strings.Join(names, ",") != want {
		t.Fatalf("names %v, want %s", names, want)
	}
}

func TestParseUsesTheTotalsOfASummarisedSuite(t *testing.T) {
	suites, err := ParseJUnit([]byte(`<testsuite name="s" tests="10" failures="2" errors="1" skipped="3" time="1,234.5"/>`), "r.xml")
	if err != nil {
		t.Fatal(err)
	}
	suite := suites[0]
	if suite.Tests != 10 || suite.Passes() != 4 || suite.Seconds != 1234.5 || len(suite.Cases) != 0 {
		t.Fatalf("suite %+v", suite)
	}
}

func TestParseRefusesWhatIsNotJUnit(t *testing.T) {
	cases := map[string]string{
		"a different root": `<html><body/></html>`,
		"broken xml":       `<testsuites><testsuite`,
		"no elements":      `   `,
	}
	for name, data := range cases {
		if _, err := ParseJUnit([]byte(data), "r.xml"); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
	if _, err := ParseJUnit([]byte(`<html/>`), "r.xml"); err == nil || !strings.Contains(err.Error(), "<html>") {
		t.Fatalf("the error names the root: %v", err)
	}
}
