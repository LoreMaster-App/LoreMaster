package coveragereporting

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

// ParseCoverage reads one coverage report, detecting its format from the content: XML is
// Cobertura, a tracefile with SF records is LCOV. workspaceRoot lets absolute paths in the
// report become workspace-relative.
func ParseCoverage(data []byte, workspaceRoot string) ([]FileCoverage, error) {
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")))
	switch {
	case len(trimmed) == 0:
		return nil, fmt.Errorf("the file is empty")
	case trimmed[0] == '<':
		return parseCobertura(trimmed, workspaceRoot)
	case bytes.Contains(trimmed, []byte("SF:")):
		return parseLCOV(trimmed, workspaceRoot)
	}

	return nil, fmt.Errorf("not a recognised coverage report (Cobertura XML and LCOV are supported)")
}

// MergeCoverage combines reports of the same file line by line, so a file covered by two test
// runs counts a line as covered when either ran it.
func MergeCoverage(reports [][]FileCoverage) []FileCoverage {
	merged := map[string]*FileCoverage{}
	for _, report := range reports {
		for _, file := range report {
			target := merged[file.Path]
			if target == nil {
				target = newFileCoverage(file.Path)
				merged[file.Path] = target
			}
			for line, hits := range file.LineHits {
				target.recordLine(line, hits)
			}
			for line, branch := range file.Branches {
				target.recordBranch(line, branch)
			}
		}
	}

	out := make([]FileCoverage, 0, len(merged))
	for _, file := range merged {
		out = append(out, *file)
	}
	slices.SortFunc(out, func(a, b FileCoverage) int { return strings.Compare(a.Path, b.Path) })

	return out
}
