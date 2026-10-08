package workspacesettings

// FileName is the configuration file at the workspace root.
const FileName = ".lore-master.yaml"

// CurrentVersion is the format version this code reads and writes.
const CurrentVersion = 1

// Settings is the whole file.
type Settings struct {
	Version int `yaml:"version"`
	// SkipGitignored leaves out Markdown that the workspace's .gitignore files ignore.
	// Nil means the default, true; see DiscoveryScope.
	SkipGitignored *bool `yaml:"skipGitignored,omitempty"`
	// Ignore is gitignore-syntax patterns, matched against workspace-relative paths, that
	// every output leaves out of the scan on top of its content[].excludes.
	Ignore  []string `yaml:"ignore,omitempty"`
	Outputs []Output   `yaml:"outputs"`
}

// Output is one place the lore goes.
type Output struct {
	Platform     string    `yaml:"platform"`
	BaseURL      string    `yaml:"baseUrl"`
	Space        string    `yaml:"space"`
	ParentPageID string    `yaml:"parentPageId"`
	TitlePrefix  string    `yaml:"titlePrefix"`
	Direction    string    `yaml:"direction"`
	Content      []Content `yaml:"content"`
	// MermaidMode is image | code (html-macro and marketplace-macro: #40).
	MermaidMode string `yaml:"mermaidMode"`
	// TitleCollision is what to do when an unannotated file's title already exists on
	// the platform: fail | adopt.
	TitleCollision string `yaml:"titleCollision"`
	// LinkMode is title | id.
	LinkMode string `yaml:"linkMode"`
	// Repo is the GitHub repository for a github-pages output ("owner/name" or a clone
	// URL); empty means the workspace's own origin remote.
	Repo string `yaml:"repo,omitempty"`
	// Branch is the branch a github-pages output publishes to; empty means gh-pages.
	Branch string `yaml:"branch,omitempty"`
}

// Content is one source feeding an output.
type Content struct {
	Type     string   `yaml:"type"`
	Roots    []string `yaml:"roots"`
	Excludes []string `yaml:"excludes,omitempty"`
	Template string   `yaml:"template"`
}

// Loaded is a settings file as read, with what saving needs to keep the author's
// comments and layout.
type Loaded struct {
	Settings Settings
	// FirstSync is true when the file is missing, or an output still lacks the
	// first-sync answers (space, parent page, title prefix).
	FirstSync bool
	// Exists is false when the file was missing and Settings are the defaults.
	Exists bool
	path   string
	tree   []byte
}

// defaultOutput is a fresh workspace's output: Markdown from the whole workspace, one
// way, to Confluence, with the first-sync answers left for the editor to ask.
func defaultOutput() Output {
	return Output{
		Platform: "confluence", Direction: "to-platform",
		Content:     []Content{{Type: "markdown", Roots: []string{"."}, Template: "default"}},
		MermaidMode: "image", TitleCollision: "fail", LinkMode: "title",
	}
}
