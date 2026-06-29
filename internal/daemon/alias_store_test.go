package daemon

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestAliasStoreBasicGetSet verifies Set persists a key and Get retrieves it.
func TestAliasStoreBasicGetSet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore: %v", err)
	}

	if err := store.Set("local:local", "ken"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok := store.Get("local:local")
	if !ok {
		t.Fatal("Get: expected true, got false")
	}
	if got != "ken" {
		t.Fatalf("Get: want %q, got %q", "ken", got)
	}
}

// TestAliasStoreGetAbsent verifies Get returns ("", false) for unknown keys.
func TestAliasStoreGetAbsent(t *testing.T) {
	dir := t.TempDir()
	store, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore: %v", err)
	}

	got, ok := store.Get("absent:web")
	if ok {
		t.Fatalf("Get: expected false, got true (value=%q)", got)
	}
	if got != "" {
		t.Fatalf("Get: want empty string, got %q", got)
	}
}

// TestAliasStoreGetOrDefault verifies GetOrDefault falls back when key is absent
// and returns the persisted alias when the key is present.
func TestAliasStoreGetOrDefault(t *testing.T) {
	dir := t.TempDir()
	store, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore: %v", err)
	}

	// Absent key uses default.
	got := store.GetOrDefault("absent:web", "laptop")
	if got != "laptop" {
		t.Fatalf("GetOrDefault absent: want %q, got %q", "laptop", got)
	}

	// After Set, persisted alias wins over default.
	if err := store.Set("local:local", "ken"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got = store.GetOrDefault("local:local", "laptop")
	if got != "ken" {
		t.Fatalf("GetOrDefault present: want %q, got %q", "ken", got)
	}
}

// TestAliasStoreReloadPersistence verifies that Set writes to disk and a fresh
// NewAliasStore over the same directory loads the persisted alias (D-02 restart
// survival).
func TestAliasStoreReloadPersistence(t *testing.T) {
	dir := t.TempDir()

	// First store: write an alias.
	store1, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore (first): %v", err)
	}
	if err := store1.Set("local:local", "ken"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Second store over the same dir: must see the persisted alias.
	store2, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore (second): %v", err)
	}
	got, ok := store2.Get("local:local")
	if !ok {
		t.Fatal("Get after reload: expected true, got false")
	}
	if got != "ken" {
		t.Fatalf("Get after reload: want %q, got %q", "ken", got)
	}
}

// TestAliasStoreCompositeKeyIsolation verifies that owner (local:local) and a
// same-machine browser (nodekey:web) keep separate, isolated aliases.
func TestAliasStoreCompositeKeyIsolation(t *testing.T) {
	dir := t.TempDir()
	store, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore: %v", err)
	}

	if err := store.Set("local:local", "owner"); err != nil {
		t.Fatalf("Set local:local: %v", err)
	}
	if err := store.Set("nodekey:web", "browser"); err != nil {
		t.Fatalf("Set nodekey:web: %v", err)
	}

	ownerAlias, ok := store.Get("local:local")
	if !ok || ownerAlias != "owner" {
		t.Fatalf("Get local:local: want %q ok=true, got %q ok=%v", "owner", ownerAlias, ok)
	}

	browserAlias, ok := store.Get("nodekey:web")
	if !ok || browserAlias != "browser" {
		t.Fatalf("Get nodekey:web: want %q ok=true, got %q ok=%v", "browser", browserAlias, ok)
	}
}

// TestAliasStoreRejectInvalidAlias verifies that Set rejects invalid aliases
// without persisting them (T-152-01 defense in depth).
func TestAliasStoreRejectInvalidAlias(t *testing.T) {
	dir := t.TempDir()
	store, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore: %v", err)
	}

	cases := []struct {
		name  string
		alias string
	}{
		{"40 runes (over limit)", "abcdefghijklmnopqrstuvwxyzabcdefghijklmn"}, // 40 chars
		{"control char U+0007", "ken\x07"},
		{"empty after trim", "   "},
		{"empty string", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := store.Set("local:local", tc.alias)
			if err == nil {
				t.Fatalf("Set(%q): expected error for invalid alias, got nil", tc.alias)
			}

			// Must not be persisted.
			_, ok := store.Get("local:local")
			if ok {
				t.Fatalf("Set(%q): invalid alias was stored despite error", tc.alias)
			}
		})
	}
}

// TestAliasStoreFilePerms verifies that aliases.json is written with 0600
// permissions (no world/group read).
func TestAliasStoreFilePerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not enforce Unix file-mode bits; aliases.json reports 0666 regardless of the 0600 requested at write time. The 0600 write intent is exercised on macOS/Linux.")
	}
	dir := t.TempDir()
	store, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore: %v", err)
	}

	if err := store.Set("local:local", "ken"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "aliases.json"))
	if err != nil {
		t.Fatalf("Stat aliases.json: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("aliases.json perm: want 0600, got %04o", info.Mode().Perm())
	}
}

// TestAliasStoreFirstRunNoFile verifies that NewAliasStore succeeds (returns
// empty store) when no aliases.json exists yet (first run).
func TestAliasStoreFirstRunNoFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore on empty dir: %v", err)
	}
	if store == nil {
		t.Fatal("NewAliasStore returned nil store")
	}
	// aliases.json must NOT exist yet (no writes on first run).
	if _, statErr := os.Stat(filepath.Join(dir, "aliases.json")); !os.IsNotExist(statErr) {
		t.Fatalf("aliases.json should not exist before any Set, got: %v", statErr)
	}
}

// TestAliasStoreFixedBasename verifies that the file is stored at
// "aliases.json" directly under configDir (no user-controlled path component,
// T-152-06 path traversal mitigation).
func TestAliasStoreFixedBasename(t *testing.T) {
	dir := t.TempDir()
	store, err := NewAliasStore(dir)
	if err != nil {
		t.Fatalf("NewAliasStore: %v", err)
	}
	if err := store.Set("local:local", "ken"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Only "aliases.json" should appear in dir — no sub-directories, no other files.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "aliases.json" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("expected only [aliases.json] in configDir, got %v", names)
	}
}
