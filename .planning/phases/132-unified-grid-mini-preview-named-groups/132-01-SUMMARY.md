---
phase: 132-unified-grid-mini-preview-named-groups
plan: "01"
subsystem: daemon-rpc
tags: [go, daemon, wails, rpc, card-07, scrollback, ansi]
dependency_graph:
  requires: []
  provides: [GetSessionTailLines-rpc]
  affects: [HubPanel-mini-preview, usePreviewPoller]
tech_stack:
  added: []
  patterns:
    - TDD RED/GREEN/REFACTOR for Go engine method
    - 4-layer RPC chain mirroring GetSessionStatus pattern
    - regexp for ANSI/OSC stripping close to source
key_files:
  created: []
  modified:
    - internal/daemon/types.go
    - internal/daemon/engine.go
    - internal/daemon/engine_test.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - app.go
    - frontend/src/wailsjs/go/main/App.d.ts
decisions:
  - "ansiEscape package-level var covers both CSI and OSC sequences (extended beyond frontend stripAnsi.ts)"
  - "n clamped to [1..20] in app.go layer to bound response size (T-132-02 DoS mitigation)"
  - "nil engine result promoted to empty []string{} at both api.go and app.go layers"
  - "Tests use io.Pipe + hub.Done() for deterministic scrollback population without real PTY"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-16"
  tasks_completed: 2
  files_modified: 7
---

# Phase 132 Plan 01: GetSessionTailLines RPC — Wave-0 Backend Gap Summary

**One-liner:** Complete 4-layer `GetSessionTailLines` RPC (daemon route → engine → daemon client → Wails binding) sourced from `relay.Hub` scrollback, with 0x01 framing bytes and ANSI/OSC sequences stripped server-side.

## What Was Built

The Wave-0 Go backend gap for CARD-07 is now closed. Before this plan, no tail-line RPC existed anywhere in the stack. This plan adds:

1. **`TailLinesResponse`** struct in `types.go` — wire type for the new route
2. **`SessionEngine.GetSessionTailLines`** in `engine.go` — strips relay framing bytes (0x01), strips ANSI/OSC sequences via `ansiEscape` regexp, trims trailing empty lines, returns last N lines; nil for unknown session
3. **`GET /sessions/{id}/tail`** route + `handleGetSessionTailLines` handler in `api.go` — parses optional `?n=` query param (default 4), promotes nil result to `[]string{}`
4. **`DaemonClient.GetSessionTailLines`** in `client.go` — typed Go client method via `doJSON`
5. **`App.GetSessionTailLines`** in `app.go` — Wails binding with n clamped to `[1..20]`, returns `[]string{}` on nil client/error/nil lines
6. **`GetSessionTailLines` declaration** in `App.d.ts` — TypeScript stub for frontend type-checking

## Task Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 RED | `77ab541` | Failing tests for GetSessionTailLines (5 test cases) |
| Task 1 GREEN | `3f656cb` | Engine method + TailLinesResponse implementation |
| Task 2 | `d89271a` | api.go route, client.go method, app.go binding, App.d.ts stub |

## Verification Results

```
go test ./internal/daemon/... -run TestGetSessionTailLines -count=1 -v
=== RUN   TestGetSessionTailLines_StripsFramingBytes   PASS
=== RUN   TestGetSessionTailLines_StripsANSI           PASS
=== RUN   TestGetSessionTailLines_ReturnsLastN         PASS
=== RUN   TestGetSessionTailLines_TrimsTrailingEmptyLines  PASS
=== RUN   TestGetSessionTailLines_UnknownSession       PASS

go build ./...  ← clean, no errors
go test ./internal/daemon/... -count=1 -short  ← all pass, no regressions
```

## Deviations from Plan

None — plan executed exactly as written. The 4-layer chain mirrors the GetSessionStatus analog precisely.

## Threat Mitigations Applied

| Threat | Mitigation Applied |
|--------|--------------------|
| T-132-01: ANSI injection | `ansiEscape` regexp strips CSI + OSC sequences server-side; downstream renders as text |
| T-132-02: DoS via large n | app.go clamps n to [1..20]; scrollback bounded at 256 KiB |

## TDD Gate Compliance

- RED gate commit: `77ab541` (`test(132-01): add failing tests...`)
- GREEN gate commit: `3f656cb` (`feat(132-01): add GetSessionTailLines engine method...`)
- REFACTOR: not needed (implementation clean on first pass)

## Known Stubs

None — this plan only adds Go backend and TypeScript type declaration. No frontend rendering stubs.

## Threat Flags

None — no new network endpoints, auth paths, or schema changes beyond what the threat model covers.

## Self-Check: PASSED

| Item | Status |
|------|--------|
| internal/daemon/types.go | FOUND |
| internal/daemon/engine.go | FOUND |
| internal/daemon/api.go | FOUND |
| internal/daemon/client.go | FOUND |
| app.go | FOUND |
| frontend/src/wailsjs/go/main/App.d.ts | FOUND |
| Commit 77ab541 (RED tests) | FOUND |
| Commit 3f656cb (GREEN impl) | FOUND |
| Commit d89271a (wiring) | FOUND |
