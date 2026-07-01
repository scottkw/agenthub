---
phase: 168-bug-fix-settings-polish
plan: 03
subsystem: frontend
tags: [react, vitest, tabs, remote-session, tailscale, web-share]

# Dependency graph
requires:
  - phase: 168-02
    provides: "WebShareSessionView baseURL?: string prop — apiBaseURL/wsURL derive from a resolved origin (default window.location.origin) instead of a hardcoded window.location reference, so remote-peer tabs can reuse the component"
provides:
  - "openWebSessionTab(sessionId, baseURL?, capToken?) — extended signature; new web-session tabs carry their own baseURL/capToken on the Tab object (TabBar.tsx), not a single mount-stable global"
  - "handleOpenRemoteSession and handleModalExchange's open-session branch open an in-app web-session tab instead of BrowserOpenURL (D-17, #118)"
  - "__websession__ render branch resolves sessionId/capToken/baseURL from the active tab, falling back to the mount-stable webParams only for the app's own web-share bootstrap tab"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-tab param carry: Tab gains optional baseURL/capToken fields (TabBar.tsx) instead of a single mount-stable webParams, so multiple concurrent remote-peer web-session tabs never cross-contaminate each other's cap/host"
    - "OpenRemoteSessionURL's daemon-composed URL is parsed client-side (origin -> baseURL, ?cap= -> capToken) instead of being handed to BrowserOpenURL — preserves the WR-01 SID-correctness guarantee (daemon builds the URL from its own RemoteCapStore entry) while routing the open in-app"

key-files:
  created: []
  modified:
    - frontend/src/App.tsx
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/__tests__/App.open-remote.test.tsx
    - TESTING.md

key-decisions:
  - "pluginConfig is suppressed (undefined) for remote-peer web-session tabs so WebShareSessionView's isWebGuest self-fetch (168-02) fires instead of applying this daemon's OWN plugin config (via App's local GetPluginSettings state) to a different peer's session — a correctness gap not explicitly named in the plan's task list but directly implied by the per-tab render-branch rewrite and flagged as intentional in 168-02's Next Phase Readiness note."
  - "Both the held-cap branch and the join-code modal-exchange success branch keep calling OpenRemoteSessionURL (not a hand-built URL) before opening the in-app tab, preserving the Phase 146 WR-01 fix (daemon composes the URL from its own cap store, so the path SID always matches the deposited cap) rather than reintroducing a mismatch-prone client-side URL."

requirements-completed: [FIX-03]

coverage:
  - id: D1
    description: "handleOpenRemoteSession opens the remote session in an in-app web-session tab (openWebSessionTab) instead of BrowserOpenURL, connecting via the remote peer's baseURL + cap"
    requirement: "FIX-03"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/App.open-remote.test.tsx#App.tsx — handleOpenRemoteSession held-cap reuse (GAP-146-A, Plan 05) — held-cap path: handleOpenRemoteSession does NOT call BrowserOpenURL (FIX-03, D-17)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/App.open-remote.test.tsx#App.tsx — handleOpenRemoteSession held-cap reuse (GAP-146-A, Plan 05) — held-cap path: handleOpenRemoteSession calls openWebSessionTab with the parsed cap-bearing URL (FIX-03)"
        status: pass
      - kind: other
        ref: "cd frontend && pnpm exec tsc --noEmit"
        status: pass
    human_judgment: false
  - id: D2
    description: "The join-code / cap-exchange flow (RemoteJoinCodeModal, intent='open-session') that supplies the cap is preserved, and its success handler opens the in-app tab the same way"
    requirement: "FIX-03"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/App.open-remote.test.tsx#App.tsx — handleModalExchange open-session branch (FIX-03) — open-session branch calls openWebSessionTab, NOT BrowserOpenURL, with the cap-bearing URL (FIX-03, D-17)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/App.open-remote.test.tsx#App.tsx — handleModalExchange open-session branch (FIX-03) — open-session branch uses OpenRemoteSessionURL to get the cap-bearing URL (WR-01 fix, preserved by FIX-03)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Opening two different remote sessions produces two independent web-session tabs, each rendering with its own sessionId, capToken, and baseURL"
    requirement: "FIX-03"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/App.open-remote.test.tsx#App.tsx — openWebSessionTab per-tab isolation (FIX-03, two remote sessions)"
        status: pass
    human_judgment: true
    rationale: "openWebSessionTab is internal to the App component (not exported) and App.tsx is not fully mounted in this codebase's test suite (established convention: source-inspection via App.tsx?raw for App-level logic, per App.test.tsx and this file's pre-existing tests). The source-inspection tests prove the mechanism (tab id keyed by sessionId argument; baseURL/capToken sourced from the call's own params, never the global webParams) but a live two-tab render was not exercised end-to-end — recommend a live UAT opening two different remote peers' sessions from the Hub and confirming both terminal tabs stream independently with correct hosts."

# Metrics
duration: 8min
completed: 2026-07-01
status: complete
---

# Phase 168 Plan 03: Remote session opens in an in-app tab Summary

**Opening a remote tailnet session from the Hub now opens an in-app terminal/chat tab (openWebSessionTab, per-tab baseURL/capToken) instead of launching an external browser window, reversing the Phase 146 out-of-band design (D-17, #118).**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-01T15:31:37-05:00
- **Completed:** 2026-07-01T15:39:00-05:00
- **Tasks:** 2
- **Files modified:** 4 (3 frontend + TESTING.md)

## Accomplishments

- `openWebSessionTab`'s signature is extended from `(sessionId)` to `(sessionId, baseURL?, capToken?)`; new web-session tabs carry their own `baseURL`/`capToken` on the `Tab` object (`TabBar.tsx` gains two optional fields), so multiple concurrent remote-peer tabs never cross-contaminate each other's cap/host.
- The `__websession__` render branch now resolves `sessionId`/`capToken`/`baseURL` from the active tab, falling back to the mount-stable `webParams` only for the app's own web-share bootstrap tab (`mode==='web'`) — unchanged behavior for that path.
- `handleOpenRemoteSession` (held-cap branch) and `handleModalExchange`'s `open-session` branch (join-code cap-exchange success path) both now call `openWebSessionTab` after parsing `OpenRemoteSessionURL`'s daemon-composed URL (`origin` -> `baseURL`, `?cap=` -> `capToken`), instead of `BrowserOpenURL`. The WR-01 SID-correctness guarantee (the daemon builds the URL from its own `RemoteCapStore` entry, so the path SID always matches the deposited cap) is preserved.
- `pluginConfig` is now suppressed (`undefined`) for remote-peer web-session tabs, so `WebShareSessionView`'s web-guest self-fetch (168-02, FIX-01) fires and pulls the CORRECT peer's plugin config instead of applying this daemon's own local plugin settings to a different peer's session.
- `App.open-remote.test.tsx` extended (21 tests, all pass): 3 tests rewritten to assert the NEW in-app contract (`BrowserOpenURL` no longer called; `openWebSessionTab` called with `session.id`), plus 4 new tests proving per-tab isolation (tab id keyed by `sessionId`; `baseURL`/`capToken` sourced from the call's own params, never the global `webParams`; the render branch reads `activeWebTab`/`isRemoteWebTab`).

## Task Commits

Each task was committed atomically:

1. **Task 1: Make web-session tab params per-tab (signature + render branch)** - `98ac41ba` (feat)
2. **Task 2: Reroute handleOpenRemoteSession to the in-app tab** - `e1aa4c32` (test, RED) -> `f3f18be3` (feat, GREEN)

**Additional commit (standing convention, TESTING.md):** `d1616496` (docs)

**Plan metadata:** commit to follow (docs: complete plan)

## Files Created/Modified

- `frontend/src/App.tsx` - `openWebSessionTab` signature extended to `(sessionId, baseURL?, capToken?)`; `__websession__` render branch resolves per-tab params via an `activeWebTab`/`isRemoteWebTab` lookup instead of the global `webParams`; `handleOpenRemoteSession` and `handleModalExchange`'s open-session branch parse `OpenRemoteSessionURL`'s result and call `openWebSessionTab` instead of `BrowserOpenURL`.
- `frontend/src/components/TabBar.tsx` - `Tab` interface gains optional `baseURL?: string` and `capToken?: string` fields.
- `frontend/src/components/__tests__/App.open-remote.test.tsx` - Rewrote the `open-session` / held-cap BrowserOpenURL assertions to require `openWebSessionTab` + absence of `BrowserOpenURL`; added a new describe block covering per-tab isolation (tab-id keying, param sourcing, render-branch source gates).
- `TESTING.md` - Section 2 (Suite Manifest) "no new file" note for Phase 168-03; Section 4 (traceability) FIX-03 row disambiguated for the v4.2 milestone (same physical test file as the pre-existing Phase 146 FIX-03 rows, per the requirement-ID-scoped-per-milestone convention established in 168-02).

## Decisions Made

- **`pluginConfig` suppressed for remote-peer tabs.** Not explicitly listed in the plan's task text, but directly implied by the per-tab render-branch rewrite and flagged as the intended design in 168-02's "Next Phase Readiness" note ("App.tsx's own pluginConfig state is irrelevant to a different peer's session"). Without this, a desktop user with a locally-loaded `pluginConfig` (from `GetPluginSettings`, this daemon's own RPC) opening a remote peer's session in-app would have had that LOCAL config silently applied to the remote peer's terminal, since `WebShareSessionView`'s `isWebGuest` gate is `pluginConfig === undefined`. Classified as Rule 2 (missing critical functionality) — this phase's own per-tab work is what makes the bug reachable for the first time.
- **`OpenRemoteSessionURL` kept in both branches; URL parsed instead of hand-building.** The plan's `<action>` text explicitly directs deriving the baseURL via `remoteBaseURLFor`/`findRemoteSession`, but the held-cap and modal-exchange branches already had a daemon-composed, SID-correct URL available via `OpenRemoteSessionURL` (the Phase 146-05 WR-01 fix). Reusing that URL (parsed via `new URL()`) rather than re-deriving `baseURL` from `remoteBaseURLFor(session)` preserves WR-01's mismatch protection — the daemon, not the client, determines which cap/SID pair the URL encodes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Suppressed pluginConfig for remote-peer web-session tabs**
- **Found during:** Task 1
- **Issue:** The plan's Task 1 action describes resolving `sessionId`/`capToken`/`baseURL` per-tab but does not mention `pluginConfig`. Left unchanged, a remote-peer tab would inherit App's local `pluginConfig` state (this daemon's own plugin settings via `GetPluginSettings`), which is unrelated to — and potentially conflicting with — the actual remote peer's plugin configuration. This would also suppress `WebShareSessionView`'s `isWebGuest` self-fetch (168-02, FIX-01) for remote tabs, since `isWebGuest = pluginConfig === undefined` and a loaded local config is not `undefined`.
- **Fix:** The render branch now passes `pluginConfig={isRemoteWebTab ? undefined : (pluginConfig ?? undefined)}` — remote tabs always see `undefined`, triggering the web-guest self-fetch against the CORRECT peer's `/api/plugin-config` endpoint (via the tab's own `baseURL`).
- **Files modified:** `frontend/src/App.tsx`
- **Verification:** `cd frontend && pnpm exec tsc --noEmit` passes; `cd frontend && pnpm vitest run` — 137 files / 2286 tests pass.
- **Committed in:** `98ac41ba` (Task 1 commit)

**2. [Rule 1 - Bug] Doc-comment string collision broke source-inspection tests**
- **Found during:** Task 2 (writing RED tests)
- **Issue:** A doc comment added above `openWebSessionTab` in Task 1 contained the literal substring `handleOpenRemoteSession`, appearing BEFORE the real `const handleOpenRemoteSession = ...` definition in the file. Several pre-existing tests use `raw.indexOf('handleOpenRemoteSession')` to locate the function's source for slicing — the comment's earlier occurrence made those tests inspect the wrong code region and fail.
- **Fix:** Reworded the comment to avoid the literal camelCase identifier (uses "the remote-open action" instead). Also removed two later occurrences of the literal string `BrowserOpenURL` from doc comments near `handleModalExchange`'s definition, which were tripping the new `not.toContain('BrowserOpenURL')` assertions on that function's own doc comments (not its executable code).
- **Files modified:** `frontend/src/App.tsx`
- **Verification:** `cd frontend && pnpm vitest run App.open-remote` — 21/21 pass.
- **Committed in:** `e1aa4c32` (test/RED commit) and `f3f18be3` (feat/GREEN commit)

**3. [Standing convention, ./CLAUDE.md + TESTING.md] Registered the App.tsx / App.open-remote.test.tsx changes in TESTING.md**
- **Found during:** post-Task-2 (before final commit)
- **Issue:** `./CLAUDE.md`'s Regression Test Convention requires documenting extensions to existing test files in TESTING.md, including a Section 4 traceability entry. The existing Phase 146 `FIX-03` row for this same file had a description that was now stale (it described the `BrowserOpenURL` contract this plan just removed).
- **Fix:** Added a Section 2 "no new file" note for Phase 168-03, and a NEW Section 4 `FIX-03` row disambiguated for the v4.2 milestone (following the exact precedent set by 168-02's `FIX-01` disambiguation), while updating the stale Phase 146 row's wording to reflect the current file contents accurately.
- **Files modified:** `TESTING.md`
- **Verification:** `bash tests/check-traceability-paths.sh` exits 0.
- **Committed in:** `d1616496`

---

**Total deviations:** 3 auto-fixed (1 Rule-2 missing-critical fix directly caused by this plan's own per-tab rewrite, 1 Rule-1 test-infrastructure bug fix, 1 standing-convention doc update)
**Impact on plan:** All three fixes are direct, necessary consequences of the plan's own instructions (per-tab param resolution) plus a mandatory project convention. No scope creep — no unrelated code was touched.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- FIX-03 (#118) is code-complete: the in-app remote-open path is implemented and unit-proven via source inspection (`App.open-remote.test.tsx`, 21/21 pass), `tsc --noEmit` clean, full frontend suite (137 files / 2286 tests) passes, and `vite build` succeeds.
- **Deferred (human judgment, D3 above):** live end-to-end confirmation that opening TWO different remote tailnet sessions from the Hub actually renders two independent, correctly-streaming terminal tabs (distinct hosts/caps) — `openWebSessionTab` is internal to `App.tsx` and not exercised by a full-App-mount test in this codebase's established testing convention. Recommend folding into this phase's other deferred live-UAT items (tracked in STATE.md) or the next `/gsd-verify-work 168` pass.
- The M-13 manual-checklist item (Category G, referenced in 168-PATTERNS.md's regression-suite wiring section) still needs rewording from "opens in an external browser" to "opens in an in-app tab" — not done in this plan (test-file scope only per the plan's `files_modified`); flag for whichever plan/pass owns the manual-checklist Category G item in this phase.

---
*Phase: 168-bug-fix-settings-polish*
*Completed: 2026-07-01*
