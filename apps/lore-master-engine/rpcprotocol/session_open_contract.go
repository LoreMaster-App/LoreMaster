package rpcprotocol

// MethodSessionOpen connects to a documentation platform: the edition is detected when
// not given, the credential is verified, and the session keeps it in memory only.
const MethodSessionOpen = "session/open"

// SessionOpenParams opens a session.
type SessionOpenParams struct {
	// BaseURL is the site's address; any page URL of the site is accepted too.
	BaseURL string `json:"baseUrl"`
	// Edition is "cloud", "datacenter" or "server"; empty detects it.
	Edition    string     `json:"edition,omitempty"`
	Credential Credential `json:"credential"`
}

// Credential is how the user signs in. Kind decides which fields are read:
// "apitoken" (Cloud) reads Email and Token, "pat" reads Token, "basic" reads User and
// Password, "oauth" (Data Center) reads AccessToken when the editor already holds one, or
// ClientID (and optional Scope) to start a fresh interactive browser sign-in whose tokens
// come back in SessionOpenResult.Tokens.
type Credential struct {
	Kind        string `json:"kind"`
	Email       string `json:"email,omitempty"`
	Token       string `json:"token,omitempty"`
	User        string `json:"user,omitempty"`
	Password    string `json:"password,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`
	ClientID    string `json:"clientId,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

// SessionOpenResult describes the open session.
type SessionOpenResult struct {
	SessionID string `json:"sessionId"`
	// BaseURL is the normalised site address the session talks to.
	BaseURL string `json:"baseUrl"`
	Edition string `json:"edition"`
	// Version is the server version on Data Center and Server ("8.5.4"); empty on Cloud.
	Version string      `json:"version,omitempty"`
	User    SessionUser `json:"user"`
	// Tokens is set only when this open ran an interactive OAuth sign-in: the editor stores
	// them in its secret store to open later sessions and to refresh. The engine keeps none.
	Tokens *SessionTokens `json:"tokens,omitempty"`
}

// SessionTokens are the OAuth tokens from an interactive sign-in.
type SessionTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresIn    int    `json:"expiresIn,omitempty"`
}

// SessionUser is who the credential signs in as.
type SessionUser struct {
	DisplayName string `json:"displayName"`
	// AccountID is set on Cloud, Username on Data Center and Server.
	AccountID string `json:"accountId,omitempty"`
	Username  string `json:"username,omitempty"`
}
