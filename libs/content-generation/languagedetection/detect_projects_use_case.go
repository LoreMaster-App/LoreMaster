package languagedetection

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// skippedFolders are never searched: dependencies, build output and tool state, which hold
// other people's projects or generated copies of ours.
var skippedFolders = []string{
	"node_modules", ".git", "vendor", "dist", "out-tsc", "bin", "obj", "build", "coverage",
	".venv", "venv", "__pycache__", ".dart_tool", ".nx", "testdata",
}

// markers lists, per language, the files that mark a project; the first that exists names it.
var markers = []struct {
	language Language
	files    []string
	// suffix marks a project by any file ending so (a .csproj has no fixed name).
	suffix string
}{
	{language: TypeScript, files: []string{"tsconfig.json", "jsconfig.json"}},
	{language: Python, files: []string{"pyproject.toml", "setup.py", "setup.cfg"}},
	{language: Go, files: []string{"go.mod"}},
	{language: Dart, files: []string{"pubspec.yaml"}},
	{language: CSharp, suffix: ".csproj"},
}

// Detect walks the workspace and returns its projects sorted by folder, then language. A
// folder that marks several languages (a TypeScript library with a Python script) is several
// projects; a folder is one project per language however many marker files it has.
func Detect(ctx context.Context, workspaceRoot string) ([]Project, error) {
	var projects []Project
	err := filepath.WalkDir(workspaceRoot, func(full string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if full != workspaceRoot && (slices.Contains(skippedFolders, entry.Name()) || strings.HasPrefix(entry.Name(), ".")) {
			return filepath.SkipDir
		}
		relative, err := filepath.Rel(workspaceRoot, full)
		if err != nil {
			return err
		}
		projects = append(projects, projectsIn(full, filepath.ToSlash(relative))...)

		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(projects, func(a, b Project) int {
		if byDir := strings.Compare(a.Dir, b.Dir); byDir != 0 {
			return byDir
		}

		return strings.Compare(string(a.Language), string(b.Language))
	})

	return projects, nil
}

// projectsIn names the projects whose marker files are in one folder.
func projectsIn(full string, relative string) []Project {
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil
	}
	present := map[string]bool{}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			present[entry.Name()] = true
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)

	var found []Project
	for _, marker := range markers {
		if file := firstMarker(marker.files, marker.suffix, present, names); file != "" {
			found = append(found, Project{Language: marker.language, Dir: relative, MarkerFile: file})
		}
	}

	return found
}

func firstMarker(files []string, suffix string, present map[string]bool, names []string) string {
	for _, file := range files {
		if present[file] {
			return file
		}
	}
	if suffix != "" {
		for _, name := range names {
			if strings.HasSuffix(name, suffix) {
				return name
			}
		}
	}

	return ""
}
