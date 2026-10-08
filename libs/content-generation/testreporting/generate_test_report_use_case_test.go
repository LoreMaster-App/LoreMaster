package testreporting

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func workspace(t *testing.T, files map[string][]byte) string {
	t.Helper()
	root := t.TempDir()
	for relative, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func pathsOf(output generatedfile.Output) []string {
	paths := make([]string, len(output.Files))
	for i, file := range output.Files {
		paths[i] = file.Path
	}

	return paths
}

func TestGenerateFindsReportsAnywhereByTheirUsualNames(t *testing.T) {
	root := workspace(t, map[string][]byte{
		"junit.xml":                       fixture(t, "jest-junit.xml"),
		"build/reports/TEST-go.xml":       fixture(t, "go-junit.xml"),
		"coverage/lcov-report/index.html": []byte("<html/>"),
		"pom.xml":                         []byte("<project/>"),
		"node_modules/dep/junit.xml":      fixture(t, "jest-junit.xml"),
	})

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "test-results"})
	if err != nil {
		t.Fatal(err)
	}

	if len(output.Warnings) != 0 {
		t.Fatalf("warnings %q", output.Warnings)
	}
	want := []string{"README.md", "lore-master-libs-confluence-client-storageformat.md", "lore-master-libs-markdown-workspace-documentparsing.md", "pages-view.md", "status-policy.md"}
	if got := pathsOf(output); !slices.Equal(got, want) {
		t.Fatalf("pages %q, want %q", got, want)
	}
	if !strings.HasPrefix(string(output.Files[0].Body), "# Test results\n") {
		t.Fatalf("index %s", output.Files[0].Body)
	}
}

func TestGenerateUsesTheConfiguredInputAndTitle(t *testing.T) {
	root := workspace(t, map[string][]byte{
		"ci/results/go.xml": fixture(t, "go-junit.xml"),
		"junit.xml":         fixture(t, "jest-junit.xml"),
	})

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "test-results", Input: []string{"ci/results/"}, Title: "CI results"})
	if err != nil {
		t.Fatal(err)
	}

	if got := pathsOf(output); len(got) != 3 {
		t.Fatalf("pages %q", got)
	}
	if !strings.HasPrefix(string(output.Files[0].Body), "# CI results\n") {
		t.Fatalf("index %s", output.Files[0].Body)
	}
}

func TestGenerateReportsABadReportAndStillRendersTheGoodOnes(t *testing.T) {
	root := workspace(t, map[string][]byte{
		"junit.xml":       fixture(t, "jest-junit.xml"),
		"junit-bad.xml":   []byte("<testsuites><testsuite"),
		"junit-other.xml": []byte("<html/>"),
	})

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "test-results"})
	if err != nil {
		t.Fatal(err)
	}

	if len(output.Warnings) != 2 || !strings.HasPrefix(output.Warnings[0], "junit-bad.xml: ") || !strings.HasPrefix(output.Warnings[1], "junit-other.xml: ") {
		t.Fatalf("warnings %q", output.Warnings)
	}
	if got := pathsOf(output); len(got) != 3 {
		t.Fatalf("pages %q", got)
	}
}

func TestGenerateWithNoReportsStillWritesTheIndexSoStaleResultsGoAway(t *testing.T) {
	output, err := Generate(context.Background(), workspace(t, nil), generatedfile.Spec{Type: "test-results"})
	if err != nil {
		t.Fatal(err)
	}

	if got := pathsOf(output); !slices.Equal(got, []string{"README.md"}) {
		t.Fatalf("pages %q", got)
	}
	if !strings.Contains(string(output.Files[0].Body), "No test reports were found.") {
		t.Fatalf("index %s", output.Files[0].Body)
	}
}

func TestGenerateStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Generate(ctx, workspace(t, map[string][]byte{"junit.xml": fixture(t, "jest-junit.xml")}), generatedfile.Spec{Type: "test-results"}); err == nil {
		t.Fatal("expected the cancellation")
	}
}
