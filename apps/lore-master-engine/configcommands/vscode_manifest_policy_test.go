package configcommands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"lore-master/libs/documentation-sync/configcatalog"
)

// VS Code reads the shape of its settings from package.json, which cannot be built at run time, so
// the manifest is the one place a copy of the catalog is allowed to live. This test is what keeps
// that copy honest: a platform, generator type or setting added to the catalog and not to the
// manifest fails here instead of being silently missing from VS Code's settings editor.

type manifestProperty struct {
	Type       string                      `json:"type"`
	Default    any                         `json:"default"`
	Enum       []any                       `json:"enum"`
	Minimum    *float64                    `json:"minimum"`
	Maximum    *float64                    `json:"maximum"`
	Properties map[string]manifestProperty `json:"properties"`
	Items      *manifestProperty           `json:"items"`
}

func loadManifest(t *testing.T) map[string]manifestProperty {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "lore-master-vscode", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Contributes struct {
			Configuration struct {
				Properties map[string]manifestProperty `json:"properties"`
			} `json:"configuration"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}

	return manifest.Contributes.Configuration.Properties
}

func enumValues(property manifestProperty) []string {
	var values []string
	for _, value := range property.Enum {
		if text, ok := value.(string); ok && text != "" {
			values = append(values, text)
		}
	}

	return values
}

func choiceValues(field configcatalog.Field) []string {
	values := make([]string, len(field.Choices))
	for i, choice := range field.Choices {
		values[i] = choice.Value
	}

	return values
}

func manifestType(field configcatalog.Field) string {
	switch field.Type {
	case configcatalog.FieldBool:
		return "boolean"
	case configcatalog.FieldNumber:
		return "number"
	case configcatalog.FieldList:
		return "array"
	}

	return "string"
}

func TestTheVSCodeManifestHasEveryPlatformAndOutputField(t *testing.T) {
	properties := loadManifest(t)
	outputs, found := properties["loreMaster.outputs"]
	if !found || outputs.Items == nil {
		t.Fatal("the manifest has no loreMaster.outputs")
	}
	description := configcatalog.Describe()

	if got := enumValues(outputs.Items.Properties["platform"]); !slices.Equal(got, configcatalog.PlatformIDs()) {
		t.Errorf("the manifest offers the platforms %v, the catalog %v", got, configcatalog.PlatformIDs())
	}
	keys := description.OutputFields
	for _, platform := range description.Platforms {
		keys = append(keys, platform.Fields...)
	}
	for _, field := range keys {
		property, found := outputs.Items.Properties[field.Key]
		if !found {
			t.Errorf("the manifest's outputs have no %q", field.Key)

			continue
		}
		if field.Type == configcatalog.FieldChoice && !slices.Equal(enumValues(property), choiceValues(field)) {
			t.Errorf("outputs[].%s: the manifest offers %v, the catalog %v", field.Key, enumValues(property), choiceValues(field))
		}
	}
}

func TestTheVSCodeManifestHasEveryGeneratorTypeAndField(t *testing.T) {
	properties := loadManifest(t)
	generators, found := properties["loreMaster.generators"]
	if !found || generators.Items == nil {
		t.Fatal("the manifest has no loreMaster.generators")
	}
	description := configcatalog.Describe()

	if got := enumValues(generators.Items.Properties["type"]); !slices.Equal(got, configcatalog.GeneratorIDs()) {
		t.Errorf("the manifest offers the generator types %v, the catalog %v", got, configcatalog.GeneratorIDs())
	}
	fields := description.GeneratorFields
	for _, generator := range description.Generators {
		fields = append(fields, generator.Fields...)
	}
	for _, field := range fields {
		if _, found := generators.Items.Properties[field.Key]; !found {
			t.Errorf("the manifest's generators have no %q", field.Key)
		}
	}
}

func TestTheVSCodeManifestHasEverySetting(t *testing.T) {
	properties := loadManifest(t)
	for _, section := range configcatalog.Describe().Sections {
		for _, field := range section.Settings {
			name := "loreMaster." + field.Key
			property, found := properties[name]
			if !found {
				t.Errorf("the manifest has no setting %s", name)

				continue
			}
			if property.Type != manifestType(field) {
				t.Errorf("%s is %q in the manifest and %q in the catalog", name, property.Type, manifestType(field))
			}
			if field.Type == configcatalog.FieldChoice && !slices.Equal(enumValues(property), choiceValues(field)) {
				t.Errorf("%s: the manifest offers %v, the catalog %v", name, enumValues(property), choiceValues(field))
			}
			if field.Default != "" && fmt.Sprint(property.Default) != field.Default {
				t.Errorf("%s defaults to %v in the manifest and %q in the catalog", name, property.Default, field.Default)
			}
			if field.Min != nil && (property.Minimum == nil || int(*property.Minimum) != *field.Min) {
				t.Errorf("%s: the manifest's minimum is %v, the catalog's %d", name, property.Minimum, *field.Min)
			}
			if field.Max != nil && (property.Maximum == nil || int(*property.Maximum) != *field.Max) {
				t.Errorf("%s: the manifest's maximum is %v, the catalog's %d", name, property.Maximum, *field.Max)
			}
		}
	}
}
