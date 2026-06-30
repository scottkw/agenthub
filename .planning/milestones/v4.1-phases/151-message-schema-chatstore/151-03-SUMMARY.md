---
phase: 151-message-schema-chatstore
plan: "03"
subsystem: daemon/relay, webserver
tags: [chat, rest, capability, persist, relay, webserver, loopback, cors]
dependency_graph:
  requires:
    - 151-02  # ChatStore with AppendMessage/Export/Delete + engine wiring
    - 151-01  # ChatMessage schema + relay.ChatSchemaVersion
  provides:
    - REST chat history+export on relay loopback (desktop GUI surface)
    - REST chat history+export on webserver (cap-gated, web-share surface)
    - TESTING.md Phase 151 traceability rows for PERSIST-01/02/03
  affects:
    - internal/daemon/api.go (RelayHandler wrap chain; AutoStartWebServer + handleWebServerStart wiring)
    - internal/webserver/server.go (chatProvider field; setupRoutes registrations)
tech_stack:
  added: []
  patterns:
    - wrap-mux relay extension (mirrors relay_remote_files.go wrapRelayWithRemoteFiles pattern)
    - provider-callback coupling (mirrors SetSessionResolver / SetFilesHandler — no import cycle)
    - requireCapability path-ID enforcement (claims.SID == {id} per-session isolation)
key_files:
  created:
    - internal/daemon/chat_routes.go
    - internal/daemon/chat_routes_test.go
    - internal/webserver/chat.go
    - internal/webserver/chat_test.go
  modified:
    - internal/daemon/api.go
    - internal/webserver/server.go
    - TESTING.md
decisions:
  - "Relay chat routes mounted in wrapRelayWithChat (parent mux, outer layer) to avoid daemon↔relay import cycle — exact same pattern as wrapRelayWithRemoteFiles"
  - "Webserver uses provider callback (func(sessionID)([]byte,string,bool)) not daemon.ChatStore import — T-151-09 mitigation"
  - "chat_routes_test.go uses package daemon (internal) to insert stores directly without spawning real PTY processes"
  - "requireCapability path-ID check enforces claims.SID=={id} for per-session isolation (T-151-04)"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-25T21:01:17Z"
  tasks_completed: 3
  tasks_total: 3
  files_created: 4
  files_modified: 3
status: complete
---

# Phase 151 Plan 03: Chat REST Routes (Relay + Webserver) Summary

REST chat history and export endpoints on both client surfaces: a CORS-wrapped loopback relay pair for the desktop GUI and a capability-gated HTTPS pair for web-share peers. TESTING.md updated with Phase 151 Suite Manifest delta and PERSIST-01/02/03 traceability rows.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Desktop relay chat routes (history + export) | 35b09bd4 | chat_routes.go, chat_routes_test.go, api.go |
| 2 | Capability-gated web chat routes via provider callback | 85ff6af3 | chat.go, chat_test.go, server.go |
| 3 | TESTING.md Suite Manifest + PERSIST-01/02/03 traceability | 8513cab4 | TESTING.md |

## What Was Built

### Task 1 — Relay Loopback Chat Routes (PERSIST-01, PERSIST-02)

**`internal/daemon/chat_routes.go`**

- `handleChatHistory`: reads `r.PathValue("id")`, calls `engine.ChatStoreFor`, marshals `store.Messages()` as JSON; empty thread returns `[]` not null; unknown session returns 404.
- `handleChatExport`: calls `store.Export()`, sets `Content-Type: text/markdown; charset=utf-8` and `Content-Disposition: attachment; filename="chat-{id}.md"`.
- `wrapRelayWithChat`: parent mux registering both GET routes wrapped in `relay.FilesCORS` (cross-origin webview→127.0.0.1 boundary) plus OPTIONS preflight routes, with `/` fallthrough to the inner handler. Called from `RelayHandler()` as the outermost wrap layer.

**`internal/daemon/api.go` changes**

- `RelayHandler()` now chains: `relay.NewServer → wrapRelayWithRemoteFiles → wrapRelayWithChat`.
- `AutoStartWebServer` and `handleWebServerStart` both call `ws.SetChatProvider(...)` with a closure over `engine.ChatStoreFor` + `store.Messages()` + `store.Export()`.

**`internal/daemon/chat_routes_test.go`** — package daemon (internal, no real PTY)

Tests: `TestChatRoutes_History` (200 + JSON array in order), `TestChatRoutes_EmptyThread` ([] not null), `TestChatRoutes_UnknownSession` (404 for both routes), `TestChatRoutes_Export` (200 + text/markdown + attachment + content body), `TestChatRoutes_RestartSurvival` (engine2 with same chats dir → same messages), `TestChatRoutes_BuildsWithoutRelayNewServer` (compile-time gate).

### Task 2 — Webserver Capability-Gated Chat Routes (PERSIST-01, PERSIST-02)

**`internal/webserver/chat.go`**

- `SetChatProvider(fn func(string)([]byte,string,bool))`: single setter, set once before Start, not mutex-protected (mirrors SetFilesHandler pattern). No daemon import — T-151-09.
- `handleChatHistory`: reads provider history bytes, writes JSON.
- `handleChatExport`: reads provider markdown string, writes text/markdown with Content-Disposition attachment.

**`internal/webserver/server.go` changes**

- Added `chatProvider` field to `WebServer` struct.
- `setupRoutes()` registers both routes under `ws.requireCapability`: `GET /api/chat/{id}/history` and `GET /api/chat/{id}/export`. The `{id}` wildcard causes `requireCapability` to enforce `claims.SID == {id}` (T-151-04 per-session isolation).

**`internal/webserver/chat_test.go`** — package webserver (internal, uses testServer + issueCapFor)

Tests: `TestChatWeb_ValidCap_History` (200 + provider bytes), `TestChatWeb_NoCapReturns401`, `TestChatWeb_InvalidCapReturns401`, `TestChatWeb_WrongSessionCapReturns403`, `TestChatWeb_ValidCap_Export` (200 + text/markdown), `TestChatWeb_ProviderSessionNotFound` (404 when provider returns ok=false), `TestChatWeb_RequireCapabilityIsWiredForBothRoutes`.

### Task 3 — TESTING.md (Standing Convention)

- Suite Manifest Go count: 350 → **354**; Total: 473 → **482** (Phase 150 accumulated delta + Phase 151 +4 Go).
- Note updated with per-file Phase 151 descriptions.
- Traceability rows added:
  - `PERSIST-01` → `internal/daemon/chat_test.go`
  - `PERSIST-02` → `internal/daemon/chat_routes_test.go`
  - `PERSIST-03` → `internal/daemon/engine_test.go`
- `bash tests/check-traceability-paths.sh` exits 0.

## Deviations from Plan

None — plan executed exactly as written.

The test file `chat_routes_test.go` uses `package daemon` (internal) rather than `package daemon_test` because direct field access (`e.chatStores`, `e.mu`) is required to insert stores without spawning real PTY processes. The plan did not prescribe a specific package, and the internal approach is more robust in CI.

## Threat Coverage

| Threat ID | Mitigation Implemented |
|-----------|------------------------|
| T-151-04 | `requireCapability` on both web routes with `{id}` path wildcard enforces `claims.SID == {id}`; test asserts 403 for wrong-session cap |
| T-151-08 | `relay.FilesCORS` wraps both relay routes; OPTIONS preflight registered |
| T-151-09 | `grep -n 'daemon' internal/webserver/chat.go` → no matches; provider callback only |

## Verification Results

```
go test ./internal/daemon/ ./internal/webserver/ ./internal/relay/ -race   → PASS
bash tests/check-traceability-paths.sh                                     → OK
gofmt -l internal/daemon/chat_routes.go internal/webserver/chat.go        → (empty)
git diff --stat go.mod                                                      → (no change)
grep -n 'relay.NewServer' internal/daemon/chat_routes.go                   → (empty)
grep -n 'daemon' internal/webserver/chat.go                                → (empty)
```

## Self-Check: PASSED

Files exist:
- FOUND: internal/daemon/chat_routes.go
- FOUND: internal/daemon/chat_routes_test.go
- FOUND: internal/webserver/chat.go
- FOUND: internal/webserver/chat_test.go

Commits exist:
- FOUND: 35b09bd4 (relay routes + tests)
- FOUND: 85ff6af3 (webserver routes + tests)
- FOUND: 8513cab4 (TESTING.md)
