---
phase: 138-hub-first-navigation
plan: 03
subsystem: frontend-hub
tags: [wave-2, card-01, card-04, hub-header-removal, peer-hint, overflow-menu, tdd, colorblind-safe]
dependency_graph:
  requires:
    - "138-02: Wave 1 prop threading, KillConfirmItem, CARD-02/03/04 tests GREEN"
  provides:
    - "Plan 04 green target: App.hub.test.tsx hub__header-absence assertion GREEN"
    - "CARD-01: .hub__header removed from HubPanel; HubFilterBar is sole New Session entry"
    - "Per-peer unreachable/empty hint rendered from remotePeers below HubFilterBar"
    - ".hub__peer-hint CSS added to style.css"
    - ".hub-card__menu-item-sub CSS added (KillConfirmItem subtext)"
    - "Dead .hub__header/.hub__title/.hub__new-session-btn CSS commented out in style.css"
    - "remotePeers destructured into HubPanel function signature (was declared but missing)"
  affects:
    - "frontend/src/components/Hub/HubPanel.tsx"
    - "frontend/src/style.css"
tech_stack:
  added: []
  patterns:
    - "Per-peer hint: filter !reachable || sessions.length===0 → hub__peer-hint <p> per peer"
    - "Header removal: CARD-01 HubFilterBar as sole creation entry point"
key_files:
  created: []
  modified:
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/style.css
key_decisions:
  - "Task 1 was already implemented by Plan 02 Wave 1 executor (early implementation); verified green (85/85 tests pass), only gap was missing .hub-card__menu-item-sub CSS class — added"
  - "remotePeers prop was declared in HubPanelProps but missing from function destructuring — fixed as Rule 3 blocking issue before tsc would fail"
  - "hub__header CSS commented out (not deleted) per plan spec ('commented-out is acceptable')"
patterns-established:
  - "Per-peer hint pattern: (remotePeers ?? []).filter(!p.reachable || p.sessions.length===0).map(<p className=hub__peer-hint>)"
requirements-completed: [CARD-01, CARD-04]
duration: ~10min
completed: 2026-06-20
---

# Phase 138 Plan 03: Card Affordances + Header Removal (Wave 2) Summary

**CARD-01 hub__header removed from HubPanel (HubFilterBar is sole New Session entry); per-peer unreachable/empty hint added; CARD-04 overflow menu verified GREEN (16/16 SessionCard.share tests, 69/69 style.hub tests).**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-06-20T15:35:00Z
- **Completed:** 2026-06-20T15:46:00Z
- **Tasks:** 2 (Task 1 verify-only, Task 2 implement)
- **Files modified:** 2

## Accomplishments

- Verified Task 1 (Kill/Open-browser/Browse-files overflow menu + KillConfirmItem) fully satisfies all CARD-04 acceptance criteria from Plan 02's early implementation; 85 tests GREEN
- Added missing `.hub-card__menu-item-sub` CSS class (used by KillConfirmItem subtext "This will stop the session", was absent from style.css)
- Removed the entire `.hub__header` block (Hub title + duplicate New session button) from HubPanel.tsx; grep count = 0
- Destructured `remotePeers` in HubPanel function signature (it was declared in HubPanelProps but not destructured — tsc error)
- Added per-peer unreachable/empty hint rendering below HubFilterBar using `remotePeers` prop
- Added `.hub__peer-hint` CSS (12px, var(--hub-text-muted)); commented out dead `.hub__header`/`.hub__title`/`.hub__new-session-btn` CSS rules
- App.hub.test.tsx `HubPanel source does NOT contain hub__header` assertion GREEN; tsc error count = 1 (pre-existing Sidebar.test.tsx)

## Task Commits

1. **Task 1: Add Kill / Open-in-browser / Browse-files items (VERIFY ONLY + CSS gap patch)** - `ecbae907` (from Plan 02) + `a54cba3c` (adds .hub-card__menu-item-sub CSS, included in Task 2 commit)
2. **Task 2: Remove .hub__header + render per-peer hint** - `a54cba3c`

## Files Created/Modified

- `frontend/src/components/Hub/HubPanel.tsx` - Removed hub__header block, destructured remotePeers, added peer-hint JSX below HubFilterBar
- `frontend/src/style.css` - Added .hub__peer-hint and .hub-card__menu-item-sub; commented out dead .hub__header/.hub__title/.hub__new-session-btn rules

## Decisions Made

- **Task 1 verify-only:** Plan 02 already implemented the overflow menu items. The only unmet acceptance criterion was `.hub-card__menu-item-sub` CSS (used in KillConfirmItem but absent from style.css). Added it in the Task 2 commit since no separate Task 1 file changed.
- **remotePeers destructuring:** The prop was declared in HubPanelProps (Plan 02 added it) but the function destructuring didn't include it. Fixed as a Rule 3 blocking issue — without it, tsc would add a new error breaking the 1-error requirement.
- **CSS commenting vs deletion:** Per plan spec ("commented-out is acceptable; an active selector is not"), dead CSS rules are commented out. Clean deletion can happen in a future pass.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing remotePeers in HubPanel function destructuring**
- **Found during:** Task 2 (after adding peer-hint JSX, tsc showed TS2304 "Cannot find name 'remotePeers'")
- **Issue:** `remotePeers` was declared in `HubPanelProps` (Plan 02) but the component function's destructuring parameter didn't include it, causing tsc errors TS2304 and TS7006
- **Fix:** Added `remotePeers,` to the function destructuring parameter list
- **Files modified:** `frontend/src/components/Hub/HubPanel.tsx`
- **Verification:** `npx tsc --noEmit` now shows only 1 error (pre-existing Sidebar.test.tsx)
- **Committed in:** a54cba3c

**2. [Rule 2 - Missing Critical] .hub-card__menu-item-sub CSS absent from style.css**
- **Found during:** Task 1 verification (acceptance criterion: "style.css contains .hub-card__menu-item-sub")
- **Issue:** The class is used in KillConfirmItem's "Confirm kill" state subtext but was never added to style.css in Plan 02
- **Fix:** Added `.hub-card__menu-item-sub { display: block; font-size: 12px; color: var(--hub-text-muted); margin-top: 2px }` after the `.hub-card__menu-item--destructive` block
- **Files modified:** `frontend/src/style.css`
- **Verification:** All 85 SessionCard.share + style.hub tests still GREEN; no regressions
- **Committed in:** a54cba3c

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** Both fixes necessary for correctness. No scope creep.

## Test Results

### GREEN (target of this plan)

- `src/components/__tests__/SessionCard.share.test.tsx` — 16/16 tests PASS (CARD-04 affordances)
- `src/components/__tests__/style.hub.test.ts` — 69/69 tests PASS (CARD-03/04 CSS assertions)
- `src/components/__tests__/App.hub.test.tsx` — `HubPanel source does NOT contain hub__header` GREEN

### Remaining RED (expected — Plan 04 targets)

Per WAVE_CONTEXT, exactly 5 App.hub.test.tsx and 12 App.nav.test.tsx/Sidebar.test.tsx reds remain — all require editing App.tsx/Sidebar.tsx which are out of scope for this plan:

- `App.hub.test.tsx`: not.toContain DAEMON_MANAGER_TAB/REMOTE_SESSIONS_TAB/DaemonManagerPanel/RemoteSessionsPanel, and toContain onOpenInBrowser= → **Plan 04**
- `App.nav.test.tsx`: onOpenRemoteSessions/onOpenDaemonManager/handleAddTab removal + routing → **Plan 04**
- `Sidebar.test.tsx`: 3-item assertions + renderSidebar prop removal → **Plan 04**

`npx tsc --noEmit` error count: **1** (the pre-existing Sidebar.test.tsx TS2322 error from Plan 01 Wave 0; Plan 04 narrows SidebarProps)

## Known Stubs

- `(session as SessionInfo & { url?: string }).url ?? ''` in SessionCard.tsx "Open in browser" onClick: URL will be empty until App.tsx is updated in Plan 04. Carried forward from Plan 02.
- `.hub-card__menu-item-sub` CSS: functional but minimal styling — no media query override for light mode. Acceptable since it uses `var(--hub-text-muted)` which is already light-mode-aware via CSS custom property.

## Colorblind Source Verification

- `.hub__peer-hint` color: `var(--hub-text-muted)` — no raw hex; text content ("is unreachable", "has no shared sessions") is the primary signal
- `.hub-card__menu-item-sub` color: `var(--hub-text-muted)` — no raw hex; subtext label carries the meaning
- All existing colorblind-safe patterns (KillConfirmItem "Kill session" text, CARD-03 chip icons) unchanged

## Next Phase Readiness

Plan 04 can now safely:
- Delete `DaemonManagerPanel.tsx` and `RemoteSessionsPanel.tsx` (types in lib/remoteSession.ts, no lib file imports from panels)
- Edit `Sidebar.tsx` to collapse to 3 items (Home/Hub/Settings)
- Edit `App.tsx` to remove DAEMON_MANAGER_TAB/REMOTE_SESSIONS_TAB constants and wire onKill/onOpenInBrowser/onBrowseFiles to HubPanel

Blocker: none. All card affordances confirmed GREEN before panel deletion proceeds.

## Threat Flags

None — this plan removes UI surface (no new network endpoints, auth paths, or schema changes). The hub__peer-hint renders hostname as a React text child (auto-escaped, T-138-06 accepted disposition).

## Self-Check: PASSED

- `grep -c "hub__header" frontend/src/components/Hub/HubPanel.tsx` → 0
- `grep -c "hub__peer-hint" frontend/src/components/Hub/HubPanel.tsx` → 1
- `grep -c "hub__peer-hint" frontend/src/style.css` → 1
- `grep -c "hub-card__menu-item-sub" frontend/src/style.css` → 1
- Commit a54cba3c exists in git log
- `npx tsc --noEmit` → 1 error (pre-existing Sidebar.test.tsx only)
- `npx vitest run src/components/__tests__/SessionCard.share.test.tsx src/components/__tests__/style.hub.test.ts` → 85/85 GREEN

---
*Phase: 138-hub-first-navigation*
*Completed: 2026-06-20*
