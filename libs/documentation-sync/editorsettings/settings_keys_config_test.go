package editorsettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"lore-master/libs/documentation-sync/workspacesettings"
)

// The extension contributes the settings the Settings UI shows; this loader reads them.
// The two lists must not drift, or an option is offered that nothing reads (or the reverse).
func TestEverySettingTheLoaderReadsIsContributedByTheExtension(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "apps", "lore-master-vscode", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Contributes struct {
			Configuration struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"configuration"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	properties := manifest.Contributes.Configuration.Properties

	for key := range configKeys {
		if _, contributed := properties[key]; !contributed {
			t.Errorf("%s is read by the loader but not contributed by the extension", key)
		}
	}
	for key := range properties {
		if !configKeys[key] && !isExtensionOnly(key) {
			t.Errorf("%s is contributed by the extension but is neither read by the loader nor listed as extension-only", key)
		}
	}

	var generators struct {
		Items struct {
			Properties struct {
				Type struct {
					Enum []string `json:"enum"`
				} `json:"type"`
			} `json:"properties"`
		} `json:"items"`
	}
	if err := json.Unmarshal(properties[keyGenerators], &generators); err != nil {
		t.Fatal(err)
	}
	if got, want := generators.Items.Properties.Type.Enum, workspacesettings.BuiltGeneratorTypes; !slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want))) {
		t.Errorf("generator types contributed %v, built %v", got, want)
	}
}
