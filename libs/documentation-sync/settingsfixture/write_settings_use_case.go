package settingsfixture

import (
	"os"
	"path/filepath"
	"testing"
)

// Write puts the YAML configuration into root/.vscode/settings.json, where the engine reads it.
func Write(t testing.TB, root string, configuration string) {
	t.Helper()
	content, err := SettingsJSON(configuration)
	if err != nil {
		t.Fatalf("the settings fixture is not YAML: %v", err)
	}
	file := filepath.Join(root, ".vscode", "settings.json")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
