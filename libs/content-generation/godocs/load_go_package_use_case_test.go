package godocs

import (
	"slices"
	"strings"
	"testing"
)

func load(t *testing.T, dir string) (goPackage, []string, bool) {
	t.Helper()
	root := fixtureWorkspace(t)
	folders, err := findFolders(t.Context(), root, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range folders {
		if folder.Dir == dir {
			return loadPackage(root, folder)
		}
	}
	t.Fatalf("no folder %s", dir)

	return goPackage{}, nil, false
}

func functionNames(p goPackage) []string {
	var names []string
	for _, function := range p.docs.Funcs {
		names = append(names, function.Name)
	}

	return names
}

func TestLoadPackageReadsTheExportedDeclarationsAndComments(t *testing.T) {
	loaded, warnings, ok := load(t, "store")

	if !ok || len(warnings) != 0 {
		t.Fatalf("ok %v, warnings %q", ok, warnings)
	}
	if loaded.docs.Name != "store" || !strings.HasPrefix(loaded.docs.Doc, "Package store keeps key/value pairs") {
		t.Fatalf("docs %+v", loaded.docs)
	}
	if len(loaded.docs.Types) != 1 || loaded.docs.Types[0].Name != "Store" || len(loaded.docs.Types[0].Methods) != 2 {
		t.Fatalf("types %+v", loaded.docs.Types)
	}
}

func TestLoadPackageSkipsAFileThatDoesNotParseAndSaysSo(t *testing.T) {
	loaded, warnings, ok := load(t, "broken")

	if !ok || !slices.Equal(functionNames(loaded), []string{"Fine"}) {
		t.Fatalf("ok %v, functions %v", ok, functionNames(loaded))
	}
	if len(warnings) != 1 || !strings.HasPrefix(warnings[0], "broken/bad.go: ") {
		t.Fatalf("warnings %q", warnings)
	}
}

func TestLoadPackageDocumentsTheLargestPackageOfAMixedFolder(t *testing.T) {
	loaded, warnings, ok := load(t, "mixed")

	if !ok || loaded.docs.Name != "mixed" || !slices.Equal(functionNames(loaded), []string{"A", "B"}) {
		t.Fatalf("ok %v, package %s, functions %v", ok, loaded.docs.Name, functionNames(loaded))
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "package other as well as mixed") {
		t.Fatalf("warnings %q", warnings)
	}
}

func TestLoadPackageIgnoresBuildConstraintsExceptIgnore(t *testing.T) {
	platform, _, ok := load(t, "platform")
	if !ok || !slices.Equal(functionNames(platform), []string{"LinuxOnly", "WindowsOnly"}) {
		t.Fatalf("ok %v, functions %v", ok, functionNames(platform))
	}

	if _, _, scratch := load(t, "scratch"); scratch {
		t.Fatal("a //go:build ignore file was documented")
	}
}

func TestLoadPackageLeavesOutAFolderWithNothingToDocument(t *testing.T) {
	if _, warnings, ok := load(t, "empty"); ok || len(warnings) != 0 {
		t.Fatalf("ok %v, warnings %q", ok, warnings)
	}
}

func TestLoadPackageKeepsACommandWithOnlyItsComment(t *testing.T) {
	loaded, _, ok := load(t, "cmd/tool")

	if !ok || loaded.docs.Name != "main" || !strings.HasPrefix(loaded.docs.Doc, "Command tool prints") {
		t.Fatalf("ok %v, docs %+v", ok, loaded.docs)
	}
}
