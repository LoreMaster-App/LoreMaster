package sessionlifecycle

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

// Store holds the open sessions. One engine serves several at once: a multi-root
// workspace may sync to more than one site.
type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

// NewStore makes an empty store.
func NewStore() *Store {
	return &Store{sessions: map[string]*Session{}}
}

// Add keeps a session under a new random id and returns it.
func (s *Store) Add(session Session) *Session {
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	session.ID = hex.EncodeToString(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = &session

	return &session
}

// Get returns the session, or an unknown-session error the editor can act on by
// opening a new one.
func (s *Store) Get(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, rpcprotocol.Errorf(rpcprotocol.CodeUnknownSession, "no open session %q; open one again", id)
	}

	return session, nil
}

// Remove forgets the session. Go cannot overwrite a string in place, so the credential
// is not zeroed byte by byte: every reference to it is dropped, and it is unreachable
// from then on.
func (s *Store) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[id]; ok {
		session.Platform = nil
		delete(s.sessions, id)
	}
}
