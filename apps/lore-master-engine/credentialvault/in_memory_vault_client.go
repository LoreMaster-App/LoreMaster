package credentialvault

import (
	"slices"
	"strings"
	"sync"
)

// InMemory is a Vault held in memory, for tests and for machines the user has not opted in on.
type InMemory struct {
	mu      sync.Mutex
	entries map[string]Entry
}

var _ Vault = (*InMemory)(nil)

// NewInMemory returns an empty vault.
func NewInMemory() *InMemory {
	return &InMemory{entries: map[string]Entry{}}
}

// Store implements Vault.
func (m *InMemory) Store(entry Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[Key(entry.BaseURL)] = entry

	return nil
}

// Get implements Vault.
func (m *InMemory) Get(baseURL string) (Entry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, found := m.entries[Key(baseURL)]

	return entry, found, nil
}

// Delete implements Vault.
func (m *InMemory) Delete(baseURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, Key(baseURL))

	return nil
}

// List implements Vault.
func (m *InMemory) List() ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries := make([]Entry, 0, len(m.entries))
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(a, b Entry) int { return strings.Compare(a.BaseURL, b.BaseURL) })

	return entries, nil
}
