---
phase: 151-message-schema-chatstore
verified: 2026-06-25T21:30:00Z
status: passed
score: 5/5
behavior_unverified: 0
overrides_applied: 0
---

# Phase 151: Message Schema + ChatStore Verification Report

**Phase Goal:** The daemon can durably persist, replay, and clean up a session's chat thread.
**Verified:** 2026-06-25T21:30:00Z
**Status:** PASSED
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | The daemon writes each chat message to a JSONL file at the daemon data dir under `chats/<sessionID>.jsonl`; after a daemon restart the file is intact and `GET /api/chat/:id/history` returns the full message history | VERIFIED | `chatsDir()` derives from `os.UserConfigDir()+"/agenthub/chats"` (platform-aware: macOS → `~/Library/Application Support/agenthub/chats/`; Linux → `~/.config/agenthub/chats/`). `NewChatStore` calls `loadFromDisk()` at construction. `TestChatStoreRestartSurvival` and `TestChatRoutes_RestartSurvival` both PASS under `-race`. |
| 2 | A participant joining after a restart or late connection receives the complete message history in a single response — no messages are missing | VERIFIED | `Messages()` returns a full copy of the in-memory mirror. `GET /api/chat/{id}/history` marshals all messages as a JSON array; empty thread returns `[]` not null. `TestChatRoutes_RestartSurvival` constructs a second engine over the same chats dir and asserts the pre-restart messages are returned. PASSES `-race`. |
| 3 | Calling `KillSession` on a session removes its JSONL file; no orphaned chat files remain after deletion | VERIFIED | `engine.go` KillSession (lines 572–576) looks up `e.chatStores[id]`, calls `store.Delete()`, and runs `delete(e.chatStores, id)` under `e.mu`. `TestEngineChatStoreFor_AfterKill` asserts `ChatStoreFor(id)` returns ok=false AND the JSONL file is gone from the temp chats dir. PASSES `-race`. |
| 4 | After 10,000 messages the hard cap is enforced and `AppendMessage` rejects further writes — the store does not grow unbounded | VERIFIED | `MaxChatMessages = 10000` const; `ErrChatCapReached` sentinel. AppendMessage enforces `len(s.messages) >= MaxChatMessages` before writing. `TestChatCapEnforcement` appends 10,000 messages, asserts the 10,001st returns `ErrChatCapReached`, and verifies the file has exactly 10,000 lines. PASSES `-race`. |
| 5 | The REST export endpoint returns a Markdown document that round-trips every `ChatMessage` field (author, alias, timestamp, body) without data loss | VERIFIED | `Export()` renders `AuthorAlias`, `AuthorID`, `time.UnixMilli().UTC().Format(time.RFC3339)`, `Content`, and a `_injected into terminal_` marker when `SessionInject==true`. `TestChatExportFields` asserts each field by exact substring match. Both relay loopback (`GET /api/chat/{id}/export`) and web-share cap-gated (`GET /api/chat/{id}/export?cap=X`) surfaces serve the output. PASSES `-race`. |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

**Path note for SC-1:** The success criterion mentions `~/.config/agenthub/chats/` but on macOS `os.UserConfigDir()` returns `~/Library/Application Support`. The implementation correctly uses `os.UserConfigDir()` (platform-aware). The actual production path on macOS is `~/Library/Application Support/agenthub/chats/<sessionID>.jsonl`. This is correct behavior — the success criterion's literal path is Linux-specific example text; the implementation matches the spirit and the verification note's guidance.

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/protocol.go` | `ChatMessage` struct + `ChatSchemaVersion` const | VERIFIED | All 9 fields with correct JSON tags (`v`, `id`, `sessionID`, `authorID`, `alias`, `content`, `mentions`, `sessionInject`, `ts`). `ChatSchemaVersion = 1`. Stability semantics documented in doc comment. |
| `internal/relay/protocol_chat_test.go` | Schema round-trip tests | VERIFIED | `TestChatRoundTrip`, `TestChatJSONKeys`, `TestChatOmitempty`, `TestChatUnknownFieldTolerance`. All PASS under `-race`. |
| `internal/daemon/chat.go` | `ChatStore` type with `NewChatStore`, `AppendMessage`, `Messages`, `Export`, `Delete` | VERIFIED | All methods implemented and substantive. JSONL append-only with mutex spanning file write + mirror append. Idempotent Delete. Markdown Export with all fields. `chatsDir()` production helper present. |
| `internal/daemon/chat_test.go` | Store unit + concurrency (-race) tests | VERIFIED | 15+ tests covering: path derivation, sessionID rejection (7 invalid cases), restart-survival, malformed-line skip, copy isolation, append basics, JSONL round-trip, 10k cap enforcement, 200-goroutine concurrent append, Export (fields + empty), Delete (removes file, idempotent, never-written). All PASS under `-race`. |
| `internal/daemon/engine.go` | `chatStores` map + `chatsBaseDir` + lifecycle wiring | VERIFIED | `chatStores map[string]*ChatStore` field, `chatsBaseDir string` field initialized from `chatsDir()` in `NewSessionEngine`. `ChatsBaseDirForTest` setter, `ChatStoreFor` accessor. `CreateSession` instantiates store (non-fatal on error). `KillSession` calls `store.Delete()` + `delete(e.chatStores, id)` under `e.mu`. |
| `internal/daemon/engine_test.go` | Teardown / no-orphan tests | VERIFIED | `TestEngineNewSessionEngine_ChatStoresInit`, `TestEngineChatStoreFor_AfterCreate`, `TestEngineChatStoreFor_AfterKill` (no-orphan, JSONL gone), `TestEngineChatStoreFor_FailedNewChatStore` (non-fatal), `TestEngineNoRealDirChatFiles`. All PASS under `-race`. |
| `internal/daemon/chat_routes.go` | Desktop relay chat routes | VERIFIED | `handleChatHistory`, `handleChatExport`, `wrapRelayWithChat`. Mounted in `RelayHandler()` as outer wrap: `relay.NewServer → wrapRelayWithRemoteFiles → wrapRelayWithChat`. CORS + OPTIONS preflight per route. No `relay.NewServer` reference in this file. |
| `internal/daemon/chat_routes_test.go` | Relay route tests | VERIFIED | `TestChatRoutes_History` (200 + ordered JSON), `TestChatRoutes_EmptyThread` (`[]` not null), `TestChatRoutes_UnknownSession` (404), `TestChatRoutes_Export` (200 + text/markdown + attachment), `TestChatRoutes_RestartSurvival` (engine2 same dir → same messages). All PASS under `-race`. |
| `internal/webserver/chat.go` | Cap-gated web chat handlers + `SetChatProvider` | VERIFIED | `SetChatProvider` callback pattern; `handleChatHistory`, `handleChatExport`. No `daemon` import (T-151-09 satisfied — confirmed by grep returning no matches). |
| `internal/webserver/chat_test.go` | Capability gate tests | VERIFIED | `TestChatWeb_ValidCap_History` (200), `TestChatWeb_NoCapReturns401`, `TestChatWeb_InvalidCapReturns401`, `TestChatWeb_WrongSessionCapReturns403`, `TestChatWeb_ValidCap_Export`, `TestChatWeb_ProviderSessionNotFound` (404), `TestChatWeb_RequireCapabilityIsWiredForBothRoutes`. All PASS under `-race`. |
| `TESTING.md` | Suite Manifest delta + PERSIST-01/02/03 traceability rows | VERIFIED | Go count updated 350→354. Three traceability rows added. `bash tests/check-traceability-paths.sh` exits 0. |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `NewChatStore(baseDir, sessionID)` | JSONL file at `baseDir/<sessionID>.jsonl` | `baseDir` injected by caller; production passes `chatsDir()`; tests pass `t.TempDir()` | VERIFIED | Containment check: `filepath.Dir(path) == filepath.Clean(baseDir)` at construction. |
| `AppendMessage` | in-memory mirror + JSONL file | `sync.Mutex` spans entire operation (cap check + file write + slice append) | VERIFIED | Mirror append only happens after successful file write. 200-goroutine -race test PASSES. |
| `KillSession` | `store.Delete()` + `delete(e.chatStores, id)` | Inside existing `e.mu.Lock` alongside `tabNames`/`sessionCLIs` deletes | VERIFIED | No orphaned file; recycled session ID starts fresh. |
| `api.go` `RelayHandler()` | `wrapRelayWithChat` | `relay.NewServer → wrapRelayWithRemoteFiles → wrapRelayWithChat` | VERIFIED | `api.go` line 271: `return a.wrapRelayWithChat(withFiles)`. |
| `api.go` `AutoStartWebServer` + `handleWebServerStart` | `ws.SetChatProvider(...)` | Closure over `engine.ChatStoreFor` + `store.Messages()` + `store.Export()` | VERIFIED | Both call sites in `api.go` (lines 472 and 1050) set the provider. |
| `webserver/server.go` `setupRoutes` | `requireCapability` wrapping chat routes | Both `/api/chat/{id}/history` and `/api/chat/{id}/export` wrapped in `ws.requireCapability` | VERIFIED | `server.go` lines 618–619 confirm both routes are registered with the cap middleware. `{id}` wildcard ensures `claims.SID == {id}` enforcement. |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `handleChatHistory` (relay) | `store.Messages()` | `ChatStore.messages` mirror, populated from JSONL file on `NewChatStore` + each `AppendMessage` | Yes — live slice backed by on-disk JSONL | FLOWING |
| `handleChatExport` (relay) | `store.Export()` | Iterates `ChatStore.messages` mirror | Yes — real message data | FLOWING |
| `handleChatHistory` (webserver) | `chatProvider` bytes | Closure in api.go marshals `store.Messages()` → JSON | Yes — same live store | FLOWING |
| `handleChatExport` (webserver) | `chatProvider` markdown | Closure in api.go calls `store.Export()` | Yes — same live store | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go test ./internal/relay/ ./internal/daemon/ ./internal/webserver/ -count=1 -race` | Full package test | All three packages PASS | PASS |
| Cap enforcement: 10k messages + 10,001st rejected | `go test ./internal/daemon/ -run TestChatCapEnforcement -race` | PASS (0.30s) | PASS |
| Concurrent 200-goroutine append | `go test ./internal/daemon/ -run TestChatConcurrentAppend -race` | PASS (0.01s) | PASS |
| Store restart-survival (load from pre-written JSONL) | `go test ./internal/daemon/ -run TestChatStoreRestartSurvival -race` | PASS (0.00s) | PASS |
| Route restart-survival (engine2 same chats dir) | `go test ./internal/daemon/ -run TestChatRoutes_RestartSurvival -race` | PASS (0.02s) | PASS |
| Export round-trips all fields | `go test ./internal/daemon/ -run TestChatExportFields -race` | PASS (0.00s) | PASS |
| KillSession no-orphan | `go test ./internal/daemon/ -run TestEngineChatStoreFor_AfterKill -race` | PASS (0.00s) | PASS |
| Capability gate: no cap → 401, wrong session → 403 | `go test ./internal/webserver/ -run TestChatWeb -race` | All 7 tests PASS | PASS |
| `go build ./...` | Full workspace build | Clean (no output) | PASS |
| `go vet ./internal/relay/ ./internal/daemon/ ./internal/webserver/` | Static analysis | Clean (no output) | PASS |
| `gofmt -l` on chat files | Format check | Clean (no output) | PASS |
| `bash tests/check-traceability-paths.sh` | Traceability paths | OK | PASS |
| `git diff --stat go.mod` | No new dependencies | No change | PASS |

---

### Requirements Coverage

| Requirement | Plans | Description | Status | Evidence |
|-------------|-------|-------------|--------|----------|
| PERSIST-01 | 01, 02, 03 | Chat thread persisted by daemon, survives restart | SATISFIED | `NewChatStore` loads existing JSONL at construction (restart-survival). `AppendMessage` writes each message as JSONL line. GET /api/chat/{id}/history returns full thread. Tests: `TestChatStoreRestartSurvival`, `TestChatRoutes_RestartSurvival`, `TestChatExportFields`. Traceability row in TESTING.md → `internal/daemon/chat_test.go`. |
| PERSIST-02 | 01, 03 | Late join loads full thread scrollback | SATISFIED | `Messages()` returns full in-order copy. GET /api/chat/{id}/history and cap-gated web path both return complete thread. Tests: `TestChatRoutes_RestartSurvival` (engine2 same dir), `TestChatWeb_ValidCap_History`. Traceability row in TESTING.md → `internal/daemon/chat_routes_test.go`. |
| PERSIST-03 | 02, 03 | Thread deleted on session delete; hard message cap | SATISFIED | `KillSession` calls `store.Delete()` + removes map entry. `AppendMessage` enforces 10,000-message cap with reject semantics. Tests: `TestEngineChatStoreFor_AfterKill`, `TestChatCapEnforcement`. Traceability row in TESTING.md → `internal/daemon/engine_test.go`. |

---

### Anti-Patterns Found

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| None | — | — | — |

No TBD/FIXME/XXX markers, placeholder returns, or stub implementations found in phase-modified files. Message content is never logged (`grep -n 'Content' internal/daemon/chat.go | grep -i 'log\|Printf\|Println'` returns no matches). `internal/webserver/chat.go` has no `daemon` import. `internal/daemon/chat_routes.go` has no `relay.NewServer` reference.

---

### Human Verification Required

None. All success criteria are verifiable programmatically. Every Phase 151 behavior has automated test coverage that passes under `-race`.

---

## Gaps Summary

No gaps. All 5 success criteria are verified, all 3 requirement IDs (PERSIST-01/02/03) are covered with traceability rows in TESTING.md, all tests pass under `-race`, and `go build ./...` is clean.

---

_Verified: 2026-06-25T21:30:00Z_
_Verifier: Claude (gsd-verifier)_
