package configcatalog_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/configcatalog"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// yamlKeys are the settings an output (or generator) struct accepts, by their file names.
func yamlKeys(t *testing.T, value any) []string {
	t.Helper()
	var keys []string
	shape := reflect.TypeOf(value)
	for i := 0; i < shape.NumField(); i++ {
		name, _, _ := strings.Cut(shape.Field(i).Tag.Get("yaml"), ",")
		if name != "" {
			keys = append(keys, name)
		}
	}

	return keys
}

func keysOf(fields []configcatalog.Field) []string {
	keys := make([]string, len(fields))
	for i, field := range fields {
		keys[i] = field.Key
	}

	return keys
}

func TestEveryPlatformFieldIsASettingOfAnOutput(t *testing.T) {
	known := yamlKeys(t, workspacesettings.Output{})
	description := configcatalog.Describe()
	for _, key := range keysOf(description.OutputFields) {
		if !slices.Contains(known, key) {
			t.Errorf("output field %q is not a setting of an output", key)
		}
	}
	for _, platform := range description.Platforms {
		for _, key := range keysOf(platform.Fields) {
			if !slices.Contains(known, key) {
				t.Errorf("%s field %q is not a setting of an output", platform.ID, key)
			}
		}
	}
}

func TestEverySettingOfAnOutputIsDescribed(t *testing.T) {
	description := configcatalog.Describe()
	described := keysOf(description.OutputFields)
	for _, platform := range description.Platforms {
		described = append(described, keysOf(platform.Fields)...)
	}
	// platform picks the descriptor and content is the roots the shells do not edit as a storage.
	for _, key := range yamlKeys(t, workspacesettings.Output{}) {
		if key != "platform" && key != "content" && !slices.Contains(described, key) {
			t.Errorf("the output setting %q is described by no platform; add it to a descriptor so editors can show it", key)
		}
	}
}

func TestEveryGeneratorFieldIsASettingOfAGenerator(t *testing.T) {
	known := yamlKeys(t, workspacesettings.Generator{})
	description := configcatalog.Describe()
	fields := description.GeneratorFields
	for _, generator := range description.Generators {
		fields = append(fields, generator.Fields...)
	}
	for _, key := range keysOf(fields) {
		if !slices.Contains(known, key) {
			t.Errorf("generator field %q is not a setting of a generator", key)
		}
	}
	for _, key := range known {
		if !slices.Contains(keysOf(fields), key) {
			t.Errorf("the generator setting %q is described nowhere", key)
		}
	}
}

func TestTheGeneratorTypeFieldOffersEveryType(t *testing.T) {
	for _, field := range configcatalog.Describe().GeneratorFields {
		if field.Key != "type" {
			continue
		}
		var offered []string
		for _, choice := range field.Choices {
			offered = append(offered, choice.Value)
		}
		if !slices.Equal(offered, configcatalog.GeneratorIDs()) {
			t.Fatalf("the type field offers %v, not %v", offered, configcatalog.GeneratorIDs())
		}
	}
}

func TestFieldsAreConsistent(t *testing.T) {
	description := configcatalog.Describe()
	all := slices.Clone(description.OutputFields)
	all = append(all, description.GeneratorFields...)
	for _, platform := range description.Platforms {
		all = append(all, platform.Fields...)
		keys := append(keysOf(platform.Fields), keysOf(description.OutputFields)...)
		if platform.Connection != nil && !slices.Contains(keys, platform.Connection.Field) {
			t.Errorf("%s: the connection field %q is not one of its fields", platform.ID, platform.Connection.Field)
		}
		for _, field := range platform.Fields {
			if field.Lookup == nil {
				continue
			}
			for _, needed := range field.Lookup.Needs {
				if !slices.Contains(keys, needed) {
					t.Errorf("%s.%s: the lookup needs %q, which is not a field", platform.ID, field.Key, needed)
				}
			}
			for _, fill := range field.Lookup.Fills {
				if !slices.Contains(keys, fill.Field) {
					t.Errorf("%s.%s: the lookup fills %q, which is not a field", platform.ID, field.Key, fill.Field)
				}
			}
		}
	}
	for _, section := range description.Sections {
		all = append(all, section.Settings...)
	}
	for _, field := range all {
		if field.Label == "" || field.Type == "" {
			t.Errorf("field %q needs a label and a type", field.Key)
		}
		if field.Type == configcatalog.FieldChoice && len(field.Choices) == 0 {
			t.Errorf("choice field %q has no choices", field.Key)
		}
		if field.Default != "" && field.Type == configcatalog.FieldChoice {
			found := false
			for _, choice := range field.Choices {
				found = found || choice.Value == field.Default
			}
			if !found {
				t.Errorf("field %q defaults to %q, which is not one of its choices", field.Key, field.Default)
			}
		}
	}
}

func TestPlatformsAreTheOnesTheValidatorAccepts(t *testing.T) {
	for _, id := range configcatalog.PlatformIDs() {
		if _, found := configcatalog.FindPlatform(id); !found {
			t.Errorf("%s is listed but cannot be found", id)
		}
	}
	if !slices.Equal(workspacesettings.BuiltGeneratorTypes, configcatalog.GeneratorIDs()) {
		t.Errorf("the validator accepts %v, the catalog lists %v", workspacesettings.BuiltGeneratorTypes, configcatalog.GeneratorIDs())
	}
}

func TestMakeUpName(t *testing.T) {
	confluence, _ := configcatalog.FindPlatform("confluence")
	pages, _ := configcatalog.FindPlatform("github-pages")
	values := func(set map[string]string) func(string) string { return func(key string) string { return set[key] } }

	if got := confluence.MakeUpName(values(map[string]string{"space": "ENG"})); got != "Confluence · ENG" {
		t.Errorf("got %q", got)
	}
	if got := confluence.MakeUpName(values(nil)); got != "Confluence · no space yet" {
		t.Errorf("got %q", got)
	}
	if got := pages.MakeUpName(values(nil)); got != "GitHub Pages · this repository" {
		t.Errorf("got %q", got)
	}
}

func TestDescribeReturnsValuesACallerMayChange(t *testing.T) {
	first := configcatalog.Describe()
	first.Platforms[0].Label = "changed"
	first.Sections[0].Settings[0].Label = "changed"

	second := configcatalog.Describe()
	if second.Platforms[0].Label == "changed" || second.Sections[0].Settings[0].Label == "changed" {
		t.Fatal("a change to one description leaked into the next")
	}
}
