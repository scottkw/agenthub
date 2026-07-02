---
phase: 168-bug-fix-settings-polish
plan: 09
subsystem: ui
tags: [react, websocket, daemon-proxy, tailscale, remote-session, xterm, flexbox, heroicons]

# Dependency graph
requires:
  - phase: 168-03
    provides: in-app __websession__ remote tab (openWebSessionTab, D-17 in-app reroute) — the surface this plan repairs
  - phase: 134
    provides: HubInteractiveModal daemon-proxy transport (remote prop → TerminalPanel → RelayClient /api/relay/remote/{id}/ws) — the proven pattern mirrored here
provides:
  - Remote in-app web-session tab streams the remote PTY through the local daemon proxy (no direct cross-origin peer wss that the Origin allowlist 403s)
  - Remote web-session tab fills full pane height (.terminal-wrapper) — no half-height terminal, no dead band
  - Chat hidden on remote tabs (terminal-only); native web-guest tab unchanged (chat present)
  - Remote-card menu item relabeled "Open in tab" with an in-app glyph (matches D-17 in-app-tab behavior)
affects: [168-verify-work, FIX-03 live UAT Test 3, remote-session, web-share]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Remote-peer terminal transport = daemon proxy (remote:true, NO wsURL) — never a client-built cross-origin wss; RelayClient looks up the cap server-side by sessionID"
    - "__websession__ render branch wrapped in .terminal-wrapper for the full-height flex-column chain rather than editing shared .hub-modal__* classes (kept intact for HubInteractiveModal)"

key-files:
  created: []
  modified:
    - frontend/src/components/Hub/WebShareSessionView.tsx
    - frontend/src/components/Hub/WebShareSessionView.test.tsx
    - frontend/src/App.tsx
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/__tests__/App.open-remote.test.tsx
    - frontend/src/components/__tests__/SessionCard.share.test.tsx
    - TESTING.md

key-decisions:
  - "Route remote tabs through the daemon proxy (remote:true, no wsURL) instead of trying to authenticate the direct peer wss — the peer's byte-exact Origin allowlist is architecturally un-satisfiable from the webview (cannot set the WS Origin header)"
  - "Hide chat for remote tabs rather than build a cross-origin chat proxy (backend, out of scope); mirrors the working Phase 134 modal, which also has no remote ChatPanel route — terminal is the reported priority"
  - "Fix RC-B with a .terminal-wrapper wrapper (the same chain normal terminal tabs use) rather than editing the shared .hub-modal__* flex classes, which must keep working in HubInteractiveModal"
  - "Keep the prop name onOpenInBrowser as-is (renaming the callback is a wider refactor, out of scope); only the visible label + glyph changed"

patterns-established:
  - "Per-tab remote transport switch: WebShareSessionView `remote` prop gates BOTH the terminal transport (proxy vs direct) AND chat visibility"

requirements-completed: [FIX-03]

coverage:
  - id: D1
    description: "Remote in-app web-session tab routes the terminal through the daemon proxy (remote:true, no wsURL → ws://127.0.0.1:{relayPort}/api/relay/remote/{id}/ws), not the direct cross-origin peer wss the Origin allowlist 403s (RC-A)"
    requirement: "FIX-03"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/WebShareSessionView.test.tsx#remote={true}: TerminalPanel gets remote:true AND no wsURL (daemon proxy, not direct peer wss)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/WebShareSessionView.test.tsx#default path (no remote): TerminalPanel still gets the direct wsURL, remote falsy"
        status: pass
    human_judgment: true
    rationale: "Unit tests prove the prop wiring (remote:true, no wsURL) at the React seam, but that the peer PTY actually streams end-to-end through the daemon proxy over a live tailnet (no 403, real bytes) is only provable in live UAT Test 3 on a prod build against a real remote peer."
  - id: D2
    description: "Remote web-session tab fills full pane height — no half-height terminal, no --hub-bg dead band (RC-B); native web-guest tab not regressed"
    requirement: "FIX-03"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/App.open-remote.test.tsx#RC-B/RC-A: the __websession__ branch wraps WebShareSessionView in .terminal-wrapper and threads remote={isRemoteWebTab}"
        status: pass
    human_judgment: true
    rationale: "Source-inspection proves the .terminal-wrapper is present, but the actual rendered pane height (full-bleed vs dead band) is a visual/layout property that jsdom does not compute — must be confirmed by eye on a live prod build (live UAT Test 3)."
  - id: D3
    description: "Chat hidden on remote tabs (terminal-only); native web-guest bootstrap tab still shows chat toggle + ChatPanel (RC-A follow-up)"
    requirement: "FIX-03"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/WebShareSessionView.test.tsx#remote={true}: chat toggle button and ChatPanel are NOT rendered (terminal-only)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/WebShareSessionView.test.tsx#no remote: chat toggle button and ChatPanel ARE rendered (native web-guest parity)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Remote-card overflow menu reads 'Open in tab' with an in-app WindowIcon; the stale 'Open in browser' external-link wording is gone (RC-C / D-17)"
    requirement: "FIX-03"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionCard.share.test.tsx#remote card overflow menu contains \"Open in tab\" and NOT the stale \"Open in browser\""
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionCard.share.test.tsx#local card overflow menu does NOT contain \"Open in tab\""
        status: pass
    human_judgment: false

# Metrics
duration: 9min
completed: 2026-07-02
status: complete
---

# Phase 168 Plan 09: FIX-03 Remote Web-Session Tab Gap Closure Summary

**Remote in-app web-session tab now streams the peer PTY through the local daemon proxy (dropping the doomed direct cross-origin wss), fills full pane height via .terminal-wrapper, hides chat, and its card menu reads "Open in tab" with an in-app glyph**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-02T15:06:14Z
- **Completed:** 2026-07-02T15:15:38Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- **RC-A closed:** `WebShareSessionView` gained a `remote?: boolean` prop. When true, the terminal routes through the daemon proxy (`remote:true` + no `wsURL` → `ws://127.0.0.1:{relayPort}/api/relay/remote/{id}/ws`, cap looked up server-side) — the same transport the working Phase 134 HubInteractiveModal uses — instead of the direct `wss://<peer>/sessions/{id}/ws?cap=` the peer's byte-exact Origin allowlist 403s (the blank-terminal cause). App threads `remote={isRemoteWebTab}` on the `__websession__` element.
- **RC-A follow-up closed:** chat toggle + ChatPanel are hidden for remote tabs (no cross-origin chat proxy route); the native web-guest path (`mode==='web'`) keeps chat unchanged.
- **RC-B closed:** the `__websession__` branch is wrapped in `<div className="terminal-wrapper" style={{ display: 'flex' }}>`, restoring the full-height flex-column chain so the modal's `flex:1` fills the pane — no half-height terminal, no `--hub-bg` dead band. Fixes both the remote tab and the native web-guest tab without touching the shared `.hub-modal__*` classes.
- **RC-C closed:** `SessionCard`'s remote menu item relabeled "Open in browser" → "Open in tab", swapping `ArrowTopRightOnSquareIcon` → in-app `WindowIcon` (D-17 opens an in-app tab, not a browser).
- Regression coverage added (4 new RED→GREEN behavior tests in WebShareSessionView.test.tsx + source-inspection/label assertions in App.open-remote + SessionCard.share); TESTING.md Suite Manifest note + 3 Section 4 FIX-03 traceability rows.

## Task Commits

Each task was committed atomically:

1. **Task 1: Route remote web-session tab through daemon proxy + hide chat (RC-A)** — `2deff5f5` (fix) — TDD: RED tests written first (proxy transport, native path unchanged, chat hidden), then implemented to GREEN.
2. **Task 2: Full-height wrapper (RC-B) + relabel remote menu (RC-C) + TESTING.md** — `3b3fea34` (fix)

_Task 1 followed the plan's per-task RED→GREEN discipline; both the test extension and the implementation are in the single atomic Task-1 commit._

## Files Created/Modified
- `frontend/src/components/Hub/WebShareSessionView.tsx` — added `remote?: boolean` prop; `useProxy` transport switch → TerminalPanel `remote={useProxy}` + `wsURL={useProxy ? undefined : wsURL}`; ChatPanel + chat-toggle gated behind `{!useProxy && …}`; updated `relayPort` doc comment (real daemon port for remote tabs).
- `frontend/src/components/Hub/WebShareSessionView.test.tsx` — added `remote` to CapturedTerminalProps; new describe block with 4 behavior tests (proxy transport, native path unchanged, chat hidden for remote, chat present for native).
- `frontend/src/App.tsx` — threaded `remote={isRemoteWebTab}` on the `__websession__` WebShareSessionView; wrapped the branch in `.terminal-wrapper` (display:flex); condensed the RC-B/CR-03 comments.
- `frontend/src/components/Hub/SessionCard.tsx` — import `WindowIcon` (dropped `ArrowTopRightOnSquareIcon`); menu item text "Open in browser" → "Open in tab" and icon swap.
- `frontend/src/components/__tests__/App.open-remote.test.tsx` — new RC-B/RC-A source-inspection test; updated menu-label behavior tests to "Open in tab"; bumped the three `__websession__` branch slice windows 1400/1800 → 2000.
- `frontend/src/components/__tests__/SessionCard.share.test.tsx` — updated 3 label tests to "Open in tab" (remote card contains it and NOT "Open in browser"; local card contains neither; enabled without roJoinCode).
- `TESTING.md` — Suite Manifest note for 168-09 (both files extended in-place, counts unchanged: 374 Go / 141 vitest / 526 total) + 3 Section 4 FIX-03 traceability rows.

## Decisions Made
- Routed remote tabs through the daemon proxy instead of attempting to authenticate the direct peer wss — the Origin allowlist is un-satisfiable from the webview (cannot set the WS Origin header), so the direct path is architecturally dead.
- Hid chat for remote tabs (matches the working Phase 134 modal, which has no remote ChatPanel route) rather than building a cross-origin chat proxy (backend, out of scope).
- Fixed RC-B with a `.terminal-wrapper` wrapper rather than editing the shared `.hub-modal__*` flex classes (which must keep working in HubInteractiveModal).
- Left the callback prop name `onOpenInBrowser` unchanged (renaming is a wider refactor, out of scope); only the visible label + glyph changed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Bumped `__websession__` branch test slice windows to accommodate the RC-B wrapper**
- **Found during:** Task 2 (RC-B wrapper)
- **Issue:** Adding the `.terminal-wrapper` div + its comment pushed `<WebShareSessionView>` and its `baseURL=`/`remote=` props past the pre-existing source-inspection tests' 1400-char slice windows (Pitfall-3 baseURL test and CR-03 keyed-remount test), causing 3 pre-existing App.open-remote tests to fail.
- **Fix:** Condensed the App.tsx RC-B/CR-03 comments AND bumped the three `__websession__` branch slice windows (1400/1800 → 2000) so all assertions reach the JSX. This is expected maintenance for a branch that the plan explicitly grew with a wrapper.
- **Files modified:** frontend/src/App.tsx (comment length), frontend/src/components/__tests__/App.open-remote.test.tsx (slice constants)
- **Verification:** `pnpm vitest run App.open-remote SessionCard.share` → 43/43 pass; full suite 2329/2329 pass.
- **Committed in:** 3b3fea34 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The slice-window bump is a direct, in-scope consequence of the plan's required RC-B wrapper — no scope creep. All plan artifacts and success criteria met exactly.

## Issues Encountered
- None beyond the deviation above. The traceability check emits a `grep: invalid option -- P` warning on macOS (BSD grep lacks `-P`), but the script still validates every path and exits 0 (`OK: all traceability paths exist`) — pre-existing environment quirk, not introduced here.

## Verification Results
- `cd frontend && pnpm exec tsc --noEmit` — clean (both tasks).
- `pnpm vitest run WebShareSessionView.test` — 15/15 pass (Task 1 gate).
- `pnpm vitest run App.open-remote SessionCard.share` — 43/43 pass (Task 2 gate).
- `pnpm vitest run` (full frontend suite) — 141 files / 2329 tests pass.
- `bash tests/check-traceability-paths.sh` — exit 0 (`OK: all traceability paths exist`).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All three FIX-03 root causes (RC-A blank/direct-wss, RC-B half-height, RC-C stale label) are closed at the source + test level. Ready for `/gsd-verify-work 168` live UAT Test 3 (remote in-app web-session tab) on a prod build against a real tailnet peer — the daemon-proxy stream and full-height render are the two remaining human-judgment (D1/D2) items that only a live remote peer can confirm.
- No blockers.

## Self-Check: PASSED

---
*Phase: 168-bug-fix-settings-polish*
*Completed: 2026-07-02*
