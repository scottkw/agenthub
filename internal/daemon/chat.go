package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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

	// Use a generous buffer to handle long messages (1 MB per line).
	const maxLineBytes = 1 << 20
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), maxLineBytes)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg relay.ChatMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			// Malformed line — skip, continue loading the rest.
			continue
		}
		s.messages = append(s.messages, msg)
	}
	return scanner.Err()
}

// Messages returns a copy of the full in-order message thread. The copy
// prevents callers from mutating internal state. This method is safe to call
// concurrently with AppendMessage.
func (s *ChatStore) Messages() []relay.ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return a copy so callers cannot mutate s.messages.
	result := make([]relay.ChatMessage, len(s.messages))
	copy(result, s.messages)
	return result
}
