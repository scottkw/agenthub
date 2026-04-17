# Phase 82: Minimize to Tray - Pattern Map

**Mapped:** 2026-04-16
**Files analyzed:** 6 modified files
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/daemon/engine.go` | service | CRUD | self (existing `UpdateCLIPath`/`GetCLIPaths` pattern) | exact |
| `internal/daemon/api.go` | middleware/route | request-response | self (existing `handleGetCLIPaths`/`handleUpdateCLIPath`) | exact |
| `internal/daemon/client.go` | service/client | request-response | self (existing `GetCLIPaths`/`UpdateCLIPath`) | exact |
| `internal/daemon/types.go` | model | — | self (existing `UpdateCLIPathRequest`, `StatusResponse`) | exact |
| `app.go` | controller | request-response | self (existing `GetCLIPaths`/`UpdateCLIPath` Wails bindings + `domReady`) | exact |
| `frontend/src/components/SettingsTab.tsx` | component | request-response | self (existing `useEffect` + async handler pattern) | exact |
| `frontend/src/style.css` | config/style | — | self (existing `.settings-panel__*` rules) | exact |

---

## Pattern Assignments

### `internal/daemon/engine.go` — extend `daemonSettings` + engine

**Role:** service, CRUD

**Analog:** same file — `daemonSettings` struct, `loadSettingsFromDisk`, `saveSettingsToDisk`, `GetCLIPaths`, `UpdateCLIPath`

**Struct extension pattern** (engine.go lines 21-37 and 63-65):
```go
// SessionEngine — add startMinimized bool field alongside cliPaths:
type SessionEngine struct {
    // ... existing fields ...
    cliPaths       map[string]string // cli name -> custom path override
    startMinimized bool              // NEW: persisted start-minimized preference
}

// daemonSettings — add StartMinimized field:
type daemonSettings struct {
    CLIPaths       map[string]string `json:"cliPaths,omitempty"`
    StartMinimized bool              `json:"startMinimized,omitempty"`
}
```

**loadSettingsFromDisk pattern** (engine.go lines 74-87 — add one line after the cliPaths loop):
```go
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
    data, err := os.ReadFile(settingsPath(dir))
    if err != nil {
        return
    }
    var s daemonSettings
    if json.Unmarshal(data, &s) == nil && s.CLIPaths != nil {
        e.mu.Lock()
        for k, v := range s.CLIPaths {
            e.cliPaths[k] = v
        }
        e.startMinimized = s.StartMinimized  // NEW line
        e.mu.Unlock()
    }
}
```

**saveSettingsToDisk pattern** (engine.go lines 91-98 — add StartMinimized to struct literal):
```go
func (e *SessionEngine) saveSettingsToDisk() {
    // Caller holds e.mu.Lock()
    s := daemonSettings{
        CLIPaths:       e.cliPaths,
        StartMinimized: e.startMinimized,  // NEW field
    }
    data, err := json.Marshal(s)
    if err != nil {
        return
    }
    _ = os.WriteFile(settingsPath(e.configDir), data, 0600)
}
```

**Engine get/set methods pattern** (engine.go lines 276-287 — copy `GetCLIPaths`/`UpdateCLIPath` lock pattern):
```go
// GetStartMinimized returns the persisted start-minimized preference.
func (e *SessionEngine) GetStartMinimized() bool {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.startMinimized
}

// SetStartMinimized updates and persists the start-minimized preference.
func (e *SessionEngine) SetStartMinimized(val bool) {
    e.mu.Lock()
    e.startMinimized = val
    e.saveSettingsToDisk()
    e.mu.Unlock()
}
```

---

### `internal/daemon/api.go` — new routes + handlers

**Role:** middleware/route, request-response

**Analog:** same file — `handleGetCLIPaths` (line 319), `handleUpdateCLIPath` (line 327), `registerRoutes` (line 43)

**Route registration pattern** (api.go lines 43-66 — add two lines to `registerRoutes`):
```go
func (a *API) registerRoutes() {
    // ... existing routes ...
    a.mux.HandleFunc("GET /settings/cli-paths", a.handleGetCLIPaths)
    a.mux.HandleFunc("PATCH /settings/cli-paths/{name}", a.handleUpdateCLIPath)
    // NEW:
    a.mux.HandleFunc("GET /settings/start-minimized", a.handleGetStartMinimized)
    a.mux.HandleFunc("PATCH /settings/start-minimized", a.handleSetStartMinimized)
}
```

**GET handler pattern** (api.go lines 319-325 — copy `handleGetCLIPaths` structure):
```go
func (a *API) handleGetStartMinimized(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]bool{"startMinimized": a.engine.GetStartMinimized()})
}
```

**PATCH handler pattern** (api.go lines 327-339 — copy `handleUpdateCLIPath` decode + 204 pattern):
```go
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
```

**writeJSON helper** (api.go line 205 — already exists, use as-is):
```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}
```

---

### `internal/daemon/client.go` — new client methods

**Role:** service/client, request-response

**Analog:** same file — `GetCLIPaths` (lines 88-94), `UpdateCLIPath` (lines 97-99), `doJSON` helper (lines 189-225)

**GET client method pattern** (client.go lines 88-94):
```go
func (c *DaemonClient) GetCLIPaths() (map[string]string, error) {
    var paths map[string]string
    if err := c.doJSON(http.MethodGet, "/settings/cli-paths", nil, &paths); err != nil {
        return nil, err
    }
    return paths, nil
}
```
Copy for `GetStartMinimized`:
```go
func (c *DaemonClient) GetStartMinimized() (bool, error) {
    var resp map[string]bool
    if err := c.doJSON(http.MethodGet, "/settings/start-minimized", nil, &resp); err != nil {
        return false, err
    }
    return resp["startMinimized"], nil
}
```

**PATCH client method pattern** (client.go lines 97-99):
```go
func (c *DaemonClient) UpdateCLIPath(name, path string) error {
    return c.doJSON(http.MethodPatch, "/settings/cli-paths/"+name, UpdateCLIPathRequest{Path: path}, nil)
}
```
Copy for `SetStartMinimized`:
```go
func (c *DaemonClient) SetStartMinimized(val bool) error {
    return c.doJSON(http.MethodPatch, "/settings/start-minimized",
        map[string]bool{"startMinimized": val}, nil)
}
```

**doJSON helper** (client.go lines 189-225 — no changes needed; already handles nil body, nil result, 204 No Content):
```go
// Key behaviors of doJSON to rely on:
// - body == nil → no Content-Type header, no request body
// - result == nil OR status == 204 → skip decode
// - status >= 400 → return error with method, path, status, body
```

---

### `internal/daemon/types.go` — no new types needed

**Role:** model

**Analog:** same file — `UpdateCLIPathRequest` (line 49), `StatusResponse` (line 37)

The `handleSetStartMinimized` handler uses an anonymous inline struct for decoding (consistent with existing pattern where the request type is simple and single-use). No new exported type is required. The GET response is `map[string]bool` (same pattern as `handleGetLocalPassword` using `map[string]string`).

If a named type is preferred for consistency, follow this pattern:
```go
// StartMinimizedRequest is the request body for PATCH /settings/start-minimized.
type StartMinimizedRequest struct {
    StartMinimized bool `json:"startMinimized"`
}
```

---

### `app.go` — new Wails bindings + `domReady` gate

**Role:** controller, request-response

**Analog:** same file — `GetCLIPaths` (lines 316-321), `UpdateCLIPath` (lines 307-312), `domReady` (lines 77-80)

**`nil`-guard + delegate pattern** (app.go lines 307-321 — the canonical Wails binding shape):
```go
// GetCLIPaths returns all stored CLI path overrides.
func (a *App) GetCLIPaths() (map[string]string, error) {
    if a.client == nil {
        return nil, fmt.Errorf("daemon not connected")
    }
    return a.client.GetCLIPaths()
}

// UpdateCLIPath stores a custom executable path for the named CLI.
func (a *App) UpdateCLIPath(name, path string) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.UpdateCLIPath(name, path)
}
```

New bindings follow this exactly:
```go
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

**`domReady` modification** (app.go lines 77-80 — current unconditional show):
```go
// CURRENT (lines 77-80):
func (a *App) domReady(ctx context.Context) {
    runtime.WindowShow(ctx)
    a.setDockVisible(true)
}

// AFTER — gate on persisted preference, fallback to show on any error:
func (a *App) domReady(ctx context.Context) {
    startMinimized := false
    if a.client != nil {
        if val, err := a.client.GetStartMinimized(); err == nil {
            startMinimized = val
        }
    }
    if !startMinimized {
        runtime.WindowShow(ctx)
        a.setDockVisible(true)
    }
}
```

Note: `a.setDockVisible` is safe to call from `app.go` on all platforms. The darwin-only definition in `tray.go` (line 1 `//go:build darwin`) is balanced by a no-op stub on other platforms. The existing `domReady` already calls it unconditionally — confirm stub location in `tray_linux.go` or `tray_common.go` before implementation.

---

### `frontend/src/components/SettingsTab.tsx` — Behavior section

**Role:** component, request-response

**Analog:** same file — `useEffect` for `GetCLIPaths` (lines 104-110), `handleSaveCLIPaths` async handler (lines 178-201), state pattern (lines 59-79)

**Import addition pattern** (SettingsTab.tsx lines 1-16 — add two new Wails imports):
```tsx
import {
  // ... existing imports ...
  UpdateCLIPath,
  GetCLIPaths,
  // NEW:
  GetStartMinimized,
  SetStartMinimized,
} from '../wailsjs/go/main/App'
```

**State pattern** (SettingsTab.tsx lines 59-79 — copy the server loading/error/value triple):
```tsx
// Existing pattern (server state):
const [isServerRunning, setIsServerRunning] = useState(false)
const [serverLoading, setServerLoading] = useState(false)
const [serverError, setServerError] = useState<string | null>(null)

// New state for toggle (same triple + loaded gate):
const [startMinimized, setStartMinimized] = useState(false)
const [toggleLoaded, setToggleLoaded] = useState(false)
const [toggleSaving, setToggleSaving] = useState(false)
const [toggleError, setToggleError] = useState<string | null>(null)
```

**`useEffect` load pattern** (SettingsTab.tsx lines 104-110 — copy `GetCLIPaths` mount effect):
```tsx
// Existing:
useEffect(() => {
  GetCLIPaths().then(paths => {
    if (paths && Object.keys(paths).length > 0) {
      setCustomPaths(prev => ({ ...prev, ...paths }))
    }
  }).catch(() => {})
}, [])

// New:
useEffect(() => {
  GetStartMinimized().then(val => {
    setStartMinimized(val)
    setToggleLoaded(true)
  }).catch(() => setToggleLoaded(true))
}, [])
```

**Async handler pattern** (SettingsTab.tsx lines 178-201 — copy `handleSaveCLIPaths` try/catch/finally):
```tsx
// Existing pattern:
async function handleSaveCLIPaths() {
  setSaving(true)
  setError(null)
  try {
    // ... calls ...
    setSaved(true)
    setTimeout(() => setSaved(false), 1500)
  } catch (err) {
    setError(err instanceof Error ? err.message : String(err))
  } finally {
    setSaving(false)
  }
}

// New handler (non-optimistic per UI-SPEC):
async function handleToggleMinimized() {
  const next = !startMinimized
  setToggleSaving(true)
  setToggleError(null)
  try {
    await SetStartMinimized(next)
    setStartMinimized(next)
  } catch (err) {
    setToggleError('Could not save preference \u2014 ' + (err instanceof Error ? err.message : String(err)))
  } finally {
    setToggleSaving(false)
  }
}
```

**Section heading + field-group JSX pattern** (SettingsTab.tsx lines 247-261 — Appearance section):
```tsx
{/* Existing Appearance section: */}
<h3>Appearance</h3>
<div className="settings-panel__field-group">
  <label className="settings-panel__label">Terminal Theme</label>
  {/* ... */}
  <p className="settings-panel__description" style={{ marginTop: '0.5rem' }}>...</p>
</div>

{/* New Behavior section — placed BEFORE Appearance (becomes h3:first-child → no border-top): */}
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

**Error display pattern** (SettingsTab.tsx line 388 — `{serverError && <p className="settings-panel__error">{serverError}</p>}`):
```tsx
{toggleError && <p className="settings-panel__error">{toggleError}</p>}
```

---

### `frontend/src/style.css` — toggle CSS rules

**Role:** config/style

**Analog:** same file — `.settings-panel__field-group` (line 496), `.settings-panel__label` (line 502), `.settings-panel__description` (line 509), `.settings-panel__error` (line 421)

**Existing rules that the toggle reuses without changes** (style.css lines 421-509):
```css
.settings-panel__error {
  margin-top: 12px;
  color: #f7768e;
  font-size: 12px;
}
.settings-panel__field-group { margin-bottom: 16px; }
.settings-panel__label { /* 13px, #9aa5ce, semibold */ }
.settings-panel__description { /* 12px, #9aa5ce */ }
```

**New CSS rules to append** (after existing `.settings-panel__*` block, per UI-SPEC):
```css
/* Toggle / switch control (Phase 82) */
.settings-panel__toggle-input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
.settings-panel__toggle-row {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  min-height: 44px;
  user-select: none;
}
.settings-panel__toggle-track {
  position: relative;
  width: 36px;
  height: 20px;
  border-radius: 10px;
  background-color: #16161e;
  border: 1px solid #292e42;
  flex-shrink: 0;
  transition: background-color 0.15s, border-color 0.15s;
}
.settings-panel__toggle-thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background-color: #565f89;
  transition: transform 0.15s, background-color 0.15s;
}
.settings-panel__toggle-row--checked .settings-panel__toggle-track {
  background-color: #7aa2f7;
  border-color: #7aa2f7;
}
.settings-panel__toggle-row--checked .settings-panel__toggle-thumb {
  transform: translateX(16px);
  background-color: #1a1b26;
}
.settings-panel__toggle-label {
  font-size: 13px;
  font-weight: 400;
  color: #c0caf5;
  line-height: 1.5;
}
```

---

## Shared Patterns

### nil-guard on `a.client` (app.go, used throughout)
**Source:** `app.go` — every Wails-bound method that calls `a.client`
**Apply to:** `GetStartMinimized`, `SetStartMinimized`, `domReady`
```go
if a.client == nil {
    return false // or return fmt.Errorf("daemon not connected")
}
```

### `doJSON` request/response helper (client.go lines 189-225)
**Source:** `internal/daemon/client.go`
**Apply to:** `GetStartMinimized`, `SetStartMinimized` client methods
- Pass `nil` as body for GET; pass a struct/map as body for PATCH
- Pass `nil` as result for 204 No Content responses
- Handles status >= 400 → returns descriptive error automatically

### Error display (SettingsTab.tsx, `.settings-panel__error`)
**Source:** `frontend/src/components/SettingsTab.tsx` line 388
**Apply to:** toggle error state below description
```tsx
{toggleError && <p className="settings-panel__error">{toggleError}</p>}
```

### `writeJSON` API helper (api.go line 205)
**Source:** `internal/daemon/api.go`
**Apply to:** `handleGetStartMinimized`
```go
writeJSON(w, http.StatusOK, map[string]bool{"startMinimized": a.engine.GetStartMinimized()})
```

### Mutex discipline (engine.go)
**Source:** `internal/daemon/engine.go` — `GetCLIPaths` (RLock), `UpdateCLIPath` (Lock + saveSettingsToDisk inside lock)
**Apply to:** `GetStartMinimized` (RLock/RUnlock), `SetStartMinimized` (Lock, mutate, saveSettingsToDisk, Unlock)

---

## No Analog Found

All modified files in this phase have direct analogs in the same file. No new files are created. No file lacks a pattern match.

---

## Post-Implementation Step

After adding `GetStartMinimized` and `SetStartMinimized` to `app.go`, the Wails TypeScript bindings must be regenerated before the frontend imports resolve:

```
wails generate module
```

Or run `wails dev` which auto-regenerates. This step must be included in the implementation plan as a task between the `app.go` changes and the `SettingsTab.tsx` changes.

---

## Metadata

**Analog search scope:** `app.go`, `internal/daemon/engine.go`, `internal/daemon/api.go`, `internal/daemon/client.go`, `internal/daemon/types.go`, `frontend/src/components/SettingsTab.tsx`, `frontend/src/style.css`, `tray.go`
**Files scanned:** 8
**Pattern extraction date:** 2026-04-16
