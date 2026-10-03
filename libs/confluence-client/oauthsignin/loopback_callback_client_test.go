package oauthsignin

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func browserHit(url string) {
	response, err := http.Get(url) //nolint:noctx // a test standing in for the user's browser
	if err == nil {
		_ = response.Body.Close()
	}
}

func TestLoopbackReceiverReturnsTheCode(t *testing.T) {
	receiver, err := ListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = receiver.Close() }()

	if !strings.HasPrefix(receiver.RedirectURI(), "http://127.0.0.1:") {
		t.Fatalf("the redirect must be a 127.0.0.1 loopback (RFC 8252), got %q", receiver.RedirectURI())
	}
	go browserHit(receiver.RedirectURI() + "?code=abc123&state=xyz")

	code, err := receiver.Wait(context.Background(), "xyz")
	if err != nil || code != "abc123" {
		t.Fatalf("code %q, err %v", code, err)
	}
}

func TestLoopbackRejectsAMismatchedState(t *testing.T) {
	receiver, err := ListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = receiver.Close() }()
	go browserHit(receiver.RedirectURI() + "?code=abc&state=forged")

	if _, err := receiver.Wait(context.Background(), "expected"); err == nil || !strings.Contains(err.Error(), "state did not match") {
		t.Fatalf("a forged state must be rejected, got %v", err)
	}
}

func TestLoopbackReportsAProviderError(t *testing.T) {
	receiver, err := ListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = receiver.Close() }()
	go browserHit(receiver.RedirectURI() + "?error=access_denied")

	if _, err := receiver.Wait(context.Background(), "s"); err == nil || !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("a provider error must surface, got %v", err)
	}
}

func TestLoopbackWaitHonoursTheContext(t *testing.T) {
	receiver, err := ListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = receiver.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := receiver.Wait(ctx, "s"); err == nil {
		t.Fatal("a cancelled context must stop the wait")
	}
}
