// Package capability_test keystore tests. Mirrors
// internal/daemon/engine_settings_test.go:11-99 for t.TempDir-based round-trip
// tests and mode-0600 file permission assertions.
package capability_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/scottkw/agenthub/internal/capability"
)

// TestFileKeyStore_RoundTrip verifies that Save followed by Load returns the
// same 32 bytes AND that the file mode is 0600 (matching the existing
// saveSettingsToDisk behavior at engine.go:132).
func TestFileKeyStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := capability.NewFileKeyStore(dir)

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	if err := store.Save(key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(store.Location())
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 0600", perm)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if !bytes.Equal(got, key) {
		t.Errorf("round-trip mismatch: got %x, want %x", got, key)
	}
}

// TestFileKeyStore_MissingFileReturnsNilNil mirrors the missing-file-is-not-error
// idiom from engine_settings_test.go:88-99 / engine.go:88-92. On first run the
// key file does not exist and Load returns (nil, nil) so the caller can
// generate+save.
func TestFileKeyStore_MissingFileReturnsNilNil(t *testing.T) {
	dir := t.TempDir()
	store := capability.NewFileKeyStore(dir)
	got, err := store.Load()
	if err != nil {
		t.Errorf("Load on missing file: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil on missing file, got %x", got)
	}
}

// TestFileKeyStore_CorruptLengthReturnsError asserts that a key file whose
// contents are not exactly 32 bytes produces an error (the file is corrupt
// rather than silently ignored).
func TestFileKeyStore_CorruptLengthReturnsError(t *testing.T) {
	dir := t.TempDir()
	store := capability.NewFileKeyStore(dir)
	// Write 16 bytes instead of 32.
	if err := os.WriteFile(filepath.Join(dir, "capability.key"), make([]byte, 16), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := store.Load()
	if err == nil {
		t.Errorf("expected error for corrupt key file, got key=%x", got)
	}
}

// TestGenerateKey_Length32 asserts that GenerateKey produces exactly 32
// bytes (the HMAC-SHA256 key size locked by D-01).
func TestGenerateKey_Length32(t *testing.T) {
	key, err := capability.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("len(key) = %d, want 32", len(key))
	}
}

// TestLoadOrGenerate_GeneratesOnFirstRun asserts that LoadOrGenerate creates
// a key file when none exists AND the generated key is persisted to disk.
func TestLoadOrGenerate_GeneratesOnFirstRun(t *testing.T) {
	dir := t.TempDir()
	store := capability.NewFileKeyStore(dir)
	key, err := capability.LoadOrGenerate(store)
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("generated key length = %d, want 32", len(key))
	}
	// File must exist after first run.
	if _, err := os.Stat(store.Location()); err != nil {
		t.Errorf("key file not persisted: %v", err)
	}
}

// TestLoadOrGenerate_ReloadsOnSecondRun asserts that calling LoadOrGenerate
// twice returns the SAME key bytes — the second call must load from disk
// rather than generate a fresh key. This is the property that lets
// previously-shared capability URLs survive a daemon restart.
func TestLoadOrGenerate_ReloadsOnSecondRun(t *testing.T) {
	dir := t.TempDir()
	store := capability.NewFileKeyStore(dir)
	first, err := capability.LoadOrGenerate(store)
	if err != nil {
		t.Fatalf("first LoadOrGenerate: %v", err)
	}
	second, err := capability.LoadOrGenerate(store)
	if err != nil {
		t.Fatalf("second LoadOrGenerate: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("key changed between runs: first=%x second=%x", first, second)
	}
}
