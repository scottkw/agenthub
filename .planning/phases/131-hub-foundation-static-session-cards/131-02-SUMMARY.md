---
phase: 131-hub-foundation-static-session-cards
plan: "02"
subsystem: frontend/Hub
tags: [react, tsx, hub, session-card, colorblind-safe, tdd, vitest]
dependency_graph:
  requires: [131-01]
  provides: [InlineSessionName, SessionCard]
  affects: [frontend/src/components/Hub/]
tech_stack:
  added: []
  patterns: [TDD red-green, BEM CSS classes, Heroicons, tab__rename-input reuse]
key_files:
  created:
    - frontend/src/components/Hub/InlineSessionName.tsx
    - frontend/src/components/Hub/InlineSessionName.test.tsx
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/Hub/SessionCard.test.tsx
  modified: []
decisions:
  - "tab__rename-input CSS class reused directly in InlineSessionName — no new rename-input CSS authored"
  - "agentBadgeModifier inlined in SessionCard to avoid cross-file import of TabBar internals"
  - "STATUS_CONFIG stopped-err label is 'Exited' (base); display label dynamically appends code"
  - "isLocal determined by empty hostname (empty string = local daemon session)"
metrics:
  duration: "~12 minutes"
  completed: "2026-06-16"
  tasks_completed: 2
  files_created: 4
  tests_added: 27
---

# Phase 131 Plan 02: InlineSessionName + SessionCard Summary

Inline-editable session name (reusing TabBar rename pattern) and colorblind-safe session card
(STATUS_CONFIG icon+label map, dimming, origin, viewer count, uptime/exit) — 27 vitest specs green.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | InlineSessionName component + test (TDD) | 61cab78 | InlineSessionName.tsx, InlineSessionName.test.tsx |
| 2 | SessionCard component + test (colorblind-safe, TDD) | 527287e | SessionCard.tsx, SessionCard.test.tsx |

## What Was Built

### InlineSessionName (Task 1)

- Renders a `<span className="hub-card__name">` in display mode; clicking enters edit mode
- Edit mode renders `<input className="tab__rename-input">` — reuses existing CSS, no new rename-input CSS
- Enter with changed non-empty value: calls `RenameSession(id, trimmed)`, fires `onRenamed`, exits edit
- Escape: restores original name, exits edit, no RPC call
- Blur with unchanged or empty-trimmed value: no RPC call (mirrors TabBar commitEdit)
- 5 vitest specs: display mode, enter-edit, Enter-commit, Escape-cancel, blur-unchanged

### SessionCard (Task 2)

- `deriveStatus()` helper: `'stopped-err'` when exitCode != 0, `'stopped-ok'` for clean exit, else status as-is
- `STATUS_CONFIG` map keyed by `HubStatus` with `{ Icon, label, spin }` for all 6 states
- Every status renders both icon (`aria-label={label}`) AND visible `.hub-card__status-label` span
- `hub-card__status-icon--spin` class applied only to `running` status (ArrowPathIcon)
- `hub-card--dim` class applied only to `stopped-ok` cards; `stopped-err` cards get full opacity
- Exit-code chip (`.hub-card__exit-chip`) rendered only for `stopped-err`
- Origin marker: empty hostname → ComputerDesktopIcon + "Local"; non-empty → GlobeAltIcon + hostname
- Viewer count: hidden when 0; singular "1 viewer", plural "N viewers"
- Uptime: `formatUptime(createdAt)` for running; `formatDuration(seconds)` prefixed "Ran" for stopped
- CLI badge: `tab__agent-badge tab__agent-badge--{modifier}` with CLI name as visible text
- Card `aria-label="{name}, {label}, {cli}, {origin}"`, `tabIndex={0}`
- 22 vitest specs covering all statuses, dimming, viewers, origin, CLI badge, time, aria-label

## Colorblind-Safety Verification

All 12 required COLORBLIND-SAFE inline comments are present in SessionCard.tsx (verified by grep -c = 14).
Each status has unique Heroicon shape + visible text label — color is reinforcement only.

## Threat Mitigations Applied

- **T-131-03 (XSS):** All session name, hostname, and cli values rendered via JSX text children and
  `aria-label` string interpolation only — no `dangerouslySetInnerHTML` in either component.
  React escapes text by default.

## Deviations from Plan

None — plan executed exactly as written.

- `agentBadgeModifier` was inlined in SessionCard.tsx rather than imported from TabBar.tsx.
  TabBar exports it as a local function, not a named export. Inlining mirrors the pattern exactly
  and avoids coupling the Hub to TabBar internals. This is consistent with the PATTERNS.md guidance
  which shows the function inline.

## Known Stubs

None — both components are fully wired to real data via props.

## Threat Flags

None — no new network endpoints, auth paths, or schema changes introduced.

## Self-Check: PASSED

- [x] `frontend/src/components/Hub/InlineSessionName.tsx` exists
- [x] `frontend/src/components/Hub/InlineSessionName.test.tsx` exists
- [x] `frontend/src/components/Hub/SessionCard.tsx` exists
- [x] `frontend/src/components/Hub/SessionCard.test.tsx` exists
- [x] Commit 61cab78 exists (InlineSessionName)
- [x] Commit 527287e exists (SessionCard)
- [x] 27 vitest tests pass (5 InlineSessionName + 22 SessionCard)
- [x] `grep -c 'tab__rename-input' InlineSessionName.tsx` = 2 >= 1
- [x] `grep -c 'RenameSession(' InlineSessionName.tsx` = 1 >= 1
- [x] `grep -c 'STATUS_CONFIG' SessionCard.tsx` = 4 >= 1
- [x] `grep -c 'COLORBLIND-SAFE' SessionCard.tsx` = 14 >= 1
- [x] No `dangerouslySetInnerHTML` in either component
- [x] SessionCard.tsx >= 80 lines (243 lines)
