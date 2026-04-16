# Phase 79: Settings Persistence & Path Browsing - Research

**Researched:** 2026-04-16
**Domain:** Wails v2 Go backend, React/TypeScript frontend, daemon settings persistence
**Confidence:** HIGH

## Summary

This phase addresses five requirements: persisting CLI path overrides (SET-01, SET-02), save confirmation feedback (SET-03), and native browse buttons for path input fields (SET-04, SET-05).

The core problem is already diagnosed from the codebase: `SessionEngine.UpdateCLIPath` stores overrides in-memory only (`e.cliPaths map[string]string`). When the daemon restarts, that map is empty. No file is ever written to `~/.config/agenthub/`. The fix is to add a `settings.json` file that is loaded at daemon startup and written whenever a path is updated.

The browse button work is straightforward: `runtime.OpenFileDialog` already exists in Wails v2.10.2 (the version in use), and the app already exposes `OpenDirectoryDialog` as a Wails-bound method. A new `OpenFileDialog` bound method is needed for selecting executables (files, not folders). The frontend connects browse buttons per path row, each calling that method and populating the input field.

For save confirmation (SET-03), the current `handleSaveCLIPaths` transitions `saving: true/false` but never shows a positive confirmation state. Adding a third state — `saved: true` for ~1.5 seconds after success — displays inline feedback using an existing CSS button modifier class pattern (`--save`, `--cancel`), requiring only a new `--saved` variant.

**Primary recommendation:** Add `settings.json` persistence in `internal/daemon/engine.go` (load at `NewSessionEngine`, write on `UpdateCLIPath`), add `OpenFileDialog` bound method to `app.go`, add browse buttons to `SettingsTab.tsx`, and add a `saved` confirmation state to the save flow.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Path persistence (disk I/O) | Daemon (Go) | — | Daemon owns `cliPaths` map and config dir; GUI is a thin shell |
| Path load on startup | Daemon (Go) | — | Loaded in `runDaemonCore` via `NewSessionEngine` |
| Save confirmation UI | Frontend (React) | — | Pure UI state, no backend involvement |
| Native file picker | Go (Wails bound method) | Frontend trigger | Wails runtime dialog; result flows back to React state |
| Path field population | Frontend (React) | — | Updates `customPaths` state from picker result |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SET-01 | User-modified agent paths persist across app restarts | Add `settings.json` persistence to `SessionEngine`; load on daemon startup |
| SET-02 | User-modified Tailscale path persists across app restarts | Same mechanism as SET-01; tailscale key handled identically in cliPaths |
| SET-03 | Clicking Save shows visible confirmation feedback | Add `saved` boolean state after successful save; show "Saved!" button text for 1.5s |
| SET-04 | Each path entry in Settings > Paths has a browse button | New `OpenFileDialog` bound method in app.go; browse button per table row in SettingsTab |
| SET-05 | Selecting a path via the browser populates the corresponding input field | `OpenFileDialog` return value sets `customPaths[cli.Name]` state |
</phase_requirements>

## Standard Stack

### Core (already in use — no new dependencies)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/wailsapp/wails/v2` | v2.10.2 | Native file picker via `runtime.OpenFileDialog` | Already used for `OpenDirectoryDialog`; same API |
| `encoding/json` | stdlib | `settings.json` marshal/unmarshal | Consistent with rest of daemon |
| React `useState` | (in use) | `saved` confirmation state | Existing pattern in SettingsTab |

No new npm or Go packages are required.

**Version verification:** Wails v2.10.2 confirmed in `go.mod`. `runtime.OpenFileDialog` confirmed available via `go doc`. [VERIFIED: go doc github.com/wailsapp/wails/v2/pkg/runtime]

## Architecture Patterns

### System Architecture Diagram

```
User clicks "Save Paths"
        │
        ▼
SettingsTab.handleSaveCLIPaths()
  ├── UpdateCLIPath(name, path) ──► app.go bound method
  │                                      │
  │                              a.client.UpdateCLIPath(name, path)
  │                                      │ (Unix socket HTTP PATCH /settings/cli-paths/{name})
  │                              api.handleUpdateCLIPath()
  │                                      │
  │                              engine.UpdateCLIPath(name, path)
  │                                      │
  │                              ┌───────┴──────────────────────┐
  │                              │ e.cliPaths[name] = path       │ (in-memory — already exists)
  │                              │ saveSettingsToDisk(dir, e)    │ (NEW: write settings.json)
  │                              └───────────────────────────────┘
  │
  └── setSaved(true) → "Saved!" button text for 1.5s → setSaved(false)

User clicks "Browse" button (per path row)
        │
        ▼
SettingsTab calls OpenFileDialog(currentPath)
        │ (Wails bound method in app.go)
        │
runtime.OpenFileDialog(ctx, OpenDialogOptions{Title: "Select executable", ...})
        │ (native OS picker)
        │
returns selected path (or "" if cancelled)
        │
setCustomPaths(prev => ({...prev, [name]: selectedPath}))

Daemon startup (runDaemonCore)
        │
NewSessionEngine()
  └── loadSettingsFromDisk(dir) ──► reads ~/.config/agenthub/settings.json
                                          └── e.cliPaths = loaded map
```

### Recommended Project Structure

No new files or directories needed. Changes are:

```
app.go                          # +OpenFileDialog bound method
internal/daemon/engine.go       # +loadSettingsFromDisk, +saveSettingsToDisk
frontend/src/components/
  SettingsTab.tsx               # +browse buttons, +saved state
  __tests__/
    SettingsTab.persistence.test.tsx   # new test file (SET-01..05)
frontend/src/wailsjs/go/main/
  App.js                        # +OpenFileDialog export
  App.d.ts                      # +OpenFileDialog type
frontend/src/style.css          # +settings-panel__btn--saved modifier
```

### Pattern 1: Settings JSON Persistence in Daemon Engine

**What:** Load `settings.json` at engine construction; write on every path update.
**When to use:** Any daemon-owned config that must survive restarts.

```go
// Source: existing pattern from app.go configDir() and engine.go daemonConfigDir()

// settingsPath returns the path to settings.json inside the agenthub config dir.
func settingsPath(dir string) string {
    return filepath.Join(dir, "settings.json")
}

// daemonSettings is the persisted settings structure.
type daemonSettings struct {
    CLIPaths map[string]string `json:"cliPaths,omitempty"`
}

// loadSettingsFromDisk reads settings.json and populates engine state.
// Missing file is not an error (first run).
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
    data, err := os.ReadFile(settingsPath(dir))
    if err != nil {
        return // file not found or unreadable — not an error
    }
    var s daemonSettings
    if json.Unmarshal(data, &s) == nil && s.CLIPaths != nil {
        e.mu.Lock()
        for k, v := range s.CLIPaths {
            e.cliPaths[k] = v
        }
        e.mu.Unlock()
    }
}

// saveSettingsToDisk writes current cliPaths to settings.json.
// Called inside UpdateCLIPath after the in-memory map is updated.
// Caller holds e.mu.Lock().
func (e *SessionEngine) saveSettingsToDisk(dir string) {
    s := daemonSettings{CLIPaths: e.cliPaths}
    data, err := json.Marshal(s)
    if err != nil {
        return
    }
    _ = os.WriteFile(settingsPath(dir), data, 0600)
}
```

The engine must hold `daemonConfigDir()` as a field (set at construction) so `saveSettingsToDisk` does not call `os.UserConfigDir()` on every save.

### Pattern 2: OpenFileDialog Bound Method in app.go

**What:** Expose Wails native file picker for executable selection.
**When to use:** When user needs to select a file path (not a folder).

```go
// Source: [VERIFIED: go doc github.com/wailsapp/wails/v2/pkg/runtime]
// Pattern mirrors existing OpenDirectoryDialog in app.go

// OpenFileDialog opens a native OS file picker and returns the selected path.
// Returns "" if the user cancels. Falls back to home directory when defaultDir is empty.
// Used by Settings > Paths browse buttons (SET-04).
func (a *App) OpenFileDialog(defaultDir string) (string, error) {
    if defaultDir == "" {
        if home, err := os.UserHomeDir(); err == nil {
            defaultDir = home
        }
    }
    return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
        Title:            "Select Executable",
        DefaultDirectory: defaultDir,
        ShowHiddenFiles:  true, // agents may be in dotfiles (e.g. ~/.local/bin)
    })
}
```

No `Filters` needed — paths to any executable file (not just a specific extension).

### Pattern 3: Save Confirmation State in SettingsTab

**What:** Three-state save button: idle → saving → saved → idle.
**When to use:** Async form saves where user needs feedback.

```typescript
// Source: existing pattern from setCopied/setUrlCopied in SettingsTab.tsx

const [saving, setSaving] = useState(false)
const [saved, setSaved] = useState(false)  // NEW

async function handleSaveCLIPaths() {
    setSaving(true)
    setError(null)
    try {
        // ... existing UpdateCLIPath calls ...
        setSaved(true)
        setTimeout(() => setSaved(false), 1500)
    } catch (err) {
        setError(err instanceof Error ? err.message : String(err))
    } finally {
        setSaving(false)
    }
}

// Button render:
// saving  → disabled, "Saving…"
// saved   → disabled, "Saved!", --saved class (green)
// idle    → enabled, "Save Paths", --save class (blue)
```

### Pattern 4: Browse Button per Path Row

**What:** Add inline browse button to each path input cell in the Paths table.
**When to use:** File/folder inputs where user may not know the path.

```typescript
// Import OpenFileDialog alongside UpdateCLIPath
import { UpdateCLIPath, OpenFileDialog } from '../wailsjs/go/main/App'

// Handler per CLI row:
async function handleBrowse(cliName: string) {
    const current = customPaths[cliName] ?? ''
    // Pass current value's directory as starting location
    const dir = current ? current.replace(/[^/\\]+$/, '') : ''
    const selected = await OpenFileDialog(dir)
    if (selected) {  // empty string = user cancelled
        setCustomPaths(prev => ({ ...prev, [cliName]: selected }))
    }
}

// In table row JSX (alongside existing input):
<td>
    <div className="settings-panel__path-row">
        <input
            className="settings-panel__path-input"
            type="text"
            value={customPaths[cli.Name] ?? cli.Path}
            onChange={...}
        />
        <button
            className="settings-panel__browse-btn"
            onClick={() => void handleBrowse(cli.Name)}
            title="Browse for executable"
        >
            Browse
        </button>
    </div>
</td>
```

### Anti-Patterns to Avoid

- **Storing paths in localStorage (frontend only):** The daemon reads paths directly, not the frontend. localStorage is invisible to Go.
- **Calling `os.UserConfigDir()` on every save:** Cache it as `e.configDir` field at engine construction.
- **Validating file existence in `loadSettingsFromDisk`:** Stored paths may not exist on every machine; validation belongs in `UpdateCLIPath` (already present) and at session creation, not at load time.
- **Using `OpenDirectoryDialog` for executables:** Executables are files, not directories. Using the directory dialog would return a folder path.
- **Showing a modal for browse result confirmation:** Wails dialog returns the selected path; just set it directly into state with no modal.
- **Re-detecting CLIs after save:** `DetectCLIs()` uses `PATH` lookup, not the stored overrides. Refreshing `detectedCLIs` after save won't show updated custom paths in the input fields — `customPaths` state is the source of truth for the inputs.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Native file picker | HTML `<input type="file">` | `runtime.OpenFileDialog` | HTML file input does not return an OS path — returns a `File` object for reading, not a path string. Wails dialog returns the actual filesystem path. |
| File picker (exe only) | Custom extension filter UI | `OpenDialogOptions` with no filters | No filter needed; all files are valid executables |
| Settings persistence | SQLite, BoltDB, etc. | `encoding/json` write to `settings.json` | One tiny map (`map[string]string`). JSON flat file is the same pattern as `ct_disclosed` flag in app.go and opencode-tui.json in engine.go. |

**Key insight:** The existing codebase already establishes `~/.config/agenthub/` as the config dir. Every other persistent artifact (CT flag, opencode-tui.json) uses a plain file write. Match that pattern.

## Common Pitfalls

### Pitfall 1: Lock Ordering in saveSettingsToDisk

**What goes wrong:** `saveSettingsToDisk` is called from `UpdateCLIPath` which already holds `e.mu.Lock()`. If `saveSettingsToDisk` tries to acquire a lock, deadlock.
**Why it happens:** Forgetting that the caller already holds the mutex.
**How to avoid:** `saveSettingsToDisk` must NOT call `e.mu.Lock()`. It accesses `e.cliPaths` directly (caller holds the lock) and writes to disk. Document the precondition: "caller holds e.mu.Lock".
**Warning signs:** Test hangs instead of panicking.

### Pitfall 2: Race Between Daemon Restart and App GUI Load

**What goes wrong:** GUI calls `DetectCLIs()` on startup, which scans PATH. If the daemon restarted but hasn't loaded `settings.json` yet, the GUI shows unmodified paths.
**Why it happens:** The GUI loads `detectedCLIs` from `DetectCLIs()` (PATH scanning), not from `GetCLIPaths()`. Custom paths in `settings.json` are not surfaced through `DetectCLIs`.
**How to avoid:** On `SettingsTab` mount, call `GetCLIPaths()` to fetch stored overrides and merge them into `customPaths` state. The SettingsTab currently initializes `customPaths` from `clis` prop (which comes from `DetectCLIs`). Add a `useEffect` that calls `GetCLIPaths()` and overwrites any keys present in the returned map. This way stored paths override detected paths in the UI.
**Warning signs:** User saves a custom path, restarts, opens Settings — sees the old detected path.

### Pitfall 3: Empty String from Cancelled Dialog

**What goes wrong:** `OpenFileDialog` returns `""` when user cancels. If the frontend doesn't check for empty, it clears the path field.
**Why it happens:** Wails returns `("", nil)` on cancel.
**How to avoid:** `if selected { setCustomPaths(...) }` — the guard is already shown in Pattern 4. TypeScript empty string is falsy.
**Warning signs:** Path field blanks when user opens browser then closes it.

### Pitfall 4: SettingsTab Receives stale `clis` Prop

**What goes wrong:** `SettingsTab` receives `clis={detectedCLIs}` from App. `detectedCLIs` is set once on mount from `DetectCLIs()`. If user saves a path that `DetectCLIs` didn't find (e.g., tailscale not on PATH), the saved path is stored on disk but `detectedCLIs` never updates.
**Why it happens:** `DetectCLIs()` is PATH-based; custom overrides live in `settings.json`.
**How to avoid:** The tailscale path row is already handled specially (rendered as a separate table if not in `clis`). The `GetCLIPaths()` fetch in SettingsTab useEffect populates `customPaths['tailscale']` from disk — that row's input is driven by `customPaths`, not `clis`.
**Warning signs:** Tailscale path field empty after restart even though it was saved.

### Pitfall 5: Test Coverage Gap — SettingsTab Mocked Wails Runtime

**What goes wrong:** Vitest tests for SettingsTab use `?raw` source inspection. Calling `GetCLIPaths` or `OpenFileDialog` would require mocking Wails bindings.
**Why it happens:** The project uses source-inspection tests (import the .tsx as raw text, assert on source patterns). This avoids jsdom complexity.
**How to avoid:** Follow the existing test pattern — assert that the source contains the right import statements, function names, and state variables. Do not attempt to execute Wails calls in tests. New test file should mirror `SettingsTab.test.tsx` source-inspection style.

## Code Examples

### Loading Settings at Engine Construction

```go
// Source: engine.go (to be added)
func NewSessionEngine() *SessionEngine {
    hostname, _ := os.Hostname()
    cfgDir := daemonConfigDir()
    tuiConfig := ensureOpenCodeTUIConfig(cfgDir)
    e := &SessionEngine{
        hostname:          hostname,
        configDir:         cfgDir,    // NEW field
        opencodeTUIConfig: tuiConfig,
        registry:          pty.NewSessionRegistry(),
        backend:           pty.NewNativePTYBackend(),
        manager:           relay.NewHubManager(),
        tabNames:          make(map[string]string),
        sessionCLIs:       make(map[string]string),
        cliPaths:          make(map[string]string),
        sessionStatuses:   make(map[string]status.SessionStatus),
    }
    e.loadSettingsFromDisk(cfgDir)   // NEW call
    return e
}
```

### Wails Binding Export (App.js manual update)

```javascript
// Append to frontend/src/wailsjs/go/main/App.js
export const OpenFileDialog = (defaultDir) => Call('main.App.OpenFileDialog', [defaultDir])
```

### Wails Type Declaration (App.d.ts manual update)

```typescript
// Append to frontend/src/wailsjs/go/main/App.d.ts
export function OpenFileDialog(defaultDir: string): Promise<string>
```

### GetCLIPaths Loading in SettingsTab

```typescript
// Source: SettingsTab.tsx (new useEffect)
import { UpdateCLIPath, GetCLIPaths, OpenFileDialog } from '../wailsjs/go/main/App'

useEffect(() => {
    GetCLIPaths().then(paths => {
        if (paths && Object.keys(paths).length > 0) {
            setCustomPaths(prev => ({ ...prev, ...paths }))
        }
    }).catch(() => { /* ignore — daemon may not be connected */ })
}, [])
```

### CSS for Saved Button State and Browse Button

```css
/* settings-panel__btn--saved: green confirmation state */
.settings-panel__btn--saved {
  background-color: #9ece6a;  /* green — matches CT acknowledged checkmark color */
  color: #1a1b26;
  font-weight: 600;
  cursor: default;
}

/* settings-panel__path-row: flex container for input + browse button */
.settings-panel__path-row {
  display: flex;
  gap: 6px;
  align-items: center;
}

/* settings-panel__browse-btn: compact browse button */
.settings-panel__browse-btn {
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  border: 1px solid #292e42;
  background: transparent;
  color: #9aa5ce;
  font-family: inherit;
  white-space: nowrap;
  flex-shrink: 0;
}
.settings-panel__browse-btn:hover {
  color: #c0caf5;
  border-color: #3b4261;
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Modal settings | Inline sidebar tab | Phase 78 | SettingsTab is already a sidebar component, no modal shell |
| Per-call detection | Cached `detectedCLIs` prop | Phase 74 | CLIs detected once on App mount; custom paths need separate load |

**Deprecated/outdated:**
- None identified for this phase.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `GetCLIPaths` is safe to call from SettingsTab useEffect (daemon is connected by the time Settings tab renders) | Code Examples | If daemon not connected, the call returns an error — caught by `.catch(() => {})`, no crash |
| A2 | Wails `App.js`/`App.d.ts` are manually maintained (not auto-generated at build time) | Standard Stack | If they ARE regenerated by `wails dev`, the OpenFileDialog export will be overwritten on next dev run — but the Go bound method in app.go will still cause it to appear correctly |

**Note on A2:** The file header says "AUTO-GENERATED by Wails — DO NOT edit manually" but the project clearly edits it manually (custom comments, hand-maintained exports). This is the established project pattern. [VERIFIED: wailsjs/go/main/App.js line 1 says auto-generated; lines 14-59 show manual additions with inline comments]

## Open Questions

1. **Does `GetCLIPaths` need to be added to App.js / App.d.ts?**
   - What we know: `GetCLIPaths` exists on `DaemonClient` (client.go line 88-94) and is exposed via `GET /settings/cli-paths`. It is NOT currently in the Wails bound methods.
   - What's unclear: Whether it was intentionally omitted or just not yet needed.
   - Recommendation: Add `GetCLIPaths` as a new Wails bound method in `app.go`, then export it in `App.js`/`App.d.ts`. The frontend needs it to load stored paths on SettingsTab mount.

## Environment Availability

Step 2.6: SKIPPED (no external tool dependencies — changes are pure code in the existing Go/React/Wails project. All required tools (go, pnpm, wails) already confirmed in use.)

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest (frontend), `go test` (daemon) |
| Config file | `frontend/vite.config.ts` |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test && cd /Users/ken/dev/agenthub && go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SET-01 | Agent paths persist across daemon restarts | unit (Go) | `go test ./internal/daemon/ -run TestSettings` | ❌ Wave 0 |
| SET-02 | Tailscale path persists across daemon restarts | unit (Go) | `go test ./internal/daemon/ -run TestSettings` | ❌ Wave 0 |
| SET-03 | Save shows "Saved!" confirmation | source inspection | `cd frontend && pnpm test` (in new test file) | ❌ Wave 0 |
| SET-04 | Browse button per path row | source inspection | `cd frontend && pnpm test` (in new test file) | ❌ Wave 0 |
| SET-05 | Browse result populates input | source inspection | `cd frontend && pnpm test` (in new test file) | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test && cd /Users/ken/dev/agenthub && go test ./internal/daemon/...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/daemon/engine_settings_test.go` — covers SET-01, SET-02 (load/save round-trip)
- [ ] `frontend/src/components/__tests__/SettingsTab.persistence.test.tsx` — covers SET-03, SET-04, SET-05 (source inspection)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes (partial) | Path validated by `os.Stat` in `UpdateCLIPath` — already exists |
| V6 Cryptography | no | — |

**Threat notes:**
- `settings.json` written with `0600` permissions (user-only) — consistent with `ct_disclosed` file pattern. [ASSUMED: 0600 is appropriate; matches existing convention]
- Path validation (`os.Stat`) already present in `UpdateCLIPath` prevents storing paths to non-existent files. No additional validation needed.
- `OpenFileDialog` returns an OS-selected path; the user is selecting their own files from their own filesystem. No injection risk.

## Sources

### Primary (HIGH confidence)
- `go doc github.com/wailsapp/wails/v2/pkg/runtime` — confirmed `OpenFileDialog`, `OpenDirectoryDialog`, `OpenDialogOptions`, `FileFilter`
- `/Users/ken/dev/agenthub/internal/daemon/engine.go` — confirmed `cliPaths` is in-memory only, no disk write
- `/Users/ken/dev/agenthub/internal/daemon/process.go` — confirmed `NewSessionEngine()` has no settings load
- `/Users/ken/dev/agenthub/app.go` — confirmed `OpenDirectoryDialog` pattern to mirror for `OpenFileDialog`
- `/Users/ken/dev/agenthub/frontend/src/components/SettingsTab.tsx` — confirmed `saving` state, no `saved` state
- `/Users/ken/dev/agenthub/frontend/src/wailsjs/go/main/App.js` — confirmed `GetCLIPaths` not exported

### Secondary (MEDIUM confidence)
- `go doc github.com/wailsapp/wails/v2/pkg/runtime FileFilter` — confirmed struct fields `DisplayName`, `Pattern`
- Source inspection of `engine_test.go` — confirmed test patterns for daemon package

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in use, no new dependencies
- Architecture: HIGH — all relevant source files read and understood
- Pitfalls: HIGH — identified from actual code (lock ordering, empty-string cancel, stale prop)
- Test patterns: HIGH — existing tests clearly establish source-inspection approach

**Research date:** 2026-04-16
**Valid until:** 2026-05-16 (stable Wails version, no fast-moving dependencies)
