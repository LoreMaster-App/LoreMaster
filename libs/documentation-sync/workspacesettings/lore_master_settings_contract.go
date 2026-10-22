package workspacesettings

// CurrentVersion is the format version this code reads and writes.
const CurrentVersion = 1

// DefaultMermaidMaxWidth is how wide, in pixels, an inline diagram is shown unless an output says otherwise.
const DefaultMermaidMaxWidth = 800

// Settings is the whole file.
type Settings struct {
	Version int `json:"version"`
	// SkipGitignored leaves out Markdown that the workspace's .gitignore files ignore.
	// Nil means the default, true; see DiscoveryScope.
	SkipGitignored *bool `json:"skipGitignored,omitempty"`
	// Ignore is gitignore-syntax patterns, matched against workspace-relative paths, that
	// every output leaves out of the scan on top of its content[].excludes.
	Ignore []string `json:"ignore,omitempty"`
	// Generators write Markdown into the workspace from other project artifacts (test
	// reports, source documentation) so it syncs like any other page.
	Generators []Generator `json:"generators,omitempty"`
	Outputs    []Output    `json:"outputs"`
}

// Generator is one generator: what it reads, and the folder it writes its pages to.
type Generator struct {
	// Type is test-results (JUnit XML), go-docs (Go package documentation), openapi-docs
	// (OpenAPI 3 descriptions) or ts-docs (TypeScript/JavaScript, via TypeDoc); the other types the
	// format names are reserved.
	Type string `json:"type"`
	// Input is gitignore-style patterns selecting what to read; empty means the type's own
	// default.
	Input []string `json:"input,omitempty"`
	// Source is where a generator reads from when that is not a set of files: a base URL or a
	// workspace-relative folder (storybook-docs: a Storybook's build).
	Source string `json:"source,omitempty"`
	// Output is the workspace-relative folder the pages are written to. Include it in an
	// output's content roots for the pages to sync.
	Output string `json:"output"`
	// Title heads the generated index page; empty means the type's own default.
	Title string `json:"title,omitempty"`
	// Prepare is an optional shell command that produces what the generator reads (a build, a
	// test run). It runs only when a run asks for it, never on its own.
	Prepare string `json:"prepare,omitempty"`
}

// Output is one place the lore goes.
type Output struct {
	// Name tells this output from the others (two Confluences, a site and a wiki): it is what
	// the tree, the sync and the form call it. Empty is allowed and shown as a name made from
	// the platform and where it goes; when set, it is unique among the outputs.
	Name         string    `json:"name,omitempty"`
	Platform     string    `json:"platform"`
	BaseURL      string    `json:"baseUrl"`
	Space        string    `json:"space"`
	ParentPageID string    `json:"parentPageId"`
	TitlePrefix  string    `json:"titlePrefix"`
	Direction    string    `json:"direction"`
	Content      []Content `json:"content"`
	// MermaidMode is image | code | html-macro (marketplace-macro: #40).
	MermaidMode string `json:"mermaidMode"`
	// MermaidMaxWidth is the widest, in pixels, a diagram is shown in html-macro mode.
	MermaidMaxWidth int `json:"mermaidMaxWidth,omitempty"`
	// TitleCollision is what to do when an unannotated file's title already exists on
	// the platform: fail | adopt.
	TitleCollision string `json:"titleCollision"`
	// LinkMode is title | id.
	LinkMode string `json:"linkMode"`
	// Repo is the GitHub repository for a github-pages output ("owner/name" or a clone
	// URL); empty means the workspace's own origin remote.
	Repo string `json:"repo,omitempty"`
	// Branch is the branch a github-pages output publishes to; empty means gh-pages.
	Branch string `json:"branch,omitempty"`
	// Path is the folder inside the branch the site is published to, relative to the branch
	// root; empty means the root. Only that folder is replaced, so another site can share the
	// branch.
	Path string `json:"path,omitempty"`
	// Include is gitignore-syntax patterns (files, or folders ending in "/") this output reads
	// even though the top-level ignore list, its Exclude or a content entry's excludes leave
	// them out. The most specific entry wins; on a tie the include does. Git-ignored files
	// stay out, and Include never reaches outside the content roots.
	Include []string `json:"include,omitempty"`
	// Exclude is gitignore-syntax patterns (files, or folders ending in "/") this output
	// leaves out, on top of the top-level ignore list and the content entries' excludes.
	Exclude []string `json:"exclude,omitempty"`
	// Template is how this output's pages are laid out: a built-in's name, a path in the
	// workspace or (confluence) a page id or address. Empty is the default layout.
	Template string `json:"template,omitempty"`
}

// Content is one source feeding an output.
type Content struct {
	Type     string   `json:"type"`
	Roots    []string `json:"roots"`
	Excludes []string `json:"excludes,omitempty"`
	Template string   `json:"template"`
}

// Loaded is the configuration as read from the editor's settings.
type Loaded struct {
	Settings Settings
	// FirstSync is true when nothing is set up yet, or an output still lacks the
	// first-sync answers (space, parent page, title prefix).
	FirstSync bool
	// Exists is false when nothing was configured and Settings are the defaults.
	Exists bool
}

// defaultOutput is a fresh workspace's output: Markdown from the whole workspace, one
// way, to Confluence, with the first-sync answers left for the editor to ask.
func defaultOutput() Output {
	return Output{
		Platform: "confluence", Direction: "to-platform",
		Content:     []Content{{Type: "markdown", Roots: []string{"."}, Template: "default"}},
		MermaidMode: "image", MermaidMaxWidth: DefaultMermaidMaxWidth, TitleCollision: "fail", LinkMode: "title",
	}
}
