// Package mcpserver runs LoreMaster's engine as an MCP server over stdio, so a
// documentation-authoring agent can learn the rules the sync enforces. The transport is
// newline-delimited JSON-RPC 2.0 (the MCP stdio transport); this is deliberately a small
// hand-rolled server — one tool, no resources or prompts yet — rather than a dependency,
// and it can be revisited (see issue #212) if the tool surface grows.
package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

// defaultProtocolVersion is used when a client does not state one; otherwise the client's
// requested version is echoed back.
const defaultProtocolVersion = "2025-06-18"

const serverName = "lore-master"

// maxFrameBytes caps a single JSON-RPC line, so a malformed stream cannot exhaust memory.
const maxFrameBytes = 8 * 1024 * 1024

// Serve reads newline-delimited JSON-RPC from in and writes answers to out until the
// input ends or ctx is cancelled. version is reported as the server version;
// workspaceRoot is the open workspace the workspace-aware tools read (empty disables
// them, with a clear message). Logs go to logger (stderr); out carries only MCP frames.
func Serve(ctx context.Context, in io.Reader, out io.Writer, version string, workspaceRoot string, logger *slog.Logger) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), maxFrameBytes)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var message request
		if err := json.Unmarshal(line, &message); err != nil {
			logger.Debug("mcp: ignoring unparseable frame", "error", err.Error())

			continue
		}
		answer, answered := dispatch(ctx, message, version, workspaceRoot)
		if !answered {
			continue
		}
		if err := encoder.Encode(answer); err != nil {
			return fmt.Errorf("writing mcp response: %w", err)
		}
	}

	return scanner.Err()
}

// dispatch routes one message; the bool is false when nothing should be written (a
// notification).
func dispatch(ctx context.Context, message request, version string, workspaceRoot string) (response, bool) {
	switch message.Method {
	case "initialize":
		return reply(message, initialize(message, version)), true
	case "notifications/initialized":
		return response{}, false
	case "ping":
		return reply(message, map[string]any{}), true
	case "tools/list":
		return reply(message, toolsListResult{Tools: []toolDescriptor{
			nestingRulesTool(),
			previewTreeTool(),
			validateDocumentTool(),
			placeDocumentTool(),
		}}), true
	case "tools/call":
		result, err := callTool(ctx, message.Params, workspaceRoot)
		if err != nil {
			return replyError(message, codeInvalidParams, err.Error()), true
		}

		return reply(message, result), true
	default:
		if message.isNotification() {
			return response{}, false
		}

		return replyError(message, codeMethodNotFound, fmt.Sprintf("method %q is not supported", message.Method)), true
	}
}

// callTool runs the named tool. A Go error here is a protocol-level problem (unknown tool,
// unparseable arguments); a tool whose work fails returns a result with IsError set, so
// the model sees the explanation.
func callTool(ctx context.Context, params json.RawMessage, workspaceRoot string) (toolCallResult, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &call); err != nil {
			return toolCallResult{}, fmt.Errorf("invalid tools/call params: %w", err)
		}
	}
	switch call.Name {
	case nestingRulesToolName:
		return nestingRulesResult(), nil
	case previewTreeToolName:
		return previewTreeResult(ctx, workspaceRoot), nil
	case validateDocumentToolName:
		return validateDocumentResult(ctx, workspaceRoot, call.Arguments), nil
	case placeDocumentToolName:
		return placeDocumentResult(ctx, workspaceRoot, call.Arguments), nil
	default:
		return toolCallResult{}, fmt.Errorf("unknown tool %q", call.Name)
	}
}

// errorResult is a tool result that reports an operational failure to the caller (not a
// protocol error): the model reads the message and can react.
func errorResult(format string, args ...any) toolCallResult {
	return toolCallResult{
		Content: []contentBlock{{Type: "text", Text: fmt.Sprintf(format, args...)}},
		IsError: true,
	}
}

// JSON-RPC 2.0 error codes used here.
const (
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

func initialize(message request, version string) initializeResult {
	protocolVersion := defaultProtocolVersion
	if len(message.Params) > 0 {
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(message.Params, &params); err == nil && params.ProtocolVersion != "" {
			protocolVersion = params.ProtocolVersion
		}
	}

	return initializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities:    map[string]any{"tools": map[string]any{}},
		ServerInfo:      serverInfo{Name: serverName, Version: version},
	}
}

func reply(message request, result any) response {
	return response{JSONRPC: "2.0", ID: message.ID, Result: result}
}

func replyError(message request, code int, text string) response {
	return response{JSONRPC: "2.0", ID: message.ID, Error: &responseError{Code: code, Message: text}}
}
