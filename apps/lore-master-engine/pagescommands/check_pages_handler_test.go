package pagescommands

import (
	"context"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

func check(t *testing.T, conn *jsonrpc2.Conn, root string) rpcprotocol.PagesCheckResult {
	t.Helper()
	var result rpcprotocol.PagesCheckResult
	err := conn.Call(context.Background(), rpcprotocol.MethodPagesCheck, rpcprotocol.PagesCheckParams{WorkspaceRoot: root, Output: 0}, &result)
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func TestCheckPagesReportsPendingChangesThenUpToDateAfterAPublish(t *testing.T) {
	conn := connect(t)
	work := pagesWorkspace(t, pagesConfig)

	before := check(t, conn, work)
	if before.UpToDate || before.ChangesTotal == 0 || before.Branch != "gh-pages" {
		t.Fatalf("a site never published should have changes: %+v", before)
	}
	if _, err := publish(t, conn, work, 0); err != nil {
		t.Fatal(err)
	}

	after := check(t, conn, work)
	if !after.UpToDate || after.ChangesTotal != 0 {
		t.Fatalf("a published site should be up to date: %+v", after)
	}

	write(t, work, "readme.md", "# Home\n\nWelcome back.")
	edited := check(t, conn, work)
	if edited.UpToDate || len(edited.Changes) == 0 {
		t.Fatalf("an edited page should be pending: %+v", edited)
	}
}
