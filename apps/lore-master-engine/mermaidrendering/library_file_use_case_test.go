package mermaidrendering

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibraryPathPrefersWhatIsAskedThenTheVariableThenTheEmbeddedCopy(t *testing.T) {
	t.Setenv("LORE_MASTER_MERMAID", "")
	if got := LibraryPath("/mine/mermaid.js"); got != "/mine/mermaid.js" {
		t.Fatalf("asked path was not kept: %q", got)
	}
	named := filepath.Join(t.TempDir(), "mermaid.js")
	if err := os.WriteFile(named, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LORE_MASTER_MERMAID", named)
	if got := LibraryPath(""); got != named {
		t.Fatalf("the variable was not used: %q", got)
	}
	t.Setenv("LORE_MASTER_MERMAID", "editor")
	if got := LibraryPath(""); got == "editor" || got == "" {
		t.Fatalf("the word editor is not a library: %q", got)
	}

	t.Setenv("LORE_MASTER_MERMAID", "")
	path := LibraryPath("")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the embedded copy was not written: %v", err)
	}
	if !strings.Contains(string(content), "mermaid") || len(content) < 100000 {
		t.Fatalf("not the library: %d bytes", len(content))
	}
	if again := LibraryPath(""); again != path {
		t.Fatalf("the copy moved between calls: %q then %q", path, again)
	}
}
