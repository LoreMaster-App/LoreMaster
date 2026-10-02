package rpcserver

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

// editorSide is the test playing the editor: it answers host/echo requests.
type editorSide struct{}

func (editorSide) Handle(ctx context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	if request.Method == "host/echo" {
		_ = conn.Reply(ctx, request.ID, map[string]string{"echo": string(*request.Params)})
	}
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// connect serves methods on one end of a pipe and returns the editor's end.
func connect(t *testing.T, methods Methods) (*jsonrpc2.Conn, *syncBuffer, chan error) {
	t.Helper()
	engineEnd, editorEnd := net.Pipe()
	log := &syncBuffer{}
	served := make(chan error, 1)
	go func() {
		served <- Serve(context.Background(), engineEnd, methods, slog.New(slog.NewTextHandler(log, nil)))
	}()
	client := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}), jsonrpc2.AsyncHandler(editorSide{}))
	t.Cleanup(func() { _ = client.Close() })

	return client, log, served
}

func code(err error) int64 {
	var wire *jsonrpc2.Error
	if errors.As(err, &wire) {
		return wire.Code
	}

	return 0
}

func TestPing(t *testing.T) {
	client, _, _ := connect(t, Methods{rpcprotocol.MethodPing: Ping("1.2.3")})
	var result rpcprotocol.PingResult
	if err := client.Call(context.Background(), rpcprotocol.MethodPing, nil, &result); err != nil || result != (rpcprotocol.PingResult{Pong: "pong", Version: "1.2.3"}) {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestErrorsReachTheEditorWithTheirCodes(t *testing.T) {
	client, _, _ := connect(t, Methods{
		"coded": func(context.Context, Call) (any, error) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeUnknownPlan, "no plan p9")
		},
		"plain":  func(context.Context, Call) (any, error) { return nil, errors.New("disk full") },
		"params": func(_ context.Context, call Call) (any, error) { var p struct{ N int }; return nil, call.Decode(&p) },
	})
	cases := map[string]struct {
		params  any
		code    int64
		message string
	}{
		"coded":   {nil, rpcprotocol.CodeUnknownPlan, "no plan p9"},
		"plain":   {nil, rpcprotocol.CodeInternalError, "disk full"},
		"params":  {map[string]string{"N": "not a number"}, rpcprotocol.CodeInvalidParams, "invalid params"},
		"missing": {nil, jsonrpc2.CodeMethodNotFound, `the engine has no method "missing"`},
	}
	for method, want := range cases {
		err := client.Call(context.Background(), method, want.params, nil)
		if code(err) != want.code || !strings.Contains(err.Error(), want.message) {
			t.Errorf("%s: %v (code %d)", method, err, code(err))
		}
	}
}

func TestACancelledRequestStopsAndSaysSo(t *testing.T) {
	started := make(chan struct{})
	client, _, _ := connect(t, Methods{"slow": func(ctx context.Context, _ Call) (any, error) {
		close(started)
		<-ctx.Done()

		return nil, ctx.Err()
	}})
	waiter, err := client.DispatchCall(context.Background(), "slow", nil, jsonrpc2.PickID(jsonrpc2.ID{Num: 42}))
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := client.Notify(context.Background(), rpcprotocol.MethodCancelRequest, map[string]int{"id": 42}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := waiter.Wait(ctx, nil); code(err) != rpcprotocol.CodeRequestCancelled {
		t.Fatalf("%v", err)
	}
}

// A method can ask the editor while the editor waits for it: requests are not handled
// one at a time, or this would deadlock.
func TestAMethodCanCallBackIntoTheEditor(t *testing.T) {
	client, _, _ := connect(t, Methods{"ask": func(ctx context.Context, call Call) (any, error) {
		var answer map[string]string
		err := call.Editor.Call(ctx, "host/echo", "hello", &answer)

		return answer, err
	}})
	var result map[string]string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Call(ctx, "ask", nil, &result); err != nil || result["echo"] != `"hello"` {
		t.Fatalf("%v %v", result, err)
	}
}

func TestTheLogNeverHoldsParamsOrResults(t *testing.T) {
	client, log, _ := connect(t, Methods{"secret": func(_ context.Context, call Call) (any, error) {
		var params rpcprotocol.Credential
		if err := call.Decode(&params); err != nil {
			return nil, err
		}

		return map[string]string{"echo": params.Token}, nil
	}})
	if err := client.Call(context.Background(), "secret", rpcprotocol.Credential{Kind: "pat", Token: "s3cr3t-token"}, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(log.String(), "s3cr3t") || !strings.Contains(log.String(), "method=secret") {
		t.Fatalf("log %q", log.String())
	}
}

func TestServeReturnsWhenTheEditorHangsUp(t *testing.T) {
	client, _, served := connect(t, Methods{})
	_ = client.Close()
	select {
	case err := <-served:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
	}
}

// Hanging up mid-request cancels it and waits for it, so it can finish cleanly.
func TestAHangUpCancelsRunningRequestsAndWaitsForThem(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	client, _, served := connect(t, Methods{"slow": func(ctx context.Context, _ Call) (any, error) {
		close(started)
		<-ctx.Done()
		time.Sleep(50 * time.Millisecond) // winding down, e.g. writing annotations back
		close(finished)

		return nil, ctx.Err()
	}})
	if _, err := client.DispatchCall(context.Background(), "slow", nil); err != nil {
		t.Fatal(err)
	}
	<-started
	_ = client.Close()
	select {
	case <-served:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
	}
	select {
	case <-finished:
	default:
		t.Fatal("Serve returned before the running request finished")
	}
}
