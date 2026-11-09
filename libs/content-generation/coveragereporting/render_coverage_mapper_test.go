package coveragereporting

import (
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func fileWith(path string, hits ...int) FileCoverage {
	file := newFileCoverage(path)
	for i, h := range hits {
		file.recordLine(i+1, h)
	}

	return *file
}

func pageAt(t *testing.T, pages []generatedfile.File, path string) string {
	t.Helper()
	for _, page := range pages {
		if page.Path == path {
			return string(page.Body)
		}
	}
	t.Fatalf("no page %s in %v", path, pathsOf(pages))

	return ""
}

func pathsOf(pages []generatedfile.File) []string {
	out := make([]string, len(pages))
	for i, page := range pages {
		out[i] = page.Path
	}

	return out
}

// deepEnough is a page depth no test tree reaches: every folder gets its own page.
const deepEnough = 99

// projects is a code base of two apps and a library, each with folders of its own.
func projects() []FileCoverage {
	return []FileCoverage{
		fileWith("apps/web/src/main.ts", 1, 0),
		fileWith("apps/web/src/cart/cart.ts", 1, 1),
		fileWith("apps/web/src/cart/pay/card.ts", 0, 0),
		fileWith("libs/core/src/index.ts", 1),
	}
}

func TestRenderCoverageGivesAPageToEachFolderDownToThePageDepth(t *testing.T) {
	pages := RenderCoverage("", projects(), 2)

	want := []string{"README.md", "apps/README.md", "apps/web/README.md", "libs/README.md", "libs/core/README.md"}
	if got := pathsOf(pages); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("pages = %v, want %v", got, want)
	}
	root := pageAt(t, pages, "README.md")
	if !strings.Contains(root, "[apps/](./apps/README.md)") {
		t.Errorf("a folder above the page depth links to its page:\n%s", root)
	}
	web := pageAt(t, pages, "apps/web/README.md")
	for _, expected := range []string{
		"| src/ |",
		"## src/\n\n**Code coverage 50.00%** · Total 2 lines · Covered 1 · Partial 0 · Not covered 1\n\n",
		"| `main.ts` | █████░░░░░ 50.00% | 2 | 1 | 0 | 1 |",
		"## src/cart/\n",
		"## src/cart/pay/\n",
		"| `card.ts` | ░░░░░░░░░░ 0.00% | 2 | 0 | 0 | 2 |",
	} {
		if !strings.Contains(web, expected) {
			t.Errorf("the page at the page depth lacks %q:\n%s", expected, web)
		}
	}
	if strings.Contains(web, "](./") {
		t.Errorf("a folder below the page depth has no page to link to:\n%s", web)
	}
}

func TestRenderCoverageWithPageDepthZeroWritesOnePage(t *testing.T) {
	pages := RenderCoverage("", projects(), 0)

	if got := pathsOf(pages); strings.Join(got, ",") != "README.md" {
		t.Fatalf("pages = %v, want only README.md", got)
	}
	body := pageAt(t, pages, "README.md")
	for _, expected := range []string{"## apps/web/src/\n", "## apps/web/src/cart/pay/\n", "## libs/core/src/\n"} {
		if !strings.Contains(body, expected) {
			t.Errorf("the one page lacks %q:\n%s", expected, body)
		}
	}
}

func TestRenderCoverageMakesOnePagePerFolderStartingWhereTheCodeDoes(t *testing.T) {
	pages := RenderCoverage("", []FileCoverage{
		fileWith("libs/app/src/cart.go", 1, 1, 0, 0),
		fileWith("libs/app/src/pay/card.go", 1, 1, 1, 1),
		fileWith("libs/app/src/pay/refund.go", 0),
	}, deepEnough)

	want := []string{"README.md", "pay/README.md"}
	if got := pathsOf(pages); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("pages = %v, want %v", got, want)
	}
	root := pageAt(t, pages, "README.md")
	for _, expected := range []string{
		"# Coverage\n", "Source folder: `libs/app/src`",
		"**Code coverage 66.67%** · Total 9 lines · Covered 6 · Partial 0 · Not covered 3",
		"| `cart.go` | █████░░░░░ 50.00% | 4 | 2 | 0 | 2 |",
		"| [pay/](./pay/README.md) | ████████░░ 80.00% | 5 | 4 | 0 | 1 |",
	} {
		if !strings.Contains(root, expected) {
			t.Errorf("root page lacks %q:\n%s", expected, root)
		}
	}
	if strings.Index(root, "`cart.go`") > strings.Index(root, "[pay/]") {
		t.Error("files should come before folders, as in Azure DevOps")
	}
	child := pageAt(t, pages, "pay/README.md")
	for _, expected := range []string{"# Coverage: libs/app/src/pay\n", "[Up](../README.md)", "| `card.go` | ██████████ 100.00% | 4 | 4 | 0 | 0 |"} {
		if !strings.Contains(child, expected) {
			t.Errorf("folder page lacks %q:\n%s", expected, child)
		}
	}
}

func TestLineStatsCountsALineThatRanWithAMissedBranchAsPartial(t *testing.T) {
	file := fileWith("a.go", 3, 0, 1)
	file.recordBranch(1, Branch{Covered: 1, Total: 2})
	file.recordBranch(3, Branch{Covered: 2, Total: 2})

	stats := file.LineStats()
	if stats != (LineStats{Total: 3, Covered: 1, Partial: 1, NotCovered: 1}) {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestNumberGroupsThousands(t *testing.T) {
	for in, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 12145: "12,145", 1234567: "1,234,567"} {
		if got := number(in); got != want {
			t.Errorf("number(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderCoverageWithNoFilesSaysSoUnderTheTitle(t *testing.T) {
	pages := RenderCoverage("Quality", nil, DefaultPageDepth)
	if len(pages) != 1 || !strings.Contains(string(pages[0].Body), "# Quality") || !strings.Contains(string(pages[0].Body), "No coverage reports") {
		t.Fatalf("pages = %+v", pages)
	}
}

func TestRenderCoverageIsDeterministic(t *testing.T) {
	files := []FileCoverage{fileWith("b/x.go", 1, 0), fileWith("a/y.go", 1), fileWith("a/z.go", 0)}
	first := RenderCoverage("T", files, DefaultPageDepth)
	reversed := []FileCoverage{files[2], files[1], files[0]}
	second := RenderCoverage("T", reversed, DefaultPageDepth)
	if len(first) != len(second) {
		t.Fatalf("page counts differ")
	}
	for i := range first {
		if first[i].Path != second[i].Path || string(first[i].Body) != string(second[i].Body) {
			t.Fatalf("page %s differs between orderings", first[i].Path)
		}
	}
}
