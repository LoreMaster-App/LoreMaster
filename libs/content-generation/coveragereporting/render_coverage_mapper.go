package coveragereporting

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
)

// DefaultTitle heads the index page when the generator is not given one.
const DefaultTitle = "Coverage"

// folder is a directory of the covered files and what is under it.
type folder struct {
	path    string
	folders map[string]*folder
	files   []FileCoverage
	stats   LineStats
}

// RenderCoverage lays the files out as one page per folder, the way Azure DevOps shows its
// Code Coverage tab: an overview of the folder, then its files and subfolders with the coverage
// bar and percentage, total, covered, partially covered and not covered lines. Each subfolder
// links to its own page and each page links back up, so the tree can be navigated. With no
// files it still returns the index saying so, so stale results never linger. The result is
// deterministic.
func RenderCoverage(title string, files []FileCoverage) []generatedfile.File {
	if title == "" {
		title = DefaultTitle
	}
	if len(files) == 0 {
		return []generatedfile.File{{Path: "README.md", Body: []byte("# " + title + "\n\nNo coverage reports were found.\n")}}
	}

	base := commonFolder(files)
	root := &folder{path: base, folders: map[string]*folder{}}
	for _, file := range files {
		relative := strings.TrimPrefix(strings.TrimPrefix(file.Path, base), "/")
		root.insert(strings.Split(relative, "/"), file)
	}
	root.total()

	var out []generatedfile.File
	root.render(title, "", &out)
	slices.SortFunc(out, func(a, b generatedfile.File) int { return strings.Compare(a.Path, b.Path) })

	return out
}

// commonFolder is the deepest folder every file is under, so the root page starts where the
// code does instead of at empty levels.
func commonFolder(files []FileCoverage) string {
	parts := strings.Split(path.Dir(files[0].Path), "/")
	if len(parts) == 1 && parts[0] == "." {
		return ""
	}
	for _, file := range files[1:] {
		other := strings.Split(path.Dir(file.Path), "/")
		n := 0
		for n < len(parts) && n < len(other) && parts[n] == other[n] {
			n++
		}
		parts = parts[:n]
	}

	return strings.Join(parts, "/")
}

func (f *folder) insert(segments []string, file FileCoverage) {
	if len(segments) == 1 {
		f.files = append(f.files, file)

		return
	}
	name := segments[0]
	child := f.folders[name]
	if child == nil {
		child = &folder{path: path.Join(f.path, name), folders: map[string]*folder{}}
		f.folders[name] = child
	}
	child.insert(segments[1:], file)
}

// total fills the folder's counts from everything under it.
func (f *folder) total() {
	for _, file := range f.files {
		f.stats = f.stats.Add(file.LineStats())
	}
	for _, child := range f.folders {
		child.total()
		f.stats = f.stats.Add(child.stats)
	}
}

// render writes this folder's page at dir (relative to the output folder) and its children's.
func (f *folder) render(rootTitle, dir string, out *[]generatedfile.File) {
	var sb strings.Builder
	heading := rootTitle
	if dir != "" {
		heading = rootTitle + ": " + f.path
	}
	fmt.Fprintf(&sb, "# %s\n\n", heading)
	if dir != "" {
		sb.WriteString("[Up](../README.md)\n\n")
	} else if f.path != "" {
		fmt.Fprintf(&sb, "Source folder: `%s`\n\n", f.path)
	}
	fmt.Fprintf(&sb, "## Overview\n\n**Code coverage %s** · Total %s lines · Covered %s · Partial %s · Not covered %s\n\n",
		percent(f.stats), number(f.stats.Total), number(f.stats.Covered), number(f.stats.Partial), number(f.stats.NotCovered))

	names := make([]string, 0, len(f.folders))
	for name := range f.folders {
		names = append(names, name)
	}
	slices.Sort(names)
	slices.SortFunc(f.files, func(a, b FileCoverage) int { return strings.Compare(a.Path, b.Path) })

	sb.WriteString("| Name | Coverage | Total lines | Covered | Partially covered | Not covered |\n|---|---|---|---|---|---|\n")
	for _, file := range f.files {
		row(&sb, "`"+path.Base(file.Path)+"`", file.LineStats())
	}
	for _, name := range names {
		child := f.folders[name]
		row(&sb, fmt.Sprintf("[%s/](./%s/README.md)", name, name), child.stats)
	}
	sb.WriteString("\n")
	*out = append(*out, generatedfile.File{Path: path.Join(dir, "README.md"), Body: []byte(sb.String())})

	for _, name := range names {
		f.folders[name].render(rootTitle, path.Join(dir, name), out)
	}
}

func row(sb *strings.Builder, name string, s LineStats) {
	fmt.Fprintf(sb, "| %s | %s %s | %s | %s | %s | %s |\n", name, bar(s), percent(s),
		number(s.Total), number(s.Covered), number(s.Partial), number(s.NotCovered))
}

// percent is the covered share of the lines to two decimals; 0.00% when there are none, as
// Azure DevOps shows it.
func percent(s LineStats) string {
	if s.Total == 0 {
		return "0.00%"
	}
	hundredths := (s.Covered*10000 + s.Total/2) / s.Total

	return fmt.Sprintf("%d.%02d%%", hundredths/100, hundredths%100)
}

// bar is ten blocks, filled in proportion to the covered share.
func bar(s LineStats) string {
	filled := 0
	if s.Total > 0 {
		filled = (s.Covered*10 + s.Total/2) / s.Total
	}

	return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
}

// number writes an integer with thousands separators: 11,384.
func number(n int) string {
	digits := strconv.Itoa(n)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}

	return digits
}
