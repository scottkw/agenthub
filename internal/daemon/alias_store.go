package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/scottkw/agenthub/internal/relay"
)

// AliasStore is a global, daemon-owned, JSON-file-backed map from composite
// personKey (TailnetID+":"+origin) to a user-chosen display alias. It is
// global (not per-session) because alias identity is per-person: the same user
// wants the same alias across all of their sessions.
//
// The backing file is always named "aliases.json" directly under the supplied
// configDir. The basename is hardcoded — no user-controlled path component can
// influence the file path (T-152-06 path traversal mitigation).
//
// Thread safety: Get/GetOrDefault hold a read lock; Set and persist hold a write
// lock. RWMutex is used because reads are frequent (every subscribe) and writes
// are rare (user explicitly changes their alias).
type AliasStore struct {
	mu       sync.RWMutex
	filePath string
	aliases  map[string]string // personKey → alias
}

// NewAliasStore constructs an AliasStore that persists to
// filepath.Join(configDir, "aliases.json"). configDir is created with
// permissions 0700 if it does not yet exist (daemon data only).
//
// If aliases.json already exists its contents are loaded; an absent file is
// not an error (first-run path). Returns a non-nil error only for OS failures
// or malformed JSON.
//
// Production callers pass daemonConfigDir(); tests pass t.TempDir().
func NewAliasStore(configDir string) (*AliasStore, error) {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("alias_store: cannot create config dir %q: %w", configDir, err)
	}

	a := &AliasStore{
		// Hardcoded basename — personKey is never part of the file path.
		filePath: filepath.Join(configDir, "aliases.json"),
		aliases:  make(map[string]string),
	}

	if err := a.loadFromDisk(); err != nil {
		return nil, err
	}
	return a, nil
}

// Get returns the persisted alias for the given personKey.
// Returns ("", false) when no alias has been set.
func (a *AliasStore) Get(personKey string) (string, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	v, ok := a.aliases[personKey]
	return v, ok
}

// GetOrDefault returns the persisted alias for personKey, or def if no alias
// has been set. Equivalent to: if v, ok := a.Get(key); ok { return v }; return def.
func (a *AliasStore) GetOrDefault(personKey, def string) string {
	if v, ok := a.Get(personKey); ok {
		return v
	}
	return def
}

// Set validates alias via relay.ValidateAlias, stores it under personKey, and
// persists the updated map to aliases.json (0600). Returns a non-nil error if
// the alias is invalid or the file cannot be written. On error the in-memory
// map is NOT modified (T-152-01 defense in depth).
func (a *AliasStore) Set(personKey, alias string) error {
	validated := relay.ValidateAlias(alias)
	if validated == "" {
		return fmt.Errorf("alias_store: invalid alias %q: must be 1–32 runes with no control characters", alias)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.aliases[personKey] = validated
	if err := a.persist(); err != nil {
		// Roll back the in-memory change so the store stays consistent with disk.
		delete(a.aliases, personKey)
		return fmt.Errorf("alias_store: failed to persist: %w", err)
	}
	return nil
}

// loadFromDisk reads aliases.json from disk and populates the in-memory map.
// A missing file is silently ignored (first run). Malformed JSON returns an error.
// Called only from the constructor before the store is shared — no locking needed.
func (a *AliasStore) loadFromDisk() error {
	data, err := os.ReadFile(a.filePath)
	if os.IsNotExist(err) {
		return nil // first run — empty map is correct
	}
	if err != nil {
		return fmt.Errorf("alias_store: cannot read %q: %w", a.filePath, err)
	}
	if err := json.Unmarshal(data, &a.aliases); err != nil {
		// Leave the map empty and surface the error — a corrupt file should not
		// silently become an empty store on every restart.
		return fmt.Errorf("alias_store: corrupt %q: %w", a.filePath, err)
	}
	return nil
}

// persist serializes the in-memory alias map to aliases.json with 0600
// permissions. Caller holds a.mu.Lock().
func (a *AliasStore) persist() error {
	data, err := json.Marshal(a.aliases)
	if err != nil {
		return fmt.Errorf("alias_store: marshal failed: %w", err)
	}
	if err := os.WriteFile(a.filePath, data, 0600); err != nil {
		return fmt.Errorf("alias_store: write failed: %w", err)
	}
	return nil
}
