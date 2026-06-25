---
phase: 151-message-schema-chatstore
fixed_at: 2026-06-25T21:40:00Z
review_path: .planning/phases/151-message-schema-chatstore/151-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 151: Code Review Fix Report

**Fixed at:** 2026-06-25T21:40:00Z
**Source review:** .planning/phases/151-message-schema-chatstore/151-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 3 (0 Critical, 3 Warning — Info findings IN-01..04 are out of scope for `critical_warning`)
- Fixed: 3
- Skipped: 0

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

---

_Fixed: 2026-06-25T21:40:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
