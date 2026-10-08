package languagedetection

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func workspace(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, file := range files {
		full := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func found(t *testing.T, root string) []Project {
	t.Helper()
	projects, err := Detect(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	return projects
}

func TestDetectFindsEveryEcosystemByItsMarkerFile(t *testing.T) {
	root := workspace(t,
		"apps/web/tsconfig.json", "apps/js/jsconfig.json",
		"libs/py/pyproject.toml", "tools/script/setup.py",
		"services/api/go.mod",
		"mobile/pubspec.yaml",
		"dotnet/App/App.csproj",
	)

	got := found(t, root)

	want := []Project{
		{Language: TypeScript, Dir: "apps/js", MarkerFile: "jsconfig.json"},
		{Language: TypeScript, Dir: "apps/web", MarkerFile: "tsconfig.json"},
		{Language: CSharp, Dir: "dotnet/App", MarkerFile: "App.csproj"},
		{Language: Python, Dir: "libs/py", MarkerFile: "pyproject.toml"},
		{Language: Dart, Dir: "mobile", MarkerFile: "pubspec.yaml"},
		{Language: Go, Dir: "services/api", MarkerFile: "go.mod"},
		{Language: Python, Dir: "tools/script", MarkerFile: "setup.py"},
	}
	if len(got) != len(want) {
		t.Fatalf("projects %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("project %d: %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestDetectCountsAFolderOncePerLanguageAndKeepsSeveralLanguages(t *testing.T) {
	root := workspace(t, "pkg/tsconfig.json", "pkg/jsconfig.json", "pkg/pyproject.toml", "pkg/setup.py", "pkg/a.csproj", "pkg/b.csproj")

	got := found(t, root)

	languages := map[Language]string{}
	for _, project := range got {
		if _, repeated := languages[project.Language]; repeated {
			t.Fatalf("%s found twice: %+v", project.Language, got)
		}
		languages[project.Language] = project.MarkerFile
	}
	if len(got) != 3 || languages[TypeScript] != "tsconfig.json" || languages[Python] != "pyproject.toml" || languages[CSharp] != "a.csproj" {
		t.Fatalf("projects %+v", got)
	}
}

func TestDetectFindsTheWorkspaceRootAsAProject(t *testing.T) {
	got := found(t, workspace(t, "go.mod", "tsconfig.json"))

	if len(got) != 2 || got[0].Dir != "." || got[1].Dir != "." {
		t.Fatalf("projects %+v", got)
	}
}

func TestDetectNeverEntersDependenciesBuildOutputOrHiddenFolders(t *testing.T) {
	root := workspace(t,
		"node_modules/dep/tsconfig.json", "vendor/v/go.mod", "dist/tsconfig.json", "bin/x.csproj", "obj/y.csproj",
		".venv/pyproject.toml", ".git/pubspec.yaml", ".hidden/go.mod", "testdata/go.mod", "real/go.mod",
	)

	got := found(t, root)

	if len(got) != 1 || got[0].Dir != "real" {
		t.Fatalf("projects %+v", got)
	}
}

func TestDetectIgnoresTheOtherConfigFilesOfAProject(t *testing.T) {
	got := found(t, workspace(t, "lib/tsconfig.app.json", "lib/tsconfig.spec.json", "lib/tsconfig.base.json"))

	if len(got) != 0 {
		t.Fatalf("projects %+v", got)
	}
}

func TestDetectStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Detect(ctx, workspace(t, "a/go.mod")); err == nil {
		t.Fatal("expected the cancellation")
	}
}
