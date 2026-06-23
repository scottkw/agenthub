# Phase 150: Shell-Sharing Warning Toggle — Pattern Map

**Mapped:** 2026-06-23
**Files analyzed:** 10 files to be created or modified
**Analogs found:** 10 / 10

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/daemon/engine.go` | model/service | CRUD | self (add field after `shellWebShareWarned`) | exact |
| `internal/daemon/api.go` | controller | request-response | self (`handleGetShellWebShareWarned` / `handleUpdateShellWebShareWarned`) | exact |
| `internal/daemon/client.go` | service | request-response | self (`GetShellWebShareWarned` / `SetShellWebShareWarned`) | exact |
| `app.go` | controller/binding | request-response | self (`GetShellWebShareWarned` / `SetShellWebShareWarned` at lines 501–519) | exact |
| `frontend/src/wailsjs/go/main/App.js` | utility | request-response | self (lines 15–16, `GetShellWebShareWarned` / `SetShellWebShareWarned`) | exact |
| `frontend/src/wailsjs/go/main/App.d.ts` | utility | request-response | self (lines 39–40, `GetShellWebShareWarned` / `SetShellWebShareWarned`) | exact |
| `frontend/src/App.tsx` | component | event-driven | self (`shellWebShareWarned` state + `handleToggleWeb` gate + `handleShellWebShareConfirm`) | exact |
| `frontend/src/components/SettingsTab.tsx` | component | request-response | self (`autoCloseSession` state machine at lines 107–110, 181–186, 326–338, 412–438) | exact |
| `frontend/src/components/Hub/SessionShareModal.tsx` | component | request-response | self (`handleShareToggle` at lines 186–199; `HomeDirWriteWarning` integration for inline-warning pattern) | exact |
| `frontend/src/components/SettingsSearch.tsx` | utility | transform | self (line 29, `Auto-close tab on exit` SEARCH_INDEX entry) | exact |

---

## Pattern Assignments

### `internal/daemon/engine.go` (model, CRUD)

**Role:** Add `shellWebShareWarningEnabled` engine field + `daemonSettings` JSON field + load/save + Get/Set accessors.

**Analog for struct field** (lines 44–47):
```go
startMinimized      bool            // persisted start-minimized preference
shellWebShareWarned bool            // Phase 101 SHELL-08: user has acknowledged the shell web-share security banner
shellPath           string          // Phase 107 SHELL-11: user-configured shell binary path; empty = use platform default
autoCloseSession    *bool           // nil = default (true); persisted pointer
```

**New field to add** (after line 45, same struct):
```go
shellWebShareWarningEnabled *bool   // Phase 150 SET-01: master warning switch; nil = default (true per D-08)
```

**Analog for `daemonSettings` JSON struct** (lines 107–116):
```go
type daemonSettings struct {
    CLIPaths            map[string]string `json:"cliPaths,omitempty"`
    StartMinimized      bool              `json:"startMinimized,omitempty"`
    ShellWebShareWarned bool              `json:"shellWebShareWarned,omitempty"`
    ShellPath           string            `json:"shellPath,omitempty"`
    AutoCloseSession    *bool             `json:"autoCloseSession,omitempty"`
    FilesWrite          bool              `json:"filesWrite,omitempty"`
    Plugins             PluginSettings    `json:"plugins"`
    SchemaVersion       int               `json:"schemaVersion"`
}
```

**New `daemonSettings` field** (add after `AutoCloseSession`):
```go
ShellWebShareWarningEnabled *bool `json:"shellWebShareWarningEnabled,omitempty"`
```

**CRITICAL:** Must be `*bool`, NOT `bool`. The `omitempty` tag omits false (zero value), making it indistinguishable from "not set". An absent key in old settings.json deserializes as nil pointer → apply default `true`. If `bool` is used instead, an old settings.json silently defaults new users to OFF (breaks D-08).

**Analog for `loadSettingsFromDisk`** (lines 194–197 — the auto-close `*bool` nil-check pattern):
```go
e.startMinimized = s.StartMinimized
e.shellWebShareWarned = s.ShellWebShareWarned
e.shellPath = s.ShellPath
e.autoCloseSession = s.AutoCloseSession
```

**New lines in `loadSettingsFromDisk`** (after `e.autoCloseSession = s.AutoCloseSession`):
```go
e.shellWebShareWarningEnabled = s.ShellWebShareWarningEnabled
// nil pointer = not set in settings.json = fresh install → default ON (D-08).
// Unlike shellWebShareWarned (plain bool, default false), this new field defaults
// to true, so we must use *bool and handle nil explicitly.
if e.shellWebShareWarningEnabled == nil {
    t := true
    e.shellWebShareWarningEnabled = &t
}
```

**Analog for `saveSettingsToDisk`** (lines 218–233):
```go
func (e *SessionEngine) saveSettingsToDisk() {
    s := daemonSettings{
        CLIPaths:            e.cliPaths,
        StartMinimized:      e.startMinimized,
        ShellWebShareWarned: e.shellWebShareWarned,
        ShellPath:           e.shellPath,
        AutoCloseSession:    e.autoCloseSession,
        FilesWrite:          e.filesWriteDefault,
        Plugins:             e.pluginSettings,
        SchemaVersion:       CurrentSchemaVersion,
    }
    ...
}
```

**New field in `saveSettingsToDisk`** literal (add after `AutoCloseSession`):
```go
ShellWebShareWarningEnabled: e.shellWebShareWarningEnabled,
```

**Analog for Get/Set accessors** — `GetAutoCloseSession` + `SetAutoCloseSession` (lines 1084–1101):
```go
func (e *SessionEngine) GetAutoCloseSession() bool {
    e.mu.RLock()
    defer e.mu.RUnlock()
    if e.autoCloseSession == nil {
        return true // default: enabled
    }
    return *e.autoCloseSession
}

func (e *SessionEngine) SetAutoCloseSession(val bool) {
    e.mu.Lock()
    e.autoCloseSession = &val
    e.saveSettingsToDisk()
    e.mu.Unlock()
}
```

**New accessors to add** (after `SetShellWebShareWarned` at line 1011):
```go
// GetShellWebShareWarningEnabled returns the warning-enabled master switch.
// Returns true when not set (nil pointer = default ON per D-08).
func (e *SessionEngine) GetShellWebShareWarningEnabled() bool {
    e.mu.RLock()
    defer e.mu.RUnlock()
    if e.shellWebShareWarningEnabled == nil {
        return true
    }
    return *e.shellWebShareWarningEnabled
}

// SetShellWebShareWarningEnabled persists the warning-enabled master switch.
// D-03 re-arm: when enabling (val=true), atomically resets shellWebShareWarned
// to false so the one-time banner shows again on the next shell web-share.
// Both fields are written in a single saveSettingsToDisk call (atomic).
func (e *SessionEngine) SetShellWebShareWarningEnabled(val bool) error {
    e.mu.Lock()
    e.shellWebShareWarningEnabled = &val
    if val {
        // Re-arm (D-03): reset the one-time acknowledged flag.
        e.shellWebShareWarned = false
    }
    e.saveSettingsToDisk()
    e.mu.Unlock()
    return nil
}
```

**Key divergence from `SetAutoCloseSession`:** The new `Set` method returns `error` (matches `SetShellWebShareWarned` signature) and has the re-arm side-effect. `SetAutoCloseSession` returns no error — the new method must return `error` for symmetry with the rest of the `shellWebShare*` family and to match the api.go error-handling expectation.

---

### `internal/daemon/api.go` (controller, request-response)

**Analog:** `handleGetShellWebShareWarned` / `handleUpdateShellWebShareWarned` (lines 725–746).

**Route registrations to add** (after line 113):
```go
// Analog (lines 112–113):
a.mux.HandleFunc("GET /settings/shell-web-share-warned", a.handleGetShellWebShareWarned)
a.mux.HandleFunc("PATCH /settings/shell-web-share-warned", a.handleUpdateShellWebShareWarned)

// New (insert at line ~114, grouping with the warned pair):
a.mux.HandleFunc("GET /settings/shell-web-share-warning-enabled", a.handleGetShellWebShareWarningEnabled)
a.mux.HandleFunc("PATCH /settings/shell-web-share-warning-enabled", a.handleSetShellWebShareWarningEnabled)
```

**GET handler pattern** (analog lines 725–729):
```go
func (a *API) handleGetShellWebShareWarned(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]bool{"value": a.engine.GetShellWebShareWarned()})
}
```

**PATCH handler pattern** (analog lines 731–746):
```go
func (a *API) handleUpdateShellWebShareWarned(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Value bool `json:"value"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    if err := a.engine.SetShellWebShareWarned(req.Value); err != nil {
        http.Error(w, fmt.Sprintf("persist: %v", err), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

**New handlers** (mirror exactly, swap method names and route string):
```go
func (a *API) handleGetShellWebShareWarningEnabled(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]bool{"value": a.engine.GetShellWebShareWarningEnabled()})
}

func (a *API) handleSetShellWebShareWarningEnabled(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Value bool `json:"value"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    if err := a.engine.SetShellWebShareWarningEnabled(req.Value); err != nil {
        http.Error(w, fmt.Sprintf("persist: %v", err), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

---

### `internal/daemon/client.go` (service, request-response)

**Analog:** `GetShellWebShareWarned` / `SetShellWebShareWarned` (lines 166–181):
```go
func (c *DaemonClient) GetShellWebShareWarned() (bool, error) {
    var resp map[string]bool
    if err := c.doJSON(http.MethodGet, "/settings/shell-web-share-warned", nil, &resp); err != nil {
        return false, err
    }
    return resp["value"], nil
}

func (c *DaemonClient) SetShellWebShareWarned(val bool) error {
    return c.doJSON(http.MethodPatch, "/settings/shell-web-share-warned",
        map[string]bool{"value": val}, nil)
}
```

**New methods to add** (after line 181):
```go
// GetShellWebShareWarningEnabled returns the warning-enabled master switch.
// Phase 150 SET-01.
func (c *DaemonClient) GetShellWebShareWarningEnabled() (bool, error) {
    var resp map[string]bool
    if err := c.doJSON(http.MethodGet, "/settings/shell-web-share-warning-enabled", nil, &resp); err != nil {
        return false, err
    }
    return resp["value"], nil
}

// SetShellWebShareWarningEnabled persists the warning-enabled master switch.
// When val=true the engine atomically resets shellWebShareWarned (D-03 re-arm).
// Phase 150 SET-01.
func (c *DaemonClient) SetShellWebShareWarningEnabled(val bool) error {
    return c.doJSON(http.MethodPatch, "/settings/shell-web-share-warning-enabled",
        map[string]bool{"value": val}, nil)
}
```

---

### `app.go` (Wails binding, request-response)

**Analog:** `GetShellWebShareWarned` / `SetShellWebShareWarned` (lines 501–519):
```go
func (a *App) GetShellWebShareWarned() bool {
    if a.client == nil {
        return false
    }
    warned, err := a.client.GetShellWebShareWarned()
    if err != nil {
        return false
    }
    return warned
}

func (a *App) SetShellWebShareWarned(v bool) error {
    if a.client == nil {
        return nil
    }
    return a.client.SetShellWebShareWarned(v)
}
```

**Secondary analog:** `GetAutoCloseSession` (lines 600–611) — note the safe-default is `true` (not `false`) when the daemon is disconnected, matching D-08's default-ON requirement.

**New methods to add** (after `SetShellWebShareWarned` at line 519):
```go
// GetShellWebShareWarningEnabled returns the warning-enabled master switch.
// Returns true (enabled) when the daemon is not connected — safe degradation per D-08.
// Phase 150 SET-01.
func (a *App) GetShellWebShareWarningEnabled() bool {
    if a.client == nil {
        return true  // default ON (not false like GetShellWebShareWarned)
    }
    val, err := a.client.GetShellWebShareWarningEnabled()
    if err != nil {
        return true  // default: enabled (safe degradation)
    }
    return val
}

// SetShellWebShareWarningEnabled persists the warning-enabled master switch.
// When enabling, the daemon atomically resets shellWebShareWarned (D-03 re-arm).
// Phase 150 SET-01.
func (a *App) SetShellWebShareWarningEnabled(v bool) error {
    if a.client == nil {
        return nil
    }
    return a.client.SetShellWebShareWarningEnabled(v)
}
```

**Key divergence from `GetShellWebShareWarned`:** The nil/error fallback is `true` not `false`, because this setting defaults ON (D-08). Copy `GetAutoCloseSession`'s `return true` fallback pattern, not `GetShellWebShareWarned`'s `return false`.

---

### `frontend/src/wailsjs/go/main/App.js` (utility, request-response)

**Pattern:** Manual maintenance — no `wails generate` step. Every prior binding addition was a manual edit in the same commit as `app.go` changes.

**Analog insertion point** — after line 16 (`SetShellWebShareWarned`):
```js
// Phase 101-01 — persisted "user has been warned about shell web-share" flag.
export const GetShellWebShareWarned  = ()    => Call('main.App.GetShellWebShareWarned', [])
export const SetShellWebShareWarned  = (v)   => Call('main.App.SetShellWebShareWarned', [v])
```

**New lines to insert** (after line 16):
```js
// Phase 150 SET-01 — warning-enabled master switch (default ON).
export const GetShellWebShareWarningEnabled = ()  => Call('main.App.GetShellWebShareWarningEnabled', [])
export const SetShellWebShareWarningEnabled = (v) => Call('main.App.SetShellWebShareWarningEnabled', [v])
```

---

### `frontend/src/wailsjs/go/main/App.d.ts` (utility, request-response)

**Analog insertion point** — after line 40 (`SetShellWebShareWarned`):
```ts
// Phase 101-01 — persisted "user has been warned about shell web-share" flag.
export function GetShellWebShareWarned(): Promise<boolean>
export function SetShellWebShareWarned(v: boolean): Promise<void>
```

**New lines to insert** (after line 40):
```ts
// Phase 150 SET-01 — warning-enabled master switch (default ON).
export function GetShellWebShareWarningEnabled(): Promise<boolean>
export function SetShellWebShareWarningEnabled(v: boolean): Promise<void>
```

---

### `frontend/src/App.tsx` (component, event-driven)

**Three changes required:**

**Change 1 — new state declaration** (analog: `shellWebShareWarned` at line 138):
```ts
// Analog (line 138):
const [shellWebShareWarned, setShellWebShareWarned] = useState(false)

// New (add after line 141, after pendingShellWebToggle):
const [shellWebShareWarningEnabled, setShellWebShareWarningEnabled] = useState(true)
```

**Change 2 — hydrate from daemon on mount** (analog: lines 532–541):
```ts
// Analog (lines 536–541):
GetShellWebShareWarned()
  .then((v) => setShellWebShareWarned(v))
  .catch((err) => {
    console.warn('[App] GetShellWebShareWarned failed:', err)
    setShellWebShareWarned(false)
  })
```

**New lines** (add after the above block, still inside the same mount effect):
```ts
GetShellWebShareWarningEnabled()
  .then((v) => setShellWebShareWarningEnabled(v))
  .catch((err) => {
    console.warn('[App] GetShellWebShareWarningEnabled failed:', err)
    setShellWebShareWarningEnabled(true) // default ON (safe degradation per D-08)
  })
```

**Change 3 — add `warningEnabled` gate to `handleToggleWeb`** (analog: line 864):
```ts
// Current (line 864):
if (tab && SHELL_CLIS.has(tab.cli) && !shellWebShareWarned) {

// After this phase (add shellWebShareWarningEnabled AND clause):
if (tab && SHELL_CLIS.has(tab.cli) && shellWebShareWarningEnabled && !shellWebShareWarned) {
```

Also add `shellWebShareWarningEnabled` to the `useCallback` dependency array at line 876:
```ts
// Current deps:
}, [webEnabled, shellWebShareWarned, tabs])

// New deps:
}, [webEnabled, shellWebShareWarned, shellWebShareWarningEnabled, tabs])
```

**Change 4 — thread `shellWebShareWarningEnabled` and an `onWarnEnabledChange` callback to SettingsTab** (to enable re-hydration after re-arm):

The prop `onShellWarnEnabledChange` allows SettingsTab to signal App.tsx that `shellWebShareWarningEnabled` changed to `true`, triggering a re-fetch of `shellWebShareWarned` (pitfall 2 mitigation):
```ts
// In SettingsTab render call (wherever App.tsx renders SettingsTab):
<SettingsTab
  {...existingProps}
  onShellWarnEnabledChange={(enabled: boolean) => {
    setShellWebShareWarningEnabled(enabled)
    if (enabled) {
      // D-03 re-arm: daemon reset shellWebShareWarned to false; sync local state.
      GetShellWebShareWarned()
        .then(setShellWebShareWarned)
        .catch(() => setShellWebShareWarned(false))
    }
  }}
/>
```

**Change 5 — thread `shellWebShareWarned` + `shellWebShareWarningEnabled` to SessionShareModal** (for cross-surface parity, D-09):
```ts
// Analog: HubPanel receives props from App.tsx. Thread through to SessionShareModal.
// Add these props to the SessionShareModal call site in HubPanel:
shellWebShareWarned={shellWebShareWarned}
shellWebShareWarningEnabled={shellWebShareWarningEnabled}
onShellWebShareConfirm={handleShellWebShareConfirm}
onShellWebShareCancel={handleShellWebShareCancel}
```

---

### `frontend/src/components/SettingsTab.tsx` (component, request-response)

**Three additions required:**

**Addition 1 — state declarations** (analog: `autoCloseSession` state quartet at lines 107–110):
```ts
// Analog (lines 107–110):
const [autoCloseSession, setAutoCloseSession] = useState(true)
const [autoCloseLoaded, setAutoCloseLoaded] = useState(false)
const [autoCloseSaving, setAutoCloseSaving] = useState(false)
const [autoCloseError, setAutoCloseError] = useState<string | null>(null)
```

**New state** (add after line 110):
```ts
// Phase 150 SET-01 — shell web-share warning enabled master switch.
const [shellWarnEnabled, setShellWarnEnabled] = useState(true) // default ON (D-08)
const [shellWarnLoaded, setShellWarnLoaded] = useState(false)
const [shellWarnSaving, setShellWarnSaving] = useState(false)
const [shellWarnError, setShellWarnError] = useState<string | null>(null)
const [showDisableWarnConfirm, setShowDisableWarnConfirm] = useState(false)
```

**Addition 2 — load effect** (analog: auto-close load at lines 181–186):
```ts
// Analog (lines 181–186):
useEffect(() => {
  GetAutoCloseSession().then(val => {
    setAutoCloseSession(val)
    setAutoCloseLoaded(true)
  }).catch(() => setAutoCloseLoaded(true))
}, [])
```

**New effect** (add after line 186):
```ts
useEffect(() => {
  GetShellWebShareWarningEnabled().then(val => {
    setShellWarnEnabled(val)
    setShellWarnLoaded(true)
  }).catch(() => setShellWarnLoaded(true))
}, [])
```

**Addition 3 — handler functions** (analog: `handleToggleAutoClose` at lines 326–338):
```ts
// Analog (lines 326–338):
async function handleToggleAutoClose() {
  const next = !autoCloseSession
  setAutoCloseSaving(true)
  setAutoCloseError(null)
  try {
    await SetAutoCloseSession(next)
    setAutoCloseSession(next)
  } catch (err) {
    setAutoCloseError('Could not save preference — ' + (err instanceof Error ? err.message : String(err)))
  } finally {
    setAutoCloseSaving(false)
  }
}
```

**New handlers** (add after `handleToggleAutoClose`):
```ts
// D-07: turning OFF requires confirmation; turning ON is instant.
async function handleToggleShellWarnEnabled() {
  const next = !shellWarnEnabled
  if (!next) {
    // Turning OFF — show confirm dialog before persisting.
    setShowDisableWarnConfirm(true)
    return
  }
  // Turning ON — instant, no confirmation.
  setShellWarnSaving(true)
  setShellWarnError(null)
  try {
    await SetShellWebShareWarningEnabled(true)
    setShellWarnEnabled(true)
    // D-03 re-arm: notify App.tsx to re-sync shellWebShareWarned state.
    onShellWarnEnabledChange?.(true)
  } catch (err) {
    setShellWarnError('Could not save preference — ' + (err instanceof Error ? err.message : String(err)))
  } finally {
    setShellWarnSaving(false)
  }
}

async function handleConfirmDisableShellWarn() {
  setShowDisableWarnConfirm(false)
  setShellWarnSaving(true)
  setShellWarnError(null)
  try {
    await SetShellWebShareWarningEnabled(false)
    setShellWarnEnabled(false)
    onShellWarnEnabledChange?.(false)
  } catch (err) {
    setShellWarnError('Could not save preference — ' + (err instanceof Error ? err.message : String(err)))
  } finally {
    setShellWarnSaving(false)
  }
}
```

**Addition 4 — JSX toggle row + confirm dialog** (analog: auto-close toggle JSX at lines 414–438):
```tsx
{/* Analog (lines 414–438): */}
{autoCloseLoaded && (
  <label
    className={`settings-panel__toggle-row${autoCloseSession ? ' settings-panel__toggle-row--checked' : ''}`}
    htmlFor="autoCloseSession"
    style={autoCloseSaving ? { pointerEvents: 'none', opacity: 0.6 } : undefined}
  >
    <span className="settings-panel__toggle-track">
      <span className="settings-panel__toggle-thumb" />
    </span>
    <span className="settings-panel__toggle-label">Auto-close tab on exit</span>
  </label>
)}
<input
  type="checkbox"
  id="autoCloseSession"
  className="settings-panel__toggle-input"
  checked={autoCloseSession}
  onChange={() => void handleToggleAutoClose()}
/>
<p className="settings-panel__description">...</p>
{autoCloseError && <p className="settings-panel__error">{autoCloseError}</p>}
```

**New JSX block** (insert new `settings-panel__field-group` div after line 438, before line 440 "Appearance section"):
```tsx
{/* Phase 150 SET-01 — shell web-share warning toggle (Session Behavior section) */}
<div className="settings-panel__field-group">
  {shellWarnLoaded && (
    <label
      className={`settings-panel__toggle-row${shellWarnEnabled ? ' settings-panel__toggle-row--checked' : ''}`}
      htmlFor="shellWebShareWarningEnabled"
      style={shellWarnSaving ? { pointerEvents: 'none', opacity: 0.6 } : undefined}
    >
      <span className="settings-panel__toggle-track">
        <span className="settings-panel__toggle-thumb" />
      </span>
      <span className="settings-panel__toggle-label">Warn before web-sharing a shell session.</span>
    </label>
  )}
  <input
    type="checkbox"
    id="shellWebShareWarningEnabled"
    className="settings-panel__toggle-input"
    checked={shellWarnEnabled}
    onChange={() => void handleToggleShellWarnEnabled()}
  />
  <p className="settings-panel__description">
    Show a one-time security reminder before web-sharing a shell session. Disabling suppresses the reminder.
  </p>
  {shellWarnError && <p className="settings-panel__error">{shellWarnError}</p>}
</div>
```

**Confirm-on-disable dialog** (analog: `RegenerateKeyModal` `.quit-modal*` CSS pattern — lines 60–98 of `RegenerateKeyModal.tsx`). Render inline in SettingsTab (same file), not as a separate component file (simpler, consistent with `showRegenModal` which renders `<RegenerateKeyModal>` inline):
```tsx
<RegenerateKeyModal
  isOpen={showDisableWarnConfirm}
  onConfirm={handleConfirmDisableShellWarn}
  onCancel={() => setShowDisableWarnConfirm(false)}
/>
```

If a distinct `DisableShellWarnModal` component is preferred, mirror `RegenerateKeyModal.tsx` exactly: `isOpen` prop, `onConfirm` async callback, `onCancel`, Escape handler, focus-cancel-on-open, `role="dialog" aria-modal="true"`, `.quit-modal*` CSS classes.

**Props change:** SettingsTab needs a new optional prop `onShellWarnEnabledChange?: (enabled: boolean) => void` to notify App.tsx after the toggle changes (used in re-arm sync, pitfall 2).

---

### `frontend/src/components/Hub/SessionShareModal.tsx` (component, request-response)

**Two changes required:**

**Change 1 — add `cli` to `ShareSession` interface** (current lines 14–20, missing `cli`):
```ts
// Current (lines 14–20):
interface ShareSession {
  id: string
  name: string
  webEnabled: boolean
  homeDir: boolean
  browseEnabled: boolean
}
```

**New interface** (add `cli: string`):
```ts
interface ShareSession {
  id: string
  name: string
  cli: string          // Phase 150 SET-01: needed to check SHELL_CLIS gate
  webEnabled: boolean
  homeDir: boolean
  browseEnabled: boolean
}
```

**Change 2 — add shell-warning props and interception in `handleShareToggle`**.

New props added to `SessionShareModalProps`:
```ts
export interface SessionShareModalProps {
  session: ShareSession
  webServerMode?: 'tailscale' | 'local' | null
  webServerRunning?: boolean
  onClose: () => void
  // Phase 150 SET-01 — shell warning cross-surface parity (D-09/D-10)
  shellWebShareWarned?: boolean
  shellWebShareWarningEnabled?: boolean
  onShellWebShareConfirm?: () => Promise<void>
  onShellWebShareCancel?: () => void
}
```

New state (add alongside existing `shareEnabled`):
```ts
const [pendingShellShare, setPendingShellShare] = useState(false)
```

**Warning gate in `handleShareToggle`** (analog: `handleToggleWeb` gate at App.tsx line 864):
```ts
// Current handleShareToggle (lines 186–199) — NO shell warning:
async function handleShareToggle(): Promise<void> {
  const next = !shareEnabled
  try {
    await ToggleWebServing(session.id, next)
    setShareEnabled(next)
    if (!next) {
      setCachedShare(null)
    }
  } catch {
    // ToggleWebServing failed — revert.
  }
}
```

**New `handleShareToggle`** (insert interception before `ToggleWebServing`):
```ts
async function handleShareToggle(): Promise<void> {
  const next = !shareEnabled
  // Phase 150 SET-01 (D-09): intercept ON-toggles for shell sessions
  // when the warning is enabled and the user hasn't yet acknowledged it.
  if (next && SHELL_CLIS.has(session.cli) && shellWebShareWarningEnabled && !shellWebShareWarned) {
    setPendingShellShare(true)
    return
  }
  try {
    await ToggleWebServing(session.id, next)
    setShareEnabled(next)
    if (!next) {
      setCachedShare(null)
    }
  } catch {
    // ToggleWebServing failed — revert (shareEnabled unchanged).
  }
}
```

**SHELL_CLIS import** (add at top of file, after React import):
```ts
// Phase 150 SET-01 — must match App.tsx:89 and engine.go:isShellSession()
const SHELL_CLIS = new Set(['shell', 'bash', 'zsh', 'pwsh', 'powershell'])
```

**Banner render** (add inside the modal body, before the share toggle, when `pendingShellShare` is true):
```tsx
{pendingShellShare && (
  <ShellWebShareBanner
    sessionName={session.name}
    onConfirm={async () => {
      setPendingShellShare(false)
      await onShellWebShareConfirm?.()
      setShareEnabled(true)
      // seeding effect will issue caps on next render.
    }}
    onCancel={() => {
      setPendingShellShare(false)
      onShellWebShareCancel?.()
    }}
  />
)}
```

Import `ShellWebShareBanner` at the top:
```ts
import { ShellWebShareBanner } from '../ShellWebShareBanner'
```

**HubPanel thread-through:** HubPanel passes `session.cli` (already in `SessionInfo`) to `SessionShareModal`. The existing call site in HubPanel at line 262 passes `shareModalSession` as `SessionInfo | null`; `SessionInfo.cli` is already a field (App.d.ts line 8). HubPanel receives `shellWebShareWarned`, `shellWebShareWarningEnabled`, `onShellWebShareConfirm`, `onShellWebShareCancel` as props from App.tsx and passes them down to `SessionShareModal`.

---

### `frontend/src/components/SettingsSearch.tsx` (utility, transform)

**Analog:** Line 29, existing entry:
```ts
{ label: 'Auto-close tab on exit', target: 'settings-session-behavior' },
```

**New entry to add** (after line 29):
```ts
{ label: 'Warn before web-sharing a shell session.', target: 'settings-session-behavior' },
```

The label string MUST match the toggle label in `SettingsTab.tsx` exactly (including trailing period) so the search result highlights the correct field.

---

## Shared Patterns

### `*bool` Default-ON Persistence Pattern
**Source:** `internal/daemon/engine.go` lines 47, 112, 1084–1101 (`autoCloseSession` field + `GetAutoCloseSession`)
**Apply to:** `engine.go` new `shellWebShareWarningEnabled` field
```go
// daemonSettings: *bool with omitempty
AutoCloseSession    *bool             `json:"autoCloseSession,omitempty"`

// loadSettingsFromDisk: assign pointer directly; no nil-check needed because
// the Get accessor handles the nil case.
e.autoCloseSession = s.AutoCloseSession

// Get accessor: nil = default true
func (e *SessionEngine) GetAutoCloseSession() bool {
    e.mu.RLock()
    defer e.mu.RUnlock()
    if e.autoCloseSession == nil {
        return true
    }
    return *e.autoCloseSession
}

// Set accessor: take address of val
func (e *SessionEngine) SetAutoCloseSession(val bool) {
    e.mu.Lock()
    e.autoCloseSession = &val
    e.saveSettingsToDisk()
    e.mu.Unlock()
}
```

### Confirm-on-Disable Dialog Pattern
**Source:** `frontend/src/components/RegenerateKeyModal.tsx` (full file, lines 1–102)
**Apply to:** `SettingsTab.tsx` confirm-on-disable dialog for the new warning toggle
Key elements:
- `isOpen` prop, `onConfirm: () => Promise<void>`, `onCancel: () => void`
- Escape handler via `window.addEventListener('keydown', ...)`
- Focus `cancelBtnRef.current?.focus()` on open (safe-action default)
- `role="dialog" aria-modal="true" aria-labelledby="..."`
- `.quit-modal-overlay` / `.quit-modal` / `.quit-modal__header` / `.quit-modal__footer` CSS classes
- `.quit-modal__btn--cancel` (safe action) and `.quit-modal__btn--quit-all` (destructive action) buttons

### Race-Mitigation (Synchronous Local State Before Await)
**Source:** `frontend/src/App.tsx` lines 883–906 (`handleShellWebShareConfirm`)
**Apply to:** Share-modal confirm path (wherever `onShellWebShareConfirm` is implemented)
```ts
// Set local "warned" flag SYNCHRONOUSLY before any await (prevents double-banner).
setShellWebShareWarned(true)
try {
  await Promise.all([
    SetShellWebShareWarned(true),
    ToggleWebServing(sessionId, true),
  ])
  setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))
  setPendingShellWebToggle(null)
} catch (err) {
  // Roll back local flag so user can retry.
  setShellWebShareWarned(false)
  setPendingShellWebToggle(null)
}
```

### Settings State Machine (loaded/saving/error trio)
**Source:** `frontend/src/components/SettingsTab.tsx` lines 107–110, 181–186, 326–338
**Apply to:** New `shellWarnEnabled` toggle in `SettingsTab.tsx`
Pattern: `[value, setValue]` + `[loaded, setLoaded]` + `[saving, setSaving]` + `[error, setError]`, with `loaded &&` guard on the JSX toggle, `pointerEvents: none` during saving, error `<p>` below description.

### Auto-Close Toggle JSX Structure
**Source:** `frontend/src/components/SettingsTab.tsx` lines 414–438
**Apply to:** New shell-warning toggle JSX in `SettingsTab.tsx`
The auto-close toggle uses `<input type="checkbox">` styled visually as a toggle via CSS (NOT `role="switch"` on a `<button>`). The new toggle follows the same `type="checkbox"` + `className="settings-panel__toggle-input"` pattern for consistency within the Session Behavior section. (The Appearance section uses a `<button role="switch">` — a different pattern for the theme toggle only.)

### Daemon-Safe Default (app.go Wails bindings)
**Source:** `app.go` lines 600–611 (`GetAutoCloseSession`)
**Apply to:** `app.go` new `GetShellWebShareWarningEnabled` — return `true` (not `false`) when daemon is disconnected or errors, matching D-08's default-ON requirement.

---

## No Analog Found

None. All files to be modified or created have exact analogs in the codebase.

---

## Metadata

**Analog search scope:** `internal/daemon/`, `app.go`, `frontend/src/`, `frontend/src/components/`, `frontend/src/components/Hub/`, `frontend/src/wailsjs/go/main/`
**Files scanned:** 12
**Pattern extraction date:** 2026-06-23

**Critical pitfalls captured in patterns above:**
1. `*bool` not `bool` for `ShellWebShareWarningEnabled` in `daemonSettings` (omitempty trap)
2. Re-arm requires App.tsx `shellWebShareWarned` state re-sync after `SetShellWebShareWarningEnabled(true)` — via `onShellWarnEnabledChange` callback
3. `cli` field must be added to `ShareSession` interface before TypeScript will compile the `SHELL_CLIS.has(session.cli)` check
4. Both surfaces share one `shellWebShareWarned` state in App.tsx — Share-modal must use props/callbacks, not independent local state
5. Wails bindings (`App.js` + `App.d.ts`) require manual edit in the same commit as `app.go` — no `wails generate` step exists
6. Confirm dialog needs `role="dialog" aria-modal="true"` and focus on Cancel button on open
