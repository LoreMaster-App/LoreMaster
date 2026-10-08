package rpcprotocol

// MethodAgentInstructions composes the instructions for an AI agent that writes this
// workspace's documentation: the nesting and title rules the sync applies, quoted from the
// code that applies them, then what is particular to the workspace and how to work with the
// LoreMaster tools. The editor wraps them in each tool's agent file.
const MethodAgentInstructions = "agent/instructions"

// AgentInstructionsParams names the workspace.
type AgentInstructionsParams struct {
	// WorkspaceRoot is the folder holding .lore-master.yaml, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
}

// AgentInstructionsResult is the instructions.
type AgentInstructionsResult struct {
	// Instructions is Markdown without a top-level heading, so each tool's file can frame it.
	Instructions string `json:"instructions"`
	// HasConfig is false when the workspace has no .lore-master.yaml yet.
	HasConfig bool `json:"hasConfig"`
}
