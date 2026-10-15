package credentialvault

import (
	"errors"
	"strings"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

// Entry is one remembered connection.
type Entry = rpcprotocol.VaultEntry

// ErrUnavailable says the machine has no keychain to keep anything in (a Linux box with no
// Secret Service, say). Nothing is lost by it: the credential is simply not remembered.
var ErrUnavailable = errors.New("this machine has no keychain to remember credentials in")

// Vault remembers connections by the site's address.
type Vault interface {
	// Store writes or replaces the connection for entry.BaseURL.
	Store(entry Entry) error
	// Get returns the connection for a site, and whether there is one.
	Get(baseURL string) (Entry, bool, error)
	// Delete forgets a site; one that is not there is not an error.
	Delete(baseURL string) error
	// List returns every remembered connection.
	List() ([]Entry, error)
}

// Key is the canonical form of a site's address, so a trailing slash or a different case
// finds the same connection.
func Key(baseURL string) string {
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
}
