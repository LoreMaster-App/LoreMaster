package editorsettings

import (
	"encoding/json"

	"lore-master/libs/documentation-sync/workspacesettings"
)

// Layer is the LoreMaster settings of one settings.json: its path, for error messages,
// and each `loreMaster.*` key with its raw JSON value.
type Layer struct {
	Source string
	Values map[string]json.RawMessage
}

// Result is the configuration the layers add up to.
type Result struct {
	// Configured is false when no layer sets loreMaster.outputs: the editor's settings hold
	// no configuration, and a caller falls back to .lore-master.yaml.
	Configured bool
	// FirstSync is true while an output still lacks its first-sync answers.
	FirstSync bool
	Settings  workspacesettings.Settings
	// Sources lists the settings files that held LoreMaster keys, least specific first.
	Sources []string
}
