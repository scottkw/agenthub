---
phase: 168-bug-fix-settings-polish
plan: 05
subsystem: ui
tags: [react, vitest, wails, hub, sharing, tailscale-funnel]

# Dependency graph
requires:
  - phase: 168-03
    provides: "openWebSessionTab / per-tab web-session param pattern (sibling App.tsx work in the same wave)"
  - phase: 168-04
    provides: "SettingsTab.tsx / App.tsx baseline this plan extends"
provides:
  - "shareModalSession/setShareModalSession lifted to App.tsx as the single controlled Share-modal state (RESEARCH Pattern 4)"
  - "A single <SessionShareModal> instance rendered at the App.tsx top level (always mounted, not inside the conditionally-unmounted HubPanel)"
  - "openShareModalForActiveSession() helper — derives the active session from hubSessions + activeId"
  - "hubSessions seeded from the existing mount-time/retry-time ListSessions() call (not just the Hub-tab-gated 3s poll)"
  - "StatusBar.tsx single 'Share Session' label + onShareSession prop (onToggleWeb/direct ToggleWebServing removed)"
affects: [168-verify-work]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Lifted-state + single-render-site pattern: when a modal needs to be triggered from a component that fully unmounts (HubPanel, off the Hub tab) AND a component that's always mounted (StatusBar, per session tab), the modal's JSX render — not just its open/closed state — must live at the always-mounted ancestor (App.tsx). A controlled-prop setter threaded down is enough for the sometimes-mounted trigger; the render itself cannot live there."
    - "Dead-code cascade from a 'stop calling X directly' plan decision (D-14): removing a component's only call site to a handler can orphan an entire parallel state machine (here: the App-level pendingShellWebToggle/handleToggleWeb/<ShellWebShareBanner> shell-warning gate, built for the now-retired direct-toggle footer path). Grep every prop/state that ONLY existed to feed the removed call site before declaring the task done — tsc's noUnusedLocals catches the leaf but not always the parallel state machine behind it."

key-files:
  created:
    - frontend/src/components/__tests__/StatusBar.shareSession.test.tsx
  modified:
    - frontend/src/App.tsx
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/components/StatusBar.tsx
    - frontend/src/components/__tests__/App.shellWebShare.test.tsx
    - TESTING.md

key-decisions:
  - "SessionShareModal's JSX render moved from HubPanel.tsx to App.tsx (single top-level instance) rather than staying inside HubPanel with just the state lifted — the plan text explicitly left this as discretion ('rendered from HubPanel or lifted alongside the state'). HubPanel fully unmounts via a `{activeId === HUB_TAB.id && (...)}` conditional (not a persistent-mount + CSS-hidden pattern) whenever the user is on any other tab — which is exactly when the footer button renders. Lifting only the state and leaving the render inside HubPanel would have reproduced RESEARCH.md's own named failure mode ('the footer button would silently do nothing outside the Hub tab')."
  - "Considered, and rejected, making HubPanel persistently-mounted (CSS display:none when inactive, mirroring the terminal-tab pattern) as the alternative fix. HubPanel has an un-gated global `window.addEventListener('keydown', ...)` for the '/' search-focus shortcut with no isActive check — always-mounting it would make '/' steal focus to the (invisible) Hub search box from any other tab, a real regression. Moving the modal render was the narrower, lower-risk fix."
  - "handleShellWebShareConfirm/Cancel are kept (still threaded into the single App-level SessionShareModal) but rewritten to source sessionId from shareModalSession instead of the retired pendingShellWebToggle. This also fixes a latent, pre-existing bug: the modal's own inline shell-warn confirm click called onShellWebShareConfirm, but that handler bailed out on `if (!pendingShellWebToggle) return` — and nothing on the modal's path ever set pendingShellWebToggle (that state was populated only by the now-removed footer-direct-toggle handler). Confirming the modal's shell-warning banner would have silently no-op'd (UI shows sharing ON, ToggleWebServing never actually called) for any shell CLI session. shareModalSession is populated on every modal-open path, so this closes the gap."
  - "hubSessions is now also seeded from the pre-existing ListSessions() call inside App.tsx's init() and retryInit() (previously it was populated only by the Hub-tab-gated 3s poll). Without this, a user who never visits the Hub tab in a given app session (e.g. SESS-02 startup restore lands them directly on a session tab) would have an empty hubSessions array, and openShareModalForActiveSession's `hubSessions.find(...)` would silently find nothing — the exact 'footer button does nothing' failure mode the state-lift was meant to eliminate, just via stale/empty data instead of an unmounted component."

requirements-completed: [UX-02]

coverage:
  - id: D1
    description: "Footer button renders a single 'Share Session' label in both the OFF and ON web-share states (no more Enable/Disable two-branch toggle text) and calls onShareSession — never onToggleWeb/ToggleWebServing directly — on click"
    requirement: "UX-02"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/StatusBar.shareSession.test.tsx (16 tests: label in both states, click→onShareSession, ToggleWebServing/onToggleWeb absent from source, no button when webServerRunning=false)"
        status: pass
      - kind: other
        ref: "cd frontend && pnpm exec tsc --noEmit"
        status: pass
    human_judgment: false
  - id: D2
    description: "Hub card's Share button drives the controlled setShareModalSession prop (not local state); HubPanel no longer renders <SessionShareModal> itself"
    requirement: "UX-02"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/HubPanel.test.tsx#card Share button calls setShareModalSession(session), does not render <SessionShareModal> itself"
        status: pass
    human_judgment: false
  - id: D3
    description: "shareModalSession is the single lifted source of truth in App.tsx; exactly one <SessionShareModal> instance exists (always mounted); the footer button opens it for the active session end-to-end (including before the user has ever visited the Hub tab); the modal's own shell-warning confirm flow sources sessionId from shareModalSession"
    requirement: "UX-02"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/StatusBar.shareSession.test.tsx (App.tsx wiring + tab-type-gate describe blocks, source-inspection) and frontend/src/components/__tests__/App.shellWebShare.test.tsx (source-inspection)"
        status: pass
    human_judgment: true
    rationale: "App.tsx is not fully mounted in this codebase's established test convention (source-inspection via App.tsx?raw, consistent with App.shellWebShare.test.tsx / App.createTab.stayOnHub.test.tsx from 168-04). The source-inspection tests prove the wiring shape (single <SessionShareModal>, onShareSession→openShareModalForActiveSession→hubSessions.find, hubSessions seeded outside the Hub-poll effect) but a live end-to-end confirmation — click the footer button on a freshly-restored session tab (no prior Hub visit) and observe the modal opens showing correct funnelActive/viewerCount/webEnabled state for that session — was not exercised. Recommend folding into the phase's live-UAT pass."

# Metrics
duration: 29min
completed: 2026-07-01
status: complete
---

# Phase 168 Plan 05: Footer "Share Session" button Summary

**Lifted `shareModalSession` state and its single `<SessionShareModal>` render from HubPanel to App.tsx (RESEARCH Pattern 4) so the renamed footer "Share Session" button — which now only opens the modal instead of calling `ToggleWebServing` directly — works from any local session tab, not just while the Hub tab happens to be mounted.**

## Performance

- **Duration:** 29 min
- **Started:** 2026-07-01T21:01:01Z
- **Completed:** 2026-07-01T21:29:30Z
- **Tasks:** 2
- **Files modified:** 6 core (App.tsx, HubPanel.tsx, HubPanel.test.tsx, StatusBar.tsx, App.shellWebShare.test.tsx, TESTING.md) + 1 new test file + 1 deleted test file

## Accomplishments

- `frontend/src/App.tsx`: `shareModalSession`/`setShareModalSession` state lifted out of `HubPanel`'s local `useState`; `openShareModalForActiveSession()` helper derives the active session from `hubSessions`/`activeId`; the single `<SessionShareModal>` instance now renders at the App.tsx top level (always mounted, sibling to `NewSessionModal`/`RemoteJoinCodeModal`) instead of inside the conditionally-unmounted `HubPanel`; `hubSessions` is now also seeded from the existing mount-time/retry-time `ListSessions()` call so the footer works even before the user ever visits Hub; the retired App-level shell-warning gate for the direct-toggle footer path (`handleToggleWeb`, `pendingShellWebToggle`, the App-level `<ShellWebShareBanner>` banner-stack slot) is removed as dead code; `handleShellWebShareConfirm`/`handleShellWebShareCancel` are kept but rewritten to source `sessionId` from `shareModalSession`.
- `frontend/src/components/Hub/HubPanel.tsx`: `setShareModalSession` threaded in as a controlled prop; local `useState`, the 3s-poll sync effect, and the `<SessionShareModal>` render are all removed; the card's Share button (`handleShare`) now simply calls the prop. `webServerMode`/`webServerRunning`/`shellWebShareWarned`/`shellWebShareWarningEnabled`/`onShellWebShareConfirm`/`onShellWebShareCancel`/`onOpenHelp` props removed (they existed only to feed the now-removed modal render).
- `frontend/src/components/StatusBar.tsx`: single "Share Session" label replaces the "Enable Web"/"Disable Web" two-branch toggle; `onShareSession` prop replaces `onToggleWeb`; the footer no longer calls `ToggleWebServing` directly.
- The existing `App.tsx` tab-type filter (`welcome`/`settings`/`file-browser`/`hub`/`help`/`web-session`) already gated the entire `StatusBar`-bearing wrapper — D-15 (hide the button on non-shareable tabs) was structurally already satisfied, confirmed exhaustive by a new source-inspection test.
- `frontend/src/components/__tests__/StatusBar.shareSession.test.tsx` (new, 16 tests) supersedes the deleted `StatusBar.test.tsx`; `frontend/src/components/Hub/HubPanel.test.tsx` and `frontend/src/components/__tests__/App.shellWebShare.test.tsx` updated for the new controlled-prop / single-render-site contract.
- `tsc --noEmit` clean; full frontend suite (139 files / 2309 tests) passes.

## Task Commits

Both tasks landed as a single functional commit — see Deviations below for why the plan's two-task split turned out to not be independently compilable (StatusBar.tsx's prop contract and App.tsx's render site had to change together for the app to build at every commit checkpoint):

1. **Tasks 1+2: Lift shareModalSession to App.tsx; rename footer to Share Session** - `57b42e27` (feat)
2. **TESTING.md standing-convention update** - `6be58318` (docs)

**Plan metadata:** commit to follow (docs: complete plan)

## Files Created/Modified

- `frontend/src/App.tsx` - `shareModalSession` state lift, `openShareModalForActiveSession`, single `<SessionShareModal>` render, `hubSessions` mount/retry seeding, retired shell-warn-for-footer machinery removed, `handleShellWebShareConfirm`/`Cancel` re-sourced from `shareModalSession`.
- `frontend/src/components/Hub/HubPanel.tsx` - `setShareModalSession` controlled prop; local state/effect/modal-render/related props removed.
- `frontend/src/components/Hub/HubPanel.test.tsx` - `renderPanel` helper updated for the controlled prop; FUI-06 onOpenHelp-through-HubPanel test replaced with a `setShareModalSession` call-through test + a source-gate proving no `<SessionShareModal>` render remains.
- `frontend/src/components/StatusBar.tsx` - single "Share Session" label; `onShareSession` prop.
- `frontend/src/components/__tests__/StatusBar.shareSession.test.tsx` (new) - StatusBar component tests + App.tsx source-inspection wiring/gating tests (16 tests).
- `frontend/src/components/__tests__/StatusBar.test.tsx` (deleted) - superseded by `StatusBar.shareSession.test.tsx`; every assertion targeted the retired Enable/Disable Web behavior.
- `frontend/src/components/__tests__/App.shellWebShare.test.tsx` - retired `handleToggleWeb`/`pendingShellWebToggle`/App-level `<ShellWebShareBanner>` assertions removed; new assertions cover `handleShellWebShareConfirm` sourcing from `shareModalSession` and the shell-warn props threading into the single App-level `<SessionShareModal>` render (not `HubPanel`).
- `TESTING.md` - Suite Manifest note (net 0 new vitest files, 139 stays 139) + two new UX-02 traceability rows.

## Decisions Made

See `key-decisions` in the frontmatter above (modal render location, HubPanel-always-mounted alternative rejected, the latent shell-warn-confirm bug fix, and the `hubSessions` mount-time seed).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1/2 - Architectural discretion explicitly granted by the plan] Moved `<SessionShareModal>`'s render from HubPanel.tsx to App.tsx instead of leaving it inside HubPanel**
- **Found during:** Task 1
- **Issue:** The plan's task 1 action text literally allows either approach ("the existing one (rendered from HubPanel or lifted alongside the state) remains the single instance"), and 168-PATTERNS.md's illustrative snippet shows the modal staying inside HubPanel with only the state becoming a controlled prop. But `HubPanel` is rendered via a full `{activeId === HUB_TAB.id && (<HubPanel .../>)}` conditional — it completely unmounts on every non-Hub tab, which is exactly where the footer button lives. Leaving the modal's render inside HubPanel would have made the footer button silently do nothing outside the Hub tab — the exact failure mode 168-RESEARCH.md's own Pattern 4 section names as the thing lifting state alone (without moving the render) fails to prevent.
- **Fix:** Moved the `<SessionShareModal>` JSX (and its keep-fresh sync effect) from `HubPanel.tsx` to `App.tsx`, rendered unconditionally alongside other top-level modals (`NewSessionModal`, `RemoteJoinCodeModal`). `HubPanel` now only receives `setShareModalSession` and calls it from the card's Share button.
- **Files modified:** `frontend/src/App.tsx`, `frontend/src/components/Hub/HubPanel.tsx`, `frontend/src/components/Hub/HubPanel.test.tsx`
- **Verification:** `StatusBar.shareSession.test.tsx` asserts exactly one `<SessionShareModal>` in App.tsx's source; `HubPanel.test.tsx` asserts none remain in HubPanel's source; `tsc --noEmit` clean; full suite passes.
- **Committed in:** `57b42e27`

**2. [Rule 3 - Blocking] Removed the now-orphaned App-level shell-warning machinery for the retired footer-direct-toggle path**
- **Found during:** Task 2 (implementing D-14: "remove the footer's direct `ToggleWebServing` call")
- **Issue:** `App.tsx`'s `handleToggleWeb` (footer-only shell-warning interception, `pendingShellWebToggle` state, and an App-level `<ShellWebShareBanner>` banner-stack slot) existed solely to feed the footer's direct-toggle onClick. Once the footer stopped calling it (D-14), `handleToggleWeb` had zero call sites — `tsc`'s `noUnusedLocals` fails the build on the now-dead `const handleToggleWeb = useCallback(...)`, and `isShellCli`/`ShellWebShareBanner` became unused imports too.
- **Fix:** Removed `handleToggleWeb`, `pendingShellWebToggle` state, the App-level `<ShellWebShareBanner>` render (and its gating expression entry), and the now-unused `isShellCli`/`ShellWebShareBanner` imports. `SessionShareModal.tsx` already has its own complete, independent shell-warning gate (Phase 150 SET-01) that is unaffected and remains the sole enforcement point.
- **Files modified:** `frontend/src/App.tsx`, `frontend/src/components/__tests__/App.shellWebShare.test.tsx`
- **Verification:** `tsc --noEmit` clean; `App.shellWebShare.test.tsx` rewritten and passing; `SessionShareModal.test.tsx` (unchanged) still passes, confirming the modal's own gate is untouched.
- **Committed in:** `57b42e27`

**3. [Rule 1 - Bug] Fixed a latent bug where the modal's own shell-warn confirm click would silently no-op**
- **Found during:** Task 2, while rewriting `handleShellWebShareConfirm`
- **Issue:** `handleShellWebShareConfirm` (threaded into the modal via `onShellWebShareConfirm`) began with `if (!pendingShellWebToggle) return`, extracting `sessionId` from that App-level state. But `pendingShellWebToggle` was populated ONLY by the now-removed `handleToggleWeb` (the footer's own, separate interception) — the modal's inline shell-warn confirm banner uses its own local `pendingShellShare` flag and never touched `pendingShellWebToggle`. Confirming the modal's shell-warning banner for a shell CLI session would therefore have called `onShellWebShareConfirm`, hit the early return, and silently done nothing — `ToggleWebServing` never invoked, while the modal's own local `shareEnabled` state still flips to `true` (UI shows "on", server state never changed). This is pre-existing behavior (present before this plan too), but it was masked by never being exercised in this codebase's test suite (only mocked in isolation).
- **Fix:** `handleShellWebShareConfirm` now sources `sessionId` from `shareModalSession.id` (the state this plan already lifts to App.tsx and populates on every modal-open path), closing the gap for both the Hub-card-click and the footer-button-open paths.
- **Files modified:** `frontend/src/App.tsx`
- **Verification:** `App.shellWebShare.test.tsx#Phase 168-05: handleShellWebShareConfirm sources sessionId from shareModalSession (not the retired pendingShellWebToggle)`; `tsc --noEmit` clean.
- **Committed in:** `57b42e27`

**4. [Rule 2 - Missing Critical] Seeded `hubSessions` from the existing mount-time/retry-time `ListSessions()` call, not just the Hub-tab-gated 3s poll**
- **Found during:** Task 1, while verifying `openShareModalForActiveSession` against the app's actual startup flow
- **Issue:** `hubSessions` is normally populated only by a `useEffect` gated on `activeId === HUB_TAB.id` (T-131-10, intentionally prevents polling while Hub is inactive). `App.tsx`'s SESS-02 startup-restore path can land the user directly on a session tab (`setActiveId(restoredTabs[0].id)`) without the Hub tab ever becoming active in that app session. In that state, `hubSessions` stays `[]`, and `openShareModalForActiveSession`'s `hubSessions.find(...)` would silently find nothing — the footer button would appear to do nothing, reproducing the exact failure mode this plan exists to eliminate, just via stale data instead of an unmounted component.
- **Fix:** Added `setHubSessions(sessions)` in both `App.tsx`'s `init()` and `retryInit()`, reusing the `ListSessions()` result those functions already fetch for tab restoration — no new RPC call.
- **Files modified:** `frontend/src/App.tsx`
- **Verification:** `tsc --noEmit` clean; no new test added for this specific timing scenario (App.tsx is not fully mountable in this codebase's test convention — see coverage D3's rationale); documented as a discretionary fix consistent with the plan's stated success criteria.
- **Committed in:** `57b42e27`

---

**Total deviations:** 4 auto-fixed (1 architectural-discretion render-location choice explicitly granted by the plan text, 1 Rule 3 blocking dead-code cleanup, 1 Rule 1 latent-bug fix, 1 Rule 2 missing-critical-functionality fix). All four are direct, necessary consequences of implementing this plan's own explicit D-13/D-14/D-15 decisions and the RESEARCH document's own named risk (Pattern 4's "footer button silently does nothing outside the Hub tab"). No unrelated scope was touched.
**Impact on plan:** None negative — the plan's core deliverable (single lifted Share-modal instance, renamed footer button, D-14/D-15 satisfied) is unchanged; these are the specific implementation choices required to make that deliverable actually work end-to-end rather than only in the Hub-card-click case the pre-existing test suite happened to already cover.

## TDD Gate Compliance

Task 2 is `tdd="true"`. A strict RED→GREEN sequence was not captured as separate commits: `StatusBar.tsx`'s prop-contract change (`onToggleWeb`→`onShareSession`) and `App.tsx`'s render-site change (the same rename, plus the modal-render move from Task 1) are mutually dependent — neither compiles in isolation without the other once `StatusBar.shareSession.test.tsx`/`App.shellWebShare.test.tsx` are also updated, because `tsc --noEmit` (a wave-merge gate) fails on any intermediate state with a dangling prop reference or a fully orphaned handler. Both tasks' implementation and their test updates landed in a single commit (`57b42e27`) after the full local test-and-tsc loop confirmed GREEN. This mirrors 168-04's precedent for tightly-coupled, mirrored-pattern work (Task 1/Task 3 there also skipped a captured RED state for the same reason — low risk given every change was verified against the full running suite before commit, not merely asserted).

## Issues Encountered

None beyond the deviations documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- UX-02 (#115) is code-complete: the single lifted Share-modal instance, the renamed "Share Session" footer button, D-14 (no direct `ToggleWebServing` from the footer), and D-15 (hidden on non-shareable tabs, already-exhaustive filter confirmed) are all implemented and unit-proven. `tsc --noEmit` clean; full frontend suite (139 files / 2309 tests) passes.
- **Deferred (human judgment, coverage D3 above):** live end-to-end confirmation that clicking the footer "Share Session" button on a freshly-restored session tab (before ever visiting the Hub tab in that app session) opens the modal with correct, live `funnelActive`/`viewerCount`/`webEnabled` state for that specific session. App.tsx is not fully mountable in this codebase's established testing convention, so this is proven only by source-inspection of the wiring shape. Recommend folding into this phase's live-UAT pass alongside 168-04's own deferred D3.
- No manual-checklist (M-NN) item was added — this behavior is a pure React modal/state-lift with no native GUI, remote-peer, live-PTY, or physical-hardware dependency, so it is fully covered by the automated suite plus the recommended live-UAT fold-in, per the standing convention's carve-out.

---
*Phase: 168-bug-fix-settings-polish*
*Completed: 2026-07-01*

## Self-Check: PASSED

- FOUND: frontend/src/App.tsx
- FOUND: frontend/src/components/Hub/HubPanel.tsx
- FOUND: frontend/src/components/Hub/HubPanel.test.tsx
- FOUND: frontend/src/components/StatusBar.tsx
- FOUND: frontend/src/components/__tests__/StatusBar.shareSession.test.tsx
- FOUND: frontend/src/components/__tests__/App.shellWebShare.test.tsx
- FOUND: TESTING.md
- CONFIRMED DELETED: frontend/src/components/__tests__/StatusBar.test.tsx
- FOUND commit: 57b42e27
- FOUND commit: 6be58318
