package coveragereporting

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateReadsReportsMergesThemAndWarnsAboutTheRest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "coverage/lcov.info", "SF:src/a.ts\nDA:1,1\nDA:2,0\nend_of_record\n")
	writeFile(t, root, "reports/coverage.xml", coberturaSample)
	writeFile(t, root, "reports/broken-cobertura.xml", "<not-coverage/>")
	writeFile(t, root, "node_modules/x/lcov.info", "SF:ignored.ts\nDA:1,1\nend_of_record\n")

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "coverage", Title: "Coverage"})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Warnings) != 1 || !strings.Contains(output.Warnings[0], "broken-cobertura.xml") {
		t.Fatalf("warnings = %v, want one for the broken report", output.Warnings)
	}
	var all strings.Builder
	for _, page := range output.Files {
		all.Write(page.Body)
	}
	for _, expected := range []string{"a.ts", "cart.py", "totals.py"} {
		if !strings.Contains(all.String(), expected) {
			t.Errorf("pages lack %s:\n%s", expected, all.String())
		}
	}
	if strings.Contains(all.String(), "ignored.ts") {
		t.Error("read a report inside node_modules")
	}
}

func TestGenerateWithNoReportsStillWritesTheIndex(t *testing.T) {
	output, err := Generate(context.Background(), t.TempDir(), generatedfile.Spec{Type: "coverage"})
	if err != nil || len(output.Files) != 1 || !strings.Contains(string(output.Files[0].Body), "No coverage reports") {
		t.Fatalf("output = %+v, err = %v", output, err)
	}
}
