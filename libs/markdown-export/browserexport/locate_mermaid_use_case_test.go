package browserexport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocateMermaidTakesAFileAndIgnoresWhatIsNotOne(t *testing.T) {
	file := filepath.Join(t.TempDir(), "mermaid.min.js")
	if err := os.WriteFile(file, []byte("//"), 0o600); err != nil {
		t.Fatal(err)
	}

	if LocateMermaid(file) != file || LocateMermaid("") != "" || LocateMermaid(t.TempDir()) != "" || LocateMermaid(filepath.Join(t.TempDir(), "missing.js")) != "" {
		t.Fatal("the library was not located as expected")
	}
	t.Setenv(MermaidVariable, file)
	if LocateMermaid("") != file {
		t.Fatal("the variable was not used")
	}
}
