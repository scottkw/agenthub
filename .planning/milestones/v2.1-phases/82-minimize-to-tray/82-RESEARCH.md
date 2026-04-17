# Phase 82: Minimize to Tray - Research

**Researched:** 2026-04-16
**Domain:** Wails v2 window lifecycle / Go daemon settings / React toggle UI
**Confidence:** HIGH

## Summary

Phase 82 wires together three already-existing systems: the Wails window lifecycle (`StartHidden: true` + `domReady` gate), the daemon settings persistence infrastructure (`daemonSettings` JSON + `settings.json`), and the SettingsTab React UI. The scope is deliberately narrow — no new subsystems, no new files beyond what the pattern already establishes. The riskiest decision is exactly when `domReady` reads the setting and how it handles a daemon-unreachable edge case.

The codebase has been read directly. All claims below are `[VERIFIED: codebase]` unless noted otherwise.

**Primary recommendation:** Follow the `GetCLIPaths`/`SaveCLIPaths` pattern precisely. The only novel work is (1) a `StartMinimized bool` field in `daemonSettings`, (2) two daemon API endpoints + two Wails bindings, and (3) the React toggle with the exact CSS from the UI-SPEC.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Add a new "Behavior" section to the Settings tab for app-level behavior toggles. The "Start minimized to system tray" toggle lives here.
- **D-02:** The toggle is a simple on/off switch with a clear label.
- **D-03:** Claude's discretion on where the Behavior section sits relative to existing sections. (Resolved in UI-SPEC: Behavior is first — before Appearance.)
- **D-04:** When "Start minimized" is enabled, the window never shows — `domReady()` skips `WindowShow()` and `setDockVisible(true)`. Only the tray icon appears.
- **D-05:** When disabled (default), current behavior preserved: `domReady()` shows window with splash as usual.
- **D-06:** All 3 platforms (macOS, Linux, Windows) from the start. Each already has tray support — the change is in `domReady` and settings, not tray code.
- **D-07:** Preference stored in `daemonSettings` struct (add `StartMinimized bool` field) and persisted to `settings.json` via `loadSettingsFromDisk`/`saveSettingsToDisk`.
- **D-08:** New Wails-bound methods: `GetStartMinimized() bool` and `SetStartMinimized(bool)` following `GetCLIPaths`/`SaveCLIPaths` pattern.

### Claude's Discretion

- Behavior section ordering within Settings tab (UI-SPEC resolves: Behavior goes first)
- Toggle component style (UI-SPEC resolves: CSS-styled native checkbox per `.settings-panel__*` conventions)
- Whether `domReady` reads the setting from the daemon client or via a startup event (open — see Architecture Patterns)
- How to handle the edge case where daemon is unreachable at startup (probably just show window as fallback)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TRAY-01 | Settings includes a toggle for "Start minimized to system tray" | UI-SPEC defines exact component, CSS, placement. SettingsTab.tsx is the target file. |
| TRAY-02 | When enabled, launching AgentHub opens with window hidden and only tray icon visible | `domReady` gate on `WindowShow`/`setDockVisible`. `StartHidden: true` is already set in main.go. |
| TRAY-03 | Minimize-to-tray preference persists across app restarts | `daemonSettings` + `settings.json` + daemon API endpoints. Identical pattern to CLIPaths. |
</phase_requirements>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Persist start-minimized preference | Daemon (Go) | — | Daemon owns all persistent settings via `daemonSettings` + `settings.json`. This is already the pattern for CLIPaths. |
| Expose preference to Wails | API / Backend | — | New daemon API endpoint + Wails binding. Same tier ownership as CLIPaths. |
| Read preference at launch | App startup (Go) | — | `domReady` is the correct hook — window is hidden by default (`StartHidden: true`) and `domReady` decides whether to reveal it. |
| Toggle UI | Frontend | — | SettingsTab.tsx + new CSS rules in style.css. Non-optimistic toggle per UI-SPEC. |
| Platform-specific hide/show | OS/Tray tier | App startup | `setDockVisible` (macOS) is already called from `domReady`; same logic gates it. Linux/Windows have no dock equivalent. |

---

## Standard Stack

### Core (all already in project)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Wails v2 | already in go.mod | `runtime.WindowShow`, `runtime.WindowHide`, app lifecycle hooks | Project framework — not changeable |
| encoding/json | stdlib | `daemonSettings` marshalling | Already used in `saveSettingsToDisk` |
| React 18 | already in frontend | SettingsTab toggle state | Project frontend framework |

No new dependencies needed. [VERIFIED: codebase]

### No Installation Step

All required packages are already in the project. No `npm install` or `go get` commands are needed for this phase.

---

## Architecture Patterns

### System Architecture Diagram

```
User clicks toggle (SettingsTab.tsx)
        |
        v
SetStartMinimized(bool) [Wails binding, app.go]
        |
        v
DaemonClient.SetStartMinimized(bool) [internal/daemon/client.go]
        |
        v
PATCH /settings/start-minimized [internal/daemon/api.go]
        |
        v
SessionEngine.SetStartMinimized(bool) [internal/daemon/engine.go]
        |--- updates daemonSettings.StartMinimized
        `--- calls saveSettingsToDisk() → settings.json


App launch (main.go → wails.Run)
        |
        v
StartHidden: true  [window hidden by Wails]
        |
        v
app.startup()  [connects daemon, starts pollers]
        |
        v
app.domReady()
        |
        +--[GetStartMinimized() → daemon API GET /settings/start-minimized]
        |       |
        |       +--[true]  → skip WindowShow, skip setDockVisible → tray only
        |       |
        |       `--[false OR daemon unreachable] → WindowShow + setDockVisible(true)
        |
        v
User clicks tray icon
        |
        v
onTrayShow() / Linux onShow() / Windows wndProc()  [already exists]
        |
        v
runtime.WindowShow + setDockVisible(true)  [already exists]
```

### Recommended Project Structure

No new directories needed. Changes are additive to existing files:

```
internal/daemon/
  engine.go          # add StartMinimized bool to daemonSettings + get/set methods
  api.go             # add GET/PATCH /settings/start-minimized routes
  client.go          # add GetStartMinimized / SetStartMinimized client methods
  types.go           # add StartMinimizedResponse type (or reuse plain bool wrapper)

app.go               # add GetStartMinimized / SetStartMinimized Wails bindings
                     # modify domReady() to gate WindowShow on setting

frontend/src/
  components/SettingsTab.tsx    # add Behavior section + toggle
  style.css                     # add .settings-panel__toggle-* CSS rules
```

### Pattern 1: daemonSettings Extension (CLIPaths precedent)

**What:** Add a `bool` field to the `daemonSettings` struct. Load/save paths already handle unknown fields via JSON omitempty — adding a new field is non-breaking.

**When to use:** Any new setting that needs to survive app restarts.

```go
// Source: internal/daemon/engine.go (VERIFIED: codebase)
// BEFORE:
type daemonSettings struct {
    CLIPaths map[string]string `json:"cliPaths,omitempty"`
}

// AFTER:
type daemonSettings struct {
    CLIPaths      map[string]string `json:"cliPaths,omitempty"`
    StartMinimized bool             `json:"startMinimized,omitempty"`
}
```

The `omitempty` on a `bool` means `false` will NOT be written to the JSON (omitted = default false). This is correct behavior — absence means "don't start minimized."

**Critical:** `omitempty` on bool in Go means `false` is treated as zero-value and omitted. The round-trip is: read file → bool is `false` if key absent → do NOT start minimized. This is the correct default. [VERIFIED: Go stdlib behavior, well-known]

### Pattern 2: Engine Methods (CLIPaths precedent)

```go
// Source: internal/daemon/engine.go GetCLIPaths / UpdateCLIPath pattern (VERIFIED: codebase)

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

The `startMinimized bool` field also needs to be added to `SessionEngine` struct (alongside `cliPaths`). And `loadSettingsFromDisk` must populate it: `e.startMinimized = s.StartMinimized`.

### Pattern 3: API Routes (CLIPaths precedent)

```go
// Source: internal/daemon/api.go registerRoutes (VERIFIED: codebase)
// New routes to add to registerRoutes():
a.mux.HandleFunc("GET /settings/start-minimized", a.handleGetStartMinimized)
a.mux.HandleFunc("PATCH /settings/start-minimized", a.handleSetStartMinimized)

// Handler implementations:
func (a *API) handleGetStartMinimized(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]bool{"startMinimized": a.engine.GetStartMinimized()})
}

func (a *API) handleSetStartMinimized(w http.ResponseWriter, r *http.Request) {
    var req struct{ StartMinimized bool `json:"startMinimized"` }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    a.engine.SetStartMinimized(req.StartMinimized)
    w.WriteHeader(http.StatusNoContent)
}
```

### Pattern 4: DaemonClient Methods (CLIPaths precedent)

```go
// Source: internal/daemon/client.go GetCLIPaths / UpdateCLIPath (VERIFIED: codebase)

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

### Pattern 5: Wails Bindings (GetCLIPaths / SaveCLIPaths precedent)

```go
// Source: app.go GetCLIPaths / UpdateCLIPath (VERIFIED: codebase)

// GetStartMinimized returns the persisted start-minimized preference.
// Called by the frontend on Settings mount.
func (a *App) GetStartMinimized() bool {
    if a.client == nil {
        return false // daemon not connected — safe default
    }
    val, err := a.client.GetStartMinimized()
    if err != nil {
        return false // err → safe default (show window)
    }
    return val
}

// SetStartMinimized persists the start-minimized preference.
// Called by the frontend toggle onChange.
func (a *App) SetStartMinimized(val bool) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.SetStartMinimized(val)
}
```

### Pattern 6: domReady Gate

```go
// Source: app.go domReady (VERIFIED: codebase)
// BEFORE:
func (a *App) domReady(ctx context.Context) {
    runtime.WindowShow(ctx)
    a.setDockVisible(true)
}

// AFTER:
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

**Edge case handling (Claude's discretion per CONTEXT.md):** When daemon is unreachable at startup (`a.client == nil` or `GetStartMinimized` errors), show the window. This is the safest fallback — the user is not stranded with an invisible window and no tray click registered. [VERIFIED: matches CONTEXT.md guidance]

### Pattern 7: React Toggle (UI-SPEC)

The UI-SPEC defines the exact HTML structure and CSS. Key points for the planner:

- Non-optimistic: call `SetStartMinimized(newValue)` before updating local state
- Show toggle only after `GetStartMinimized()` resolves (use a `loaded` boolean in state to prevent flash)
- Error state: revert to previous value, show `.settings-panel__error` below description
- Loading state: `pointer-events: none` + `opacity: 0.6` on the toggle row
- The checked state class (`--checked`) is applied to the `label` element to drive CSS sibling selectors

```tsx
// Source: UI-SPEC (VERIFIED: 82-UI-SPEC.md)
// State vars needed:
const [startMinimized, setStartMinimized] = useState(false)
const [toggleLoaded, setToggleLoaded] = useState(false)
const [toggleSaving, setToggleSaving] = useState(false)
const [toggleError, setToggleError] = useState<string | null>(null)

// On mount (alongside existing useEffect for web state):
useEffect(() => {
  GetStartMinimized().then(val => {
    setStartMinimized(val)
    setToggleLoaded(true)
  }).catch(() => setToggleLoaded(true))
}, [])

// Toggle handler:
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

### Anti-Patterns to Avoid

- **Optimistic toggle update:** UI-SPEC explicitly requires non-optimistic flow. Call backend first, update state only on success.
- **Flash of unchecked:** Do not render the toggle until `GetStartMinimized()` resolves (use `toggleLoaded` gate). [VERIFIED: UI-SPEC]
- **Reading setting before daemon is connected:** `domReady` runs after `startup`, so `a.client` should be set. But the daemon connect can fail. Always guard with `a.client != nil` and error check.
- **`omitempty` false positive suppression:** Because `StartMinimized bool` with `omitempty` omits `false`, the JSON key will be absent for new installs. This is correct — `false` is the desired default. Do NOT use a pointer (`*bool`) unless the semantics require distinguishing "explicitly false" from "not set." For this feature, absent = not-set = don't minimize, which is correct.
- **Adding StartMinimized to SessionEngine.saveSettingsToDisk without also reading in loadSettingsFromDisk:** Both must be updated together. [ASSUMED: standard Go pattern, but easy to miss]

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Settings persistence | Custom file format / separate DB | Existing `daemonSettings` + `settings.json` | Already proven, JSON-serializable, tested by CLIPaths |
| Toggle animation | JS-based animation | CSS `transition` per UI-SPEC | No JS timer needed; CSS handles it |
| Window show/hide | Direct OS calls | `runtime.WindowShow` / `runtime.WindowHide` (Wails) | Cross-platform abstraction already in use |
| Dock visibility | Platform-specific NSApp calls | `a.setDockVisible()` (already implemented) | macOS-only; already build-tag guarded in tray.go |

---

## Common Pitfalls

### Pitfall 1: `setDockVisible` Build Tag Scope

**What goes wrong:** `setDockVisible` is defined only in `tray.go` with `//go:build darwin`. On Linux and Windows, calling it from `domReady` (which is in `app.go`, no build tag) causes a compile error.

**Why it happens:** `app.go` is compiled for all platforms. `tray.go` is Darwin-only.

**How to avoid:** Check the existing `domReady` — it calls `a.setDockVisible(true)` unconditionally. This already works because `setDockVisible` has a no-op stub on non-Darwin platforms (look in `tray_linux.go`, `tray_windows.go`, or `tray_common.go`).

**Verification:** [VERIFIED: codebase] — `tray.go` (darwin) defines `setDockVisible`. Stubs must exist elsewhere. Confirm before adding the gate to `domReady`.

**Warning sign:** Compile error `undefined: setDockVisible` on Linux/Windows build.

### Pitfall 2: Race Between `startup` and `domReady`

**What goes wrong:** `domReady` calls `a.client.GetStartMinimized()` but `a.client` is set in `startup`. If Wails calls `domReady` before `startup` completes, `a.client` is nil.

**Why it happens:** Wails guarantees `OnStartup` runs before `OnDomReady`, but `startup` in this project starts goroutines (tray poller, health poller). The daemon connection itself (`daemon.EnsureDaemon`) is synchronous in `startup`.

**How to avoid:** Guard with `if a.client != nil` (already the pattern for all other `a.client` uses). Fallback to show window. [VERIFIED: app.go - existing `if a.client == nil { return ... }` guards throughout]

**Warning sign:** Window never shows on daemon failure. Test: kill daemon before launch.

### Pitfall 3: `loadSettingsFromDisk` Partial Update

**What goes wrong:** `loadSettingsFromDisk` currently only populates `e.cliPaths`. Adding `StartMinimized` to `daemonSettings` struct without also reading it back into `e.startMinimized` means the field is persisted but never loaded on restart.

**How to avoid:** After `json.Unmarshal`, also set `e.startMinimized = s.StartMinimized`. [VERIFIED: engine.go L80-86 — explicit field copy, not struct assignment]

### Pitfall 4: Toggle Flash on Mount

**What goes wrong:** Toggle renders as `false` (unchecked) for the async moment before `GetStartMinimized()` resolves. If the user has it enabled, they see the toggle snap from off to on.

**How to avoid:** UI-SPEC mandates a `toggleLoaded` state gate. Render the toggle only after the value resolves. [VERIFIED: UI-SPEC]

### Pitfall 5: Wails Bindings Not Regenerated

**What goes wrong:** New Wails-bound methods (`GetStartMinimized`, `SetStartMinimized`) are added to `app.go`, but the generated TypeScript bindings in `frontend/src/wailsjs/go/main/App.ts` are not regenerated. The frontend imports fail at runtime.

**How to avoid:** Run `wails generate module` or `wails dev` (which auto-regenerates) after adding new bound methods. The executor must include this step. [ASSUMED: standard Wails v2 workflow — verified against general Wails knowledge]

---

## Code Examples

### Full `domReady` Change

```go
// Source: app.go domReady — VERIFIED: codebase (current implementation at L77-80)
// Minimal conditional gate preserving all existing behavior:
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

### `loadSettingsFromDisk` Update

```go
// Source: internal/daemon/engine.go L74-87 (VERIFIED: codebase)
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
    data, err := os.ReadFile(settingsPath(dir))
    if err != nil {
        return
    }
    var s daemonSettings
    if json.Unmarshal(data, &s) == nil {
        e.mu.Lock()
        if s.CLIPaths != nil {
            for k, v := range s.CLIPaths {
                e.cliPaths[k] = v
            }
        }
        e.startMinimized = s.StartMinimized  // NEW: load persisted value
        e.mu.Unlock()
    }
}
```

### `saveSettingsToDisk` Update

```go
// Source: internal/daemon/engine.go L91-98 (VERIFIED: codebase)
func (e *SessionEngine) saveSettingsToDisk() {
    // Caller holds e.mu.Lock()
    s := daemonSettings{
        CLIPaths:       e.cliPaths,
        StartMinimized: e.startMinimized,  // NEW: persist value
    }
    data, err := json.Marshal(s)
    if err != nil {
        return
    }
    _ = os.WriteFile(settingsPath(e.configDir), data, 0600)
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Show window unconditionally in domReady | Gate on persisted setting | Phase 82 | TRAY-02 satisfied |
| `daemonSettings` with only `CLIPaths` | Add `StartMinimized` field | Phase 82 | TRAY-03 satisfied |
| No Behavior section in Settings | Add Behavior section first | Phase 82 | TRAY-01 satisfied |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `setDockVisible` has a no-op stub on Linux/Windows (not darwin) | Pitfall 1 | Compile error on non-darwin build; executor must verify `tray_linux.go` / `tray_windows.go` define stub before modifying `domReady` |
| A2 | Wails v2 regenerates TypeScript bindings via `wails generate module` or auto on `wails dev` | Pitfall 5 | Frontend imports of new Wails methods fail at runtime; executor must include regen step |

---

## Open Questions

1. **`setDockVisible` stub location**
   - What we know: `tray.go` (darwin) defines it. `app.go` calls it unconditionally today. The build succeeds cross-platform.
   - What's unclear: Whether the stub lives in `tray_common.go`, `tray_linux.go`, or `tray_windows.go`.
   - Recommendation: Executor reads `tray_common.go` and `tray_linux.go` before touching `domReady`. The existing call in `domReady` already proves the stub exists — this is a zero-risk concern.

2. **`domReady` reads setting synchronously — acceptable latency?**
   - What we know: `GetStartMinimized` is a Unix socket call to the local daemon. Expected latency: <5ms.
   - What's unclear: Whether Wails has a timeout on `OnDomReady`.
   - Recommendation: Acceptable. The daemon is already confirmed reachable before `domReady` runs (EnsureDaemon succeeds in `startup`). A guard-and-fallback approach means even a slow call degrades gracefully.

---

## Environment Availability

Step 2.6: SKIPPED — no new external dependencies. All required tools (Go, Wails, Node/pnpm) are already in use by the project.

---

## Validation Architecture

`workflow.nyquist_validation` is absent from `.planning/config.json` — treated as enabled.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package (existing) + vitest (frontend, if present) |
| Config file | none — `go test ./...` from project root |
| Quick run command | `go test ./internal/daemon/... -run TestSettings -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TRAY-01 | Behavior section + toggle renders in SettingsTab | manual/visual | `wails dev` → open Settings | N/A (UI) |
| TRAY-02 | Window hidden on launch when enabled | integration/manual | Launch app after enabling; verify no window | N/A (manual) |
| TRAY-03 | Setting persists across restarts | unit (engine) | `go test ./internal/daemon/... -run TestStartMinimized` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/daemon/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/daemon/engine_test.go` or an equivalent settings test file — covers TRAY-03 (`TestStartMinimized` — persist bool, reload from disk, default false when absent)

---

## Security Domain

This phase stores a simple `bool` preference (no credentials, no paths, no user data). The setting is stored in `~/.config/agenthub/settings.json` at `0600` permissions (existing pattern). No authentication, no crypto, no external network calls involved.

**Applicable ASVS Categories:** None apply at a level requiring mitigation work. The existing file permission model (`0600`) satisfies storage confidentiality for a local non-sensitive preference.

---

## Sources

### Primary (HIGH confidence)

- `app.go` — `domReady`, `startup`, `beforeClose`, `GetCLIPaths`, `SetStartMinimized` pattern location [VERIFIED: codebase, full file read]
- `internal/daemon/engine.go` — `daemonSettings`, `loadSettingsFromDisk`, `saveSettingsToDisk`, `SessionEngine` struct [VERIFIED: codebase, full file read]
- `internal/daemon/api.go` — route registration pattern, `handleGetCLIPaths`, `handleUpdateCLIPath` [VERIFIED: codebase, full file read]
- `internal/daemon/client.go` — `GetCLIPaths`, `UpdateCLIPath`, `doJSON` pattern [VERIFIED: codebase, full file read]
- `internal/daemon/types.go` — existing type definitions [VERIFIED: codebase, full file read]
- `tray.go` — `onTrayShow`, `setDockVisible`, `initTray` [VERIFIED: codebase, full file read]
- `main.go` — `StartHidden: true`, Wails options, `runGUI` [VERIFIED: codebase, full file read]
- `frontend/src/components/SettingsTab.tsx` — existing sections, patterns, imports [VERIFIED: codebase, full file read]
- `.planning/phases/82-minimize-to-tray/82-UI-SPEC.md` — CSS spec, component HTML, interaction contract, section order [VERIFIED: file read]
- `.planning/phases/82-minimize-to-tray/82-CONTEXT.md` — locked decisions, canonical refs [VERIFIED: file read]

### Secondary (MEDIUM confidence)

None needed — all claims verified directly from codebase.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; all verified in codebase
- Architecture: HIGH — direct code reads confirm extension points exactly as CONTEXT.md describes
- Pitfalls: HIGH for code pitfalls (verified); MEDIUM for Wails regen step (standard Wails workflow, assumed)

**Research date:** 2026-04-16
**Valid until:** 2026-05-16 (stable Go/Wails codebase; no third-party dependencies changing)
