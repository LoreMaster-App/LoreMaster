package clicommands

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// engineDouble stands in for the engine: it records every call and answers from the table.
type engineDouble struct {
	calls   []string
	params  map[string]json.RawMessage
	methods rpcserver.Methods
}

func newEngineDouble(answers map[string]func(call rpcserver.Call) (any, error)) *engineDouble {
	double := &engineDouble{params: map[string]json.RawMessage{}, methods: rpcserver.Methods{}}
	for name, answer := range answers {
		double.methods[name] = func(_ context.Context, call rpcserver.Call) (any, error) {
			double.calls = append(double.calls, name)
			double.params[name] = call.Params

			return answer(call)
		}
	}

	return double
}

// answer replaces how the double answers one method.
func (d *engineDouble) answer(name string, answer func(call rpcserver.Call) (any, error)) {
	d.methods[name] = func(_ context.Context, call rpcserver.Call) (any, error) {
		d.calls = append(d.calls, name)
		d.params[name] = call.Params

		return answer(call)
	}
}

func (d *engineDouble) called(name string) bool {
	for _, call := range d.calls {
		if call == name {
			return true
		}
	}

	return false
}

func (d *engineDouble) paramsOf(t *testing.T, name string, into any) {
	t.Helper()
	if err := json.Unmarshal(d.params[name], into); err != nil {
		t.Fatalf("params of %s: %v", name, err)
	}
}

// run executes a command line against the double and returns what it printed and its exit code.
func (d *engineDouble) run(t *testing.T, env map[string]string, args ...string) (stdout string, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = Run(context.Background(), args, Environment{
		Getenv: func(name string) string { return env[name] }, Stdout: &out, Stderr: &errOut, WorkingDir: t.TempDir(), Version: "1.2.3",
	}, d.methods)

	return out.String(), errOut.String(), code
}

func TestTheEnginesProgressReachesTheUserOnStderrAndDiagramsAreRefused(t *testing.T) {
	var refusal string
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSettingsRead: func(rpcserver.Call) (any, error) {
			return rpcprotocol.SettingsReadResult{Settings: rpcprotocol.Settings{Outputs: []rpcprotocol.Output{{Platform: "github-pages"}}}}, nil
		},
		rpcprotocol.MethodPagesPublish: func(call rpcserver.Call) (any, error) {
			if err := call.Editor.Notify(context.Background(), rpcprotocol.MethodHostProgress, rpcprotocol.ProgressParams{PlanID: "p", Message: "writing a.md", Done: 1, Total: 2}); err != nil {
				return nil, err
			}
			err := call.Editor.Call(context.Background(), rpcprotocol.MethodHostRenderDiagram, rpcprotocol.RenderDiagramParams{Language: "mermaid", Source: "graph TD"}, &rpcprotocol.RenderDiagramResult{})
			if err != nil {
				refusal = err.Error()
			}

			return rpcprotocol.PagesPublishResult{Changed: true, Files: 2, Branch: "gh-pages", Commit: "abc1234"}, nil
		},
	})

	_, stderr, code := double.run(t, nil, "sync", "--yes")

	if code != ExitOK {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "[1/2] writing a.md") {
		t.Fatalf("progress did not reach stderr: %q", stderr)
	}
	if !strings.Contains(refusal, "the command line draws no diagrams") {
		t.Fatalf("the diagram request was not refused: %q", refusal)
	}
}
