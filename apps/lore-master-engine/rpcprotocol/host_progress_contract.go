package rpcprotocol

// MethodHostProgress is a notification from the engine to the editor while a sync runs.
const MethodHostProgress = "host/progress"

// ProgressParams is one step.
type ProgressParams struct {
	// PlanID is the plan being executed.
	PlanID  string `json:"planId"`
	Message string `json:"message"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
}
