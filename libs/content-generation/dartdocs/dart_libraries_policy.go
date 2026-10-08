package dartdocs

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var packageName = regexp.MustCompile(`(?m)^name:\s*['"]?([A-Za-z0-9_]+)['"]?\s*$`)

// readPackageName is the name in the project's pubspec.yaml, or the folder's name when it has
// none a regular expression can find.
func readPackageName(projectDir string) string {
	if content, err := os.ReadFile(filepath.Join(projectDir, "pubspec.yaml")); err == nil {
		if match := packageName.FindSubmatch(content); match != nil {
			return string(match[1])
		}
	}

	return filepath.Base(projectDir)
}

// libraryFiles lists the Dart files under the project's lib/ folder, project-relative and
// '/'-separated, sorted: that is what a package publishes. Generated code (*.g.dart,
// *.freezed.dart, *.mocks.dart and the like), part files of generated code and anything in a
// hidden or "generated" folder are left out, as are files whose name starts with an underscore.
func libraryFiles(projectDir string) []string {
	var files []string
	lib := filepath.Join(projectDir, "lib")
	_ = filepath.WalkDir(lib, func(full string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := entry.Name()
		if entry.IsDir() {
			if full != lib && (strings.HasPrefix(name, ".") || strings.EqualFold(name, "generated")) {
				return filepath.SkipDir
			}

			return nil
		}
		if isDocumentedFile(name) {
			relative, relErr := filepath.Rel(projectDir, full)
			if relErr == nil {
				files = append(files, filepath.ToSlash(relative))
			}
		}

		return nil
	})
	sort.Strings(files)

	return files
}

var generatedSuffixes = []string{".g.dart", ".freezed.dart", ".mocks.dart", ".gr.dart", ".config.dart", ".chopper.dart"}

func isDocumentedFile(name string) bool {
	if !strings.HasSuffix(name, ".dart") || strings.HasPrefix(name, "_") {
		return false
	}
	for _, suffix := range generatedSuffixes {
		if strings.HasSuffix(name, suffix) {
			return false
		}
	}

	return true
}

// libraryURI is how a file is imported: package:shop/src/cart.dart for lib/src/cart.dart.
func libraryURI(packageName string, file string) string {
	return "package:" + packageName + "/" + strings.TrimPrefix(file, "lib/")
}

// pagePath places a library's page beside its source: lib/src/cart.dart is src/cart.md.
func pagePath(prefix string, file string) string {
	return path.Join(prefix, strings.TrimSuffix(strings.TrimPrefix(file, "lib/"), ".dart")+".md")
}
