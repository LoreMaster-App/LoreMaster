package formatdetection

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func workspace(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

const (
	junit     = `<testsuite name="a"></testsuite>`
	cobertura = `<coverage line-rate="1"></coverage>`
	index     = `{"v":5,"entries":{"a--b":{"id":"a--b","type":"story","title":"A","importPath":"./a.tsx"}}}`
)

func TestDetectRecognisesTheFormatTheInputSelects(t *testing.T) {
	root := workspace(t, map[string]string{"build/a.xml": junit, "build/b.xml": junit, "build/log.txt": "noise", "other/c.xml": cobertura})
	got, err := Detect(context.Background(), root, generatedfile.Spec{Input: []string{"build/"}, Output: "docs/tests"})
	if err != nil || got.Type != TestResults {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestDetectFindsTheStorybookFolderAmongTheInputs(t *testing.T) {
	root := workspace(t, map[string]string{"storybook-static/index.json": index, "storybook-static/iframe.html": "<html>"})
	got, err := Detect(context.Background(), root, generatedfile.Spec{Input: []string{"storybook-static/"}, Output: "docs/ui"})
	if err != nil || got.Type != Storybook || got.Source != "storybook-static" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestDetectTakesASourceAsAStorybook(t *testing.T) {
	got, err := Detect(context.Background(), t.TempDir(), generatedfile.Spec{Source: "https://ui.example.test"})
	if err != nil || got.Type != Storybook || got.Source != "https://ui.example.test" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestDetectSaysWhenItCannotIdentifyTheOutput(t *testing.T) {
	root := workspace(t, map[string]string{"out/pom.xml": "<project/>", "out/data.json": `{"a":1}`})
	_, err := Detect(context.Background(), root, generatedfile.Spec{Input: []string{"out/"}, Output: "docs/x"})
	if err == nil || !strings.Contains(err.Error(), "could not identify the output") || !strings.Contains(err.Error(), "2 file(s)") || !strings.Contains(err.Error(), "out/pom.xml is XML") || !strings.Contains(err.Error(), "no parser available") {
		t.Fatalf("err = %v", err)
	}

	_, err = Detect(context.Background(), root, generatedfile.Spec{Input: []string{"missing/"}, Output: "docs/x"})
	if err == nil || !strings.Contains(err.Error(), "selects no file") {
		t.Fatalf("err = %v", err)
	}
}

func TestDetectRefusesSeveralFormatsAndNamesThem(t *testing.T) {
	root := workspace(t, map[string]string{"out/a.xml": junit, "out/b.xml": cobertura})
	_, err := Detect(context.Background(), root, generatedfile.Spec{Input: []string{"out/"}, Output: "docs/x"})
	if err == nil || !strings.Contains(err.Error(), "test-results (out/a.xml)") || !strings.Contains(err.Error(), "coverage (out/b.xml)") {
		t.Fatalf("err = %v", err)
	}
}

func TestDetectNeedsAnInputToLookAt(t *testing.T) {
	if _, err := Detect(context.Background(), t.TempDir(), generatedfile.Spec{Output: "docs/x"}); err == nil {
		t.Fatal("want an error without input")
	}
}

func TestDetectNeverReadsTheGeneratorsOwnOutput(t *testing.T) {
	root := workspace(t, map[string]string{"docs/tests/README.md": "# generated", "reports/a.xml": junit})
	got, err := Detect(context.Background(), root, generatedfile.Spec{Input: []string{"docs/", "reports/"}, Output: "docs/tests"})
	if err != nil || got.Type != TestResults {
		t.Fatalf("got %+v, %v", got, err)
	}
}
