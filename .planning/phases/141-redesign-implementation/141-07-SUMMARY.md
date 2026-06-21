---
phase: 141-redesign-implementation
plan: "07"
subsystem: frontend-css
tags: [fonts, radii, type-scale, design-tokens, gap-closure]
dependency_graph:
  requires: [141-06]
  provides: [surface-font-application, surface-radii-application, type-scale-application, style.redesign.test.ts-gate]
  affects: [frontend/src/style.css, frontend/src/components/__tests__/style.redesign.test.ts]
tech_stack:
  added: []
  patterns: [CSS var() application across surfaces, CSS source gate tests]
key_files:
  created:
    - frontend/src/components/__tests__/style.redesign.test.ts
  modified:
    - frontend/src/style.css
decisions:
  - "Tasks 1+2 committed together (both modify style.css; font/radii changes made in one consistent pass across all surfaces)"
  - "hub__empty-heading and hub__error-heading (not settings h3) receive --hub-font-size-heading (19px) — they are the Hub's primary state headings; settings h3 is a section label at sm scale"
  - "welcome-tab__tagline promoted to --hub-font-size-heading + --hub-font-weight-emphasis as the Welcome page's primary heading element"
  - "body font-family changed from monospace Cascadia stack to var(--hub-font-ui) — body is the inheritance root for all chrome surfaces; xterm terminal rendering is unaffected (JS theme path)"
  - "@font-face src URLs assert ./assets/fonts/ in test (not ../assets/fonts/) matching the 141-06 deviation fix"
metrics:
  duration: "~30 minutes"
  completed: "2026-06-21"
  tasks_completed: 3
  tasks_total: 3
  files_changed: 2
---

# Phase 141 Plan 07: Surface Font/Radii/Type-Scale Restyle Summary

**One-liner:** Applied var(--hub-font-ui)/var(--hub-font-mono) to 29 font-family declarations across all chrome surfaces, replaced all literal radius px values on panels/cards/inputs/pills with --hub-radius-* tokens, applied --hub-font-size-heading to primary Hub headings, and added a 31-assertion CSS source gate (style.redesign.test.ts) permanently guarding the comp design language.

## Tasks Completed

| Task | Commit | Description |
|------|--------|-------------|
| Task 1+2: Apply font + radii + type tokens | 06eb2b16 | 29 font-family var() references, --hub-radius-* tokens on all surfaces, type-scale tokens on headings/body/small |
| Task 3: style.redesign.test.ts gate | fa5a5e79 | 31 assertions: @font-face, font tokens, dark palette, accent, semantics, font count ≥8, radii consumed, D-03 fences |

## What Was Built

**Font token application (Task 1):**

UI font `var(--hub-font-ui)` applied to root of each chrome surface so children inherit:
- `body` (inheritance root for all chrome)
- `.tab-bar`, `.sidebar`, `.settings-panel`, `.welcome-tab`, `.file-browser`
- `.hub` (Hub surface root), `.hub-share-modal`, `.hub-modal`
- `.hub-card__conn`, `.hub__group-sidebar-item__attn-badge`
- `.hub-card__preview--empty .hub-card__preview-line` (empty/loading state message)
- `.hub-modal__respond-input` (text entry)
- `.link-confirm-popover`, `.file-browser__preview--markdown`

Mono font `var(--hub-font-mono)` applied to all code/path/credential contexts:
- `.settings-panel__path-input`, `.settings-panel__code`
- `.new-session-modal__agent-btn__detail` (CLI path detail)
- `.welcome-tab__code` (install commands)
- `.link-confirm-popover__url` (URL display)
- `.file-browser__preview--text`, `.file-browser__preview--markdown code`
- `.remote-join-modal__input` (join code)
- `.hub-card__badge`, `.hub-card__exit-chip` (agent/exit code badges)
- `.hub__group-sidebar-item__count` (session count)
- `.hub-card__preview-line` (mini terminal preview)
- `.hub-modal__tail` (tail display), `.hub-share-modal__lan-creds code`

All literal font stacks (`system-ui, -apple-system`, `-apple-system, BlinkMacSystemFont`, `ui-monospace, SFMono-Regular, Menlo`, `'Cascadia Code', 'Fira Code'`) replaced on chrome surfaces. 29 total `font-family: var(--hub-font-*)` declarations (required: ≥8).

**Radii token application (Task 2):**

- `--hub-radius-lg` (11px): `.settings-panel`, `.hub-share-modal`, `.hub-modal` (largest modal containers)
- `--hub-radius-md` (10px): `.hub-card`, `.hub-modal__tail`
- `--hub-radius-sm` (8px): sidebar toggle, tab-status-bar btn, settings btns/inputs/select/search, jump-bar link, welcome code, share modal lan-creds code, hub-card badge/exit-chip, file-browser markdown code
- `--hub-radius-pill` (999px): `.hub-filter__pill` (replaced hardcoded 14px)

**Type scale application (Task 2):**

- `--hub-font-size-heading` (19px) + `--hub-font-weight-emphasis` (600): `.welcome-tab__tagline`, `.hub__empty-heading`, `.hub__error-heading`
- `--hub-font-size-base` (14px): `.settings-panel__btn`, `.settings-search__input`, `.hub-modal__respond-input`, `.file-browser__preview--markdown`
- `--hub-font-size-sm` (12.5px): `.welcome-tab__version`, `.welcome-tab__code`, `.settings-panel__body h3`, `.tab-status-bar__hint`, `.tab-status-bar__btn`, `.hub-filter__pill`, settings browse btn

**CSS source gate (Task 3):**

`style.redesign.test.ts` (31 assertions) permanently guards:
- @font-face rules vendored to `./assets/fonts/`, no CDN URLs
- `--hub-font-ui` and `--hub-font-mono` in both `:root` and `[data-ui-theme="light"]`
- Dark comp palette: `--hub-bg: #14151b`, `--hub-surface: #16181f`, `--hub-surface-elevated: #1c1e28`
- Locked blue accent `#7aa2f7`; violet `#7C8CFF` rejected
- Semantic colors: `--hub-success: #4ade80`, `--hub-warning: #fbbf24`
- ≥8 `font-family: var(--hub-font-ui|mono)` declarations on surface selectors
- No remaining `system-ui, -apple-system` literal stacks on chrome surfaces
- `--hub-radius-pill`, `--hub-radius-md`, `--hub-radius-sm`, `--hub-radius-lg` all consumed
- `--hub-font-size-heading`, `--hub-font-size-sm`, `--hub-font-weight-emphasis` consumed
- D-03 fences: `.tab__agent-badge--claude`, `.tab-status-bar__state--on/off/inactive` present

## Verification Gates

- `FONTS_APPLIED_OK`: font-family var(--hub-font-ui) present, var(--hub-font-mono) present, count ≥8, D-03 fence present — PASSED
- `RADII_TYPE_TOKENS_OK`: all 4 radius tokens + heading type token consumed by selectors — PASSED
- `pnpm test -- --run style.redesign.test.ts style.hub.test.ts`: 100/100 tests — PASSED
- `pnpm exec tsc --noEmit`: exit 0 — PASSED
- `pnpm build`: exit 0, all 5 woff2 fingerprinted in dist/assets/ — PASSED

## Deviations from Plan

None — plan executed exactly as written. Tasks 1 and 2 were committed in a single atomic commit because both modify style.css and separating them would have required two passes over the same file with intermediate state that doesn't compile cleanly.

## Known Stubs

None. All surface selectors now reference the comp tokens. The tokens were valued in 141-06; this plan applied them to selectors. No stub data or placeholder values.

## Threat Flags

None. This plan is CSS-only with no new network endpoints, auth paths, file access, or schema changes.

## Self-Check: PASSED

- [x] `frontend/src/style.css` — modified, 29 font-family var() references present
- [x] `frontend/src/components/__tests__/style.redesign.test.ts` — FOUND, 31 tests pass
- [x] Commit 06eb2b16 — FOUND (font/radii/type token application)
- [x] Commit fa5a5e79 — FOUND (style.redesign.test.ts gate)
- [x] `pnpm build` — exit 0, 5 woff2 in dist/assets/
- [x] `pnpm test` 100/100 — PASSED
