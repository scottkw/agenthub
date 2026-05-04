# Phase 92: Plugin Settings Foundation — Pattern Map

**Mapped:** 2026-05-04
**Files analyzed:** 13 (10 Go + frontend create/modify, 3 test/fixture)
**Analogs found:** 13 / 13 — every file has at least one strong existing analog

> Every Phase 92 file is a **mechanical extension of an existing v3.1 pattern**.
> No new architectural primitives. The single load-bearing novelty is the
> defaults-merge load order in `loadSettingsFromDisk` — see Shared Pattern A.

---

## File Classification

| New / Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/daemon/plugin_settings.go` (NEW) | model | CRUD (struct + defaults constructor + schema-version constant) | `internal/daemon/engine.go:66-71` (`daemonSettings` struct) | role-match |
| `internal/daemon/engine.go` (MODIFIED) | service | CRUD + persistence | self (extend in place) | exact |
| `internal/daemon/api.go` (MODIFIED) | controller (HTTP handler) | request-response | `api.go:474-487` (`handleGetStartMinimized` / `handleSetStartMinimized`) | exact |
| `internal/daemon/client.go` (MODIFIED) | service (RPC client) | request-response | `client.go:111-124` (`GetStartMinimized` / `SetStartMinimized`) | exact |
| `app.go` (MODIFIED) | controller (Wails binding) | request-response + event-emit | `app.go:415-432` + `app.go:302` (`session:exit` emit) | exact |
| `internal/daemon/plugin_settings_test.go` (NEW) | test | unit | `engine_settings_test.go:101-160` | role-match |
| `internal/daemon/engine_plugins_test.go` (NEW) | test | unit (round-trip) | `engine_settings_test.go:11-56` | exact |
| `internal/daemon/engine_migration_test.go` (NEW) | test | fixture (file-based) | `engine_settings_test.go:88-99` (load-missing-file) | role-match |
| `tests/fixtures/settings_v3.1.json` (NEW) | config (test fixture) | file-I/O | none — first fixture in repo | no-analog |
| `frontend/src/types/plugins.ts` (NEW) | model (TS type) | type-only | `frontend/src/wailsjs/go/main/App.d.ts:DetectedCLI` (auto-generated) | role-match |
| `frontend/src/components/PluginsSection.tsx` (NEW) | component | event-driven (toggle + save) | `SettingsTab.tsx:336-361` (Behavior section) + `:695-703` (Save Paths button) | exact |
| `frontend/src/components/SettingsTab.tsx` (MODIFIED) | component | event-driven | self (insert after Paths section) | exact |
| `frontend/src/App.tsx` (MODIFIED) | component (root) | event-driven (`EventsOn` + prop drill) | `App.tsx:250-307, 401-415, 865-872` | exact |
| `frontend/src/components/TerminalPanel.tsx` (MODIFIED) | component | type-only prop addition | `TerminalPanel.tsx:37-44` (existing props interface) | exact |
| `frontend/src/components/__tests__/PluginsSection.test.tsx` (NEW) | test | source-inspection | `__tests__/SettingsTab.persistence.test.tsx:1-38` | exact |

---

## Pattern Assignments

### `internal/daemon/plugin_settings.go` (NEW — model)

**Analog:** `internal/daemon/engine.go:66-71` (the existing `daemonSettings` struct)

**Existing struct shape to mirror** (`engine.go:66-71`):
```go
// daemonSettings is the persisted settings structure.
type daemonSettings struct {
    CLIPaths         map[string]string `json:"cliPaths,omitempty"`
    StartMinimized   bool              `json:"startMinimized,omitempty"`
    AutoCloseSession *bool             `json:"autoCloseSession,omitempty"`
}
```

**For Phase 92, add a new file `plugin_settings.go`** containing:
- `PluginSettings` struct (8 boolean fields — `WebGL`, `Unicode11`, `Search`, `WebLinks`, `Image`, `Serialize`, `Clipboard`, `Progress`)
- `defaultPluginSettings()` constructor that returns a `PluginSettings` value with 7 ON, Progress OFF (per UI-SPEC §"Default ON / OFF on first launch", lines 282-294)
- `CurrentSchemaVersion` constant = `2`
- Top-level convention: `PluginSettings` field uses `json:"plugins"` (NO `omitempty` per Pitfall #14); `SchemaVersion` uses `json:"schemaVersion"` (NO `omitempty`); sub-fields MAY use `omitempty` but Phase 92 keeps it simple — none do.

**JSON tag pattern** (mirror `engine.go:66-71` style — lowercase camelCase keys):
```go
type PluginSettings struct {
    WebGL     bool `json:"webgl"`
    Unicode11 bool `json:"unicode11"`
    Search    bool `json:"search"`
    WebLinks  bool `json:"webLinks"`
    Image     bool `json:"image"`
    Serialize bool `json:"serialize"`
    Clipboard bool `json:"clipboard"`
    Progress  bool `json:"progress"`
}
```

**Defaults** (per UI-SPEC table, lines 285-294): all `true` except `Progress: false`.

---

### `internal/daemon/engine.go` (MODIFIED — service)

**Analog:** itself. Extend in place at three known coordinates.

**Coordinate 1 — extend `daemonSettings` struct** (`engine.go:66-71`):
```go
// CURRENT (engine.go:66-71)
type daemonSettings struct {
    CLIPaths         map[string]string `json:"cliPaths,omitempty"`
    StartMinimized   bool              `json:"startMinimized,omitempty"`
    AutoCloseSession *bool             `json:"autoCloseSession,omitempty"`
}

// MODIFIED — add Plugins + SchemaVersion (no omitempty on these two)
type daemonSettings struct {
    CLIPaths         map[string]string `json:"cliPaths,omitempty"`
    StartMinimized   bool              `json:"startMinimized,omitempty"`
    AutoCloseSession *bool             `json:"autoCloseSession,omitempty"`
    Plugins          PluginSettings    `json:"plugins"`
    SchemaVersion    int               `json:"schemaVersion"`
}
```

**Coordinate 2 — extend `SessionEngine` struct** (`engine.go:22-41`):
```go
// MODIFIED — add pluginSettings field next to startMinimized/autoCloseSession (line 36-37)
startMinimized   bool
autoCloseSession *bool
pluginSettings   PluginSettings  // NEW — value type, populated by load-merge
```

**Coordinate 3 — REPLACE `loadSettingsFromDisk`** (`engine.go:86-118`) with the **defaults-merge** pattern. Current implementation (load-bearing change):
```go
// CURRENT (engine.go:86-118) — naive zero-value Unmarshal
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
    data, err := os.ReadFile(settingsPath(dir))
    if err != nil {
        return // file not found — first run
    }
    var s daemonSettings        // <-- ZERO VALUE, this is Pitfall #14
    if json.Unmarshal(data, &s) != nil {
        return
    }
    // ... copy into engine fields ...
}
```

**MODIFIED — defaults-merge** (per Pattern 1 in RESEARCH.md, mandatory for PLUG-02):
```go
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
    data, err := os.ReadFile(settingsPath(dir))

    // Pre-populate defaults BEFORE Unmarshal — load-bearing for v3.1 migration.
    s := daemonSettings{
        SchemaVersion: CurrentSchemaVersion,
        Plugins:       defaultPluginSettings(),
    }

    if err == nil {
        // Fields present in JSON overwrite defaults; missing fields keep defaults.
        if err := json.Unmarshal(data, &s); err != nil {
            return // corrupt file — keep defaults, don't crash
        }
    }
    // ... existing CLIPaths shell-mismatch cleanup logic ...

    e.mu.Lock()
    // ... existing assignments ...
    e.pluginSettings = s.Plugins

    // Detect upgrade-path: re-save when SchemaVersion < current so future loads
    // observe the new fields. Idempotent on second start.
    if s.SchemaVersion < CurrentSchemaVersion {
        s.SchemaVersion = CurrentSchemaVersion
        e.saveSettingsToDisk()
    }
    e.mu.Unlock()
}
```

**Coordinate 4 — extend `saveSettingsToDisk`** (`engine.go:120-133`):
```go
// CURRENT (engine.go:122-127)
s := daemonSettings{
    CLIPaths:         e.cliPaths,
    StartMinimized:   e.startMinimized,
    AutoCloseSession: e.autoCloseSession,
}

// MODIFIED — add the two new fields
s := daemonSettings{
    CLIPaths:         e.cliPaths,
    StartMinimized:   e.startMinimized,
    AutoCloseSession: e.autoCloseSession,
    Plugins:          e.pluginSettings,
    SchemaVersion:    CurrentSchemaVersion,
}
```

**Coordinate 5 — add Get/Set engine methods** (mirror `engine.go:377-390` exactly):
```go
// PATTERN SOURCE — engine.go:377-390 (GetStartMinimized / SetStartMinimized)
func (e *SessionEngine) GetStartMinimized() bool {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.startMinimized
}

func (e *SessionEngine) SetStartMinimized(val bool) {
    e.mu.Lock()
    e.startMinimized = val
    e.saveSettingsToDisk()
    e.mu.Unlock()
}

// FOR PHASE 92 — add immediately below SetAutoCloseSession (~engine.go:409)
func (e *SessionEngine) GetPluginSettings() PluginSettings {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.pluginSettings
}

func (e *SessionEngine) SetPluginSettings(s PluginSettings) {
    e.mu.Lock()
    e.pluginSettings = s
    e.saveSettingsToDisk()
    e.mu.Unlock()
}
```

**Coordinate 6 — initialize in `NewSessionEngine`** (`engine.go:140-153`): no change required if `pluginSettings` is a value type — `loadSettingsFromDisk` populates it via the defaults-merge.

---

### `internal/daemon/api.go` (MODIFIED — controller)

**Analog:** `api.go:474-487` (`handleGetStartMinimized` / `handleSetStartMinimized`)

**Route registration pattern** (`api.go:70-73`):
```go
// CURRENT (api.go:70-73)
a.mux.HandleFunc("GET /settings/start-minimized", a.handleGetStartMinimized)
a.mux.HandleFunc("PATCH /settings/start-minimized", a.handleSetStartMinimized)
a.mux.HandleFunc("GET /settings/auto-close-session", a.handleGetAutoCloseSession)
a.mux.HandleFunc("PATCH /settings/auto-close-session", a.handleSetAutoCloseSession)

// FOR PHASE 92 — add immediately below (line ~74)
a.mux.HandleFunc("GET /settings/plugins", a.handleGetPluginSettings)
a.mux.HandleFunc("PATCH /settings/plugins", a.handleSetPluginSettings)
```

**Handler pattern** (`api.go:474-487` — copy verbatim shape, swap struct):
```go
// PATTERN SOURCE — api.go:474-487
func (a *API) handleGetStartMinimized(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]bool{"startMinimized": a.engine.GetStartMinimized()})
}

func (a *API) handleSetStartMinimized(w http.ResponseWriter, r *http.Request) {
    var req struct {
        StartMinimized bool `json:"startMinimized"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    a.engine.SetStartMinimized(req.StartMinimized)
    w.WriteHeader(http.StatusNoContent)
}

// FOR PHASE 92 — note SetPluginSettings sends the FULL PluginSettings struct,
// NOT a wrapper map (pointed out in RESEARCH.md Open Question 2).
func (a *API) handleGetPluginSettings(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, a.engine.GetPluginSettings())
}

func (a *API) handleSetPluginSettings(w http.ResponseWriter, r *http.Request) {
    // Defense-in-depth body cap (RESEARCH.md Security §)
    r.Body = http.MaxBytesReader(w, r.Body, 8192)

    var req PluginSettings
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()  // V5 Input Validation — reject unknown plugin keys
    if err := dec.Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    a.engine.SetPluginSettings(req)
    w.WriteHeader(http.StatusNoContent)
}
```

---

### `internal/daemon/client.go` (MODIFIED — service)

**Analog:** `client.go:111-124` (`GetStartMinimized` / `SetStartMinimized`)

```go
// PATTERN SOURCE — client.go:111-124
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

// FOR PHASE 92 — note GET unmarshals INTO PluginSettings directly (no wrapper map);
// PATCH sends the struct directly (no wrapper map).
func (c *DaemonClient) GetPluginSettings() (PluginSettings, error) {
    var resp PluginSettings
    if err := c.doJSON(http.MethodGet, "/settings/plugins", nil, &resp); err != nil {
        return PluginSettings{}, err
    }
    return resp, nil
}

func (c *DaemonClient) SetPluginSettings(s PluginSettings) error {
    return c.doJSON(http.MethodPatch, "/settings/plugins", s, nil)
}
```

---

### `app.go` (MODIFIED — Wails binding)

**Analog:** `app.go:415-432` (`GetStartMinimized` / `SetStartMinimized`) + `app.go:302-309` (`runtime.EventsEmit("session:exit", ...)`)

**Wails binding pattern** (`app.go:413-432`):
```go
// PATTERN SOURCE — app.go:413-432
// GetStartMinimized returns the persisted start-minimized preference.
// Returns false (show window) when daemon is not connected.
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

// SetStartMinimized persists the start-minimized preference.
func (a *App) SetStartMinimized(val bool) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.SetStartMinimized(val)
}
```

**EventsEmit pattern** (`app.go:302-309`):
```go
// PATTERN SOURCE — app.go:302-309 (existing session:exit emit)
runtime.EventsEmit(a.ctx, "session:exit", map[string]any{
    "sessionId":   sessionID,
    "exitCode":    exitCode,
    // ...
})
```

**FOR PHASE 92** — combine both patterns. Add immediately below `SetAutoCloseSession` (~`app.go:455`):
```go
// GetPluginSettings returns the persisted plugin enable/disable preferences.
// Returns zero-value (all OFF) when daemon is not connected — caller MUST
// also handle the loading state via toggleLoaded-style guard in PluginsSection.
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

// SetPluginSettings persists plugin preferences AND broadcasts the change to all
// open desktop terminals via the settings:plugins runtime event (PLUG-03).
// EventsEmit lives in app.go ONLY — internal/daemon has no Wails runtime context.
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

**Critical:** the EventsEmit fires AFTER `client.SetPluginSettings` succeeds, NOT inside the daemon. See Shared Pattern C.

---

### `internal/daemon/plugin_settings_test.go` (NEW — test)

**Analog:** `engine_settings_test.go:101-160` (the `TestStartMinimizedPersistence` shape — checking on-disk JSON contents).

**Test pattern** (`engine_settings_test.go:11-56` is the canonical "engine round-trip with t.TempDir" pattern):
```go
// PATTERN SOURCE — engine_settings_test.go:11-56
func TestSettingsPersistence(t *testing.T) {
    dir := t.TempDir()
    // ... setup ...
    e := &SessionEngine{
        configDir: dir,
        cliPaths:  make(map[string]string),
    }
    // ... action ...
    // ... read settings.json back ...
    var s daemonSettings
    if err := json.Unmarshal(data, &s); err != nil { /* ... */ }
    // ... assert ...
    // Create a new engine from same dir — verify it loads the saved paths
    e2 := &SessionEngine{
        configDir: dir,
        cliPaths:  make(map[string]string),
    }
    e2.loadSettingsFromDisk(dir)
    got := e2.GetCLIPaths()
}
```

**FOR PHASE 92, `plugin_settings_test.go` adds:**

1. `TestDefaultPluginSettings` — pure unit test, asserts the 8 defaults match UI-SPEC table:
```go
func TestDefaultPluginSettings(t *testing.T) {
    s := defaultPluginSettings()
    if !s.WebGL || !s.Unicode11 || !s.Search || !s.WebLinks ||
       !s.Image || !s.Serialize || !s.Clipboard {
        t.Errorf("expected 7 ON defaults, got %+v", s)
    }
    if s.Progress {
        t.Error("expected Progress=false default (UI-SPEC line 294)")
    }
}
```

---

### `internal/daemon/engine_plugins_test.go` (NEW — test)

**Analog:** `engine_settings_test.go:101-160` exactly (same shape, different field).

```go
// PATTERN SOURCE — engine_settings_test.go:101-130 (TestStartMinimizedPersistence)
func TestStartMinimizedPersistence(t *testing.T) {
    dir := t.TempDir()
    e := &SessionEngine{
        configDir: dir,
        cliPaths:  make(map[string]string),
    }

    if e.GetStartMinimized() { t.Error("expected default=false") }

    e.SetStartMinimized(true)

    data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
    if err != nil { t.Fatalf("settings.json not found: %v", err) }
    var s daemonSettings
    // ... assert s.StartMinimized ...

    // New engine, same dir — verify load
    e2 := &SessionEngine{configDir: dir, cliPaths: make(map[string]string)}
    e2.loadSettingsFromDisk(dir)
    if !e2.GetStartMinimized() { t.Error("expected loaded=true") }
}
```

**FOR PHASE 92** — `TestSetPluginSettingsRoundTrip` mirrors this exactly. **Caveat:** when constructing `SessionEngine` directly (not via `NewSessionEngine`), seed `pluginSettings = defaultPluginSettings()` to mimic the load-merge that `NewSessionEngine` provides via `loadSettingsFromDisk`.

---

### `internal/daemon/engine_migration_test.go` (NEW — fixture-based test)

**Analog:** `engine_settings_test.go:88-99` (the `TestSettingsLoadMissingFile` shape demonstrates the load-from-disk testing pattern, but Phase 92 adds the **first** repo test that uses a checked-in fixture file).

**Pattern** — write the fixture into `t.TempDir()` at test start, then call `loadSettingsFromDisk`:
```go
func TestSettingsMigrationV3_1ToV3_2(t *testing.T) {
    dir := t.TempDir()

    // Read the v3.1 fixture from tests/fixtures/ and copy into temp dir as settings.json
    fixture, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "settings_v3.1.json"))
    if err != nil { t.Fatalf("read fixture: %v", err) }
    if err := os.WriteFile(filepath.Join(dir, "settings.json"), fixture, 0600); err != nil {
        t.Fatal(err)
    }

    e := &SessionEngine{
        configDir: dir,
        cliPaths:  make(map[string]string),
    }
    e.loadSettingsFromDisk(dir)

    // PLUG-02 assertions: 7 ON, Progress OFF, schemaVersion: 2 written back
    s := e.GetPluginSettings()
    if !s.WebGL { t.Error("WebGL=false after migration; expected true (Pitfall #14)") }
    if s.Progress { t.Error("Progress=true after migration; expected false") }

    // Re-read settings.json to verify schemaVersion was written
    data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
    var raw daemonSettings
    json.Unmarshal(data, &raw)
    if raw.SchemaVersion != 2 {
        t.Errorf("schemaVersion=%d, want 2", raw.SchemaVersion)
    }
}

// Idempotency: second load does not re-trigger the upgrade write.
// (Easiest assertion: file mtime stable across two loads.)
```

---

### `tests/fixtures/settings_v3.1.json` (NEW — fixture, NO ANALOG)

No existing fixture in the repo. The directory `tests/` exists with one shell test (`build-script.test.sh`); the `fixtures/` subdirectory does NOT exist and must be created.

**Fixture shape** (must reflect a realistic v3.1 `settings.json` — no `plugins` key, no `schemaVersion` key):
```json
{
    "cliPaths": {
        "claude": "/usr/local/bin/claude",
        "tailscale": "/Applications/Tailscale.app/Contents/MacOS/Tailscale"
    },
    "startMinimized": false,
    "autoCloseSession": true
}
```

The presence of all three v3.1 keys is intentional — it exercises the merge across all existing fields, not just the new plugin block (per RESEARCH.md Wave 0 Gap §720).

---

### `frontend/src/types/plugins.ts` (NEW — TS type)

**Analog:** `frontend/src/wailsjs/go/main/App.d.ts` (auto-generated; verified to contain `DetectedCLI` per `SettingsTab.tsx:21`).

**Existing TS-from-Go binding pattern** (`SettingsTab.tsx:21`):
```typescript
import type { DetectedCLI } from '../wailsjs/go/main/App'
```

**Phase 92 strategy** (per RESEARCH.md "Don't Hand-Roll" table line 348 + Open Question 1):
- After `wails generate` runs, `App.d.ts` will expose `GetPluginSettings(): Promise<daemon.PluginSettings>` and the type will be reachable as a Wails-generated type. **Confirm before hand-writing.**
- If for any reason the auto-generation does not surface `PluginSettings`, hand-write the file to mirror the Go struct exactly:

```typescript
// frontend/src/types/plugins.ts (hand-written fallback only)
export interface PluginSettings {
  webgl: boolean
  unicode11: boolean
  search: boolean
  webLinks: boolean
  image: boolean
  serialize: boolean
  clipboard: boolean
  progress: boolean
}
```

**Field naming** must match the Go `json:"..."` tags exactly (lowercase `webgl`, `webLinks` camelCase per Go tag).

---

### `frontend/src/components/PluginsSection.tsx` (NEW — component)

**Analogs:**
- Imports / state hooks shape: `SettingsTab.tsx:1-22, 65-67, 87-91`
- Toggle markup: `SettingsTab.tsx:336-361` (Behavior section, copied verbatim per UI-SPEC §"Reused class")
- Three-state Save button: `SettingsTab.tsx:695-703` + handler `:221-247`
- `toggleLoaded` guard: `SettingsTab.tsx:88-89, 154-159, 338`

**Imports pattern** (`SettingsTab.tsx:1-22`):
```typescript
import React, { useState, useEffect } from 'react'
import {
  // ... Wails-generated bindings ...
  GetPluginSettings,
  SetPluginSettings,
} from '../wailsjs/go/main/App'
import type { PluginSettings } from '../types/plugins'  // OR auto-generated
```

**State hooks pattern** (mirror `SettingsTab.tsx:65-67, 87-91`):
```typescript
// Save state (three-state)
const [saving, setSaving] = useState(false)
const [saved, setSaved] = useState(false)
const [error, setError] = useState<string | null>(null)

// Plugin config state + load guard
const [pluginConfig, setPluginConfig] = useState<PluginSettings | null>(null)
const [pluginsLoaded, setPluginsLoaded] = useState(false)
```

**Load on mount** (mirror `SettingsTab.tsx:154-159`):
```typescript
useEffect(() => {
  GetPluginSettings().then(s => {
    setPluginConfig(s)
    setPluginsLoaded(true)
  }).catch(() => setPluginsLoaded(true))
}, [])
```

**Toggle row markup** — repeat 8 times with the per-plugin labels and descriptions from UI-SPEC §"Per-plugin one-sentence descriptions" (lines 158-166). Source pattern (`SettingsTab.tsx:336-361`):
```tsx
<h3>Plugins</h3>
<div className="settings-panel__field-group">
  {pluginsLoaded && pluginConfig && (
    <label
      className={`settings-panel__toggle-row${pluginConfig.webgl ? ' settings-panel__toggle-row--checked' : ''}`}
      htmlFor="plugin-webgl"
      style={saving ? { pointerEvents: 'none', opacity: 0.6 } : undefined}
    >
      <span className="settings-panel__toggle-track">
        <span className="settings-panel__toggle-thumb" />
      </span>
      <span className="settings-panel__toggle-label">WebGL renderer</span>
    </label>
  )}
  <input
    type="checkbox"
    id="plugin-webgl"
    className="settings-panel__toggle-input"
    checked={pluginConfig?.webgl ?? false}
    onChange={() => setPluginConfig(prev => prev ? { ...prev, webgl: !prev.webgl } : prev)}
  />
  <p className="settings-panel__description">
    GPU-accelerated terminal rendering with automatic DOM fallback if the GPU context is lost.
  </p>
</div>
{/* ... 7 more rows in fixed UI-SPEC order ... */}
```

**Save button** (mirror `SettingsTab.tsx:695-703` exactly — copy text changes only):
```tsx
{error && <p className="settings-panel__error">Could not save plugin settings — {error}</p>}
<div className="settings-panel__save-paths-row">
  <button
    className={`settings-panel__btn ${saved ? 'settings-panel__btn--saved' : 'settings-panel__btn--save'}`}
    onClick={handleSavePlugins}
    disabled={saving || saved || !pluginsLoaded}
  >
    {saving ? 'Saving…' : saved ? 'Saved!' : 'Save Plugins'}
  </button>
</div>
```

**Save handler** (mirror `SettingsTab.tsx:221-247` shape — change call to `SetPluginSettings`):
```typescript
async function handleSavePlugins() {
  if (!pluginConfig) return
  setSaving(true); setError(null)
  try {
    await SetPluginSettings(pluginConfig)
    setSaved(true)
    setTimeout(() => setSaved(false), 1500)
  } catch (err) {
    setError(err instanceof Error ? err.message : String(err))
  } finally {
    setSaving(false)
  }
}
```

**8-row order** (UI-SPEC line 32 — non-negotiable):
WebGL → Unicode 11 → Search → Web Links → Inline Images → Serialize → Clipboard → Progress

---

### `frontend/src/components/SettingsTab.tsx` (MODIFIED)

**Analog:** itself — single-line insertion after the existing Paths section's Save button.

**Insertion point:** After `SettingsTab.tsx:703` (closing `</div>` of `settings-panel__save-paths-row` for the Paths section), before `:705` (closing `</div>` of `settings-panel__body`).

```tsx
// CURRENT (SettingsTab.tsx:693-705)
{error && <p className="settings-panel__error">{error}</p>}

<div className="settings-panel__save-paths-row">
  <button ... >
    {saving ? 'Saving…' : saved ? 'Saved!' : 'Save Paths'}
  </button>
</div>

      </div>  {/* settings-panel__body */}
    </div>
  )
}

// MODIFIED — insert <PluginsSection /> between the Save Paths div and the body close
{/* ... existing Paths-section close ... */}
<div className="settings-panel__save-paths-row">
  <button ... >Save Paths</button>
</div>

<PluginsSection />  {/* NEW — Phase 92 insertion */}

      </div>
    </div>
```

Add at the top of `SettingsTab.tsx`:
```typescript
import { PluginsSection } from './PluginsSection'
```

---

### `frontend/src/App.tsx` (MODIFIED)

**Analogs:**
- `App.tsx:250-307` (existing `EventsOn` registrations inside the same `useEffect`)
- `App.tsx:401-415` (cleanup return — every `EventsOn` gets a matching `off*()` call)
- `App.tsx:865-872` (TerminalPanel render with all current props)
- `App.tsx:6, 30` (TerminalPanel + `EventsOn` imports — already present)

**State addition** (top of `App` function, near other top-level state):
```typescript
const [pluginConfig, setPluginConfig] = useState<PluginSettings | null>(null)
```

**Subscription pattern** (`App.tsx:250-307` — every existing `EventsOn` lives inside the same `useEffect(() => { ... }, [])`):
```typescript
// PATTERN SOURCE — App.tsx:250-307 (5 existing EventsOn registrations in one useEffect)
const offStatus = EventsOn('session:status', (data) => { ... })
const offHealth = EventsOn('tailscale:health', (h) => { ... })
const offDaemonError = EventsOn('daemon:error', (msg) => { ... })
const cancelTrayFocus = EventsOn('tray:focus-session', (sessionId) => { ... })
const offExit = EventsOn('session:exit', (data) => { ... })
const offQuit = EventsOn('app:quit-requested', () => { ... })

// CLEANUP — App.tsx:401-415
return () => {
  offStatus()
  offHealth()
  offDaemonError()
  cancelTrayFocus()
  offExit()
  offQuit()
  // ...
}
```

**FOR PHASE 92** — add at the same indent level inside that same `useEffect`:
```typescript
// Initial fetch (NOT inside EventsOn — only the change subscription is inside EventsOn)
GetPluginSettings().then(s => setPluginConfig(s)).catch(() => {})

const offPlugins = EventsOn('settings:plugins', (s: PluginSettings) => {
  setPluginConfig(s)
})

// And in the cleanup return, add:
offPlugins()
```

**Add `GetPluginSettings` to the import block** (`App.tsx:8-28`):
```typescript
import {
  // ... existing imports ...
  GetPluginSettings,
} from './wailsjs/go/main/App'
```

**Prop threading** — pass `pluginConfig` into every `<TerminalPanel>` render. Source (`App.tsx:865-872`):
```tsx
// CURRENT (App.tsx:865-872)
<TerminalPanel
  sessionId={tab.sessionId}
  isActive={isActive}
  relayPort={relayPort}
  fontSize={fontSizes[tab.sessionId] ?? DEFAULT_FONT_SIZE}
  onFontSizeChange={(delta) => handleFontSizeChange(tab.sessionId, delta)}
  theme={terminalTheme}
/>

// MODIFIED — append pluginConfig prop (drilled but unconsumed in Phase 92)
<TerminalPanel
  sessionId={tab.sessionId}
  isActive={isActive}
  relayPort={relayPort}
  fontSize={fontSizes[tab.sessionId] ?? DEFAULT_FONT_SIZE}
  onFontSizeChange={(delta) => handleFontSizeChange(tab.sessionId, delta)}
  theme={terminalTheme}
  pluginConfig={pluginConfig}
/>
```

Only ONE call site exists (`App.tsx:865`). The pattern is exact-match.

---

### `frontend/src/components/TerminalPanel.tsx` (MODIFIED — type-only)

**Analog:** itself — extend the existing `TerminalPanelProps` interface.

**Existing props interface** (`TerminalPanel.tsx:37-44`):
```typescript
interface TerminalPanelProps {
  sessionId: string
  isActive: boolean
  relayPort: number
  fontSize: number
  onFontSizeChange: (delta: number) => void
  theme: ITheme
}
```

**MODIFIED** — append the optional, unconsumed Phase 92 prop:
```typescript
import type { PluginSettings } from '../types/plugins'  // OR Wails-generated

interface TerminalPanelProps {
  sessionId: string
  isActive: boolean
  relayPort: number
  fontSize: number
  onFontSizeChange: (delta: number) => void
  theme: ITheme
  pluginConfig?: PluginSettings | null  // NEW — Phase 92 wires the prop; Phase 93 consumes it
}
```

**The function signature** (`TerminalPanel.tsx:51`) destructures props — add `pluginConfig` to the destructure list, then **do nothing with it** in Phase 92. The prop must be observable in source (so the source-inspection test asserts `expect(raw).toContain('pluginConfig')`) but no `useEffect` reads it.

---

### `frontend/src/components/__tests__/PluginsSection.test.tsx` (NEW — test)

**Analog:** `frontend/src/components/__tests__/SettingsTab.persistence.test.tsx:1-38` (the `?raw` import + string-assertion pattern).

**Existing source-inspection pattern** (`SettingsTab.persistence.test.tsx:1-38`):
```typescript
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'
import raw from '../../components/SettingsTab.tsx?raw'

const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

describe('SET-03: Save confirmation feedback', () => {
    it('has saved state variable', () => {
        expect(raw).toContain('setSaved')
    })

    it('uses --saved CSS modifier', () => {
        expect(raw).toContain('settings-panel__btn--saved')
    })
})
```

**FOR PHASE 92** — `PluginsSection.test.tsx` follows the same shape. Required assertions per UI-SPEC + RESEARCH §"Phase Requirements → Test Map":

```typescript
import { describe, it, expect } from 'vitest'
import raw from '../PluginsSection.tsx?raw'

describe('PUI-01: 8 toggle rows in UI-SPEC order', () => {
  it('contains all 8 plugin labels', () => {
    expect(raw).toContain('WebGL renderer')
    expect(raw).toContain('Unicode 11 widths')
    expect(raw).toContain('Find in scrollback')
    expect(raw).toContain('Clickable web links')
    expect(raw).toContain('Inline images')
    expect(raw).toContain('Save terminal as text')
    expect(raw).toContain('Clipboard (OSC 52)')
    expect(raw).toContain('Progress (OSC 9;4)')
  })

  it('renders rows in UI-SPEC order (Pitfall #5 guard)', () => {
    const order = ['webgl', 'unicode11', 'search', 'webLinks', 'image',
                   'serialize', 'clipboard', 'progress']
    for (let i = 0; i < order.length - 1; i++) {
      expect(raw.indexOf(order[i])).toBeLessThan(raw.indexOf(order[i + 1]))
    }
  })
})

describe('PUI-01: Three-state Save button', () => {
  it('has Save Plugins copy', () => {
    expect(raw).toContain("'Save Plugins'")
  })
  it('has Saving… and Saved! states', () => {
    expect(raw).toContain('Saving\\u2026')
    expect(raw).toContain("'Saved!'")
  })
  it('reuses settings-panel__btn--saved class', () => {
    expect(raw).toContain('settings-panel__btn--saved')
  })
})

describe('PLUG-02: pluginsLoaded flicker guard (Pitfall #3)', () => {
  it('gates toggle rows behind pluginsLoaded', () => {
    expect(raw).toContain('pluginsLoaded')
  })
})
```

---

## Shared Patterns

### Shared Pattern A — Defaults-Merge Settings Load (load-bearing for PLUG-02)

**Source:** new pattern (no existing analog inside the repo) — driven by Pitfall #14, captured in RESEARCH.md §"Pattern 1".

**Apply to:** `internal/daemon/engine.go:loadSettingsFromDisk` (replace the single `var s daemonSettings` line with a defaults-pre-populated literal).

**Key code:**
```go
// BEFORE (engine.go:93)
var s daemonSettings

// AFTER (Phase 92)
s := daemonSettings{
    SchemaVersion: CurrentSchemaVersion,
    Plugins:       defaultPluginSettings(),
}
```

**Then** `json.Unmarshal(data, &s)` proceeds as before. Go stdlib semantics: missing JSON keys leave existing struct fields untouched.

**Guarded by:** `engine_migration_test.go:TestSettingsMigrationV3_1ToV3_2` — the load-bearing CI gate (ROADMAP SC-1).

### Shared Pattern B — Wails RPC Triple (4-file extension)

**Source:** existing across 4 files for `StartMinimized`:
- engine method: `engine.go:377-390`
- HTTP route + handler: `api.go:70-71, 474-487`
- DaemonClient method: `client.go:111-124`
- Wails App binding: `app.go:415-432`

**Apply to:** every new daemon-persisted setting that the GUI reads/writes. Skipping any layer breaks the round-trip.

**For Phase 92:** mirror the four-file extension exactly for `PluginSettings` — but pass the **full struct** in the body (not a wrapper map), and the client uses HTTP PATCH (not PUT) for consistency with the surrounding routes (per RESEARCH.md Open Question 2).

### Shared Pattern C — EventsEmit lives ONLY in `app.go`

**Source:** `[VERIFIED: app.go grep — all 8 emits in app.go]` — Pitfall #2 in RESEARCH.md §"Common Pitfalls".

**Apply to:** any new Wails runtime event. The daemon process has no Wails runtime context; emit happens in the GUI process's `app.go` AFTER the daemon RPC succeeds.

**Key code:**
```go
// app.go (Phase 92)
func (a *App) SetPluginSettings(s daemon.PluginSettings) error {
    if err := a.client.SetPluginSettings(s); err != nil {
        return err
    }
    runtime.EventsEmit(a.ctx, "settings:plugins", s)  // <-- HERE, not in engine.go
    return nil
}
```

### Shared Pattern D — `EventsOn` subscription with cleanup in `App.tsx`

**Source:** `App.tsx:250-307` (registration) + `App.tsx:401-415` (cleanup return). Six existing subscriptions follow this shape.

**Apply to:** `frontend/src/App.tsx` only — every new global runtime-event subscription belongs in the existing `useEffect(() => { ... }, [])` block, with its `off*` function added to the cleanup return.

```typescript
// Inside the existing useEffect:
const offPlugins = EventsOn('settings:plugins', (s: PluginSettings) => {
  setPluginConfig(s)
})

// In the existing cleanup return:
return () => {
  // ... existing offs ...
  offPlugins()
}
```

### Shared Pattern E — `toggleLoaded` flicker guard

**Source:** `SettingsTab.tsx:88-89` (state) + `:154-159` (load) + `:338-349` (gate).

**Apply to:** every toggle whose `checked` state comes from an async daemon read. For Phase 92's `PluginsSection`, a single `pluginsLoaded` boolean gates ALL 8 rows (one round-trip fills 8 toggle states from the returned `PluginSettings` struct).

```tsx
// All 8 toggle rows live inside one guard:
{pluginsLoaded && pluginConfig && (
  <>
    {/* 8 <label> rows */}
  </>
)}
```

The hidden `<input type="checkbox">` elements MAY render unconditionally (so test selectors can find them) — only the visible `<label>` markup is gated.

### Shared Pattern F — Three-State Save Button

**Source:** `SettingsTab.tsx:695-703` (button JSX) + `:221-247` (handler with `setSaved(true)` + `setTimeout(..., 1500)`).

**Apply to:** the new `Save Plugins` button. Reuse `.settings-panel__btn--save` / `.settings-panel__btn--saved` classes verbatim; reuse `.settings-panel__save-paths-row` container class (yes, the class name is misleading — UI-SPEC §"Component Inventory" line 224 confirms it is the established Save-button cadence container, not a CSS rename target).

### Shared Pattern G — Source-Inspection Vitest Tests

**Source:** `frontend/src/components/__tests__/SettingsTab.persistence.test.tsx:1-38`.

**Apply to:** every new vitest test in Phase 92. The codebase precedent for visual-contract testing is `?raw` imports + `expect(raw).toContain('...')` — NOT `render()` / `userEvent` / DOM assertions. WebGL/Canvas-touching components don't run cleanly in jsdom; source-inspection is the documented v3.1 precedent.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `tests/fixtures/settings_v3.1.json` | config (test fixture) | file-I/O | First fixture file in the repo. The `tests/` directory exists (with `build-script.test.sh`) but `tests/fixtures/` does not. Fixture shape derived from `daemonSettings` JSON tags at `engine.go:66-71`, NOT from an existing fixture. |

---

## Metadata

**Analog search scope:**
- `internal/daemon/` — engine.go, api.go, client.go, engine_settings_test.go
- `app.go` — (root, Wails bindings)
- `frontend/src/` — App.tsx, components/SettingsTab.tsx, components/TerminalPanel.tsx, components/__tests__/SettingsTab.persistence.test.tsx
- `frontend/src/wailsjs/go/main/App.d.ts` — auto-generated bindings (verified GetStartMinimized/SetStartMinimized presence)
- `tests/` — confirmed only `build-script.test.sh`; `fixtures/` does not exist
- `frontend/src/types/` — confirmed does not exist (per RESEARCH Assumption A5 — note: research said it exists; verified 2026-05-04 it does NOT, so the planner must create the directory)

**Files scanned:** 8 source files + 1 generated `.d.ts` + 2 directory listings

**Pattern extraction date:** 2026-05-04

**Confidence:** HIGH — every Phase 92 file maps to a verified existing analog with concrete file:line references. Only the defaults-merge pattern (Shared Pattern A) is novel within this codebase, and its rationale + safety net (the migration fixture test) are already specified in RESEARCH.md.
