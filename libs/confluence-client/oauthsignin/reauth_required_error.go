package oauthsignin

// ReauthRequired means the OAuth sign-in is no longer valid — the user revoked access, or
// the refresh token expired — so the person must sign in again. Callers surface it as a
// re-authentication prompt rather than a plain failure.
type ReauthRequired struct {
	// Reason is the provider's explanation, when it gave one.
	Reason string
}

func (e *ReauthRequired) Error() string {
	if e.Reason == "" {
		return "the Confluence sign-in is no longer valid; sign in again"
	}

	return "the Confluence sign-in is no longer valid (" + e.Reason + "); sign in again"
}
