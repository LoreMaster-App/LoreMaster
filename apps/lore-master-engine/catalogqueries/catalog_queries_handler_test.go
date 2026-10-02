package catalogqueries

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// fake is the in-memory platform as a session platform.
type fake struct{ *platformport.InMemoryPlatform }

func (f fake) ForOutput(workspacesettings.Output) platformport.DocumentationPlatform { return f }

// engine serves the catalog over a pipe, with one session on a fake platform.
func engine(t *testing.T) (*jsonrpc2.Conn, string, map[string]string) {
	t.Helper()
	platform := platformport.NewInMemoryPlatform(
		platformport.Space{ID: "1", Key: "ENG", Name: "Engineering"},
		platformport.Space{ID: "2", Key: "OPS", Name: "Operations"},
	)
	ids := map[string]string{}
	ids["home"] = platform.SeedPage("ENG", "", "Engineering Home")
	ids["guide"] = platform.SeedPage("ENG", ids["home"], "Setup guide")
	ids["arch"] = platform.SeedPage("ENG", ids["home"], "Architecture")
	for i := range 250 {
		platform.SeedPage("ENG", ids["arch"], fmt.Sprintf("Decision %03d", i))
	}
	sessions := sessionlifecycle.NewStore()
	session := sessions.Add(sessionlifecycle.Session{Platform: fake{platform}})

	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodSpaceList:    ListSpaces(sessions),
			rpcprotocol.MethodPageChildren: ListChildren(sessions),
			rpcprotocol.MethodPageSearch:   SearchPages(sessions),
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	editor := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}), jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (any, error) { return nil, nil })))
	t.Cleanup(func() { _ = editor.Close() })

	return editor, session.ID, ids
}

func code(err error) int64 {
	var wire *jsonrpc2.Error
	if errors.As(err, &wire) {
		return wire.Code
	}

	return 0
}

func titles(result rpcprotocol.PagesResult) []string {
	var out []string
	for _, page := range result.Pages {
		out = append(out, page.Title)
	}

	return out
}

func TestSpaceList(t *testing.T) {
	editor, session, _ := engine(t)
	var result rpcprotocol.SpaceListResult
	if err := editor.Call(context.Background(), rpcprotocol.MethodSpaceList, rpcprotocol.SpaceListParams{SessionID: session}, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Spaces) != 2 || result.Spaces[0] != (rpcprotocol.Space{ID: "1", Key: "ENG", Name: "Engineering"}) {
		t.Fatalf("%+v", result.Spaces)
	}
}

func TestPageChildren(t *testing.T) {
	editor, session, ids := engine(t)
	var result rpcprotocol.PagesResult
	if err := editor.Call(context.Background(), rpcprotocol.MethodPageChildren, rpcprotocol.PageChildrenParams{SessionID: session, PageID: ids["home"]}, &result); err != nil {
		t.Fatal(err)
	}
	if got := titles(result); len(got) != 2 || result.Pages[0].ParentID != ids["home"] || result.Pages[0].URL == "" {
		t.Fatalf("%+v", result.Pages)
	}
	var many rpcprotocol.PagesResult
	if err := editor.Call(context.Background(), rpcprotocol.MethodPageChildren, rpcprotocol.PageChildrenParams{SessionID: session, PageID: ids["arch"]}, &many); err != nil || len(many.Pages) != 250 {
		t.Fatalf("every child, no paging left to the editor: %d %v", len(many.Pages), err)
	}
}

func TestPageSearch(t *testing.T) {
	editor, session, _ := engine(t)
	search := func(params rpcprotocol.PageSearchParams) rpcprotocol.PagesResult {
		t.Helper()
		params.SessionID = session
		var result rpcprotocol.PagesResult
		if err := editor.Call(context.Background(), rpcprotocol.MethodPageSearch, params, &result); err != nil {
			t.Fatal(err)
		}

		return result
	}
	if got := titles(search(rpcprotocol.PageSearchParams{SpaceKey: "ENG", Query: "GUIDE"})); len(got) != 1 || got[0] != "Setup guide" {
		t.Fatalf("%q", got)
	}
	if got := search(rpcprotocol.PageSearchParams{SpaceKey: "ENG", Query: "decision"}); len(got.Pages) != 50 {
		t.Fatalf("50 by default, got %d", len(got.Pages))
	}
	if got := search(rpcprotocol.PageSearchParams{SpaceKey: "ENG", Query: "decision", Limit: 1000}); len(got.Pages) != 200 {
		t.Fatalf("at most 200, got %d", len(got.Pages))
	}
	if got := search(rpcprotocol.PageSearchParams{SpaceKey: "ENG", Query: "decision", Limit: 5}); len(got.Pages) != 5 {
		t.Fatalf("5 asked, got %d", len(got.Pages))
	}
	if got := search(rpcprotocol.PageSearchParams{SpaceKey: "OPS"}); len(got.Pages) != 0 {
		t.Fatalf("other space: %d", len(got.Pages))
	}
}

func TestCatalogErrors(t *testing.T) {
	editor, session, _ := engine(t)
	cases := map[string]struct {
		method string
		params any
		code   int64
	}{
		"unknown session": {rpcprotocol.MethodSpaceList, rpcprotocol.SpaceListParams{SessionID: "gone"}, rpcprotocol.CodeUnknownSession},
		"missing page":    {rpcprotocol.MethodPageChildren, rpcprotocol.PageChildrenParams{SessionID: session, PageID: "p999"}, rpcprotocol.CodeNotFound},
		"no page id":      {rpcprotocol.MethodPageChildren, rpcprotocol.PageChildrenParams{SessionID: session}, rpcprotocol.CodeInvalidParams},
		"no space":        {rpcprotocol.MethodPageSearch, rpcprotocol.PageSearchParams{SessionID: session, Query: "x"}, rpcprotocol.CodeInvalidParams},
	}
	for name, c := range cases {
		if err := editor.Call(context.Background(), c.method, c.params, nil); code(err) != c.code {
			t.Errorf("%s: %v", name, err)
		}
	}
}
