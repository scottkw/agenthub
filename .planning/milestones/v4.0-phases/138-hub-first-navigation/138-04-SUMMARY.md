---
phase: 138-hub-first-navigation
plan: "04"
subsystem: frontend-navigation
tags: [nav-cleanup, sidebar, app-routing, panel-deletion, green-gate]
dependency_graph:
  requires: [138-01, 138-02, 138-03]
  provides: [NAV-02, NAV-03, NAV-04, NAV-05]
  affects: [frontend/src/components/Sidebar.tsx, frontend/src/App.tsx, frontend/src/components/TabBar.tsx]
tech_stack:
  added: []
  patterns:
    - Sidebar collapsed to 3-item nav (Home/Hub/Settings)
    - Remote poll guard narrowed to HUB_TAB.id only (T-138-08 preservation)
    - HubPanel wired with onKill/onOpenInBrowser/onBrowseFiles/remotePeers parity props
key_files:
  created: []
  modified:
    - frontend/src/components/Sidebar.tsx
    - frontend/src/App.tsx
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/__tests__/App.remoteFileBrowser.test.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
  deleted:
    - frontend/src/components/DaemonManagerPanel.tsx
    - frontend/src/components/RemoteSessionsPanel.tsx
    - frontend/src/components/__tests__/RemoteSessionsPanel.test.tsx
    - frontend/src/components/__tests__/DaemonManagerPanel.test.tsx
    - frontend/src/components/__tests__/App.wiring.test.tsx
decisions:
  - Removed orphan DaemonManagerPanel.test.tsx and App.wiring.test.tsx in addition to the
    plan-specified RemoteSessionsPanel.test.tsx — all tested deleted components (Rule 1 auto-fix)
  - Used hubSessions (already polled by Hub poll) to replace panelSessions in FileBrowserTab
    local-path resolution — semantically equivalent, avoids dead state
  - Removed remoteLoading/setRemoteLoading and remoteError/setRemoteError state vars —
    only consumers were the deleted panels; Hub error handling already covered by hubError
metrics:
  duration: ~20 minutes
  completed: "2026-06-20"
  tasks_completed: 3
  files_modified: 5
  files_deleted: 5
---

# Phase 138 Plan 04: Deletion + Routing Cleanup (GREEN GATE) Summary

**One-liner:** Hub-first navigation complete — sidebar collapsed to 3 items, Sessions/Remote pages deleted, dead routing/polls/Tab-type members removed, HubPanel wired with Kill/Open/Browse parity props, full 1704-test suite green.

## Objective Achieved

This plan executed the FINAL wave of Phase 138: deleted DaemonManagerPanel and RemoteSessionsPanel,
collapsed the sidebar to Home/Hub/Settings, cleaned all dead routing/consts/handlers from App.tsx,
trimmed the TabBar Tab.type union, and wired the HubPanel with the Kill/Open/Browse parity handlers
that Plans 02 and 03 migrated to the Hub. After this plan, NAV-02..05 are all true and the full
vitest suite passes with 0 failures.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Collapse sidebar to Home/Hub/Settings | fa5bbde3 | Sidebar.tsx |
| 2 | Clean App.tsx routing + wire HubPanel; trim TabBar | 4c75c057 | App.tsx, TabBar.tsx |
| 3 | Delete panels + orphan tests; final suite gate | 9bba2e3e | 5 deleted, 2 modified |

## Phase Gate Results

- **`npx tsc --noEmit`**: exits 0 (0 errors) — clean
- **`npx vitest run`**: **1704 / 1704 tests pass** across 104 test files (0 failures)
- **No dangling imports**: `grep -rln "components/RemoteSessionsPanel|components/DaemonManagerPanel" src/` returns nothing

## Success Criteria Verification

| Criterion | Status |
|-----------|--------|
| Sidebar renders exactly Home/Hub/Settings | PASS — 3 sidebar__item buttons; Sidebar.test.tsx 26/26 |
| No Sessions/Remote/New Session buttons | PASS — App.hub.test.tsx, Sidebar.test.tsx |
| App.tsx has no DAEMON_MANAGER_TAB/REMOTE_SESSIONS_TAB | PASS — grep -c returns 0 |
| No DaemonManagerPanel/RemoteSessionsPanel imports | PASS — grep -c returns 0 |
| Remote poll preserved (guard = HUB_TAB.id only) | PASS — two guards exist, both on HUB_TAB.id |
| HubPanel wired: onKill/onOpenInBrowser/onBrowseFiles/remotePeers | PASS — App.tsx lines 1313-1316 |
| Tab.type union omits daemon-manager/remote-sessions | PASS — TabBar.tsx updated |
| No dangling imports to deleted panels | PASS — grep returns nothing |
| tsc exits 0 | PASS |
| Full vitest suite 0 failures | PASS — 1704/1704 |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Additional orphan test files deleted beyond plan specification**

- **Found during:** Task 3 safety grep (`grep -rln "components/RemoteSessionsPanel|components/DaemonManagerPanel" src/`)
- **Issue:** The plan specified deleting only RemoteSessionsPanel.test.tsx. The grep revealed 2 additional orphan test files:
  - `DaemonManagerPanel.test.tsx`: directly imported the deleted DaemonManagerPanel.tsx (would have caused module-not-found failures)
  - `App.wiring.test.tsx`: source-inspection tests asserting `REMOTE_SESSIONS_TAB`, `<RemoteSessionsPanel`, and `onOpenRemoteSessions` in App.tsx (all removed — would have failed)
- **Fix:** `git rm` both additional files alongside the plan-specified deletions
- **Files deleted:** DaemonManagerPanel.test.tsx, App.wiring.test.tsx
- **Commit:** 9bba2e3e

**2. [Rule 1 - Bug] Stale test assertions in non-deleted test files**

- **Found during:** Task 3 full suite run
- **Issue:** After deletions, 2 tests in non-deleted files failed:
  - `App.remoteFileBrowser.test.tsx` line 122: asserted `<RemoteSessionsPanel` exists in App.tsx
  - `HubPanel.test.tsx` line 349: asserted `.hub__header` exists (removed in Plan 03)
- **Fix:** Removed the stale assertions and added explanatory comments pointing to where the behavior is now covered
- **Files modified:** App.remoteFileBrowser.test.tsx, HubPanel.test.tsx
- **Commit:** 9bba2e3e

**3. [Rule 1 - Bug] Dead state variables removed (remoteLoading, remoteError, panelSessions)**

- **Found during:** Task 2 — tsc reported TS6133 unused variable errors
- **Issue:** Removing the panel state setters left 3 state variables unreferenced:
  - `panelSessions`/`setPanelSessions`: was fed by the deleted DaemonManager poll
  - `remoteLoading`/`setRemoteLoading`: only consumed by deleted RemoteSessionsPanel
  - `remoteError`/`setRemoteError`: only consumed by deleted RemoteSessionsPanel
- **Fix:** Removed the 3 state declarations; replaced `panelSessions` with `hubSessions` in the FileBrowserTab local-path resolution (semantically equivalent)
- **Commit:** 4c75c057

## Known Stubs

None — plan executed against already-migrated functionality (Plans 02/03 migrated all parity affordances).

## Threat Flags

None — this plan is pure deletion with no new network endpoints, auth paths, or trust boundaries introduced.

## Phase 138 Readiness for `/gsd:verify-work`

This plan completes Phase 138. The phase is ready for live UAT. The following must be verified:

1. **3-item sidebar** — only Home, Hub, Settings visible (no Sessions/Remote/New Session)
2. **Sole New Session entry** — only HubFilterBar button creates sessions (Hub tab must be active)
3. **Hub remote cards still populate** — remote poll survived (guard: `HUB_TAB.id` only)
4. **Kill/Open/Browse affordances on Hub cards** — overflow menu per Plan 02/03 delivery
5. **Colorblind-safe indicators** — verify at source (hex constants) not by eye per MEMORY.md user_colorblind

## Self-Check: PASSED

- Sidebar.tsx: `[ -f frontend/src/components/Sidebar.tsx ]` — FOUND
- App.tsx: no DAEMON_MANAGER_TAB/REMOTE_SESSIONS_TAB (grep -c = 0) — VERIFIED
- Commits: fa5bbde3, 4c75c057, 9bba2e3e — all exist in git log
- tsc --noEmit: 0 errors — VERIFIED
- vitest run: 1704/1704 — VERIFIED
