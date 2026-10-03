// Package oauthsignin runs the Confluence Data Center OAuth 2.0 sign-in: the
// authorization-code flow with PKCE (RFC 7636) and a loopback redirect (RFC 8252). It
// generates the verifier and challenge, builds the authorize URL, waits on a 127.0.0.1
// port for the browser redirect, exchanges the code for tokens, refreshes them, and
// reports a revoked or expired sign-in as *ReauthRequired.
//
// It holds nothing: callers store the tokens in the editor's secret store and wrap the
// access token as an authentication.OAuth credential to sign requests. Cloud is not here —
// Cloud stays API-token only (docs/architecture/oauth.md). This slice does I/O, which is
// why it is separate from the pure authentication package.
package oauthsignin
