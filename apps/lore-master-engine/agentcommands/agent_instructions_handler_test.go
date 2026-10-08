package agentcommands

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documenttree"
)

func connect(t *testing.T) *jsonrpc2.Conn {
	t.Helper()
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodAgentInstructions: AgentInstructions(),
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	conn := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}), jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (any, error) { return nil, nil })))
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

func code(err error) int64 {
	var wire *jsonrpc2.Error
	if errors.As(err, &wire) {
		return wire.Code
	}

	return 0
}

func instructions(t *testing.T, conn *jsonrpc2.Conn, root string) (rpcprotocol.AgentInstructionsResult, error) {
	t.Helper()
	var result rpcprotocol.AgentInstructionsResult
	err := conn.Call(context.Background(), rpcprotocol.MethodAgentInstructions, rpcprotocol.AgentInstructionsParams{WorkspaceRoot: root}, &result)

	return result, err
}

func TestInstructionsQuoteTheLiveRulesAndTheWorkspace(t *testing.T) {
	root := t.TempDir()
	yaml := "version: 1\nignore: [drafts/]\ngenerators:\n  - type: test-results\n    output: docs/tests\noutputs:\n  - platform: confluence\n    baseUrl: https://acme.atlassian.net/wiki\n    space: ENG\n    parentPageId: \"1\"\n    titlePrefix: ENG\n    content:\n      - type: markdown\n        roots: [docs]\n"
	if err := os.WriteFile(filepath.Join(root, workspacesettings.FileName), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := instructions(t, connect(t), root)
	if err != nil {
		t.Fatal(err)
	}

	if !result.HasConfig {
		t.Fatal("the workspace has a config")
	}
	for _, convention := range documenttree.NestingConventions() {
		if !strings.Contains(result.Instructions, convention.Summary) {
			t.Errorf("the rule %q is not quoted", convention.Key)
		}
	}
	for _, want := range []string{"Confluence space `ENG`", "`ENG: <the page's first heading>`", "Only Markdown under `docs` is synced", "`drafts/`", "`docs/tests` — written by the `test-results` generator"} {
		if !strings.Contains(result.Instructions, want) {
			t.Errorf("the instructions lack %q", want)
		}
	}
}

func TestInstructionsForAWorkspaceWithoutAConfig(t *testing.T) {
	result, err := instructions(t, connect(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if result.HasConfig || !strings.Contains(result.Instructions, "There is no `.lore-master.yaml` yet") {
		t.Fatalf("result %+v", result)
	}
}

func TestInstructionsRefuseBadParamsAndSettings(t *testing.T) {
	conn := connect(t)
	if _, err := instructions(t, conn, "relative"); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("relative root: %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, workspacesettings.FileName), []byte("version: 1\nnonsense: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := instructions(t, conn, root); code(err) != rpcprotocol.CodeInvalidSettings {
		t.Fatalf("bad settings: %v", err)
	}
}
