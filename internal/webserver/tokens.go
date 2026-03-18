package webserver

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
)

// GenerateToken produces a cryptographically random 32-byte token encoded as
// base64url without padding (43 chars). Uses crypto/rand only — never math/rand.
func GenerateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// TokenStore maps opaque tokens to session IDs bidirectionally, enabling per-session
// sharing links without exposing session IDs in URLs.
type TokenStore struct {
	mu              sync.RWMutex
	tokenToSession  map[string]string
	sessionToTokens map[string][]string
}

// NewTokenStore initializes a TokenStore with empty maps.
func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokenToSession:  make(map[string]string),
		sessionToTokens: make(map[string][]string),
	}
}

// Create generates a new token for the given sessionID, stores it in both maps,
// and returns the token. Returns an error if token generation fails.
func (ts *TokenStore) Create(sessionID string) (string, error) {
	tok, err := GenerateToken()
	if err != nil {
		return "", err
	}

	ts.mu.Lock()
	ts.tokenToSession[tok] = sessionID
	ts.sessionToTokens[sessionID] = append(ts.sessionToTokens[sessionID], tok)
	ts.mu.Unlock()

	return tok, nil
}

// Lookup returns the sessionID for the given token and true if found,
// or ("", false) if not present.
func (ts *TokenStore) Lookup(token string) (sessionID string, ok bool) {
	ts.mu.RLock()
	sid, found := ts.tokenToSession[token]
	ts.mu.RUnlock()
	return sid, found
}

// Revoke removes a single token from both maps.
func (ts *TokenStore) Revoke(token string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	sessionID, exists := ts.tokenToSession[token]
	if !exists {
		return
	}
	delete(ts.tokenToSession, token)

	// Remove from sessionToTokens slice.
	tokens := ts.sessionToTokens[sessionID]
	filtered := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t != token {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		delete(ts.sessionToTokens, sessionID)
	} else {
		ts.sessionToTokens[sessionID] = filtered
	}
}

// RevokeBySession removes all tokens associated with a session from both maps.
func (ts *TokenStore) RevokeBySession(sessionID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	tokens := ts.sessionToTokens[sessionID]
	for _, t := range tokens {
		delete(ts.tokenToSession, t)
	}
	delete(ts.sessionToTokens, sessionID)
}

// TokensForSession returns a copy of all tokens associated with the given session.
// Used by the UI to display existing sharing links.
func (ts *TokenStore) TokensForSession(sessionID string) []string {
	ts.mu.RLock()
	tokens := ts.sessionToTokens[sessionID]
	ts.mu.RUnlock()

	result := make([]string, len(tokens))
	copy(result, tokens)
	return result
}
