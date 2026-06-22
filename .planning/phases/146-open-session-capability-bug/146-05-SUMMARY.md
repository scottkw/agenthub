---
phase: 146-open-session-capability-bug
plan: "05"
subsystem: remote-session-open
tags: [fix, remote, capability, browser-open, held-cap-reuse, gap-closure]
dependency_graph:
  requires: [146-01, 146-02, 146-03, 146-04]
  provides: [GAP-146-A-closed, FIX-03-complete]
  affects: [app.go, internal/daemon/remote_files.go, frontend/src/App.tsx, frontend/src/components/RemoteJoinCodeModal.tsx]
tech_stack:
  added: []
  patterns:
    - RemoteCapStore.Get used for cap-bearing open URL composition (no new minting)
    - Held-cap reuse gate mirrors handleBrowseFilesRemote remoteCapsCached.has pattern
    - Wails binding returns composed URL string; cap never returns to React state
key_files:
  created:
    - internal/daemon/open_remote_session_url_test.go
  modified:
    - internal/daemon/remote_files.go
    - internal/daemon/api.go
    - internal/daemon/client_remote_files.go
    - app.go
    - frontend/src/App.tsx
    - frontend/src/components/RemoteJoinCodeModal.tsx
    - frontend/src/components/__tests__/App.open-remote.test.tsx
    - frontend/src/components/__tests__/RemoteJoinCodeModal.test.tsx
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts
    - TESTING.md
decisions:
  - "WR-01: fallback open-session branch now deposits cap then calls OpenRemoteSessionURL for daemon-composed URL, eliminating /sessions/A?cap=capB mismatch"
  - "WR-03: 'not-found' error (single-use code consumed) split from 'invalid' (typo) into 'already used or expired' copy per D-11"
  - "WR-02: behavior tests cross held-cap + no-cap paths via source inspection + RemoteJoinCodeModal component render"
  - "Wails bindings App.js + App.d.ts hand-extended for OpenRemoteSessionURL (wails build regenerates these in production)"
metrics:
  duration: "~45 minutes"
  completed: "2026-06-22T18:47:00Z"
  tasks_completed: 3
  tasks_total: 3
  files_changed: 11
---

# Phase 146 Plan 05: GAP-146-A Closure — Held-Cap Reuse for Open-in-Browser Summary

**One-liner:** Daemon read endpoint + Wails binding + frontend held-cap reuse for remote "Open in browser" — second open reuses stored cap without prompting for a join code (GAP-146-A closed).

## What Was Built

### Task 1: Daemon read path + Wails binding (TDD)

- **`internal/daemon/remote_files.go`**: Added `handleRemoteSessionOpenURL` handler serving `GET /api/remote-files/caps/{sessionID}/open-url`. Reads `RemoteCapStore.Get(sessionID)` and composes `baseURL+/sessions/{id}?cap=TOKEN` via `strings.TrimRight`+`url.PathEscape`+`url.QueryEscape`. Returns 404 with `{"error":"no cap registered for session"}` when absent (mirrors `proxyRemoteFiles` shape). 500 on nil store.
- **`internal/daemon/api.go`**: Route registered next to existing cap deposit route.
- **`internal/daemon/client_remote_files.go`**: Added `RemoteSessionOpenURL(sessionID string) (string, error)` using `doJSON(GET, "/api/remote-files/caps/"+url.PathEscape(sessionID)+"/open-url", nil, &result)`.
- **`app.go`**: Added `OpenRemoteSessionURL(sessionID string) (string, error)` Wails binding. Guards `a.client == nil`. Forwards to client helper. Does NOT call `BrowserOpenURL` (frontend opens the returned URL, matching existing exchange→open convention).
- **`internal/daemon/open_remote_session_url_test.go`**: Go test file with `TestRemoteSessionOpenURL_HeldCap` (deposit + GET → `{"url": "https://peer:8443/sessions/sess-1?cap=TOK"}`) and `TestRemoteSessionOpenURL_NoCap` (no deposit → 404 + `{"error": "no cap registered for session"}`). Used `proxyTestAPI` helper pattern.

### Task 2: Frontend held-cap reuse, WR-01 fix, WR-03 fix, WR-02 tests (TDD)

- **`frontend/src/App.tsx`**:
  - `handleOpenRemoteSession` rewritten to check `remoteCapsCached.has(session.id)` FIRST. On hit: call `OpenRemoteSessionURL(session.id)` + `BrowserOpenURL(url)` + return (no modal). On failure (stale cap): fall through to modal for self-heal. On miss: existing `setJoinModalForSession` path unchanged (D-03 parity).
  - `handleModalExchange` open-session branch (WR-01 fix): after `ExchangeJoinCodeAtURL`, now calls `RegisterRemoteCap` + marks `remoteCapsCached` + calls `OpenRemoteSessionURL(pending.id)` for the daemon-composed URL instead of the hand-built `pending.id + '?cap='` form. Eliminates `/sessions/A?cap=<capForB>` mismatch.
  - `OpenRemoteSessionURL` added to Wails import line.
- **`frontend/src/components/RemoteJoinCodeModal.tsx`**:
  - `mapErrorMessage` WR-03 fix: added `not-found`/`already used`/`already-used` → `'Code already used or expired — ask the owner for a fresh code or use the share link.'` branch BEFORE the `invalid` branch. Keeps `invalid` → "Code invalid. Double-check the 8-character code (XXXX-XXXX)." separate.
- **`frontend/src/wailsjs/go/main/App.js`** + **`App.d.ts`**: Added `OpenRemoteSessionURL` export/declaration.
- **`frontend/src/components/__tests__/App.open-remote.test.tsx`**: Added WR-02 behavior tests — source-inspection of `handleOpenRemoteSession` for `remoteCapsCached.has` + `OpenRemoteSessionURL`, WR-01 absence of `pending.id + '?cap='`, and WR-03 modal component render test (`not-found` → "already used or expired", NOT "Double-check").
- **`frontend/src/components/__tests__/RemoteJoinCodeModal.test.tsx`**: Updated `not-found` test expectation to WR-03 corrected copy.

### Task 3: TESTING.md compliance

- **§2**: Go count 347→348, Total 467→468. Phase 146 Plan 05 note appended.
- **§4**: New FIX-03 row for `internal/daemon/open_remote_session_url_test.go`. Existing `App.open-remote.test.tsx` FIX-03 notes updated to include held-cap reuse + WR-03.
- **§5 M-13**: Updated to reflect two sub-scenarios — first open (no cap, modal) and second open (held cap, direct reuse without modal).
- Traceability gate exits 0.

## Verification Results

- `go build ./...` exits 0.
- `go test ./internal/daemon/ -run RemoteSessionOpenURL -count=1` passes (held→URL, absent→404).
- `cd frontend && pnpm exec tsc --noEmit` exits 0.
- `cd frontend && pnpm exec vitest run src/components/__tests__/App.open-remote.test.tsx` — 15 tests pass.
- `bash tests/check-traceability-paths.sh` exits 0 ("OK: all traceability paths exist").

## Acceptance Criteria Results

| Criterion | Status |
|-----------|--------|
| `go build ./...` exits 0 | PASS |
| `grep -v '^//' app.go \| grep -c 'func (a \*App) OpenRemoteSessionURL'` = 1 | PASS (1) |
| `grep -v '^//' internal/daemon/remote_files.go \| grep -c 'handleRemoteSessionOpenURL'` >= 1 | PASS (1) |
| `grep -c 'GET /api/remote-files/caps/{sessionID}/open-url' internal/daemon/api.go` = 1 | PASS (1) |
| `go test ./internal/daemon/ -run RemoteSessionOpenURL -count=1` passes | PASS |
| `grep -c 'remoteCaps.Get' internal/daemon/remote_files.go` >= 2 | PASS (2) |
| `pnpm exec tsc --noEmit` exits 0 | PASS |
| `grep -c 'remoteCapsCached.has' frontend/src/App.tsx` >= 2 | PASS (3) |
| `grep -c 'OpenRemoteSessionURL' frontend/src/App.tsx` >= 2 | PASS (4) |
| `grep -c "pending.id + '?cap='" frontend/src/App.tsx` = 0 | PASS (0) |
| `grep -c 'already used or expired' frontend/src/components/RemoteJoinCodeModal.tsx` >= 1 | PASS (1) |
| vitest passes including held-cap behavior, no-cap behavior, WR-03 error-copy tests | PASS (15/15) |
| `bash tests/check-traceability-paths.sh` exits 0 | PASS |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test slice size for handleOpenRemoteSession source-inspection**
- **Found during:** Task 2 GREEN phase
- **Issue:** Existing test `opens RemoteJoinCodeModal via setJoinModalForSession` used 600-char slice; new held-cap block pushed `setJoinModalForSession` to ~749 chars in.
- **Fix:** Updated test to use 1000-char slice (still covers just the handler body). Legitimate update as new code structure is longer.
- **Files modified:** `frontend/src/components/__tests__/App.open-remote.test.tsx`

**2. [Rule 1 - Bug] handleModalExchange test for /sessions/{id}?cap= URL no longer valid post-WR-01**
- **Found during:** Task 2 GREEN phase
- **Issue:** Existing test expected `/sessions/.*\?cap=` regex in `handleModalExchange` open-session branch (1000 chars); WR-01 fix removed the hand-built URL. The spirit of the test (verifying the cap-bearing URL is produced) is now satisfied by `OpenRemoteSessionURL` call.
- **Fix:** Updated test to assert `OpenRemoteSessionURL` appears in the branch and `pending.id + '?cap='` does NOT appear. Documents the WR-01 fix intent.
- **Files modified:** `frontend/src/components/__tests__/App.open-remote.test.tsx`

**3. [Rule 2 - Missing Critical] Wails bindings not regenerated**
- **Found during:** Task 2 implementation
- **Issue:** `App.js` and `App.d.ts` are auto-generated by `wails build/dev` but that toolchain was not run here. The frontend cannot import `OpenRemoteSessionURL` without the binding.
- **Fix:** Hand-extended `App.js` and `App.d.ts` with the `OpenRemoteSessionURL` export/declaration following existing patterns. `wails build` will regenerate these correctly from the Go source in production.
- **Files modified:** `frontend/src/wailsjs/go/main/App.js`, `frontend/src/wailsjs/go/main/App.d.ts`

## Threat Surface Scan

No new network endpoints, auth paths, or schema changes beyond what is in the plan's threat model:
- `GET /api/remote-files/caps/{sessionID}/open-url` is a daemon unix-socket local read (T-146-05-04 accept — same trust level as existing cap deposit endpoint).
- Cap token never returned to React except inside the composed URL (T-146-05-01 accept — identical to existing exchange→open path).

## Known Stubs

None. All data paths are wired end-to-end.

## Self-Check: PASSED

Files verified on disk:
- `internal/daemon/open_remote_session_url_test.go` — FOUND
- `app.go` — FOUND (OpenRemoteSessionURL binding)
- `frontend/src/App.tsx` — FOUND (held-cap reuse + WR-01)
- `frontend/src/components/RemoteJoinCodeModal.tsx` — FOUND (WR-03)

Commits verified in git log:
- `2fa1dd46` test(146-05): add failing tests for RemoteSessionOpenURL daemon read path
- `5e2abd9b` feat(146-05): daemon read path + Wails binding for cap-bearing open URL
- `c62d1c9b` test(146-05): add WR-02 held-cap/no-cap behavior tests + WR-03 error-copy assertions
- `a8cdc408` feat(146-05): held-cap reuse path, WR-01 SID-correct fallback, WR-03 error copy
- `21131505` chore(146-05): TESTING.md regression-convention compliance for open_remote_session_url_test.go
