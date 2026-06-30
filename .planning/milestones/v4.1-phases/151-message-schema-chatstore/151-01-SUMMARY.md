---
phase: 151-message-schema-chatstore
plan: "01"
subsystem: daemon/relay
tags: [chat, persistence, schema, jsonl, tdd]
dependency_graph:
  requires: []
  provides:
    - relay.ChatMessage struct and ChatSchemaVersion const
    - daemon.ChatStore (NewChatStore, AppendMessage, Messages)
    - daemon.chatsDir() production helper
    - daemon.MaxChatMessages cap, daemon.ErrChatCapReached sentinel
  affects:
    - internal/relay/protocol.go (new exported types)
    - internal/daemon/chat.go (new file)
tech_stack:
  added: []
  patterns:
    - JSONL append-only persistence (one line per ChatMessage)
    - Injected baseDir for constructor-level test isolation
    - sync.Mutex spanning file write + mirror append (no divergence)
    - crypto/rand hex IDs (stdlib only, no new Go module)
    - TDD RED/GREEN/REFACTOR per task
key_files:
  created:
    - internal/relay/protocol_chat_test.go
    - internal/daemon/chat.go
    - internal/daemon/chat_test.go
  modified:
    - internal/relay/protocol.go
decisions:
  - "Reject-not-trim at cap: AppendMessage rejects the 10001st write (ErrChatCapReached) rather than trimming the oldest message; trimming would require rewriting the entire JSONL file under concurrent load, destroying the append-only invariant"
  - "Injected baseDir: NewChatStore accepts baseDir as a parameter (not derived internally from daemonConfigDir); production callers pass chatsDir(), tests pass t.TempDir() — constructor-level isolation with zero special-casing in test code"
  - "Open/close per AppendMessage call: simplest correct approach; mutex already serializes concurrent callers; O_APPEND kernel semantics provide atomicity for writes smaller than PIPE_BUF"
  - "Strict sessionID allowlist [A-Za-z0-9_-]: defense-in-depth against path traversal; containment check (filepath.Dir == chatsDir) as second layer"
metrics:
  duration: "~6 minutes"
  completed: "2026-06-25"
  tasks_completed: 3
  tasks_total: 3
  files_changed: 4
status: complete
---

# Phase 151 Plan 01: Message Schema + ChatStore Summary

**One-liner:** JSONL-backed ChatStore with 10k-cap append, restart-survival load, and path-traversal-hardened sessionID validation, anchored by a stable ChatMessage wire contract in relay/protocol.go.

## What Was Built

### Task 1: ChatMessage schema in protocol.go (TDD)

Added `relay.ChatMessage` struct and `relay.ChatSchemaVersion = 1` const to `internal/relay/protocol.go`. The struct carries all nine fields with their exact JSON tags (`v`, `id`, `sessionID`, `authorID`, `alias`, `content`, `mentions`, `sessionInject`, `ts`). The doc comment explicitly documents the stability semantics: `AuthorID` is the stable routing identity, `AuthorAlias` is a per-message snapshot label, `TimestampMs` is UNIX milliseconds UTC. The struct is the stable wire contract that Plans 02–03 and all later phases serialize to.

Test file `internal/relay/protocol_chat_test.go` covers: full round-trip equality via `reflect.DeepEqual`, exact JSON key name verification, `omitempty` for `Mentions` and `SessionInject`, and unknown-field forward-compatibility (Go default behavior locked by a test).

### Task 2: ChatStore construction, sessionID hardening, restart-survival load (TDD)

Created `internal/daemon/chat.go` with:
- `chatsDir()` — production helper returning `filepath.Join(daemonConfigDir(), "chats")`
- `MaxChatMessages = 10000` const and `ErrChatCapReached` sentinel
- `validChatSessionID(id)` — strict allowlist `[A-Za-z0-9_-]`, rejects empty, `.`, `..`, any separator or NUL
- `ChatStore` struct with `sync.Mutex`, `filePath`, `sessionID`, `messages []relay.ChatMessage` mirror
- `NewChatStore(baseDir, sessionID)` — validates sessionID, `MkdirAll(baseDir)`, containment check via `filepath.Dir(path) == filepath.Clean(baseDir)`, loads existing JSONL with `bufio.Scanner` (malformed lines skipped)
- `Messages()` — returns copy of mirror under mutex

Tests cover: path derivation with `t.TempDir()` baseDir, rejection of 7 invalid sessionIDs (empty, `../escape`, `a/b`, `a\\b`, `..`, `.`, traversal paths), restart-survival with 3 pre-written JSONL lines, malformed-line skip, `Messages()` copy isolation.

### Task 3: AppendMessage with 10k cap and concurrent-write safety (TDD)

Added `AppendMessage(msg relay.ChatMessage) (relay.ChatMessage, error)` to `ChatStore`. The method:
1. Acquires the mutex for the ENTIRE operation (cap check + file write + mirror append)
2. Enforces the cap at the beginning (REJECT, not trim)
3. Fills defaults: `ID` from `crypto/rand` hex (16 bytes, 32 chars), `TimestampMs` from `time.Now().UnixMilli()`, `SchemaVersion = relay.ChatSchemaVersion`, `SessionID = s.sessionID`
4. Marshals to JSON, opens the JSONL file with `O_APPEND|O_CREATE|O_WRONLY`, writes line + `\n`, closes
5. Appends to mirror only after successful file write

Tests cover: cap enforcement (10000 appends, 10001st returns `ErrChatCapReached`, file has exactly 10000 lines), concurrent 200-goroutine append under `-race` (all succeed, file count and mirror length both equal 200), round-trip JSONL decode, basic defaults assignment.

## Verification Results

```
go test ./internal/relay/ ./internal/daemon/ -race   PASS
go vet ./internal/relay/ ./internal/daemon/           clean
gofmt -l ...                                          clean (no output)
git diff --stat go.mod                                no change (zero new dependencies)
```

## Deviations from Plan

None — plan executed exactly as written. All three TDD tasks followed RED/GREEN/REFACTOR:
- Task 1: tests failed (undefined `ChatSchemaVersion`/`ChatMessage`) → implemented struct + const → all pass
- Task 2: tests failed (undefined `NewChatStore`/`chatsDir`) → implemented `chat.go` with construction + load → all pass
- Task 3: tests failed (undefined `AppendMessage`) → implemented method → all pass under `-race`

## Threat Mitigations Applied

| Threat | Mitigation |
|--------|-----------|
| T-151-01: sessionID path traversal | `validChatSessionID` allowlist + `filepath.Dir` containment check; no file created before validation |
| T-151-02: unbounded store growth | `MaxChatMessages = 10000` enforced atomically at AppendMessage entry; reject semantics (not trim) |
| T-151-03: concurrent JSONL write race | `sync.Mutex` spans file write + mirror append; 200-goroutine `-race` test gate |
| T-151-05: content in logs | `AppendMessage` logs only metadata; `grep -n 'Content' chat.go \| grep -i 'log\|Printf\|Println'` returns nothing |
| T-151-SC: supply chain | Zero new Go modules added; `go.mod` unchanged |

## Known Stubs

None. The store is fully wired with no placeholder data.

## Self-Check: PASSED
