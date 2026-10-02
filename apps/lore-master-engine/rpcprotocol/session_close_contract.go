package rpcprotocol

// MethodSessionClose forgets a session and its credential. Closing an unknown session
// is not an error. Result: null.
const MethodSessionClose = "session/close"

// SessionCloseParams names the session.
type SessionCloseParams struct {
	SessionID string `json:"sessionId"`
}
