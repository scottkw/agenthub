---
phase: 133-attention-pulse
plan: 02
subsystem: frontend/style.css
tags: [css, attention, pulse, a11y, colorblind-safe, reduced-motion]
dependency_graph:
  requires: []
  provides: [hub-attn-tokens, hub-card--attention, hub-attn-pulse, hub-attn-badge-css, hub-attn-icon-css]
  affects: [frontend/src/style.css]
tech_stack:
  added: []
  patterns: [BEM-modifier, CSS-custom-properties, prefers-reduced-motion-guard, @keyframes-outside-media-query]
key_files:
  created: []
  modified:
    - frontend/src/style.css
decisions:
  - keyframe-outside-media-query: @keyframes hub-attn-pulse declared at root scope (not inside @media); the animation: property reference is gated inside the no-preference guard — mirrors the existing hub-spin pattern
  - hover-override-inside-guard: .hub-card--attention:hover placed inside @media no-preference block so reduced-motion users also keep the override without animation side effects
  - 400ms-transition-in-guard: .hub-card transition override (400ms) placed inside no-preference guard so reduced-motion users retain the base 100ms instant-feel behavior
  - dark-equals-static: --hub-attn-static-border #e0af68 equals --hub-attn-border #e0af68 in dark theme — intentional (same hex, no animation is the only difference under reduce)
metrics:
  duration: ~8 minutes
  completed: 2026-06-17
  tasks_completed: 3
  tasks_total: 3
  files_changed: 1
---

# Phase 133 Plan 02: CSS Attention Tokens, Pulse Animation, and Badge Sizing Summary

CSS-only groundwork layer: --hub-attn-* tokens (dark + light), .hub-card--attention BEM modifier with prefers-reduced-motion-gated pulse keyframe, static reduced-motion fallback, hover override, 400ms clear transition, .hub-card__attn-icon wrapper + svg sizing, and .hub__group-sidebar-item__attn-badge + svg sizing — all amber-gold (#e0af68 / #b45309) with COLORBLIND-SAFE source comments.

## Tasks Completed

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 | Add --hub-attn-* tokens to dark + light theme blocks | `440df5d8` | frontend/src/style.css |
| 2 | Add .hub-card--attention modifier, hub-attn-pulse keyframe, reduced-motion guards, hover override, clear transition | `d0700126` | frontend/src/style.css |
| 3 | Add .hub-card__attn-icon wrapper + .hub__group-sidebar-item__attn-badge with explicit CSS sizing | `af91acfc` | frontend/src/style.css |

## Verification Results

- `grep -c -- '--hub-attn-border:' frontend/src/style.css` → **2** (one dark, one light)
- `animation: hub-attn-pulse` only inside `@media (prefers-reduced-motion: no-preference)` — confirmed
- `@keyframes hub-attn-pulse` declared outside any media query — confirmed
- `.hub-card--attention:hover { border-color: var(--hub-attn-border) }` present — confirmed
- `.hub-card__attn-icon svg { width: 16px; height: 16px }` — explicit CSS, no Tailwind
- `.hub__group-sidebar-item__attn-badge svg { width: 12px; height: 12px }` — explicit CSS, no Tailwind
- `pnpm build` succeeds — confirmed

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None. This plan is CSS-only; no data-wiring stubs exist.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. T-133-02 (GPU-accelerated box-shadow on infinite animation) acknowledged in plan threat register — accepted; gated off under prefers-reduced-motion.

## Self-Check: PASSED

- frontend/src/style.css: modified (verified by grep + build)
- Commit 440df5d8: present
- Commit d0700126: present
- Commit af91acfc: present
