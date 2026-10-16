package exportcommands

import (
	"context"
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
			rpcprotocol.MethodExportFormats: ListFormats(),
			rpcprotocol.MethodExportRun:     RunExport(),
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	conn := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}), jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (any, error) { return nil, nil })))
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

func run(conn *jsonrpc2.Conn, params rpcprotocol.ExportRunParams) (rpcprotocol.ExportRunResult, error) {
	var result rpcprotocol.ExportRunResult
	err := conn.Call(context.Background(), rpcprotocol.MethodExportRun, params, &result)

	return result, err
}

func TestFormatsAreListedWithWhetherTheyAreText(t *testing.T) {
	var result rpcprotocol.ExportFormatsResult
	if err := connect(t).Call(context.Background(), rpcprotocol.MethodExportFormats, nil, &result); err != nil {
		t.Fatal(err)
	}

	binary := map[string]bool{}
	for _, format := range result.Formats {
		binary[format.ID] = format.Binary
	}
	if !binary["docx"] || !binary["odt"] || binary["html"] || binary["jira-wiki"] {
		t.Fatalf("unexpected formats: %+v", result.Formats)
	}
}

func TestExportWritesNextToTheDocumentOrIntoAFolder(t *testing.T) {
	conn := connect(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	if err := os.WriteFile(path, []byte("# Guide\n\nOn disk.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := run(conn, rpcprotocol.ExportRunParams{Path: path, Format: "txt"})
	if err != nil || result.Path != filepath.Join(dir, "guide.txt") {
		t.Fatalf("result %+v, err %v", result, err)
	}
	if written, _ := os.ReadFile(result.Path); !strings.Contains(string(written), "On disk.") {
		t.Fatalf("written %q", written)
	}

	out := filepath.Join(dir, "dist")
	if result, err = run(conn, rpcprotocol.ExportRunParams{Path: path, Format: "docx", OutDir: out}); err != nil || result.Path != filepath.Join(out, "guide.docx") {
		t.Fatalf("result %+v, err %v", result, err)
	}
}

func TestExportUsesUnsavedContentAndCanReturnText(t *testing.T) {
	conn := connect(t)
	path := filepath.Join(t.TempDir(), "guide.md")
	if err := os.WriteFile(path, []byte("# Guide\n\nOn disk.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	unsaved := "# Guide\n\nNot saved yet.\n"

	result, err := run(conn, rpcprotocol.ExportRunParams{Path: path, Format: "jira-wiki", Content: &unsaved, Inline: true})
	if err != nil || !strings.Contains(result.Text, "Not saved yet.") || result.Path != "" {
		t.Fatalf("result %+v, err %v", result, err)
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(path), "guide.jira.txt")); statErr == nil {
		t.Error("an inline export wrote a file")
	}
}

func TestExportRefusesWhatItCannotDo(t *testing.T) {
	conn := connect(t)
	path := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(path, []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for name, params := range map[string]rpcprotocol.ExportRunParams{
		"relative path":   {Path: "a.md", Format: "txt"},
		"unknown format":  {Path: path, Format: "pdf-from-the-future"},
		"inline binary":   {Path: path, Format: "docx", Inline: true},
		"relative outdir": {Path: path, Format: "txt", OutDir: "dist"},
		"missing file":    {Path: filepath.Join(filepath.Dir(path), "missing.md"), Format: "txt"},
	} {
		if _, err := run(conn, params); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
