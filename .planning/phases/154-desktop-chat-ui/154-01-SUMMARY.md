---
phase: 154-desktop-chat-ui
plan: "01"
subsystem: relay-chat-dispatch
tags: [chat, relay, webserver, sec-01, tdd]
status: complete

dependency_graph:
  requires: []
  provides:
    - relay.ChatSendPayload
    - relay.ErrChatReadOnly
    - relay.Hub.HandleChatSend
    - relay/server.go case MsgChatSend
    - webserver/server.go case MsgChatSend
  affects:
    - internal/relay/protocol.go
    - internal/relay/hub.go
    - internal/relay/server.go
    - internal/webserver/server.go

tech_stack:
  added: []
  patterns:
    - TDD (RED→GREEN) for all new behavior
    - unlock-before-IO discipline (Pitfall 4) in HandleChatSend
    - silent-drop error policy for chat send (no NAK per RESEARCH Open Question 1)

key_files:
  created:
    - internal/relay/hub_chatsend_test.go
    - internal/relay/server_chatsend_test.go
    - internal/webserver/server_chatsend_test.go
  modified:
    - internal/relay/protocol.go
    - internal/relay/hub.go
    - internal/relay/server.go
    - internal/webserver/server.go

decisions:
  - HandleChatSend uses SanitizeChatContent (not SanitizePTYText) — chat frames are display surfaces, not PTY stdin
  - Silent-drop on HandleChatSend error (no MsgInjectError NAK) per RESEARCH Open Question 1 — chat send has no client error display path in Phase 154
  - ErrChatReadOnly is a distinct error from ErrReadOnly to enable precise error-type matching in tests and future callers
  - chatAppendFn nil-check returns error (fail-loud, not silent) — matches "let it crash" CLAUDE.md principle

metrics:
  duration: 6 minutes
  completed: "2026-06-26"
  tasks_completed: 2
  files_modified: 4
  files_created: 3
---

# Phase 154 Plan 01: MsgChatSend Server-Side Dispatch Summary

**One-liner:** MsgChatSend (0x31) dispatch wired in relay + webserver read pumps via Hub.HandleChatSend with ErrChatReadOnly SEC-01 gate and SanitizeChatContent sanitization; no PTY write ever.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Failing tests for Hub.HandleChatSend | 5fb8b3ca | hub_chatsend_test.go |
| 1 (GREEN) | ChatSendPayload + Hub.HandleChatSend | 7a8074e0 | protocol.go, hub.go |
| 2 (RED) | Failing read-pump dispatch tests | d71c76ae | server_chatsend_test.go, webserver/server_chatsend_test.go |
| 2 (GREEN) | case MsgChatSend in both read pumps | 91203a34 | relay/server.go, webserver/server.go |

## What Was Built

### ChatSendPayload (protocol.go)
New struct `ChatSendPayload{ Content string json:"content" }` placed alongside `InjectPayload`. This is the client-to-server wire type for MsgChatSend (0x31) frames.

### ErrChatReadOnly (hub.go)
New error variable `"relay: chat rejected: read-only capability"` near `ErrReadOnly`. Distinct to allow precise `errors.Is` matching.

### Hub.HandleChatSend (hub.go)
Modeled on `HandleInject` with three key differences:
1. Gates on `!sub.ReadOnly` with `ErrChatReadOnly` (SEC-01, T-154-03)
2. Sanitizes via `SanitizeChatContent` (not `SanitizePTYText`) and returns nil on empty result (silent no-op)
3. No `WriteInput` call — chat send must never touch PTY stdin (T-154-02, D-02)

Follows unlock-before-IO discipline: reads `chatAppendFn` under `hub.mu` then releases before calling, mirroring `HandleInject` line 520-522.

### case MsgChatSend in relay/server.go read pump
Placed immediately before `case MsgSessionInject`. Unmarshals `ChatSendPayload`, continues on unmarshal error or empty `Content`, calls `hub.HandleChatSend`, logs+drops on error (no NAK).

### case MsgChatSend in webserver/server.go read pump
Structurally identical. SEC-01 gate lives inside `HandleChatSend`; `sub.ReadOnly` is derived from the signed JWT `HasPerm("write")` check (D-24/SEC-04 safe).

## Test Coverage

All tests run via `go test ./internal/relay/... ./internal/webserver/... -run ChatSend -count=1`.

| Test | Package | What It Proves |
|------|---------|----------------|
| TestHandleChatSend_ROReturnsError | relay | ErrChatReadOnly returned, zero persist, zero broadcast (SEC-01/T-154-03) |
| TestHandleChatSend_EmptyAfterSanitize | relay | nil return, zero persist, zero broadcast for control-char-only input |
| TestHandleChatSend_RWPersistsAndBroadcasts | relay | chatAppendFn called once, MsgChat broadcast received, SessionInject=false |
| TestHandleChatSend_NoPTYWrite | relay | PTY write count = 0 after HandleChatSend (T-154-02) |
| TestChatSend_RWBroadcasts_RelayPath | relay | End-to-end relay read pump: RW client gets MsgChat broadcast |
| TestChatSend_RODropped_RelayPath | relay | RO client: no broadcast, no NAK (silent drop) |
| TestChatSend_MalformedIgnored_RelayPath | relay | Malformed/empty JSON: no broadcast, no NAK |
| TestChatSend_RWBroadcasts_WebPath | webserver | End-to-end webserver read pump: RW JWT client gets MsgChat broadcast |
| TestChatSend_RODropped_WebPath/browse_off | webserver | RO JWT (perms="read"): no broadcast (SEC-01 web path) |
| TestChatSend_RODropped_WebPath/browse_on | webserver | RO JWT (perms="read,files.read"): no broadcast (SEC-01 web path) |
| TestChatSend_MalformedIgnored_WebPath | webserver | Malformed/empty JSON: no broadcast, no NAK |

## Verification Results

- `go test ./internal/relay/... -run ChatSend -count=1` — PASS (7 tests)
- `go test ./internal/webserver/... -run ChatSend -count=1` — PASS (4 tests, including 2 sub-tests)
- `go test ./internal/relay/... ./internal/webserver/... -count=1` — fully green (all packages)
- `go build ./...` — clean
- `go vet ./internal/relay/... ./internal/webserver/...` — clean
- `grep -c 'case MsgChatSend' internal/relay/server.go` — 1
- `grep -c 'case relay.MsgChatSend' internal/webserver/server.go` — 1
- `grep 'WriteInput' internal/relay/hub.go` — HandleChatSend mentions WriteInput in comment only (no call)

## Deviations from Plan

None — plan executed exactly as written.

## TDD Gate Compliance

Both tasks followed the TDD RED→GREEN pattern:
1. `test(154-01): add failing tests for Hub.HandleChatSend` (5fb8b3ca) — RED gate
2. `feat(154-01): add ChatSendPayload + Hub.HandleChatSend with SEC-01 gate` (7a8074e0) — GREEN gate
3. `test(154-01): add failing read-pump ChatSend dispatch tests` (d71c76ae) — RED gate
4. `feat(154-01): add case MsgChatSend to relay + webserver read pumps` (91203a34) — GREEN gate

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced beyond what the plan's threat model already accounts for (T-154-01 through T-154-04). All mitigations present:
- T-154-03 (RO elevation): `ErrChatReadOnly` gate in `HandleChatSend` ✓
- T-154-02 (PTY write): no `WriteInput` call in `HandleChatSend` ✓
- T-154-01 (content tampering): `SanitizeChatContent` applied before persist ✓
- T-154-04 (DoS via error storm): error logged and dropped, no per-message NAK ✓

## Self-Check: PASSED

All created files confirmed on disk. All 4 task commits found in git log.
