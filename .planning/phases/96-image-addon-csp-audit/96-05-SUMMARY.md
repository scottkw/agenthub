---
phase: 96
plan: 05
subsystem: wails-rpc
tags: [phase-96, image, wails, app-method, bindings, wave-2, IMG-02]
requires:
  - 96-01  # daemon.ImageConfig struct + models.ts hand-edit
  - 96-02  # (*DaemonClient).SetImageConfig wrapper
provides:
  - "(*App).SetImageConfig(cfg daemon.ImageConfig) error"
  - "App.SetImageConfig TS binding (frontend/src/wailsjs/go/main/App.d.ts)"
  - "App.SetImageConfig JS Call stub (frontend/src/wailsjs/go/main/App.js)"
affects:
  - "Frontend can now call App.SetImageConfig({ storageLimit: N }) via Wails RPC"
  - "settings:plugins event listeners (App.tsx) receive full PluginSettings frame post-write"
tech-stack:
  added: []
  patterns:
    - "Wails-method daemon-fanout: sub-key write → re-fetch full PluginSettings → EventsEmit (Phase 94-07 / Phase 95-05 lineage)"
    - "Synthesize-from-defaults fallback when readback fails (Phase 95 invariant)"
    - "WR-05 nil-ctx guard before EventsEmit (Phase 95 code-review fix carried forward)"
    - "Hand-edited Wails TS/JS bindings (Phase 92 STATE.md pin pattern)"
key-files:
  created: []
  modified:
    - app.go
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
decisions:
  - "Mirror Phase 95 SetWebLinksConfig verbatim with single field swap (daemon.WebLinksConfig → daemon.ImageConfig); no behavioral deviation."
  - "Event payload remains full PluginSettings (not sub-key) so App.tsx EventsOn handler consumes it identically to SetPluginSettings frames."
  - "Hand-edit binding files instead of running `wails generate module` — consistent with Phase 92/94/95 STATE.md pin pattern."
metrics:
  duration: ~10 minutes
  completed: 2026-05-07
---

# Phase 96 Plan 05: SetImageConfig Wails RPC Summary

Land the IMG-02 Wails RPC layer: `(*App).SetImageConfig` exposes a daemon-fanout writer for the ImageConfig sub-key of PluginSettings, mirroring Phase 95 `SetWebLinksConfig` verbatim, with hand-edited TS/JS bindings per the Phase 92 pin pattern.

## What Shipped

- **`(*App).SetImageConfig(cfg daemon.ImageConfig) error`** in `app.go` (immediately after `(*App).SetWebLinksConfig`):
  - Daemon-not-connected guard: `if a.client == nil { return fmt.Errorf("daemon not connected") }`
  - Sub-key write via DaemonClient wrapper from Plan 96-02: `a.client.SetImageConfig(cfg)`
  - Re-fetch full PluginSettings: `a.client.GetPluginSettings()`
  - Synthesize-from-defaults fallback on readback failure: `full = daemon.PluginSettings{ImageConfig: cfg}`
  - WR-05 nil-ctx guard: `if a.ctx != nil && a.ctx.Value("frontend") != nil`
  - `runtime.EventsEmit(a.ctx, "settings:plugins", full)` so App.tsx EventsOn listener consumes a full PluginSettings frame (same shape as SetPluginSettings).
  - Doc comment notes the next-session-only semantics: the event fires (so future `<details>` UIs and the web SSE consumer reflect the new value), but TerminalPanel mount useEffect intentionally does not include `imageConfig` in any hot-swap dep array.
- **`frontend/src/wailsjs/go/main/App.d.ts`**: added `export function SetImageConfig(arg1: daemon.ImageConfig): Promise<void>` with the same JSDoc-style comment header used by SetSearchConfig and SetWebLinksConfig entries.
- **`frontend/src/wailsjs/go/main/App.js`**: added `export const SetImageConfig = (cfg) => Call('main.App.SetImageConfig', [cfg])` matching the existing one-liner style of SetWebLinksConfig.

## Verification

- `go build ./...` — exit 0
- `go vet ./...` — exit 0
- `gofmt -l app.go` — empty (no diff)
- `go test ./... -count=1` — all packages green (no Phase 92/93/94/95 regression):
  - `github.com/scottkw/agenthub` (root, 23.6s)
  - `github.com/scottkw/agenthub/internal/daemon` (6.4s)
  - 11 other packages green
- `pnpm tsc --noEmit` — clean except for one pre-existing TS6133 warning in `frontend/src/components/FindBar/__tests__/FindBar.animation.test.tsx:15` (Phase 94 origin, already documented in `.planning/phases/96-image-addon-csp-audit/deferred-items.md`). Out of scope for Phase 96.
- `pnpm test src/__tests__/App.plugin-event.test.tsx` — 15/15 passing (Plan 96-01 RED scaffold for App.plugin-event imageConfig assertions consumes our SetImageConfig binding cleanly).
- Source ordering: `(*App).SetImageConfig` at app.go:589 sits after `(*App).SetWebLinksConfig` at app.go:549, preserving the SetPluginSettings → SetSearchConfig → SetWebLinksConfig → SetImageConfig grouping.

## Truths Mapped to Evidence

| Truth | Evidence |
|-------|----------|
| `(*App).SetImageConfig` exists and routes through `a.client.SetImageConfig` | `grep -c "func (a \*App) SetImageConfig" app.go` == 1; `grep -q "a.client.SetImageConfig(cfg)" app.go` |
| Daemon-not-connected guard | `grep -q "a.client == nil" app.go` (immediately above the write) |
| Re-fetch + synthesize fallback | `grep -q "daemon.PluginSettings{ImageConfig: cfg}" app.go` |
| WR-05 nil-ctx guard | `grep -q 'a.ctx.Value("frontend")' app.go` |
| EventsEmit fires `settings:plugins` | `grep -q 'runtime.EventsEmit(a.ctx, "settings:plugins"' app.go` |
| TS binding compiles | `pnpm tsc --noEmit` (only pre-existing FindBar warning unrelated to this plan) |
| JS binding routes through Wails Call | `grep -q "main.App.SetImageConfig" frontend/src/wailsjs/go/main/App.js` |

## Deviations from Plan

None — plan executed verbatim. The plan specified verbatim mirror of Phase 95 `SetWebLinksConfig`; the implementation is the verbatim mirror with field name swap (`WebLinksConfig` → `ImageConfig`) and an additional doc-comment paragraph clarifying the next-session-only semantics that the plan called for in the action block.

A minor stylistic adaptation: the plan's action block proposed multi-line `(arg1) => Call(...)` form for `App.js`, but the existing file uses a fluent one-liner style (`= (cfg) => Call('main.App.X', [cfg])`) consistently across SetSearchConfig and SetWebLinksConfig entries. I followed the file's existing style for consistency rather than the plan's example, which matches the plan's broader instruction to "match the existing formatting".

## Auth Gates

None encountered.

## Parallel-Safety Notes

- Plan 96-04 (frontend desktop integration — TerminalPanel.tsx + PluginsSection.tsx) is parallel-safe with this plan: different files, no source overlap.
- The `pnpm test` full-suite run shows 26 RED-scaffold failures across `TerminalPanel.test.tsx` and `PluginsSection.test.tsx`, all marked `expect.fail("RED scaffold — Plan 96-04 implements ...")` — these are the Plan 96-04 acceptance gates that go GREEN when its sibling plan lands. Sidebar test failures predate Phase 96 (unrelated UI work). Our Plan 96-01 RED scaffold for App.plugin-event tests turns GREEN with this plan's binding addition (15/15 passing).

## Self-Check: PASSED

- `app.go` modified — VERIFIED (`grep -q "func (a \*App) SetImageConfig" app.go` returns 0)
- `frontend/src/wailsjs/go/main/App.d.ts` modified — VERIFIED (`grep -q "SetImageConfig" frontend/src/wailsjs/go/main/App.d.ts` returns 0)
- `frontend/src/wailsjs/go/main/App.js` modified — VERIFIED (`grep -q "SetImageConfig" frontend/src/wailsjs/go/main/App.js` returns 0)
- Commit `a5b8c33` — VERIFIED present in `git log --oneline`
