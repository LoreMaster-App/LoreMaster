package sessionlifecycle

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestOpenRunsTheInteractiveOAuthSignIn(t *testing.T) {
	// A fake Data Center site that also serves the OAuth 2.0 provider. Its token endpoint
	// hands back an access token equal to the secret the site accepts, so the sign-in flows
	// straight into a verified session.
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/oauth2/latest/authorize":
			query := r.URL.Query()
			http.Redirect(w, r, query.Get("redirect_uri")+"?code=THECODE&state="+query.Get("state"), http.StatusFound)
		case "/rest/oauth2/latest/token":
			_ = r.ParseForm()
			if r.Form.Get("code") != "THECODE" || r.Form.Get("code_verifier") == "" {
				w.WriteHeader(http.StatusBadRequest)

				return
			}
			_, _ = w.Write([]byte(`{"access_token":"` + secret + `","refresh_token":"refresh-1","expires_in":3600}`))
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

	store := NewStore()
	log := &logBuffer{}
	logger := slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: slog.LevelDebug}))
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodSessionOpen: OpenSession(store, Environment{HTTPClient: site.Client(), Logger: logger}),
		}, logger)
	}()
	// The editor "opens the browser" by fetching the authorize URL, which redirects to the
	// engine's loopback and delivers the code.
	editor := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(func(_ context.Context, _ *jsonrpc2.Conn, request *jsonrpc2.Request) (any, error) {
			if request.Method != rpcprotocol.MethodHostOpenExternal || request.Params == nil {
				return nil, nil
			}
			var params rpcprotocol.OpenExternalParams
			_ = json.Unmarshal(*request.Params, &params)
			response, err := site.Client().Get(params.URL)
			if err != nil {
				return nil, err
			}
			_ = response.Body.Close()

			return rpcprotocol.OpenExternalResult{}, nil
		})))
	t.Cleanup(func() { _ = editor.Close() })

	var opened rpcprotocol.SessionOpenResult
	err := editor.Call(context.Background(), rpcprotocol.MethodSessionOpen, rpcprotocol.SessionOpenParams{
		BaseURL: site.URL, Credential: rpcprotocol.Credential{Kind: "oauth", ClientID: "cid", Scope: "WRITE"},
	}, &opened)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Edition != "datacenter" || opened.User.DisplayName != "Ada Lovelace" || opened.SessionID == "" {
		t.Fatalf("session not verified: %+v", opened)
	}
	if opened.Tokens == nil || opened.Tokens.AccessToken != secret || opened.Tokens.RefreshToken != "refresh-1" || opened.Tokens.ExpiresIn != 3600 {
		t.Fatalf("the interactive sign-in returns the tokens for the editor to store: %+v", opened.Tokens)
	}
	if strings.Contains(log.String(), secret) || strings.Contains(log.String(), "refresh-1") {
		t.Fatalf("no token in the log:\n%s", log.String())
	}
}

// refreshSite is a fake Data Center that accepts only freshToken for the current user, and
// whose token endpoint renews goodRefresh into freshToken but refuses any other refresh
// token with invalid_grant.
func refreshSite(t *testing.T, freshToken string, goodRefresh string) *httptest.Server {
	t.Helper()
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/applinks/1.0/manifest":
			_, _ = w.Write([]byte(`<manifest><typeId>confluence</typeId><version>8.5.4</version></manifest>`))
		case "/rest/oauth2/latest/token":
			_ = r.ParseForm()
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != goodRefresh {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"revoked"}`))

				return
			}
			_, _ = w.Write([]byte(`{"access_token":"` + freshToken + `","refresh_token":"rt-new","expires_in":3600}`))
		case "/rest/api/user/current":
			if r.Header.Get("Authorization") != "Bearer "+freshToken {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}
			_, _ = w.Write([]byte(`{"type":"known","username":"ada","userKey":"k1","displayName":"Ada Lovelace"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(site.Close)

	return site
}

func sessionEngine(t *testing.T, site *httptest.Server) (*jsonrpc2.Conn, *logBuffer) {
	t.Helper()
	log := &logBuffer{}
	logger := slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: slog.LevelDebug}))
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodSessionOpen: OpenSession(NewStore(), Environment{HTTPClient: site.Client(), Logger: logger}),
		}, logger)
	}()
	editor := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (any, error) { return nil, nil })))
	t.Cleanup(func() { _ = editor.Close() })

	return editor, log
}

func TestOpenRefreshesAnExpiredOAuthTokenAndRetries(t *testing.T) {
	const fresh = "access-token-fresh"
	site := refreshSite(t, fresh, "rt-good")
	editor, log := sessionEngine(t, site)

	var opened rpcprotocol.SessionOpenResult
	err := editor.Call(context.Background(), rpcprotocol.MethodSessionOpen, rpcprotocol.SessionOpenParams{
		BaseURL:    site.URL,
		Credential: rpcprotocol.Credential{Kind: "oauth", AccessToken: "stale", RefreshToken: "rt-good", ClientID: "cid"},
	}, &opened)
	if err != nil {
		t.Fatal(err)
	}
	if opened.User.DisplayName != "Ada Lovelace" || opened.SessionID == "" {
		t.Fatalf("the stale token is renewed and the session opens: %+v", opened)
	}
	if opened.Tokens == nil || opened.Tokens.AccessToken != fresh || opened.Tokens.RefreshToken != "rt-new" {
		t.Fatalf("the refreshed tokens come back for the editor to re-store: %+v", opened.Tokens)
	}
	if strings.Contains(log.String(), fresh) || strings.Contains(log.String(), "rt-good") {
		t.Fatalf("no token in the log:\n%s", log.String())
	}
}

func TestOpenReportsReauthWhenTheRefreshTokenIsRevoked(t *testing.T) {
	site := refreshSite(t, "whatever", "rt-good")
	editor, _ := sessionEngine(t, site)

	err := editor.Call(context.Background(), rpcprotocol.MethodSessionOpen, rpcprotocol.SessionOpenParams{
		BaseURL:    site.URL,
		Credential: rpcprotocol.Credential{Kind: "oauth", AccessToken: "stale", RefreshToken: "rt-revoked", ClientID: "cid"},
	}, nil)
	if code(err) != rpcprotocol.CodeReauthRequired {
		t.Fatalf("a revoked refresh token asks for re-auth, got code %d: %v", code(err), err)
	}
}

func asWire(err error) error {
	var coded *rpcprotocol.Error
	if errors.As(err, &coded) {
		return &jsonrpc2.Error{Code: coded.Code, Message: coded.Message}
	}

	return err
}
