package daemon

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/scottkw/agenthub/internal/relay"
)

// MaxChatMessages is the hard per-session cap on the number of messages that
// can be stored in a ChatStore. Once the cap is reached, AppendMessage rejects
// further writes (ErrChatCapReached) rather than trimming. Reject-not-trim
// preserves the append-only JSONL invariant: trimming would require rewriting
// the entire file under load, introducing a rewrite race and destroying the
// durability guarantee.
const MaxChatMessages = 10000

// ErrChatCapReached is returned by AppendMessage when the store has reached
// MaxChatMessages. The caller should inform the sender that the session chat
// thread is full.
var ErrChatCapReached = errors.New("chat: session message cap reached")

// maxChatLineBytes bounds the size of a single serialized JSONL line. The SAME
// bound is enforced on both paths so the write path and read path agree (WR-02):
//   - AppendMessage rejects any message whose serialized line would exceed it
//     (ErrChatMessageTooLarge), so an unreplayable line can never be written.
//   - loadFromDisk caps its read accumulation at this size and skips any
//     over-length line rather than aborting the whole thread load.
const maxChatLineBytes = 1 << 20 // 1 MiB

// ErrChatMessageTooLarge is returned by AppendMessage when the serialized
// message line would exceed maxChatLineBytes. Rejecting on write keeps the
// on-disk invariant in sync with loadFromDisk's read buffer so every persisted
// line is guaranteed replayable on restart.
var ErrChatMessageTooLarge = errors.New("chat: message exceeds maximum line size")

// chatsDir returns the production base directory for chat JSONL files:
// filepath.Join(daemonConfigDir(), "chats"). This is the value that production
// callers pass into NewChatStore. Tests pass t.TempDir() instead so the store
// never touches the real daemon data directory.
func chatsDir() string {
	return filepath.Join(daemonConfigDir(), "chats")
}

// validChatSessionID returns true only when id is a non-empty string that
// consists entirely of characters from the strict allowlist [A-Za-z0-9_-].
// This rejects empty strings, ".", "..", any path separator, embedded NUL,
// and any character outside the allowlist.
func validChatSessionID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// ChatStore is a per-session, append-only message store backed by a JSONL
// file under the injected base directory. It mirrors the on-disk state
// in-memory for fast reads (Messages()) and serializes all writes via a
// mutex so concurrent AppendMessage calls are safe under -race.
//
// Construction: NewChatStore(baseDir, sessionID)
//   - baseDir must be the chats directory (chatsDir() in production, t.TempDir() in tests)
//   - sessionID must satisfy validChatSessionID
//
// The JSONL file path is: filepath.Join(baseDir, sessionID+".jsonl")
// Each line is exactly one json.Marshal of a relay.ChatMessage followed by "\n".
type ChatStore struct {
	mu        sync.Mutex
	filePath  string
	sessionID string
	messages  []relay.ChatMessage
}

// NewChatStore constructs a ChatStore rooted at baseDir for the given
// sessionID. baseDir is the chats directory — callers (production) pass
// chatsDir(); tests pass t.TempDir(). The constructor does NOT call
// daemonConfigDir() internally; isolation is achieved at the call site.
//
// If a JSONL file already exists at filepath.Join(baseDir, sessionID+".jsonl"),
// its lines are loaded into the in-memory mirror. Malformed lines are skipped
// without aborting the load so daemon restarts are robust.
func NewChatStore(baseDir, sessionID string) (*ChatStore, error) {
	if !validChatSessionID(sessionID) {
		return nil, fmt.Errorf("chat: invalid sessionID %q: must match [A-Za-z0-9_-]+ and be non-empty", sessionID)
	}

	// Create the directory if needed (0700 — daemon data only).
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("chat: cannot create base dir %q: %w", baseDir, err)
	}

	path := filepath.Join(baseDir, sessionID+".jsonl")

	// Defense-in-depth containment check: the resolved parent of the JSONL file
	// must equal the cleaned baseDir. This ensures that even if filepath.Join
	// somehow produced a path outside baseDir (e.g., due to a symlink), we
	// reject it before any file operation.
	if filepath.Dir(path) != filepath.Clean(baseDir) {
		return nil, fmt.Errorf("chat: derived path %q escapes baseDir %q", path, baseDir)
	}

	store := &ChatStore{
		filePath:  path,
		sessionID: sessionID,
	}

	// Load any existing messages from disk (daemon restart survival).
	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}

	return store, nil
}

// loadFromDisk reads the JSONL file (if it exists) and populates the
// in-memory mirror. Malformed lines are silently skipped. This method is
// only called once from the constructor before the store is shared, so no
// locking is required.
func (s *ChatStore) loadFromDisk() error {
	f, err := os.Open(s.filePath)
	if os.IsNotExist(err) {
		// No prior messages — fresh store. Normal path.
		return nil
	}
	if err != nil {
		return fmt.Errorf("chat: cannot open %q for replay: %w", s.filePath, err)
	}
	defer f.Close()

	// Read line-by-line with a hard per-line cap (maxChatLineBytes). Unlike a
	// bufio.Scanner — which stops entirely and returns bufio.ErrTooLong the
	// moment a single line exceeds its buffer, dropping every subsequent
	// message — readCappedLine fully consumes (and discards) an over-length
	// line so loading continues with the next one. This makes an over-length
	// line skip-and-continue, exactly like a malformed-JSON line, so one bad
	// line never makes the whole thread unavailable on restart (WR-02).
	reader := bufio.NewReaderSize(f, 64*1024)
	for {
		line, tooLong, rerr := readCappedLine(reader, maxChatLineBytes)
		switch {
		case tooLong:
			// Over-length line — skip it (metadata only; never log content).
			log.Printf("chat: skipping over-length line (>%d bytes) while loading %q", maxChatLineBytes, s.filePath)
		case len(line) > 0:
			var msg relay.ChatMessage
			if err := json.Unmarshal(line, &msg); err != nil {
				// Malformed line — skip, continue loading the rest.
				break
			}
			s.messages = append(s.messages, msg)
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return nil
			}
			return fmt.Errorf("chat: read %q during replay: %w", s.filePath, rerr)
		}
	}
}

// readCappedLine reads a single logical line (terminated by '\n') from r,
// accumulating up to max bytes. If the line exceeds max bytes, tooLong is true
// and the returned line is empty — the caller should skip it. The full logical
// line is always consumed from r (even when over-length), so the next call
// begins at the following line. The returned err is io.EOF once the underlying
// reader is exhausted; any bytes read before EOF are returned alongside it.
func readCappedLine(r *bufio.Reader, max int) (line []byte, tooLong bool, err error) {
	var buf []byte
	for {
		chunk, isPrefix, e := r.ReadLine()
		if !tooLong {
			if len(buf)+len(chunk) <= max {
				buf = append(buf, chunk...)
			} else {
				// Exceeded the cap: reject the whole line but keep draining the
				// remaining prefix chunks below so r advances past it.
				tooLong = true
				buf = nil
			}
		}
		if !isPrefix || e != nil {
			return buf, tooLong, e
		}
	}
}

// Messages returns a copy of the full in-order message thread. The copy
// prevents callers from mutating internal state. This method is safe to call
// concurrently with AppendMessage.
func (s *ChatStore) Messages() []relay.ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return a copy so callers cannot mutate s.messages. copy() is a shallow
	// struct copy: it duplicates each ChatMessage's fields but the Mentions
	// []string still aliases the mirror's backing array. Deep-copy Mentions so
	// a caller doing msgs[i].Mentions[j] = ... cannot corrupt internal state
	// (WR-01).
	result := make([]relay.ChatMessage, len(s.messages))
	copy(result, s.messages)
	for i := range result {
		if src := s.messages[i].Mentions; src != nil {
			result[i].Mentions = append([]string(nil), src...)
		}
	}
	return result
}

// AppendMessage persists msg to the JSONL file and adds it to the in-memory
// mirror. The entire operation (cap check + file write + slice append) is
// serialized under the mutex so the mirror and disk state never diverge.
//
// Defaults filled when the caller leaves them zero:
//   - msg.ID is set to a crypto/rand hex string (16 bytes = 32 hex chars).
//   - msg.TimestampMs is set to time.Now().UnixMilli().
//   - msg.SchemaVersion is set to relay.ChatSchemaVersion.
//   - msg.SessionID is set to the store's session ID.
//
// AppendMessage rejects further writes when len(mirror) >= MaxChatMessages,
// returning (zero, ErrChatCapReached) without writing to the file. REJECT
// (not trim) is chosen because the file is append-only: trimming would require
// rewriting the entire JSONL under concurrent load, destroying the
// append-only invariant and introducing a rewrite race.
//
// Logging: only metadata (sessionID, count, byte length) is logged, never
// message content.
func (s *ChatStore) AppendMessage(msg relay.ChatMessage) (relay.ChatMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Enforce the hard cap BEFORE writing so the file never exceeds MaxChatMessages.
	// We REJECT (not trim) because the file is append-only: trimming requires a
	// full file rewrite under load, which breaks the append-only invariant and
	// introduces a rewrite race (ROADMAP SC#4 specifies reject semantics).
	if len(s.messages) >= MaxChatMessages {
		return relay.ChatMessage{}, ErrChatCapReached
	}

	// Fill defaults for fields the caller left zero.
	if msg.ID == "" {
		id, err := randomHexID()
		if err != nil {
			return relay.ChatMessage{}, fmt.Errorf("chat: cannot generate message ID: %w", err)
		}
		msg.ID = id
	}
	if msg.TimestampMs == 0 {
		msg.TimestampMs = time.Now().UnixMilli()
	}
	msg.SchemaVersion = relay.ChatSchemaVersion
	msg.SessionID = s.sessionID

	// Marshal to a single JSON line.
	line, err := json.Marshal(msg)
	if err != nil {
		return relay.ChatMessage{}, fmt.Errorf("chat: cannot marshal message: %w", err)
	}

	// Reject messages whose serialized line (including the trailing newline)
	// would exceed the read buffer, so the on-disk invariant matches
	// loadFromDisk's cap and every written line is guaranteed replayable on
	// restart (WR-02). Without this, a >1 MiB line is writable but would be
	// skipped on the next load.
	if len(line)+1 > maxChatLineBytes {
		return relay.ChatMessage{}, ErrChatMessageTooLarge
	}

	// Append the line to the JSONL file.  We open+close per call for simplicity
	// and durability: O_APPEND semantics are atomic at the kernel level for
	// writes smaller than PIPE_BUF on most platforms, and the mutex already
	// serializes concurrent callers so ordering is guaranteed.
	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return relay.ChatMessage{}, fmt.Errorf("chat: cannot open %q for append: %w", s.filePath, err)
	}
	lineWithNL := append(line, '\n')
	_, werr := f.Write(lineWithNL)
	cerr := f.Close()
	if werr != nil {
		return relay.ChatMessage{}, fmt.Errorf("chat: write failed: %w", werr)
	}
	if cerr != nil {
		return relay.ChatMessage{}, fmt.Errorf("chat: close failed: %w", cerr)
	}

	// Mirror must be updated AFTER the file write succeeds so the two never
	// diverge (if the write fails the message is not in the mirror).
	s.messages = append(s.messages, msg)

	return msg, nil
}

// Export renders the full message thread to Markdown and returns the document
// as a string. The document contains a header naming the session followed by
// one block per message in chronological order. Each block includes:
//   - AuthorAlias and AuthorID (stable identity preserved for round-trip)
//   - TimestampMs formatted as an ISO-8601 / RFC3339 UTC string
//   - Content body
//   - An explicit "injected into terminal" marker when SessionInject == true
//
// Export() on an empty thread returns a header-only document and nil error.
// The richer YAML-frontmatter format is Phase 155 / EXPORT-01; the contract
// here is field-completeness, not presentation polish.
func (s *ChatStore) Export() (string, error) {
	s.mu.Lock()
	msgs := make([]relay.ChatMessage, len(s.messages))
	copy(msgs, s.messages)
	sessionID := s.sessionID
	s.mu.Unlock()

	var b strings.Builder
	fmt.Fprintf(&b, "# Chat Thread: %s\n\n", sessionID)
	for _, msg := range msgs {
		ts := time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339)
		fmt.Fprintf(&b, "## %s (%s)\n\n", msg.AuthorAlias, ts)
		fmt.Fprintf(&b, "**Author ID:** %s\n\n", msg.AuthorID)
		fmt.Fprintf(&b, "%s\n\n", msg.Content)
		if msg.SessionInject {
			fmt.Fprintf(&b, "_injected into terminal_\n\n")
		}
		fmt.Fprintf(&b, "---\n\n")
	}
	return b.String(), nil
}

// Delete removes the JSONL file from disk and clears the in-memory mirror.
// If the file does not exist (already deleted or never written), Delete returns
// nil — it is idempotent. This is called by KillSession to ensure no orphaned
// chat files remain after a session is torn down (T-151-06 mitigation).
func (s *ChatStore) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("chat: delete %q: %w", s.filePath, err)
	}
	s.messages = nil
	return nil
}

// randomHexID returns a 32-character lowercase hex string generated from
// 16 bytes of crypto/rand. Uses only stdlib — no new dependency.
func randomHexID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
