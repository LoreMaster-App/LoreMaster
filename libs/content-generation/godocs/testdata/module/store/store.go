// Package store keeps key/value pairs in memory.
package store

import "sync"

// ErrMissing is returned when a key is not in the store.
var ErrMissing = missing{}

type missing struct{}

func (missing) Error() string { return "missing" }

// Store is safe for concurrent use.
type Store struct {
	mu   sync.Mutex
	data map[string]string
}

// New returns an empty Store.
func New() *Store { return &Store{data: map[string]string{}} }

// Get returns the value for key, or [ErrMissing].
func (s *Store) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.data[key]
	if !ok {
		return "", ErrMissing
	}
	return value, nil
}

// Set stores value under key.
func (s *Store) Set(key, value string) { s.data[key] = value }
