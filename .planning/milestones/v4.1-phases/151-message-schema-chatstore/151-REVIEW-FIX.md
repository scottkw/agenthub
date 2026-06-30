---
phase: 151-message-schema-chatstore
fixed_at: 2026-06-25T22:30:00Z
review_path: .planning/phases/151-message-schema-chatstore/151-REVIEW.md
iteration: 2
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 151: Code Review Fix Report

**Fixed at:** 2026-06-25T22:30:00Z (iteration 2: Info findings) · 2026-06-25T21:40:00Z (iteration 1: Warnings)
**Source review:** .planning/phases/151-message-schema-chatstore/151-REVIEW.md
**Iterations:** 2

**Summary:**
- Findings in scope: 7 (0 Critical, 3 Warning, 4 Info — full `--fix --all` scope)
- Fixed: 7 (3 Warning in iteration 1, 4 Info in iteration 2)
- Skipped: 0

> **Recovery note:** The iteration-2 fixer agent applied all 4 Info fixes as atomic
> commits in an isolated worktree, then its connection dropped before completing the
> transactional fast-forward of `main`. The orchestrator verified the worktree was
> clean and green (`go build ./...` + affected package tests), confirmed `main` was a
> clean ancestor, fast-forwarded `main` to the fix HEAD (`1ce7dab3`), removed the
> worktree/branch, and cleared the recovery sentinel. Full `go test ./... -count=1`
> passes on `main` after recovery.

All fixes were verified with `gofmt`, `go build ./...`, and the affected package
test suites (`go test ./internal/daemon/ ./internal/webserver/ -count=1`), each of
which passed after every change. New regression tests were added for each finding.

## Fixed Issues

### WR-01: `Messages()` shallow copy aliased the `Mentions` slice

**Files modified:** `internal/daemon/chat.go`, `internal/daemon/chat_test.go`
**Commit:** 8d4e54cb
**Applied fix:** `Messages()` now deep-copies the `Mentions []string` field for each
returned `ChatMessage` (`append([]string(nil), src...)`) after the shallow
`copy(result, s.messages)`. Previously every returned element's `Mentions` shared
its backing array with the in-memory mirror, so `msgs[i].Mentions[j] = ...`
corrupted internal state. Added `TestChatStoreMessagesDeepCopiesMentions`, which
mutates a returned message's `Mentions` element and asserts a subsequent
`Messages()` read is unchanged (the prior `TestChatStoreMessagesReturnsCopy` only
mutated the immutable `Content` string and did not catch this). The internal
`Export()` copy was left as-is — that copy never escapes the store, so it is not an
aliasing hazard.

### WR-02: An oversized JSONL line aborted loading the entire thread on restart

**Files modified:** `internal/daemon/chat.go`, `internal/daemon/chat_test.go`
**Commit:** 509ecd0a
**Applied fix:** Two coordinated changes so the read path and write path agree on a
single shared `maxChatLineBytes` (1 MiB) bound:
- **Read path:** Replaced the `bufio.Scanner` in `loadFromDisk` (which stops
  entirely and returns `bufio.ErrTooLong` on the first over-length line, dropping
  every subsequent message and failing `NewChatStore`) with a new
  `readCappedLine(*bufio.Reader, max)` helper built on `bufio.Reader.ReadLine`. An
  over-length line is now fully consumed and skipped — exactly like a
  malformed-JSON line — and loading continues with the following line. A
  metadata-only log line is emitted (never message content). The whole thread is
  no longer lost to one bad line on restart.
- **Write path:** `AppendMessage` now rejects any message whose serialized line
  (plus the trailing newline) would exceed `maxChatLineBytes`, returning the new
  `ErrChatMessageTooLarge` and writing nothing. This keeps the on-disk invariant
  in sync with the reader so every persisted line is guaranteed replayable.

Added `TestChatStoreOversizedLineSkip` (an oversized valid-JSON line between two
good messages is skipped, both neighbors load, `NewChatStore` does not fail) and
`TestChatAppendRejectsOversized` (an oversized append returns
`ErrChatMessageTooLarge`, persists nothing, leaves the mirror empty).

### WR-03: `chatProvider` mapped internal serialization/export errors to 404

**Files modified:** `internal/webserver/server.go`, `internal/webserver/chat.go`,
`internal/webserver/chat_test.go`, `internal/daemon/api.go`
**Commit:** 0c19c150
**Applied fix:** Widened the provider callback signature from
`func(sessionID) (history []byte, markdown string, ok bool)` to
`func(sessionID) (history []byte, markdown string, found bool, err error)` so the
handlers can distinguish "no store" (404) from "internal error on an existing
session" (500). The webserver handlers (`handleChatHistory`, `handleChatExport`)
now return `500 internal error` when `err != nil` and only return
`404 session not found` when `err == nil && !found`. Both daemon provider closures
in `api.go` (`AutoStartWebServer` and `handleWebServerStart`) now log the
underlying `json.Marshal` / `store.Export()` error and propagate it (with
`found == true`) instead of silently collapsing it into the not-found path —
removing the silent fallback flagged by CLAUDE.md. Updated the four existing
webserver test stubs to the new signature and added
`TestChatWeb_ProviderInternalError`, which asserts both routes return 500 (not 404)
when the provider reports an error for an existing session. No import cycle is
introduced — the callback still exchanges only `[]byte`/`string`/`bool`/`error`
(T-151-09 preserved).

## Fixed Issues — Iteration 2 (Info, `--fix --all`)

### IN-01: Dead `msgs == nil` guard after `Messages()`

**Files modified:** `internal/daemon/chat_routes.go`
**Commit:** 749ac77a
**Applied fix:** Removed the unreachable `if msgs == nil { msgs = []relay.ChatMessage{} }`
guard. `ChatStore.Messages()` builds its result with `make([]relay.ChatMessage, len)`,
so it never returns nil and an empty thread already serializes as `[]` (not `null`).
Replaced the dead branch with a comment documenting the invariant. The `api.go`
provider paths were superseded by the IN-04 split (below), so no separate guard
removal was needed there.

### IN-02: Export route interpolated an unvalidated path param into `Content-Disposition`

**Files modified:** `internal/daemon/chat_routes.go`, `internal/daemon/chat_routes_test.go`
**Commit:** 96b05227
**Applied fix:** Defense-in-depth — the relay chat routes now reject any `id` that
fails the `validChatSessionID` allowlist (same allowlist `NewChatStore` uses) with a
404 before touching the store or building the `Content-Disposition` header. Added
`TestChatRoutes_InvalidSessionID`, which asserts an invalid id returns 404 on the
export route. Header injection was not reachable before (keys are crypto-random hex),
but the safety is now local and explicit rather than relying on an upstream invariant.

### IN-03: `NewChatStore` filesystem I/O held the engine-wide `e.mu`

**Files modified:** `internal/daemon/engine.go`
**Commit:** eb8d69fd
**Applied fix:** `CreateSession` now snapshots `e.chatsBaseDir` under a short `e.mu`
hold, releases the lock, constructs the `ChatStore` (`os.MkdirAll` + possible
`loadFromDisk`) outside the lock, then re-takes `e.mu` only to insert the ready
pointer. Concurrent `ChatStoreFor`/`ListSessions` readers no longer block on chat
filesystem I/O. The non-fatal-on-error behavior is preserved, and reading
`chatsBaseDir` under the lock keeps it race-free with `ChatsBaseDirForTest`.

### IN-04: Provider computed both history JSON and Markdown on every call

**Files modified:** `internal/daemon/api.go`, `internal/webserver/server.go`,
`internal/webserver/chat.go`, `internal/webserver/chat_test.go`
**Commit:** 1ce7dab3
**Applied fix:** Split the single `SetChatProvider` closure into two narrow callbacks —
`SetChatHistoryProvider` (`func(sessionID) (history []byte, found bool, err error)`)
and `SetChatExportProvider` (`func(sessionID) (markdown string, found bool, err error)`).
Each route now does only the work it serves (history route no longer runs `Export()`;
export route no longer marshals history), and each computation takes its own single
`store.mu` hold. WR-03's error semantics are preserved exactly: `err != nil` → 500 on
an existing session, `err == nil && !found` → 404. Both daemon closures in `api.go`
and the webserver test stubs were updated to the split signatures.

All iteration-2 fixes verified with `gofmt`, `go build ./...`, and
`go test ./internal/daemon/ ./internal/webserver/ ./internal/relay/ -count=1` (green),
plus a full `go test ./... -count=1` on `main` after worktree recovery.

---

_Fixed: 2026-06-25T22:30:00Z (iteration 2) · 2026-06-25T21:40:00Z (iteration 1)_
_Fixer: Claude (gsd-code-fixer) + orchestrator worktree recovery_
_Iterations: 2_
