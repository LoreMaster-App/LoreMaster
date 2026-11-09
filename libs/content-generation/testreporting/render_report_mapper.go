package testreporting

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/markdownwriting"
)

const (
	// maxDetailRunes bounds one failure's stack trace.
	maxDetailRunes = 4000
	// maxListedFailures bounds the list of failing suites on the index page.
	maxListedFailures = 50

	indexPath = "README.md"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// report is the suites of one report file as they will be written: the file of the page, the
// title it will have, and the suites.
type report struct {
	source string
	file   string
	title  string
	suites []Suite
}

// RenderReport turns the suites into pages, one page for each report file. A page has its totals,
// every failure with its message and stack, and a table with a row for each suite. The tests that
// passed are counted and not listed, so a run of thousands of tests is a few pages and not one
// for every suite. One report is the index page itself. Several reports are an index with a row
// for each report, and its pages nest under it. Names are made unique in a stable order, so the same
// reports always give the same pages.
func RenderReport(title string, suites []Suite) []generatedfile.File {
	ordered := slices.Clone(suites)
	slices.SortStableFunc(ordered, func(a, b Suite) int {
		if bySource := strings.Compare(a.Source, b.Source); bySource != 0 {
			return bySource
		}

		return strings.Compare(a.Name, b.Name)
	})
	reports := groupByReport(title, ordered)

	switch len(reports) {
	case 0:
		return []generatedfile.File{{Path: indexPath, Body: []byte("# " + title + "\n\nNo test reports were found.\n")}}
	case 1:
		return []generatedfile.File{{Path: indexPath, Body: []byte(renderReport(title, reports[0]))}}
	}

	files := []generatedfile.File{{Path: indexPath, Body: []byte(renderIndex(title, reports))}}
	for _, r := range reports {
		files = append(files, generatedfile.File{Path: r.file, Body: []byte(renderReport(r.title, r))})
	}

	return files
}

// groupByReport puts the suites of each report file together, and gives each report a file name
// and a page title that no other page has. The suites arrive sorted by report.
func groupByReport(indexTitle string, suites []Suite) []report {
	files := map[string]bool{"readme": true}
	var reports []report
	for _, suite := range suites {
		if n := len(reports); n > 0 && reports[n-1].source == suite.Source {
			reports[n-1].suites = append(reports[n-1].suites, suite)

			continue
		}
		slug := slugOf(suite.Source)
		unique := slug
		for n := 2; files[unique]; n++ {
			unique = fmt.Sprintf("%s-%d", slug, n)
		}
		files[unique] = true

		reports = append(reports, report{source: suite.Source, file: unique + ".md", title: indexTitle + ": " + suite.Source, suites: []Suite{suite}})
	}

	return reports
}

// slugOf is a file name stem: lower-case letters and digits, with no dots, because a dotted
// file name nests the page under another one.
func slugOf(name string) string {
	slug := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		return "report"
	}

	return slug
}

// totalsOf adds the counts of the suites.
func totalsOf(suites []Suite) Suite {
	var totals Suite
	for _, suite := range suites {
		totals.Tests += suite.Tests
		totals.Failures += suite.Failures
		totals.Errors += suite.Errors
		totals.SkippedCount += suite.SkippedCount
		totals.Seconds += suite.Seconds
	}

	return totals
}

// sentence sums the counts up in one line.
func sentence(totals Suite) string {
	return fmt.Sprintf("%d passed, %d failed, %s, %d skipped, in %s", totals.Passes(), totals.Failures, plural(totals.Errors, "error"), totals.SkippedCount, duration(totals.Seconds))
}

// isBroken tells whether a test failed or ended in an error.
func isBroken(test Case) bool {
	return test.Status == Failed || test.Status == Errored
}

func renderIndex(title string, reports []report) string {
	var all []Suite
	for _, r := range reports {
		all = append(all, r.suites...)
	}

	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	fmt.Fprintf(&out, "%s in %s in %s: %s.\n\n", plural(totalsOf(all).Tests, "test"), plural(len(all), "suite"), plural(len(reports), "report"), sentence(totalsOf(all)))

	writeFailingList(&out, reports)

	out.WriteString("| Report | Suites | Tests | Passed | Failed | Errors | Skipped | Time |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range reports {
		totals := totalsOf(r.suites)
		fmt.Fprintf(&out, "| [%s](%s) | %d | %d | %d | %d | %d | %d | %s |\n",
			cell(r.source), r.file, len(r.suites), totals.Tests, totals.Passes(), totals.Failures, totals.Errors, totals.SkippedCount, duration(totals.Seconds))
	}

	return out.String()
}

// writeFailingList names the suites that have a failed test or an error, so the index answers
// "what is broken" without opening every report.
func writeFailingList(out *strings.Builder, reports []report) {
	var lines []string
	for _, r := range reports {
		for _, suite := range r.suites {
			if broken := len(brokenIn(suite)); broken > 0 {
				lines = append(lines, fmt.Sprintf("- [%s](%s): %s (%s)", cell(r.source), r.file, cell(suite.Name), plural(broken, "test")))
			}
		}
	}
	if len(lines) == 0 {
		return
	}
	out.WriteString("## Failing suites\n\n")
	for i, line := range lines {
		if i == maxListedFailures {
			fmt.Fprintf(out, "- … and %d more\n", len(lines)-maxListedFailures)

			break
		}
		out.WriteString(line + "\n")
	}
	out.WriteString("\n")
}

// brokenIn is the tests of a suite that failed or ended in an error.
func brokenIn(suite Suite) []Case {
	var broken []Case
	for _, test := range suite.Cases {
		if isBroken(test) {
			broken = append(broken, test)
		}
	}

	return broken
}

func renderReport(title string, r report) string {
	totals := totalsOf(r.suites)
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	fmt.Fprintf(&out, "%s in %s: %s. Report: %s.", plural(totals.Tests, "test"), plural(len(r.suites), "suite"), sentence(totals), code(r.source))
	if stamp := r.suites[0].Timestamp; stamp != "" {
		fmt.Fprintf(&out, " Run at %s.", stamp)
	}
	out.WriteString("\n\n")

	var heading bool
	for _, suite := range r.suites {
		broken := brokenIn(suite)
		if len(broken) == 0 {
			continue
		}
		if !heading {
			out.WriteString("## Failures\n\n")
			heading = true
		}
		fmt.Fprintf(&out, "### %s\n\n", cell(suite.Name))
		for _, test := range broken {
			fmt.Fprintf(&out, "#### %s\n\n%s.\n\n%s\n", code(caseName(test)), capitalise(string(test.Status)), markdownwriting.CodeBlock("text", failureText(test)))
		}
	}
	if heading {
		out.WriteString("\n")
	}

	out.WriteString("## Suites\n\n| Suite | Tests | Passed | Failed | Errors | Skipped | Time |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, suite := range r.suites {
		fmt.Fprintf(&out, "| %s | %d | %d | %d | %d | %d | %s |\n",
			cell(suite.Name), suite.Tests, suite.Passes(), suite.Failures, suite.Errors, suite.SkippedCount, duration(suite.Seconds))
	}

	return out.String()
}

// failureText is a failure's message followed by its details, bounded.
func failureText(test Case) string {
	text := strings.TrimSpace(strings.Join([]string{test.Message, test.Details}, "\n\n"))
	if text == "" {
		return "(the report gives no message)"
	}
	runes := []rune(text)
	if len(runes) > maxDetailRunes {
		return string(runes[:maxDetailRunes]) + "\n… (truncated)"
	}

	return text
}

func caseName(test Case) string {
	switch {
	case test.ClassName == "" || test.ClassName == test.Name:
		return test.Name
	case test.Name == "":
		return test.ClassName
	}

	return test.ClassName + "." + test.Name
}

// code is text as inline code safe inside a table cell.
func code(text string) string {
	return "`" + strings.NewReplacer("`", "'", "|", `\|`, "\n", " ", "\r", "").Replace(strings.TrimSpace(text)) + "`"
}

// cell is text safe inside a table cell or a link label.
func cell(text string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ", "\r", "", "[", `\[`, "]", `\]`, "<", "&lt;").Replace(strings.TrimSpace(text))
}

func duration(seconds float64) string {
	return fmt.Sprintf("%.3fs", seconds)
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}

	return fmt.Sprintf("%d %ss", n, noun)
}

func capitalise(text string) string {
	if text == "" {
		return text
	}

	return strings.ToUpper(text[:1]) + text[1:]
}
