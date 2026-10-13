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

func TestRenderCoverageMakesOnePagePerFolderStartingWhereTheCodeDoes(t *testing.T) {
	pages := RenderCoverage("", []FileCoverage{
		fileWith("libs/app/src/cart.go", 1, 1, 0, 0),
		fileWith("libs/app/src/pay/card.go", 1, 1, 1, 1),
		fileWith("libs/app/src/pay/refund.go", 0),
	})

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
	pages := RenderCoverage("Quality", nil)
	if len(pages) != 1 || !strings.Contains(string(pages[0].Body), "# Quality") || !strings.Contains(string(pages[0].Body), "No coverage reports") {
		t.Fatalf("pages = %+v", pages)
	}
}

func TestRenderCoverageIsDeterministic(t *testing.T) {
	files := []FileCoverage{fileWith("b/x.go", 1, 0), fileWith("a/y.go", 1), fileWith("a/z.go", 0)}
	first := RenderCoverage("T", files)
	reversed := []FileCoverage{files[2], files[1], files[0]}
	second := RenderCoverage("T", reversed)
	if len(first) != len(second) {
		t.Fatalf("page counts differ")
	}
	for i := range first {
		if first[i].Path != second[i].Path || string(first[i].Body) != string(second[i].Body) {
			t.Fatalf("page %s differs between orderings", first[i].Path)
		}
	}
}
