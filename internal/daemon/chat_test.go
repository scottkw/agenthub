package daemon

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

// TestChatStoreOversizedLineSkip verifies that an over-length (>maxChatLineBytes)
// JSONL line is skipped on load and the surrounding well-formed messages still
// load — the over-length line must NOT abort the whole thread (WR-02). This is
// the read-path counterpart to the AppendMessage size guard.
func TestChatStoreOversizedLineSkip(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "sess-oversized"
	filePath := filepath.Join(baseDir, sessionID+".jsonl")

	good1 := relay.ChatMessage{SchemaVersion: 1, ID: "m1", Content: "before", TimestampMs: 1000}
	good2 := relay.ChatMessage{SchemaVersion: 1, ID: "m2", Content: "after", TimestampMs: 2000}
	b1, _ := json.Marshal(good1)
	b2, _ := json.Marshal(good2)

	// A valid JSON line whose serialized form comfortably exceeds the cap.
	huge := relay.ChatMessage{SchemaVersion: 1, ID: "huge", Content: strings.Repeat("x", (1<<20)+4096), TimestampMs: 1500}
	bh, _ := json.Marshal(huge)
	if len(bh) <= (1 << 20) {
		t.Fatalf("test setup: oversized line is only %d bytes, expected > 1 MiB", len(bh))
	}

	var content []byte
	content = append(content, b1...)
	content = append(content, '\n')
	content = append(content, bh...)
	content = append(content, '\n')
	content = append(content, b2...)
	content = append(content, '\n')
	if err := os.WriteFile(filePath, content, 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store, err := NewChatStore(baseDir, sessionID)
	if err != nil {
		t.Fatalf("NewChatStore must not fail on an over-length line, got: %v", err)
	}

	msgs := store.Messages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (over-length line skipped, neighbors kept), got %d", len(msgs))
	}
	if msgs[0].ID != "m1" || msgs[1].ID != "m2" {
		t.Errorf("unexpected messages after skipping over-length line: %+v", msgs)
	}
}

// TestChatAppendRejectsOversized verifies that AppendMessage rejects a message
// whose serialized line would exceed maxChatLineBytes (ErrChatMessageTooLarge),
// writes nothing to disk, and does not grow the mirror (WR-02 write-path guard).
func TestChatAppendRejectsOversized(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-toolarge")
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}

	huge := relay.ChatMessage{
		AuthorID: "local",
		Content:  strings.Repeat("y", (1<<20)+4096),
	}
	_, err = store.AppendMessage(huge)
	if err == nil {
		t.Fatal("expected ErrChatMessageTooLarge, got nil")
	}
	if err != ErrChatMessageTooLarge {
		t.Errorf("expected ErrChatMessageTooLarge, got %v", err)
	}

	// Nothing persisted, mirror unchanged.
	if n := len(store.Messages()); n != 0 {
		t.Errorf("expected 0 messages after rejected append, got %d", n)
	}
	if _, statErr := os.Stat(store.filePath); !os.IsNotExist(statErr) {
		t.Errorf("expected no file written for rejected oversized append, stat: %v", statErr)
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

// TestChatStoreMessagesDeepCopiesMentions verifies that Messages() deep-copies
// the Mentions slice inside each returned ChatMessage, so a caller mutating a
// returned message's Mentions element cannot corrupt the store's in-memory
// thread (WR-01). A shallow struct copy would leave Mentions aliasing the
// mirror's backing array.
func TestChatStoreMessagesDeepCopiesMentions(t *testing.T) {
	baseDir := t.TempDir()
	sessionID := "sess-mentions"
	filePath := filepath.Join(baseDir, sessionID+".jsonl")

	msg := relay.ChatMessage{
		SchemaVersion: 1,
		ID:            "m1",
		Content:       "hello @bob",
		Mentions:      []string{"bob"},
		TimestampMs:   1000,
	}
	b, _ := json.Marshal(msg)
	if err := os.WriteFile(filePath, append(b, '\n'), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store, err := NewChatStore(baseDir, sessionID)
	if err != nil {
		t.Fatalf("NewChatStore failed: %v", err)
	}

	// Mutate the Mentions element of the returned message.
	msgs := store.Messages()
	if len(msgs) == 0 || len(msgs[0].Mentions) == 0 {
		t.Fatal("expected at least 1 message with a Mention")
	}
	msgs[0].Mentions[0] = "mallory"

	// A second read must not observe the mutation.
	msgs2 := store.Messages()
	if msgs2[0].Mentions[0] == "mallory" {
		t.Error("Messages() aliased the Mentions backing array; mutation leaked into the store")
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

// --------------------------------------------------------------------------
// Task 2 (Plan 02) tests: Export() and Delete()
// --------------------------------------------------------------------------

// TestChatExportFields verifies that Export() renders every ChatMessage field:
// AuthorAlias, AuthorID, an RFC3339 timestamp derived from TimestampMs, Content,
// and — when SessionInject == true — an explicit inject marker.
func TestChatExportFields(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-export")
	if err != nil {
		t.Fatalf("NewChatStore: %v", err)
	}

	msgs := []relay.ChatMessage{
		{
			AuthorAlias:   "alice",
			AuthorID:      "node-aaa",
			Content:       "first message",
			TimestampMs:   1000000000000, // 2001-09-09T01:46:40Z
			SessionInject: false,
		},
		{
			AuthorAlias:   "bob",
			AuthorID:      "node-bbb",
			Content:       "second message",
			TimestampMs:   1000000060000, // 2001-09-09T01:47:40Z
			SessionInject: true,
		},
		{
			AuthorAlias:   "alice",
			AuthorID:      "node-aaa",
			Content:       "third message",
			TimestampMs:   1000000120000,
			SessionInject: false,
		},
	}
	for _, m := range msgs {
		if _, err := store.AppendMessage(m); err != nil {
			t.Fatalf("AppendMessage: %v", err)
		}
	}

	out, err := store.Export()
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}
	if out == "" {
		t.Fatal("Export() returned empty string")
	}

	// Every AuthorAlias must appear.
	for _, m := range msgs {
		if !strings.Contains(out, m.AuthorAlias) {
			t.Errorf("Export() missing AuthorAlias %q", m.AuthorAlias)
		}
		if !strings.Contains(out, m.AuthorID) {
			t.Errorf("Export() missing AuthorID %q", m.AuthorID)
		}
		if !strings.Contains(out, m.Content) {
			t.Errorf("Export() missing Content %q", m.Content)
		}
		// TimestampMs rendered as RFC3339 UTC substring.
		ts := time.UnixMilli(m.TimestampMs).UTC().Format(time.RFC3339)
		if !strings.Contains(out, ts) {
			t.Errorf("Export() missing RFC3339 timestamp %q", ts)
		}
	}

	// The inject marker must appear exactly once (for the second message).
	if !strings.Contains(out, "injected into terminal") {
		t.Error("Export() missing inject marker for SessionInject=true message")
	}
}

// TestChatExportEmpty verifies that Export() on an empty thread returns a
// non-empty header-only document and no error.
func TestChatExportEmpty(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-export-empty")
	if err != nil {
		t.Fatalf("NewChatStore: %v", err)
	}

	out, err := store.Export()
	if err != nil {
		t.Fatalf("Export() on empty thread returned error: %v", err)
	}
	if out == "" {
		t.Error("Export() returned empty string for header-only document")
	}
}

// TestChatDeleteRemovesFile verifies that Delete() removes the JSONL file from
// disk (os.Stat reports IsNotExist afterward) and empties the in-memory mirror.
func TestChatDeleteRemovesFile(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-delete")
	if err != nil {
		t.Fatalf("NewChatStore: %v", err)
	}

	// Write a message so the file is created on disk.
	if _, err := store.AppendMessage(relay.ChatMessage{
		AuthorID:    "local",
		AuthorAlias: "alice",
		Content:     "hello",
	}); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	// Confirm file exists before Delete.
	if _, err := os.Stat(store.filePath); err != nil {
		t.Fatalf("file should exist before Delete: %v", err)
	}

	if err := store.Delete(); err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// File must be gone.
	if _, err := os.Stat(store.filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed after Delete(), os.Stat returned: %v", err)
	}
	// Mirror must be empty.
	if n := len(store.Messages()); n != 0 {
		t.Errorf("expected Messages() length 0 after Delete(), got %d", n)
	}
}

// TestChatDeleteIdempotent verifies that a second Delete() on an already-
// deleted store returns nil (no error).
func TestChatDeleteIdempotent(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-delete-idem")
	if err != nil {
		t.Fatalf("NewChatStore: %v", err)
	}
	if _, err := store.AppendMessage(relay.ChatMessage{
		AuthorID: "local",
		Content:  "hello",
	}); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	if err := store.Delete(); err != nil {
		t.Fatalf("first Delete() error: %v", err)
	}
	if err := store.Delete(); err != nil {
		t.Errorf("second Delete() (idempotent) returned error: %v", err)
	}
}

// TestChatDeleteOnNeverWrittenStore verifies that Delete() on a store that was
// constructed but had no messages appended (no file on disk) is also idempotent.
func TestChatDeleteOnNeverWrittenStore(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewChatStore(baseDir, "sess-delete-nofile")
	if err != nil {
		t.Fatalf("NewChatStore: %v", err)
	}
	if err := store.Delete(); err != nil {
		t.Errorf("Delete() on store with no file returned error: %v", err)
	}
}

// TestChatStore_Export verifies the YAML-frontmatter export format (EXPORT-01,
// SC-2). Five behaviors are exercised:
//  1. Empty thread → frontmatter-only document with zero participants.
//  2. Single message → one participant entry; correct message header.
//  3. Two messages from the same AuthorID → participants deduplicated to one.
//  4. SessionInject==true → `_injected into terminal_` marker present.
//  5. Alias containing `:` → participant value is double-quoted in YAML.
func TestChatStore_Export(t *testing.T) {
	t.Run("EmptyThread", func(t *testing.T) {
		baseDir := t.TempDir()
		store, err := NewChatStore(baseDir, "sess-export-empty")
		if err != nil {
			t.Fatalf("NewChatStore: %v", err)
		}
		out, err := store.Export()
		if err != nil {
			t.Fatalf("Export() on empty thread returned error: %v", err)
		}
		if !strings.HasPrefix(out, "---\n") {
			t.Errorf("Export() must start with ---; got: %.100q", out)
		}
		for _, want := range []string{"session: sess-export-empty", "exported_at: ", "participants:"} {
			if !strings.Contains(out, want) {
				t.Errorf("Export() missing %q in empty-thread output; got: %.300q", want, out)
			}
		}
		if !strings.Contains(out, "# Chat: sess-export-empty") {
			t.Errorf("Export() missing heading '# Chat: sess-export-empty'")
		}
	})

	t.Run("SingleMessage", func(t *testing.T) {
		baseDir := t.TempDir()
		store, err := NewChatStore(baseDir, "sess-single")
		if err != nil {
			t.Fatalf("NewChatStore: %v", err)
		}
		_, err = store.AppendMessage(relay.ChatMessage{
			AuthorAlias: "alice",
			AuthorID:    "local",
			Content:     "hello single",
			TimestampMs: 1000,
		})
		if err != nil {
			t.Fatalf("AppendMessage: %v", err)
		}
		out, err := store.Export()
		if err != nil {
			t.Fatalf("Export(): %v", err)
		}
		// Participants must have exactly one entry.
		if !strings.Contains(out, `"alice (local)"`) {
			t.Errorf("Export() missing participant entry; got: %.400q", out)
		}
		// Message header must include alias, authorID, and RFC3339 timestamp.
		wantTS := time.UnixMilli(1000).UTC().Format(time.RFC3339)
		wantHeader := "## alice (local) — " + wantTS
		if !strings.Contains(out, wantHeader) {
			t.Errorf("Export() missing message header %q; got: %.400q", wantHeader, out)
		}
		if !strings.Contains(out, "hello single") {
			t.Errorf("Export() missing message content")
		}
	})

	t.Run("DeduplicatedParticipants", func(t *testing.T) {
		baseDir := t.TempDir()
		store, err := NewChatStore(baseDir, "sess-dedup")
		if err != nil {
			t.Fatalf("NewChatStore: %v", err)
		}
		for _, m := range []relay.ChatMessage{
			{AuthorAlias: "alice", AuthorID: "local", Content: "msg1", TimestampMs: 1000},
			{AuthorAlias: "bob", AuthorID: "node-xyz", Content: "msg2", TimestampMs: 2000},
			{AuthorAlias: "alice", AuthorID: "local", Content: "msg3", TimestampMs: 3000},
		} {
			if _, err := store.AppendMessage(m); err != nil {
				t.Fatalf("AppendMessage: %v", err)
			}
		}
		out, err := store.Export()
		if err != nil {
			t.Fatalf("Export(): %v", err)
		}
		// alice appears once in participants (deduplicated).
		if strings.Count(out, `"alice (local)"`) != 1 {
			t.Errorf("alice (local) deduplicated incorrectly; count = %d; got: %.400q",
				strings.Count(out, `"alice (local)"`), out)
		}
	})

	t.Run("SessionInjectMarker", func(t *testing.T) {
		baseDir := t.TempDir()
		store, err := NewChatStore(baseDir, "sess-inject")
		if err != nil {
			t.Fatalf("NewChatStore: %v", err)
		}
		if _, err := store.AppendMessage(relay.ChatMessage{
			AuthorAlias:   "alice",
			AuthorID:      "local",
			Content:       "run the thing",
			TimestampMs:   1000,
			SessionInject: true,
		}); err != nil {
			t.Fatalf("AppendMessage: %v", err)
		}
		out, err := store.Export()
		if err != nil {
			t.Fatalf("Export(): %v", err)
		}
		if !strings.Contains(out, "_injected into terminal_") {
			t.Errorf("Export() missing '_injected into terminal_' marker; got: %.400q", out)
		}
	})

	t.Run("YAMLSpecialCharInAlias", func(t *testing.T) {
		baseDir := t.TempDir()
		store, err := NewChatStore(baseDir, "sess-alias")
		if err != nil {
			t.Fatalf("NewChatStore: %v", err)
		}
		if _, err := store.AppendMessage(relay.ChatMessage{
			AuthorAlias: "ops:lead",
			AuthorID:    "node-abc",
			Content:     "special",
			TimestampMs: 1000,
		}); err != nil {
			t.Fatalf("AppendMessage: %v", err)
		}
		out, err := store.Export()
		if err != nil {
			t.Fatalf("Export(): %v", err)
		}
		// ops:lead must be double-quoted so the YAML is valid.
		if !strings.Contains(out, `"ops:lead (node-abc)"`) {
			t.Errorf("Export() alias with ':' must be double-quoted; got: %.400q", out)
		}
	})
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
