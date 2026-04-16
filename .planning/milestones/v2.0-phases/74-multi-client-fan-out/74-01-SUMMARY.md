---
phase: 74-multi-client-fan-out
plan: 01
subsystem: relay/hub
tags: [multi-client, fan-out, resize, subscriber-metadata, tdd]
dependency_graph:
  requires: []
  provides: [Subscriber.ReadOnly, Subscriber.Name, Subscriber.Cols, Subscriber.Rows, Hub.ptyCols, Hub.ptyRows, Hub.SubscriberCount, Hub.ResizeClient]
  affects: [internal/relay/hub.go, internal/relay/hub_test.go]
tech_stack:
  added: []
  patterns: [max-wins-resize-arbitration, unlock-before-callback]
key_files:
  created: []
  modified: [internal/relay/hub.go, internal/relay/hub_test.go]
decisions:
  - "ResizeClient releases hub.mu before calling resizeFn to avoid blocking broadcast drain loop"
  - "Max-wins policy computes max(Cols) and max(Rows) independently across all subscribers"
  - "ptyCols/ptyRows track last-applied PTY size to avoid redundant resize syscalls"
metrics:
  duration: 2m
  completed: "2026-04-15T02:32:10Z"
  tasks: 2
  files: 2
---

# Phase 74 Plan 01: Hub Subscriber Metadata & Resize Arbitration Summary

Extend relay Hub Subscriber with per-client metadata (ReadOnly, Name, Cols, Rows) and add SubscriberCount/ResizeClient methods with max-wins PTY resize arbitration -- foundation layer for multi-client fan-out.

## What Was Done

### Task 1: Write tests for Hub extensions (RED phase)
**Commit:** 72a91c7

Added six new test functions to `internal/relay/hub_test.go`:

| Test | Behavior Verified |
|------|------------------|
| TestHub_SubscriberCountTracksConcurrentSubscribers | Count tracks 0 -> 1 -> 2 -> 1 -> 0 through subscribe/unsubscribe lifecycle |
| TestHub_ReadOnlyFlagStored | ReadOnly bool field persists on Subscriber struct |
| TestHub_ClientNameStored | Name string field persists on Subscriber struct |
| TestHub_ResizeMaxWinsPolicy | PTY only resizes when max dimensions change (2 of 3 calls trigger resize) |
| TestHub_ResizeClientNoOpWhenDimensionsUnchanged | Duplicate dimensions produce exactly 1 resize call |
| TestHub_ResizeClientUnsubscribeDoesNotShrink | After unsubscribe, recomputed max triggers resize when it differs from current PTY |

All tests failed as expected (RED) because Subscriber lacked fields and Hub lacked methods.

### Task 2: Implement Hub extensions (GREEN phase)
**Commit:** beb8e3a

Modified `internal/relay/hub.go` with:

1. **Subscriber struct extensions:** Added `ReadOnly bool`, `Name string`, `Cols int`, `Rows int` fields
2. **Hub struct extensions:** Added `ptyCols int`, `ptyRows int` for tracking current PTY dimensions
3. **Hub.SubscriberCount():** Returns `len(h.subscribers)` under mutex lock
4. **Hub.ResizeClient():** Stores subscriber dimensions, computes max across all subscribers, calls resizeFn only when max differs from current PTY size. Critically, releases `hub.mu` before calling `resizeFn` to avoid blocking the broadcast drain loop.

All 6 new tests pass. All 8 existing hub_test.go tests pass.

## Decisions Made

1. **Unlock before callback pattern:** `ResizeClient` releases `hub.mu` before invoking `resizeFn` because PTY resize is a potentially blocking syscall that would contend with the broadcast loop holding the same lock.
2. **Independent max computation:** `maxCols` and `maxRows` are computed independently (not as a pair from a single subscriber), matching the plan's max-wins specification.
3. **Zero-value guard:** Resize is skipped when both maxCols and maxRows are 0, preventing spurious resize calls before any client reports dimensions.

## Deviations from Plan

None -- plan executed exactly as written.

## TDD Gate Compliance

1. RED gate: `test(74-01)` commit 72a91c7 -- 6 failing tests
2. GREEN gate: `feat(74-01)` commit beb8e3a -- all tests pass
3. REFACTOR gate: not needed -- implementation is minimal and clean

## Out-of-Scope Observations

- `TestHub_SlowClientDisconnected` in `server_test.go` is flaky (pre-existing, unrelated to hub.go changes). Logged but not fixed per deviation scope rules.

## Self-Check: PASSED

- [x] internal/relay/hub.go exists
- [x] internal/relay/hub_test.go exists
- [x] .planning/phases/74-multi-client-fan-out/74-01-SUMMARY.md exists
- [x] Commit 72a91c7 (RED) found in git log
- [x] Commit beb8e3a (GREEN) found in git log
