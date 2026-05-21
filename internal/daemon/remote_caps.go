package daemon

// Phase 122-01 — Remote capability store.
//
// RemoteCapStore caches per-session (baseURL, capToken) tuples that the local
// daemon's `/api/files/remote/{sessionID}/...` proxy uses to reach a remote
// peer's webserver. The store lives in process memory only — there is no disk
// persistence (matches JoinCodeManager invariant). A daemon restart wipes the
// cache and the GUI/TUI must re-exchange the join code to repopulate.
//
// Security invariant: nothing in this file writes a cap token to disk. The
// companion test asserts this via a source-grep guard so that future edits
// cannot accidentally introduce persistence without also touching the test.

import (
	"errors"
	"sync"
)

// remoteCapEntry is the per-session (baseURL, capToken) tuple. baseURL is the
// scheme://host[:port] root of the remote peer's webserver (no trailing slash);
// capToken is the opaque HMAC-signed capability the remote webserver issued
// when the user exchanged the join code.
type remoteCapEntry struct {
	baseURL  string
	capToken string
}

// RemoteCapStore is a thread-safe in-memory map of sessionID → cap entry.
// Reads use the read side of the mutex (Get); writes (Put, Delete) use the
// write side. The zero value is NOT usable — call NewRemoteCapStore.
//
// Persistence is intentionally absent. The store rebuilds from user actions
// (paste join code → exchange → register) after every daemon restart; this
// matches the JoinCodeManager design and keeps the cap-token footprint out
// of any on-disk artefact (Threat T-122-02 in PLAN.md).
type RemoteCapStore struct {
	mu      sync.RWMutex
	entries map[string]remoteCapEntry
}

// NewRemoteCapStore constructs an empty store ready for use.
func NewRemoteCapStore() *RemoteCapStore {
	return &RemoteCapStore{entries: make(map[string]remoteCapEntry)}
}

// Put deposits (baseURL, capToken) for sessionID, overwriting any prior entry
// for the same sessionID. Empty inputs are rejected with an explicit error
// rather than silently no-op'd, so the POST /api/remote-files/caps handler
// can surface 400 to the caller (Pitfall 4 in the plan).
func (s *RemoteCapStore) Put(sessionID, baseURL, capToken string) error {
	if sessionID == "" {
		return errors.New("remote caps: sessionID required")
	}
	if baseURL == "" {
		return errors.New("remote caps: baseURL required")
	}
	if capToken == "" {
		return errors.New("remote caps: capToken required")
	}
	s.mu.Lock()
	s.entries[sessionID] = remoteCapEntry{baseURL: baseURL, capToken: capToken}
	s.mu.Unlock()
	return nil
}

// Get returns the (baseURL, capToken, true) tuple for sessionID, or
// ("", "", false) if no entry exists. Safe for concurrent readers via RWMutex.
func (s *RemoteCapStore) Get(sessionID string) (baseURL, capToken string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[sessionID]
	if !ok {
		return "", "", false
	}
	return e.baseURL, e.capToken, true
}

// Delete removes the entry for sessionID. A subsequent Get returns (_, _, false).
// No-op when sessionID is unknown.
func (s *RemoteCapStore) Delete(sessionID string) {
	s.mu.Lock()
	delete(s.entries, sessionID)
	s.mu.Unlock()
}
