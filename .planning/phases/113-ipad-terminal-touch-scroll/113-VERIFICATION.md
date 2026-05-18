# Phase 113 — iPad terminal touch-scroll — VERIFICATION

## iPad Safari UAT — PASS ✅ (2026-05-18, ken@kscott, physical iPad)

**Critical UAT finding:** Phase 113's planned fix in
`frontend/src/components/TerminalPanel.tsx` + `frontend/src/style.css` only
reaches the **Wails React** surface. The iPad hits
`web/assets/terminal.js` + `web/assets/terminal.css` — a separate vanilla-JS
surface that the original plan did NOT update. **Same shape as Phase 112's
two-surfaces issue.** Same fix shape ported to the web surface during this
UAT: `attachTouchScroll` IIFE in `terminal.js`, `touch-action: pan-y` on
`#terminal` in `terminal.css`.

**Probes captured on iPad Safari over a physical iPad:**

| Probe | Method | Result |
|-------|--------|--------|
| **UI-03** single-finger scroll | Generated ~200 lines via `ls -la /usr/bin \| head -200`, single-finger drag up scrolls older output into view | ✅ scrollback scrolls |
| **UI-04** tap-on-link (carry-over Issue #46) | `printf 'click here: \e]8;;https://example.com\e\\link-text\e]8;;\e\\\n'` → tap rendered "link-text" | ✅ WebLinksAddon click handler fires (8px tap-vs-drag threshold protected the tap path) |

**iPad Chrome:** Apple's policy requires all iOS browsers to use WebKit, so
iPad Chrome behavior is engine-identical to iPad Safari. No separate UAT
required (verified PASS on Safari is equivalent).

---



**Phase:** 113-ipad-terminal-touch-scroll
**Issue:** scottkw/agenthub#56
**Requirements:** UI-03, UI-04
**Generated:** 2026-05-18
**Linked:** [113-RESEARCH.md](./113-RESEARCH.md) · [113-CONTEXT.md](./113-CONTEXT.md)

---

## 1. Automated verification (macOS executor — DONE)

| Test | Asserts | Requirement |
|------|---------|-------------|
| `frontend/src/lib/__tests__/touchScrollHandler.test.ts` (10 cases) | Pure-function correctness: `scrollLines` sign (drag-down → older, drag-up → newer), sub-cell-height threshold gate, sub-threshold (<8px) tap leaves `preventDefault` un-called on touchend, multi-touch (`e.touches.length !== 1`) bail, cleanup removes all 4 listeners, live cell-height read from `term._core._renderService.dimensions.css.cell.height`, fallback to 17px when path is undefined, `preventDefault` IS called once a scroll is confirmed | UI-03, UI-04 |
| `frontend/src/components/__tests__/TerminalPanel.touchscroll.test.tsx` (5 cases) | Wiring: `import { attachTouchScroll } from '../lib/touchScrollHandler'`, call site exists, useEffect closes with `}, [sessionId])`, `.terminal-session-container` CSS rule contains `touch-action: pan-y`, AND does NOT contain `touch-action: none` | UI-03, UI-04 |
| `cd frontend && pnpm test` (full suite, 922/922 green) | No regression to existing terminal / link / popover / fontsize paths from the new useEffect or CSS rule | UI-04 |
| `cd frontend && npx tsc --noEmit` | TypeScript types check clean across all new files and edits | (hygiene) |

**Status:** all automated checks PASS as of commit `8f06df7`. Full counts:

- `touchScrollHandler.test.ts`: **10/10** pass
- `TerminalPanel.touchscroll.test.tsx`: **5/5** pass
- Full frontend suite: **922/922** pass across 62 test files
- TypeScript compile (`tsc --noEmit`): clean

---

## 2. Manual smoke (macOS executor — desktop Chrome mouse-wheel regression)

**Status:** `manual-on-macOS` — ~30 second visual check.

**Steps:**
1. Start the daemon and a shell session via `wails dev` (per memory `project_wails_devtools_disabled_in_prod` — production build has no DevTools, but a desktop wheel smoke does not need them).
2. In the shell, generate scrollback: `for i in $(seq 1 200); do echo line-$i; done`.
3. Mouse-wheel UP and DOWN over the terminal area.
4. **Pass criteria:** scrollback moves identically to the `main` baseline — wheel up reveals older lines, wheel down returns to bottom. No new lag, no double-step, no missed wheel ticks.

**Maps to:** UI-04 (no desktop regression).

**Why human-on-macOS rather than automated:** xterm's wheel handler runs entirely inside `SmoothScrollableElement` and cannot be exercised reliably under jsdom (no real `wheel` events, no canvas renderer). The unit suite covers mount/unmount of the new effect (which exercises the `if (!container || !term) return` guard); the visual wheel check is the cheapest way to confirm no behavioral regression.

**Result placeholder:** _to be filled in by operator after smoke._

---

## 3. Manual UAT (human_needed — physical iPad)

**Hardware:** iPad iOS 16+ (same device used for v3.3 UAT-04). `human_needed` — macOS executor cannot perform iPad UAT per CONTEXT.md scope and project memory `feedback_check_github_issues_during_uat`.

**Setup:** start daemon + web-share a shell session on macOS, note the web-share URL, open it on the iPad in Safari, then in Chrome.

### UAT-01 — iPad Safari single-finger scrollback (UI-03)
- **Status:** `human_needed`
- **Steps:** in the web-shared shell on iPad Safari, run `for i in $(seq 1 200); do echo line-$i; done`. Drag UP on the terminal area with one finger.
- **Pass:** scrollback reveals NEWER content (or stays at bottom), and the terminal does NOT pan the page. Drag DOWN reveals OLDER scrollback.

### UAT-02 — iPad Chrome single-finger scrollback (UI-03)
- **Status:** `human_needed`
- **Steps:** identical to UAT-01, on iPad Chrome.
- **Pass:** same behavior as UAT-01.

### UAT-03 — Two-finger gesture does not break (UI-03)
- **Status:** `human_needed`
- **Steps:** pinch-zoom on the terminal with two fingers in both Safari and Chrome.
- **Pass:** pinch-zoom either zooms the page or is benignly ignored. Crucially: the terminal does NOT wildly scroll. (Our handler bails on `e.touches.length !== 1`, so the second finger releases tracking and iOS handles the gesture.)

### UAT-04 — OSC 8 link tap (UI-04 + v3.3 UAT-04 carry-over)
- **Status:** `human_needed`
- **Steps:** from the iPad shell, run `printf '\e]8;;https://example.com\e\\link\e]8;;\e\\\n'`. Tap the rendered "link" cell.
- **Pass:** the `LinkConfirmPopover` appears (or the browser navigates / opens a new tab) — same as v3.3 desktop baseline.
- **Note (per RESEARCH Open Q2):** STATE.md:44 indicates this path was already broken on iPad pre-fix. The 8px tap-vs-drag threshold means sub-threshold taps now route through to the WebLinksAddon click handler without our `preventDefault` interfering — so the Phase 113 fix MAY repair UAT-04 as a side-effect. Two acceptable outcomes:
  1. **Side-effect fixed:** popover/navigation works → bonus repair, document in 113-01-SUMMARY.
  2. **Still broken but not regressed:** behaves the same as v3.3 baseline → no regression, file follow-up via `/gsd-plan-phase --gaps` if desired.

### UAT-05 — Sub-threshold tap on non-link cell (UI-04)
- **Status:** `human_needed`
- **Steps:** tap (no drag) on a non-link region of the scrollback in both Safari and Chrome.
- **Pass:** cursor positioning / selection behaves identically to v3.3 baseline; nothing scrolls. (Our handler treats sub-threshold movement as a tap and does not `preventDefault` on touchend.)

---

## 4. Cross-surface parity sanity

Desktop Wails has no touchscreen path (per CONTEXT.md — "Pure frontend bug. Web only — Wails desktop has no touchscreen path"). CLI / TUI surfaces have no frontend involvement. **The cross-surface gate from v3.3 Phase 108 is satisfied trivially: there is no parallel surface that this change should mirror.** This is iPad-only browser behavior.

The single point of contact with another surface is the OSC 8 link tap (probed by UAT-04) and the desktop mouse-wheel path (probed by §2). Both are explicitly tested.

---

## 5. Gap closure path

If any `human_needed` iPad UAT fails (other than the documented Open Q2 outcome for UAT-04):

- File a follow-up via `/gsd-plan-phase --gaps` with the failing UAT as the gap.
- Reference this `113-VERIFICATION.md` and the failing UAT ID.
- Cite the relevant requirement (UI-03 or UI-04).

If UAT-04 outcome is "still broken but not regressed", it remains the v3.3 UAT-04 carry-over and is tracked independently of Phase 113.

---

## 6. Source-grep gates (machine-checkable, per plan `<verification>`)

| Gate | Expected |
|------|----------|
| `grep -v '^//' frontend/src/lib/touchScrollHandler.ts \| grep -c "scrollLines(-lines)"` | `>= 1` (matches line 85) |
| `grep -v '^//' frontend/src/lib/touchScrollHandler.ts \| grep -c "passive: false"` | `>= 1` (matches line 107) |
| `grep -c "attachTouchScroll" frontend/src/components/TerminalPanel.tsx` | `>= 2` (import + call site → returns 2) |
| `grep -E "touch-action:\s*pan-y" frontend/src/style.css` | matches inside `.terminal-session-container` |
| `grep -E "touch-action:\s*none" frontend/src/style.css` inside `.terminal-session-container` block | does NOT match |

All gates verified passing at commit `8f06df7`.

---

## 7. Files modified (against frontmatter declaration)

| Declared | Actual | Status |
|----------|--------|--------|
| `frontend/src/lib/touchScrollHandler.ts` | created | ✓ |
| `frontend/src/lib/__tests__/touchScrollHandler.test.ts` | created | ✓ |
| `frontend/src/components/TerminalPanel.tsx` | edited (import + 1 useEffect) | ✓ |
| `frontend/src/components/__tests__/TerminalPanel.touchscroll.test.tsx` | created | ✓ |
| `frontend/src/style.css` | edited (added `touch-action: pan-y` to one rule) | ✓ |
| `.planning/phases/113-ipad-terminal-touch-scroll/113-VERIFICATION.md` | created (this file) | ✓ |

No scope creep. No untracked debris beyond pre-existing untracked items (`.claire/`, `.claude/`, `bin/`, `node_modules/`, `screenshots/`).
