package godocs

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// goPackage is one folder's package, parsed and ready to render.
type goPackage struct {
	folder sourceFolder
	fset   *token.FileSet
	docs   *doc.Package
}

// loadPackage parses a folder's non-test Go files — every file, whatever its build
// constraints, except those marked "ignore" — and reads the package's documentation. A file
// that does not parse is skipped with a warning. The bool is false for a folder with nothing
// to document: no exported declarations and no package comment.
func loadPackage(workspaceRoot string, folder sourceFolder) (goPackage, []string, bool) {
	var warnings []string
	fset := token.NewFileSet()
	byPackage := map[string][]*ast.File{}
	for _, name := range folder.Files {
		display := folder.Dir + "/" + name
		full := filepath.Join(workspaceRoot, filepath.FromSlash(folder.Dir), name)
		source, err := os.ReadFile(full)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", display, err))

			continue
		}
		file, err := parser.ParseFile(fset, display, source, parser.ParseComments)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", display, err))

			continue
		}
		if isIgnoredByConstraint(file) {
			continue
		}
		byPackage[file.Name.Name] = append(byPackage[file.Name.Name], file)
	}
	if len(byPackage) == 0 {
		return goPackage{}, warnings, false
	}

	name := chosenPackage(byPackage)
	for other := range byPackage {
		if other != name {
			warnings = append(warnings, fmt.Sprintf("%s: files declare package %s as well as %s; only %s is documented", folder.Dir, other, name, name))
		}
	}
	docs, err := doc.NewFromFiles(fset, byPackage[name], folder.ImportPath)
	if err != nil {
		return goPackage{}, append(warnings, fmt.Sprintf("%s: %v", folder.Dir, err)), false
	}
	if strings.TrimSpace(docs.Doc) == "" && len(docs.Consts)+len(docs.Vars)+len(docs.Funcs)+len(docs.Types) == 0 {
		return goPackage{}, warnings, false
	}

	return goPackage{folder: folder, fset: fset, docs: docs}, warnings, true
}

// chosenPackage is the package name with the most files; a tie goes to the name that sorts
// first, so the choice never depends on map order.
func chosenPackage(byPackage map[string][]*ast.File) string {
	names := make([]string, 0, len(byPackage))
	for name := range byPackage {
		names = append(names, name)
	}
	slices.Sort(names)
	chosen := names[0]
	for _, name := range names[1:] {
		if len(byPackage[name]) > len(byPackage[chosen]) {
			chosen = name
		}
	}

	return chosen
}

// isIgnoredByConstraint reports whether the file opts out of every build with "//go:build
// ignore", the convention for scratch programs that share a folder with a package.
func isIgnoredByConstraint(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}
		for _, comment := range group.List {
			if strings.TrimSpace(comment.Text) == "//go:build ignore" || strings.TrimSpace(comment.Text) == "// +build ignore" {
				return true
			}
		}
	}

	return false
}
