package credentialvault

import (
	"encoding/json"
	"errors"
	"slices"
	"sync"

	"github.com/zalando/go-keyring"
)

// service is the name the entries are filed under in the keychain.
const service = "LoreMaster"

// indexAccount holds the list of remembered sites, because keychains cannot be enumerated.
const indexAccount = "connections-index"

// OSKeychain is a Vault over the operating system's keychain.
type OSKeychain struct {
	mu sync.Mutex
}

var _ Vault = (*OSKeychain)(nil)

// NewOSKeychain returns the keychain-backed vault.
func NewOSKeychain() *OSKeychain {
	return &OSKeychain{}
}

// Store implements Vault.
func (k *OSKeychain) Store(entry Entry) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	key := Key(entry.BaseURL)
	if err := keyring.Set(service, key, string(data)); err != nil {
		return wrap(err)
	}
	index, err := k.index()
	if err != nil {
		return err
	}
	if !slices.Contains(index, key) {
		return k.writeIndex(append(index, key))
	}

	return nil
}

// Get implements Vault.
func (k *OSKeychain) Get(baseURL string) (Entry, bool, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	return k.get(Key(baseURL))
}

func (k *OSKeychain) get(key string) (Entry, bool, error) {
	text, err := keyring.Get(service, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, wrap(err)
	}
	var entry Entry
	if err := json.Unmarshal([]byte(text), &entry); err != nil {
		return Entry{}, false, nil
	}

	return entry, true, nil
}

// Delete implements Vault.
func (k *OSKeychain) Delete(baseURL string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	key := Key(baseURL)
	if err := keyring.Delete(service, key); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return wrap(err)
	}
	index, err := k.index()
	if err != nil {
		return err
	}
	if slices.Contains(index, key) {
		return k.writeIndex(slices.DeleteFunc(index, func(each string) bool { return each == key }))
	}

	return nil
}

// List implements Vault.
func (k *OSKeychain) List() ([]Entry, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	index, err := k.index()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(index))
	for _, key := range index {
		entry, found, err := k.get(key)
		if err != nil {
			return nil, err
		}
		if found {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (k *OSKeychain) index() ([]string, error) {
	text, err := keyring.Get(service, indexAccount)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, wrap(err)
	}
	var keys []string
	if err := json.Unmarshal([]byte(text), &keys); err != nil {
		return nil, nil
	}

	return keys, nil
}

func (k *OSKeychain) writeIndex(keys []string) error {
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	return wrap(keyring.Set(service, indexAccount, string(data)))
}

// wrap tells "there is no keychain here" apart from a real failure.
func wrap(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrUnsupportedPlatform) {
		return ErrUnavailable
	}

	return err
}
