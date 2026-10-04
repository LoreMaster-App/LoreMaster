package hostbridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// editor plays the VS Code side: it draws "graph" sources, refuses "fail", answers
// junk to "junk", never answers "slow", and records progress.
type editor struct {
	mu       sync.Mutex
	asked    []string
	progress []rpcprotocol.ProgressParams
}

func (e *editor) Handle(ctx context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	switch request.Method {
	case rpcprotocol.MethodHostRenderDiagram:
		var params rpcprotocol.RenderDiagramParams
		_ = json.Unmarshal(*request.Params, &params)
		e.mu.Lock()
		e.asked = append(e.asked, params.Source)
		e.mu.Unlock()
		switch params.Source {
		case "slow":
			return
		case "fail":
			_ = conn.ReplyWithError(ctx, request.ID, &jsonrpc2.Error{Code: 1, Message: "mermaid: parse error on line 1"})
		case "junk":
			_ = conn.Reply(ctx, request.ID, rpcprotocol.RenderDiagramResult{SVG: "<html>oops</html>"})
		default:
			_ = conn.Reply(ctx, request.ID, rpcprotocol.RenderDiagramResult{SVG: "  <svg>" + params.Language + ":" + params.Source + "</svg>\n"})
		}
	case rpcprotocol.MethodHostProgress:
		var params rpcprotocol.ProgressParams
		_ = json.Unmarshal(*request.Params, &params)
		e.mu.Lock()
		e.progress = append(e.progress, params)
		e.mu.Unlock()
	}
}

// draw is a test method: it renders each source in turn with one renderer, as one
// sync would, and returns what happened to each.
func draw(timeout time.Duration) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var sources []string
		if err := call.Decode(&sources); err != nil {
			return nil, err
		}
		renderer := NewDiagramRenderer(call.Editor, timeout)
		var results []string
		for _, source := range sources {
			svg, err := renderer.Render(ctx, "mermaid", source)
			if err != nil {
				results = append(results, "error: "+err.Error())
			} else {
				results = append(results, string(svg))
			}
		}
		progress := Progress(ctx, call.Editor, "plan-7")
		progress(1, 2, "written: ENG: Home")
		progress(2, 2, "unchanged: ENG: Guide")

		return results, nil
	}
}

func connect(t *testing.T, timeout time.Duration) (*jsonrpc2.Conn, *editor) {
	t.Helper()
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{"test/draw": draw(timeout)}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	side := &editor{}
	conn := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}), jsonrpc2.AsyncHandler(side))
	t.Cleanup(func() { _ = conn.Close() })

	return conn, side
}

func TestTheEditorDrawsAndHearsProgress(t *testing.T) {
	conn, side := connect(t, time.Second)
	var results []string
	if err := conn.Call(context.Background(), "test/draw", []string{"graph TD; A-->B", "fail", "junk"}, &results); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"<svg>mermaid:graph TD; A-->B</svg>",
		"error: the editor could not draw the diagram: jsonrpc2: code 1 message: mermaid: parse error on line 1",
		"error: the editor answered with something that is not an SVG",
	}
	if strings.Join(results, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s", strings.Join(results, "\n"))
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		side.mu.Lock()
		got := append([]rpcprotocol.ProgressParams(nil), side.progress...)
		side.mu.Unlock()
		if len(got) == 2 {
			// The editor handles notifications asynchronously, so the two may be recorded in
			// either order; assert both arrived with their content, not their arrival order.
			byMessage := map[string]rpcprotocol.ProgressParams{got[0].Message: got[0], got[1].Message: got[1]}
			if byMessage["written: ENG: Home"] != (rpcprotocol.ProgressParams{PlanID: "plan-7", Message: "written: ENG: Home", Done: 1, Total: 2}) ||
				byMessage["unchanged: ENG: Guide"] != (rpcprotocol.ProgressParams{PlanID: "plan-7", Message: "unchanged: ENG: Guide", Done: 2, Total: 2}) {
				t.Fatalf("%+v", got)
			}

			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("progress %+v", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAnEditorThatMissesTheDeadlineIsNotAskedAgain(t *testing.T) {
	conn, side := connect(t, 100*time.Millisecond)
	var results []string
	started := time.Now()
	if err := conn.Call(context.Background(), "test/draw", []string{"slow", "graph LR; X-->Y", "graph LR; Z"}, &results); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(results[0], "error: the editor did not draw the diagram within 100ms") ||
		results[1] != "error: the editor did not answer an earlier diagram in time" || results[2] != results[1] {
		t.Fatalf("%q", results)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("one timeout, not three: %s", elapsed)
	}
	side.mu.Lock()
	defer side.mu.Unlock()
	if len(side.asked) != 1 {
		t.Fatalf("asked %q", side.asked)
	}
}

func TestACancelledSyncStopsWaitingForTheEditor(t *testing.T) {
	renderer := NewDiagramRenderer(silent{}, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := renderer.Render(ctx, "mermaid", "graph"); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	if renderer.gaveUp.Load() {
		t.Fatal("a cancellation is not the editor's fault")
	}
	if NewDiagramRenderer(silent{}, 0).timeout != DefaultRenderTimeout {
		t.Fatal("default timeout")
	}
}

// silent never answers until its context ends.
type silent struct{}

func (silent) Call(ctx context.Context, _ string, _ any, _ any) error {
	<-ctx.Done()

	return ctx.Err()
}

func (silent) Notify(context.Context, string, any) error { return nil }

// The sync's own deadline running out is not the editor missing its own.
func TestTheSyncsDeadlineIsNotTheEditors(t *testing.T) {
	renderer := NewDiagramRenderer(silent{}, time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := renderer.Render(ctx, "mermaid", "graph"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("%v", err)
	}
	if renderer.gaveUp.Load() {
		t.Fatal("the editor was blamed for the sync's deadline")
	}
}
