package coveragereporting

import "testing"

const lcovSample = `TN:
SF:/work/repo/src/a.ts
DA:1,1
DA:2,0
DA:3,5
BRDA:2,0,0,1
BRDA:2,0,1,-
LF:3
LH:2
end_of_record
SF:src/b.ts
DA:1,0
end_of_record
`

func TestParseLCOVReadsLinesAndBranchesPerFile(t *testing.T) {
	files, err := ParseCoverage([]byte(lcovSample), "/work/repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != "src/a.ts" || files[1].Path != "src/b.ts" {
		t.Fatalf("files = %+v", files)
	}
	covered, total := files[0].Lines()
	if covered != 2 || total != 3 {
		t.Fatalf("a.ts lines %d/%d, want 2/3", covered, total)
	}
	branchCovered, branchTotal := files[0].BranchTotals()
	if branchCovered != 1 || branchTotal != 2 {
		t.Fatalf("a.ts branches %d/%d, want 1/2", branchCovered, branchTotal)
	}
}

func TestMergeCoverageCountsALineCoveredWhenAnyRunCoveredIt(t *testing.T) {
	first, _ := ParseCoverage([]byte("SF:src/a.ts\nDA:1,1\nDA:2,0\nend_of_record\n"), "/w")
	second, _ := ParseCoverage([]byte("SF:src/a.ts\nDA:1,0\nDA:2,3\nDA:3,0\nend_of_record\n"), "/w")

	merged := MergeCoverage([][]FileCoverage{first, second})
	if len(merged) != 1 {
		t.Fatalf("merged = %+v", merged)
	}
	covered, total := merged[0].Lines()
	if covered != 2 || total != 3 {
		t.Fatalf("merged lines %d/%d, want 2/3", covered, total)
	}
}
