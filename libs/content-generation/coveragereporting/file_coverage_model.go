package coveragereporting

// Branch counts the branches of one line.
type Branch struct {
	Covered int
	Total   int
}

// FileCoverage is what a report says about one source file. Keeping the result per line,
// rather than as totals, is what lets two reports of the same file be merged correctly.
type FileCoverage struct {
	// Path is the file's path, '/'-separated: relative to the workspace when the report named
	// it inside the workspace, else as the report wrote it.
	Path string
	// LineHits maps an executable line number to how many times it ran.
	LineHits map[int]int
	// Branches maps a line number to its branch counts.
	Branches map[int]Branch
}

// Lines returns how many executable lines ran and how many there are.
func (f FileCoverage) Lines() (covered, total int) {
	for _, hits := range f.LineHits {
		total++
		if hits > 0 {
			covered++
		}
	}

	return covered, total
}

// LineStats splits the executable lines the way Azure DevOps does: a line that ran with every
// branch taken is covered, one that ran with a branch missed is partially covered, one that
// never ran is not covered. The three always add up to the total.
func (f FileCoverage) LineStats() LineStats {
	var stats LineStats
	for line, hits := range f.LineHits {
		stats.Total++
		branch := f.Branches[line]
		switch {
		case hits <= 0:
			stats.NotCovered++
		case branch.Total > 0 && branch.Covered < branch.Total:
			stats.Partial++
		default:
			stats.Covered++
		}
	}

	return stats
}

// LineStats counts a file's executable lines by how well they ran.
type LineStats struct {
	Total      int
	Covered    int
	Partial    int
	NotCovered int
}

// Add returns the sum of two sets of counts.
func (s LineStats) Add(other LineStats) LineStats {
	return LineStats{
		Total: s.Total + other.Total, Covered: s.Covered + other.Covered,
		Partial: s.Partial + other.Partial, NotCovered: s.NotCovered + other.NotCovered,
	}
}

// BranchTotals returns how many branches were taken and how many there are.
func (f FileCoverage) BranchTotals() (covered, total int) {
	for _, branch := range f.Branches {
		covered += branch.Covered
		total += branch.Total
	}

	return covered, total
}

func newFileCoverage(path string) *FileCoverage {
	return &FileCoverage{Path: path, LineHits: map[int]int{}, Branches: map[int]Branch{}}
}

// recordLine keeps the highest hit count seen for a line.
func (f *FileCoverage) recordLine(line, hits int) {
	if current, seen := f.LineHits[line]; !seen || hits > current {
		f.LineHits[line] = hits
	}
}

// recordBranch keeps the most complete branch counts seen for a line.
func (f *FileCoverage) recordBranch(line int, branch Branch) {
	current := f.Branches[line]
	f.Branches[line] = Branch{Covered: max(current.Covered, branch.Covered), Total: max(current.Total, branch.Total)}
}
