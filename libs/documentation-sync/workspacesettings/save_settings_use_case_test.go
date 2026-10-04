package workspacesettings

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const authored = `# Our team's sync. Ask #docs before changing the parent page.
version: 1
outputs:
  - platform: confluence
    # The space and parent are filled in by the first sync.
    baseUrl: https://acme.atlassian.net/wiki
    space: ""
    parentPageId: ""
    titlePrefix: ""
    content:
      - type: markdown
        roots: [docs, README.md] # only these
        excludes: ["drafts/**"]
        template: default
    linkMode: title # keep links readable
    mermaidMode: image
`

func TestSavingKeepsTheAuthorsCommentsAndOrder(t *testing.T) {
	root := workspaceWith(t, authored)
	loaded, err := LoadSettings(root)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.FirstSync {
		t.Fatal("empty first-sync answers mean a first sync")
	}
	settings := loaded.Settings
	settings.Outputs[0].Space, settings.Outputs[0].ParentPageID, settings.Outputs[0].TitlePrefix = "ENG", "123000", "MNCI"
	if err := SaveSettings(loaded, settings); err != nil {
		t.Fatal(err)
	}
	written, _ := os.ReadFile(filepath.Join(root, FileName))
	text := string(written)
	for _, kept := range []string{
		"# Our team's sync. Ask #docs before changing the parent page.",
		"# The space and parent are filled in by the first sync.",
		"# only these",
		"# keep links readable",
		"space: ENG",
		`parentPageId: "123000"`,
		"titlePrefix: MNCI",
	} {
		if !strings.Contains(text, kept) {
			t.Errorf("saved file lacks %q:\n%s", kept, text)
		}
	}
	// The author's order (linkMode before mermaidMode) survives; defaults the author
	// left out are appended after it.
	if strings.Index(text, "linkMode") > strings.Index(text, "mermaidMode") || strings.Index(text, "mermaidMode") > strings.Index(text, "titleCollision") {
		t.Errorf("key order changed:\n%s", text)
	}
	if !strings.HasSuffix(text, "\n") || strings.Contains(text, "\r") {
		t.Errorf("expected LF endings and a trailing newline")
	}

	reloaded, err := LoadSettings(root)
	if err != nil || reloaded.FirstSync || !reflect.DeepEqual(reloaded.Settings, settings) {
		t.Fatalf("reloaded %+v, err %v", reloaded, err)
	}
}

func TestSavingANewFileExplainsItself(t *testing.T) {
	root := workspaceWith(t, "")
	loaded, _ := LoadSettings(root)
	settings := loaded.Settings
	settings.Outputs[0].BaseURL, settings.Outputs[0].Space, settings.Outputs[0].ParentPageID, settings.Outputs[0].TitlePrefix = "https://acme.atlassian.net/wiki", "ENG", "1", "MNCI"
	if err := SaveSettings(loaded, settings); err != nil {
		t.Fatal(err)
	}
	written, _ := os.ReadFile(filepath.Join(root, FileName))
	if !strings.HasPrefix(string(written), "# Lore Master configuration. Commit this file; it never holds a secret\n") {
		t.Fatalf("new file:\n%s", written)
	}
	reloaded, err := LoadSettings(root)
	if err != nil || !reflect.DeepEqual(reloaded.Settings, settings) {
		t.Fatalf("reloaded %+v, err %v", reloaded.Settings, err)
	}
}

func TestSavingRefusesInvalidSettings(t *testing.T) {
	root := workspaceWith(t, "")
	loaded, _ := LoadSettings(root)
	settings := loaded.Settings
	settings.Outputs[0].Direction = "sideways"
	if err := SaveSettings(loaded, settings); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(filepath.Join(root, FileName)); !os.IsNotExist(err) {
		t.Fatal("an invalid save must not write the file")
	}
}
