# Phase 92: Plugin Settings Foundation — Research

**Researched:** 2026-05-04
**Domain:** Daemon settings persistence + Wails RPC + Wails runtime event propagation + React Settings UI shell
**Confidence:** HIGH

## Summary

Phase 92 lays the foundation pipeline for v3.2's xterm.js plugin suite. It ships **persistence, propagation, and the Settings UI shell** end-to-end — but **no addon code consumes the new `pluginConfig` prop in this phase**. Every requirement is achievable by extending established v3.1 patterns; no new architectural mechanism is required.

The four requirements (PLUG-01, PLUG-02, PLUG-03, PUI-01) decompose into three implementation surfaces with strict boundaries:

1. **Daemon (Go):** extend `daemonSettings` with a `Plugins *PluginSettings` pointer + `SchemaVersion int`; add a `defaultPluginSettings()` constructor; route load through a **defaults-merge** pattern (NOT naive `json.Unmarshal`) to neutralize Pitfall #14 (zero-value plugin defaults on v3.1 upgrade); add `GetPluginSettings()` / `SetPluginSettings()` engine methods + matching API routes + DaemonClient methods; emit `runtime.EventsEmit(ctx, "settings:plugins", newConfig)` from `app.go` after every successful Set.
2. **Frontend (React):** new `PluginsSection.tsx` + `types/plugins.ts`; insert into `SettingsTab.tsx` after the Paths section; `App.tsx` subscribes to `EventsOn("settings:plugins", ...)`, holds `pluginConfig` state, threads as a prop into every `<TerminalPanel>`; `TerminalPanel.tsx` accepts `pluginConfig?: PluginSettings` on its props interface but does **not** consume it (that wiring lives in Phase 93).
3. **Migration test (Go):** `tests/fixtures/settings_v3.1.json` + a `TestSettingsMigrationV3_1ToV3_2` test that loads the fixture through `loadSettingsFromDisk`, asserts all 8 plugin defaults are populated correctly (7 ON, Progress OFF), asserts `schemaVersion: 2` is written, and asserts idempotency on a second load.

**Primary recommendation:** Execute as 3-4 small parallel-friendly plans — daemon Go work, frontend React work, migration fixture+test, integration glue. The hardest single decision (defaults-merge over naive Unmarshal) is non-negotiable and its test is the phase's load-bearing CI gate. Everything else is mechanical extension of v3.1 patterns.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Plugin settings persistence (read/write `settings.json`) | Daemon (Go) | — | Already the source of truth for `cliPaths`, `startMinimized`, `autoCloseSession` — extending the same struct keeps one file, one mutex, one mechanism. |
| Plugin defaults seeding (returning v3.1 user gets sensible values) | Daemon (Go) | — | Defaults must be applied during load, NOT in the frontend, so a fresh daemon read always reflects truth regardless of which client (Wails GUI or future web Settings UI) reads first. |
| Wails RPC (`GetPluginSettings` / `SetPluginSettings`) | API / Backend (`app.go` + `internal/daemon/api.go` + `client.go`) | — | Identical pattern to existing `GetStartMinimized` / `SetStartMinimized` triple. |
| Wails runtime event emission (`settings:plugins`) | API / Backend (`app.go`) | — | Wails `runtime.EventsEmit` runs in the desktop process (the GUI side); the daemon HTTP API does not have access to the Wails context. The flow is GUI sets via `app.go`, app.go calls `client.SetPluginSettings`, then `app.go` emits the event after success. |
| Cross-component React state propagation (subscribe + thread prop) | Browser / Client (`App.tsx`) | — | Existing precedent: `tray:focus-session`, `session:exit`, `app:quit-requested` are all `EventsOn` in `App.tsx`. |
| Settings UI rendering (Plugins h3 + 8 toggle rows + Save button) | Browser / Client (`PluginsSection.tsx`) | — | Pure presentation; reuses every existing `.settings-panel__*` class. |
| TerminalPanel prop acceptance (`pluginConfig?` in props interface) | Browser / Client (`TerminalPanel.tsx`) | — | Type-only addition in Phase 92. Consumption is Phase 93 territory. |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PLUG-01 | User's plugin enable/disable choices and per-plugin config persist via the existing daemon `settings.json` mechanism, surviving GUI restarts and daemon restarts | §1 (existing settings persistence model in `engine.go`); §3 (daemon `PluginSettings` struct extension + defaults-merge load); §6 (round-trip persistence test) |
| PLUG-02 | A returning user upgrading from v3.1 lands on v3.2 with sensible plugin defaults populated (no zero-value addons-disabled state, no zero-value `storageLimit`); v3.1 `settings.json` files migrate cleanly with `schemaVersion: 2` written | §3 (defaults-merge pattern, NOT naive `json.Unmarshal`); §5 (Pitfall #14 from v3.2 PITFALLS.md); §6 (`tests/fixtures/settings_v3.1.json` + `TestSettingsMigrationV3_1ToV3_2`) |
| PLUG-03 | Plugin state changes propagate from Settings save to all open desktop terminals via a Wails runtime event without requiring an app restart | §4 (existing `runtime.EventsEmit` pattern: 8 events in `app.go` already; `EventsOn` in `App.tsx` with prior precedents `tray:focus-session`, `session:exit`); §7 (test asserts event fires + App-state updates + prop threads) |
| PUI-01 | Settings tab includes a "Plugins" section listing all v3.2 plugins (WebGL, Unicode 11, Search, Web Links, Inline Images, Serialize, Clipboard, Progress) with name, short description, and an enable/disable toggle each | UI-SPEC §"Component Inventory" + §"Per-row layout"; existing `.settings-panel__toggle-row` precedent (verified in `SettingsTab.tsx` lines 337-361 for `Behavior`, 365-391 for `Session Behavior`); §6 (8 hardcoded rows in fixed order matching UI-SPEC) |
</phase_requirements>

## Standard Stack

### Core (verified in repo, no new dependencies in Phase 92)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/xterm` | 6.0.0 | Terminal core (already shipping; not modified in Phase 92) | Already locked since v3.1; no addon work in this phase. |
| React | 19.2.4 | UI framework | Already locked; v3.1 codebase. |
| Wails v2 | (Go module, version pinned in `go.mod`) | Desktop shell + RPC binding generation + runtime events | Already locked. |
| vitest | 4.1.0 | Frontend test framework | `[VERIFIED: frontend/package.json]` — test script `vitest run`; `?raw` source-inspection precedent (`SettingsTab.persistence.test.tsx`). |
| Go testing | stdlib | Daemon test framework | `[VERIFIED: internal/daemon/engine_settings_test.go]` exists and uses `t.TempDir()` + temp config dir + reload-engine pattern. |

**No new pnpm packages, no new Go modules.** `[VERIFIED: STACK.md]` and `[VERIFIED: frontend/package.json read 2026-05-04]` — Phase 92 ships zero new dependencies. The 8-plugin Settings UI is just toggle rows; the propagation pipeline is `runtime.EventsEmit` (already used 8 times in `app.go`).

### Supporting (existing patterns to reuse verbatim)

| Pattern | Source File | Purpose |
|---------|-------------|---------|
| `daemonSettings` struct + `loadSettingsFromDisk` + `saveSettingsToDisk` | `internal/daemon/engine.go:66-133` | Add `Plugins *PluginSettings` field; modify load to merge over defaults |
| `Get<X>()` / `Set<X>(v) error` Wails binding pair | `app.go:415-432` (`GetStartMinimized` / `SetStartMinimized`) | Add `GetPluginSettings` / `SetPluginSettings` |
| HTTP route registration via `a.mux.HandleFunc` | `internal/daemon/api.go:68-73` | Add `GET /settings/plugins` + `PUT /settings/plugins` |
| DaemonClient method via `c.doJSON` | `internal/daemon/client.go:111-139` | Add `GetPluginSettings` / `SetPluginSettings` (using HTTP PUT, mirroring SetStartMinimized's PATCH style) |
| `runtime.EventsEmit(ctx, eventName, payload)` in `app.go` | 8 existing emits at lines 102, 176, 267, 302, 356, 795, 818, 861 | Emit `settings:plugins` after `client.SetPluginSettings` succeeds |
| `EventsOn(eventName, callback)` subscription in `App.tsx` | 6 existing subscriptions at lines 250, 257, 309, 313, 323, 384, 615 | Add `EventsOn("settings:plugins", ...)` |
| Source-inspection vitest test | `frontend/src/components/__tests__/SettingsTab.persistence.test.tsx` | Use `?raw` import + `expect(raw).toContain('...')` for jsdom-incompatible assertions |
| Three-state Save button | `SettingsTab.tsx:695-703` (`Save Paths`) + state hooks lines 65-67 | Mirror exactly for `Save Plugins` |
| `toggleLoaded` flicker-prevention guard | `SettingsTab.tsx:338-349` (`Behavior` toggle) | Apply identical pattern for the 8 plugin toggles (`{toggleLoaded && (...)}`) |

**Installation:** none. `[VERIFIED: frontend/package.json]` — all needed packages are already present.

**Version verification:** N/A. No new packages introduced in Phase 92 — version verification is deferred to Phase 93 where the new `@xterm/addon-*` packages are added.

### Alternatives Considered

| Instead of | Could Use | Tradeoff | Verdict |
|------------|-----------|----------|---------|
| Defaults-merge constructor in `loadSettingsFromDisk` | `omitempty` + Go zero values | Zero values yield disabled plugins on v3.1 upgrade — Pitfall #14 | **Defaults-merge mandatory** |
| Single `GetPluginSettings`/`SetPluginSettings` pair | 16 per-toggle Get/Set methods | Frontend would need 8 round-trips on Save; no batched commit; doesn't match three-state Save button UX | **Single struct round-trip** |
| Wails `EventsEmit` with full struct payload | `EventsEmit` with sentinel "changed" + frontend re-fetches via `GetPluginSettings` | Fewer races (always read fresh from daemon) but extra round-trip | **Emit full payload** — matches existing `session:exit` and `tailscale:health` payload patterns; Wails runtime serializes payload as JSON |
| HTTP PUT route | HTTP PATCH route | Existing settings routes use PATCH (`/settings/start-minimized`) | **PATCH** for consistency, even though we send the entire `PluginSettings` struct |
| Per-tab `pluginConfig` from per-session daemon state | Global `pluginConfig` (chosen) | Per-tab adds combinatorial UI complexity; STATE.md `## Decisions` says "Server-shared plugin config" + "global theme precedent" | **Global** |

## Architecture Patterns

### System Architecture Diagram

```
                         ┌──────────────────────────┐
                         │ User toggles 8 plugin    │
                         │ rows + clicks Save       │
                         └────────────┬─────────────┘
                                      │
                                      ▼
   ┌────────────────────────────────────────────────────────────────┐
   │ PluginsSection.tsx (NEW)                                       │
   │  - Holds local edited state (8 booleans)                       │
   │  - On Save: SetPluginSettings(state) via Wails binding         │
   │  - Three-state Save button: idle → "Saving…" → "Saved!" 1.5s   │
   └────────────────────┬───────────────────────────────────────────┘
                        │ Wails RPC (window.go.main.App.SetPluginSettings)
                        ▼
   ┌────────────────────────────────────────────────────────────────┐
   │ app.go (MODIFIED)                                              │
   │  func (a *App) SetPluginSettings(s daemon.PluginSettings) err  │
   │   1. err := a.client.SetPluginSettings(s)                      │
   │   2. if err == nil:                                            │
   │      runtime.EventsEmit(a.ctx, "settings:plugins", s)          │
   └────────────────────┬───────────────────────────────────────────┘
                        │ HTTP PATCH /settings/plugins via Unix socket / pipe
                        ▼
   ┌────────────────────────────────────────────────────────────────┐
   │ internal/daemon/client.go (MODIFIED)                           │
   │  c.doJSON(http.MethodPatch, "/settings/plugins", &s, nil)      │
   └────────────────────┬───────────────────────────────────────────┘
                        │
                        ▼
   ┌────────────────────────────────────────────────────────────────┐
   │ internal/daemon/api.go (MODIFIED)                              │
   │  PATCH /settings/plugins → handleSetPluginSettings             │
   │   1. json.Decode body into PluginSettings                      │
   │   2. e.SetPluginSettings(s)                                    │
   │   3. 204 No Content                                            │
   └────────────────────┬───────────────────────────────────────────┘
                        │
                        ▼
   ┌────────────────────────────────────────────────────────────────┐
   │ internal/daemon/engine.go + plugin_settings.go (MODIFIED+NEW)  │
   │  e.SetPluginSettings(s):                                       │
   │   1. e.mu.Lock(); e.pluginSettings = s                         │
   │   2. e.saveSettingsToDisk() (under lock, atomic write)         │
   │   3. e.mu.Unlock()                                             │
   │  saveSettingsToDisk now writes:                                │
   │    {"cliPaths":..., "startMinimized":..., "autoCloseSession":  │
   │     ..., "plugins":{...}, "schemaVersion": 2}                  │
   └────────────────────────────────────────────────────────────────┘

   ───────────────── Event back-propagation (live) ─────────────────

   ┌────────────────────────────────────────────────────────────────┐
   │ App.tsx (MODIFIED)                                             │
   │  EventsOn("settings:plugins", (s: PluginSettings) =>           │
   │    setPluginConfig(s)                                          │
   │  )                                                             │
   │  Initial state: GetPluginSettings() on mount                   │
   └────────────────────┬───────────────────────────────────────────┘
                        │ pluginConfig prop
                        ▼
   ┌────────────────────────────────────────────────────────────────┐
   │ TerminalPanel.tsx (MODIFIED — type-only)                       │
   │  Props gain `pluginConfig?: PluginSettings`                    │
   │  Phase 92 does NOT consume the prop (no useEffect, no addon    │
   │  load gating). The wire is real (test asserts the prop is      │
   │  passed) but inert. Phase 93 wires consumption.                │
   └────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/daemon/
├── engine.go                    # MODIFIED — add Plugins field + defaults merge
├── plugin_settings.go           # NEW — PluginSettings struct, defaults constructor, schema version constant
├── plugin_settings_test.go      # NEW — defaults-merge unit tests (no fixture file)
├── engine_plugins_test.go       # NEW — engine round-trip + Set/Get + persistence
├── engine_migration_test.go     # NEW — fixture-based v3.1→v3.2 migration test (CI gate per ROADMAP SC-1)
├── api.go                       # MODIFIED — 2 new HTTP routes
└── client.go                    # MODIFIED — 2 new RPC methods

tests/fixtures/                  # NEW directory (not yet present)
└── settings_v3.1.json           # NEW — golden v3.1-shape settings.json (no plugins, no schemaVersion)

app.go                           # MODIFIED — GetPluginSettings + SetPluginSettings + EventsEmit

frontend/src/
├── types/
│   └── plugins.ts               # NEW — TS mirror of Go PluginSettings
├── components/
│   ├── PluginsSection.tsx       # NEW — 8-toggle Plugins section
│   ├── SettingsTab.tsx          # MODIFIED — insert <PluginsSection /> after Paths
│   ├── TerminalPanel.tsx        # MODIFIED — add pluginConfig?: prop (type only, unused)
│   └── __tests__/
│       └── PluginsSection.test.tsx     # NEW — source-inspection (visual contract from UI-SPEC)
└── App.tsx                      # MODIFIED — pluginConfig state + EventsOn + prop drilling

frontend/src/__tests__/          # MAYBE NEW
└── App.plugin-event.test.tsx    # NEW — asserts EventsOn("settings:plugins") subscription wires
```

### Pattern 1: Defaults-Merge Settings Load (CRITICAL — load-bearing for PLUG-02)

**What:** Replace `var s daemonSettings; json.Unmarshal(data, &s)` with a defaults-populated zero-value, then unmarshal-on-top so the user's saved fields override defaults but missing fields keep their default values (NOT Go zero values).

**When to use:** Every time we add a new field to `daemonSettings` that has a non-zero "intended default."

**Example (DO):**
```go
// internal/daemon/engine.go - loadSettingsFromDisk modified
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
    data, err := os.ReadFile(settingsPath(dir))

    // Start with defaults — this is the load-bearing change.
    s := daemonSettings{
        SchemaVersion: CurrentSchemaVersion, // 2
        Plugins:       defaultPluginSettings(),
    }

    if err == nil {
        // Unmarshal user-saved fields ON TOP OF defaults.
        // Fields present in JSON overwrite; missing fields keep defaults.
        // This is the ONLY safe way to add fields without zeroing existing users.
        if err := json.Unmarshal(data, &s); err != nil {
            return // corrupt file — keep defaults, don't crash
        }
    }
    // ... rest of load logic ...

    // Detect upgrade-path (v3.1 file had no schemaVersion, no plugins) and
    // re-write so future loads observe schemaVersion: 2 + plugins block.
    needsRewrite := s.SchemaVersion < CurrentSchemaVersion
    if needsRewrite {
        s.SchemaVersion = CurrentSchemaVersion
        e.saveSettingsToDisk() // idempotent on second start
    }
}
```

**Source:** `[CITED: PITFALLS.md Pitfall #14]` — exactly this pattern. `[VERIFIED: engine.go:88-117]` — current load is naive `json.Unmarshal`. `[ASSUMED]` — Go's `json.Unmarshal` semantics: existing struct fields are NOT cleared before unmarshal; missing JSON keys leave struct fields untouched. This is documented Go stdlib behavior since 1.0; confidence is HIGH but flagged as ASSUMED because it's the load-bearing assumption of the entire pattern. **Verify in plan with a tiny test before relying on it.**

**Anti-pattern (DO NOT):**
```go
// Naive — produces zero values for missing keys
var s daemonSettings
json.Unmarshal(data, &s)  // s.Plugins is nil, all bools false → all addons disabled
```

### Pattern 2: Wails RPC Triple — engine method + API route + DaemonClient method + app.go binding

**What:** Settings additions follow a strict 4-file pattern. Skipping any layer breaks either the daemon API or the GUI.

**When to use:** Every new settings field that the GUI needs to read/write.

**Example sources to mirror exactly:**
- Engine: `engine.go:378-406` (`GetStartMinimized` / `SetStartMinimized`)
- API route: `api.go:68-73` (registration) + `474-502` (handlers)
- DaemonClient: `client.go:111-139`
- app.go: `415-432` (`GetStartMinimized` / `SetStartMinimized`)

For Phase 92 the same 4-file pattern produces:

```go
// engine.go
func (e *SessionEngine) GetPluginSettings() PluginSettings {
    e.mu.Lock(); defer e.mu.Unlock()
    return *e.pluginSettings // pluginSettings is *PluginSettings; never nil after load
}
func (e *SessionEngine) SetPluginSettings(s PluginSettings) {
    e.mu.Lock()
    e.pluginSettings = &s
    e.saveSettingsToDisk()
    e.mu.Unlock()
}

// api.go
a.mux.HandleFunc("GET /settings/plugins", a.handleGetPluginSettings)
a.mux.HandleFunc("PATCH /settings/plugins", a.handleSetPluginSettings)

// client.go
func (c *DaemonClient) GetPluginSettings() (PluginSettings, error) { ... }
func (c *DaemonClient) SetPluginSettings(s PluginSettings) error  { ... }

// app.go — and HERE is the only place EventsEmit fires
func (a *App) SetPluginSettings(s daemon.PluginSettings) error {
    if err := a.client.SetPluginSettings(s); err != nil { return err }
    runtime.EventsEmit(a.ctx, "settings:plugins", s)
    return nil
}
```

**Source:** `[VERIFIED: app.go grep for EventsEmit returns 8 lines]` — every existing emit happens in `app.go`, not in `internal/daemon/`. The daemon process has no Wails runtime context — only the GUI process (which embeds `app.go`) does. This is **load-bearing** for PLUG-03: emit must fire from `app.go`'s Set wrapper, after the daemon RPC returns success.

### Pattern 3: `EventsOn` subscription with cleanup in App.tsx

**What:** New runtime events subscribe in `App.tsx`'s top-level `useEffect`(s) with `[]` deps; the returned `off` function gets pushed into the cleanup return.

**When to use:** Every new Wails runtime event that drives global App state.

**Example pattern from `[VERIFIED: App.tsx:250-323]`:**
```ts
useEffect(() => {
  const offStatus = EventsOn('session:status', (data) => { ... })
  const offHealth = EventsOn('tailscale:health', (h) => { ... })
  const offDaemonError = EventsOn('daemon:error', (msg) => { ... })
  const cancelTrayFocus = EventsOn('tray:focus-session', (sessionId) => { ... })
  const offExit = EventsOn('session:exit', (data) => { ... })

  return () => {
    offStatus(); offHealth(); offDaemonError(); cancelTrayFocus(); offExit()
  }
}, [])
```

For Phase 92, add (at the same indentation level inside the same `useEffect`):
```ts
const offPlugins = EventsOn('settings:plugins', (s: PluginSettings) => {
  setPluginConfig(s)
})
// ... add offPlugins() to the cleanup return
```

### Pattern 4: `toggleLoaded` flicker-prevention pattern

**What:** Hide toggle rows until the daemon's initial `GetPluginSettings()` resolves, then show all 8 simultaneously. Prevents the "checkbox flashes off, then flips on" jitter.

**When to use:** Any toggle whose `checked` state comes from an async daemon read.

**Example from `[VERIFIED: SettingsTab.tsx:88-89, 338-349]`:**
```tsx
const [toggleLoaded, setToggleLoaded] = useState(false)
// ... useEffect: GetStartMinimized().then(v => { setStartMinimized(v); setToggleLoaded(true) })
{toggleLoaded && (
  <label className={`settings-panel__toggle-row${startMinimized ? ' settings-panel__toggle-row--checked' : ''}`} ...>
    ...
  </label>
)}
<input type="checkbox" id="..." className="settings-panel__toggle-input" checked={startMinimized} ... />
```

For Phase 92's PluginsSection: a single `pluginsLoaded` boolean gates all 8 rows simultaneously (one round-trip fills all 8 toggle states from the returned `PluginSettings` struct).

### Anti-Patterns to Avoid

- **`omitempty` on plugin keys:** Hides fields from the on-disk `settings.json`, makes user inspection harder, makes future migrations harder. `[CITED: PITFALLS.md #14]` says "Never use `omitempty` on plugin keys". Apply to top-level `Plugins` and `SchemaVersion` only — sub-fields like `Search.DefaultRegex` may still use `omitempty` for forward compatibility.
- **Emitting `settings:plugins` from inside `internal/daemon/`:** Daemon has no Wails runtime context. Emit only in `app.go` after `client.SetPluginSettings` succeeds.
- **Naive `json.Unmarshal` after adding the new struct field:** Pitfall #14 — would zero plugin defaults for every returning v3.1 user.
- **Auto-save on toggle:** UI-SPEC explicitly forbids; PUI-04 mandates the three-state Save button (Phase 99 requirement, but the pattern is locked in Phase 92 for consistency).
- **Per-tab plugin config:** STATE.md `## Decisions` chose global; per-tab is out of scope for v3.2.
- **Consuming `pluginConfig` inside `TerminalPanel`'s `useEffect([sessionId])` in Phase 92:** Phase 93 territory. Phase 92's contract is "the prop is threaded but inert."
- **Skipping the `toggleLoaded` guard:** Will produce a 200-300ms flash where all 8 toggles render off, then snap to the saved state. UI-SPEC §"Empty / loading state" mandates the guard.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-process settings change notification | Polling timer in `App.tsx` | `runtime.EventsEmit` + `EventsOn` (already used 8 times) | Polling adds latency + load; events are immediate and idiomatic Wails. |
| Atomic JSON file write | `os.WriteFile` directly | The existing `saveSettingsToDisk` (`[VERIFIED: engine.go:122-133]` — already uses single `os.WriteFile` under mutex) | The existing pattern is what `cliPaths`/`startMinimized` use. Don't introduce a new write path. |
| Settings schema versioning | Bespoke "if old → migrate" branches | Single `SchemaVersion int` field + defaults-merge load + idempotent re-save on upgrade | One-shot migration via re-save; `schemaVersion: 2` is the only sentinel. Future v3.3 bumps to 3 with the same mechanism. |
| TS type for `PluginSettings` | Hand-write to mirror Go | Hand-write IS the right tool here — Wails v2 generates TS types from Go in `frontend/src/wailsjs/go/main/App.d.ts`, but only for types referenced from `App` methods. Add `(*App).GetPluginSettings()` and the type appears automatically. | `[VERIFIED: SettingsTab.tsx:21]` already uses `import type { DetectedCLI } from '../wailsjs/go/main/App'` — same machinery. Don't duplicate type by hand. |
| Three-state Save button | New CSS classes | Reuse `.settings-panel__btn--save` / `--saved` and the `saving`/`saved` state pattern | Already locked in v3.0. Reuse verbatim. |
| Toggle row visual | New CSS for "Plugins-only" toggles | Reuse `.settings-panel__toggle-row` + `.settings-panel__toggle-track` + `.settings-panel__toggle-thumb` + `.settings-panel__toggle-label` | UI-SPEC §"Component Inventory" says "Phase 92 adds zero new lines to `style.css`." Verified-load-bearing. |

**Key insight:** Phase 92 is **mechanical extension** of three v3.1 patterns (settings persistence, Wails RPC triple, event subscription) plus one new pattern (defaults-merge load) — all of which already have working examples in the codebase. The risk surface is small because no new architectural primitives are introduced. The single load-bearing decision is the defaults-merge load, which is captured by the migration test.

## Runtime State Inventory

> Phase 92 adds new state but does not rename or migrate existing runtime state. The migration concern is **zero-value defaults**, not stale references. The following inventory is brief because most categories are not applicable.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `<configDir>/settings.json` is rewritten with new top-level keys `plugins` and `schemaVersion` | **Code edit only** — defaults-merge load handles existing users. NO data migration step needed: on first v3.2 launch, load reads old JSON → merges over defaults → re-saves with new fields. Idempotent. |
| Live service config | None — the Wails event name `settings:plugins` is **new** in this milestone, not a rename. No existing event named anything like this. | None |
| OS-registered state | None | None — verified via grep across `app.go`, no Task Scheduler / launchd / systemd integrations touch settings file |
| Secrets/env vars | None — `PluginSettings` struct contains booleans and (Phase 99 territory) numeric `storageLimit` only; no secrets | None |
| Build artifacts / installed packages | None — Phase 92 adds no pnpm packages and no Go modules | None |

**The canonical question (per researcher protocol):** *After every file in the repo is updated, what runtime systems still have the old string cached, stored, or registered?*

**Answer:** Nothing. Phase 92 introduces `plugins` and `schemaVersion` keys — there is no "old string" being renamed. The migration concern is one-directional: v3.1 settings.json files (without these keys) get the keys added on first v3.2 load via defaults-merge. The migration test verifies this end-to-end.

## Common Pitfalls

### Pitfall 1: Naive `json.Unmarshal` zeroes plugin defaults on v3.1 upgrade

**What goes wrong:** Returning v3.1 user opens v3.2 → `loadSettingsFromDisk` reads old `settings.json` (no `plugins` key) into a fresh `daemonSettings{}` zero value → `s.Plugins == nil` → after assignment, all 8 plugin booleans are `false` → `GetPluginSettings()` returns all-off → terminal looks worse than v3.1.

**Why it happens:** Go's encoding/json leaves struct fields at their pre-unmarshal values when the JSON key is missing. If the struct was zero-initialized, missing keys stay zero. We must pre-populate defaults before Unmarshal so missing keys keep defaults.

**How to avoid:** Defaults-merge pattern in §"Pattern 1" above. The migration test (`tests/fixtures/settings_v3.1.json` + `TestSettingsMigrationV3_1ToV3_2`) is the load-bearing CI gate; if it ever goes red, refuse to merge.

**Warning signs:**
- Test red: `expected pluginSettings.WebGL.Enabled == true, got false` after loading the v3.1 fixture
- Manual smoke: copy a v3.1 `~/.config/agenthub/settings.json` (or platform-equivalent) to a v3.2 dev build, observe Plugins section shows all toggles off

**Source:** `[CITED: PITFALLS.md #14]`

### Pitfall 2: `runtime.EventsEmit` from inside `internal/daemon/`

**What goes wrong:** Plan author tries to emit `settings:plugins` from `engine.go:SetPluginSettings`. Compile error or worse, runtime panic — `runtime.EventsEmit` requires a Wails context, which only exists in the GUI process (rooted at `app.go`).

**Why it happens:** Two-process architecture is not obvious from grepping for `EventsEmit` (all 8 hits are in `app.go`, but a developer skimming `engine.go` won't notice the absence).

**How to avoid:** EventsEmit lives in `app.go` only. Engine `SetPluginSettings` returns; `app.go.SetPluginSettings` calls engine via DaemonClient, then emits on success.

**Warning signs:**
- Plan task says "in engine.go's SetPluginSettings, emit `settings:plugins`" — reject in plan-checker
- Compile error: `undefined: runtime.EventsEmit` in `internal/daemon/`

**Source:** `[VERIFIED: app.go grep — all 8 emits in app.go]`

### Pitfall 3: Settings UI flicker — toggles flash off, then on

**What goes wrong:** `PluginsSection` renders before `GetPluginSettings()` resolves → 8 toggles render with default `false` checked state → 200ms later state arrives → toggles flip to saved values. User sees a visible glitch.

**Why it happens:** React renders before the async fetch resolves; default useState initializer is `false` for booleans.

**How to avoid:** `pluginsLoaded` boolean guard around the toggle markup (mirroring `toggleLoaded` at `[VERIFIED: SettingsTab.tsx:88]`). UI-SPEC §"Empty / loading state" already mandates this.

**Warning signs:**
- Manual: open Settings tab, scroll to Plugins, observe a flash of all-off toggles before they snap to saved state
- Test: a deliberate-delay mock of `GetPluginSettings` should NOT cause toggle markup to render before the promise resolves

**Source:** `[VERIFIED: existing pattern in SettingsTab.tsx]`

### Pitfall 4: TerminalPanel prop addition breaks existing tests

**What goes wrong:** Adding `pluginConfig?: PluginSettings` to `TerminalPanelProps` may trip TS exhaustive-prop tests in `__tests__/TerminalPanel.test.tsx` if any test asserts the props shape exhaustively or constructs the props object without passing `pluginConfig`.

**Why it happens:** Existing tests were written against the v3.1 prop signature.

**How to avoid:** Mark prop optional (`pluginConfig?:`); existing test invocations stay valid. Add one new test: `<TerminalPanel pluginConfig={{...}} />` renders without error and the prop is observable on the component (source-inspection assertion `expect(raw).toContain('pluginConfig')` is sufficient given vitest's jsdom limits with WebGL).

**Warning signs:**
- TS compile fail in `frontend/src/components/__tests__/TerminalPanel.test.tsx`
- vitest red on `App.test.tsx` if it constructs TerminalPanel via props spread

### Pitfall 5: PUI-01 plugin order drifts from UI-SPEC

**What goes wrong:** Implementer hardcodes the 8 toggle rows in alphabetical or arbitrary order. UI-SPEC mandates: WebGL → Unicode 11 → Search → Web Links → Inline Images → Serialize → Clipboard → Progress.

**Why it happens:** The 8-plugin list appears in many places (REQUIREMENTS, UI-SPEC, ROADMAP) but the ordering is only authoritative in UI-SPEC §"Phase Scope" and §"Per-plugin one-sentence descriptions" table.

**How to avoid:** Source-inspection test in `PluginsSection.test.tsx` asserting the 8 toggle rows appear in PluginsSection.tsx in the exact UI-SPEC order — `expect(raw.indexOf('webgl')).toBeLessThan(raw.indexOf('unicode11'))` etc.

### Pitfall 6: Defaults table mismatch between Go and TypeScript

**What goes wrong:** `defaultPluginSettings()` in `plugin_settings.go` says Progress=false, but `PluginsSection.tsx` initial-state hardcodes Progress=true. First-run users see toggles already enabled before daemon load resolves. Migration fixture test passes (Go side) but UI smoke test fails (frontend side).

**Why it happens:** Defaults exist in two places — daemon (load-bearing source of truth) and frontend (only for the brief pre-load window). The frontend's pre-load defaults should not be observable due to `pluginsLoaded` guard, but if the guard is forgotten, the discrepancy surfaces.

**How to avoid:** Combine `pluginsLoaded` guard (Pitfall #3) with a TS test that asserts PluginsSection.tsx does NOT initialize plugin state with hardcoded booleans — it should use empty/null state until `GetPluginSettings()` resolves. Or: state initializer is `null`, and the toggle markup is gated on `pluginConfig != null`.

## Code Examples

Verified patterns from sources read this session.

### Example 1: Existing Wails RPC triple — `GetStartMinimized`

```go
// Source: app.go:415-432 [VERIFIED 2026-05-04]
func (a *App) GetStartMinimized() bool {
    if a.client == nil {
        return false
    }
    val, err := a.client.GetStartMinimized()
    if err != nil {
        return false
    }
    return val
}

func (a *App) SetStartMinimized(val bool) error {
    if a.client == nil {
        return fmt.Errorf("daemon client unavailable")
    }
    return a.client.SetStartMinimized(val)
}
```

```go
// Source: internal/daemon/api.go:474-487 [VERIFIED 2026-05-04]
func (a *API) handleGetStartMinimized(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]bool{"startMinimized": a.engine.GetStartMinimized()})
}

func (a *API) handleSetStartMinimized(w http.ResponseWriter, r *http.Request) {
    var req struct {
        StartMinimized bool `json:"startMinimized"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest); return
    }
    a.engine.SetStartMinimized(req.StartMinimized)
    w.WriteHeader(http.StatusNoContent)
}
```

```go
// Source: internal/daemon/client.go:111-124 [VERIFIED 2026-05-04]
func (c *DaemonClient) GetStartMinimized() (bool, error) {
    var resp map[string]bool
    if err := c.doJSON(http.MethodGet, "/settings/start-minimized", nil, &resp); err != nil {
        return false, err
    }
    return resp["startMinimized"], nil
}

func (c *DaemonClient) SetStartMinimized(val bool) error {
    return c.doJSON(http.MethodPatch, "/settings/start-minimized",
        map[string]bool{"startMinimized": val}, nil)
}
```

For Phase 92, mirror with: `GetPluginSettings() → PluginSettings`, `SetPluginSettings(s PluginSettings) error`, route `/settings/plugins`, full struct in body (not a wrapper map).

### Example 2: Existing toggle markup — `Behavior` section

```tsx
// Source: SettingsTab.tsx:336-361 [VERIFIED 2026-05-04]
<h3>Behavior</h3>
<div className="settings-panel__field-group">
  {toggleLoaded && (
    <label
      className={`settings-panel__toggle-row${startMinimized ? ' settings-panel__toggle-row--checked' : ''}`}
      htmlFor="startMinimized"
      style={toggleSaving ? { pointerEvents: 'none', opacity: 0.6 } : undefined}
    >
      <span className="settings-panel__toggle-track">
        <span className="settings-panel__toggle-thumb" />
      </span>
      <span className="settings-panel__toggle-label">Start minimized to system tray</span>
    </label>
  )}
  <input
    type="checkbox"
    id="startMinimized"
    className="settings-panel__toggle-input"
    checked={startMinimized}
    onChange={() => void handleToggleMinimized()}
  />
  <p className="settings-panel__description">
    When enabled, AgentHub launches with the window hidden. Click the tray icon to open it.
  </p>
  {toggleError && <p className="settings-panel__error">{toggleError}</p>}
</div>
```

For Phase 92 PluginsSection, repeat this row 8 times with the per-plugin labels and descriptions from UI-SPEC §"Per-plugin one-sentence descriptions".

### Example 3: Three-state Save button — `Save Paths`

```tsx
// Source: SettingsTab.tsx:695-703 [VERIFIED 2026-05-04]
<div className="settings-panel__save-paths-row">
  <button
    className={`settings-panel__btn ${saved ? 'settings-panel__btn--saved' : 'settings-panel__btn--save'}`}
    onClick={handleSaveCLIPaths}
    disabled={saving || saved}
  >
    {saving ? 'Saving…' : saved ? 'Saved!' : 'Save Paths'}
  </button>
</div>
```

```tsx
// Source: SettingsTab.tsx:221-247 [VERIFIED 2026-05-04]
async function handleSaveCLIPaths() {
  setSaving(true); setError(null)
  try {
    // ... do the work ...
    setSaved(true)
    setTimeout(() => setSaved(false), 1500)
  } catch (err) {
    setError(err instanceof Error ? err.message : String(err))
  } finally {
    setSaving(false)
  }
}
```

For Phase 92 PluginsSection: identical pattern, button label `Save Plugins`, calls `SetPluginSettings(state)` instead of looping `UpdateCLIPath`.

### Example 4: Existing `EventsEmit` — `session:exit`

```go
// Source: app.go:302-309 [VERIFIED 2026-05-04]
runtime.EventsEmit(a.ctx, "session:exit", map[string]any{
    "sessionId": id,
    "exitCode":  code,
    // ...
})
```

For Phase 92, in `(a *App) SetPluginSettings(s daemon.PluginSettings) error`:
```go
if err := a.client.SetPluginSettings(s); err != nil { return err }
runtime.EventsEmit(a.ctx, "settings:plugins", s)
return nil
```

### Example 5: Existing `EventsOn` subscription — `session:exit`

```ts
// Source: App.tsx:323-355 [VERIFIED 2026-05-04 — abbreviated]
const offExit = EventsOn(
  'session:exit',
  (data: { sessionId: string; exitCode: number; ... }) => {
    setSessionExits(prev => ({ ...prev, [data.sessionId]: { ... } }))
  }
)
// ... later: return () => { offExit(); ... }
```

For Phase 92, add (in the same useEffect):
```ts
const offPlugins = EventsOn('settings:plugins', (s: PluginSettings) => {
  setPluginConfig(s)
})
// cleanup: offPlugins()
```

### Example 6: Existing source-inspection test pattern

```ts
// Source: SettingsTab.persistence.test.tsx [VERIFIED 2026-05-04]
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'
import raw from '../../components/SettingsTab.tsx?raw'

describe('SET-03: Save confirmation feedback', () => {
    it('has saved state variable', () => {
        expect(raw).toContain('setSaved')
    })
    it('uses --saved CSS modifier', () => {
        expect(raw).toContain('settings-panel__btn--saved')
    })
})
```

For Phase 92: `frontend/src/components/__tests__/PluginsSection.test.tsx` follows this exact shape — no `render()`, no `userEvent`, just `?raw` import + string assertions. WebGL/Canvas-touching tests don't run cleanly in jsdom; source-inspection is the documented v3.1 precedent.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Per-setting Wails Get/Set pairs (one PATCH route per field) | Pattern still used for `startMinimized`/`autoCloseSession`; **but for grouped settings like `PluginSettings` use a single struct round-trip** | Phase 92 introduces this for the first time | Frontend Save fires once per click; all 8 toggles commit atomically |
| Direct `json.Unmarshal` into zero-value struct | **Defaults-merge** (pre-populate struct, then unmarshal on top) | Phase 92 (load-bearing for v3.1→v3.2 migration) | Returning users keep functional defaults; missing JSON keys do not zero new fields |
| No schema versioning | `SchemaVersion int` top-level field | Phase 92 introduces; v3.2 = 2 | Future v3.3+ migrations have a sentinel |

**Deprecated/outdated:**
- Naive `json.Unmarshal` of `daemonSettings` is deprecated in Phase 92 — replace with defaults-merge load. The PR that lands Phase 92's `engine.go` change must include a comment explaining this.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Go's `encoding/json.Unmarshal` does NOT clear struct fields before populating; missing JSON keys leave struct fields untouched. | Pattern 1 (defaults-merge) | If wrong, the entire defaults-merge pattern fails silently (would zero everything every load). Confidence: HIGH per Go stdlib documentation, but verify with a 5-line unit test in the plan before relying on it. |
| A2 | Wails v2's `runtime.EventsEmit(ctx, name, payload)` accepts arbitrary JSON-serializable Go structs as payload (not just primitives or maps). | Architecture diagram, Example 4 | If wrong, must change emit payload to a `map[string]any` like the existing `session:exit` pattern. Confidence: HIGH — `tailscale:install:done` (`[VERIFIED: app.go:867,871]`) emits a `map[string]interface{}` with mixed types; struct serialization should work identically. Verify in the plan with a tiny end-to-end test. |
| A3 | Wails v2's TS type generation in `frontend/src/wailsjs/go/main/App.d.ts` will produce a `PluginSettings` type once `(a *App) GetPluginSettings()` is exposed. | "Don't Hand-Roll" table — TS type | If wrong, hand-write the `PluginSettings` TS shape in `frontend/src/types/plugins.ts`. Cost: small; risk: zero. Confidence: HIGH — `[VERIFIED: SettingsTab.tsx:21]` already imports `DetectedCLI` from the generated bindings. |
| A4 | The `tests/fixtures/` directory does not yet exist. | Project Structure | If it does exist, just add `settings_v3.1.json` to it. Confidence: HIGH — `[VERIFIED: find` returned only `tests/build-script.test.sh` 2026-05-04`]`. |
| A5 | `frontend/src/types/` already exists. | Project Structure | If not, create it. Confidence: HIGH — `[VERIFIED: ls /Users/ken/dev/agenthub/frontend/src/types/]` returned no error. |

## Open Questions

1. **Should the engine's `pluginSettings` field be `*PluginSettings` or `PluginSettings`?**
   - What we know: existing precedent is mixed — `startMinimized` is `bool` (zero-value-meaningful), `autoCloseSession` is `*bool` (tri-state nil/true/false to distinguish "user explicitly disabled" from "field missing").
   - What's unclear: do we need the tri-state for plugins? The defaults-merge pattern means we always populate non-nil at load time, so the field could be `PluginSettings` (value type). But pointer makes "first-run, no settings.json yet" detection cleaner.
   - Recommendation: use **value type** `PluginSettings` after load. Defaults-merge always produces a populated struct. The "first-run" detection happens at file-read time (file missing) rather than struct-shape time. Simpler; matches `cliPaths` (a populated map, not a pointer-map).

2. **Should `SetPluginSettings` API route be PATCH (partial) or PUT (full replace)?**
   - What we know: existing `/settings/start-minimized` and `/settings/auto-close-session` use PATCH; both have wrapper-map bodies (`{"startMinimized": true}`).
   - What's unclear: PATCH typically implies partial-update semantics; we're sending the full PluginSettings every time.
   - Recommendation: **PATCH**, with the full struct as the body. Consistency with the surrounding routes outweighs HTTP-semantic purity for an internal-only daemon API. Document the full-replace semantic in a code comment.

3. **What payload shape does `settings:plugins` carry?**
   - Options: (a) the full `PluginSettings` struct serialized as JSON; (b) `null` (sentinel — App refetches via `GetPluginSettings`).
   - Recommendation: **(a)** — matches `session:exit` and `tailscale:health` precedent; one fewer round-trip; the user just saved this exact value, so the round-trip would re-fetch the same thing.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Daemon build | ✓ | (whatever `go.mod` pins; not relevant for Phase 92's logic) | — |
| Wails v2 CLI | Frontend rebuild after `app.go` changes | ✓ | (per `build.sh`) | — |
| pnpm | Frontend test+build | ✓ | — | — |
| vitest | Frontend tests | ✓ | 4.1.0 (`[VERIFIED: frontend/package.json]`) | — |

**No external services required** for Phase 92 (no DB, no network, no Docker). All work is in-process Go + React.

## Validation Architecture

> nyquist_validation status in `.planning/config.json`: not set — treat as enabled. Section included.

### Test Framework

| Property | Value |
|----------|-------|
| Frontend framework | vitest 4.1.0 + jsdom 29.0.0 (`[VERIFIED: frontend/package.json]`) |
| Frontend config file | `frontend/vite.config.ts` (`[VERIFIED 2026-05-04]`) — defines `test.environment: 'jsdom', globals: true` plus alias stubs for `wailsjs/runtime/runtime` |
| Backend framework | Go stdlib `testing` (`[VERIFIED: internal/daemon/engine_settings_test.go]`) |
| Backend config file | none — standard `go test ./...` |
| Quick run command (frontend) | `cd frontend && pnpm test` |
| Quick run command (backend) | `go test ./internal/daemon/... -run "TestSettings\|TestPlugin\|TestMigration"` |
| Full suite command | `cd frontend && pnpm test && cd .. && go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PLUG-01 | Plugin choices persist via settings.json across daemon restart | Go integration | `go test ./internal/daemon -run TestPluginSettingsRoundTrip -v` | ❌ Wave 0 (`internal/daemon/engine_plugins_test.go`) |
| PLUG-01 | Wails RPC `SetPluginSettings` writes to disk | Go unit | `go test ./internal/daemon -run TestSetPluginSettings -v` | ❌ Wave 0 (same file) |
| PLUG-02 | v3.1 fixture upgrades to v3.2 with defaults populated + schemaVersion: 2 | Go fixture | `go test ./internal/daemon -run TestSettingsMigrationV3_1ToV3_2 -v` | ❌ Wave 0 (`internal/daemon/engine_migration_test.go` + `tests/fixtures/settings_v3.1.json`) |
| PLUG-02 | Migration is idempotent on second load | Go fixture | `go test ./internal/daemon -run TestSettingsMigrationIdempotent -v` | ❌ Wave 0 (same) |
| PLUG-02 | `defaultPluginSettings()` matches UI-SPEC default ON/OFF table | Go unit | `go test ./internal/daemon -run TestDefaultPluginSettings -v` | ❌ Wave 0 (`internal/daemon/plugin_settings_test.go`) |
| PLUG-03 | `app.go:SetPluginSettings` emits `settings:plugins` runtime event | Source-inspection (jsdom can't observe Wails events without mock) | `pnpm test -- App.plugin-event` | ❌ Wave 0 — likely a vitest source-inspection on `app.go`-grep test OR a vitest test asserting `App.tsx` registers `EventsOn('settings:plugins', ...)` |
| PLUG-03 | `App.tsx` subscribes to `settings:plugins` and threads `pluginConfig` prop | vitest source-inspection | `pnpm test -- App.plugin-event` | ❌ Wave 0 (`frontend/src/__tests__/App.plugin-event.test.tsx`) |
| PLUG-03 | `TerminalPanel` props interface accepts `pluginConfig?: PluginSettings` | vitest source-inspection | `pnpm test -- TerminalPanel.plugin-prop` | ❌ Wave 0 (`frontend/src/components/__tests__/TerminalPanel.test.tsx` extension OR new file) |
| PUI-01 | PluginsSection renders 8 toggle rows in UI-SPEC order with correct labels | vitest source-inspection | `pnpm test -- PluginsSection` | ❌ Wave 0 (`frontend/src/components/__tests__/PluginsSection.test.tsx`) |
| PUI-01 | PluginsSection exposes Save button with three-state semantics | vitest source-inspection | `pnpm test -- PluginsSection` | ❌ Wave 0 (same file) |
| PUI-01 | SettingsTab.tsx imports and renders `<PluginsSection />` after Paths section | vitest source-inspection | `pnpm test -- SettingsTab` | extends `SettingsTab.test.tsx` |

### Sampling Rate

- **Per task commit:** `pnpm test -- <relevant-pattern>` and/or `go test ./internal/daemon -run <Test>` — 5-15s budget per test
- **Per wave merge:** full frontend `pnpm test` + full Go `go test ./...` — under 60s combined
- **Phase gate:** all of the above + the manual smoke from UI-SPEC §"Acceptance Snapshot" (boot v3.2 dev build with a v3.1 fixture; observe Plugins section renders 8 rows correctly + Save flow + persistence after restart)

### Wave 0 Gaps

- [ ] `tests/fixtures/settings_v3.1.json` — golden v3.1-shape fixture, no `plugins` key, no `schemaVersion` key, must include realistic `cliPaths` + `startMinimized: false` + `autoCloseSession: true` so the test exercises the merge across all existing keys (covers REQ-PLUG-02)
- [ ] `internal/daemon/plugin_settings.go` — `PluginSettings` struct + `defaultPluginSettings()` + `CurrentSchemaVersion` constant (covers REQ-PLUG-01, PLUG-02)
- [ ] `internal/daemon/plugin_settings_test.go` — `TestDefaultPluginSettings` asserts the 8 defaults match UI-SPEC table (WebGL=true, Unicode11=true, Search=true, WebLinks=true, Image=true, Serialize=true, Clipboard=true, Progress=false) (covers PLUG-02)
- [ ] `internal/daemon/engine_plugins_test.go` — `TestPluginSettingsRoundTrip`, `TestSetPluginSettingsPersistsToDisk` (covers PLUG-01)
- [ ] `internal/daemon/engine_migration_test.go` — `TestSettingsMigrationV3_1ToV3_2`, `TestSettingsMigrationIdempotent` (covers PLUG-02 — load-bearing CI gate)
- [ ] `frontend/src/types/plugins.ts` — TS mirror of `PluginSettings` (or rely on Wails-generated types; see Open Question 1 + Don't Hand-Roll)
- [ ] `frontend/src/components/PluginsSection.tsx` — the UI shell (covers PUI-01)
- [ ] `frontend/src/components/__tests__/PluginsSection.test.tsx` — source-inspection visual contract (covers PUI-01)
- [ ] `frontend/src/__tests__/App.plugin-event.test.tsx` (or extend an existing `App.*.test.tsx`) — source-inspection that App.tsx registers `EventsOn('settings:plugins', ...)` and threads `pluginConfig` (covers PLUG-03)
- [ ] No framework install needed — vitest + Go test both already in place

## Security Domain

> `security_enforcement` config: not set explicitly — treat as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Phase 92 introduces no new auth surface; settings.json is local-only, daemon socket is loopback |
| V3 Session Management | no | No web-served plugin config in this phase (PLUG-04 is Phase 93) |
| V4 Access Control | partially | The new `PATCH /settings/plugins` and `GET /settings/plugins` API routes inherit the existing daemon-socket access control (Unix socket `0600` perms / Windows named pipe ACL) — same posture as `/settings/start-minimized`. **No new control needed**, but plan must verify the routes register on the same `a.mux` and inherit the same middleware (or lack thereof — daemon socket is presumed loopback-trusted). |
| V5 Input Validation | yes | `handleSetPluginSettings` MUST `json.Decode` into a typed struct (not `interface{}`); reject body > 8KB; reject unknown fields (`json.Decoder.DisallowUnknownFields()`) |
| V6 Cryptography | no | No crypto in Phase 92 — booleans only |

### Known Threat Patterns for Phase 92

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed JSON in PATCH body crashes daemon | DoS | Use `json.NewDecoder(r.Body).Decode(&req)` + return 400 on err — same as `handleSetStartMinimized` `[VERIFIED: api.go:478-487]` |
| Body-size DoS (multi-MB JSON) | DoS | `http.MaxBytesReader(w, r.Body, 8192)` before Decode (existing routes don't enforce this; recommend Phase 92 adds it as a defense-in-depth) |
| Schema version downgrade attack (write-back lower number) | Tampering | `SetPluginSettings` always writes `CurrentSchemaVersion` regardless of input — input struct does not contain a SchemaVersion field (it's daemonSettings-level, not PluginSettings-level) |
| Settings file outside config dir (path traversal via configDir poisoning) | Tampering | `configDir` is owned by the daemon (`daemonConfigDir()` in `engine.go`); not user-influenced. Inherits v3.1 trust model. No new mitigation needed. |

**Insight:** Phase 92 is **low-risk security-wise**. The settings.json file already exists, the daemon API already exposes it, and the new fields are booleans + integers (in Phase 99) — no new attack surface. The one defense-in-depth recommendation: `MaxBytesReader` on the PATCH route. Not strictly required if existing routes don't have it; flagged for the planner.

## Sources

### Primary (HIGH confidence)

- `[VERIFIED 2026-05-04]` `internal/daemon/engine.go:60-450` — `daemonSettings` struct, load/save functions, existing setting Get/Set methods
- `[VERIFIED 2026-05-04]` `internal/daemon/api.go:68-73, 474-502` — HTTP route registration + handler shape
- `[VERIFIED 2026-05-04]` `internal/daemon/client.go:111-139` — RPC client method shape
- `[VERIFIED 2026-05-04]` `app.go:415-432, 102-901` — Wails binding patterns + 8 existing `runtime.EventsEmit` sites
- `[VERIFIED 2026-05-04]` `frontend/src/components/SettingsTab.tsx:88-708` — toggleLoaded pattern, three-state Save button, existing toggle markup, section structure
- `[VERIFIED 2026-05-04]` `frontend/src/components/TerminalPanel.tsx:1-210` — current addon loading (FitAddon, Unicode11, WebGL), props interface, useEffect structure
- `[VERIFIED 2026-05-04]` `frontend/src/App.tsx:6, 30, 250-865` — TerminalPanel import + EventsOn subscriptions + TerminalPanel prop usage
- `[VERIFIED 2026-05-04]` `frontend/src/components/__tests__/SettingsTab.persistence.test.tsx` — source-inspection vitest pattern
- `[VERIFIED 2026-05-04]` `frontend/package.json` — vitest 4.1.0, jsdom 29, no new deps needed
- `[VERIFIED 2026-05-04]` `frontend/vite.config.ts` — vitest config + Wails runtime stub aliases
- `[VERIFIED 2026-05-04]` `internal/daemon/engine_settings_test.go` — Go test pattern: `t.TempDir()` + temp configDir + reload-engine round-trip
- `[VERIFIED 2026-05-04]` `frontend/src/wailsjs/runtime/runtime.js` — confirms `EventsOn`/`EventsEmit` runtime stubs
- `[CITED]` `.planning/milestones/v3.2-phases/92-plugin-settings-foundation/92-UI-SPEC.md` — full UI design contract (decisions on labels, copy, ordering, default ON/OFF, visual specs, accessibility)
- `[CITED]` `.planning/REQUIREMENTS.md:13-19, 89-96` — PLUG-01..03, PUI-01 verbatim
- `[CITED]` `.planning/ROADMAP.md:321-330` — Phase 92 goal + 4 success criteria
- `[CITED]` `.planning/STATE.md:43-58, 80-86` — locked decisions + blocker on settings migration zero-defaults
- `[CITED]` `.planning/research/STACK.md:5-14, 12-15` — base versions confirmed in v3.1 lock
- `[CITED]` `.planning/research/ARCHITECTURE.md:27-138` — existing architecture snapshot, settings struct shape, lifecycle matrix, decision rationale for daemon-as-source-of-truth + emit-after-save
- `[CITED]` `.planning/research/PITFALLS.md:362-383` — Pitfall #14 (settings.json migration zero-defaults) — defaults-merge pattern, schema version field, fixture test mandate
- `[CITED]` `.planning/research/SUMMARY.md:11-17` — phase split rationale (foundation first)

### Secondary (MEDIUM confidence)

- `[ASSUMED]` Go's `encoding/json.Unmarshal` semantics for missing keys (preserves existing field values) — see Assumption A1. This is documented Go stdlib behavior; verify with a 5-line plan-level test before relying on it.
- `[ASSUMED]` Wails v2's `runtime.EventsEmit` accepts arbitrary serializable Go structs as payload — see Assumption A2. Existing `app.go:867,871` emits `map[string]interface{}` which is a stronger guarantee; struct-payload should follow.
- `[ASSUMED]` Wails v2's TS type generation includes types from new `(*App)` method signatures — see Assumption A3. Existing pattern `[VERIFIED: SettingsTab.tsx:21]` confirms generation works for `DetectedCLI`; should extend identically.

### Tertiary (LOW confidence)

None for Phase 92. Every claim that drives an implementation decision is either VERIFIED in the repo or CITED from a v3.2 research artifact. The three ASSUMED claims are stdlib/framework semantics that any plan task can verify in <5 minutes.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; all reuse-existing
- Architecture: HIGH — every pattern has a working v3.1 example in the repo
- Pitfalls: HIGH — Pitfall #14 (settings migration) is the only critical one and it's already captured by ROADMAP SC-1 + a dedicated test
- Assumptions: 3 ASSUMED items, all stdlib/framework semantics, all easily verifiable in a single plan task

**Research date:** 2026-05-04
**Valid until:** 2026-06-04 (30 days — stable foundation work, low ecosystem-churn surface)
