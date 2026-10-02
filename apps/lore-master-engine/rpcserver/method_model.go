package rpcserver

import (
	"context"
	"encoding/json"
)

// Method handles one JSON-RPC method. It returns the result to send, or an error:
// an *rpcprotocol.Error keeps its code, a cancelled context becomes "request
// cancelled", and anything else is an internal error carrying its message.
type Method func(ctx context.Context, call Call) (any, error)

// Methods maps method names to their handlers.
type Methods map[string]Method

// Call is one incoming request.
type Call struct {
	// Params is the raw params, null when absent.
	Params json.RawMessage
	// Editor reaches back to the editor on the same connection.
	Editor Peer
}

// Peer is the other end of the connection: the engine asks the editor through it.
type Peer interface {
	// Call sends a request and waits for its result, or for ctx.
	Call(ctx context.Context, method string, params any, result any) error
	// Notify sends a notification.
	Notify(ctx context.Context, method string, params any) error
}

// Decode unmarshals the call's params into v. A malformed value is an invalid-params
// error naming what was wrong.
func (c Call) Decode(v any) error {
	if len(c.Params) == 0 || string(c.Params) == "null" {
		return nil
	}
	if err := json.Unmarshal(c.Params, v); err != nil {
		return invalidParams(err)
	}

	return nil
}
