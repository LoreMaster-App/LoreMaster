package rpcserver

import (
	"context"
	"errors"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

// toWireError maps a handler's error onto the JSON-RPC error the editor receives.
func toWireError(err error) *jsonrpc2.Error {
	var coded *rpcprotocol.Error
	switch {
	case errors.As(err, &coded):
		return &jsonrpc2.Error{Code: coded.Code, Message: coded.Message}
	case errors.Is(err, context.Canceled):
		return &jsonrpc2.Error{Code: rpcprotocol.CodeRequestCancelled, Message: "the request was cancelled"}
	}

	return &jsonrpc2.Error{Code: rpcprotocol.CodeInternalError, Message: err.Error()}
}

func invalidParams(err error) error {
	return rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "invalid params: %v", err)
}
