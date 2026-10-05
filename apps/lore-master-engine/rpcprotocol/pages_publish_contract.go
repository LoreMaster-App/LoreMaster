package rpcprotocol

// MethodPagesPublish reads the workspace and its .lore-master.yaml, generates the static
// site for one github-pages output and publishes it to that output's branch by shelling
// out to the user's git. Unlike the Confluence sync it needs no session: git carries the
// credentials.
const MethodPagesPublish = "pages/publish"

// PagesPublishParams asks to publish one github-pages output.
type PagesPublishParams struct {
	// WorkspaceRoot is the folder holding .lore-master.yaml, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
	// Output is the index of the github-pages output in the settings' outputs list.
	Output int `json:"output"`
}

// PagesPublishResult reports what the publish did. With Errors nothing was published; fix
// them and publish again.
type PagesPublishResult struct {
	// Branch and Remote are where the site was published.
	Branch string `json:"branch,omitempty"`
	Remote string `json:"remote,omitempty"`
	// Commit is the new commit's short hash, or empty when nothing changed.
	Commit string `json:"commit,omitempty"`
	// Changed is false when the site already matched the branch (a no-op publish).
	Changed bool `json:"changed"`
	// Files is how many site files were written.
	Files int `json:"files"`
	// URL is the published site's address when it can be derived from a github.com remote.
	URL      string   `json:"url,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}
