package pty

import (
	"sort"
	"sync"
)

// SessionRegistry is a thread-safe, in-memory store for active sessions.
// Sessions persist in the registry independent of any UI or WebSocket lifecycle.
type SessionRegistry struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionRegistry returns an initialised, empty SessionRegistry.
func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{
		sessions: make(map[string]*Session),
	}
}

// Add stores the session under its ID. If a session with the same ID already
// exists it is replaced.
func (r *SessionRegistry) Add(s *Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.ID] = s
}

// Get returns the session for the given ID and whether it was found.
func (r *SessionRegistry) Get(id string) (*Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[id]
	return s, ok
}

// List returns a snapshot of all sessions currently in the registry,
// sorted by creation time (oldest first) for stable display order.
func (r *SessionRegistry) List() []*Session {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Session, 0, len(r.sessions))
	for _, s := range r.sessions {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID // stable tiebreaker
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

// Remove deletes the session with the given ID from the registry.
// Returns true if the session existed and was removed, false otherwise.
func (r *SessionRegistry) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.sessions[id]
	if ok {
		delete(r.sessions, id)
	}
	return ok
}

// Len returns the number of sessions currently in the registry.
func (r *SessionRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}

// KillAll marks every session as StateStopped and clears the registry.
// The actual process teardown is wired in Plan 02; this implementation
// updates state and removes entries so callers get a clean slate.
func (r *SessionRegistry) KillAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sessions {
		s.mu.Lock()
		s.State = StateStopped
		s.mu.Unlock()
	}
	r.sessions = make(map[string]*Session)
}
