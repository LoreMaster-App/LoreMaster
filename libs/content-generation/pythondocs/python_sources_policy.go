package pythondocs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// sources is what to document in one Python project.
type sources struct {
	// SearchPath is the folder, relative to the project, the modules are imported from.
	SearchPath string
	// Packages and Modules are top-level names under SearchPath.
	Packages []string
	Modules  []string
}

// empty reports whether there is nothing to document.
func (s sources) empty() bool { return len(s.Packages) == 0 && len(s.Modules) == 0 }

var skippedFolders = map[string]bool{
	"tests": true, "test": true, "testing": true, "docs": true, "doc": true, "examples": true,
	"build": true, "dist": true, "node_modules": true, "venv": true, "env": true, "site-packages": true,
}

// chooseSources decides what to document in a project folder: the top-level packages (folders
// with an __init__.py) and modules, from "src" when that holds any, else from the project folder.
// Tests, docs, examples, virtual environments, build output and setup scripts are not API.
func chooseSources(projectDir string) sources {
	for _, searchPath := range []string{"src", "."} {
		found := sources{SearchPath: searchPath}
		entries, err := os.ReadDir(filepath.Join(projectDir, searchPath))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			switch {
			case entry.IsDir():
				if isDocumentedFolder(name) && fileExists(filepath.Join(projectDir, searchPath, name, "__init__.py")) {
					found.Packages = append(found.Packages, name)
				}
			case isDocumentedModule(name):
				found.Modules = append(found.Modules, strings.TrimSuffix(name, ".py"))
			}
		}
		if !found.empty() {
			sort.Strings(found.Packages)
			sort.Strings(found.Modules)

			return found
		}
	}

	return sources{}
}

func isDocumentedFolder(name string) bool {
	return !strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "_") && !strings.HasSuffix(name, ".egg-info") && !skippedFolders[strings.ToLower(name)]
}

func isDocumentedModule(name string) bool {
	if !strings.HasSuffix(name, ".py") || strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
		return false
	}
	switch base := strings.TrimSuffix(name, ".py"); {
	case base == "setup" || base == "conftest" || base == "noxfile" || base == "fabfile":
		return false
	case strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test"):
		return false
	}

	return true
}

func fileExists(full string) bool {
	info, err := os.Stat(full)

	return err == nil && !info.IsDir()
}
