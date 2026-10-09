package sitedeployment

// Kind is how a repository deploys to GitHub Pages.
type Kind string

const (
	// KindActionsSource deploys an artifact with actions/deploy-pages; each deploy replaces
	// the whole site and there is no branch to push to.
	KindActionsSource Kind = "actions-source"
	// KindBranchPush writes a branch (usually gh-pages) that Pages serves.
	KindBranchPush Kind = "branch-push"
	// KindNone means no Pages deploy was found in the workflows.
	KindNone Kind = "none"
)

// Deployment is what the workflow files say about the repository's Pages deploy.
type Deployment struct {
	Kind Kind `json:"kind"`
	// Workflows are the files that carry a Pages deploy.
	Workflows []string `json:"workflows,omitempty"`
	// ArtifactPath is the folder an Actions-source deploy uploads (its upload-pages-artifact
	// path), empty when it could not be read.
	ArtifactPath string `json:"artifactPath,omitempty"`
	// Branch is the branch a branch-push deploy writes, empty when it could not be read.
	Branch string `json:"branch,omitempty"`
	// KeepsOtherFiles is true when a branch-push deploy is configured to leave files it did
	// not write (keep_files: true, clean: false).
	KeepsOtherFiles bool `json:"keepsOtherFiles,omitempty"`
	// UsesLoreMasterAction is true when a workflow already uses the LoreMaster pages action.
	UsesLoreMasterAction bool `json:"usesLoreMasterAction,omitempty"`
	// Notes are facts the detection could not settle, for the caller to show.
	Notes []string `json:"notes,omitempty"`
}

// Output is the part of a github-pages output the recommendation needs.
type Output struct {
	Repo   string
	Branch string
	Path   string
}

// Plan is the recommendation: what to do, the snippets to paste and what to watch for.
type Plan struct {
	Deployment Deployment `json:"deployment"`
	// Approach is build (write the site into a folder the other deploy publishes), publish
	// (push to a branch, into Path) or publish-root (nothing else deploys; own the branch).
	Approach string `json:"approach"`
	// OutDir is the folder to build into, for the build approach.
	OutDir string `json:"outDir,omitempty"`
	// Path is the folder inside the branch, for the publish approach.
	Path string `json:"path,omitempty"`
	// OutputSnippet is the .lore-master.yaml output to have; WorkflowSnippet is the workflow
	// step to add. Either may be empty when it does not apply.
	OutputSnippet   string   `json:"outputSnippet,omitempty"`
	WorkflowSnippet string   `json:"workflowSnippet,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}
