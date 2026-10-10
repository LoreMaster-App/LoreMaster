package sitepublish

// PublishOptions says where and how to publish a generated site.
type PublishOptions struct {
	// WorkspaceRoot is the local git repository the site was generated from; its origin
	// remote is used when Repo is empty.
	WorkspaceRoot string
	// Repo overrides the repository: "owner/name" or a clone URL. Empty means the
	// workspace's own origin remote.
	Repo string
	// Branch is the branch to publish to; empty means gh-pages.
	Branch string
	// Path is the folder inside the branch to publish into, relative to the branch root and
	// '/'-separated; empty means the root. Only that folder is replaced.
	Path string
	// Wiki publishes to the repository's wiki (<repo>.wiki.git, branch master by default)
	// instead of a site branch, and lays the pages over what is there rather than replacing
	// the whole folder, so pages written on GitHub survive.
	Wiki bool
	// CommitMessage is the publish commit's message; empty means a default.
	CommitMessage string
}

// PublishResult reports what a publish did.
type PublishResult struct {
	// Branch and Remote are where the site was published.
	Branch string
	Remote string
	// Commit is the new commit's short hash, or empty when nothing changed.
	Commit string
	// Changed is false when the site already matched the branch (a no-op publish).
	Changed bool
	// Files is how many site files were written.
	Files int
}
