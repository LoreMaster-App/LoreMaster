package slicecheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// roleFile matches a role-suffixed Go file: <snake>_<role>.go or
// <snake>_<role>_test.go, with the role list shared with the TypeScript rule.
var roleFile = regexp.MustCompile(
	`^[a-z0-9]+(?:_[a-z0-9]+)*_(?:handler|use_case|algorithm|policy|model|contract|mapper|validator|repository|client|store|error|config|enum)(?:_test)?\.go$`,
)

// slicePackage matches a valid slice directory: lowercase letters and digits.
var slicePackage = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// packageClause captures the package name of a Go source file.
var packageClause = regexp.MustCompile(`(?m)^package\s+([a-z0-9_]+)\s*$`)

// forbiddenPackages are names that say nothing about an outcome.
var forbiddenPackages = map[string]bool{
	"util": true, "utils": true, "helper": true, "helpers": true,
	"common": true, "shared": true, "internal": true, "lib": true,
}

// rootFiles lists the only Go files allowed at a project root, per kind.
var rootFiles = map[string]map[string]bool{
	"libs": {"doc.go": true},
	"apps": {"main.go": true, "main_test.go": true, "doc.go": true},
}

// Violations walks every Go project under <root>/apps and <root>/libs and
// returns one human-readable finding per broken rule, sorted. A project is a
// Go project when its root directory holds at least one .go file or a
// sub-directory that does.
func Violations(root string) ([]string, error) {
	var findings []string
	for kind, allowed := range rootFiles {
		projects, err := os.ReadDir(filepath.Join(root, kind))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			if !project.IsDir() {
				continue
			}
			projectFindings, err := checkProject(filepath.Join(root, kind, project.Name()), kind+"/"+project.Name(), allowed)
			if err != nil {
				return nil, err
			}
			findings = append(findings, projectFindings...)
		}
	}
	sort.Strings(findings)

	return findings, nil
}

func checkProject(dir, label string, allowedRootFiles map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var findings []string
	for _, entry := range entries {
		name := entry.Name()
		switch {
		case !entry.IsDir() && strings.HasSuffix(name, ".go") && !allowedRootFiles[name]:
			findings = append(findings, fmt.Sprintf("%s/%s: only %s may sit at a project root; move it into a slice package", label, name, keys(allowedRootFiles)))
		case entry.IsDir() && name != "testdata" && hasGoFiles(filepath.Join(dir, name)):
			sliceFindings, err := checkSlice(filepath.Join(dir, name), label+"/"+name, name)
			if err != nil {
				return nil, err
			}
			findings = append(findings, sliceFindings...)
		}
	}

	return findings, nil
}

func checkSlice(dir, label, name string) ([]string, error) {
	var findings []string
	if !slicePackage.MatchString(name) {
		findings = append(findings, fmt.Sprintf("%s: slice package names are lowercase letters and digits only", label))
	}
	if forbiddenPackages[name] {
		findings = append(findings, fmt.Sprintf("%s: %q says nothing about an outcome; name the slice after what it delivers", label, name))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		file := entry.Name()
		if entry.IsDir() {
			if file != "testdata" {
				findings = append(findings, fmt.Sprintf("%s/%s: a slice is flat; only testdata/ may sit inside it", label, file))
			}

			continue
		}
		if !strings.HasSuffix(file, ".go") {
			continue
		}
		if file != "doc.go" && !roleFile.MatchString(file) {
			findings = append(findings, fmt.Sprintf("%s/%s: name it <snake>_<role>.go with one role from the ADR", label, file))
		}
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			return nil, err
		}
		if match := packageClause.FindSubmatch(source); match == nil || string(match[1]) != name {
			findings = append(findings, fmt.Sprintf("%s/%s: package clause must equal the directory name %q", label, file, name))
		}
	}

	return findings, nil
}

func hasGoFiles(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(path, ".go") {
			found = true

			return filepath.SkipAll
		}

		return nil
	})

	return found
}

func keys(set map[string]bool) string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)

	return strings.Join(names, ", ")
}
