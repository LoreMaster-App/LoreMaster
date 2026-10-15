package rpcprotocol

// The credential vault is the operating system's keychain, which the user opts in to: a
// connection remembered there is available to every editor profile and to the command line.

// MethodCredentialStore remembers a connection.
const MethodCredentialStore = "credential/store"

// MethodCredentialGet returns the remembered connection for a site, if any.
const MethodCredentialGet = "credential/get"

// MethodCredentialDelete forgets a site's connection.
const MethodCredentialDelete = "credential/delete"

// MethodCredentialList returns every remembered connection.
const MethodCredentialList = "credential/list"

// VaultEntry is one remembered connection: the site, who signs in, and the credential.
type VaultEntry struct {
	BaseURL     string `json:"baseUrl"`
	Edition     string `json:"edition,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	// User is the account id (Cloud) or username (Data Center, Server) the credential signs in as.
	User       string     `json:"user,omitempty"`
	Credential Credential `json:"credential"`
}

// CredentialStoreParams carries the connection to remember.
type CredentialStoreParams struct {
	Entry VaultEntry `json:"entry"`
}

// CredentialGetParams names the site.
type CredentialGetParams struct {
	BaseURL string `json:"baseUrl"`
}

// CredentialGetResult is the remembered connection, or Found false.
type CredentialGetResult struct {
	Found bool        `json:"found"`
	Entry *VaultEntry `json:"entry,omitempty"`
}

// CredentialDeleteParams names the site to forget.
type CredentialDeleteParams struct {
	BaseURL string `json:"baseUrl"`
}

// CredentialListResult is every remembered connection.
type CredentialListResult struct {
	Entries []VaultEntry `json:"entries"`
}
