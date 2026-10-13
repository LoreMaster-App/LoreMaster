package coveragereporting

import (
	"bufio"
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// parseLCOV reads LCOV tracefiles (lcov.info): SF opens a file, DA gives a line's hits, BRDA a
// branch of a line, end_of_record closes the file.
func parseLCOV(data []byte, workspaceRoot string) ([]FileCoverage, error) {
	files := map[string]*FileCoverage{}
	var current *FileCoverage
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "SF:"):
			path := resolveReportPath(workspaceRoot, nil, strings.TrimPrefix(line, "SF:"))
			current = files[path]
			if current == nil {
				current = newFileCoverage(path)
				files[path] = current
			}
		case current == nil:
			continue
		case strings.HasPrefix(line, "DA:"):
			fields := strings.Split(strings.TrimPrefix(line, "DA:"), ",")
			if len(fields) < 2 {
				continue
			}
			number, errNumber := strconv.Atoi(fields[0])
			hits, errHits := strconv.Atoi(fields[1])
			if errNumber == nil && errHits == nil {
				current.recordLine(number, hits)
			}
		case strings.HasPrefix(line, "BRDA:"):
			fields := strings.Split(strings.TrimPrefix(line, "BRDA:"), ",")
			if len(fields) < 4 {
				continue
			}
			number, err := strconv.Atoi(fields[0])
			if err != nil {
				continue
			}
			branch := current.Branches[number]
			branch.Total++
			if taken, err := strconv.Atoi(fields[3]); err == nil && taken > 0 {
				branch.Covered++
			}
			current.Branches[number] = branch
		case line == "end_of_record":
			current = nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("not an LCOV report: no SF records")
	}

	out := make([]FileCoverage, 0, len(files))
	for _, file := range files {
		out = append(out, *file)
	}
	slices.SortFunc(out, func(a, b FileCoverage) int { return strings.Compare(a.Path, b.Path) })

	return out, nil
}
