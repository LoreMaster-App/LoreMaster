package rpcprotocol

// MethodPagesBuild reads the workspace and its .lore-master.yaml, generates the static site
// for one github-pages output and writes it into a local folder. It involves no git and no
// network: a pipeline that deploys the folder itself (a Pages artifact, an object store)
// uses this instead of MethodPagesPublish.
const MethodPagesBuild = "pages/build"

// PagesBuildParams asks to build one github-pages output into a folder.
type PagesBuildParams struct {
	// WorkspaceRoot is the folder holding .lore-master.yaml, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
	// Output is the index of the github-pages output in the settings' outputs list.
	Output int `json:"output"`
	// OutDir is the folder to write the site into, as an absolute path. It is created when
	// missing, replaced when an earlier build wrote it, and refused when it holds other files.
	OutDir string `json:"outDir"`
}

// PagesBuildResult reports what the build did. With Errors nothing was written; fix them and
// build again.
type PagesBuildResult struct {
	// OutDir is the folder the site was written into.
	OutDir string `json:"outDir,omitempty"`
	// Files is how many site files were written.
	Files    int      `json:"files"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}
