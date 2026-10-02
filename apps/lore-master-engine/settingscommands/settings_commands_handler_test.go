package settingscommands

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
)

func connect(t *testing.T) *jsonrpc2.Conn {
	t.Helper()
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodSettingsRead: ReadSettings(),
			rpcprotocol.MethodSettingsSave: SaveSettings(),
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

func read(t *testing.T, conn *jsonrpc2.Conn, root string) rpcprotocol.SettingsReadResult {
	t.Helper()
	var result rpcprotocol.SettingsReadResult
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsRead, rpcprotocol.SettingsReadParams{WorkspaceRoot: root}, &result); err != nil {
		t.Fatal(err)
	}

	return result
}

// answer fills in what the first-sync wizard asks.
func answer(settings rpcprotocol.Settings) rpcprotocol.Settings {
	settings.Outputs[0].BaseURL = "https://acme.atlassian.net/wiki"
	settings.Outputs[0].Space = "ENG"
	settings.Outputs[0].ParentPageID = "98306"
	settings.Outputs[0].TitlePrefix = "ENG"

	return settings
}

func TestTheFirstSyncWizardRoundTrip(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	fresh := read(t, conn, root)
	if fresh.Exists || !fresh.FirstSync || len(fresh.Settings.Outputs) != 1 || fresh.Settings.Outputs[0].MermaidMode != "image" {
		t.Fatalf("defaults: %+v", fresh)
	}

	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: answer(fresh.Settings)}, nil); err != nil {
		t.Fatal(err)
	}
	saved := read(t, conn, root)
	if !saved.Exists || saved.FirstSync || saved.Settings.Outputs[0].ParentPageID != "98306" || saved.Settings.Outputs[0].Content[0].Roots[0] != "." {
		t.Fatalf("saved: %+v", saved)
	}
	content, _ := os.ReadFile(filepath.Join(root, ".lore-master.yaml"))
	if !strings.HasPrefix(string(content), "# Lore Master configuration.") {
		t.Fatalf("a new file explains itself:\n%s", content)
	}
}

func TestSavingKeepsTheAuthorsComments(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	file := filepath.Join(root, ".lore-master.yaml")
	authored := "version: 1\n# Our handbook goes to the ENG space.\noutputs:\n  - platform: confluence\n    baseUrl: https://acme.atlassian.net/wiki\n    space: ENG # the team space\n    parentPageId: \"98306\"\n    titlePrefix: ENG\n"
	if err := os.WriteFile(file, []byte(authored), 0o600); err != nil {
		t.Fatal(err)
	}
	settings := read(t, conn, root).Settings
	settings.Outputs[0].Space = "DOCS"
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: settings}, nil); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(file)
	if !strings.Contains(string(content), "# Our handbook goes to the ENG space.") || !strings.Contains(string(content), "space: DOCS # the team space") {
		t.Fatalf("comments survive:\n%s", content)
	}
}

func TestBadSettingsAreRefusedWithTheReason(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	settings := answer(read(t, conn, root).Settings)
	settings.Outputs[0].MermaidMode = "crayon"
	err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: settings}, nil)
	if code(err) != rpcprotocol.CodeInvalidSettings || !strings.Contains(err.Error(), "crayon") {
		t.Fatalf("%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".lore-master.yaml")); !os.IsNotExist(statErr) {
		t.Fatal("nothing is written when refused")
	}

	file := filepath.Join(root, ".lore-master.yaml")
	_ = os.WriteFile(file, []byte("version: 1\noutputs:\n  - platform: confluence\n    apiToken: abc\n"), 0o600)
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsRead, rpcprotocol.SettingsReadParams{WorkspaceRoot: root}, nil); code(err) != rpcprotocol.CodeInvalidSettings {
		t.Fatalf("a secret in the file: %v", err)
	}

	_ = os.WriteFile(file, []byte("version: 1\noutputs:\n  - platform: confluence\n    baseUrl: https://acme.atlassian.net/wiki\n    space: ENG\n    parentPageId: \"1\"\n    titlePrefix: ENG\n    linkMode: rainbow\n"), 0o600)
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsRead, rpcprotocol.SettingsReadParams{WorkspaceRoot: root}, nil); code(err) != rpcprotocol.CodeInvalidSettings || !strings.Contains(err.Error(), `linkMode "rainbow" is not one of title, id`) {
		t.Fatalf("a bad value names itself: %v", err)
	}

	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsRead, rpcprotocol.SettingsReadParams{WorkspaceRoot: "relative"}, nil); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("relative: %v", err)
	}
}
