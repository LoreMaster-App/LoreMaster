package coveragereporting

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type coberturaReport struct {
	XMLName  xml.Name `xml:"coverage"`
	Sources  []string `xml:"sources>source"`
	Packages []struct {
		Classes []struct {
			Filename string `xml:"filename,attr"`
			Lines    []struct {
				Number    int    `xml:"number,attr"`
				Hits      int    `xml:"hits,attr"`
				Branch    bool   `xml:"branch,attr"`
				Condition string `xml:"condition-coverage,attr"`
			} `xml:"lines>line"`
		} `xml:"classes>class"`
	} `xml:"packages>package"`
}

var conditionCounts = regexp.MustCompile(`\((\d+)/(\d+)\)`)

// parseCobertura reads Cobertura XML, the report coverlet, coverage.py and Istanbul write. A
// class's own line list is read, not its methods', which repeat the same lines.
func parseCobertura(data []byte, workspaceRoot string) ([]FileCoverage, error) {
	var report coberturaReport
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&report); err != nil {
		return nil, fmt.Errorf("not a Cobertura report: %w", err)
	}
	if len(report.Packages) == 0 {
		return nil, fmt.Errorf("not a Cobertura report: no packages")
	}

	files := map[string]*FileCoverage{}
	for _, pkg := range report.Packages {
		for _, class := range pkg.Classes {
			if class.Filename == "" {
				continue
			}
			path := resolveReportPath(workspaceRoot, report.Sources, class.Filename)
			file := files[path]
			if file == nil {
				file = newFileCoverage(path)
				files[path] = file
			}
			for _, line := range class.Lines {
				file.recordLine(line.Number, line.Hits)
				if !line.Branch {
					continue
				}
				if match := conditionCounts.FindStringSubmatch(line.Condition); match != nil {
					covered, _ := strconv.Atoi(match[1])
					total, _ := strconv.Atoi(match[2])
					file.recordBranch(line.Number, Branch{Covered: covered, Total: total})
				}
			}
		}
	}

	out := make([]FileCoverage, 0, len(files))
	for _, file := range files {
		out = append(out, *file)
	}
	slices.SortFunc(out, func(a, b FileCoverage) int { return strings.Compare(a.Path, b.Path) })

	return out, nil
}

// resolveReportPath turns a class's filename into a workspace-relative path when it can: a
// filename is relative to one of the report's source folders, and a path inside the workspace
// loses the workspace prefix. Anything else stays as the report wrote it.
func resolveReportPath(workspaceRoot string, sources []string, filename string) string {
	name := slashes(filename)
	if relative, ok := insideWorkspace(workspaceRoot, name); ok {
		return relative
	}
	for _, source := range sources {
		joined := strings.TrimRight(slashes(strings.TrimSpace(source)), "/") + "/" + strings.TrimLeft(name, "/")
		if relative, ok := insideWorkspace(workspaceRoot, joined); ok {
			return relative
		}
	}

	return strings.TrimPrefix(name, "./")
}

func slashes(p string) string {
	return strings.ReplaceAll(p, `\`, "/")
}

// insideWorkspace strips the workspace root from an absolute path, ignoring case as Windows
// and macOS volumes do.
func insideWorkspace(workspaceRoot, p string) (string, bool) {
	root := strings.TrimRight(slashes(workspaceRoot), "/") + "/"
	if len(p) > len(root) && strings.EqualFold(p[:len(root)], root) {
		return p[len(root):], true
	}

	return "", false
}
