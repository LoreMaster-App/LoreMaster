package rpcprotocol

// MethodSettingsRead reads .lore-master.yaml, with defaults applied: what the editor's
// first-sync wizard starts from. A file that is not valid settings is an
// invalid-settings error naming each problem.
const MethodSettingsRead = "settings/read"

// MethodSettingsSave validates and saves .lore-master.yaml, keeping the author's
// comments and key order. Result: null.
const MethodSettingsSave = "settings/save"

// SettingsReadParams names the workspace.
type SettingsReadParams struct {
	// WorkspaceRoot is the folder holding .lore-master.yaml, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
}

// SettingsReadResult is the file as the engine understands it.
type SettingsReadResult struct {
	// Exists is false when there is no file yet and Settings are the defaults.
	Exists bool `json:"exists"`
	// FirstSync is true while an output lacks its site, space, parent page or prefix.
	FirstSync bool     `json:"firstSync"`
	Settings  Settings `json:"settings"`
}

// SettingsSaveParams is the whole file's content.
type SettingsSaveParams struct {
	WorkspaceRoot string   `json:"workspaceRoot"`
	Settings      Settings `json:"settings"`
}

// Settings mirrors .lore-master.yaml.
type Settings struct {
	Version int `json:"version"`
	// SkipGitignored leaves out Markdown the workspace's .gitignore files ignore; absent means true.
	SkipGitignored *bool `json:"skipGitignored,omitempty"`
	// Ignore is gitignore-syntax patterns every output leaves out of the scan.
	Ignore  []string `json:"ignore,omitempty"`
	Outputs []Output `json:"outputs"`
}

// Output is one place the lore goes.
type Output struct {
	Platform     string    `json:"platform"`
	BaseURL      string    `json:"baseUrl"`
	Space        string    `json:"space"`
	ParentPageID string    `json:"parentPageId"`
	TitlePrefix  string    `json:"titlePrefix"`
	Direction    string    `json:"direction"`
	Content      []Content `json:"content"`
	// MermaidMode is image or code.
	MermaidMode string `json:"mermaidMode"`
	// TitleCollision is fail or adopt.
	TitleCollision string `json:"titleCollision"`
	// LinkMode is title or id.
	LinkMode string `json:"linkMode"`
}

// Content is one kind of lore an output syncs.
type Content struct {
	Type     string   `json:"type"`
	Roots    []string `json:"roots"`
	Excludes []string `json:"excludes,omitempty"`
	Template string   `json:"template"`
}
