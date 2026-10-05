package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
)

// run feeds the frames to Serve (one JSON object per line) and returns the decoded
// responses, in order. Notifications produce no response.
func run(t *testing.T, frames ...string) []response {
	t.Helper()
	input := strings.Join(frames, "\n") + "\n"
	var out bytes.Buffer
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := Serve(context.Background(), strings.NewReader(input), &out, "test-version", logger); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var responses []response
	scanner := bufio.NewScanner(&out)
	scanner.Buffer(make([]byte, 0, 64*1024), maxFrameBytes)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var r response
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatalf("response is not JSON: %q (%v)", line, err)
		}
		responses = append(responses, r)
	}

	return responses
}

func TestInitializeEchoesProtocolAndReportsServer(t *testing.T) {
	responses := run(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{}}}`)
	if len(responses) != 1 {
		t.Fatalf("want 1 response, got %d", len(responses))
	}
	var result initializeResult
	decodeResult(t, responses[0], &result)
	if result.ProtocolVersion != "2025-06-18" {
		t.Errorf("protocolVersion = %q", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != serverName || result.ServerInfo.Version != "test-version" {
		t.Errorf("serverInfo = %+v", result.ServerInfo)
	}
	if _, ok := result.Capabilities["tools"]; !ok {
		t.Errorf("capabilities missing tools: %+v", result.Capabilities)
	}
}

func TestInitializeDefaultsProtocolWhenClientOmitsIt(t *testing.T) {
	responses := run(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	var result initializeResult
	decodeResult(t, responses[0], &result)
	if result.ProtocolVersion != defaultProtocolVersion {
		t.Errorf("protocolVersion = %q, want default %q", result.ProtocolVersion, defaultProtocolVersion)
	}
}

func TestNotificationsAreNotAnswered(t *testing.T) {
	responses := run(t,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)
	if len(responses) != 1 {
		t.Fatalf("want only the tools/list answer, got %d responses", len(responses))
	}
}

func TestToolsListAdvertisesTheNestingRulesTool(t *testing.T) {
	responses := run(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	var result toolsListResult
	decodeResult(t, responses[0], &result)
	if len(result.Tools) != 1 || result.Tools[0].Name != nestingRulesToolName {
		t.Fatalf("tools = %+v", result.Tools)
	}
	if typ, _ := result.Tools[0].InputSchema["type"].(string); typ != "object" {
		t.Errorf("inputSchema.type = %v", result.Tools[0].InputSchema["type"])
	}
}

func TestToolCallReturnsTheRulesFromDocumenttree(t *testing.T) {
	responses := run(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"loremaster_nesting_rules","arguments":{}}}`)
	var result toolCallResult
	decodeResult(t, responses[0], &result)
	if result.IsError {
		t.Fatal("tool reported an error")
	}
	if len(result.Content) == 0 || result.Content[0].Type != "text" {
		t.Fatalf("content = %+v", result.Content)
	}
	text := result.Content[0].Text
	for _, want := range []string{"explicit-parent", "dotted-name", "directory-index", "selected-parent", "title-prefix"} {
		if !strings.Contains(text, want) {
			t.Errorf("rules text missing %q:\n%s", want, text)
		}
	}
}

func TestUnknownToolIsAnError(t *testing.T) {
	responses := run(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nope"}}`)
	if responses[0].Error == nil || responses[0].Error.Code != codeInvalidParams {
		t.Fatalf("want invalid-params error, got %+v", responses[0])
	}
}

func TestUnknownMethodIsMethodNotFound(t *testing.T) {
	responses := run(t, `{"jsonrpc":"2.0","id":1,"method":"resources/list"}`)
	if responses[0].Error == nil || responses[0].Error.Code != codeMethodNotFound {
		t.Fatalf("want method-not-found, got %+v", responses[0])
	}
}

func TestUnparseableFrameIsSkipped(t *testing.T) {
	responses := run(t,
		`not json`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
	)
	if len(responses) != 1 || responses[0].Error != nil {
		t.Fatalf("want one clean ping answer, got %+v", responses)
	}
}

func decodeResult(t *testing.T, r response, into any) {
	t.Helper()
	if r.Error != nil {
		t.Fatalf("unexpected error response: %+v", r.Error)
	}
	raw, err := json.Marshal(r.Result)
	if err != nil {
		t.Fatalf("remarshal result: %v", err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("decode result into %T: %v", into, err)
	}
}
