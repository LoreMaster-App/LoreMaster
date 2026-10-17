package settingsfixture

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// SettingsJSON turns a configuration written as YAML (version, outputs, generators, ignore, skipGitignored) into the content of a
// settings.json: each key becomes the loreMaster.* setting of that name. The version is dropped, as the editor has none.
// A configuration that lists no outputs gets an empty list, so it still reads as configured.
func SettingsJSON(configuration string) (string, error) {
	var tree map[string]any
	if err := yaml.Unmarshal([]byte(configuration), &tree); err != nil {
		return "", err
	}
	settings := map[string]any{}
	for key, value := range tree {
		if key != "version" {
			settings["loreMaster."+key] = value
		}
	}
	// A configuration with no outputs is still a configuration (the user has set up only generators, say), as an empty list.
	if _, has := settings["loreMaster.outputs"]; !has {
		settings["loreMaster.outputs"] = []any{}
	}
	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}

	return string(content) + "\n", nil
}
