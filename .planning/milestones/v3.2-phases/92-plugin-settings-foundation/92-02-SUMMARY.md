---
phase: 92
plan: 02
subsystem: app-go
tags:
  - wails
  - rpc
  - events
  - go
  - typescript
requires:
  - 92-01 (PluginSettings struct, DaemonClient.{Get,Set}PluginSettings)
provides:
  - "(*App).GetPluginSettings — Wails binding returning daemon.PluginSettings (zero value on disconnect/RPC fail)"
  - "(*App).SetPluginSettings — Wails binding that delegates to DaemonClient then emits runtime.EventsEmit(\"settings:plugins\", s)"
  - "frontend/src/wailsjs/go/main/App.d.ts: GetPluginSettings/SetPluginSettings declarations + daemon namespace import"
  - "frontend/src/wailsjs/go/main/App.js: Call('main.App.{Get,Set}PluginSettings') wrappers"
  - "frontend/src/wailsjs/go/models.ts (NEW): pinned daemon namespace with PluginSettings class (8 boolean fields)"
affects:
  - app.go (added 36 lines below SetAutoCloseSession)
  - frontend/src/wailsjs/go/main/App.d.ts (added daemon import + 2 declarations)
  - frontend/src/wailsjs/go/main/App.js (added 2 Call() wrappers)
  - frontend/src/wailsjs/go/models.ts (new file)
tech-stack:
  added: []
  patterns:
    - "Wails event emission lives in app.go ONLY (Pitfall #2 — daemon process has no Wails runtime context)"
    - "EventsEmit fires AFTER daemon RPC succeeds; failed Set() short-circuits via `return err` BEFORE the emit (T-92-08 mitigation)"
    - "Hand-maintained Wails TS stubs: project's App.d.ts/App.js use Call() runtime bridge convention (not window['go'][...]); models.ts pinned in-repo with auto-generator output"
key-files:
  created:
    - frontend/src/wailsjs/go/models.ts
  modified:
    - app.go
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
key-decisions:
  - "Pinned the wails-generated models.ts in-repo (under the project's existing wailsjs/go/ tree) rather than running 'wails generate module' on every build. Reason: project already maintains hand-edited App.d.ts/App.js stubs with the Call()-based convention (different from the auto-generated window['go'][...] convention). Replacing them wholesale would break the existing test stub aliasing in vite.config.ts and would lose hand-maintained inline type definitions (DetectedCLI, RemoteSession, IssueCapabilitiesResponse, anonymous return types) that downstream callers import from '../wailsjs/go/main/App'. Surgical addition of just the new symbols + a fresh models.ts mirrors how prior team additions (SET-04, SESS-01/02, APP-01/02) were merged."
  - "PluginSettings ships as the auto-generator's full class form (createFrom + constructor + JSON.parse pathway) rather than a bare interface. Reason: matches what 'wails generate module' produces verbatim, so future regeneration is a clean diff with zero conflicts. Plan 92-03 can import either as a value (new daemon.PluginSettings({})) or as a type (Promise<daemon.PluginSettings>)."
  - "Did NOT replace the project's existing App.d.ts/App.js stubs with the wails-generated versions, despite the auto-generated header saying 'DO NOT EDIT'. The in-repo stubs are the project's effective hand-maintained truth (Mar/Apr 2026 timestamps, prior phase additions); 'wails generate module' actually creates a parallel tree at frontend/src/wailsjs/wailsjs/... which the team historically discards."
requirements-completed:
  - PLUG-03
duration: "12 min"
completed: "2026-05-04"
---

# Phase 92 Plan 02: GUI-side Plugin Settings (App Bindings + Wails TS) Summary

Two Wails App bindings (`GetPluginSettings`, `SetPluginSettings`) on `app.go` that delegate to the DaemonClient, plus the `runtime.EventsEmit("settings:plugins", s)` broadcast that drives prop propagation to all open TerminalPanels (Plan 92-03's React subscription is the consumer half). Wails TS bindings regenerated and pinned: `App.d.ts` declares the two new methods, `models.ts` (new file) exposes `daemon.PluginSettings` for Plan 92-03's frontend imports.

**Duration:** ~12 min · **Started:** 2026-05-04T17:30Z (approx) · **Completed:** 2026-05-04 · **Tasks:** 2/2 · **Files:** 4 (1 created, 3 modified) · **Commits:** 2

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add (*App).GetPluginSettings + SetPluginSettings + settings:plugins EventsEmit | `028f0b4` | `app.go` |
| 2 | Regenerate Wails TS bindings + verify daemon.PluginSettings type surfaced | `087e29a` | `frontend/src/wailsjs/go/main/App.d.ts`, `frontend/src/wailsjs/go/main/App.js`, `frontend/src/wailsjs/go/models.ts` (new) |

## Implementation Anchors

### app.go: GetPluginSettings + SetPluginSettings (Pattern: Wails RPC binding + EventsEmit)

Inserted directly below `SetAutoCloseSession` (line 454+). Mirrors the existing `Get/SetStartMinimized` analog at lines 415-432, with the EventsEmit-on-success addition modeled after the `session:exit` precedent at line 302.

```go
func (a *App) GetPluginSettings() daemon.PluginSettings {
    if a.client == nil {
        return daemon.PluginSettings{}
    }
    s, err := a.client.GetPluginSettings()
    if err != nil {
        return daemon.PluginSettings{}
    }
    return s
}

func (a *App) SetPluginSettings(s daemon.PluginSettings) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    if err := a.client.SetPluginSettings(s); err != nil {
        return err
    }
    runtime.EventsEmit(a.ctx, "settings:plugins", s)
    return nil
}
```

Critical contract: the `return err` line short-circuits the function on RPC failure BEFORE the emit fires. A consumer that subscribes via `EventsOn("settings:plugins", ...)` will only ever observe values that the daemon successfully persisted (T-92-08 mitigation).

### Wails TS bindings (regenerated via `wails generate module`, pinned in-repo)

Generation command that produced the canonical type shapes:

```bash
$(go env GOPATH)/bin/wails generate module
# wails v2.10.2 — produced output at frontend/src/wailsjs/wailsjs/{go/main/App.d.ts, go/models.ts, runtime/*}
```

The auto-generator wrote into a parallel `wailsjs/wailsjs/` tree (because the project's `wailsjsdir: ./frontend/src/wailsjs` is itself the wailsjs root). I extracted just the new symbols and merged them into the project's existing `wailsjs/go/` tree, then deleted the generator's nested `wailsjs/wailsjs/go/` and `wailsjs/wailsjs/runtime/` outputs (the runtime/ files were already tracked and unchanged so I restored them via `git checkout HEAD --`).

**For Plan 92-03 consumers:** the canonical import path is

```typescript
import type { daemon } from '../wailsjs/go/models'
// then use: daemon.PluginSettings
// OR for value construction:
import { daemon } from '../wailsjs/go/models'
const s = new daemon.PluginSettings({ webgl: true, ... })
```

`App.d.ts` itself imports `daemon` from `'../models'` (relative to `wailsjs/go/main/`) so the function signatures resolve correctly.

### Pitfall #2 invariant: zero EventsEmit in internal/daemon/

```bash
$ grep -rn 'runtime\.EventsEmit' internal/daemon/
# (no output)
```

The daemon process is a separate runtime from the Wails GUI; `runtime.EventsEmit` requires the Wails-managed `a.ctx` which only exists in `app.go`. All 9 EventsEmit sites in this repo (8 pre-existing + 1 new) live in `app.go`. This invariant is the architectural backbone of PLUG-03's event propagation.

## Verification

- `grep -c '^func (a \*App) GetPluginSettings()' app.go` → 1 — **OK**
- `grep -c '^func (a \*App) SetPluginSettings(s daemon.PluginSettings)' app.go` → 1 — **OK**
- `grep -q 'runtime\.EventsEmit(a\.ctx, "settings:plugins"' app.go` → 0 (found) — **OK**
- Order check: `awk '/SetPluginSettings/,/^}/' app.go | grep -nE 'a\.client\.SetPluginSettings|runtime\.EventsEmit'` → client call at line 5, EventsEmit at line 8 (client BEFORE emit) — **OK**
- Error short-circuit: `awk '/SetPluginSettings/,/^}/' app.go | grep -q 'return err'` → 0 — **OK**
- `grep -rn 'runtime\.EventsEmit' internal/daemon/ 2>/dev/null` → no matches — **OK** (Pitfall #2 guard)
- `go build .` (root package) → exit 0 — **OK**
- `go vet .` → exit 0 — **OK**
- `go test ./internal/daemon/... -count=1` → PASS (6.617s; full Plan 92-01 suite still green) — **OK**
- `frontend/src/wailsjs/go/main/App.d.ts` declares `GetPluginSettings(): Promise<daemon.PluginSettings>` and `SetPluginSettings(arg1: daemon.PluginSettings): Promise<void>` — **OK**
- `frontend/src/wailsjs/go/main/App.js` exports `GetPluginSettings`/`SetPluginSettings` Call() wrappers — **OK**
- `frontend/src/wailsjs/go/models.ts` declares `daemon.PluginSettings` class with all 8 boolean fields (`webgl`, `unicode11`, `search`, `webLinks`, `image`, `serialize`, `clipboard`, `progress`) — **OK**
- `pnpm exec tsc --noEmit` reports zero errors specific to PluginSettings/App.d.ts/models.ts (pre-existing baseline warnings on `wailsjs/wailsjs/runtime/runtime` import paths are out of scope; logged in `deferred-items.md`) — **OK**

### Frontend test suite

`pnpm test` reports 485 passing / 20 failing tests (`Sidebar.test.tsx` only). Verified pre-existing on baseline `main@028f0b4` (pre-92-02). Documented in `.planning/phases/92-plugin-settings-foundation/deferred-items.md`. **OK** — no 92-02 regression.

## Deviations from Plan

### [Rule 3 — Blocker] `wails generate module` placed output at wrong path

- **Found during:** Task 2
- **Issue:** The plan prescribed running `wails generate module` and then verifying that `frontend/src/wailsjs/go/main/App.d.ts` and `frontend/src/wailsjs/go/models.ts` were updated. In practice, Wails v2.10.2 reads `wailsjsdir: ./frontend/src/wailsjs` from `wails.json` and treats THAT as the wailsjs root, then generates into a `wailsjs/` subdirectory of it — so the actual output landed at `frontend/src/wailsjs/wailsjs/go/...`, not `frontend/src/wailsjs/go/...`.
- **Fix:** Used the auto-generated output as authoritative for the new symbols (`daemon.PluginSettings` class shape + `GetPluginSettings`/`SetPluginSettings` declarations), then surgically merged just the new symbols into the project's existing `frontend/src/wailsjs/go/main/{App.d.ts, App.js}` (preserving the project's hand-maintained `Call()`-based convention and inline type definitions used by downstream callers). Created `frontend/src/wailsjs/go/models.ts` with the auto-generated `daemon.PluginSettings` class verbatim. Deleted the generator's nested `frontend/src/wailsjs/wailsjs/go/` output and restored the runtime/ files (which had been deleted by the generator) via `git checkout HEAD --`.
- **Files modified:** `frontend/src/wailsjs/go/main/App.d.ts` (added `import { daemon } from '../models'` at top + 2 function declarations), `frontend/src/wailsjs/go/main/App.js` (added 2 `Call()` wrappers), `frontend/src/wailsjs/go/models.ts` (created).
- **Verification:** `tsc --noEmit` clean for new code; `grep` acceptance criteria all pass; commit `087e29a` lands the change.

### Total deviations

**1 auto-fixed** (Rule 3 — Blocker around generator-output path mismatch). **Impact:** None to the user-facing API or contract. The two App bindings, the runtime event name, and the TS import path that Plan 92-03 will use (`'../wailsjs/go/models'`) all match the plan's frontmatter `must_haves` and `key_links` exactly.

## Authentication Gates

None — entirely local Go + TS work.

## Threat Flags

None — no new attack surface beyond the threat model in `92-02-PLAN.md`. The Wails-marshalling boundary (T-92 trust boundaries) is unchanged: `(*App).SetPluginSettings` accepts a typed `daemon.PluginSettings` argument; the daemon HTTP layer's `DisallowUnknownFields()` (Plan 92-01) still rejects schema-poisoning attempts at the only network boundary.

## Self-Check: PASSED

- File checks (existence):
  - `frontend/src/wailsjs/go/models.ts` — FOUND
  - `frontend/src/wailsjs/go/main/App.d.ts` — FOUND (modified, contains GetPluginSettings + SetPluginSettings)
  - `frontend/src/wailsjs/go/main/App.js` — FOUND (modified, contains Call wrappers)
  - `app.go` — FOUND (modified, contains both bindings + EventsEmit)
- Commit checks (`git log --oneline --all | grep`):
  - `028f0b4` — FOUND (Task 1)
  - `087e29a` — FOUND (Task 2)
- Verification commands re-run after writing this SUMMARY:
  - All 8 grep acceptance criteria — PASS
  - Plan-level verification block — PASS
  - `go test ./internal/daemon/...` — PASS

## Ready for 92-03

Plan 03 (Wave 3, frontend) consumes:

- **TS type:** `import { daemon } from '../wailsjs/go/models'` then `daemon.PluginSettings` for both type annotations and value construction. The class supports `new daemon.PluginSettings({...})` for cases where Plan 03 needs to hand-build a settings object before send.
- **Wails bindings:** `import { GetPluginSettings, SetPluginSettings } from '../wailsjs/go/main/App'` — both return `Promise<...>`. `GetPluginSettings()` resolves with the daemon's current state (or zero-valued struct on disconnect — UI MUST gate on a `pluginsLoaded` boolean per Pitfall #3). `SetPluginSettings(s)` resolves with no value on success; rejects with the daemon error on failure.
- **Wails event:** `EventsOn('settings:plugins', (s: daemon.PluginSettings) => setPluginConfig(s))` in `App.tsx` (PLUG-03 propagation half). The event payload is the FULL `PluginSettings` struct (not a sentinel) — App.tsx sets `pluginConfig` state directly without re-fetching.
- **Architectural invariant for Plan 03 testing:** any new EventsEmit calls MUST live in `app.go`, never in `internal/daemon/`. The vitest source-inspection tests in Plan 03 should grep `app.go` (relative path `../../../app.go` from `frontend/src/components/__tests__/`) to assert this.
