package rpcprotocol

// MethodGeneratorsRun runs the generators of .lore-master.yaml: each reads project artifacts
// (JUnit test reports today) and writes ordinary Markdown pages into its output folder, which
// then sync like any other page. A generator that fails does not stop the others; its
// error comes back in its own entry.
const MethodGeneratorsRun = "generators/run"

// GeneratorsRunParams names the workspace and, optionally, which generators to run.
type GeneratorsRunParams struct {
	// WorkspaceRoot is the folder holding .lore-master.yaml, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
	// Generators are indexes into the settings' generators list; empty runs them all.
	Generators []int `json:"generators,omitempty"`
}

// GeneratorsRunResult has one entry per generator that ran, in index order.
type GeneratorsRunResult struct {
	Runs []GeneratorRun `json:"runs"`
}

// GeneratorRun is what one generator did. The paths are workspace-relative and '/'-separated.
type GeneratorRun struct {
	// Index is the generator's position in the settings' generators list.
	Index  int    `json:"index"`
	Type   string `json:"type"`
	Output string `json:"output"`
	// Written are pages created or rewritten; Unchanged were already up to date; Removed are
	// generated pages the generator no longer produces.
	Written   []string `json:"written,omitempty"`
	Unchanged []string `json:"unchanged,omitempty"`
	Removed   []string `json:"removed,omitempty"`
	// Warnings are inputs skipped and files left alone because they were not generated.
	Warnings []string `json:"warnings,omitempty"`
	// Error is why the generator could not run at all; the lists above are then empty.
	Error string `json:"error,omitempty"`
}
