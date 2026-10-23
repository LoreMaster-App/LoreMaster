package rpcprotocol

// MethodDiagramCheck reports what is wrong with the source of some diagrams, without drawing them.
// It needs no session. D2 is checked by the engine itself, on every call; Mermaid is checked in the
// engine's own browser when the machine has one, and the answer says so when it has not.
const MethodDiagramCheck = "diagram/check"

// DiagramCheckParams are the diagrams to check, in the order the answers come back.
type DiagramCheckParams struct {
	Diagrams []DiagramCheckItem `json:"diagrams"`
}

// DiagramCheckItem is one diagram.
type DiagramCheckItem struct {
	// Language is the fence language: "d2" or "mermaid"; anything else has no problems.
	Language string `json:"language"`
	Source   string `json:"source"`
}

// DiagramCheckResult has one entry per diagram asked about.
type DiagramCheckResult struct {
	Results []DiagramCheckOutcome `json:"results"`
}

// DiagramCheckOutcome is what is wrong with one diagram.
type DiagramCheckOutcome struct {
	Problems []DiagramProblem `json:"problems"`
	// Unavailable says why the diagram could not be checked at all (no browser for Mermaid);
	// empty when it was checked.
	Unavailable string `json:"unavailable,omitempty"`
}

// DiagramProblem is one thing wrong: Line and Column are 1-based within the diagram's source, 0 when unknown.
type DiagramProblem struct {
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
}
