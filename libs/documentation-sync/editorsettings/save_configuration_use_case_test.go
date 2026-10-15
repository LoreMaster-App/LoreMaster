package editorsettings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/workspacesettings"
)

func settingsWithOutput(space string, prefix string) workspacesettings.Settings {
	return workspacesettings.Settings{
		Version: workspacesettings.CurrentVersion,
		Outputs: []workspacesettings.Output{{
			Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: space, ParentPageID: "42", TitlePrefix: prefix,
			Direction: "to-platform", MermaidMode: "image", TitleCollision: "fail", LinkMode: "title",
			Content: []workspacesettings.Content{{Type: "markdown", Roots: []string{"."}, Template: "default"}},
		}},
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func TestAWorkspaceFileReceivesTheConfigurationAndNoVSCodeFolderIsCreated(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, filepath.Join(t.TempDir(), "none.json"))
	workspaceFile := filepath.Join(t.TempDir(), "app.code-workspace")
	writeSettings(t, workspaceFile, "{\n  // the folders\n  \"folders\": [{ \"path\": \".\" }],\n  \"settings\": {\n    // keep this\n    \"editor.tabSize\": 2\n  }\n}\n")
	t.Setenv(WorkspaceFileVariable, workspaceFile)

	target, err := SaveConfiguration(root, settingsWithOutput("AUT", "QM"))
	if err != nil {
		t.Fatal(err)
	}

	if !target.InWorkspaceFile || target.Path != workspaceFile {
		t.Fatalf("target = %+v", target)
	}
	if exists(filepath.Join(root, ".vscode")) {
		t.Fatal("the .vscode folder was created although a workspace file is in use")
	}
	saved, _ := os.ReadFile(workspaceFile)
	text := string(saved)
	for _, want := range []string{"// the folders", "// keep this", `"editor.tabSize": 2`, `"loreMaster.outputs"`, `"space": "AUT"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in\n%s", want, text)
		}
	}

	loaded, err := LoadEditorSettings(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Configured || loaded.Settings.Outputs[0].Space != "AUT" || loaded.Settings.Outputs[0].TitlePrefix != "QM" {
		t.Fatalf("loaded %+v", loaded.Settings.Outputs)
	}
}

func TestASecondSaveReplacesTheOutputsInTheWorkspaceFileInPlace(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, filepath.Join(t.TempDir(), "none.json"))
	workspaceFile := filepath.Join(t.TempDir(), "app.code-workspace")
	writeSettings(t, workspaceFile, `{"folders": [{"path": "."}], "settings": {}}`)
	t.Setenv(WorkspaceFileVariable, workspaceFile)

	if _, err := SaveConfiguration(root, settingsWithOutput("AUT", "QM")); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveConfiguration(root, settingsWithOutput("OPS", "OPS")); err != nil {
		t.Fatal(err)
	}

	saved, _ := os.ReadFile(workspaceFile)
	if strings.Count(string(saved), `"loreMaster.outputs"`) != 1 || strings.Contains(string(saved), "AUT") || !strings.Contains(string(saved), `"space": "OPS"`) {
		t.Fatalf("workspace file:\n%s", saved)
	}
}

func TestAWorkspaceFileWithoutSettingsGetsThem(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, filepath.Join(t.TempDir(), "none.json"))
	workspaceFile := filepath.Join(t.TempDir(), "app.code-workspace")
	writeSettings(t, workspaceFile, `{"folders": [{"path": "."}]}`)
	t.Setenv(WorkspaceFileVariable, workspaceFile)

	if _, err := SaveConfiguration(root, settingsWithOutput("AUT", "QM")); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadEditorSettings(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Configured || len(loaded.Settings.Outputs) != 1 {
		t.Fatalf("loaded %+v", loaded)
	}
}

func TestWithoutAWorkspaceFileTheFolderIsUsed(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, filepath.Join(t.TempDir(), "none.json"))
	t.Setenv(WorkspaceFileVariable, "")

	target, err := SaveConfiguration(root, settingsWithOutput("AUT", "QM"))
	if err != nil {
		t.Fatal(err)
	}

	if target.InWorkspaceFile || target.Path != FolderSettingsPath(root) || !exists(target.Path) {
		t.Fatalf("target = %+v", target)
	}
}

func TestAFolderThatAlreadyHoldsTheConfigurationKeepsIt(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, filepath.Join(t.TempDir(), "none.json"))
	workspaceFile := filepath.Join(t.TempDir(), "app.code-workspace")
	writeSettings(t, workspaceFile, `{"folders": [{"path": "."}], "settings": {}}`)
	t.Setenv(WorkspaceFileVariable, workspaceFile)
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.ignore": ["drafts/"]}`)

	target, err := SaveConfiguration(root, settingsWithOutput("AUT", "QM"))
	if err != nil {
		t.Fatal(err)
	}

	if target.InWorkspaceFile || target.Path != FolderSettingsPath(root) {
		t.Fatalf("a folder's own configuration would shadow the workspace file: %+v", target)
	}
}

func TestAFolderWithOnlyOtherSettingsDoesNotCountAsConfigured(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, filepath.Join(t.TempDir(), "none.json"))
	workspaceFile := filepath.Join(t.TempDir(), "app.code-workspace")
	writeSettings(t, workspaceFile, `{"folders": [{"path": "."}], "settings": {}}`)
	t.Setenv(WorkspaceFileVariable, workspaceFile)
	writeSettings(t, FolderSettingsPath(root), `{"editor.tabSize": 4}`)

	target, err := SaveConfiguration(root, settingsWithOutput("AUT", "QM"))
	if err != nil {
		t.Fatal(err)
	}

	if !target.InWorkspaceFile {
		t.Fatalf("target = %+v", target)
	}
	if got, _ := os.ReadFile(FolderSettingsPath(root)); string(got) != `{"editor.tabSize": 4}` {
		t.Fatalf("the folder's settings were touched: %s", got)
	}
}

func TestAWorkspaceFileThatIsGoneFallsBackToTheFolder(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, filepath.Join(t.TempDir(), "none.json"))
	t.Setenv(WorkspaceFileVariable, filepath.Join(t.TempDir(), "missing.code-workspace"))

	target, err := SaveConfiguration(root, settingsWithOutput("AUT", "QM"))
	if err != nil {
		t.Fatal(err)
	}

	if target.InWorkspaceFile {
		t.Fatalf("target = %+v", target)
	}
}

func TestSettingsInTheWorkspaceFileAreNotCopiedIntoWhatIsWritten(t *testing.T) {
	root := t.TempDir()
	t.Setenv(UserSettingsOverrideVariable, filepath.Join(t.TempDir(), "none.json"))
	workspaceFile := filepath.Join(t.TempDir(), "app.code-workspace")
	writeSettings(t, workspaceFile, `{"folders": [{"path": "."}], "settings": {"loreMaster.defaults.mermaidMode": "code", "loreMaster.defaults.titlePrefix": "QM"}}`)
	t.Setenv(WorkspaceFileVariable, workspaceFile)
	settings := settingsWithOutput("AUT", "QM")
	settings.Outputs[0].MermaidMode = "code"

	if _, err := SaveConfiguration(root, settings); err != nil {
		t.Fatal(err)
	}

	saved, _ := os.ReadFile(workspaceFile)
	outputs := string(saved)[strings.Index(string(saved), `"loreMaster.outputs"`):]
	if strings.Contains(outputs, "mermaidMode") || strings.Contains(outputs, "titlePrefix") {
		t.Fatalf("inherited values were written into the outputs:\n%s", saved)
	}
}
