---
phase: 153-session-pty-bridge
plan: "02"
subsystem: relay-inject-machinery
tags: [relay, security, inject, hub, tdd, protocol]
dependency_graph:
  requires: [153-01]
  provides: [Hub.HandleInject, Hub.BroadcastChat, Hub.SetChatAppendFn, ErrReadOnly, relay-inject-case, engine-chatAppendFn-wiring]
  affects: [internal/relay/hub.go, internal/relay/server.go, internal/daemon/engine.go, internal/relay/server_inject_test.go]
tech_stack:
  added: []
  patterns: [unlock-before-io, callback-import-cycle-break, non-blocking-subscriber-send, tdd-test-after]
key_files:
  created:
    - internal/relay/server_inject_test.go
  modified:
    - internal/relay/hub.go
    - internal/relay/server.go
    - internal/daemon/engine.go
decisions:
  - "HandleInject reads sub.ReadOnly without a lock (set once at subscribe time, never mutated) — no mutex held before any I/O (ResizeClient unlock-before-IO discipline)"
  - "chatAppendFn read under hub.mu then released before calling — lock never held across disk I/O or BroadcastChat"
  - "Test setup wires in-memory chatAppendFn so BroadcastChat fires in tests; mirrors what engine.go does in production"
  - "malformed/empty InjectPayload frames silently dropped (same as MsgTyping/MsgAliasSet); only ErrReadOnly produces an explicit NAK"
metrics:
  duration: "9 minutes"
  completed: "2026-06-26"
  tasks_completed: 3
  files_changed: 4
status: complete
---

# Phase 153 Plan 02: Core Inject Machinery Summary

Hub-level HandleInject (RW-gate → sanitize → WriteInput → persist-original → broadcast), relay read-pump MsgSessionInject case, engine.go chatStore callback wiring (import-cycle break), and relay-path integration tests proving MENTION-02, MENTION-03, and SEC-01.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add ErrReadOnly, chatAppendFn, SetChatAppendFn, BroadcastChat, HandleInject to Hub | 73575767 | internal/relay/hub.go |
| 2 | Add relay read-pump inject case and wire engine callback | d595854a | internal/relay/server.go, internal/daemon/engine.go |
| 3 (tests) | Relay-path inject tests — RW write, dedicated-frame-only, RO rejection | 108b5f09 | internal/relay/server_inject_test.go, internal/relay/hub.go (gofmt) |

## What Was Built

**Task 1 — Hub inject machinery** (`internal/relay/hub.go`):

- `ErrReadOnly` — package-level sentinel returned by HandleInject for RO clients (SEC-01, D-04)
- `chatAppendFn func(ChatMessage) (ChatMessage, error)` — nil-safe Hub field; wired by engine.go after Hub+ChatStore are both constructed; guarded by hub.mu
- `SetChatAppendFn(fn)` — acquires hub.mu, assigns fn; mirrors AliasSetFn assignment style
- `BroadcastChat(frame []byte)` — fan-out identical to BroadcastMeta; separated for clarity (MsgChat frame type)
- `HandleInject(sub *Subscriber, text string) error`:
  1. Returns ErrReadOnly immediately if sub.ReadOnly (gate, no lock needed — field set once at subscribe time)
  2. Calls SanitizePTYText(text) → sanitized
  3. Calls WriteInput([]byte(sanitized)) — returns error if write fails
  4. Reads chatAppendFn under hub.mu, then releases lock before calling
  5. If chatAppendFn is non-nil: persists ChatMessage{Content: **original text** (Pitfall 6), SessionInject:true, AuthorID: sub.TailnetID, AuthorAlias: sub.Alias}
  6. On success: broadcasts MakeChatFrame(returnedMsg) via BroadcastChat (which acquires its own lock)
  7. No hub.mu held across any I/O (ResizeClient unlock-before-IO discipline)

**Task 2 — Read-pump case + engine wiring** (`internal/relay/server.go`, `internal/daemon/engine.go`):

- `case MsgSessionInject:` in relay read-pump switch (after MsgAliasSet, before switch close):
  - JSON-unmarshals InjectPayload; silently drops malformed/empty frames
  - Calls `hub.HandleInject(sub, ip.Text)` — gate + sanitize run inside hub
  - On error: sends MakeInjectErrorFrame(err.Error()) to sub.Msgs via non-blocking select/CloseSlow
  - Code comment explicitly notes MsgChatSend (0x31) is chat-only and NEVER writes to PTY (D-02)
- Engine wiring in CreateSession (after `e.mu.Unlock()` that follows chatStore registration):
  - `if chatStore != nil { hub.SetChatAppendFn(func(msg relay.ChatMessage) (relay.ChatMessage, error) { return chatStore.AppendMessage(msg) }) }`
  - Closure captures `*ChatStore` without relay importing daemon (import-cycle break, Research Pattern 5)
  - Nil-guarded: non-fatal chatStore failures leave chatAppendFn nil; PTY write still occurs

**Task 3 — Relay-path inject tests** (`internal/relay/server_inject_test.go`):

- `writerFunc` adapter type — `type writerFunc func([]byte)(int, error)` with Write method
- `setupInjectTestServer` — creates HubManager with counting PTY writer + in-memory chatAppendFn
- `dialInjectWS` — dials WebSocket with optional query string (mirrors dialIdentityWS)
- `waitForFrameType` — reads frames, skipping non-target types, returns first match within 3s timeout
- `assertNoFrameType` — verifies no frame of given type received within deadline
- `TestInject_RWCap_WritesToPTY` — RW client sends MsgSessionInject, PTY write count > 0, MsgChat broadcast with SessionInject:true and original Content received (MENTION-02)
- `TestInject_OnlyDedicatedFrame` — MsgChatSend (0x31) + stray frame never write to PTY; count stays 0 (MENTION-03/D-02)
- `TestInject_ROCap_RelayPath` — RO client hand-crafted MsgSessionInject frame receives MsgInjectError NAK; PTY write count stays 0 (SEC-01 relay path adversarial proof)

## Verification

```
go build ./...                                                          PASS
go test -race -short -run TestInject ./internal/relay/...              PASS (3 tests)
go vet ./internal/relay/... ./internal/daemon/...                      PASS
gofmt -l (all modified files)                                          PASS (no output)
Manual read: HandleInject holds no hub.mu across WriteInput/chatAppendFn/BroadcastChat PASS
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] gofmt alignment drift in hub.go**
- **Found during:** Task 3 commit preparation
- **Issue:** `gofmt -l` reported `hub.go` and `server.go` needed formatting (struct field alignment, whitespace)
- **Fix:** Ran `gofmt -w` on both files; re-verified build and tests pass
- **Files modified:** `internal/relay/hub.go`, `internal/relay/server.go`
- **Commit:** 108b5f09 (included in Task 3 commit)

**2. [Rule 2 - Missing functionality] Test setup missing chatAppendFn wiring**
- **Found during:** Task 3 RED run
- **Issue:** Initial test server setup used `io.Discard` for the hub writer and did not call `SetChatAppendFn`. Without the callback, `HandleInject` skips the broadcast step and `TestInject_RWCap_WritesToPTY` timed out waiting for a `MsgChat` frame
- **Fix:** Added in-memory `chatAppendFn` to `setupInjectTestServer` that fills in missing ChatMessage fields (ID, TimestampMs, SchemaVersion, SessionID) — mirrors what engine.go's ChatStore.AppendMessage does in production
- **Files modified:** `internal/relay/server_inject_test.go`
- **Commit:** 108b5f09

### TDD Gate Compliance

Task 3 is `tdd="true"`. The cycle:
- **RED gate:** Initial `server_inject_test.go` failed at runtime — `TestInject_RWCap_WritesToPTY` timed out (3.08s) waiting for `MsgChat` because `chatAppendFn` was nil in test setup; `TestInject_ROCap_RelayPath` and `TestInject_OnlyDedicatedFrame` passed as expected
- **GREEN gate:** Wired in-memory `chatAppendFn` in `setupInjectTestServer`; all 3 tests pass
- **REFACTOR:** Not required — implementation clean; only gofmt alignment applied

Note: This plan follows an atypical TDD sequence (implementation in Tasks 1-2, tests in Task 3). The RED→GREEN gap was in test infrastructure (chatAppendFn wiring), not in production code. This is correct behavior: the production code (`HandleInject`) is nil-safe and correctly skips BroadcastChat when `chatAppendFn` is unset; the test needs to wire it (just as engine.go does) to observe the complete behavior.

## Known Stubs

None. All paths are fully implemented.

## Threat Flags

No new threat surface beyond what the plan's threat model documents:
- No new network endpoints
- No new auth paths
- The MsgSessionInject read-pump case uses the existing `sub.ReadOnly` gate (from signed JWT or query param — set at subscribe time, trust boundary is at the relay/webserver entry point)
- T-153-06 mitigated: HandleInject returns ErrReadOnly for RO subscribers; proven by TestInject_ROCap_RelayPath
- T-153-07 mitigated: only MsgSessionInject routes to WriteInput; proven by TestInject_OnlyDedicatedFrame
- T-153-08 mitigated: no fast path — relay loopback owner runs through same SanitizePTYText as web participant

## Self-Check: PASSED
