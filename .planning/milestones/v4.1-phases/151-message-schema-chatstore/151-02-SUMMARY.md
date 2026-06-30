---
phase: 151-message-schema-chatstore
plan: "02"
subsystem: daemon
tags: [chat, persistence, export, lifecycle, tdd]
dependency_graph:
  requires:
    - relay.ChatMessage struct (relay/protocol.go — Plan 01)
    - daemon.ChatStore with AppendMessage/Messages (daemon/chat.go — Plan 01)
    - daemon.chatsDir() helper (daemon/chat.go — Plan 01)
  provides:
    - daemon.(*ChatStore).Export() — full-thread Markdown renderer
    - daemon.(*ChatStore).Delete() — idempotent file removal + mirror clear
    - daemon.SessionEngine.chatStores map (guarded by e.mu)
    - daemon.SessionEngine.chatsBaseDir field (init from chatsDir())
    - daemon.(*SessionEngine).ChatsBaseDirForTest(dir) — test-only setter
    - daemon.(*SessionEngine).ChatStoreFor(sessionID) — read accessor
    - CreateSession ChatStore instantiation
    - KillSession ChatStore teardown (no-orphan guarantee)
  affects:
    - internal/daemon/chat.go (Export + Delete methods added)
    - internal/daemon/engine.go (chatStores + chatsBaseDir fields + lifecycle wiring)
    - internal/daemon/engine_test.go (wiring tests)
tech_stack:
  added: []
  patterns:
    - Mutex snapshot for Export (lock → copy → unlock → build string)
    - Idempotent Delete (os.IsNotExist treated as success)
    - Non-fatal ChatStore creation in CreateSession (chat is a side channel)
    - Injected chatsBaseDir enables constructor-level test isolation
    - TDD RED/GREEN/REFACTOR per task
key_files:
  created: []
  modified:
    - internal/daemon/chat.go
    - internal/daemon/chat_test.go
    - internal/daemon/engine.go
    - internal/daemon/engine_test.go
decisions:
  - "Mutex snapshot in Export: lock → copy slice → unlock → build string outside the lock; avoids holding the mutex during string building while preserving snapshot consistency"
  - "Delete idempotency: os.IsNotExist treated as success so KillSession teardown is safe to call even when AppendMessage was never invoked (store file never created)"
  - "Non-fatal ChatStore in CreateSession: NewChatStore failure logs metadata only (sessionID + error) and allows session creation to proceed; chat is a side channel, not a prerequisite for PTY operation"
  - "Separate chatsBaseDir from configDir: ConfigDirForTest only affects settings persistence (settingsPath); it does NOT affect chatsDir(), so engine wiring tests need ChatsBaseDirForTest to redirect chat files away from the real daemon data dir"
  - "delete(e.chatStores, id) in KillSession under e.mu: mirrors the existing sessionBrowse delete so a recycled session ID starts with a fresh empty store, never inheriting a stale thread"
metrics:
  duration: "~8 minutes"
  completed: "2026-06-25"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 4
status: complete
---

# Phase 151 Plan 02: ChatStore Export/Delete + Engine Lifecycle Summary

**One-liner:** Markdown Export and idempotent Delete added to ChatStore; SessionEngine gains a chatStores map with Create/Kill lifecycle wiring that leaves zero orphaned JSONL files.

## What Was Built

### Task 1: ChatStore Export (Markdown) and Delete (teardown)

Added two methods to `internal/daemon/chat.go`:

**`Export() (string, error)`** — renders the full message thread to a valid GitHub-renderable Markdown document:
- Header: `# Chat Thread: <sessionID>`
- One block per message (chronological): `AuthorAlias`, `AuthorID` (stable routing identity preserved for round-trip), `TimestampMs` rendered as `time.RFC3339` UTC via `time.UnixMilli(...).UTC().Format(time.RFC3339)`, `Content` body
- When `SessionInject == true`: `_injected into terminal_` marker appears in that block
- Empty thread: returns header-only document and `nil` error
- Implementation: mutex lock → copy mirror slice → unlock → build with `strings.Builder` + `fmt.Fprintf`

**`Delete() error`** — removes the JSONL file from disk and clears the in-memory mirror:
- `os.Remove(s.filePath)` with `os.IsNotExist` treated as success (idempotent teardown)
- Sets `s.messages = nil` so `Messages()` returns length 0 after deletion
- Called by `KillSession` to satisfy the no-orphan guarantee (T-151-06)

Tests added to `chat_test.go`:
- `TestChatExportFields` — 3 messages (1 with `SessionInject=true`), asserts each field (alias, authorID, RFC3339 timestamp substring, content, inject marker) present in output
- `TestChatExportEmpty` — empty thread returns non-empty header-only document
- `TestChatDeleteRemovesFile` — file gone (`os.IsNotExist`), `Messages()` length 0
- `TestChatDeleteIdempotent` — second `Delete()` returns nil
- `TestChatDeleteOnNeverWrittenStore` — `Delete()` on a store with no file returns nil

### Task 2: Wire chatStores into SessionEngine lifecycle

Modified `internal/daemon/engine.go`:

**New fields on `SessionEngine`:**
- `chatStores map[string]*ChatStore` — per-session store map, guarded by `e.mu`
- `chatsBaseDir string` — base directory for chat JSONL files; initialized in `NewSessionEngine` to `chatsDir()` (= `daemonConfigDir()/chats`) keeping the production path identical to before

**New `NewSessionEngine` initialization:**
```go
chatStores:   make(map[string]*ChatStore),
chatsBaseDir: chatsDir(),
```

**`ChatsBaseDirForTest(dir string)`** — test-only setter (mirrors `ConfigDirForTest`); sets `e.chatsBaseDir = dir` under `e.mu`; production code never calls this.

**`ChatStoreFor(sessionID string) (*ChatStore, bool)`** — read accessor using `e.mu.RLock`; used by Plan 03 REST endpoints.

**`CreateSession` wiring** (under existing `e.mu.Lock`):
```go
if store, storeErr := NewChatStore(e.chatsBaseDir, id); storeErr != nil {
    log.Printf("chat: NewChatStore for session %q: %v (...)", id, storeErr)
} else {
    e.chatStores[id] = store
}
```
Chat-store failure is non-fatal — session creation proceeds. Only metadata (sessionID + error) is logged, never content (T-151-05).

**`KillSession` wiring** (inside existing `e.mu.Lock` alongside `tabNames`/`sessionCLIs` deletes):
```go
if store, ok := e.chatStores[id]; ok {
    if delErr := store.Delete(); delErr != nil {
        log.Printf("chat: Delete for session %q: %v", id, delErr)
    }
    delete(e.chatStores, id)
}
```
Ensures no orphaned JSONL file and no stale thread on recycled session IDs (T-151-06, T-151-07).

Tests added to `engine_test.go` (all call `e.ChatsBaseDirForTest(t.TempDir())` to redirect away from real data dir):
- `TestEngineNewSessionEngine_ChatStoresInit` — map non-nil, chatsBaseDir non-empty
- `TestEngineChatStoreFor_AfterCreate` — `ChatStoreFor(id)` ok=true after `CreateSession`
- `TestEngineChatStoreFor_AfterKill` — `ChatStoreFor(id)` ok=false AND no JSONL file after `KillSession`
- `TestEngineChatStoreFor_FailedNewChatStore` — unreachable chatsBaseDir, `CreateSession` still returns valid session, `ChatStoreFor` ok=false
- `TestEngineNoRealDirChatFiles` — full lifecycle under temp dir, no JSONL created under real `chatsDir()`

## Verification Results

```
go test ./internal/daemon/ -race            PASS (all tests, including all 151-01 tests)
gofmt -l internal/daemon/chat.go engine.go  clean (no output)
go vet ./internal/daemon/                   clean
git diff --stat go.mod                      no change (zero new Go modules)
```

## Deviations from Plan

None — plan executed exactly as written. All two TDD tasks followed RED/GREEN/REFACTOR:
- Task 1: tests failed (Export/Delete undefined) → implemented Export+Delete → all pass
- Task 2: tests failed (ChatsBaseDirForTest/ChatStoreFor/chatStores undefined) → wired engine → all pass

## Threat Mitigations Applied

| Threat | Mitigation |
|--------|-----------|
| T-151-06: orphaned JSONL after KillSession | `KillSession` calls `store.Delete()` + `delete(e.chatStores, id)` under `e.mu`; no-orphan test asserts file absence in temp dir |
| T-151-07: recycled session ID inherits stale thread | `delete(e.chatStores, id)` in KillSession mirrors the existing `sessionBrowse` delete; new session ID starts with empty store |
| T-151-05: chat content in logs | CreateSession chat-store error logs only `sessionID + error`, never Content |

## Known Stubs

None. Export, Delete, and engine lifecycle wiring are fully implemented with no placeholder data.

## Threat Flags

None. No new network endpoints, auth paths, or file access patterns introduced beyond what the plan specified.

## Self-Check: PASSED
