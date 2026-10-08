package godocs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func joined(files []generatedfile.File) string {
	var out strings.Builder
	for _, file := range files {
		fmt.Fprintf(&out, "<!-- file: %s -->\n%s\n", file.Path, file.Body)
	}

	return out.String()
}

func bodyOf(t *testing.T, files []generatedfile.File, path string) string {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return string(file.Body)
		}
	}
	t.Fatalf("no page %s in\n%s", path, joined(files))

	return ""
}

func TestGenerateMatchesTheGoldenPages(t *testing.T) {
	output, err := Generate(context.Background(), fixtureWorkspace(t), generatedfile.Spec{Type: "go-docs"})
	if err != nil {
		t.Fatal(err)
	}
	got := joined(output.Files)

	const golden = "testdata/module.golden.md"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if normalised := strings.ReplaceAll(string(want), "\r\n", "\n"); got != normalised {
		t.Fatalf("pages differ from %s (run with UPDATE_GOLDEN=1 to refresh it)\n got:\n%s\nwant:\n%s", golden, got, normalised)
	}
	if len(output.Warnings) != 2 {
		t.Fatalf("warnings %q", output.Warnings)
	}
}

func TestGenerateWritesAnIndexAndOnePagePerPackageNestedLikeTheFolders(t *testing.T) {
	output, err := Generate(context.Background(), fixtureWorkspace(t), generatedfile.Spec{Type: "go-docs", Title: "App API"})
	if err != nil {
		t.Fatal(err)
	}

	var paths []string
	for _, file := range output.Files {
		paths = append(paths, file.Path)
	}
	want := "README.md,app.md,broken/README.md,cmd/tool/README.md,internal/secret/README.md,mixed/README.md,platform/README.md,store/README.md,tools/ext/README.md"
	if strings.Join(paths, ",") != want {
		t.Fatalf("pages %v, want %s", paths, want)
	}
	index := bodyOf(t, output.Files, "README.md")
	if !strings.HasPrefix(index, "# App API\n") || !strings.Contains(index, "8 packages") || strings.Contains(index, "empty") || strings.Contains(index, "scratch") {
		t.Fatalf("index %s", index)
	}
	if !strings.Contains(index, "| [example.com/app/store](store/README.md) | Package store keeps key/value pairs in memory. |") {
		t.Fatalf("index row missing:\n%s", index)
	}
}

func TestGenerateHonoursTheInputPatterns(t *testing.T) {
	output, err := Generate(context.Background(), fixtureWorkspace(t), generatedfile.Spec{Type: "go-docs", Input: []string{"store/"}})
	if err != nil {
		t.Fatal(err)
	}

	if len(output.Files) != 2 || output.Files[1].Path != "store/README.md" || len(output.Warnings) != 0 {
		t.Fatalf("pages %s, warnings %q", joined(output.Files), output.Warnings)
	}
}

func TestGenerateWithNoPackagesStillWritesTheIndexSoStalePagesGoAway(t *testing.T) {
	output, err := Generate(context.Background(), t.TempDir(), generatedfile.Spec{Type: "go-docs"})
	if err != nil {
		t.Fatal(err)
	}

	if len(output.Files) != 1 || !strings.Contains(string(output.Files[0].Body), "No Go packages were found.") {
		t.Fatalf("pages %s", joined(output.Files))
	}
}

func TestGenerateStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Generate(ctx, fixtureWorkspace(t), generatedfile.Spec{Type: "go-docs"}); err == nil {
		t.Fatal("expected the cancellation")
	}
}
