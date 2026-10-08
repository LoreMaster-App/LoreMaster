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
	// maxCaseRows bounds a suite page's table: a suite of thousands of passing tests would
	// otherwise exceed what a wiki page holds. Failures are never cut.
	maxCaseRows = 500
	// maxDetailRunes bounds one failure's stack trace.
	maxDetailRunes = 4000
	// maxListedFailures bounds the failing-tests list on the index page.
	maxListedFailures = 50

	indexPath = "README.md"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// page is one suite as it will be written: its file, the title it will have, and its tests.
type page struct {
	suite Suite
	file  string
	title string
}

// RenderReport turns the suites into pages: an index named README.md, so every suite page
// nests under it, and one page per suite. Names are made unique (file names and page
// titles, which a wiki space requires) in a stable order, so the same reports always give
// the same pages.
func RenderReport(title string, suites []Suite) []generatedfile.File {
	ordered := slices.Clone(suites)
	slices.SortStableFunc(ordered, func(a, b Suite) int {
		if byName := strings.Compare(a.Source, b.Source); byName != 0 {
			return byName
		}

		return strings.Compare(a.Name, b.Name)
	})
	pages := nameSuites(title, ordered)

	files := []generatedfile.File{{Path: indexPath, Body: []byte(renderIndex(title, pages))}}
	for _, p := range pages {
		files = append(files, generatedfile.File{Path: p.file, Body: []byte(renderSuite(p))})
	}

	return files
}

// nameSuites gives each suite a file name and a page title that no other page has.
func nameSuites(indexTitle string, suites []Suite) []page {
	files := map[string]bool{"readme": true}
	titles := map[string]bool{strings.ToLower(indexTitle): true}
	pages := make([]page, 0, len(suites))
	for _, suite := range suites {
		slug := slugOf(suite.Name)
		unique := slug
		for n := 2; files[unique]; n++ {
			unique = fmt.Sprintf("%s-%d", slug, n)
		}
		files[unique] = true

		heading := suite.Name
		if titles[strings.ToLower(heading)] {
			heading = fmt.Sprintf("%s (%s)", suite.Name, suite.Source)
		}
		for n := 2; titles[strings.ToLower(heading)]; n++ {
			heading = fmt.Sprintf("%s (%s) #%d", suite.Name, suite.Source, n)
		}
		titles[strings.ToLower(heading)] = true

		pages = append(pages, page{suite: suite, file: unique + ".md", title: heading})
	}

	return pages
}

// slugOf is a file name stem: lower-case letters and digits, with no dots, because a dotted
// file name nests the page under another one.
func slugOf(name string) string {
	slug := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		return "suite"
	}

	return slug
}

func renderIndex(title string, pages []page) string {
	var totals Suite
	for _, p := range pages {
		totals.Tests += p.suite.Tests
		totals.Failures += p.suite.Failures
		totals.Errors += p.suite.Errors
		totals.SkippedCount += p.suite.SkippedCount
		totals.Seconds += p.suite.Seconds
	}

	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	if len(pages) == 0 {
		out.WriteString("No test reports were found.\n")

		return out.String()
	}
	fmt.Fprintf(&out, "%s in %s: %d passed, %d failed, %s, %d skipped, in %s.\n\n",
		plural(totals.Tests, "test"), plural(len(pages), "suite"), totals.Passes(), totals.Failures, plural(totals.Errors, "error"), totals.SkippedCount, duration(totals.Seconds))

	writeFailingList(&out, pages)

	out.WriteString("| Suite | Tests | Passed | Failed | Errors | Skipped | Time |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, p := range pages {
		fmt.Fprintf(&out, "| [%s](%s) | %d | %d | %d | %d | %d | %s |\n",
			cell(p.title), p.file, p.suite.Tests, p.suite.Passes(), p.suite.Failures, p.suite.Errors, p.suite.SkippedCount, duration(p.suite.Seconds))
	}

	return out.String()
}

// writeFailingList names the tests that failed or errored, so the index answers "what is
// broken" without opening every suite.
func writeFailingList(out *strings.Builder, pages []page) {
	var lines []string
	for _, p := range pages {
		for _, test := range p.suite.Cases {
			if test.Status == Failed || test.Status == Errored {
				lines = append(lines, fmt.Sprintf("- [%s](%s): %s (%s)", cell(p.title), p.file, code(caseName(test)), test.Status))
			}
		}
	}
	if len(lines) == 0 {
		return
	}
	out.WriteString("## Failing tests\n\n")
	for i, line := range lines {
		if i == maxListedFailures {
			fmt.Fprintf(out, "- … and %d more\n", len(lines)-maxListedFailures)

			break
		}
		out.WriteString(line + "\n")
	}
	out.WriteString("\n")
}

func renderSuite(p page) string {
	suite := p.suite
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", p.title)
	fmt.Fprintf(&out, "%s: %d passed, %d failed, %s, %d skipped, in %s. Report: %s.", plural(suite.Tests, "test"),
		suite.Passes(), suite.Failures, plural(suite.Errors, "error"), suite.SkippedCount, duration(suite.Seconds), code(suite.Source))
	if suite.Timestamp != "" {
		fmt.Fprintf(&out, " Run at %s.", suite.Timestamp)
	}
	out.WriteString("\n\n")

	var broken []Case
	for _, test := range suite.Cases {
		if test.Status == Failed || test.Status == Errored {
			broken = append(broken, test)
		}
	}
	if len(broken) > 0 {
		out.WriteString("## Failures\n\n")
		for _, test := range broken {
			fmt.Fprintf(&out, "### %s\n\n%s.\n\n%s\n", code(caseName(test)), capitalise(string(test.Status)), markdownwriting.CodeBlock("text", failureText(test)))
		}
		out.WriteString("\n")
	}

	if len(suite.Cases) == 0 {
		out.WriteString("The report lists no individual tests for this suite, only its totals.\n")

		return out.String()
	}
	out.WriteString("## Tests\n\n| Test | Result | Time |\n|---|---|---:|\n")
	for i, test := range suite.Cases {
		if i == maxCaseRows {
			break
		}
		fmt.Fprintf(&out, "| %s | %s | %s |\n", code(caseName(test)), test.Status, duration(test.Seconds))
	}
	if len(suite.Cases) > maxCaseRows {
		fmt.Fprintf(&out, "\nShowing the first %d of %d tests; every failure is listed above.\n", maxCaseRows, len(suite.Cases))
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

// caseName is how a test is shown: its name, qualified by its class when that adds something.
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
