---
phase: 152
plan: "05"
subsystem: relay-identity-presence
tags: [relay, identity, presence, alias, typing, daemon, go]
dependency_graph:
  requires: ["152-01", "152-02", "152-03"]
  provides: ["IDENT-01", "IDENT-02", "PRESENCE-01", "PRESENCE-02"]
  affects: ["internal/daemon/engine.go", "internal/relay/server.go", "internal/daemon/api.go"]
tech_stack:
  added: []
  patterns:
    - "callback closure pattern (relay→daemon import cycle prevention)"
    - "nil-safe identity provider wiring in daemon.RelayHandler"
    - "readDataFrame skip-list extended for new server-push frame types"
key_files:
  created:
    - internal/relay/server_identity_test.go
  modified:
    - internal/daemon/engine.go
    - internal/relay/server.go
    - internal/daemon/api.go
    - internal/relay/server_test.go
decisions:
  - "Identity provider set via SetIdentityProviders setter rather than constructor param — matches NewServer's existing nil-safe filesHandler pattern and avoids import cycle"
  - "ConfigDirForTest also reconstructs AliasStore — mirrors ChatsBaseDirForTest pattern for isolated test persistence"
  - "AliasStore.Set error discarded in api.go closure wrapper — alias persistence failure must not break relay connection"
metrics:
  duration: "~20 minutes"
  completed: "2026-06-26"
  tasks_completed: 3
  files_changed: 5
status: complete
---

# Phase 152 Plan 05: Relay Identity Wiring Summary

AliasStore wired into SessionEngine and relay.Server; handleSession stamps owner identity, broadcasts presence on join/leave, and dispatches MsgAliasSet and MsgTyping in the read pump — delivering IDENT-01/02 and PRESENCE-01/02 for the loopback relay path.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add AliasStore to SessionEngine | 799d383a | internal/daemon/engine.go |
| 2 | Identity stamping + presence/alias/typing dispatch | 8cbb6599 | internal/relay/server.go, internal/daemon/api.go |
| 3 | Relay integration test + readDataFrame fix | 6fd9c760 | internal/relay/server_identity_test.go, internal/relay/server_test.go |

## What Was Built

**engine.go:** Added `aliasStore *AliasStore` field to `SessionEngine`. Constructed in `NewSessionEngine` over `daemonConfigDir` (tolerate error — daemon startup must not crash on a missing or corrupt aliases.json). Added `Aliases() *AliasStore` accessor. `ConfigDirForTest` also reconstructs the AliasStore over the test dir for isolated persistence (mirrors `ChatsBaseDirForTest`).

**server.go:** `Server` gains three identity provider fields (`ownerDefaultAlias`, `getAlias`, `setAlias`) and a `SetIdentityProviders` setter. In `handleSession`, before `hub.Subscribe`:
- Stamps `TailnetID="local"`, `Origin="local"`, `PersonKey="local:local"` (T-152-11: loopback-only path — spoofing accept)
- Resolves `Alias` via `getAlias("local:local", ownerDefault)` (persisted alias or engine hostname fallback)
- Wires `AliasSetFn` to call `setAlias` on alias change

After `Subscribe`: calls `NotifyPresence(hub)` (PRESENCE-01 join). The defer now captures `presenceChanged := hub.Unsubscribe(sub)` and calls `NotifyPresence(hub)` only when the last connection for `local:local` drops (PRESENCE-01 leave).

Read pump switch gained two new cases outside the `ReadOnly` guard (D-06 — RO clients are full chat participants; only `MsgInput` remains gated):
- `MsgTyping`: `hub.UpdateTyping(sub, tp.Typing)` (PRESENCE-02 with sender-exclusion)
- `MsgAliasSet`: `ValidateAlias` → update sub.Alias + AliasSetFn + `hub.UpdateAlias` + `NotifyPresence` (IDENT-02, T-152-01 control-char rejection)

**api.go:** `RelayHandler` wires `engine.hostname` + `AliasStore.GetOrDefault` + `AliasStore.Set` into `server.SetIdentityProviders` via closures (nil-guard for failed AliasStore construction).

**server_identity_test.go:** Three integration tests over a real WebSocket with an in-memory alias map (no daemon import):
- `TestRelayIdentity_AliasPropagation` — alias-set propagates to both clients in one round-trip (IDENT-02)
- `TestRelayIdentity_TypingExcludesSender` — typing-true delivered to B, not echoed to sender A (PRESENCE-02 + T-152-03)
- `TestRelayIdentity_ReadOnlyCanChat` — RO client sends MsgAliasSet and receives confirming MsgPresence (D-06)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-152 relay tests failed on new MsgPresence frames**
- **Found during:** Task 3 full-suite test run (`go test ./...`)
- **Issue:** `readDataFrame` in `server_test.go` skipped `MsgMeta` but not `MsgPresence`. After Task 2 added `NotifyPresence` on every `Subscribe`, pre-152 tests that expected `MsgOutput` as the first frame received `MsgPresence` instead — 5 tests failed.
- **Fix:** Extended `readDataFrame` skip condition: `if msgType == MsgMeta || msgType == MsgPresence { continue }` — server-push housekeeping frames (viewer count + presence roster) are transparent to pre-152 PTY-output tests.
- **Files modified:** `internal/relay/server_test.go`
- **Commit:** 6fd9c760

## Verification

- `go build ./...` — clean
- `go test -race ./internal/relay/ ./internal/daemon/ ./...` — all green (full module)
- `grep -n "MsgAliasSet\|MsgTyping" internal/relay/server.go` confirms both new cases are top-level switch branches, not inside the `if !sub.ReadOnly` block (D-06 compliance)

## Self-Check: PASSED

Files exist:
- FOUND: internal/relay/server_identity_test.go
- FOUND: internal/relay/server.go (SetIdentityProviders + handleSession changes)
- FOUND: internal/daemon/engine.go (aliasStore field + Aliases() accessor)
- FOUND: internal/daemon/api.go (RelayHandler identity provider wiring)

Commits exist:
- FOUND: 799d383a (engine AliasStore)
- FOUND: 8cbb6599 (server.go + api.go identity wiring)
- FOUND: 6fd9c760 (integration test + readDataFrame fix)

Build: go build ./... clean
Tests: go test -race ./... all green (14 packages)
