---
phase: 141
slug: redesign-implementation
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-20
validated: 2026-06-21
---

# Phase 141 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: 141-RESEARCH.md §Validation Architecture. This is a recolor-only CSS
> token migration + ARIA fix + copy reword; most verification is grep-based on
> hex constants (the user is colorblind — never verify color by eye) plus
> targeted component-test assertions.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest ^4.1.0 |
| **Config file** | `frontend/vite.config.ts` (vitest embedded) |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `cd frontend && pnpm exec tsc --noEmit && pnpm test` |
| **Estimated runtime** | ~30 seconds |

**Critical gate note:** `tsc --noEmit && vite build` rejects TS errors that
vitest tolerates (project memory: "Run tsc in the frontend gate, not just
vitest"). The per-wave and phase gate MUST run `tsc --noEmit`, not just vitest.

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test -- <surface-test-file>`
- **After every plan wave:** Run `cd frontend && pnpm exec tsc --noEmit && pnpm test`
- **Before `/gsd:verify-work`:** Full suite green AND no hardcoded hex in migrated
  CSS sections (grep, excluding D-03-fenced selectors)
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| — | — | 1 | RDS-02 (no raw hex) | — / — | N/A | unit | `pnpm exec vitest run src/__tests__/themeTokens.test.ts` — asserts no raw hex outside D-03 fence (replaces brittle line-anchored greps) | ✅ | ✅ green |
| — | — | 1 | RDS-02 (S-07) | — / — | N/A | unit | `cd frontend && pnpm test -- SessionShareModal` | ✅ | ✅ green |
| — | — | 1 | RDS-03 (D-11) | — / — | N/A | unit | `cd frontend && pnpm test -- StatusBar` | ✅ | ✅ green |
| — | — | 1 | RDS-04 (reduced-motion) | — / — | N/A | unit | `themeTokens.test.ts` — asserts `@media (prefers-reduced-motion)` guards + share-modal motion gating | ✅ | ✅ green |
| — | — | 1 | RDS-04 (theme parity) | — / — | N/A | unit | `themeTokens.test.ts` — asserts every `--hub-*` token defined in BOTH `:root` and `[data-ui-theme="light"]` (53↔53) | ✅ | ✅ green |
| — | — | 1 | CARRY-01 (#97) | — / — | N/A | unit | `cd frontend && pnpm test -- Sidebar` — asserts `aria-pressed` on group buttons (GroupSidebar refactored into Sidebar.tsx in Phase 142) | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> **2026-06-21 audit note:** The three RDS-02/RDS-04 grep rows were converted from
> brittle `sed -n 'A,Bp' | grep` commands (line ranges broke after Phase 142 edited
> style.css) into a single stable selector-based vitest file:
> `frontend/src/__tests__/themeTokens.test.ts`. CARRY-01 moved from a planned
> `GroupSidebar.test.tsx` to `Sidebar.test.tsx` because Phase 142 (POL-05) lifted the
> group controls out of the standalone GroupSidebar component into the Sidebar.

---

## Colorblind Verification (grep-based — user is colorblind)

Run after each surface migration; must show **only** D-03-fenced hex remaining:

```bash
# Confirm no hardcoded hex remains in migrated CSS sections (post-migration):
sed -n '82,368p'   frontend/src/style.css | grep -E '#[0-9a-fA-F]{3,6}'   # tab bar / status bar
sed -n '1302,1404p' frontend/src/style.css | grep -E '#[0-9a-fA-F]{3,6}'  # welcome tab
sed -n '370,771p'  frontend/src/style.css | grep -E '#[0-9a-fA-F]{3,6}'   # settings
sed -n '2646,3380p' frontend/src/style.css | grep -E '#[0-9a-fA-F]{3,6}'  # file browser
# Accent must appear ONLY in token declarations + agent-badge + status annotations:
grep -rn '#7aa2f7' frontend/src/style.css   # dark accent
grep -rn '#3d6fe8' frontend/src/style.css   # light accent
```

**Permitted residual hex (D-03 fence — DO NOT flag):**
- Agent badge colors (`.tab__agent-badge--*`) — semantic per-agent identifiers
- Status state colors (`.tab-status-bar__state--*`, hub card dots) — colorblind-safe via icon+text
- `rgba(0,0,0,...)` shadow opacity values — not theme colors

---

## Wave 0 Requirements

All assertion updates landed; CARRY-01 relocated per Phase 142 refactor:

- [x] `Sidebar.test.tsx` — asserts group item buttons expose `aria-pressed` true/false
      (CARRY-01 / #97; GroupSidebar component was folded into Sidebar.tsx in Phase 142 POL-05)
- [x] `StatusBar.test.tsx` — asserts new D-11 copy ("WEB ON/OFF/SERVER NOT RUNNING", no "Sessions tab")
- [x] `SessionShareModal.test.tsx` — smoke test asserts `.hub-share-modal__header`
      and `.hub-share-modal__body` render (S-07)
- [x] `themeTokens.test.ts` (NEW) — selector-based RDS-02/RDS-04 color guardrails

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Light/dark toggle renders every migrated surface correctly | RDS-04 | Visual rendering can't be asserted in unit tests; user is colorblind so grep-backs the colors | Run `wails dev` (or web-share to Chrome), toggle `[data-ui-theme="light"]`, confirm each surface (Welcome, tab bar, status bar, File Browser, Editor chrome, Settings, Share Modal) repaints with no stuck dark hex; cross-check any doubt via the grep block above |
| Share modal enter/exit motion | RDS-04 / S-07 | Animation timing is observational | Open SessionShareModal with `prefers-reduced-motion: no-preference`, confirm fade; with `reduce`, confirm static |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify (unit) — grep checks converted to vitest
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all assertion updates above
- [x] No watch-mode flags (`pnpm test` runs once, not `--watch`)
- [x] Feedback latency < 30s (themeTokens ~0.9s; full suite ~36s)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-21

---

## Validation Audit 2026-06-21

| Metric | Count |
|--------|-------|
| Gaps found | 3 (RDS-02 hex-scan, RDS-04 reduced-motion, RDS-04 theme-parity — all grep-only, line ranges broken by Phase 142) |
| Resolved (automated) | 3 → consolidated into `frontend/src/__tests__/themeTokens.test.ts` |
| Impl bugs surfaced | 1 (RDS-02) |
| Impl bugs fixed | 1 |

**Bug found & fixed (RDS-02 — light-theme orphan):** The new colorblind guard
caught ~40 raw-hex values across migrated File-Browser/Editor sub-surfaces and
`.settings-panel__btn--destructive` that Phase 141 commit `3efffd5e` never migrated
— they hardcoded the **old Tokyo Night palette** (`#1e2030`/`#292e42`/`#c0caf5`/
`#9aa5ce`/`#a9b1d6`/`#1a1b26`/`#7aa2f7`/`#f7768e`) with no `[data-ui-theme="light"]`
override, so those surfaces rendered stuck-dark in light mode (and stale vs. the
redesign in dark mode). Source-verified per the colorblind contract: the redesign
token *values* differ from the old hex, so this was a real un-migrated surface, not
a cosmetic nit. Fixed by tokenizing each by role to `var(--hub-*)`; status icons
(`save-indicator-icon--saving/saved/error`) mapped to semantic `--hub-warning/
--hub-success/--hub-destructive` (icon shape carries the non-color cue).

**Gate after fix:** `tsc --noEmit` exit 0 · full vitest 1771/1771 green · themeTokens 5/5 green.
