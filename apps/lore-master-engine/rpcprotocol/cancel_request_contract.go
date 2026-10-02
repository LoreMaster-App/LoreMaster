package rpcprotocol

// MethodCancelRequest is a notification asking the engine to stop the request with the
// given id. The request then fails with CodeRequestCancelled, unless it already ended.
const MethodCancelRequest = "$/cancelRequest"

// CancelParams names the request to stop. ID is a number or a string, as sent.
type CancelParams struct {
	ID any `json:"id"`
}
