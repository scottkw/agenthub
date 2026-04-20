package capability

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// KeyStore abstracts signing key persistence. v3.1 ships FileKeyStore only;
// the interface exists specifically so a keychain-backed implementation can
// land in a future phase without protocol or API changes (D-05).
type KeyStore interface {
	// Load returns the persisted key, or (nil, nil) when no key has been
	// stored yet (first run). Any other failure returns (nil, err).
	Load() ([]byte, error)
	// Save persists the supplied key. Callers pass the full 32 bytes.
	Save(key []byte) error
	// Location returns the filesystem (or otherwise-opaque) location of the
	// backing store. Used for diagnostic logging; callers MUST NOT parse it.
	Location() string
}

// FileKeyStore persists the 32-byte signing key as a raw binary file at path.
// File permissions are 0600 (owner read/write only), matching the existing
// settings.json persistence pattern at internal/daemon/engine.go:132.
type FileKeyStore struct {
	path string
}

// NewFileKeyStore returns a FileKeyStore rooted at dir/capability.key.
func NewFileKeyStore(dir string) *FileKeyStore {
	return &FileKeyStore{path: filepath.Join(dir, "capability.key")}
}

// Location returns the absolute path of the backing file.
func (s *FileKeyStore) Location() string { return s.path }

// Load reads the key file. Returns the 32-byte key on success. Returns
// (nil, nil) when the file does not exist (first run — caller generates via
// GenerateKey). Returns an error for any other read failure or when the file
// exists but has the wrong length (corrupt).
func (s *FileKeyStore) Load() ([]byte, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("capability: read key file: %w", err)
	}
	if len(data) != 32 {
		return nil, fmt.Errorf("capability: key file corrupt (got %d bytes, want 32)", len(data))
	}
	return data, nil
}

// Save writes the key to the file with mode 0600. os.WriteFile is an atomic
// overwrite on POSIX (truncate + write in one syscall path) and is the same
// primitive used by internal/daemon/engine.go:132 for settings.json.
func (s *FileKeyStore) Save(key []byte) error {
	return os.WriteFile(s.path, key, 0600)
}

// GenerateKey returns a new 32-byte cryptographically random key suitable for
// HMAC-SHA256 signing. In Go 1.20+, crypto/rand.Read never returns an error,
// but the return shape is preserved for API stability.
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// LoadOrGenerate returns an existing key from store, or generates-and-saves a
// new one on first run. This is the standard daemon-startup bootstrap: one
// call replaces the read-then-check-then-maybe-generate dance callers would
// otherwise write.
func LoadOrGenerate(store KeyStore) ([]byte, error) {
	key, err := store.Load()
	if err != nil {
		return nil, err
	}
	if key != nil {
		return key, nil
	}
	key, err = GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("capability: generate key: %w", err)
	}
	if err := store.Save(key); err != nil {
		return nil, fmt.Errorf("capability: save key: %w", err)
	}
	return key, nil
}
