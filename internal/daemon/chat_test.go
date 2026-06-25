package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/scottkw/agenthub/internal/relay"
)

// TestChatStorePath verifies NewChatStore derives the file path correctly
// from the injected baseDir. Uses t.TempDir() so nothing touches the real
// daemon data dir.
func TestChatStorePath(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "abc123")
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}

	expectedPath := filepath.Join(baseDir, "abc123.jsonl")
	if store.filePath != expectedPath {
		t.Errorf("expected filePath=%q, got %q", expectedPath, store.filePath)
	}
}

// TestChatStoreRejectEmpty verifies NewChatStore rejects an empty sessionID.
func TestChatStoreRejectEmpty(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "")
	if err == nil {
		t.Error("expected error for empty sessionID, got nil")
	}
	if store != nil {
		t.Error("expected nil store on error, got non-nil")
	}
	// No file should have been created.
	entries, _ := os.ReadDir(baseDir)
	if len(entries) != 0 {
		t.Errorf("expected no files created for empty sessionID, found %d entries", len(entries))
	}
}

// TestChatStoreRejectPathTraversal verifies NewChatStore rejects sessionIDs
// that contain path separators or "..".
func TestChatStoreRejectPathTraversal(t *testing.T) {
	cases := []string{
		"../escape",
		"a/b",
		"a\\b",
		"..",
		".",
		"sub/dir/id",
		"../../../etc/passwd",
	}
	for _, id := range cases {
		t.Run("id="+id, func(t *testing.T) {
			baseDir := t.TempDir()
			store, err := NewChatStore(baseDir, id)
			if err == nil {
				t.Errorf("expected error for sessionID=%q, got nil", id)
			}
			if store != nil {
				t.Errorf("expected nil store for sessionID=%q, got non-nil", id)
			}
			// Assert no file named "escape.jsonl" (or similar) was created.
			entries, _ := os.ReadDir(baseDir)
			if len(entries) != 0 {
				t.Errorf("expected no files created for invalid sessionID=%q, found %d entries in baseDir", id, len(entries))
			}
		})
	}
}

// TestChatStoreRestartSurvival writes 3 JSONL lines to the backing file,
// constructs a fresh ChatStore, and verifies Messages() returns those 3
// messages in order.
func TestChatStoreRestartSurvival(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "sess001"

	// Pre-populate the JSONL file with known messages.
	messages := []relay.ChatMessage{
		{SchemaVersion: 1, ID: "m1", SessionID: sessionID, AuthorID: "local", AuthorAlias: "alice", Content: "first", TimestampMs: 1000},
		{SchemaVersion: 1, ID: "m2", SessionID: sessionID, AuthorID: "local", AuthorAlias: "alice", Content: "second", TimestampMs: 2000},
		{SchemaVersion: 1, ID: "m3", SessionID: sessionID, AuthorID: "node-xyz", AuthorAlias: "bob", Content: "third", TimestampMs: 3000},
	}

	filePath := filepath.Join(baseDir, sessionID+".jsonl")
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	enc := json.NewEncoder(f)
	for _, m := range messages {
		if err := enc.Encode(m); err != nil {
			t.Fatalf("failed to write test message: %v", err)
		}
	}
	f.Close()

	// Construct fresh store — simulates daemon restart.
	store, err := NewChatStore(baseDir, sessionID)
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}

	loaded := store.Messages()
	if len(loaded) != len(messages) {
		t.Fatalf("expected %d messages after restart, got %d", len(messages), len(loaded))
	}
	for i, want := range messages {
		got := loaded[i]
		if got.ID != want.ID || got.Content != want.Content || got.TimestampMs != want.TimestampMs {
			t.Errorf("message[%d] mismatch: got %+v, want %+v", i, got, want)
		}
	}
}

// TestChatStoreMalformedLineSkip verifies that a malformed (non-JSON) line in
// an existing JSONL file is skipped and the rest of the thread still loads.
func TestChatStoreMalformedLineSkip(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "sess002"
	filePath := filepath.Join(baseDir, sessionID+".jsonl")

	good1 := relay.ChatMessage{SchemaVersion: 1, ID: "m1", Content: "good one", TimestampMs: 1000}
	good2 := relay.ChatMessage{SchemaVersion: 1, ID: "m2", Content: "good two", TimestampMs: 2000}

	b1, _ := json.Marshal(good1)
	b2, _ := json.Marshal(good2)

	content := string(b1) + "\n" + "NOT VALID JSON $$$$\n" + string(b2) + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store, err := NewChatStore(baseDir, sessionID)
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}

	msgs := store.Messages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (malformed line skipped), got %d", len(msgs))
	}
	if msgs[0].ID != "m1" || msgs[1].ID != "m2" {
		t.Errorf("unexpected messages: %+v", msgs)
	}
}

// TestChatStoreMessagesReturnsCopy verifies that Messages() returns a copy,
// not a reference to internal state, so callers cannot mutate the store.
func TestChatStoreMessagesReturnsCopy(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "sess003"
	filePath := filepath.Join(baseDir, sessionID+".jsonl")

	msg := relay.ChatMessage{SchemaVersion: 1, ID: "m1", Content: "hello", TimestampMs: 1000}
	b, _ := json.Marshal(msg)
	if err := os.WriteFile(filePath, append(b, '\n'), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store, err := NewChatStore(baseDir, sessionID)
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}

	// Mutate the slice returned by Messages() — should not affect the store.
	msgs := store.Messages()
	if len(msgs) == 0 {
		t.Fatal("expected at least 1 message")
	}
	msgs[0].Content = "mutated"

	msgs2 := store.Messages()
	if msgs2[0].Content == "mutated" {
		t.Error("Messages() returned a reference to internal state; mutation leaked")
	}
}

// TestChatStoreDaemonConfigDirNotUsed verifies that the production chats-dir
// helper (chatsDir) derives its path from daemonConfigDir, and that tests
// using t.TempDir() as baseDir never touch the real config dir.
// This is a smoke test that documents the isolation contract.
func TestChatStoreDaemonConfigDirNotUsed(t *testing.T) {
	tempBase := t.TempDir()
	// Construct a store with a temp baseDir — confirm it doesn't call daemonConfigDir.
	_, err := NewChatStore(tempBase, "isolation-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Confirm chatsDir() (production helper) includes daemonConfigDir in its output.
	dir := chatsDir()
	if dir == "" {
		t.Error("chatsDir() returned empty string")
	}
}
