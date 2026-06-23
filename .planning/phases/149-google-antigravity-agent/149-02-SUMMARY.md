---
phase: 149-google-antigravity-agent
plan: "02"
subsystem: frontend-badge-color
tags: [badge, css, colorblind-safe, wcag, agent-identity, agy]
dependency_graph:
  requires: []
  provides: [agy-badge-color-identity]
  affects: [frontend/src/lib/agentBadge.ts, frontend/src/style.css]
tech_stack:
  added: []
  patterns: [TDD-red-green, source-level-wcag-documentation, BEM-modifier-lockstep]
key_files:
  created: []
  modified:
    - frontend/src/lib/agentBadge.ts
    - frontend/src/lib/agentBadge.test.ts
    - frontend/src/style.css
    - frontend/src/components/__tests__/style.hub.test.ts
decisions:
  - "agy key (not antigravity): BEM modifier === data-agent key === knownCLIs key === binary name (D-09)"
  - "Color #ff9e64 locked (D-06): TokyoNight orange, only remaining main accent not in use"
  - "WCAG honest (D-07): dark 8.72:1 AA PASS, light 2.03:1 FAIL — same gap as all 7 existing agents; documented at source, no false AA blanket claim"
  - "All three color sites updated in one pass (D-08 lockstep): tab dot + card spine + card chip"
  - "Chip test regex approach: accounts for CSS whitespace alignment padding without hardcoding space counts"
metrics:
  duration: "5 minutes"
  completed: "2026-06-23T04:09:00Z"
  tasks_completed: 3
  files_modified: 4
---

# Phase 149 Plan 02: agy Badge Color Identity Summary

agy (Google Antigravity agent) given a distinct #ff9e64 badge color across all three per-agent CSS color sites (tab dot, card spine, card chip) with honest WCAG documentation and TDD source-gate tests.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add agy case to agentBadge.ts switch + test | 23e150ef | agentBadge.ts, agentBadge.test.ts |
| 2 | Add agy color to all three CSS sites in lockstep + WCAG comment + source-gate tests | 502e17d2 | style.css, style.hub.test.ts |
| 3 | Wave-merge frontend gate — tsc + full vitest | (verification only) | — |

## What Was Built

**agentBadge.ts switch (D-09):** Added `case 'agy':` to the existing fall-through group alongside `aider`/`cursor`. The returned modifier key is `agy` — matching the binary name, the knownCLIs key, and the CSS `data-agent` attribute. Using `antigravity` would never resolve to a CSS rule.

**style.css — three color sites in lockstep (D-06, D-08):**
- **Site 1 (tab dot, line ~1719):** `.tab__agent-badge--agy { background: #ff9e64; }` with WCAG-honest comment
- **Site 2 (card spine, line ~4813):** `.hub-card[data-agent="agy"] { border-left: 3px solid #ff9e64; }`
- **Site 3 (card chip, line ~5030):** `.hub-card[data-agent="agy"] .hub-card__badge { color: #ff9e64; border-color: rgba(255, 158, 100, 0.45); }`

**WCAG honest documentation (D-07):** The tab-dot rule carries:
```
/* agy — orange; dark: 8.72:1 AA PASS; light: 2.03:1 FAIL (same gap as all existing agents — text chip carries identity) */
```
This documents the real numbers. It does NOT claim a blanket AA pass. The light-mode gap is the same as all 7 existing agent badges — the text chip carries the identity cue, making color reinforcement-only.

**Source-gate tests:** 7 new assertions in style.hub.test.ts covering all three sites, the WCAG numbers, and the D-09 anti-antigravity guard. The GAP-04 "all agent spine rules" test updated from 8 to 9 agents.

## Verification

- `agentBadgeModifier('agy') === 'agy'` — verified by agentBadge.test.ts (14/14 pass)
- `grep -c "case 'antigravity'" frontend/src/lib/agentBadge.ts` → 0
- `grep -c 'data-agent="antigravity"' frontend/src/style.css` → 0
- `cd frontend && pnpm vitest run style.hub` → 100/100 pass
- `cd frontend && pnpm vitest run` → 1878/1878 pass (no regression)
- `cd frontend && pnpm exec tsc --noEmit` → exits 0 (type-check clean)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Chip test used single-space selector but CSS uses alignment padding**
- **Found during:** Task 2 (GREEN phase)
- **Issue:** Tests originally searched `indexOf('.hub-card[data-agent="agy"] .hub-card__badge')` (single space), but CSS alignment follows the existing pattern of multiple spaces. The `indexOf` returned -1.
- **Fix:** Replaced both chip assertions with a regex match (`/\.hub-card\[data-agent="agy"\]\s+\.hub-card__badge\s*\{([^}]+)\}/`) that tolerates any whitespace between selector parts — consistent with existing claude chip test which uses `indexOf` only because that test has the exact padded string.
- **Files modified:** frontend/src/components/__tests__/style.hub.test.ts
- **Commit:** 502e17d2

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes. This plan is pure frontend presentation (CSS + TS switch case). The `T-149-02F` colorblind information-disclosure threat is accepted per threat model — color is reinforcement only, agent identity also shown as `{cli}` text chip.

## Known Stubs

None. All three color sites are wired. The agy modifier key resolves to CSS rules that will render immediately when a session with `cli: "agy"` appears in the Hub.

## Self-Check: PASSED

- [x] `frontend/src/lib/agentBadge.ts` exists and contains `case 'agy'`
- [x] `frontend/src/lib/agentBadge.test.ts` exists and contains agy test
- [x] `frontend/src/style.css` contains `.tab__agent-badge--agy`, `.hub-card[data-agent="agy"]` spine, and chip rules
- [x] `frontend/src/components/__tests__/style.hub.test.ts` contains 7 new agy assertions
- [x] Commit 23e150ef exists (Task 1)
- [x] Commit 502e17d2 exists (Task 2)
- [x] Full vitest: 1878/1878 pass
- [x] tsc --noEmit: clean
