---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: "03"
subsystem: relay/inject
tags: [tdd, test-only, in-02, regression]
dependency_graph:
  requires: []
  provides: [IN-02-regression-coverage]
  affects: [internal/relay]
tech_stack:
  added: []
  patterns: [table-driven Go test, atomic counter assertion, WebSocket test server]
key_files:
  created: []
  modified:
    - internal/relay/server_inject_test.go
decisions:
  - "Reused setupInjectTestServer + dialInjectWS without adding helpers, per plan prohibition"
  - "Used \\x1b[2J (CSI clear-screen) as the control-only text — collapses via SanitizePTYText to \\n, then TrimSpace to empty"
metrics:
  duration: "~3 minutes"
  completed: "2026-06-28T03:14:57Z"
  tasks_completed: 1
  tasks_total: 1
status: complete
requirements: [IN-02]
---

# Phase 160 Plan 03: IN-02 Control-Inject Regression Test Summary

Locked the IN-02 guard with an automated regression test proving zero PTY writes for control-only inject text.

## One-liner

Go regression test `TestInject_ControlOnlyInput` exercises hub.go:608 TrimSpace guard end-to-end via a live WebSocket -> HandleInject path with a CSI clear-screen escape.

## Tasks

| # | Name | Status | Commit |
|---|------|--------|--------|
| 1 | Add TestInject_ControlOnlyInput (IN-02 regression) | Done | a51f406e |

## What Was Built

Added `TestInject_ControlOnlyInput` to `internal/relay/server_inject_test.go`. The test:

1. Stands up a test relay.Server with counting PTY writer via `setupInjectTestServer(t)`
2. Dials an RW inject WebSocket via `dialInjectWS`
3. Sends a `MsgSessionInject` frame with `InjectPayload{Text: "\x1b[2J"}` (CSI clear-screen escape)
4. Asserts `ptyWriteCount.Load() == 0` after 100ms settle

The frame passes the read-pump `ip.Text != ""` guard (non-empty raw) but `SanitizePTYText` collapses it to `"\n"`, then `strings.TrimSpace` yields `""`, triggering the IN-02 early-return at hub.go:608 — no PTY write, no spurious Enter.

## Test Results

```
ok  	github.com/scottkw/agenthub/internal/relay	1.203s
```

`go test -race -run TestInject_ControlOnly ./internal/relay/...` PASSED.

## Verification

- `git diff --stat` confirms only `server_inject_test.go` changed (1 file, 26 insertions, 0 production files touched)
- No new helpers added — `setupInjectTestServer` and `dialInjectWS` reused as required

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None — test-only change; no new network surface introduced.

## Self-Check: PASSED

- File exists: `internal/relay/server_inject_test.go` — FOUND
- Commit a51f406e exists — FOUND
- No production files changed — CONFIRMED
