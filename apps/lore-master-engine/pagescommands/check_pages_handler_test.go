package pagescommands

import (
	"context"
	"strings"
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

	if after.NewPages != 0 || len(after.Warnings) != 0 {
		t.Fatalf("a published site has no new pages: %+v", after)
	}

	write(t, work, "readme.md", "# Home\n\nWelcome back.")
	edited := check(t, conn, work)
	if edited.UpToDate || len(edited.Changes) == 0 {
		t.Fatalf("an edited page should be pending: %+v", edited)
	}
	if edited.NewPages != 0 {
		t.Fatalf("editing a published page does not make it new: %+v", edited)
	}
}

func TestCheckPagesFlagsPagesThatWouldBecomePublic(t *testing.T) {
	conn := connect(t)
	work := pagesWorkspace(t, pagesConfig)

	first := check(t, conn, work)
	if first.NewPages == 0 || len(first.NewPagePaths) == 0 {
		t.Fatalf("a site never published makes everything public: %+v", first)
	}
	if len(first.Warnings) == 0 || !strings.Contains(first.Warnings[len(first.Warnings)-1], "new to this public output") {
		t.Fatalf("the first publish should warn: %v", first.Warnings)
	}
	if _, err := publish(t, conn, work, 0); err != nil {
		t.Fatal(err)
	}

	write(t, work, "secret-plans.md", "# Secret plans\n\nNot meant for everyone.")
	added := check(t, conn, work)

	if added.NewPages != 1 || len(added.NewPagePaths) != 1 || !strings.Contains(added.NewPagePaths[0], "Secret") && !strings.Contains(added.NewPagePaths[0], "secret") {
		t.Fatalf("one new page should be flagged: %+v", added)
	}
	if !strings.Contains(strings.Join(added.Warnings, "\n"), "1 page is new to this public output") {
		t.Fatalf("the warning should name it: %v", added.Warnings)
	}
}
