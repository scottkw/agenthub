---
phase: 139-card-rendering-tab-strip
plan: "04"
subsystem: frontend-vt-rendering
tags: [card-rendering, styled-tail, mini-preview, briefing-modal, CARD-05, vtColor, headless-xterm]
dependency_graph:
  requires: [139-01, 139-03]
  provides: [vtColor-resolveColor, MiniPreview-StyledSpan-render, HubBriefingModal-styled-tail]
  affects:
    - frontend/src/lib/vtColor.ts
    - frontend/src/components/Hub/MiniPreview.tsx
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/SessionCardGrid.tsx
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/Hub/HubBriefingModal.tsx
    - frontend/src/components/Hub/HubModal.tsx
tech_stack:
  added: []
  patterns:
    - resolveColor ITheme mapper (ansi:N → ITheme slot, #rrggbb passthrough, empty → fg/undefined)
    - MiniPreview StyledSpan[][] render (no xterm, per-span inline style via resolveColor, aria-hidden)
    - Atomic prop-chain migration (HubPanel → SessionCardGrid → SessionCard → MiniPreview)
    - HubBriefingModal headless xterm remote path (Terminal.write(callback) + serializeAsHTML + dispose, no term.open())
    - HubBriefingModal local styled-span path (React children auto-escaped, no dangerouslySetInnerHTML)
key_files:
  created:
    - frontend/src/lib/vtColor.ts
  modified:
    - frontend/src/components/Hub/MiniPreview.tsx
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/SessionCardGrid.tsx
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/Hub/HubBriefingModal.tsx
    - frontend/src/components/Hub/HubModal.tsx
    - frontend/src/components/Hub/HubBriefingModal.test.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/components/Hub/SessionCard.test.tsx
    - frontend/src/components/Hub/SessionCardGrid.test.tsx
decisions:
  - "term.write() is asynchronous — use callback form for serializeAsHTML to get flushed content (discovered during implementation, not documented in RESEARCH.md Pitfalls)"
  - "Theme prop threaded through HubModal → HubBriefingModal; HubModal already had theme: ITheme; zero plumbing needed at HubPanel level"
  - "No dangerouslySetInnerHTML on local path (T-139-07 mitigated by React children auto-escape)"
  - "Only dangerouslySetInnerHTML is remote path serializeAsHTML output (T-139-08 accepted per plan)"
metrics:
  duration: ~28 minutes
  completed: "2026-06-21"
  tasks_completed: 2
  tasks_total: 3
  files_created: 1
  files_modified: 10
---

# Phase 139 Plan 04: CARD-05 Frontend (vtColor + MiniPreview + HubBriefingModal) Summary

CARD-05 consumer half: vtColor resolveColor lib, MiniPreview StyledSpan[][] xterm-free render, and HubBriefingModal dual-path styled tail (local React children + remote headless xterm serializeAsHTML).

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | vtColor lib + MiniPreview StyledSpan render + atomic poller/prop-type migration | a948673c | vtColor.ts, MiniPreview.tsx, HubPanel.tsx, SessionCardGrid.tsx, SessionCard.tsx + test files |
| 2 | HubBriefingModal — local styled-span tail + remote headless-xterm tail | af8ad656 | HubBriefingModal.tsx, HubModal.tsx, HubBriefingModal.test.tsx |
| 3 (checkpoint) | Human verify: live legibility on local card, local briefing, remote briefing | PASSED (2026-06-20) | — |

## What Was Built

### Task 1 — vtColor lib + MiniPreview StyledSpan[][] + atomic poller migration

**`frontend/src/lib/vtColor.ts` (new):**
- `ANSI_THEME_KEYS` array mapping ANSI 0..15 to ITheme slots (black..white, brightBlack..brightWhite)
- `resolveColor(val, theme, isFg)` — maps `"ansi:N"` → `theme[slot]`, `"#rrggbb"` → passthrough, `""` / undefined → `theme.foreground` (isFg=true) or `undefined` (isFg=false)
- All 12 vtColor.test.ts RED tests turned GREEN

**`frontend/src/components/Hub/MiniPreview.tsx` (modified):**
- Props: `lines: string[] | undefined` → `lines: StyledSpan[][] | undefined`, added `theme: ITheme`
- Inner render: `<div>{line || ' '}</div>` → `<div key={i}><span key={j} style={{color, background, fontWeight}}>{span.c||' '}</span>...</div>`
- Colors resolved via `resolveColor(span.fg/bg, theme, isFg)` — maps through ITheme
- `aria-hidden="true"` preserved on ALL three states (loading/empty/data)
- MUST NOT import `Terminal` from `@xterm/xterm` (CARD-07) — only `import type { ITheme }` allowed
- All 10 MiniPreview.test.tsx RED tests turned GREEN

**`frontend/src/components/Hub/HubPanel.tsx` (modified):**
- `usePreviewPoller`: `GetSessionTailLines` → `GetSessionStyledTailLines`, `Map<string,string[]>` → `Map<string,daemon.StyledSpan[][]>`
- Passes `previewTheme={terminalTheme}` to `SessionCardGrid`

**`frontend/src/components/Hub/SessionCardGrid.tsx` (modified):**
- `previewTails?: Map<string, string[]>` → `Map<string, daemon.StyledSpan[][]>`, added `previewTheme?: ITheme`
- Threads `previewTheme` to both `SessionCard` call sites

**`frontend/src/components/Hub/SessionCard.tsx` (modified):**
- `previewLines?: string[]` → `daemon.StyledSpan[][]`, added `previewTheme?: ITheme`
- MiniPreview call: `<MiniPreview lines={previewLines} theme={previewTheme ?? {} as ITheme} />`

### Task 2 — HubBriefingModal dual-path styled tail

**`frontend/src/components/Hub/HubBriefingModal.tsx` (rewritten):**
- Deleted `stripAnsi()` and `extractTailLines()` regex helpers
- Added `theme: ITheme` prop (threaded from HubModal which already had it)
- **LOCAL path:** `GetSessionStyledTailLines(session.id, 20)` → `StyledSpan[][] | null` state; rendered as nested `<div><span style={...}>` rows via `resolveColor` (React children, auto-escaped, T-139-07 — NO `dangerouslySetInnerHTML`)
- **REMOTE path:** WS chunk accumulation unchanged; `finish()` now: `new Terminal({...,theme})` + `loadAddon(new SerializeAddon())` + `term.write(merged, callback)` (callback form — write is async!) + `serializeAsHTML({scrollback:20})` + `term.dispose()` → `setRemoteHtml(html)`; rendered via `<div dangerouslySetInnerHTML={{__html: remoteHtml}}/>` (ONLY usage of dangerouslySetInnerHTML — T-139-08 accepted)
- `term.open()` is NEVER called (Pitfall 5 — headless pattern)

**`frontend/src/components/Hub/HubModal.tsx` (modified):**
- Added `theme={theme}` to `<HubBriefingModal>` call site

## Key Decisions

| Decision | Resolution |
|----------|------------|
| `term.write()` async | Discovered write is asynchronous — switched to `term.write(merged, callback)` form where serializeAsHTML is called inside the callback after content is flushed. Without this, serializeAsHTML returns empty/blank rows. |
| theme threading in HubModal | HubModal already had `theme: ITheme` prop; just added it to the HubBriefingModal call site. Zero plumbing change at HubPanel level. |
| Local path rendering | Styled React `<span>` children, NOT `dangerouslySetInnerHTML` (T-139-07 satisfied) |
| Remote path rendering | Single `dangerouslySetInnerHTML` on the terminal-controlled output only (T-139-08 accepted per threat model) |
| Empty stub for `previewTheme` | `previewTheme ?? {} as ITheme` at SessionCard → MiniPreview call site; remote cards always have empty StyledSpan[][] so resolveColor is never called with an empty theme in practice |

## Deviations from Plan

### Auto-fixed: term.write() async callback form

**Found during:** Task 2 (TAIL-01a test revealed the issue)
**Issue:** `term.write(merged)` without a callback returned empty HTML from `serializeAsHTML` — the terminal buffer wasn't flushed yet. The A2 verification test used `await new Promise(resolve => term.write(data, resolve))` but the PATTERNS.md pseudocode showed the synchronous form.
**Fix:** Changed to `term.write(merged, () => { html = serAddon.serializeAsHTML(...); term.dispose(); setRemoteHtml(html) })` — the callback fires after the internal parser flushes the write queue.
**Files modified:** `frontend/src/components/Hub/HubBriefingModal.tsx`
**Commit:** af8ad656 (included in Task 2 commit)

### Auto-fixed (Rule 1): Test files broken by prop type migration

**Found during:** Task 1 (typecheck + test run)
**Issue:** `HubPanel.test.tsx`, `SessionCard.test.tsx`, `SessionCardGrid.test.tsx` used `GetSessionTailLines` mock and `string[]` / `Map<string,string[]>` fixtures that became type-incompatible after the prop chain migration.
**Fix:** Updated all four test files to use `GetSessionStyledTailLines` mock + `StyledSpan[][]` / `daemon.StyledSpan[][]` fixture data.
**Commits:** a948673c (Task 1)

### Auto-fixed (Rule 1): HubBriefingModal.test.tsx broken by API/prop changes

**Found during:** Task 2 (test run after HubBriefingModal rewrite)
**Issue:** Test mock used `GetSessionTailLines`, TAIL-01a checked for `<pre>` (now `div.hub-modal__tail-remote`), TAIL-01b/c referenced old function name, missing `theme` prop on `renderBriefing`.
**Fix:** Updated all assertions to match new implementation: `GetSessionStyledTailLines`, `hub-modal__tail-remote` div check, added `STUB_THEME` and `theme` prop threading.
**Commit:** af8ad656 (Task 2)

## Test Results

| Test File | Tests | Status |
|-----------|-------|--------|
| `frontend/src/lib/vtColor.test.ts` | 12 | GREEN |
| `frontend/src/components/Hub/MiniPreview.test.tsx` | 10 | GREEN |
| `frontend/src/components/Hub/HubBriefingModal.test.tsx` | 13 | GREEN |
| `frontend/src/components/Hub/HubPanel.test.tsx` | 44 | GREEN |
| `frontend/src/components/Hub/SessionCard.test.tsx` | pass | GREEN |
| `frontend/src/components/Hub/SessionCardGrid.test.tsx` | pass | GREEN |
| Full suite | 1732 | GREEN |

## Human-Verify Checkpoint (PASSED 2026-06-20)

**Status:** APPROVED via live `wails dev` UAT.

Verified live: Hub mini-preview card renders the real claude TUI (prompt, model line, styled "auto mode on") with legible columns and no leaked escapes/doubled lines; tab strip shows full named tabs.

### UAT-discovered gap fixes (two release-blocking bugs found during this checkpoint)

Live UAT surfaced two regressions that the source-level/jsdom tests could not catch. Both were root-caused (systematic-debugging) and fixed before approval:

1. **Mini-preview stuck on "Loading…" (backend hang).** `charmbracelet/x/vt` writes terminal-query responses (Device Attributes `ESC[c`/`ESC[>c`, DSR cursor-position `ESC[6n`, DECRQM, OSC color queries) into an unbuffered `io.Pipe`. With no reader draining `emu.Read`, the first such query in the replayed scrollback blocked `emu.Write` forever — `GetSessionStyledTailLines` / `/sessions/{id}/styled-tail` / the Wails binding all hung, so `usePreviewPoller`'s `Promise.all` never resolved and `lines` stayed `undefined`. Claude Code's TUI emits these queries at startup and they are captured in the PTY scrollback. **Fix (`5f221a39`):** drain+discard the emulator response pipe in a goroutine for the lifetime of `Write`, then `Close()`. Timeout-guarded regression test `TestGetSessionStyledTailLines_QueryNoHang` added (RED→GREEN).

2. **Tab strip collapsed to bare status dots (CSS).** `container-type: inline-size` on `.tab` establishes inline-size containment, removing the tab's own content from its intrinsic-size calc; with `flex-basis: auto` every tab collapsed to `min-width` (32px) regardless of available space, firing the icon-only `@container (max-width: 59px)` query on all tabs at once. **Fix (`214690a1`):** explicit `flex-basis: 180px` gives a concrete starting width that `flex-shrink` reduces toward the 32px floor only under real space pressure (TAB-01..03 affordances intact).

3. **`tsc` production-build break (`0c36c07a`).** Dead test vars in `TabBar.test.tsx` (used-before-assigned `container`/`root`; unread `renamedName`) passed `vitest` but failed `tsc && vite build`, blocking `wails dev`. Removed.

Note: User is colorblind — color fidelity verified at source level (`resolveColor` ANSI→ITheme tests); live check confirmed legibility, column spacing, and absence of leaked escapes.

## Threat Surface Scan

No new network endpoints, auth paths, or schema changes introduced. The threat surface changes documented in the plan's threat model are all covered:

| Flag | File | Coverage |
|------|------|---------|
| T-139-07 | HubBriefingModal.tsx local path | MITIGATED: React children (auto-escaped), no dangerouslySetInnerHTML |
| T-139-08 | HubBriefingModal.tsx remote path | ACCEPTED: terminal-controlled output only, scope limited to briefing modal |
| T-139-09 | MiniPreview.tsx | MITIGATED: no Terminal import or instantiation; type-only ITheme import |

## Known Stubs

None. All production code is fully wired:
- `resolveColor` maps real ITheme slots
- `MiniPreview` renders real StyledSpan[][] data from the daemon
- `usePreviewPoller` calls real `GetSessionStyledTailLines` Wails binding
- `HubBriefingModal` local path calls real `GetSessionStyledTailLines`
- `HubBriefingModal` remote path processes real WS bytes through headless xterm

## Self-Check

- [x] `frontend/src/lib/vtColor.ts` exists and contains `export function resolveColor` and `ANSI_THEME_KEYS`
- [x] `frontend/src/components/Hub/MiniPreview.tsx` prop type is `StyledSpan[][]`, takes `theme: ITheme`, renders per-span styled spans, imports NO `Terminal` from `@xterm/xterm`
- [x] `frontend/src/components/Hub/HubPanel.tsx` calls `GetSessionStyledTailLines`, tails is `Map<string, daemon.StyledSpan[][]>`
- [x] `frontend/src/components/Hub/SessionCard.tsx` MiniPreview call site passes `StyledSpan[][]` + `previewTheme`
- [x] `frontend/src/components/Hub/HubBriefingModal.tsx` has no `stripAnsi` or `extractTailLines`, has `serializeAsHTML`, has `GetSessionStyledTailLines`, has `theme: ITheme` prop
- [x] `frontend/src/components/Hub/HubModal.tsx` passes `theme={theme}` to `HubBriefingModal`
- [x] Commit a948673c exists (Task 1)
- [x] Commit af8ad656 exists (Task 2)
- [x] Full test suite: 1732 passed

## Self-Check: PASSED
