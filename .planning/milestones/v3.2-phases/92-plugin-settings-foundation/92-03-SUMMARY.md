---
phase: 92
plan: 03
subsystem: frontend
tags:
  - frontend
  - react
  - settings-ui
  - events
  - wails
requires:
  - 92-01 (daemon PluginSettings + HTTP routes + DaemonClient)
  - 92-02 ((*App).GetPluginSettings/SetPluginSettings + EventsEmit + Wails TS bindings + daemon.PluginSettings model)
provides:
  - "PluginsSection.tsx — h3 'Plugins' + 8 toggle rows in UI-SPEC order with one-sentence descriptions and three-state Save Plugins button (Save Plugins → Saving… → Saved! → Save Plugins)"
  - "SettingsTab integration — <PluginsSection /> renders as the LAST section below the Save Paths row"
  - "App.tsx pluginConfig pipeline — initial GetPluginSettings fetch + EventsOn('settings:plugins') subscription that updates state, threaded as a prop into the single <TerminalPanel> render site"
  - "TerminalPanelProps.pluginConfig?: PluginSettings | null — declared optional, destructured, void-discarded (Phase 92 inert-prop invariant)"
affects:
  - frontend/src/App.tsx (added GetPluginSettings import, daemon type import, pluginConfig state, EventsOn subscription, offPlugins cleanup, prop drill into TerminalPanel)
  - frontend/src/components/SettingsTab.tsx (added PluginsSection import + render below Save Paths row)
  - frontend/src/components/TerminalPanel.tsx (added daemon type import, optional pluginConfig prop on interface, destructured + void-discarded in function body)
tech-stack:
  added: []
  patterns:
    - "React inert-prop invariant for staged rollouts: optional prop accepted in interface, destructured, but only `void`-discarded — keeps tests green for downstream callers that don't pass it (Pitfall #4) AND lets a vitest source-inspection regex enforce that no useEffect ever reads it (Phase 93 will lift this)"
    - "Wails event subscription co-located with other EventsOn calls in App.tsx's mount useEffect, with a paired off* cleanup in the return — mirrors the 6 pre-existing subscriptions (offStatus, offHealth, offDaemonError, cancelTrayFocus, offExit, offQuit)"
    - "pluginsLoaded flicker guard (Pitfall #3) gates only the visible <label>, NOT the underlying <input> — mirrors SettingsTab Behavior toggle so test selectors find the input even before settings resolve"
    - "Three-state Save button cadence with 1500ms setTimeout reset, reusing settings-panel__btn--saved CSS class — mirrors the existing Save Paths handler verbatim"
key-files:
  created:
    - frontend/src/components/PluginsSection.tsx
    - frontend/src/components/__tests__/PluginsSection.test.tsx
    - frontend/src/__tests__/App.plugin-event.test.tsx
  modified:
    - frontend/src/App.tsx
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/components/TerminalPanel.tsx
key-decisions:
  - "Imported PluginSettings via `import type { daemon } from '../wailsjs/go/models'` then `type PluginSettings = daemon.PluginSettings`. Resolved Plan 92-02's documented fork in favor of the generated path (92-02-SUMMARY.md key-decisions confirmed models.ts is the canonical source; the alternative `../types/plugins` hand-written path was never created)."
  - "Phase 92 inert-prop invariant in TerminalPanel implemented via a single `void pluginConfig` line at the top of the function body (line 60) rather than dropping the destructure. This satisfies TypeScript's no-unused-variable rule, keeps the prop visible in source for the source-inspection test, and explicitly does NOT consume the prop in any useEffect — the vitest assertion `consumesInEffect` regex matches `useEffect(...pluginConfig` and `void` is not a useEffect, so the contract is mechanically enforced."
  - "Top-level `GetPluginSettings()` fetch in App.tsx is BOTH redundant with PluginsSection's own fetch AND necessary: PluginsSection drives the Settings UI; App.tsx's fetch ensures pluginConfig prop is populated before any TerminalPanel mounts. The EventsOn subscription handles all subsequent updates from any save path (PluginsSection's Save Plugins button or any future daemon-side change)."
requirements-completed:
  - PUI-01
  - PLUG-03
duration: "10 min"
completed: "2026-05-04"
---

# Phase 92 Plan 03: Plugin Settings Frontend Surface Summary

React surface of the Plugin Settings Foundation: a new `PluginsSection` component (h3 + 8 toggle rows in UI-SPEC order + three-state Save Plugins button + pluginsLoaded flicker guard), insertion as the final section in `SettingsTab.tsx`, App.tsx subscription to `EventsOn('settings:plugins', ...)` that propagates `pluginConfig` to all open TerminalPanels via prop drilling, and TerminalPanel's type-only acceptance of the new `pluginConfig?: PluginSettings | null` prop (Phase 92 inert-prop invariant — received but only `void`-discarded; Phase 93 wires consumption).

**Duration:** ~10 min · **Started:** 2026-05-04T15:55Z (approx) · **Completed:** 2026-05-04T16:04Z · **Tasks:** 3/3 · **Files:** 6 (3 created, 3 modified) · **Commits:** 3

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | RED — vitest source-inspection tests for PluginsSection + App event wiring | `25efe3d` | `frontend/src/components/__tests__/PluginsSection.test.tsx`, `frontend/src/__tests__/App.plugin-event.test.tsx` (NEW dir) |
| 2 | GREEN core — PluginsSection (8 rows, three-state Save, flicker guard) + SettingsTab insertion | `6f06d9b` | `frontend/src/components/PluginsSection.tsx` (new), `frontend/src/components/SettingsTab.tsx` |
| 3 | GREEN — App.tsx EventsOn subscription + prop-drill, TerminalPanel inert-prop typing | `9aedbb8` | `frontend/src/App.tsx`, `frontend/src/components/TerminalPanel.tsx` |

## File Roles

| File | Role |
|------|------|
| `frontend/src/components/PluginsSection.tsx` | New self-contained component. Owns `pluginConfig` (local edit state, null until first fetch), `pluginsLoaded` (Pitfall #3 flicker guard), `saving`/`saved`/`error` (three-state Save cadence), `loadError` (initial fetch failure surface). Calls `GetPluginSettings()` on mount, `SetPluginSettings(pluginConfig)` on Save Plugins. Renders 8 toggle rows in fixed UI-SPEC order via a `renderRow` helper that mirrors SettingsTab Behavior toggle markup (visible `<label>` gated by `pluginsLoaded && pluginConfig`; `<input type="checkbox">` always rendered for test selector stability). Save button reuses `settings-panel__btn--saved` CSS and the 1500ms setTimeout reset cadence verbatim from Save Paths. |
| `frontend/src/components/SettingsTab.tsx` | Added `import { PluginsSection } from './PluginsSection'`; rendered `<PluginsSection />` immediately AFTER the Save Paths row's closing `</div>` and BEFORE the closing `</div>` of `settings-panel__body`. Plugins is the LAST section per UI-SPEC layout. |
| `frontend/src/App.tsx` | Added `GetPluginSettings` to the existing Wails import block; added `import type { daemon } from './wailsjs/go/models'` + `type PluginSettings = daemon.PluginSettings` alias; declared `const [pluginConfig, setPluginConfig] = useState<PluginSettings \| null>(null)` adjacent to `daemonError`; inside the existing mount useEffect, added a fire-and-forget `GetPluginSettings().then(setPluginConfig).catch(() => {})` initial fetch and an `EventsOn('settings:plugins', (s) => setPluginConfig(s))` subscription registered as `offPlugins`; added `offPlugins()` to the cleanup return alongside the other 6 off* calls; threaded `pluginConfig={pluginConfig}` into the single `<TerminalPanel>` render site at line 866. |
| `frontend/src/components/TerminalPanel.tsx` | Added `import type { daemon } from '../wailsjs/go/models'` + `type PluginSettings = daemon.PluginSettings`; extended `TerminalPanelProps` with `pluginConfig?: PluginSettings \| null` (optional — Pitfall #4 guard so existing 36 TerminalPanel tests still construct the component without this prop); added `pluginConfig` to the function-signature destructure; added a single `void pluginConfig` line in the function body to satisfy the unused-variable rule WITHOUT placing the read inside any addon useEffect (Phase 92 inert-prop invariant). |
| `frontend/src/components/__tests__/PluginsSection.test.tsx` | 10 source-inspection assertions: 8 UI-SPEC labels present, 8 UI-SPEC one-sentence descriptions present, 8 plugin keys appear in UI-SPEC order (`indexOf` loop — Pitfall #5 guard), Save Plugins copy, `Saving…`/`Saved!` states, `settings-panel__btn--saved` CSS class reuse, 1500ms timeout, `pluginsLoaded` state declaration, JSX gating on `pluginsLoaded`, Wails RPC binding imports. |
| `frontend/src/__tests__/App.plugin-event.test.tsx` | 7 source-inspection assertions: `EventsOn('settings:plugins'` registration, `pluginConfig` state + `setPluginConfig` setter, initial `GetPluginSettings` fetch, `offPlugins()` cleanup call, `pluginConfig={pluginConfig}` prop drill, TerminalPanel optional-prop typing (`pluginConfig?:`), inert-prop invariant (`useEffect\([^)]*pluginConfig` regex must NOT match — fires loud if Phase 93 consumption sneaks in). |

## UI-SPEC Fidelity Confirmation

All 8 labels and 8 one-sentence descriptions match UI-SPEC.md verbatim:

| # | Key | Label | First words of description |
|---|-----|-------|---------------------------|
| 1 | `webgl` | WebGL renderer | GPU-accelerated terminal rendering… |
| 2 | `unicode11` | Unicode 11 widths | Correct cell widths for emoji and CJK… |
| 3 | `search` | Find in scrollback | Open a find bar with Cmd-F… |
| 4 | `webLinks` | Clickable web links | Detect URLs in terminal output… |
| 5 | `image` | Inline images | Render images sent via sixel… |
| 6 | `serialize` | Save terminal as text | Right-click a tab to export… |
| 7 | `clipboard` | Clipboard (OSC 52) | Allow the running CLI to place text… |
| 8 | `progress` | Progress (OSC 9;4) | Show a per-tab progress underline… |

`Saving…` uses U+2026 horizontal ellipsis (single character) verbatim — matches the existing Save Paths convention; `grep -q $'Saving…'` confirms the literal codepoint is present in source.

## Phase 92 Inert-Prop Invariant

The vitest test asserts `useEffect\([^)]*pluginConfig|useEffect\([^}]*\bpluginConfig\b` does NOT match in `TerminalPanel.tsx`. Verification:

```bash
$ grep -n -A5 'useEffect' frontend/src/components/TerminalPanel.tsx | grep -B1 -A1 pluginConfig
58:  // consuming the prop in any addon useEffect (inert-prop invariant).
59-  void pluginConfig
60-  const containerRef = useRef<HTMLDivElement>(null)
```

The only adjacency between `useEffect` and `pluginConfig` is in a doc comment 5 lines above the `void` line; the actual `void pluginConfig` at line 59 lives between the comment block and the `containerRef = useRef…` declaration — outside every useEffect body. Phase 93 will delete this `void` line and wire consumption inside the addon-load useEffect; that future change will simultaneously flip this test from green to red, prompting the Phase 93 author to delete the now-stale invariant. This is the intended hand-off.

## Verification

- `pnpm exec vitest run src/components/__tests__/PluginsSection.test.tsx` — **10/10 PASS**
- `pnpm exec vitest run src/__tests__/App.plugin-event.test.tsx` — **7/7 PASS**
- `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` — **36/36 PASS** (Pitfall #4 — existing tests construct TerminalPanel without `pluginConfig`; optional prop kept all green)
- `pnpm test` (full frontend) — **502 passed / 20 failed**. The 20 failures are pre-existing in `Sidebar.test.tsx` (documented in `92-02-SUMMARY.md` and `deferred-items.md` against `main@028f0b4` baseline). Net change vs Plan 92-02: **+17 passing tests** (10 PluginsSection + 7 App.plugin-event); zero new failures.
- `pnpm exec tsc --noEmit` — **clean** (zero type errors)
- `go build .` (root package) — **clean**
- `go test ./internal/daemon/... -count=1` — **PASS** (6.592s; full Plan 92-01 suite still green)
- `awk '/Save Paths/{seen=1} /<PluginsSection/{if (seen) print "OK"}' frontend/src/components/SettingsTab.tsx` — **OK** (PluginsSection appears AFTER Save Paths row)
- All 8 plugin keys grep individually: `for K in webgl unicode11 search webLinks image serialize clipboard progress; do grep -q "renderRow('$K'" frontend/src/components/PluginsSection.tsx; done` — **all match**

## Manual UAT Walkthrough Notes

Per `92-VALIDATION.md` §"Manual-Only Verifications" — the following scenarios were **NOT** exercised live in this plan execution and are deferred to the phase-level verification (`/gsd-verify-work 92`):

- `wails build -tags wailsassets` end-to-end build smoke (requires the Wails toolchain + signing certificate — out of scope for plan-level execution).
- App-launch UAT showing the Plugins section below Paths with 8 rows.
- Toggle persistence across app + daemon restart.
- Multiple-TerminalPanel tabs receiving `pluginConfig` within 200ms of save (verifiable via React DevTools).

The source-inspection tests cover the static contract (labels, ordering, button states, subscription registration, prop threading, inert-prop invariant). Live UAT is the right tool to verify the `EventsEmit` → `EventsOn` round-trip wall-clock behavior and is the gate for Phase 92 SC-3 / SC-5.

## Deviations from Plan

### Total deviations

**None — plan executed exactly as written.**

The plan specified two possible PluginSettings import paths depending on Plan 92-02's outcome (generated `daemon.PluginSettings` vs hand-written `../types/plugins`). Plan 92-02's SUMMARY confirmed the generated path was the canonical choice; Plan 92-03 used the generated path verbatim. This is not a deviation — it is the plan's documented fork-resolution.

## Authentication Gates

None — entirely local TypeScript + React work; no CLI tools requiring login.

## Threat Flags

None — no new attack surface beyond the threat model in `92-03-PLAN.md`. The Wails event-bus boundary (T-92-10) and the React DOM input boundary (no `dangerouslySetInnerHTML`, no `innerHTML`) are unchanged from the plan's threat register. The TerminalPanel inert-prop invariant (T-92-12) is mitigated by the `consumesInEffect` source-inspection test, which is in place and green.

## Self-Check: PASSED

- File checks (existence):
  - `frontend/src/components/PluginsSection.tsx` — FOUND
  - `frontend/src/components/__tests__/PluginsSection.test.tsx` — FOUND
  - `frontend/src/__tests__/App.plugin-event.test.tsx` — FOUND
- Modified file checks (`grep` for inserted symbols):
  - `frontend/src/components/SettingsTab.tsx` contains `<PluginsSection` — FOUND
  - `frontend/src/App.tsx` contains `EventsOn('settings:plugins'` — FOUND
  - `frontend/src/App.tsx` contains `pluginConfig={pluginConfig}` — FOUND
  - `frontend/src/components/TerminalPanel.tsx` contains `pluginConfig?: PluginSettings` — FOUND
- Commit checks (`git log --oneline | grep`):
  - `25efe3d` — FOUND (Task 1, RED)
  - `6f06d9b` — FOUND (Task 2, GREEN core)
  - `9aedbb8` — FOUND (Task 3, GREEN wiring)
- Verification commands re-run after writing this SUMMARY:
  - 10/10 PluginsSection tests — PASS
  - 7/7 App.plugin-event tests — PASS
  - 36/36 TerminalPanel tests — PASS (Pitfall #4)
  - tsc --noEmit — clean
  - daemon Go suite — PASS

## Phase 92 Status: All 3 Plans Complete

Plan 92-03 is the final plan in Phase 92 (Plugin Settings Foundation). All three success criteria for the phase are now observable in source:

1. **Settings UI** — Plugins h3 below Paths with 8 rows in UI-SPEC order, defaults from daemon (7 ON / Progress OFF), three-state Save (PUI-01 ✓).
2. **Persistence** — Save → daemon settings.json → defaults-merge load on next startup (PLUG-01 / PLUG-02 from Plans 92-01 + 92-02 ✓).
3. **Propagation pipeline** — `(*App).SetPluginSettings` emits `settings:plugins` → App.tsx EventsOn → `pluginConfig` state → prop drill into every TerminalPanel (PLUG-03 ✓; consumption deferred to Phase 93 by design).

Phase 92 is ready for `/gsd-verify-work 92` and the manual UAT smoke (`wails build -tags wailsassets` + app-launch settings-panel walkthrough). Phase 93 will lift the inert-prop invariant in TerminalPanel and wire actual addon-load gating off the `pluginConfig` prop.
