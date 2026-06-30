---
phase: 152-relay-protocol-identity-presence
plan: "03"
subsystem: relay
tags: [presence, typing, hub, websocket, identity, relay]
dependency_graph:
  requires: ["152-01"]
  provides: [hub-presence-roster, hub-typing-ttl, subscriber-identity-fields]
  affects: ["152-05", "152-06"]
tech_stack:
  added: []
  patterns:
    - Reference-counted presence roster keyed by PersonKey (D-03/D-04)
    - time.AfterFunc typing TTL with h.closed shutdown guard (Pitfall 2/T-152-07)
    - Release-mu-before-broadcast discipline (mirrors ResizeClient pattern)
    - BroadcastExcept for sender exclusion (Pitfall 5/T-152-03)
    - Per-person 500ms rate limit for typing-start broadcasts (T-152-03)
key_files:
  created:
    - internal/relay/hub_presence_test.go
  modified:
    - internal/relay/hub.go
decisions:
  - "Unsubscribe returns (presenceChanged bool): Plans 05/06 update the two call sites; existing call sites that discard the return value continue to compile"
  - "typingTTL is a Hub field (default 5s) injectable for deterministic TTL tests — no real 5s sleeps in tests"
  - "lastTypingBcast map[string]time.Time: rate-limit typing-start broadcasts at 500ms per personKey to coalesce keystrokes (T-152-03)"
  - "NotifyTyping reuses BroadcastPresence for typing:false fan-out (send to ALL subscribers including sender — they should see their own typing cleared)"
  - "UpdateTyping takes *Subscriber not (personKey,alias string) so BroadcastExcept can exclude the exact pointer"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-25"
  tasks_completed: 2
  files_modified: 2
status: complete
---

# Phase 152 Plan 03: Hub Presence + Typing Layer Summary

Extends `internal/relay/hub.go` with a reference-counted presence roster and a
server-managed typing TTL layer — the engine for D-03 (per-person collapsed
presence), D-04 (composite PersonKey disambiguation), and PRESENCE-02
(typing clears on abrupt disconnect via time.AfterFunc).

## What Was Built

**Subscriber identity fields** (`hub.go`):
- `TailnetID`, `Origin`, `PersonKey`, `Alias`, `AliasSetFn` added after `Rows`
- Set once at subscribe time by the read pump goroutines (Plans 05/06)

**Hub presence/typing state** (`hub.go`):
- `presenceRoster map[string]*presenceState` — personKey → collapsed entry with ConnCount
- `typingRoster map[string]*time.Timer` — personKey → 5s TTL timer
- `lastTypingBcast map[string]time.Time` — rate-limit tracking (T-152-03)
- `typingTTL time.Duration` — injectable; default 5s; tests use 5ms/30ms
- `presenceState` struct: TailnetID, Origin, Alias, ConnCount
- All three maps initialized in `NewHub` (Pitfall 4 nil-map-panic prevention)

**New Hub methods** (`hub.go`):
- `Subscribe` extended: increments ConnCount or creates presenceState for non-empty PersonKey
- `Unsubscribe` signature changed to `(presenceChanged bool)`: decrements ConnCount, removes entry + typing timer on last connection, returns true when broadcast warranted
- `BroadcastPresence(frame []byte)` — verbatim copy of BroadcastMeta
- `BroadcastExcept(frame []byte, exclude *Subscriber)` — BroadcastMeta with sender skip
- `CurrentPresence() []PresenceEntry` — snapshot of presenceRoster under mu
- `UpdateAlias(personKey, alias string)` — updates roster entry Alias under mu
- `UpdateTyping(sub *Subscriber, typing bool)` — rate-limited, TTL-managed, h.closed-guarded
- Package-level `NotifyPresence(hub *Hub)` — mirrors NotifyViewerCount
- Package-level `NotifyTyping(hub *Hub, personKey, alias string, typing bool)` — typing:false fan-out

**Tests** (`hub_presence_test.go` — 14 test functions):
- TestPresenceCollapse, TestPresenceRefCount, TestCompositePersonKey
- TestUnsubscribePresenceChanged, TestBroadcastPresence, TestUpdateAlias
- TestEmptyPersonKeyNoPresenceEntry, TestNotifyPresence
- TestTypingSenderExclusion, TestTypingTTL, TestTypingTimerReset
- TestTypingExplicitStop, TestUnsubscribeCancelsTypingTimer
- TestHubShutdownWithActiveTypingTimer (race-tested, no panic under h.closed guard)

## Commits

| Hash | Type | Description |
|------|------|-------------|
| `64fc2a1f` | test | RED: add failing presence/typing tests in hub_presence_test.go |
| `6df05c38` | feat | GREEN: extend hub.go with presence/typing layer |

## Deviations from Plan

**None** — plan executed exactly as written.

The only minor implementation detail resolved at coding time: `UpdateTyping` takes
`*Subscriber` (not `personKey, alias string`) so `BroadcastExcept` can receive the
exact pointer for sender exclusion. This matches the plan's behavior spec and the
PATTERNS.md guidance on Pitfall 5 (BroadcastExcept for sender exclusion).

## Threat Mitigations Applied

| Threat ID | Mitigation Applied |
|-----------|-------------------|
| T-152-03 | `lastTypingBcast` rate-limit map (500ms per personKey); `BroadcastExcept` sender exclusion; non-blocking fan-out with CloseSlow drop-on-slow |
| T-152-07 | `h.closed` guard in `UpdateTyping` and in `time.AfterFunc` callback; `Unsubscribe` Stops+deletes timer on last connection |
| T-152-08 | Zero ChatStore/AppendMessage references in typing code path (verified by grep) |
| T-152-SC | No new packages — stdlib only (time added to hub.go imports) |

## Known Stubs

None. The Hub methods are fully functional. Plans 05/06 will wire the two existing
`Unsubscribe` call sites (relay/server.go:232, webserver/server.go:1028) to consume
the new bool return value and call NotifyPresence accordingly.

## Self-Check

### Created files exist
- /Users/ken/dev/agenthub/internal/relay/hub_presence_test.go: FOUND
- /Users/ken/dev/agenthub/internal/relay/hub.go: FOUND (modified)

### Commits exist
- 64fc2a1f: FOUND
- 6df05c38: FOUND

### Test results
- go test -race ./internal/relay/...: PASS (all 14 new tests + full existing suite)

## Self-Check: PASSED
