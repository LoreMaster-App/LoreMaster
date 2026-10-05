package sessionlifecycle

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

const secret = "pat-s3cr3t-value"

type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// engine serves the session methods against a fake Data Center 8.5.4 site that accepts
// only the PAT secret, and returns the editor's end, the store and the whole log.
func engine(t *testing.T) (*jsonrpc2.Conn, *Store, *logBuffer, string) {
	t.Helper()
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/applinks/1.0/manifest":
			_, _ = w.Write([]byte(`<manifest><typeId>confluence</typeId><version>8.5.4</version></manifest>`))
		case "/rest/api/user/current":
			if r.Header.Get("Authorization") != "Bearer "+secret {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}
			_, _ = w.Write([]byte(`{"type":"known","username":"ada","userKey":"k1","displayName":"Ada Lovelace"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(site.Close)

	log := &logBuffer{}
	logger := slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: slog.LevelDebug}))
	store := NewStore()
	environment := Environment{HTTPClient: site.Client(), Logger: logger}
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodSessionOpen:  OpenSession(store, environment),
			rpcprotocol.MethodSessionClose: CloseSession(store),
		}, logger)
	}()
	editor := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}), jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (any, error) { return nil, nil })))
	t.Cleanup(func() { _ = editor.Close() })

	return editor, store, log, site.URL
}

func code(err error) int64 {
	var wire *jsonrpc2.Error
	if errors.As(err, &wire) {
		return wire.Code
	}

	return 0
}

func TestOpenVerifiesAndKeepsTheSessionInMemory(t *testing.T) {
	editor, store, log, siteURL := engine(t)
	var opened rpcprotocol.SessionOpenResult
	err := editor.Call(context.Background(), rpcprotocol.MethodSessionOpen, rpcprotocol.SessionOpenParams{
		BaseURL: siteURL, Credential: rpcprotocol.Credential{Kind: "pat", Token: secret},
	}, &opened)
	if err != nil {
		t.Fatal(err)
	}
	if opened.SessionID == "" || opened.Edition != "datacenter" || opened.Version != "8.5.4" ||
		opened.User != (rpcprotocol.SessionUser{DisplayName: "Ada Lovelace", Username: "ada"}) {
		t.Fatalf("%+v", opened)
	}
	session, err := store.Get(opened.SessionID)
	if err != nil || session.Platform == nil || session.BaseURL != siteURL {
		t.Fatalf("%+v %v", session, err)
	}

	if err := editor.Call(context.Background(), rpcprotocol.MethodSessionClose, rpcprotocol.SessionCloseParams{SessionID: opened.SessionID}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(opened.SessionID); code(asWire(err)) != rpcprotocol.CodeUnknownSession {
		t.Fatalf("closed: %v", err)
	}
	if session.Platform != nil {
		t.Fatal("closing drops the session's connection, credential included")
	}
	if err := editor.Call(context.Background(), rpcprotocol.MethodSessionClose, rpcprotocol.SessionCloseParams{SessionID: opened.SessionID}, nil); err != nil {
		t.Fatalf("closing twice is fine: %v", err)
	}

	if !strings.Contains(log.String(), "method=session/open") || strings.Contains(log.String(), secret) {
		t.Fatalf("the log records the request, never the secret:\n%s", log.String())
	}
}

func TestOpenTellsTheEditorWhatToFix(t *testing.T) {
	editor, store, log, siteURL := engine(t)
	cases := map[string]struct {
		params rpcprotocol.SessionOpenParams
		code   int64
	}{
		"wrong token":       {rpcprotocol.SessionOpenParams{BaseURL: siteURL, Credential: rpcprotocol.Credential{Kind: "pat", Token: "pat-wrong-" + secret}}, rpcprotocol.CodeUnauthorized},
		"cloud credential":  {rpcprotocol.SessionOpenParams{BaseURL: siteURL, Credential: rpcprotocol.Credential{Kind: "apitoken", Email: "a@b.c", Token: secret}}, rpcprotocol.CodeUnsupported},
		"missing token":     {rpcprotocol.SessionOpenParams{BaseURL: siteURL, Credential: rpcprotocol.Credential{Kind: "pat"}}, rpcprotocol.CodeInvalidParams},
		"not an address":    {rpcprotocol.SessionOpenParams{BaseURL: "::", Credential: rpcprotocol.Credential{Kind: "pat", Token: secret}}, rpcprotocol.CodeInvalidParams},
		"nothing listening": {rpcprotocol.SessionOpenParams{BaseURL: "http://127.0.0.1:1", Credential: rpcprotocol.Credential{Kind: "pat", Token: secret}}, rpcprotocol.CodePlatformUnreachable},
	}
	for name, c := range cases {
		err := editor.Call(context.Background(), rpcprotocol.MethodSessionOpen, c.params, nil)
		if code(err) != c.code || strings.Contains(err.Error(), secret) {
			t.Errorf("%s: %v (code %d)", name, err, code(err))
		}
	}
	if len(store.sessions) != 0 || strings.Contains(log.String(), secret) {
		t.Fatalf("no session kept, no secret logged: %d\n%s", len(store.sessions), log.String())
	}
}

func asWire(err error) error {
	var coded *rpcprotocol.Error
	if errors.As(err, &coded) {
		return &jsonrpc2.Error{Code: coded.Code, Message: coded.Message}
	}

	return err
}
