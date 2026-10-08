package godocs

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/content-generation/inputselection"
)

// skippedFolders are never searched: dependencies, version control, vendored code, test data,
// and anything hidden or underscore-prefixed, which the Go tool also ignores.
var skippedFolders = []string{"node_modules", "vendor", "testdata", "dist", "out-tsc"}

// sourceFolder is one folder of Go source: where it is and which files are in it.
type sourceFolder struct {
	// Dir is workspace-relative and '/'-separated; "." is the workspace root.
	Dir string
	// Files are the non-test .go files' names, sorted.
	Files []string
	// ImportPath is the module path joined with Dir, or Dir when there is no go.mod.
	ImportPath string
}

// findFolders lists the folders holding non-test Go source, sorted. Patterns use gitignore
// syntax against folder paths, so "libs/" selects a tree and "!libs/internal/" leaves one out;
// with none selecting, every folder is.
func findFolders(ctx context.Context, workspaceRoot string, patterns []string) ([]sourceFolder, error) {
	selector := inputselection.NewSelector(patterns, nil)
	modules := moduleResolver{root: workspaceRoot, cache: map[string]moduleInfo{}}

	var folders []sourceFolder
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
		name := entry.Name()
		if full != workspaceRoot && (slices.Contains(skippedFolders, name) || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
			return filepath.SkipDir
		}
		relative, err := filepath.Rel(workspaceRoot, full)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !selector.Selects(relative, true) {
			return nil
		}
		files := goFiles(full)
		if len(files) == 0 {
			return nil
		}
		folders = append(folders, sourceFolder{Dir: relative, Files: files, ImportPath: modules.importPath(relative)})

		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(folders, func(a, b sourceFolder) int { return strings.Compare(a.Dir, b.Dir) })

	return folders, nil
}

// goFiles names the non-test Go files in a folder, sorted.
func goFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		files = append(files, name)
	}
	slices.Sort(files)

	return files
}

// moduleInfo is the nearest go.mod above a folder.
type moduleInfo struct {
	// Path is the module path; Dir is the module's folder, workspace-relative.
	Path string
	Dir  string
}

// moduleResolver finds the module a folder belongs to, remembering the answers.
type moduleResolver struct {
	root  string
	cache map[string]moduleInfo
}

// importPath is the package's import path: its module path plus its folder below the module.
func (r moduleResolver) importPath(dir string) string {
	module := r.moduleOf(dir)
	if module.Path == "" {
		return dir
	}
	below := dir
	if module.Dir != "." {
		below = strings.TrimPrefix(strings.TrimPrefix(dir, module.Dir), "/")
		if dir == module.Dir {
			below = "."
		}
	}
	if below == "." || below == "" {
		return module.Path
	}

	return module.Path + "/" + below
}

func (r moduleResolver) moduleOf(dir string) moduleInfo {
	if info, known := r.cache[dir]; known {
		return info
	}
	info := moduleInfo{}
	if path, found := readModulePath(filepath.Join(r.root, filepath.FromSlash(dir), "go.mod")); found {
		info = moduleInfo{Path: path, Dir: dir}
	} else if dir != "." {
		info = r.moduleOf(parentDir(dir))
	}
	r.cache[dir] = info

	return info
}

func parentDir(dir string) string {
	parent := path.Dir(dir)
	if parent == "" {
		return "."
	}

	return parent
}

// readModulePath reads the "module" line of a go.mod.
func readModulePath(goMod string) (string, bool) {
	file, err := os.Open(goMod)
	if err != nil {
		return "", false
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if rest, found := strings.CutPrefix(line, "module"); found && (rest == "" || rest[0] == ' ' || rest[0] == '\t') {
			return strings.Trim(strings.TrimSpace(rest), `"`), strings.TrimSpace(rest) != ""
		}
	}

	return "", false
}
