# Phase 11: New-Session Modal - Research

**Researched:** 2026-03-19
**Domain:** React modal UI, Wails v2 dialog API, Go PTY working directory, localStorage persistence
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SESS-01 | Clicking + opens a modal (not a dropdown) for creating a new session | Replace `showCLIPicker` dropdown with a `showNewSessionModal` boolean and a full modal component; same trigger point in `handleAddTab` |
| SESS-02 | New-session modal includes an agent picker showing available CLIs | `detectedCLIs` already loaded in App state; render radio buttons or option buttons per `DetectedCLI.DisplayName` inside the modal |
| SESS-03 | New-session modal includes a native folder browser for selecting the working directory | Add `OpenDirectoryDialog` Go binding; call it from a "Browse" button in the modal; pass selected path back as `workDir` state |
| SESS-04 | Folder browser defaults to home directory, or last-used folder if one exists | Persist last-used path to `localStorage`; pass it as `DefaultDirectory` in `OpenDialogOptions`; fall back to `os.UserHomeDir()` resolved on the Go side |
</phase_requirements>

---

## Summary

Phase 11 replaces the existing CLI-picker dropdown with a proper modal dialog for new session creation. The modal must show an agent picker, a folder browser trigger, and a path display. This touches three layers: the React frontend (new modal component), the Wails App Go struct (new `OpenDirectoryDialog` bound method), and the PTY backend (`CreateRequest.WorkDir` field and `cmd.Dir` assignment).

The Wails v2 runtime already provides `runtime.OpenDirectoryDialog(ctx, opts)` which opens a native OS folder picker and returns the selected path. The go-pty `Cmd` struct already exposes a `Dir` field that sets the process working directory — it is assigned from `req.CLI`'s `CommandContext` return value's `Dir` field. Both ends of the plumbing exist; only the wiring is new.

Last-used folder memory is cleanly solved with `localStorage` on the frontend — no backend persistence needed. The key is written after the user confirms a folder and read when the modal opens next time.

**Primary recommendation:** New `NewSessionModal` component in React, one new Go bound method `OpenDirectoryDialog`, one new `OpenDirectoryDialog` TypeScript stub, and `WorkDir` added to `CreateRequest` + `CreateSession` signature.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React (already in project) | 18.x | Modal component | Already used everywhere |
| Wails v2 runtime | v2.10.2 | `OpenDirectoryDialog` native OS dialog | Built into the framework already in use |
| go-pty `Cmd.Dir` | v0.2.2 | Working directory for PTY process | Already a field on `Cmd`; zero new deps |
| `localStorage` (browser) | Web API | Last-used folder persistence | Zero deps, survives app restarts |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os.UserHomeDir()` (stdlib) | Go stdlib | Fallback home directory | When no last-used folder is stored |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `localStorage` | Go-side config file | localStorage is simpler — no new Go storage layer needed; acceptable since it's UI-only state |
| Wails `OpenDirectoryDialog` | Custom file tree in HTML | Native dialog is required by SESS-03; don't build custom UI |

**Installation:** No new packages needed. Everything required is already in go.mod and package.json.

---

## Architecture Patterns

### Recommended Project Structure

No new directories. New files:

```
frontend/src/components/
├── NewSessionModal.tsx       # new — the modal component
└── __tests__/
    └── NewSessionModal.test.tsx  # new — source-inspection tests
```

Changes to existing files:

```
frontend/src/
└── App.tsx                   # add showNewSessionModal state, update handleAddTab,
                              # update createTab signature to accept workDir

frontend/src/wailsjs/go/main/
└── App.d.ts                  # add OpenDirectoryDialog binding stub

app.go                        # add OpenDirectoryDialog method + update CreateSession/CreateRequest
internal/pty/
└── backend.go                # add WorkDir to CreateRequest
└── native.go                 # assign cmd.Dir = req.WorkDir
```

### Pattern 1: Modal controlled by App state (consistent with SettingsPanel)

**What:** Modal open/close state held in App with a boolean, passed as `isOpen` prop. The modal fires `onConfirm(cli, workDir)` when the user clicks Create.
**When to use:** All modals in this codebase follow this pattern (SettingsPanel, QRModal).

```typescript
// App.tsx — analogous to showSettings
const [showNewSessionModal, setShowNewSessionModal] = useState(false)

// handleAddTab — REPLACES current dropdown logic
const handleAddTab = useCallback(() => {
  if (detectedCLIs.length === 0) {
    setShowSettings(true)
    return
  }
  setShowNewSessionModal(true)  // always open modal, regardless of CLI count
}, [detectedCLIs])

// createTab updated to accept workDir
const createTab = useCallback(async (cliName: string, workDir: string) => {
  // ... same as current, passes workDir to CreateSession
}, [tabCounter])
```

### Pattern 2: JSX conditional rendering (consistent with Phase 8/9 decisions)

**What:** `{showNewSessionModal && <NewSessionModal ... />}` — not CSS display toggle.
**When to use:** All modal state in this codebase (Phase 8/9 decision: JSX conditionals, not CSS).

```typescript
{showNewSessionModal && (
  <NewSessionModal
    isOpen={showNewSessionModal}
    clis={detectedCLIs}
    onConfirm={(cli, workDir) => {
      setShowNewSessionModal(false)
      void createTab(cli, workDir)
    }}
    onClose={() => setShowNewSessionModal(false)}
  />
)}
```

### Pattern 3: Wails dialog via bound Go method (not frontend JS)

**What:** Native folder picker requires a Go-side call to `runtime.OpenDirectoryDialog`. Expose it as a bound method on `App`. Call it from React via the generated binding.
**When to use:** Any native OS dialog in a Wails app.

```go
// app.go — new bound method
func (a *App) OpenDirectoryDialog(defaultDir string) (string, error) {
    return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
        Title:            "Select Working Directory",
        DefaultDirectory: defaultDir,
    })
}
```

```typescript
// frontend — after calling, empty string means user cancelled
const path = await OpenDirectoryDialog(lastUsedDir)
if (path !== '') {
  setSelectedDir(path)
  localStorage.setItem('agenthub:lastWorkDir', path)
}
```

### Pattern 4: localStorage for last-used folder

**What:** Read on modal open, write on user confirms a folder selection.
**When to use:** Lightweight UI-only state that should survive restarts but needs no server round-trip.

```typescript
const LAST_DIR_KEY = 'agenthub:lastWorkDir'

// In NewSessionModal on mount / open
const saved = localStorage.getItem(LAST_DIR_KEY) ?? ''
// Pass saved (or empty string) as defaultDir to OpenDirectoryDialog

// After successful folder pick
localStorage.setItem(LAST_DIR_KEY, selectedPath)
```

The Go-side fallback: when `defaultDir` is empty string, Wails `OpenDirectoryDialog` will open to the OS default (usually home). Alternatively, the `OpenDirectoryDialog` Go method can call `os.UserHomeDir()` when `defaultDir == ""`.

### Pattern 5: WorkDir propagation through Go layers

**What:** `CreateRequest.WorkDir` string flows from App bound method → backend → PTY `cmd.Dir`.
**When to use:** Any time the PTY must start in a specific directory.

```go
// internal/pty/backend.go — add field to CreateRequest
type CreateRequest struct {
    CLI     string
    Args    []string
    Env     []string
    Cols    int
    Rows    int
    WorkDir string  // new: working directory for the CLI process
}

// internal/pty/native.go — assign after CommandContext
cmd := p.CommandContext(childCtx, req.CLI, req.Args...)
cmd.Dir = req.WorkDir  // go-pty Cmd.Dir is supported on both Unix and Windows

// app.go — update CreateSession signature
func (a *App) CreateSession(cli, name, workDir string) (string, error) {
    sess, err := a.backend.Create(a.ctx, pty.CreateRequest{
        CLI:     cliPath,
        WorkDir: workDir,
        Cols:    80,
        Rows:    24,
    })
    // ...
}
```

### NewSessionModal Component Structure

```typescript
interface NewSessionModalProps {
  isOpen: boolean
  clis: DetectedCLI[]
  onConfirm: (cli: string, workDir: string) => void
  onClose: () => void
}

// Internal state:
// selectedCLI: string (default to clis[0].Name)
// selectedDir: string (from localStorage or empty)
// browseLoading: boolean
```

### Anti-Patterns to Avoid

- **CSS display toggle for modal:** Use JSX conditional `{showModal && <Modal />}` — consistent with Phase 8/9 pattern.
- **Calling `window.showDirectoryPicker`:** Browser File System Access API is blocked by Wails webview security policies; always use `runtime.OpenDirectoryDialog` via the bound method.
- **Storing last-used folder in Go config file:** localStorage is sufficient for this UI preference; no need for a new Go persistence layer.
- **Passing `workDir` as empty string to PTY when cancelled:** If the user does not pick a folder, pass `""` and let the PTY default to the app's working directory (this is fine behavior).
- **Showing the modal only when multiple CLIs detected:** SESS-01 requires the modal always opens on `+` click. Remove the single-CLI fast-path that bypasses the modal.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Native OS folder picker | Custom file tree HTML component | `runtime.OpenDirectoryDialog` from Wails | Browser File System Access API unreliable in Wails webview; native dialog is platform-correct |
| Last-used folder persistence | Go config file / JSON store | `localStorage.setItem/getItem` | No server needed; survives restarts; zero complexity |
| Working directory for PTY | Custom `chdir` call in Go | `cmd.Dir = req.WorkDir` via go-pty `Cmd.Dir` | go-pty already supports `Dir` on both Unix and Windows (verified in source) |

**Key insight:** All primitives exist. This phase is plumbing, not invention.

---

## Common Pitfalls

### Pitfall 1: `showCLIPicker` state left dangling

**What goes wrong:** The old `showCLIPicker` boolean and its JSX block remain in App.tsx alongside the new modal. Two conflicting code paths create confusing behavior.
**Why it happens:** Incremental addition without removing old path.
**How to avoid:** Delete `showCLIPicker` state, the `setShowCLIPicker` calls, and the `.cli-picker-overlay` JSX block entirely in the same commit. Also remove the `.cli-picker*` CSS classes from style.css.
**Warning signs:** Both picker dropdown and modal appear simultaneously.

### Pitfall 2: `OpenDirectoryDialog` not added to the TypeScript stub

**What goes wrong:** The bound method exists in Go but the frontend import fails at build time because `App.d.ts` has no declaration for it.
**Why it happens:** `App.d.ts` is manually maintained in this project (not auto-generated at dev time per the file header).
**How to avoid:** Add the declaration to `App.d.ts` and the corresponding `Call(...)` wrapper to `App.js` in the same wave as the Go method.
**Warning signs:** TypeScript error on import; runtime `Call` errors in console.

### Pitfall 3: CreateSession called with wrong argument count

**What goes wrong:** `CreateSession` signature changes from `(cli, name)` to `(cli, name, workDir)` in Go; the old TypeScript call in App.tsx still passes two args.
**Why it happens:** Go and TypeScript layers change independently.
**How to avoid:** Update `App.d.ts`, `App.js`, and all call sites in `App.tsx` in the same task.
**Warning signs:** Wails RPC error in console ("wrong number of arguments").

### Pitfall 4: Empty string `workDir` on first use

**What goes wrong:** `localStorage.getItem(LAST_DIR_KEY)` returns `null` on first use. Passing `null` (not `""`) to the Go method causes a type error.
**Why it happens:** `localStorage.getItem` returns `null` not `""` for missing keys.
**How to avoid:** Use `localStorage.getItem(LAST_DIR_KEY) ?? ''` everywhere.

### Pitfall 5: `OpenDirectoryDialog` returns `""` on cancel

**What goes wrong:** User cancels the OS dialog; `""` is returned. Code treats this as a valid path, overwriting the displayed directory with an empty string.
**Why it happens:** Wails returns `""` on user cancel (documented behavior).
**How to avoid:** Check `if (path !== '') { setSelectedDir(path); localStorage.setItem(...) }`. Do not update state on cancel.

### Pitfall 6: `cmd.Dir` not set before `cmd.Start()` on Windows

**What goes wrong:** On Windows, `cmd.Dir` must be set before `Start()` is called (go-pty assigns it in `cmd_windows.go` during `start()`). Setting it after `Start` is a no-op.
**Why it happens:** Order matters on Windows.
**How to avoid:** The assignment `cmd.Dir = req.WorkDir` in `native.go` must appear before `cmd.Start()` — which it naturally does if added immediately after `p.CommandContext(...)`.

---

## Code Examples

### Go: OpenDirectoryDialog bound method

```go
// Source: Wails v2 runtime docs + pkg.go.dev/github.com/wailsapp/wails/v2/pkg/runtime
// app.go

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

### TypeScript: App.d.ts stub addition

```typescript
// frontend/src/wailsjs/go/main/App.d.ts — add alongside existing declarations
export function OpenDirectoryDialog(defaultDir: string): Promise<string>
```

### TypeScript: App.js binding addition

```javascript
// frontend/src/wailsjs/go/main/App.js — add alongside existing Call wrappers
export function OpenDirectoryDialog(defaultDir) {
    return Call("main.App.OpenDirectoryDialog", [defaultDir])
}
```

### Go: CreateRequest WorkDir + native.go assignment

```go
// internal/pty/backend.go
type CreateRequest struct {
    CLI     string
    Args    []string
    Env     []string
    Cols    int
    Rows    int
    WorkDir string
}

// internal/pty/native.go — after p.CommandContext(...)
cmd := p.CommandContext(childCtx, req.CLI, req.Args...)
cmd.Dir = req.WorkDir  // Source: go-pty v0.2.2 cmd.go:32
```

### React: NewSessionModal skeleton

```typescript
// frontend/src/components/NewSessionModal.tsx
import React, { useState } from 'react'
import { OpenDirectoryDialog } from '../wailsjs/go/main/App'
import type { DetectedCLI } from '../wailsjs/go/main/App'

const LAST_DIR_KEY = 'agenthub:lastWorkDir'

interface NewSessionModalProps {
  isOpen: boolean
  clis: DetectedCLI[]
  onConfirm: (cli: string, workDir: string) => void
  onClose: () => void
}

export function NewSessionModal({ isOpen, clis, onConfirm, onClose }: NewSessionModalProps) {
  const [selectedCLI, setSelectedCLI] = useState(clis[0]?.Name ?? '')
  const [selectedDir, setSelectedDir] = useState(() => localStorage.getItem(LAST_DIR_KEY) ?? '')
  const [browseLoading, setBrowseLoading] = useState(false)

  if (!isOpen) return null

  async function handleBrowse() {
    setBrowseLoading(true)
    try {
      const path = await OpenDirectoryDialog(selectedDir)
      if (path !== '') {
        setSelectedDir(path)
        localStorage.setItem(LAST_DIR_KEY, path)
      }
    } finally {
      setBrowseLoading(false)
    }
  }

  function handleConfirm() {
    onConfirm(selectedCLI, selectedDir)
  }

  return (
    <div className="new-session-overlay" onClick={onClose}>
      <div className="new-session-modal" onClick={(e) => e.stopPropagation()}>
        {/* ... header, agent picker, folder picker, confirm/cancel */}
      </div>
    </div>
  )
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `showCLIPicker` dropdown (inline below tab bar) | `NewSessionModal` full center modal | Phase 11 | SESS-01 compliance; better UX for multi-step flow |
| `CreateSession(cli, name)` — no working directory | `CreateSession(cli, name, workDir)` — explicit working directory | Phase 11 | CLIs start in user's chosen project folder |
| No folder memory | `localStorage` persists last-used path | Phase 11 | SESS-04 compliance |

**Deprecated/outdated after this phase:**
- `showCLIPicker` state + JSX + CSS classes — remove entirely.
- Single-CLI fast-path bypassing the modal in `handleAddTab` — remove; modal always opens.

---

## Open Questions

1. **Default tab name when workDir is selected**
   - What we know: Current `createTab` names tabs `${cliName} ${tabCounter}`.
   - What's unclear: Should the tab name default to the basename of `workDir` (e.g., "my-project") when a folder is chosen?
   - Recommendation: Keep the current numeric naming for now; SESS-01 through SESS-04 do not require name defaulting. Can be added as a UX polish later.

2. **What happens when the selected workDir no longer exists at session create time**
   - What we know: `cmd.Dir` will cause `start()` to fail with "no such file or directory" if the path doesn't exist.
   - What's unclear: Should Go validate path existence before spawning?
   - Recommendation: Let it fail naturally — the error propagates to `CreateSession` which returns `(string, error)`, and the React `catch` block already logs the error. No special handling required for v1.1.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest (via `vitest/config`) |
| Config file | `frontend/vitest.config.ts` |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SESS-01 | `+` click shows modal (not dropdown) | source-inspection | `pnpm test -- NewSessionModal` | Wave 0 |
| SESS-02 | Modal lists detected CLIs | source-inspection | `pnpm test -- NewSessionModal` | Wave 0 |
| SESS-03 | Modal has a "Browse" button that calls `OpenDirectoryDialog` | source-inspection | `pnpm test -- NewSessionModal` | Wave 0 |
| SESS-04 | Modal reads last-used dir from localStorage on open | source-inspection | `pnpm test -- NewSessionModal` | Wave 0 |

All tests follow the project's source-inspection pattern (import file as `?raw`, assert text presence). No DOM rendering required given complexity of Wails bindings in jsdom.

### Sampling Rate

- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/components/__tests__/NewSessionModal.test.tsx` — covers SESS-01, SESS-02, SESS-03, SESS-04

---

## Sources

### Primary (HIGH confidence)

- go-pty v0.2.2 source (`~/go/pkg/mod/github.com/aymanbagabas/go-pty@v0.2.2/cmd.go:32`) — `Dir string` field confirmed in `Cmd` struct, used in both `cmd_unix.go:34` and `cmd_windows.go:36/40/69`
- Wails v2.10.2 already in go.mod — `runtime.OpenDirectoryDialog` API verified via pkg.go.dev and GitHub source
- Project source code (`app.go`, `App.d.ts`, `App.js`, `SettingsPanel.tsx`, `QRModal.tsx`) — modal pattern, binding pattern, CSS naming conventions all confirmed by direct read

### Secondary (MEDIUM confidence)

- [Wails v2 Dialog API docs](https://wails.io/docs/reference/runtime/dialog/) — `OpenDirectoryDialog(ctx, OpenDialogOptions) (string, error)`; returns `""` on cancel (403 on fetch, verified via pkg.go.dev)
- [pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options/dialog](https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options/dialog) — `OpenDialogOptions.DefaultDirectory` field confirmed

### Tertiary (LOW confidence)

- None.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in project; no new dependencies
- Architecture: HIGH — modal/binding patterns confirmed by reading existing components (SettingsPanel, QRModal)
- Pitfalls: HIGH — go-pty `Dir` field verified in source; Wails cancel behavior (`""`) from docs; localStorage null-vs-empty verified from Web API knowledge

**Research date:** 2026-03-19
**Valid until:** 2026-06-19 (Wails v2, go-pty, React — stable libraries)
