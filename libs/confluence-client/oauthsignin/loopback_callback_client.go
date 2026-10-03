package oauthsignin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// LoopbackReceiver waits on a loopback TCP port for the browser redirect that ends the
// OAuth flow and hands back the authorization code. Per RFC 8252 §7.3 it binds
// "127.0.0.1" (not "localhost", which some providers' redirect allow-lists reject) on an
// OS-assigned port.
type LoopbackReceiver struct {
	listener net.Listener
	server   *http.Server
	result   chan callback
}

type callback struct {
	code  string
	state string
	err   error
}

// ListenLoopback binds a loopback port and starts serving the callback. Close it when the
// flow ends, whether it succeeded or not.
func ListenLoopback() (*LoopbackReceiver, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("could not open a loopback port for the sign-in redirect: %w", err)
	}
	receiver := &LoopbackReceiver{listener: listener, result: make(chan callback, 1)}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", receiver.handle)
	receiver.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = receiver.server.Serve(listener) }()

	return receiver, nil
}

// RedirectURI is the address to register as the OAuth redirect and send the browser back
// to.
func (r *LoopbackReceiver) RedirectURI() string {
	return "http://" + r.listener.Addr().String() + "/callback"
}

// Wait blocks until the browser hits the callback, the context is done, or the redirect
// carries a provider error. It rejects a redirect whose state does not match, which would
// be a forged request (CSRF).
func (r *LoopbackReceiver) Wait(ctx context.Context, state string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case cb := <-r.result:
		if cb.err != nil {
			return "", cb.err
		}
		if cb.state != state {
			return "", errors.New("the sign-in redirect's state did not match the request; it may have been forged")
		}

		return cb.code, nil
	}
}

// Close shuts the loopback server down.
func (r *LoopbackReceiver) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return r.server.Shutdown(ctx)
}

func (r *LoopbackReceiver) handle(w http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	cb := callback{code: query.Get("code"), state: query.Get("state")}
	message := "You are signed in to Confluence. You can close this tab and return to your editor."
	switch {
	case query.Get("error") != "":
		cb = callback{err: fmt.Errorf("the authorization server refused the sign-in: %s", query.Get("error"))}
		message = "Sign-in failed (" + query.Get("error") + "). You can close this tab."
	case cb.code == "":
		cb = callback{err: errors.New("the sign-in redirect carried no authorization code")}
		message = "Sign-in failed: no authorization code was returned. You can close this tab."
	}
	// The channel is buffered for one; a second hit (a reload) is dropped, not blocked.
	select {
	case r.result <- cb:
	default:
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, "<!doctype html><meta charset=\"utf-8\"><title>Lore Master</title><p>"+message+"</p>")
}
