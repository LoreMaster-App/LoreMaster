package rpcprotocol

// MethodSettingsRead reads .lore-master.yaml, with defaults applied: what the editor's
// first-sync wizard starts from. A file that is not valid settings is an
// invalid-settings error naming each problem.
const MethodSettingsRead = "settings/read"

// MethodSettingsSave validates and saves .lore-master.yaml, keeping the author's
// comments and key order. Result: null.
const MethodSettingsSave = "settings/save"

// MethodSettingsMigrate imports an existing .lore-master.yaml into the folder's
// .vscode/settings.json (#308), unless the editor's settings already hold the
// configuration. The yaml is left in place. Result: SettingsMigrateResult.
const MethodSettingsMigrate = "settings/migrate"

// SettingsMigrateResult says whether anything was written.
type SettingsMigrateResult struct {
	// Migrated is true when the yaml was imported on this call.
	Migrated bool `json:"migrated"`
	// Path is the settings file written; empty when nothing was.
	Path string `json:"path,omitempty"`
}

// SettingsReadParams names the workspace.
type SettingsReadParams struct {
	// WorkspaceRoot is the workspace folder, as an absolute path.
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
	Ignore []string `json:"ignore,omitempty"`
	// Generators write Markdown into the workspace from test reports and other artifacts.
	Generators []Generator `json:"generators,omitempty"`
	Outputs    []Output    `json:"outputs"`
}

// Generator is one generator of .lore-master.yaml.
type Generator struct {
	Type   string   `json:"type"`
	Input  []string `json:"input,omitempty"`
	// Source is where a generator reads from when that is not a set of files: a base URL or a
	// workspace-relative folder.
	Source string `json:"source,omitempty"`
	Output string   `json:"output"`
	Title  string   `json:"title,omitempty"`
	// Prepare is the shell command that produces what the generator reads; it runs only when a
	// generators/run asks for it.
	Prepare string `json:"prepare,omitempty"`
}

// Output is one place the lore goes.
type Output struct {
	// Name is what the editor calls this output; empty means a name made from where it goes.
	Name         string    `json:"name,omitempty"`
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
	// Repo is the GitHub repository of a github-pages output ("owner/name" or a clone URL);
	// empty means the workspace's own origin remote.
	Repo string `json:"repo,omitempty"`
	// Branch is the branch a github-pages output publishes to; empty means gh-pages.
	Branch string `json:"branch,omitempty"`
	// Path is the folder inside the branch the site is published to; empty means the root.
	Path string `json:"path,omitempty"`
	// Include is gitignore-syntax patterns this output reads even though the ignore list or an
	// exclude leaves them out; the most specific entry wins.
	Include []string `json:"include,omitempty"`
	// Exclude is gitignore-syntax patterns this output leaves out, on top of the ignore list.
	Exclude []string `json:"exclude,omitempty"`
	// Template is the layout of the output's pages: a built-in's name, a workspace path or, for
	// Confluence, a page id or address; empty is the default.
	Template string `json:"template,omitempty"`
}

// Content is one kind of lore an output syncs.
type Content struct {
	Type     string   `json:"type"`
	Roots    []string `json:"roots"`
	Excludes []string `json:"excludes,omitempty"`
	Template string   `json:"template"`
}
