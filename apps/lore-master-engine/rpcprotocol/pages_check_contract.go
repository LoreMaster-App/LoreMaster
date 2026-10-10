package rpcprotocol

// MethodPagesCheck generates the static site for one github-pages output and compares it with
// what is published, without publishing: it reports the files a publish would add, modify or
// remove. It is the GitHub Pages counterpart of a read-only sync plan, and like a publish it
// needs no session, only the user's git.
const MethodPagesCheck = "pages/check"

// PagesCheckParams asks to check one github-pages output.
type PagesCheckParams struct {
	// WorkspaceRoot is the folder holding .lore-master.yaml, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
	// Output is the index of the github-pages output in the settings' outputs list.
	Output int `json:"output"`
}

// Change kinds of a PagesChange.
const (
	PagesChangeAdded    = "added"
	PagesChangeModified = "modified"
	PagesChangeRemoved  = "removed"
)

// PagesChange is one file a publish would touch.
type PagesChange struct {
	Path string `json:"path"`
	// Kind is added, modified or removed.
	Kind string `json:"kind"`
}

// PagesCheckResult says whether the published site is up to date. With Errors nothing was
// compared: the Markdown has problems a publish would stop on.
type PagesCheckResult struct {
	// Branch and Remote are what the site was compared with.
	Branch string `json:"branch,omitempty"`
	Remote string `json:"remote,omitempty"`
	// UpToDate is true when a publish would change nothing.
	UpToDate bool `json:"upToDate"`
	// Changes lists the files, capped; ChangesTotal counts them all.
	Changes      []PagesChange `json:"changes,omitempty"`
	ChangesTotal int           `json:"changesTotal"`
	// Files is how many site files were generated.
	Files    int      `json:"files"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}
