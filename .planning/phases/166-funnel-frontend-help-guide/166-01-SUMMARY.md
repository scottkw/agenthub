---
phase: 166-funnel-frontend-help-guide
plan: "01"
subsystem: frontend-stubs-css
tags: [wails-stubs, typescript, css-tokens, funnel, tdd, colorblind-safe]
requires: []
provides: [SetSessionFunnel-stub, funnelActive-field, phase-166-css]
affects: [frontend/src/wailsjs/go/main/App.d.ts, frontend/src/wailsjs/go/main/App.js, frontend/src/style.css]
tech-stack:
  added: []
  patterns: [wails-hand-authored-stub-extension, raw-import-contract-test, css-bem-tokens]
key-files:
  created:
    - frontend/src/components/__tests__/funnelBinding.contract.test.tsx
  modified:
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/style.css
    - frontend/src/lib/remoteAdapter.ts
    - frontend/src/components/__tests__/App.open-remote.test.tsx
    - frontend/src/components/__tests__/SessionCard.share.test.tsx
    - frontend/src/components/Hub/HubBriefingModal.test.tsx
    - frontend/src/components/Hub/HubFilterBar.test.tsx
    - frontend/src/components/Hub/HubInteractiveModal.test.tsx
    - frontend/src/components/Hub/HubModal.test.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/components/Hub/SessionCard.test.tsx
    - frontend/src/components/Hub/SessionCardGrid.test.tsx
    - frontend/src/components/Hub/useChatUnreadListeners.test.tsx
    - frontend/src/lib/hubGroupCounts.test.ts
key-decisions:
  - "funnelActive required (not optional) in SessionInfo to match server NOT omitempty — 12 test fixtures updated with funnelActive: false"
  - "Contract test uses ?raw vitest import to guard stub identity against future wails dev regeneration"
  - "CSS transition for hub-funnel-risk-panel guarded by prefers-reduced-motion: no-preference / reduce pair"
requirements-completed: [FUI-01, FUI-03, FUI-04, FUI-05]
coverage:
  - deliverable: "SetSessionFunnel + funnelActive in hand-authored App.d.ts"
    verification:
      - kind: test
        ref: frontend/src/components/__tests__/funnelBinding.contract.test.tsx
        status: pass
    human_judgment: false
  - deliverable: "SetSessionFunnel Call wrapper in App.js"
    verification:
      - kind: test
        ref: frontend/src/components/__tests__/funnelBinding.contract.test.tsx#App.js registers main.App.SetSessionFunnel Call wrapper
        status: pass
    human_judgment: false
  - deliverable: "tsc --noEmit clean across all importers"
    verification:
      - kind: command
        ref: "cd frontend && npx tsc --noEmit"
        status: pass
    human_judgment: false
  - deliverable: "Phase-166 CSS tokens and component classes in style.css"
    verification:
      - kind: command
        ref: "grep -c 'COLORBLIND-SAFE' frontend/src/style.css returns 35 (was 33)"
        status: pass
      - kind: command
        ref: "grep '#43ddb2' frontend/src/style.css returns --hub-internet-badge-text"
        status: pass
      - kind: command
        ref: "grep '#0d7a5c' frontend/src/style.css returns --hub-internet-badge-text (light)"
        status: pass
    human_judgment: false
metrics:
  duration: "7 min"
  completed: "2026-07-01"
  tasks: 3
  files: 16
status: complete
---

# Phase 166 Plan 01: Wails Stubs + CSS Foundation Summary

Wave 0 foundation that unblocks all Wave 1-3 plans: adds `SetSessionFunnel` and `SessionInfo.funnelActive` to the hand-authored Wails stubs the app actually imports, pins the stub contract with a vitest `?raw` test, and adds all Phase-166 CSS tokens and component classes to `style.css`.

## Duration

7 minutes (2026-07-01T00:25:46Z to 2026-07-01T00:32:46Z)

## Tasks Completed

3 / 3

## Accomplishments

- **Stub extension (App.d.ts):** Added `funnelActive: boolean` to `SessionInfo` after `browseEnabled`, with JSDoc `/** Phase 165 / FNL-01: true when Tailscale Funnel is active. NOT omitempty — false must serialize. */`. Added `SetSessionFunnel(sessionID: string, enabled: boolean, expiresIn: number): Promise<void>` export with phase-tag comment matching the existing SetSessionBrowse style.
- **Stub extension (App.js):** Added `SetSessionFunnel = (sessionID, enabled, expiresIn) => Call('main.App.SetSessionFunnel', [sessionID, enabled, expiresIn])` matching the positional-arg arrow style.
- **Import-contract test:** Created `funnelBinding.contract.test.tsx` with 3 assertions using `?raw` vitest imports — pins `funnelActive: boolean` on the interface, `SetSessionFunnel` 3-arg signature, and the `main.App.SetSessionFunnel` Call registration. Guards RESEARCH Pitfalls 1 + 2. Green (3/3 pass).
- **CSS tokens (dark):** Added `--hub-internet-badge-bg: rgba(67, 221, 178, 0.15)` and `--hub-internet-badge-text: #43ddb2` with COLORBLIND-SAFE comment (globe shape + INTERNET text as state carriers, color reinforcement only).
- **CSS tokens (light):** Added `--hub-internet-badge-bg: rgba(13, 122, 92, 0.13)` and `--hub-internet-badge-text: #0d7a5c` with COLORBLIND-SAFE comment (~5.2:1 AA on white).
- **CSS component classes:** Added `.hub-internet-badge` + `__icon` + `__label`, `.tab__internet-icon`, `.hub-funnel-risk-panel` + `--open` + 7 sub-elements, `.hub-share-internet-section` + 5 sub-elements. All use existing `--hub-*` tokens; no new spacing steps introduced.
- **Motion guard:** `hub-funnel-risk-panel` transition (`max-height 200ms ease-out`) added exclusively inside `@media (prefers-reduced-motion: no-preference)`. Matching `reduce` block sets `transition: none`.

## Verification Results

| Check | Result |
|-------|--------|
| `cd frontend && npx tsc --noEmit` | CLEAN |
| `pnpm test -- funnelBinding.contract` | 3/3 PASS |
| `grep -c 'COLORBLIND-SAFE' src/style.css` | 35 (was 33, +2) |
| `grep '#43ddb2' src/style.css` | FOUND as `--hub-internet-badge-text` |
| `grep '#0d7a5c' src/style.css` | FOUND as `--hub-internet-badge-text` |
| All 9 required CSS classes present | PASS |
| risk-panel transition inside no-preference only | PASS |

## Commits

| Hash | Message |
|------|---------|
| `887975df` | feat(166-01): add SetSessionFunnel + funnelActive to hand-authored Wails stubs |
| `3e5a35f2` | test(166-01): add import-contract test for Funnel binding gap |
| `9a02f81e` | feat(166-01): add all Phase-166 CSS tokens + component classes to style.css |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed 12 test fixtures missing funnelActive field**
- **Found during:** Task 1 (tsc --noEmit after adding funnelActive: boolean to SessionInfo)
- **Issue:** 12 test files creating `SessionInfo` objects omitted `funnelActive`, causing TS2741 errors. Two types of error: (a) missing field entirely, (b) one file had `funnelActive?: boolean` in a helper that conflicted with the required `boolean` type.
- **Fix:** Added `funnelActive: false` to all 11 `makeSession()`/object-literal fixtures in test files and `remoteAdapter.ts`.
- **Files modified:** App.open-remote.test.tsx, SessionCard.share.test.tsx, HubBriefingModal.test.tsx, HubFilterBar.test.tsx, HubInteractiveModal.test.tsx, HubModal.test.tsx, HubPanel.test.tsx (×3 factories), SessionCard.test.tsx, SessionCardGrid.test.tsx, useChatUnreadListeners.test.tsx, hubGroupCounts.test.ts, remoteAdapter.ts
- **Rationale for required vs optional:** The Go struct field has no `omitempty` tag — `false` must serialize. Optional (`funnelActive?: boolean`) would allow undefined to slip through and trigger incorrect behavior. Required is correct.
- **Commit:** `887975df`

**Total deviations:** 1 auto-fixed (Rule 1 — test fixture update required by required field addition). **Impact:** All 12 existing test files compile and run correctly; no behavioral change to any test assertions.

## Threat Flags

None. No new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries introduced. CSS tokens and stub additions are compile-time artifacts only.

## Known Stubs

None. All artifacts are complete for their stated Wave 0 scope. Wave 1+ components will reference these CSS classes and stubs.

## Next Step

Ready for Wave 1 plans (166-02 through 166-03): SessionShareModal Funnel toggle + SessionCard badge + TabBar globe icon.

## Self-Check: PASSED

- [x] `frontend/src/wailsjs/go/main/App.d.ts` modified — funnelActive + SetSessionFunnel present
- [x] `frontend/src/wailsjs/go/main/App.js` modified — SetSessionFunnel Call wrapper present
- [x] `frontend/src/components/__tests__/funnelBinding.contract.test.tsx` created
- [x] `frontend/src/style.css` modified — tokens + classes present
- [x] Commits `887975df`, `3e5a35f2`, `9a02f81e` all exist in git log
- [x] `tsc --noEmit` clean
- [x] Contract test 3/3 green
