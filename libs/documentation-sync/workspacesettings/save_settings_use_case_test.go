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
	if !strings.HasPrefix(string(written), "# LoreMaster configuration. Commit this file; it never holds a secret\n") {
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

const withOptionalKeys = `version: 1
# Keep these generators in step with the CI reports.
skipGitignored: false
ignore: [drafts/]
generators:
  - type: test-results
    input: [reports/]
    output: docs/tests
    title: CI results
outputs:
  - platform: github-pages
    repo: acme/handbook # published here
    branch: docs-site
    content:
      - type: markdown
        roots: [docs]
        excludes: ["drafts/**"]
        template: default
`

// Clearing an optional value in the editor sends it as absent; saving must remove it from the
// file rather than keep what was there, or the change silently does not happen.
func TestSavingRemovesAnOptionalValueThatWasCleared(t *testing.T) {
	root := workspaceWith(t, withOptionalKeys)
	loaded, err := LoadSettings(root)
	if err != nil {
		t.Fatal(err)
	}
	settings := loaded.Settings
	settings.SkipGitignored = nil
	settings.Ignore = nil
	settings.Generators = nil
	settings.Outputs[0].Repo, settings.Outputs[0].Branch = "", ""
	settings.Outputs[0].Content[0].Excludes = nil
	if err := SaveSettings(loaded, settings); err != nil {
		t.Fatal(err)
	}

	written, _ := os.ReadFile(filepath.Join(root, FileName))
	text := string(written)
	for _, gone := range []string{"skipGitignored", "ignore:", "generators:", "test-results", "docs/tests", "repo:", "acme/handbook", "branch:", "docs-site", "excludes", "drafts"} {
		if strings.Contains(text, gone) {
			t.Errorf("a cleared value survived (%q):\n%s", gone, text)
		}
	}
	if !strings.Contains(text, "roots: [docs]") {
		t.Errorf("a value that was not cleared was lost:\n%s", text)
	}
	again, err := LoadSettings(root)
	if err != nil || len(again.Settings.Generators) != 0 || again.Settings.Outputs[0].Repo != "" {
		t.Fatalf("reloaded %+v (%v)", again.Settings, err)
	}
}

func TestSavingRemovesOneEntryFromAListAndKeepsTheComments(t *testing.T) {
	root := workspaceWith(t, withOptionalKeys)
	loaded, err := LoadSettings(root)
	if err != nil {
		t.Fatal(err)
	}
	settings := loaded.Settings
	settings.Generators = append(settings.Generators, Generator{Type: "go-docs", Output: "docs/api"})
	if err := SaveSettings(loaded, settings); err != nil {
		t.Fatal(err)
	}
	loaded, err = LoadSettings(root)
	if err != nil {
		t.Fatal(err)
	}
	settings = loaded.Settings
	settings.Generators = settings.Generators[1:]
	if err := SaveSettings(loaded, settings); err != nil {
		t.Fatal(err)
	}

	written, _ := os.ReadFile(filepath.Join(root, FileName))
	text := string(written)
	if strings.Contains(text, "test-results") || !strings.Contains(text, "go-docs") || !strings.Contains(text, "# Keep these generators in step with the CI reports.") {
		t.Fatalf("saved file:\n%s", text)
	}
}

func TestANewFileHeaderMentionsGeneratorsNotReservedContentTypes(t *testing.T) {
	if strings.Contains(newFileHeader, "test-results and code-docs") || !strings.Contains(newFileHeader, "generators") {
		t.Fatalf("header %q", newFileHeader)
	}
}
