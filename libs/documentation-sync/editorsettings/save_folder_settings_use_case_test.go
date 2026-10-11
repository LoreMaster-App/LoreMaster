package editorsettings

import (
	"os"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/workspacesettings"
)

func TestSaveFolderSettingsRoundTripsAndDoesNotCopyInheritedValues(t *testing.T) {
	root := t.TempDir()
	user := t.TempDir() + "/settings.json"
	writeSettings(t, user, `{"loreMaster.confluence.baseUrl": "https://user.atlassian.net/wiki", "loreMaster.defaults.mermaidMode": "code"}`)
	t.Setenv(UserSettingsOverrideVariable, user)
	writeSettings(t, FolderSettingsPath(root), "{\n  // keep me\n  \"editor.tabSize\": 2\n}\n")

	settings := workspacesettings.Settings{
		Version: workspacesettings.CurrentVersion,
		Ignore:  []string{"internal/"},
		Outputs: []workspacesettings.Output{{
			Platform: "confluence", BaseURL: "https://user.atlassian.net/wiki", Space: "ENG", ParentPageID: "1", TitlePrefix: "ENG",
			Direction: "to-platform", MermaidMode: "code", TitleCollision: "fail", LinkMode: "title",
			Content: []workspacesettings.Content{{Type: "markdown", Roots: []string{"."}, Template: "default"}},
			Exclude: []string{"drafts/"},
		}},
	}
	if err := SaveFolderSettings(root, settings); err != nil {
		t.Fatal(err)
	}

	saved, err := os.ReadFile(FolderSettingsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	text := string(saved)
	if !strings.Contains(text, "// keep me") || !strings.Contains(text, `"editor.tabSize": 2`) {
		t.Fatalf("lost the author's content:\n%s", text)
	}
	for _, copied := range []string{"baseUrl", "mermaidMode", "content", "direction"} {
		if strings.Contains(text, copied) {
			t.Fatalf("wrote the inherited or default %q into the folder:\n%s", copied, text)
		}
	}

	loaded, err := LoadConfiguration(root)
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.Settings.Outputs[0]
	if got.BaseURL != "https://user.atlassian.net/wiki" || got.MermaidMode != "code" || got.Space != "ENG" || len(got.Exclude) != 1 {
		t.Fatalf("round trip lost something: %+v", got)
	}
	if len(loaded.Settings.Ignore) != 1 {
		t.Fatalf("ignore = %v", loaded.Settings.Ignore)
	}
}

func TestSaveFolderSettingsKeepsAnOutputsOwnBaseURLAndDropsClearedLists(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, t.TempDir()+"/none.json")
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.ignore": ["old/"]}`)

	settings := workspacesettings.Settings{
		Version: workspacesettings.CurrentVersion,
		Outputs: []workspacesettings.Output{{
			Platform: "confluence", BaseURL: "https://other.example.com/wiki", Space: "X", ParentPageID: "1", TitlePrefix: "P",
			Direction: "to-platform", MermaidMode: "image", TitleCollision: "fail", LinkMode: "title",
			Content: []workspacesettings.Content{{Type: "markdown", Roots: []string{"docs"}, Template: "default"}},
		}},
	}
	if err := SaveFolderSettings(root, settings); err != nil {
		t.Fatal(err)
	}
	saved, _ := os.ReadFile(FolderSettingsPath(root))
	text := string(saved)
	if !strings.Contains(text, "https://other.example.com/wiki") || !strings.Contains(text, `"docs"`) {
		t.Fatalf("lost the output's own values:\n%s", text)
	}
	if strings.Contains(text, "loreMaster.ignore") {
		t.Fatalf("a cleared list stayed:\n%s", text)
	}
}

func TestSaveFolderSettingsRefusesInvalidSettings(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, t.TempDir()+"/none.json")
	err := SaveFolderSettings(root, workspacesettings.Settings{Version: workspacesettings.CurrentVersion})
	if err == nil {
		t.Fatal("expected a validation error for no outputs")
	}
	if _, statErr := os.Stat(FolderSettingsPath(root)); statErr == nil {
		t.Fatal("a refused save wrote the file")
	}
}
