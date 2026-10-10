package sitepublish

// Change kinds a check reports for one file.
const (
	ChangeAdded    = "added"
	ChangeModified = "modified"
	ChangeRemoved  = "removed"
)

// SiteChange is one file a publish would add, modify or remove.
type SiteChange struct {
	// Path is relative to the repository root, '/'-separated.
	Path string
	// Kind is ChangeAdded, ChangeModified or ChangeRemoved.
	Kind string
}

// CheckResult reports how the generated site compares with the published branch.
type CheckResult struct {
	// Branch and Remote are what the site was compared with.
	Branch string
	Remote string
	// UpToDate is true when a publish would change nothing.
	UpToDate bool
	// Changes are the files a publish would touch, sorted by path.
	Changes []SiteChange
	// Files is how many site files were generated.
	Files int
}
