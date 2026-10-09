package pagescommands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

func build(t *testing.T, conn *jsonrpc2.Conn, root, outDir string) (rpcprotocol.PagesBuildResult, error) {
	t.Helper()
	var result rpcprotocol.PagesBuildResult
	err := conn.Call(context.Background(), rpcprotocol.MethodPagesBuild, rpcprotocol.PagesBuildParams{WorkspaceRoot: root, Output: 0, OutDir: outDir}, &result)

	return result, err
}

func TestBuildPagesWritesTheSiteIntoTheFolder(t *testing.T) {
	conn := connect(t)
	work := pagesWorkspace(t, pagesConfig)
	out := filepath.Join(work, "dist", "docs")

	result, err := build(t, conn, work, out)
	if err != nil {
		t.Fatal(err)
	}
	if result.Files == 0 || result.OutDir != out {
		t.Fatalf("unexpected result: %+v", result)
	}
	for _, name := range []string{"index.html", filepath.Join("assets", "lore-master.css")} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("built site is missing %s: %v", name, err)
		}
	}
}

func TestBuildPagesTwiceGivesTheSameFiles(t *testing.T) {
	conn := connect(t)
	work := pagesWorkspace(t, pagesConfig)
	out := filepath.Join(work, "site")

	if _, err := build(t, conn, work, out); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if _, err := build(t, conn, work, out); err != nil {
		t.Fatalf("a rebuild into the folder it wrote must work: %v", err)
	}
	second, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if string(first) != string(second) {
		t.Error("two builds of the same workspace differ")
	}
}

func TestBuildPagesRefusesAFolderWithOtherContent(t *testing.T) {
	conn := connect(t)
	work := pagesWorkspace(t, pagesConfig)
	out := filepath.Join(work, "dist")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, out, "app.js", "the web app")

	_, err := build(t, conn, work, out)
	if code(err) != rpcprotocol.CodeInvalidParams || !strings.Contains(err.Error(), "did not write") {
		t.Fatalf("%v", err)
	}
	if kept, _ := os.ReadFile(filepath.Join(out, "app.js")); string(kept) != "the web app" {
		t.Error("the other site's file was touched")
	}
}

func TestBuildPagesRefusesTheWorkspaceItself(t *testing.T) {
	conn := connect(t)
	work := pagesWorkspace(t, pagesConfig)

	for _, out := range []string{work, filepath.Dir(work)} {
		if _, err := build(t, conn, work, out); code(err) != rpcprotocol.CodeInvalidParams || !strings.Contains(err.Error(), "would replace the workspace") {
			t.Errorf("%s: %v", out, err)
		}
	}
	if _, err := os.Stat(filepath.Join(work, "readme.md")); err != nil {
		t.Error("the workspace was touched")
	}
}

func TestBuildPagesNeedsAnAbsoluteFolder(t *testing.T) {
	conn := connect(t)
	work := pagesWorkspace(t, pagesConfig)

	if _, err := build(t, conn, work, "dist/docs"); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("%v", err)
	}
}
