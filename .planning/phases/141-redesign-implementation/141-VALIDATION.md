---
phase: 141
slug: redesign-implementation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-20
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
| (planner-assigned) | — | 1 | RDS-02 | — / — | N/A | grep | `sed -n '<surface-range>p' frontend/src/style.css \| grep -E '#[0-9a-fA-F]{3,6}'` returns only D-03-fenced selectors | ✅ | ⬜ pending |
| (planner-assigned) | — | 1 | RDS-02 (S-07) | — / — | N/A | unit | `cd frontend && pnpm test -- SessionShareModal` | ✅ | ⬜ pending |
| (planner-assigned) | — | 1 | RDS-03 (D-11) | — / — | N/A | unit | `cd frontend && pnpm test -- StatusBar` | ✅ | ⬜ pending |
| (planner-assigned) | — | 1 | RDS-04 | — / — | N/A | grep | `grep -n 'transition\|animation' frontend/src/style.css` cross-checked vs `@media (prefers-reduced-motion: no-preference)` blocks | ✅ | ⬜ pending |
| (planner-assigned) | — | 1 | RDS-04 (theme parity) | — / — | N/A | grep | every migrated selector's token resolved in both `:root` and `[data-ui-theme="light"]` | ✅ | ⬜ pending |
| (planner-assigned) | — | 1 | CARRY-01 | — / — | N/A | unit | `cd frontend && pnpm test -- GroupSidebar` (asserts no `role="listbox"`/`role="option"`; roving-tabindex OR plain control list consistent) | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*Task IDs assigned by planner; rows map requirements → verification commands.*

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

No new test files needed — only assertion updates to existing files:

- [ ] `GroupSidebar.test.tsx` — assert items expose the chosen consistent pattern
      (e.g. `role="button"` + `aria-pressed`), `<ul>` has no `role="listbox"`, no
      `role="option"` items (CARRY-01 / #97)
- [ ] `StatusBar.test.tsx:70` — update expected string to new D-11 copy (no "Sessions tab")
- [ ] `StatusBar.test.tsx:66` — update test description string
- [ ] `SessionShareModal.test.tsx` — add smoke test asserting `.hub-share-modal__header`
      and `.hub-share-modal__body` render (S-07)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Light/dark toggle renders every migrated surface correctly | RDS-04 | Visual rendering can't be asserted in unit tests; user is colorblind so grep-backs the colors | Run `wails dev` (or web-share to Chrome), toggle `[data-ui-theme="light"]`, confirm each surface (Welcome, tab bar, status bar, File Browser, Editor chrome, Settings, Share Modal) repaints with no stuck dark hex; cross-check any doubt via the grep block above |
| Share modal enter/exit motion | RDS-04 / S-07 | Animation timing is observational | Open SessionShareModal with `prefers-reduced-motion: no-preference`, confirm fade; with `reduce`, confirm static |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify (grep or unit) or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all assertion updates above
- [ ] No watch-mode flags (`pnpm test` runs once, not `--watch`)
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
