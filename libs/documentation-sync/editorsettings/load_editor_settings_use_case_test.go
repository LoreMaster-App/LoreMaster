package editorsettings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/workspacesettings"
)

func writeSettings(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFolderOverridesUserAndUserFillsTheGaps(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(t.TempDir(), "settings.json")
	writeSettings(t, user, `{
  // set once, every workspace inherits it
  "loreMaster.confluence.baseUrl": "https://user.atlassian.net/wiki",
  "loreMaster.defaults.mermaidMode": "code",
  "loreMaster.ignore": ["user-only/"],
}`)
	writeSettings(t, FolderSettingsPath(root), `{
  "loreMaster.ignore": ["folder-only/"],
  "loreMaster.outputs": [{ "platform": "confluence", "space": "DOC", "parentPageId": "123", "titlePrefix": "Lore" }]
}`)

	result, err := LoadEditorSettings(root, user)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Configured || result.FirstSync {
		t.Fatalf("configured=%v firstSync=%v", result.Configured, result.FirstSync)
	}
	output := result.Settings.Outputs[0]
	if output.BaseURL != "https://user.atlassian.net/wiki" {
		t.Fatalf("baseUrl = %q, want the user's", output.BaseURL)
	}
	if output.MermaidMode != "code" {
		t.Fatalf("mermaidMode = %q, want the user's default", output.MermaidMode)
	}
	if got := strings.Join(result.Settings.Ignore, ","); got != "folder-only/" {
		t.Fatalf("ignore = %q, want the folder's list to replace the user's", got)
	}
	if len(result.Sources) != 2 {
		t.Fatalf("sources = %v", result.Sources)
	}
}

func TestAnOutputsBaseURLBeatsTheUserDefault(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(t.TempDir(), "settings.json")
	writeSettings(t, user, `{"loreMaster.confluence.baseUrl": "https://user.atlassian.net/wiki"}`)
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.outputs": [{"baseUrl": "https://other.example.com/wiki", "space": "X", "parentPageId": "1", "titlePrefix": "P"}]}`)

	result, err := LoadEditorSettings(root, user)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Settings.Outputs[0].BaseURL; got != "https://other.example.com/wiki" {
		t.Fatalf("baseUrl = %q", got)
	}
}

func TestNoOutputsMeansNotConfigured(t *testing.T) {
	root := t.TempDir()
	result, err := LoadEditorSettings(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Configured {
		t.Fatal("an empty workspace reported as configured")
	}
}

func TestASecretKeyIsRefusedAndNamed(t *testing.T) {
	root := t.TempDir()
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.confluence.token": "abc"}`)

	_, err := LoadEditorSettings(root, "")
	if err == nil || !strings.Contains(err.Error(), "looks like a secret") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnUnknownKeyAndAMisspeltFieldAreErrorsNamingTheirFile(t *testing.T) {
	root := t.TempDir()
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.outputz": []}`)
	if _, err := LoadEditorSettings(root, ""); err == nil || !strings.Contains(err.Error(), "loreMaster.outputz") {
		t.Fatalf("err = %v", err)
	}

	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.outputs": [{"platfrom": "confluence"}]}`)
	if _, err := LoadEditorSettings(root, ""); err == nil || !strings.Contains(err.Error(), "platfrom") {
		t.Fatalf("err = %v", err)
	}
}

func TestExtensionOnlySettingsAreIgnored(t *testing.T) {
	root := t.TempDir()
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.pages.label": "title", "loreMaster.watchDebounceSeconds": 5, "loreMaster.engine.path": ""}`)
	if _, err := LoadEditorSettings(root, ""); err != nil {
		t.Fatal(err)
	}
}

func TestUserSettingsPathPerSystem(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}
	if got := UserSettingsPath("linux", env(map[string]string{UserSettingsOverrideVariable: "/x/s.json"}), "/home/a"); got != "/x/s.json" {
		t.Fatalf("override = %q", got)
	}
	if got, want := UserSettingsPath("windows", env(map[string]string{"APPDATA": "C:/Users/a/AppData/Roaming"}), ""), filepath.Join("C:/Users/a/AppData/Roaming", "Code", "User", "settings.json"); got != want {
		t.Fatalf("windows = %q, want %q", got, want)
	}
	if got, want := UserSettingsPath("linux", env(nil), "/home/a"), filepath.Join("/home/a", ".config", "Code", "User", "settings.json"); got != want {
		t.Fatalf("linux = %q, want %q", got, want)
	}
	if got := UserSettingsPath("windows", env(nil), ""); got != "" {
		t.Fatalf("unknown location = %q", got)
	}
}

const oneOutput = `[{"platform":"confluence","space":"ENG","parentPageId":"1","titlePrefix":"ENG"}]`

func TestWorkspaceFileSitsBetweenUserAndFolder(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(t.TempDir(), "settings.json")
	writeSettings(t, user, `{"loreMaster.ignore": ["user/"], "loreMaster.skipGitignored": false}`)
	workspaceFile := filepath.Join(t.TempDir(), "team.code-workspace")
	writeSettings(t, workspaceFile, `{
  "folders": [{"path": "."}],
  "settings": {"loreMaster.outputs": `+oneOutput+`, "loreMaster.ignore": ["workspace/"]}
}`)
	t.Setenv(WorkspaceFileVariable, workspaceFile)

	result, err := LoadEditorSettings(root, user)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Configured || len(result.Settings.Outputs) != 1 {
		t.Fatalf("the workspace file's outputs were not read: %+v", result)
	}
	if got := result.Settings.Ignore; len(got) != 1 || got[0] != "workspace/" {
		t.Errorf("the workspace file must override the user's ignore, got %v", got)
	}

	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.ignore": ["folder/"]}`)
	result, err = LoadEditorSettings(root, user)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Settings.Ignore; len(got) != 1 || got[0] != "folder/" {
		t.Errorf("the folder must override the workspace file, got %v", got)
	}
}

func TestOverridesAreTheMostSpecificLayer(t *testing.T) {
	root := t.TempDir()
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.outputs": `+oneOutput+`, "loreMaster.ignore": ["folder/"]}`)
	t.Setenv(OverridesVariable, `{"loreMaster.ignore": ["ci/"]}`)

	result, err := LoadEditorSettings(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Settings.Ignore; len(got) != 1 || got[0] != "ci/" {
		t.Errorf("overrides must win over the folder, got %v", got)
	}
	if len(result.Settings.Outputs) != 1 {
		t.Errorf("a key the overrides leave alone must stay, got %d outputs", len(result.Settings.Outputs))
	}
}

func TestOverridesCanCarryTheWholeConfigurationFromAFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(t.TempDir(), "ci.json")
	writeSettings(t, file, `{"loreMaster.outputs": `+oneOutput+`}`)
	t.Setenv(OverridesVariable, "@"+file)

	result, err := LoadEditorSettings(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Configured || len(result.Settings.Outputs) != 1 {
		t.Errorf("the overrides file was not read: %+v", result)
	}
}

func TestOverridesRefuseASecretAndAnUnknownKey(t *testing.T) {
	for name, text := range map[string]string{
		"secret":  `{"loreMaster.confluence.token": "abc"}`,
		"unknown": `{"loreMaster.outputz": []}`,
		"broken":  `{`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(OverridesVariable, text)
			if _, err := LoadEditorSettings(t.TempDir(), ""); err == nil || !strings.Contains(err.Error(), OverridesVariable) {
				t.Errorf("want an error naming %s, got %v", OverridesVariable, err)
			}
		})
	}
}

func TestTheTitlePrefixDefaultFillsAnOutputThatSetsNone(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(t.TempDir(), "settings.json")
	writeSettings(t, user, `{"loreMaster.defaults.titlePrefix": "Docs"}`)
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.outputs": [
  {"space": "A", "parentPageId": "1"},
  {"space": "B", "parentPageId": "2", "titlePrefix": "Own"}
]}`)

	result, err := LoadEditorSettings(root, user)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Settings.Outputs[0].TitlePrefix; got != "Docs" {
		t.Fatalf("titlePrefix = %q, want the default", got)
	}
	if got := result.Settings.Outputs[1].TitlePrefix; got != "Own" {
		t.Fatalf("titlePrefix = %q, want the output's own", got)
	}
}

func TestAnEmptyOutputsListInTheFolderEmptiesWhatAnotherLayerSet(t *testing.T) {
	root := t.TempDir()
	workspaceFile := filepath.Join(t.TempDir(), "app.code-workspace")
	writeSettings(t, workspaceFile, `{"settings": {"loreMaster.outputs": [{"space": "AUT", "parentPageId": "1", "titlePrefix": "QM", "baseUrl": "https://x.atlassian.net/wiki"}]}}`)
	t.Setenv(WorkspaceFileVariable, workspaceFile)
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.outputs": []}`)

	result, err := LoadEditorSettings(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Configured || len(result.Settings.Outputs) != 0 {
		t.Fatalf("configured=%v outputs=%+v", result.Configured, result.Settings.Outputs)
	}
}

func TestSavingNoOutputsKeepsTheKeyAndTheRestOfTheFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, filepath.Join(t.TempDir(), "none.json"))
	writeSettings(t, FolderSettingsPath(root), "{\n  // keep\n  \"editor.tabSize\": 2,\n  \"loreMaster.outputs\": [{\"space\": \"AUT\", \"parentPageId\": \"1\", \"titlePrefix\": \"QM\", \"baseUrl\": \"https://x.atlassian.net/wiki\"}]\n}\n")

	if err := SaveFolderSettings(root, workspacesettings.Settings{Version: workspacesettings.CurrentVersion, Outputs: nil}); err != nil {
		t.Fatal(err)
	}

	saved, err := os.ReadFile(FolderSettingsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	text := string(saved)
	if !strings.Contains(text, "// keep") || !strings.Contains(text, `"loreMaster.outputs": []`) || strings.Contains(text, "AUT") {
		t.Fatalf("saved:\n%s", text)
	}
}
