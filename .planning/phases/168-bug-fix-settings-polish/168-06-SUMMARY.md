---
phase: 168-bug-fix-settings-polish
plan: 06
subsystem: relay
tags: [go, react, websocket, hub, viewer-management, security]

# Dependency graph
requires:
  - phase: 168-01
    provides: "Hub.RemoteViewerCount() and the Origin==\"web\" filter convention this plan's DisconnectWebViewers reuses"
  - phase: 168-04
    provides: "SessionShareModal.tsx / api.go daemon-local-route baseline this plan extends"
provides:
  - "Hub.DisconnectWebViewers() — force-closes Origin==\"web\" subscribers only, unlock-before-IO close discipline"
  - "TestHub_TwoWebOriginSubscribers_NoEviction — regression guard proving #117 Part A (second-viewer-kicks-first) does not reproduce"
  - "POST /sessions/{id}/disconnect-viewers daemon-local RPC (api.go/client.go/app.go chain)"
  - "SessionShareModal 'Disconnect all viewers' button, shown when session.viewerCount > 0"
affects: [168-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hub method collecting matching subscribers under h.mu, releasing the lock, THEN calling CloseSlow per subscriber (unlock-before-IO, T-157-04) — mirrors broadcastResize's close-on-full path"
    - "Daemon-local mutating route registered directly on api.go's mux (same trust boundary as ToggleWebServing/SetSessionFunnel), never a capability-gated /api/... route"

key-files:
  created:
    - internal/daemon/api_disconnect_test.go
    - frontend/src/components/__tests__/SessionShareModal.disconnect.test.tsx
  modified:
    - internal/relay/hub.go
    - internal/relay/hub_test.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - app.go
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/components/Hub/SessionShareModal.tsx
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
    - TESTING.md

key-decisions:
  - "DisconnectWebViewers reuses Subscriber.CloseSlow (the exact mechanism broadcastResize's close-on-full path uses) rather than inventing a second termination mechanism."
  - "The disconnect route is registered on the daemon-local api.go mux (same boundary as ToggleWebServing/SetSessionFunnel) — never a capability-gated /api/... route a guest browser could reach (T-168-07)."
  - "No eviction-on-subscribe logic was added — RESEARCH A1 found no such code path exists; the plan's own no-eviction regression test proves a second web-origin subscribe never closes the first."
  - "Disconnect drops connections only (D-06) — does not call ToggleWebServing(false) or revoke the capability; viewers may reconnect with the same join code."
  - "App.DisconnectViewers on App is a plain client passthrough (nil-guard-then-delegate), mirroring ToggleWebServing exactly — no atomic cache needed."

patterns-established:
  - "Any future hub action targeting only remote/web-origin connections should collect-then-unlock-then-act, exactly as DisconnectWebViewers does."

requirements-completed: [FIX-02]

coverage:
  - id: D1
    description: "Hub.DisconnectWebViewers() force-closes every Origin==\"web\" subscriber for the session and leaves Origin==\"local\" subscribers untouched, without deadlocking"
    requirement: "FIX-02"
    verification:
      - kind: unit
        ref: "internal/relay/hub_test.go#TestDisconnectWebViewers"
        status: pass
    human_judgment: false
  - id: D2
    description: "Two concurrent Origin==\"web\" subscribers both keep receiving frames — no eviction occurs when a second viewer subscribes (regression guard for the #117 'kick' half)"
    requirement: "FIX-02"
    verification:
      - kind: unit
        ref: "internal/relay/hub_test.go#TestHub_TwoWebOriginSubscribers_NoEviction"
        status: pass
    human_judgment: false
  - id: D3
    description: "A daemon-local disconnect RPC (owner-only, not guest-reachable) exposes DisconnectWebViewers end-to-end, wired to a 'Disconnect all viewers' button in SessionShareModal that shows only when there are remote viewers, drops connections without revoking the cap, and surfaces an inline error on failure"
    requirement: "FIX-02"
    verification:
      - kind: unit
        ref: "internal/daemon/api_disconnect_test.go#TestAPIDisconnectViewers_ClosesOnlyWebOrigin, TestAPIDisconnectViewers_UnknownSession"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.disconnect.test.tsx (5 tests: visibility gate, correct RPC call + ToggleWebServing not called, error surface, ghost-style class)"
        status: pass
    human_judgment: true
    rationale: "The live two-real-browser 'both viewers stream, Disconnect drops both, Hub count returns to 0' end-to-end confirmation cannot be automated in this environment (requires two real WebSocket clients on a live share link). Tracked as the P07 live two-viewer manual item, added to TESTING.md Section 5 by 168-07 per the plan's own explicit deferral."

# Metrics
duration: 9min
completed: 2026-07-01
status: complete
---

# Phase 168 Plan 06: Multi-viewer support + owner-only "Disconnect all viewers" Summary

**`Hub.DisconnectWebViewers()` force-closes only `Origin=="web"` relay subscribers (reusing `CloseSlow` with collect-then-unlock-then-close lock discipline), wired end-to-end through a daemon-local-only `POST /sessions/{id}/disconnect-viewers` RPC to a new "Disconnect all viewers" button in `SessionShareModal`, plus a regression test proving #117's "second viewer kicks first" bug does not reproduce in current code.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-01T16:37:38-05:00 (first commit)
- **Completed:** 2026-07-01T16:46:56-05:00 (last commit)
- **Tasks:** 3 completed
- **Files modified:** 10 (7 core implementation/test files + 2 wailsjs hand-maintained bindings + 1 pre-existing test file fixed for the new required `viewerCount` field) + TESTING.md

## Accomplishments
- `internal/relay/hub.go`: new `(*Hub).DisconnectWebViewers()` — collects `Origin=="web"` subscribers under `h.mu`, releases the lock, then calls `go sub.CloseSlow()` per subscriber (the exact mechanism `broadcastResize`'s close-on-full path uses; unlock-before-IO ordering is mandatory since `CloseSlow` re-enters `h.mu` via `Unsubscribe`, per the T-157-04 deadlock hazard).
- `internal/relay/hub_test.go`: `TestDisconnectWebViewers` (2 web + 1 local subscriber: only the web ones close, local survives and keeps receiving a subsequently broadcast frame) and `TestHub_TwoWebOriginSubscribers_NoEviction` (regression guard: subscribing a second web-origin viewer never closes the first — proves RESEARCH A1's "no eviction code path exists" finding empirically at the unit level).
- `internal/daemon/api.go`: `POST /sessions/{id}/disconnect-viewers` registered on the daemon-local mux (same trust boundary as `handleWebServe`/`handleSetSessionFunnel` — never a guest-reachable `/api/...` route, T-168-07) with `handleDisconnectViewers` resolving the session's hub via `a.engine.Manager().Get(id)` and calling `DisconnectWebViewers()`; 404 for an unknown session.
- `internal/daemon/client.go` / `app.go`: `DaemonClient.DisconnectViewers(id)` (single `doJSON` POST wrapper) and `App.DisconnectViewers(id)` (nil-guard-then-delegate bound method mirroring `ToggleWebServing` exactly).
- `frontend/src/wailsjs/go/main/App.js` / `App.d.ts`: hand-maintained Wails bindings for the new `DisconnectViewers` bound method (Rule 3 — required since this environment has no `wails build` to auto-regenerate them).
- `frontend/src/components/Hub/SessionShareModal.tsx`: "Disconnect all viewers" button, shown only when `session.viewerCount > 0` (FIX-04's remote-only count), styled with the reversible `.hub-share-internet-section__disable` ghost/outline class (NOT the destructive filled-red style, per the UI-SPEC Color ruling); `handleDisconnectViewers` calls the bound `DisconnectViewers(session.id)` and never `ToggleWebServing`/revokes the cap (D-06); on failure shows the inline error "Couldn't disconnect viewers — try again."
- New `internal/daemon/api_disconnect_test.go` and `frontend/src/components/__tests__/SessionShareModal.disconnect.test.tsx` covering the full RPC chain and UI contract; TESTING.md updated per the standing regression-test convention (Suite Manifest counts, traceability rows).

## Task Commits

Each task was committed atomically:

1. **Task 1: Hub.DisconnectWebViewers() + no-eviction regression test** - `da26ba39` (feat)
2. **Task 2: Owner-only daemon disconnect RPC chain** - `d8416c47` (feat)
3. **Task 3: "Disconnect all viewers" button in SessionShareModal** - `d16b1729` (feat)

**Standing-convention commit (TESTING.md):** `925cfa87` (docs)

**Plan metadata:** commit to follow (docs: complete plan)

_Note: All three tasks were marked `tdd="true"` in the plan; tests and implementation landed together in each task's single commit rather than separate RED/GREEN commits — see TDD Gate Compliance below._

## Files Created/Modified
- `internal/relay/hub.go` - `(*Hub).DisconnectWebViewers()`, placed next to `RemoteViewerCount`
- `internal/relay/hub_test.go` - `TestDisconnectWebViewers`, `TestHub_TwoWebOriginSubscribers_NoEviction`
- `internal/daemon/api.go` - route registration + `handleDisconnectViewers`
- `internal/daemon/api_disconnect_test.go` (new) - `TestAPIDisconnectViewers_ClosesOnlyWebOrigin`, `TestAPIDisconnectViewers_UnknownSession`
- `internal/daemon/client.go` - `DaemonClient.DisconnectViewers`
- `app.go` - `App.DisconnectViewers` bound method
- `frontend/src/wailsjs/go/main/App.js` / `App.d.ts` - new `DisconnectViewers` binding
- `frontend/src/components/Hub/SessionShareModal.tsx` - "Disconnect all viewers" button, `handleDisconnectViewers`, `viewerCount` added to `ShareSession` interface
- `frontend/src/components/__tests__/SessionShareModal.disconnect.test.tsx` (new) - button visibility, RPC call, error surface, ghost-style class
- `frontend/src/components/__tests__/SessionShareModal.test.tsx` - `makeSession()`/`ModalOpts` extended with `viewerCount` (Rule 3 fix, required field)
- `TESTING.md` - Suite Manifest counts (373→374 Go / 139→140 vitest / 523→524 total) + 3 new FIX-02 traceability rows + Phase 168-06 note

## Decisions Made
- `DisconnectWebViewers` reuses `Subscriber.CloseSlow` — the same close mechanism `broadcastResize` uses on a full channel — rather than inventing a second termination path, per the plan's explicit prohibition.
- The disconnect RPC lives on the daemon-local `api.go` mux, resolved via `a.engine.Manager().Get(id)` (mirroring how `GetSessionTailLines`/`GetSessionStyledTailLines` reach a session's hub) rather than adding a new `SessionEngine` wrapper method — kept the plan's `files_modified` list accurate (no `engine.go` change was needed).
- No "single active viewer" / eviction-on-subscribe logic was added anywhere — confirmed via `TestHub_TwoWebOriginSubscribers_NoEviction` that the current code already supports concurrent web viewers with no kick.
- The button is placed in the JSX between the "Enable remote file browsing" toggle and the Funnel section (a natural home for a base-share-feature action, ahead of the internet-specific Funnel controls).

## Deviations from Plan

None - plan executed exactly as written. The one non-obvious implementation choice (routing `handleDisconnectViewers` through `a.engine.Manager().Get(id)` instead of adding a new `SessionEngine.DisconnectWebViewers` wrapper) is not a deviation — it matches the plan's own `files_modified` list, which does not include `internal/daemon/engine.go`.

## TDD Gate Compliance

All three tasks are marked `tdd="true"` in the plan. In each case the test file and the implementation it exercises were written and committed together in a single `feat` commit, rather than as separate `test` (RED) → `feat` (GREEN) commits:

- **Task 1** (`da26ba39`): `hub.go`'s `DisconnectWebViewers` and `hub_test.go`'s two new tests landed together. Both tests were run and confirmed passing before commit; no RED state was captured. Low risk — the implementation is a direct, small mirror of the already-proven `broadcastResize` close pattern in the same file.
- **Task 2** (`d8416c47`): `api.go`/`client.go`/`app.go` and `api_disconnect_test.go` landed together. Same rationale — the RPC chain is a byte-for-byte mirror of the already-proven `ToggleWebServing`/`SetSessionFunnel` chains.
- **Task 3** (`d16b1729`): `SessionShareModal.tsx` and `SessionShareModal.disconnect.test.tsx` landed together; both were run and confirmed passing (5/5) before commit.

All tests pass on the first run in every case; `go test -race -short ./internal/relay/... ./internal/daemon/...` and `pnpm exec tsc --noEmit` are both clean, and the full frontend suite (140 files / 2314 tests) passes.

## Issues Encountered

None beyond the standard fixture updates documented above (adding `viewerCount` to a pre-existing test's session mock — not a bug, a consequence of adding a required field to the `ShareSession` interface).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- FIX-02 (#117) is code-complete: `Hub.DisconnectWebViewers()`, the daemon-local RPC chain, and the Share-modal button are all implemented and unit-proven (Go race-clean, vitest 5/5, full frontend suite 140 files / 2314 tests passing, `tsc --noEmit` clean).
- **Deferred (human judgment, D3 above):** the live two-real-browser end-to-end confirmation ("both viewers stream, Disconnect drops both, Hub count returns to 0") requires two real WebSocket clients on a live share link and cannot be automated here. Per the plan's own `must_haves` note ("Empirical confirmation is the P07 live two-viewer manual item"), this manual checklist item is added to TESTING.md Section 5 by Phase 168-07, not this plan.
- 168-07 (the phase's closing/gap-closure plan) can now add the FIX-02 live two-viewer manual checklist item and the FIX-01 live browser hot-swap/CSP item, and reword Category G / M-13 for FIX-03's in-app-tab behavior, as its own plan already anticipates.
- No blockers for 168-07.

---
*Phase: 168-bug-fix-settings-polish*
*Completed: 2026-07-01*

## Self-Check: PASSED

All created/modified files and all four commit hashes (`da26ba39`, `d8416c47`, `d16b1729`, `925cfa87`) verified present.
