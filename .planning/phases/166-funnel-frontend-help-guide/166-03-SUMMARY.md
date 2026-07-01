---
phase: 166-funnel-frontend-help-guide
plan: "03"
subsystem: frontend
tags: [funnel, hub-card, tab-bar, internet-badge, colorblind-safe, fui-03]
status: complete

dependency_graph:
  requires: [166-01]
  provides: [hub-card-internet-badge, tab-internet-icon, funnelActiveSessions-prop]
  affects: [frontend/src/components/Hub/SessionCard.tsx, frontend/src/components/TabBar.tsx, frontend/src/App.tsx]

tech_stack:
  added: []
  patterns: [TDD RED-GREEN, colorblind-safe shape+text state carrier, IIFE-free JSX prop derivation]

key_files:
  created: []
  modified:
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/Hub/SessionCard.test.tsx
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/__tests__/TabBar.test.tsx
    - frontend/src/App.tsx

decisions:
  - "Wrapped GlobeAltIcon in span.tab__internet-icon to carry title/aria-label as HTML attributes — heroicons v2 renders title prop as SVG <title> child element, not as a DOM attribute, so getAttribute('title') would fail on the raw SVG"
  - "funnelActiveSessions derived inline in JSX via reduce on hubSessions (no IIFE, no useMemo) — computation is O(n) and n is bounded by active sessions; rides the existing 3s poll"

metrics:
  duration: "~7 minutes"
  completed: "2026-06-30"
  tasks_completed: 3
  files_modified: 5
---

# Phase 166 Plan 03: Internet Exposure Indicators Summary

Persistent, colorblind-safe internet-exposure indicator added to the Hub card (globe + "INTERNET" badge) and the session tab (globe icon with "Internet exposed" aria-label/title). Both are driven by `SessionInfo.funnelActive` via the existing 3s daemon poll.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | SessionCard badge failing tests | c554bc1a | SessionCard.test.tsx |
| 1 (GREEN) | SessionCard internet badge implementation | 40a5b8ef | SessionCard.tsx |
| 2 (RED) | TabBar icon failing tests | d3f2c11a | TabBar.test.tsx |
| 2 (GREEN) | TabBar funnelActiveSessions prop + icon | a772bdb9 | TabBar.tsx |
| 3 | App.tsx derivation + prop pass | c6e1b544 | App.tsx |

## Artifacts Produced

**SessionCard.tsx** — `.hub-internet-badge` conditional render:
- `GlobeAltIcon.hub-internet-badge__icon` (aria-hidden, shape carrier)
- `span.hub-internet-badge__label` with text "INTERNET" (text carrier)
- COLORBLIND-SAFE comment with dark hex #43ddb2 / light hex #0d7a5c

**TabBar.tsx** — new `funnelActiveSessions?: Record<string, boolean>` prop + icon:
- `span.tab__internet-icon` with `aria-label="Internet exposed"` and `title="Internet exposed"`
- `GlobeAltIcon` inside span (aria-hidden; label on wrapper per D-09)
- COLORBLIND-SAFE comment

**App.tsx** — `funnelActiveSessions` derived inline:
- `hubSessions.reduce<Record<string,boolean>>((acc,s) => ({...acc,[s.id]:s.funnelActive}),{})`
- Passed to `<TabBar funnelActiveSessions={...} />`

## Verification Results

- `pnpm test -- SessionCard TabBar App`: **443/443 passing**
- `npx tsc --noEmit`: **clean (zero errors)**
- `grep -c 'COLORBLIND-SAFE' SessionCard.tsx` → 21 (increased by 1)
- `grep -c 'COLORBLIND-SAFE' TabBar.tsx` → 1 (new)
- `grep -c 'hub-internet-badge' SessionCard.tsx` → 3
- `grep -c 'tab__internet-icon' TabBar.tsx` → 1

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] GlobeAltIcon title prop renders as SVG child, not HTML attribute**
- **Found during:** Task 2 GREEN — `icon.getAttribute('title')` returned null
- **Issue:** `@heroicons/react` v2 renders the `title` prop as a `<title>` SVG child element (for screen-reader accessibility), not as the HTML `title` attribute on the `<svg>` root. `getAttribute('title')` on the SVG element returns null.
- **Fix:** Wrapped `GlobeAltIcon` in `span.tab__internet-icon` and moved `aria-label` and `title` to the `<span>` wrapper. The icon inside is `aria-hidden="true"`. This matches the plan's D-09 intent (label in aria/title, not visual chrome) and is the correct a11y pattern for an SVG icon with an accessible label.
- **Files modified:** `frontend/src/components/TabBar.tsx`, `frontend/src/components/__tests__/TabBar.test.tsx` (test updated to check `.tab__internet-icon` element, not the SVG directly — no behavior change to acceptance criteria)
- **Commit:** a772bdb9

## Threat Surface Scan

No new network endpoints, auth paths, or trust boundary crossings introduced. All additions are pure read-only UI rendering of `funnelActive` state from the daemon poll — matching T-166-06/T-166-07 mitigations.

| Flag | File | Description |
|------|------|-------------|
| (none) | — | No new threat surface |

## Known Stubs

None. Badge and icon are fully wired to `session.funnelActive` / `funnelActiveSessions` derived from the 3s poll.

## Self-Check: PASSED

Files exist:
- FOUND: frontend/src/components/Hub/SessionCard.tsx
- FOUND: frontend/src/components/Hub/SessionCard.test.tsx
- FOUND: frontend/src/components/TabBar.tsx
- FOUND: frontend/src/components/__tests__/TabBar.test.tsx
- FOUND: frontend/src/App.tsx

Commits exist (verified via git log):
- FOUND: c554bc1a (test RED SessionCard)
- FOUND: 40a5b8ef (feat GREEN SessionCard)
- FOUND: d3f2c11a (test RED TabBar)
- FOUND: a772bdb9 (feat GREEN TabBar)
- FOUND: c6e1b544 (feat App.tsx)
