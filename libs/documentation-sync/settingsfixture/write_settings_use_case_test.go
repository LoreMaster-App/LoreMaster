package settingsfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAYamlConfigurationIsWrittenAsTheEditorsSettings(t *testing.T) {
	root := t.TempDir()

	Write(t, root, "version: 1\nignore: [drafts/]\noutputs:\n  - platform: confluence\n    space: ENG\n")

	raw, err := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatal(err)
	}
	if _, kept := settings["loreMaster.version"]; kept || settings["loreMaster.ignore"] == nil || settings["loreMaster.outputs"] == nil {
		t.Fatalf("settings: %v", settings)
	}
}
