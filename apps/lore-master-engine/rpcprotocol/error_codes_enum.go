package rpcprotocol

import "fmt"

// Error codes beyond JSON-RPC's own (-32700 parse error, -32600 invalid request,
// -32601 method not found, -32602 invalid params, -32603 internal error). The
// application range is -32000 to -32099; request cancelled is the Language Server
// Protocol's code, which vscode-jsonrpc already understands.
const (
	CodeInvalidParams    = -32602
	CodeInternalError    = -32603
	CodeRequestCancelled = -32800

	// CodeUnauthorized: the credential was refused.
	CodeUnauthorized = -32001
	// CodeUnknownSession: no open session has that id (closed, or the engine restarted).
	CodeUnknownSession = -32002
	// CodeUnknownPlan: no plan has that id (already executed, or the engine restarted).
	CodeUnknownPlan = -32003
	// CodeInvalidSettings: .lore-master.yaml is missing a value or holds a wrong one.
	CodeInvalidSettings = -32004
	// CodePlanHasErrors: the plan has errors and cannot be executed.
	CodePlanHasErrors = -32005
	// CodeNotFound: a page or space does not exist or is not visible to the user.
	CodeNotFound = -32006
	// CodeTitleTaken: another page already has the title.
	CodeTitleTaken = -32007
	// CodeVersionConflict: the page changed on the platform meanwhile.
	CodeVersionConflict = -32008
	// CodePlatformUnreachable: the platform did not answer, or not as expected.
	CodePlatformUnreachable = -32009
	// CodeUnsupported: the platform or edition cannot do what was asked.
	CodeUnsupported = -32010
)

// Error is an error with a code for the editor. Handlers return it; the server sends it
// as the JSON-RPC error. Message is for people and never holds a secret.
type Error struct {
	Code    int64
	Message string
}

// Errorf makes an Error.
func Errorf(code int64, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func (e *Error) Error() string { return e.Message }
