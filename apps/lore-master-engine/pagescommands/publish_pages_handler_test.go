package pagescommands

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
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
			rpcprotocol.MethodPagesPublish: PublishPages(),
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

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// pagesWorkspace builds a bare "remote" and a work repo whose origin points at it, with a
// Markdown file and a github-pages .lore-master.yaml. It returns the work repo path.
func pagesWorkspace(t *testing.T, config string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	remote := filepath.ToSlash(t.TempDir())
	runGit(t, remote, "init", "--bare", "--initial-branch=main")

	work := t.TempDir()
	runGit(t, work, "init", "--initial-branch=main")
	runGit(t, work, "config", "user.name", "Test")
	runGit(t, work, "config", "user.email", "test@example.com")
	write(t, work, "readme.md", "# Home\n\nWelcome.")
	write(t, work, ".lore-master.yaml", config)
	runGit(t, work, "add", "-A")
	runGit(t, work, "commit", "-m", "initial")
	runGit(t, work, "remote", "add", "origin", remote)
	runGit(t, work, "push", "-u", "origin", "main")

	return work
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const pagesConfig = "version: 1\noutputs:\n  - platform: github-pages\n    direction: to-platform\n    content:\n      - type: markdown\n        roots: [\".\"]\n        template: default\n"

func publish(t *testing.T, conn *jsonrpc2.Conn, root string, output int) (rpcprotocol.PagesPublishResult, error) {
	t.Helper()
	var result rpcprotocol.PagesPublishResult
	err := conn.Call(context.Background(), rpcprotocol.MethodPagesPublish, rpcprotocol.PagesPublishParams{WorkspaceRoot: root, Output: output}, &result)

	return result, err
}

func TestPublishPagesGeneratesAndPushesTheSite(t *testing.T) {
	conn := connect(t)
	work := pagesWorkspace(t, pagesConfig)

	result, err := publish(t, conn, work, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Branch != "gh-pages" || result.Commit == "" || result.Files == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}

	// Clone what was pushed and confirm the generated site is there.
	remote, err := exec.Command("git", "-C", work, "remote", "get-url", "origin").Output()
	if err != nil {
		t.Fatal(err)
	}
	clone := t.TempDir()
	runGit(t, "", "clone", "--branch", "gh-pages", "--single-branch", strings.TrimSpace(string(remote)), clone)
	if _, err := os.Stat(filepath.Join(clone, "index.html")); err != nil {
		t.Errorf("published site is missing index.html: %v", err)
	}
	if _, err := os.Stat(filepath.Join(clone, "assets", "lore-master.css")); err != nil {
		t.Errorf("published site is missing the stylesheet: %v", err)
	}
}

func TestPublishPagesRejectsANonPagesOutput(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	confluence := "version: 1\noutputs:\n  - platform: confluence\n    baseUrl: https://acme.atlassian.net/wiki\n    space: ENG\n    parentPageId: \"1\"\n    titlePrefix: ENG\n"
	write(t, root, ".lore-master.yaml", confluence)

	_, err := publish(t, conn, root, 0)
	if code(err) != rpcprotocol.CodeInvalidParams || !strings.Contains(err.Error(), "not github-pages") {
		t.Fatalf("%v", err)
	}
}

func TestPublishPagesRejectsAMissingOutput(t *testing.T) {
	conn := connect(t)
	work := pagesWorkspace(t, pagesConfig)

	_, err := publish(t, conn, work, 5)
	if code(err) != rpcprotocol.CodeInvalidParams || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("%v", err)
	}
}

func TestPublishPagesNeedsAnAbsoluteWorkspace(t *testing.T) {
	conn := connect(t)
	if _, err := publish(t, conn, "relative", 0); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("%v", err)
	}
}
