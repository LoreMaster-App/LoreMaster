package changedetection

import (
	"path"
	"strings"
)

// skippedFolders are never looked into: dependencies, build output and tool caches.
var skippedFolders = map[string]bool{
	"node_modules": true, "vendor": true, "bin": true, "obj": true, "dist": true, "build": true,
	"out": true, "target": true, "venv": true, "env": true, "__pycache__": true, "coverage": true,
	"packages-cache": true, "testdata": true,
}

// watchedExtensions are the files that can matter: Markdown, and what the generators read
// (JUnit reports, Go, OpenAPI descriptions, TypeScript and JavaScript, Python, C#, Dart and their
// project files).
var watchedExtensions = map[string]bool{
	".md": true, ".xml": true, ".go": true, ".mod": true, ".yaml": true, ".yml": true, ".json": true,
	".ts": true, ".tsx": true, ".mts": true, ".cts": true, ".js": true, ".jsx": true,
	".py": true, ".toml": true, ".cfg": true, ".cs": true, ".csproj": true, ".props": true, ".dart": true,
}

// IsSkippedFolder reports whether a folder is left out of the look: hidden folders, and the usual
// dependency and build-output folders.
func IsSkippedFolder(name string) bool {
	return strings.HasPrefix(name, ".") || skippedFolders[strings.ToLower(name)]
}

// IsWatchedFile reports whether a file's changes matter. rel is workspace-relative and
// '/'-separated.
func IsWatchedFile(rel string) bool {
	name := path.Base(rel)
	if name == ".lore-master.yaml" {
		return true
	}

	return !strings.HasPrefix(name, ".") && watchedExtensions[strings.ToLower(path.Ext(name))]
}
