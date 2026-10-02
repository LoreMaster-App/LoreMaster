package documentdiscovery

// Options says where to look. Every path is relative to WorkspaceRoot.
type Options struct {
	// WorkspaceRoot is the directory the editor has open, as an OS path.
	WorkspaceRoot string
	// Roots are the directories to scan, '/'-separated; empty means the whole workspace.
	Roots []string
	// Excludes are gitignore-syntax patterns matched against workspace-relative paths.
	Excludes []string
}

// DefaultExcludedDirectories are never entered, at any depth, whatever the options say:
// they hold dependencies, build output or VCS state, never the project's own lore.
var DefaultExcludedDirectories = []string{"node_modules", ".git", "dist", "out-tsc", "coverage", ".venv"}
