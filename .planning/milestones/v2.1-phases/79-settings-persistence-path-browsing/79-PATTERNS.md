# Phase 79: Settings Persistence & Path Browsing - Pattern Map

**Mapped:** 2026-04-16
**Files analyzed:** 7
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/daemon/engine.go` | service | file-I/O | `internal/daemon/engine.go` (self — add methods) | exact |
| `app.go` | service | request-response | `app.go` `OpenDirectoryDialog` method (lines 474-487) | exact |
| `frontend/src/components/SettingsTab.tsx` | component | request-response | `SettingsTab.tsx` (self — add state + JSX) | exact |
| `frontend/src/wailsjs/go/main/App.js` | utility | request-response | `App.js` existing exports (lines 36-59) | exact |
| `frontend/src/wailsjs/go/main/App.d.ts` | utility | request-response | `App.d.ts` existing declarations (lines 49-98) | exact |
| `frontend/src/style.css` | config | — | `style.css` `.settings-panel__btn--save/--cancel` (lines 441-461) | exact |
| `internal/daemon/engine_settings_test.go` | test | file-I/O | `internal/daemon/engine_test.go` `TestOpenCodeTUIConfig` (lines 404-429) | exact |
| `frontend/src/components/__tests__/SettingsTab.persistence.test.tsx` | test | — | `SettingsTab.web-link-ux.test.tsx` + `SettingsTab.test.tsx` (source-inspection style) | exact |

---

## Pattern Assignments

### `internal/daemon/engine.go` — add `configDir` field + `loadSettingsFromDisk` + `saveSettingsToDisk`

**Analog:** `internal/daemon/engine.go` — `ensureOpenCodeTUIConfig` (file write) + `UpdateCLIPath` (mutex discipline)

**Existing struct + constructor** (`engine.go` lines 20-75):
```go
type SessionEngine struct {
    hostname          string
    opencodeTUIConfig string
    // ... other fields ...
    mu          sync.RWMutex
    cliPaths    map[string]string
}

func NewSessionEngine() *SessionEngine {
    hostname, _ := os.Hostname()
    tuiConfig := ensureOpenCodeTUIConfig(daemonConfigDir())
    return &SessionEngine{
        hostname:          hostname,
        opencodeTUIConfig: tuiConfig,
        // ... maps initialized ...
        cliPaths:          make(map[string]string),
    }
}
```

**File write pattern to copy** (`engine.go` lines 53-58 — `ensureOpenCodeTUIConfig`):
```go
func ensureOpenCodeTUIConfig(dir string) string {
    path := filepath.Join(dir, "opencode-tui.json")
    content := []byte("{\"$schema\":\"https://opencode.ai/tui.json\",\"theme\":\"system\"}\n")
    _ = os.WriteFile(path, content, 0644)
    return path
}
```
Note: `settings.json` uses `0600` (user-only) and `encoding/json.Marshal` instead of a raw literal.

**Mutex discipline to copy** (`engine.go` lines 234-242 — `UpdateCLIPath`):
```go
func (e *SessionEngine) UpdateCLIPath(name, path string) error {
    if _, err := os.Stat(path); err != nil {
        return fmt.Errorf("custom CLI path %q: %w", path, err)
    }
    e.mu.Lock()
    e.cliPaths[name] = path
    e.mu.Unlock()
    return nil
}
```
Critical constraint: `saveSettingsToDisk` is called **inside** the `e.mu.Lock()` block in `UpdateCLIPath`. It must NOT re-acquire the lock (deadlock). Accessing `e.cliPaths` directly is safe because the caller holds the lock.

**Required imports** (`engine.go` lines 3-13 — already present):
```go
import (
    "encoding/json"   // add for Marshal/Unmarshal
    "os"
    "path/filepath"
    "sync"
    // ... existing imports ...
)
```
`encoding/json` must be added; all others are already imported.

---

### `app.go` — add `OpenFileDialog` bound method + `GetCLIPaths` bound method

**Analog:** `app.go` `OpenDirectoryDialog` method (lines 474-487)

**Exact pattern to mirror** (`app.go` lines 474-487):
```go
// OpenDirectoryDialog opens a native OS folder picker and returns the selected path.
// Returns "" if the user cancels. Falls back to the user's home directory when
// defaultDir is empty.
func (a *App) OpenDirectoryDialog(defaultDir string) (string, error) {
    if defaultDir == "" {
        if home, err := os.UserHomeDir(); err == nil {
            defaultDir = home
        }
    }
    return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
        Title:            "Select Working Directory",
        DefaultDirectory: defaultDir,
    })
}
```

**New method copies this pattern verbatim**, changing:
- Function name: `OpenDirectoryDialog` → `OpenFileDialog`
- Title: `"Select Working Directory"` → `"Select Executable"`
- Runtime call: `runtime.OpenDirectoryDialog` → `runtime.OpenFileDialog`
- Add `ShowHiddenFiles: true` (agents may be in `~/.local/bin`)

**`GetCLIPaths` bound method** — copy pattern from `UpdateCLIPath` (lines 307-312):
```go
func (a *App) UpdateCLIPath(name, path string) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.UpdateCLIPath(name, path)
}
```
New method: same nil-client guard, delegates to `a.client.GetCLIPaths()`.

---

### `frontend/src/components/SettingsTab.tsx` — add `saved` state + `handleBrowse` + browse buttons + `useEffect` for `GetCLIPaths`

**Analog:** `SettingsTab.tsx` itself — existing patterns for `saving` state (line 54), `setCopied` pattern (line 67), `useEffect` (lines 76-95), `setCustomPaths` (lines 161-175).

**Existing `saving` state + two-state button** (lines 54, 161-182, 456-462):
```typescript
const [saving, setSaving] = useState(false)

async function handleSaveCLIPaths() {
    setSaving(true)
    setError(null)
    try {
        for (const cli of clis) { /* ... */ }
        // ... tailscale path handling ...
    } catch (err) {
        setError(err instanceof Error ? err.message : String(err))
    } finally {
        setSaving(false)
    }
}

// Button render:
<button
    className="settings-panel__btn settings-panel__btn--save"
    onClick={handleSaveCLIPaths}
    disabled={saving}
>
    {saving ? 'Saving\u2026' : 'Save Paths'}
</button>
```

**`setCopied` transient-state pattern to copy for `saved`** (lines 67, 115-120):
```typescript
const [copied, setCopied] = useState(false)

async function handleCopyPassword() {
    if (!localPassword) return
    await navigator.clipboard.writeText(localPassword)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
}
```
Apply same pattern: `const [saved, setSaved] = useState(false)`, set `true` after successful save, `setTimeout(() => setSaved(false), 1500)`.

**`useEffect` mount pattern to copy for `GetCLIPaths` load** (lines 76-95):
```typescript
useEffect(() => {
    async function loadWebState() {
        try {
            const [running, ctAck] = await Promise.all([...])
            // ...
        } catch (err) {
            console.error('[SettingsTab] loadWebState:', err)
        }
    }
    void loadWebState()
}, [])
```
New `useEffect` for `GetCLIPaths` follows the same pattern: async inner function, `void` invocation, empty dependency array `[]`, `.catch(() => {})` (silent — daemon may be reconnecting).

**Import block to extend** (lines 3-13):
```typescript
import {
    UpdateCLIPath,
    // ... existing imports ...
} from '../wailsjs/go/main/App'
```
Add `GetCLIPaths` and `OpenFileDialog` to this import block.

**Path input cell pattern to augment** (lines 402-419):
```tsx
{clis.map((cli) => (
    <tr key={cli.Name}>
        <td className="settings-panel__cli-name">{cli.Name}</td>
        <td>
            <input
                className="settings-panel__path-input"
                type="text"
                value={customPaths[cli.Name] ?? cli.Path}
                onChange={(e) =>
                    setCustomPaths((prev) => ({ ...prev, [cli.Name]: e.target.value }))
                }
                placeholder={cli.Path || `Path to ${cli.Name}`}
            />
        </td>
    </tr>
))}
```
The `<td>` containing `<input>` gains a `<div className="settings-panel__path-row">` wrapper holding both the input and a new `<button className="settings-panel__browse-btn">`.

---

### `frontend/src/wailsjs/go/main/App.js` — add `OpenFileDialog` and `GetCLIPaths` exports

**Analog:** `App.js` existing export lines — copy the comment-grouped pattern exactly (lines 35-59):

```javascript
// Directory dialog bound method
export const OpenDirectoryDialog  = (defaultDir)            => Call('main.App.OpenDirectoryDialog', [defaultDir])

// Tailscale health bound method
export const GetTailscaleStatus   = ()                      => Call('main.App.GetTailscaleStatus', [])
```

New lines follow the same column-aligned `=>` format and comment grouping:
```javascript
// File dialog bound method (SET-04)
export const OpenFileDialog  = (defaultDir)  => Call('main.App.OpenFileDialog', [defaultDir])

// CLI paths getter bound method (SET-01/02)
export const GetCLIPaths     = ()            => Call('main.App.GetCLIPaths', [])
```

---

### `frontend/src/wailsjs/go/main/App.d.ts` — add `OpenFileDialog` and `GetCLIPaths` type declarations

**Analog:** `App.d.ts` existing declarations — copy the comment + single-line function declaration pattern (lines 48-51):

```typescript
// Directory dialog bound method
export function OpenDirectoryDialog(defaultDir: string): Promise<string>
```

New declarations:
```typescript
// File dialog bound method (SET-04)
export function OpenFileDialog(defaultDir: string): Promise<string>

// CLI paths getter bound method (SET-01/02)
export function GetCLIPaths(): Promise<Record<string, string>>
```

---

### `frontend/src/style.css` — add `--saved` modifier + browse button + path-row container

**Analog:** `style.css` `.settings-panel__btn--save` and `.settings-panel__btn--cancel` (lines 441-461):

```css
.settings-panel__btn--save {
  background-color: #7aa2f7;
  color: #1a1b26;
  font-weight: 600;
}
.settings-panel__btn--save:hover:not(:disabled) {
  background-color: #89b4fa;
}
.settings-panel__btn--save:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
```

New `--saved` modifier follows this pattern with green background (`#9ece6a` — the CT-acknowledged checkmark color already in the file):
```css
.settings-panel__btn--saved {
  background-color: #9ece6a;
  color: #1a1b26;
  font-weight: 600;
  cursor: default;
}
.settings-panel__btn--saved:disabled {
  opacity: 1;    /* green stays fully opaque — it's a confirmation, not disabled state */
}
```

Browse button and path-row container follow the existing `.settings-panel__btn--cancel` outline-button pattern (`background: transparent; border: 1px solid #292e42; color: #9aa5ce`).

---

### `internal/daemon/engine_settings_test.go` — new Go test file (SET-01, SET-02)

**Analog:** `internal/daemon/engine_test.go` — `TestOpenCodeTUIConfig` (lines 404-429) and `TestEngineResolveCLI` (lines 214-233)

**`TestOpenCodeTUIConfig` pattern to copy** (lines 404-429):
```go
func TestOpenCodeTUIConfig(t *testing.T) {
    dir := t.TempDir()
    path := ensureOpenCodeTUIConfig(dir)

    expected := filepath.Join(dir, "opencode-tui.json")
    if path != expected {
        t.Errorf("path = %q, want %q", path, expected)
    }

    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("managed opencode-tui.json not found: %v", err)
    }
    // ... content assertion ...
}
```
New settings tests use `t.TempDir()` as `configDir`, create an engine with the temp dir, call `UpdateCLIPath`, then assert that the temp dir contains `settings.json` with the expected content. A second engine constructed from the same dir asserts that `GetCLIPaths()` returns the persisted values.

**`TestEngineResolveCLI` pattern for path validation** (lines 214-233):
```go
if err := e.UpdateCLIPath("claude", "/bin/cat"); err != nil {
    t.Fatalf("UpdateCLIPath: %v", err)
}
got = e.ResolveCLI("claude")
if got != "/bin/cat" {
    t.Errorf("ResolveCLI after update: got %q, want %q", got, "/bin/cat")
}
```

---

### `frontend/src/components/__tests__/SettingsTab.persistence.test.tsx` — new source-inspection test file

**Analog:** `SettingsTab.web-link-ux.test.tsx` (full file — 109 lines) and `SettingsTab.test.tsx`

**Source-inspection import pattern** (`SettingsTab.web-link-ux.test.tsx` lines 1-7):
```typescript
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'
import raw from '../../components/SettingsTab.tsx?raw'

// Use fs.readFileSync for CSS — vitest/jsdom does not support ?raw imports for CSS.
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')
```

**Assertion pattern for state variable existence** (`SettingsTab.test.tsx` line 73-76):
```typescript
it('has useEffect with empty dependency array []', () => {
    expect(raw).toContain('}, [])')
})
```

**Assertion pattern for import existence** (`SettingsTab.web-link-ux.test.tsx` lines 9-12):
```typescript
describe('WEB-01: Open in browser button', () => {
    it('imports BrowserOpenURL from Wails runtime', () => {
        expect(raw).toContain('BrowserOpenURL')
    })
})
```

**CSS class assertion pattern** (`SettingsTab.web-link-ux.test.tsx` lines 77-86):
```typescript
describe('CSS: URL action row classes', () => {
    it('has settings-web-server__url-row class', () => {
        expect(cssRaw).toContain('.settings-web-server__url-row')
    })
})
```

All new tests use `expect(raw).toContain(...)` and `expect(cssRaw).toContain(...)`. No DOM rendering or Wails mock calls.

---

## Shared Patterns

### Nil-client guard (Go)
**Source:** `app.go` — every bound method (e.g., lines 307-312)
**Apply to:** new `GetCLIPaths` and `OpenFileDialog` bound methods in `app.go`
```go
if a.client == nil {
    return fmt.Errorf("daemon not connected")
}
```

### Mutex discipline — caller-holds-lock
**Source:** `engine.go` `UpdateCLIPath` (lines 234-242)
**Apply to:** `saveSettingsToDisk` call inside `UpdateCLIPath`
The lock is acquired by `UpdateCLIPath` before the map write. `saveSettingsToDisk` accesses `e.cliPaths` without acquiring the lock. Precondition is documented in a comment: `// Caller holds e.mu.Lock()`.

### File write with 0600 permissions
**Source:** `app.go` `AcknowledgeCTDisclosure` (line 422): `os.WriteFile(ctDisclosurePath(), []byte("1"), 0600)`
**Apply to:** `saveSettingsToDisk` in `engine.go`
All user-private config files in this project use `0600`.

### Transient boolean state with `setTimeout` reset
**Source:** `SettingsTab.tsx` `handleCopyPassword` (lines 115-120), `handleCopyURL` (lines 122-127)
**Apply to:** `saved` state after successful `handleSaveCLIPaths`
```typescript
setCopied(true)
setTimeout(() => setCopied(false), 1500)
```

### Silent async `useEffect` (daemon may not be connected)
**Source:** `SettingsTab.tsx` `useEffect` for LAN password (lines 98-104):
```typescript
GetLocalNetworkPassword().then(setLocalPassword).catch(() => setLocalPassword(''))
```
**Apply to:** `GetCLIPaths` `useEffect` — `.catch(() => {})` (no user-visible error; daemon connectivity is handled elsewhere).

### Wails `App.js` export format
**Source:** `App.js` lines 36-59 — column-aligned `=>` assignments with comment group headers
**Apply to:** all new exports in `App.js`

### Source-inspection test pattern
**Source:** `SettingsTab.web-link-ux.test.tsx` lines 1-7 + `SettingsTab.test.tsx` lines 1-3
**Apply to:** `SettingsTab.persistence.test.tsx`
Import component as `?raw`, CSS with `readFileSync`. Assert on string content only. No DOM, no mock Wails calls.

---

## No Analog Found

None. All files have close analogs in the codebase.

---

## Metadata

**Analog search scope:** `/Users/ken/dev/agenthub/app.go`, `internal/daemon/engine.go`, `internal/daemon/engine_test.go`, `frontend/src/components/SettingsTab.tsx`, `frontend/src/wailsjs/go/main/App.js`, `frontend/src/wailsjs/go/main/App.d.ts`, `frontend/src/style.css`, `frontend/src/components/__tests__/SettingsTab.test.tsx`, `frontend/src/components/__tests__/SettingsTab.web-link-ux.test.tsx`
**Files scanned:** 9
**Pattern extraction date:** 2026-04-16
