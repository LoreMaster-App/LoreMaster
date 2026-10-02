package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

// shutdownPatience is how long a hang-up waits for running requests to wind down.
const shutdownPatience = 30 * time.Second

// Serve answers JSON-RPC on stream until the other end hangs up (the editor closed
// stdin) or ctx ends. Every request runs in its own goroutine. On the way out every
// running request is cancelled and waited for, so a sync stops between pages and still
// writes back what it did, rather than dying half-way. The log records methods,
// outcomes and durations only, never params or results, which may hold credentials.
func Serve(ctx context.Context, stream io.ReadWriteCloser, methods Methods, logger *slog.Logger) error {
	server := &server{methods: methods, logger: logger, cancellations: newCancellations()}
	conn := jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(stream, jsonrpc2.VSCodeObjectCodec{}), jsonrpc2.AsyncHandler(server))
	var err error
	select {
	case <-conn.DisconnectNotify():
	case <-ctx.Done():
		_ = conn.Close()
		err = ctx.Err()
	}
	if !server.cancellations.stopAll(shutdownPatience) {
		logger.Warn("requests still running at shutdown")
	}

	return err
}

type server struct {
	methods       Methods
	logger        *slog.Logger
	cancellations *cancellations
}

// Handle implements jsonrpc2.Handler.
func (s *server) Handle(ctx context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	if request.Method == rpcprotocol.MethodCancelRequest {
		var params struct {
			ID jsonrpc2.ID `json:"id"`
		}
		if request.Params != nil && json.Unmarshal(*request.Params, &params) == nil {
			s.cancellations.cancel(params.ID)
		}

		return
	}

	method, known := s.methods[request.Method]
	if !known {
		if !request.Notif {
			s.reply(ctx, conn, request, nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("the engine has no method %q", request.Method)})
		}

		return
	}

	ctx, done := s.cancellations.start(ctx, request.ID)
	defer done()
	call := Call{Editor: editor{conn}}
	if request.Params != nil {
		call.Params = *request.Params
	}
	started := time.Now()
	result, err := method(ctx, call)
	if request.Notif {
		return
	}
	var wireError *jsonrpc2.Error
	if err != nil {
		wireError = toWireError(err)
	}
	s.logger.Info("request", "method", request.Method, "ok", err == nil, "code", codeOf(wireError), "ms", time.Since(started).Milliseconds())
	s.reply(ctx, conn, request, result, wireError)
}

func (s *server) reply(ctx context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request, result any, wireError *jsonrpc2.Error) {
	// The reply goes out even when the request's own context was cancelled.
	ctx = context.WithoutCancel(ctx)
	var err error
	if wireError != nil {
		err = conn.ReplyWithError(ctx, request.ID, wireError)
	} else {
		err = conn.Reply(ctx, request.ID, result)
	}
	if err != nil {
		s.logger.Warn("reply not sent", "method", request.Method, "error", err.Error())
	}
}

func codeOf(wireError *jsonrpc2.Error) int64 {
	if wireError == nil {
		return 0
	}

	return wireError.Code
}

// editor is the connection seen as the Peer a method calls back through.
type editor struct{ conn *jsonrpc2.Conn }

func (e editor) Call(ctx context.Context, method string, params any, result any) error {
	return e.conn.Call(ctx, method, params, result)
}

func (e editor) Notify(ctx context.Context, method string, params any) error {
	return e.conn.Notify(ctx, method, params)
}

// Ping answers rpcprotocol.MethodPing with the engine's version.
func Ping(version string) Method {
	return func(context.Context, Call) (any, error) {
		return rpcprotocol.PingResult{Pong: "pong", Version: version}, nil
	}
}
