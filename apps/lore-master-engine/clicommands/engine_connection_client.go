package clicommands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// connection is a client of the engine's own methods, over an in-process pipe.
type connection struct {
	conn *jsonrpc2.Conn
	stop func()
}

// connect serves methods on one end of a pipe and returns a client on the other. The engine's
// requests to its host are answered here: a progress notification prints a line to progress,
// and a request to draw a diagram is refused, so a Mermaid diagram stays a code block (with
// the sync's usual warning) instead of the command line needing a browser.
func connect(ctx context.Context, methods rpcserver.Methods, logger *slog.Logger, progress io.Writer) *connection {
	serverEnd, clientEnd := net.Pipe()
	serving, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = rpcserver.Serve(serving, serverEnd, methods, logger)
	}()

	handler := jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(func(_ context.Context, _ *jsonrpc2.Conn, request *jsonrpc2.Request) (any, error) {
		switch request.Method {
		case rpcprotocol.MethodHostProgress:
			printProgress(progress, request.Params)

			return nil, nil
		case rpcprotocol.MethodHostRenderDiagram:
			return nil, &jsonrpc2.Error{Code: rpcprotocol.CodeInternalError, Message: "the command line draws no diagrams; Mermaid blocks stay code"}
		default:
			return nil, nil
		}
	}))
	conn := jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(clientEnd, jsonrpc2.VSCodeObjectCodec{}), handler)

	return &connection{conn: conn, stop: func() {
		_ = conn.Close()
		cancel()
		<-done
	}}
}

func printProgress(out io.Writer, params *json.RawMessage) {
	if out == nil || params == nil {
		return
	}
	var step rpcprotocol.ProgressParams
	if json.Unmarshal(*params, &step) == nil {
		_, _ = fmt.Fprintf(out, "[%d/%d] %s\n", step.Done, step.Total, step.Message)
	}
}

// call asks the engine and decodes the result into result (which may be nil). An engine error
// comes back as its message, without the JSON-RPC wrapping.
func (c *connection) call(ctx context.Context, method string, params any, result any) error {
	err := c.conn.Call(ctx, method, params, result)
	var wire *jsonrpc2.Error
	if errors.As(err, &wire) {
		return errors.New(wire.Message)
	}

	return err
}

func (c *connection) close() {
	c.stop()
}
