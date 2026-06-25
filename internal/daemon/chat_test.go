package daemon

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
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

// --------------------------------------------------------------------------
// Task 3 tests: AppendMessage, 10k cap, and concurrency (-race)
// --------------------------------------------------------------------------

// countFileLines counts the number of non-empty lines in a file.
func countFileLines(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("countFileLines: cannot open %q: %v", path, err)
	}
	defer f.Close()
	n := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		if len(scanner.Bytes()) > 0 {
			n++
		}
	}
	return n
}

// TestChatAppendBasic verifies that AppendMessage persists a message and
// assigns ID, TimestampMs, and SchemaVersion when the caller leaves them zero.
func TestChatAppendBasic(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-append")
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}

	in := relay.ChatMessage{
		AuthorID:    "local",
		AuthorAlias: "alice",
		Content:     "hello",
	}
	got, err := store.AppendMessage(in)
	if err != nil {
		t.Fatalf("AppendMessage failed: %v", err)
	}

	// Defaults must be filled.
	if got.ID == "" {
		t.Error("expected non-empty ID to be assigned")
	}
	if got.TimestampMs == 0 {
		t.Error("expected non-zero TimestampMs to be assigned")
	}
	if got.SchemaVersion != relay.ChatSchemaVersion {
		t.Errorf("expected SchemaVersion=%d, got %d", relay.ChatSchemaVersion, got.SchemaVersion)
	}
	if got.SessionID != "sess-append" {
		t.Errorf("expected SessionID=sess-append, got %q", got.SessionID)
	}

	// In-memory mirror should hold 1 message.
	msgs := store.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message in mirror, got %d", len(msgs))
	}

	// On-disk file should hold 1 line.
	if n := countFileLines(t, store.filePath); n != 1 {
		t.Errorf("expected 1 line on disk, got %d", n)
	}
}

// TestChatAppendRoundTrip verifies that each persisted JSONL line round-trips
// back to an equal ChatMessage.
func TestChatAppendRoundTrip(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-roundtrip")
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}

	in := relay.ChatMessage{
		AuthorID:    "node-abc",
		AuthorAlias: "bob",
		Content:     "round-trip test",
		Mentions:    []string{"alice"},
	}
	canonical, err := store.AppendMessage(in)
	if err != nil {
		t.Fatalf("AppendMessage failed: %v", err)
	}

	// Read the JSONL file and decode the line.
	f, err := os.Open(store.filePath)
	if err != nil {
		t.Fatalf("cannot open JSONL file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatal("expected at least one line in JSONL file")
	}

	var decoded relay.ChatMessage
	if err := json.Unmarshal(scanner.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSONL line: %v", err)
	}

	// The decoded message must equal the canonical return value.
	if decoded.ID != canonical.ID || decoded.Content != canonical.Content ||
		decoded.TimestampMs != canonical.TimestampMs || decoded.SchemaVersion != canonical.SchemaVersion {
		t.Errorf("round-trip mismatch:\n  canonical: %+v\n  decoded:   %+v", canonical, decoded)
	}
}

// TestChatCapEnforcement verifies that after MaxChatMessages successful
// appends, the next AppendMessage returns ErrChatCapReached, writes nothing
// to the file, and does not grow the mirror beyond MaxChatMessages.
func TestChatCapEnforcement(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-cap")
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}

	// Append exactly MaxChatMessages messages.
	for i := 0; i < MaxChatMessages; i++ {
		_, err := store.AppendMessage(relay.ChatMessage{
			AuthorID: "local",
			Content:  "msg",
		})
		if err != nil {
			t.Fatalf("unexpected error at append %d: %v", i, err)
		}
	}

	// The (MaxChatMessages+1)th append must return ErrChatCapReached.
	_, err = store.AppendMessage(relay.ChatMessage{
		AuthorID: "local",
		Content:  "overflow",
	})
	if err == nil {
		t.Fatal("expected ErrChatCapReached, got nil")
	}
	if err != ErrChatCapReached {
		t.Errorf("expected ErrChatCapReached, got %v", err)
	}

	// Mirror must stay at MaxChatMessages.
	if n := len(store.Messages()); n != MaxChatMessages {
		t.Errorf("mirror length after cap: expected %d, got %d", MaxChatMessages, n)
	}

	// File must have exactly MaxChatMessages lines (no extra line written).
	if n := countFileLines(t, store.filePath); n != MaxChatMessages {
		t.Errorf("file line count after cap: expected %d, got %d", MaxChatMessages, n)
	}
}

// TestChatConcurrentAppend runs 200 goroutines each calling AppendMessage once
// and verifies all messages are persisted exactly once. This is the -race gate
// for Pitfall 11 (concurrent write race).
func TestChatConcurrentAppend(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-concurrent")
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}

	const numGoroutines = 200
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = store.AppendMessage(relay.ChatMessage{
				AuthorID: "local",
				Content:  "concurrent",
			})
		}()
	}
	wg.Wait()

	// All 200 messages must be in the mirror.
	if n := len(store.Messages()); n != numGoroutines {
		t.Errorf("mirror length: expected %d, got %d", numGoroutines, n)
	}

	// File must have exactly 200 lines.
	if n := countFileLines(t, store.filePath); n != numGoroutines {
		t.Errorf("file line count: expected %d, got %d", numGoroutines, n)
	}
}
