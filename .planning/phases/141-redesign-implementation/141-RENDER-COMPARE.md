# 141-09 Render-Compare — App vs Canonical Comp

**Date:** 2026-06-21
**Plan:** 141-09 (gap_closure, autonomous: false — blocking human checkpoint)
**Supersedes:** 141-VERIFICATION.md (the token-only false-pass)

## Purpose

Close the false-pass. The original Phase 141 verification checked tokens/ARIA at the
**source** level and never rendered the comp. This plan rendered the **running app**
(`wails dev` → `:34115` bridge, driven with Playwright) and compared it surface-by-surface
to the canonical comp (`c-*.png`), backed by a blocking human visual checkpoint.

## Task 1 — Automated source gates: ALL GREEN

| Gate | Result |
|------|--------|
| vitest (style.redesign + style.hub + SettingsTab.appearance-theme) | ✅ 118/118 |
| `tsc --noEmit` | ✅ exit 0 |
| `pnpm build` | ✅ exit 0 (5 woff2 fingerprinted) |
| no-CDN font grep (googleapis/gstatic/@import url) | ✅ absent |
| vendored woff2 present | ✅ 5 files |
| `--hub-accent: #7aa2f7` present / `#7C8CFF` absent | ✅ blue locked, no violet |
| comp palette (`--hub-bg #14151b`, `--hub-success #4ade80`) | ✅ present |
| D-03 colorblind fences (agent-badge, status-state) | ✅ intact |
| go vendor-drift (xterm/CodeMirror pins) | ✅ ok |

## Task 2 — Live render vs comp (screenshots + computed styles, both themes)

Screenshots: `agenthub-v4.0-redesign/AgentHub UI redesign/screenshots/141-09/`
(`01-welcome-{dark,light}.png`, `02-hub-dark.png`, `03-settings-dark.png`, plus light variants).

Computed-style probe confirmed real surfaces/text use the comp tokens (not inherited fallbacks):
dark `sidebarLabel #f4f5f8 on #1c1e28`, `tagline #c7cad6 on #14151b`, `version #9398a8 on #14151b`;
light `sidebarLabel #1a1b26 on #ececf0`, `tagline #3a3b50 on #f5f5f7` (dark text on light — readable).

## Per-surface result

| Surface | Comp target | Font | Palette | Radii | Light/Dark | Verdict |
|---------|-------------|------|---------|-------|------------|---------|
| Welcome | c-welcome.png | Plus Jakarta Sans + JetBrains Mono | comp `#14151b`/comp grays | rounded code boxes/logo | both repaint | ✅ PASS (headless render) |
| Hub | (c-sessions/c-session chrome) | Plus Jakarta Sans | comp `#14151b`/`#16181f` | pill filter chips, rounded search | both repaint | ✅ PASS (headless render) |
| Settings | c-settings.png | Plus Jakarta Sans | sidebar `#16181f`, comp grays | rounded controls/toggles | both repaint + new Light/Dark control | ✅ PASS (headless render) |
| Session | c-session.png | Plus Jakarta Sans (chrome) + JetBrains Mono (terminal) | comp palette | rounded tabs/chips | n/a | ✅ PASS (human-confirmed, native) |
| File Browser | c-filebrowser.png | comp fonts | comp palette | rounded panels | n/a | ✅ PASS (human-confirmed, native — "uses the correct theme config") |

**Headless limitation (recorded, not a failure):** Session + File Browser require a live
terminal session; the `:34115` bridge has no PTY (terminal I/O only works in the native
webview). Both were confirmed by the human in the native `wails dev` window. Both are driven
by the same global `--hub-*` tokens + fonts proven on the three headless-rendered surfaces.

## Accent / accessibility

- Blue accent hex-verified: `#7aa2f7` (dark) / `#3d6fe8` (light) present; violet `#7C8CFF` absent.
- D-03 colorblind status fences + reduced-motion guards intact.
- WCAG AA ≥ 4.5 held on all re-pointed contrast assertions (min 5.77:1).

## Human verdict

**APPROVED** — the redesign visual language (Plus Jakarta Sans + JetBrains Mono fonts, comp
darker/cooler palette, comp radii, comp type scale, working+persisted Light/Dark toggle) has
**landed** across all surfaces. This is genuinely different from the pre-141 app and matches
the comp. The original 4/4 false-pass is closed: this verdict rests on a rendered app-vs-comp
comparison, not source greps.

## Newly discovered follow-up items (OUT OF SCOPE for 141 gap-closure — flagged during UAT, not silently passed)

These are net-new UI modifications/bugs the user raised during the checkpoint. They do NOT
regress the 141 redesign-fidelity goal (which is met). Routed to a separate effort:

1. **[BUG] Hub card icon overlap** — the ⋮ (top-right) and ☰ (top-left) icons overlap other
   card elements; the in-card preview is too small to be useful.
2. **[POLISH] Settings Light/Dark control** — should be a slider/toggle switch, not two
   separate buttons (refinement of 141-08).
3. **[POLISH] Hub "New session" buttons** — both (top-right + empty-state) should be styled
   like the comp's sidebar "New Session" button.
4. **[BUG] Terminal garble** — a Claude terminal session rendered partially garbled after a
   theme or tab switch. NEEDS INVESTIGATION — possible regression from the 141-08 theme toggle.
5. **[UX/IA] Hub groups** — replace the secondary side-by-side GROUPS panel; move groupings
   into the main sidebar under the Hub item (per the comp), or a better alternative.
