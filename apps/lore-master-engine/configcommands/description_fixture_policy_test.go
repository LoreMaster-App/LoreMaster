package configcommands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"lore-master/libs/documentation-sync/configcatalog"
)

// The VS Code tests describe the configuration from a copy of the catalog, so they test against what
// the engine really says. This keeps that copy current: when the catalog changes, run
//
//	UPDATE_CONFIG_FIXTURE=1 go test ./apps/lore-master-engine/configcommands
//
// and commit the file.
const fixturePath = "../../lore-master-vscode/test/config-description/config-description.fixture.json"

func TestTheExtensionTestFixtureIsTheCatalog(t *testing.T) {
	current, err := json.MarshalIndent(configcatalog.Describe(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.FromSlash(fixturePath)
	if os.Getenv("UPDATE_CONFIG_FIXTURE") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(current, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v; run with UPDATE_CONFIG_FIXTURE=1 to write it", err)
	}
	var want, got any
	if err := json.Unmarshal(current, &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stored, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatal("config-description.fixture.json is not the catalog any more; run with UPDATE_CONFIG_FIXTURE=1 and commit the file")
	}
}
