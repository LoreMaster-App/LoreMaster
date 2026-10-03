package oauthsignin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewPKCEDerivesTheS256Challenge(t *testing.T) {
	pkce, err := NewPKCE(bytes.NewReader(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	wantVerifier := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	sum := sha256.Sum256([]byte(wantVerifier))
	wantChallenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if pkce.Verifier != wantVerifier || pkce.Challenge != wantChallenge {
		t.Fatalf("pkce %+v", pkce)
	}
	if pkce.Challenge == pkce.Verifier {
		t.Fatal("the challenge must be the hash, never the verifier itself")
	}
}

func TestAuthorizeURLCarriesThePKCEChallengeAndState(t *testing.T) {
	raw := AuthorizeURL(AuthorizeRequest{
		BaseURL: "https://dc.example/confluence/", ClientID: "cid", RedirectURI: "http://127.0.0.1:5000/callback",
		Scope: "WRITE", State: "st", Challenge: "ch",
	})
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(parsed.Path, authorizePath) {
		t.Fatalf("path %q", parsed.Path)
	}
	query := parsed.Query()
	for key, want := range map[string]string{
		"client_id": "cid", "response_type": "code", "redirect_uri": "http://127.0.0.1:5000/callback",
		"state": "st", "code_challenge": "ch", "code_challenge_method": "S256", "scope": "WRITE",
	} {
		if query.Get(key) != want {
			t.Errorf("%s = %q, want %q", key, query.Get(key), want)
		}
	}
}

func TestExchangeCodeReturnsTokensAndSendsTheVerifier(t *testing.T) {
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/confluence"+tokenPath {
			t.Errorf("path %s", r.URL.Path)
		}
		_ = r.ParseForm()
		form = r.Form
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "refresh_token": "rt", "expires_in": 3600})
	}))
	defer server.Close()

	tokens, err := ExchangeCode(context.Background(), server.Client(), ExchangeParams{
		BaseURL: server.URL + "/confluence", ClientID: "cid", Code: "code1", RedirectURI: "http://127.0.0.1:1/callback", Verifier: "ver",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "at" || tokens.RefreshToken != "rt" || tokens.ExpiresIn != 3600 {
		t.Fatalf("tokens %+v", tokens)
	}
	for key, want := range map[string]string{
		"grant_type": "authorization_code", "code": "code1", "code_verifier": "ver", "client_id": "cid",
	} {
		if form.Get(key) != want {
			t.Errorf("form %s = %q, want %q", key, form.Get(key), want)
		}
	}
}

func TestRefreshMapsInvalidGrantToReauthRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type %s", r.Form.Get("grant_type"))
		}
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant", "error_description": "token revoked"})
	}))
	defer server.Close()

	_, err := Refresh(context.Background(), server.Client(), server.URL, "cid", "old")
	var reauth *ReauthRequired
	if !errors.As(err, &reauth) || !strings.Contains(reauth.Reason, "revoked") {
		t.Fatalf("want *ReauthRequired, got %T %v", err, err)
	}
}

func TestRefreshKeepsOtherFailuresGeneric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := Refresh(context.Background(), server.Client(), server.URL, "cid", "old")
	var reauth *ReauthRequired
	if err == nil || errors.As(err, &reauth) {
		t.Fatalf("a 500 is not a re-auth, got %v", err)
	}
}

func TestSignInRunsThePKCEFlowEndToEnd(t *testing.T) {
	challenge := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(authorizePath, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		challenge <- query.Get("code_challenge")
		http.Redirect(w, r, query.Get("redirect_uri")+"?code=THECODE&state="+query.Get("state"), http.StatusFound)
	})
	mux.HandleFunc(tokenPath, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != <-challenge {
			t.Error("the verifier presented at the token endpoint does not match the challenge sent to authorize")
		}
		if r.Form.Get("code") != "THECODE" {
			t.Errorf("code %s", r.Form.Get("code"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "refresh_token": "rt", "expires_in": 60})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var opened string
	tokens, err := SignIn(context.Background(), SignInRequest{
		BaseURL: server.URL, ClientID: "cid", Scope: "WRITE", HTTPClient: server.Client(),
		OpenBrowser: func(authorizeURL string) error {
			opened = authorizeURL
			response, err := server.Client().Get(authorizeURL)
			if err != nil {
				return err
			}
			_ = response.Body.Close()

			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "at" || tokens.RefreshToken != "rt" {
		t.Fatalf("tokens %+v", tokens)
	}
	if !strings.Contains(opened, "code_challenge=") || strings.Contains(opened, "code_verifier") {
		t.Fatalf("the browser is given the challenge, never the verifier: %q", opened)
	}
}

func TestSignInNeedsABrowser(t *testing.T) {
	if _, err := SignIn(context.Background(), SignInRequest{BaseURL: "https://x", ClientID: "c"}); err == nil {
		t.Fatal("sign-in without OpenBrowser must fail")
	}
}
