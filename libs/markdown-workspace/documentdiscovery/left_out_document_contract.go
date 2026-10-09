package documentdiscovery

// LeftOutRule names why a Markdown file is not part of the scan.
type LeftOutRule string

const (
	// RuleIgnore is a pattern of the settings' top-level ignore list.
	RuleIgnore LeftOutRule = "ignore"
	// RuleExcludes is a pattern of a content entry's excludes.
	RuleExcludes LeftOutRule = "excludes"
	// RuleGitignore is a .gitignore that the sync honours (skipGitignored).
	RuleGitignore LeftOutRule = "gitignore"
	// RuleOutsideRoots means no content entry's roots contain the file.
	RuleOutsideRoots LeftOutRule = "outside-roots"
)

// LeftOut is a Markdown file the scan does not read, and the first rule that says so.
type LeftOut struct {
	Path DocumentPath `json:"path"`
	Rule LeftOutRule  `json:"rule"`
	// Pattern is the line of the ignore list, excludes or .gitignore that matched; empty for
	// outside-roots.
	Pattern string `json:"pattern,omitempty"`
	// Source is the .gitignore the pattern is in, workspace-relative; empty for the
	// settings' own lists.
	Source string `json:"source,omitempty"`
}

// LeftOutOptions says what the scan was given; it mirrors Options, with the settings' ignore
// list and the content excludes apart so each can be named.
type LeftOutOptions struct {
	WorkspaceRoot string
	// Roots are the roots of every content entry of the output.
	Roots []string
	// Ignore is the settings' top-level ignore list; Excludes is every content entry's.
	Ignore   []string
	Excludes []string
	// IncludeGitignored mirrors Options.IncludeGitignored.
	IncludeGitignored bool
	// Included are the files the scan did read; they are never reported.
	Included []DocumentPath
	// Limit caps how many are returned (the total is always counted); zero means 500.
	Limit int
}

// LeftOutReport is the files the scan skipped, sorted by path.
type LeftOutReport struct {
	Documents []LeftOut `json:"documents"`
	// Total is how many there are, which may exceed len(Documents) when Limit applied.
	Total int `json:"total"`
}
