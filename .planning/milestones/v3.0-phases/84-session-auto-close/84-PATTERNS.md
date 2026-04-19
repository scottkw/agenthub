# Phase 84: Session Auto-Close - Pattern Map

**Mapped:** 2026-04-19
**Files analyzed:** 10 new/modified files
**Analogs found:** 10 / 10

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/ExitToast.tsx` | component | event-driven | `frontend/src/components/LocalNetworkBanner.tsx` | role-match |
| `frontend/src/components/ExitCountdownBanner.tsx` | component | event-driven | `frontend/src/components/UpdateBanner.tsx` | role-match |
| `frontend/src/components/__tests__/ExitToast.test.tsx` | test | — | `frontend/src/components/__tests__/UpdateBanner.test.tsx` | exact |
| `frontend/src/components/__tests__/ExitCountdownBanner.test.tsx` | test | — | `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` | exact |
| `frontend/src/components/__tests__/App.exit.test.tsx` | test | — | `frontend/src/components/__tests__/App.wiring.test.tsx` | exact |
| `frontend/src/App.tsx` (modify) | provider/store | event-driven | self | exact |
| `frontend/src/components/TabBar.tsx` (modify) | component | request-response | self | exact |
| `frontend/src/style.css` (modify) | config | — | self (`.banner-exit`, `.tab` blocks) | exact |
| `internal/daemon/engine.go` (modify) | service | event-driven | self (settings pattern) | exact |
| `internal/daemon/types.go` (modify) | model | — | self | exact |
| `internal/daemon/api.go` (modify) | middleware/route | request-response | self (`handleGetStartMinimized` / `handleSetStartMinimized`) | exact |
| `internal/daemon/client.go` (modify) | service | request-response | self (`GetStartMinimized` / `SetStartMinimized`) | exact |
| `app.go` (modify) | provider | event-driven | self (`pollSessionStatus`, `KillSession` event emit) | exact |

---

## Pattern Assignments

### `frontend/src/components/ExitToast.tsx` (component, event-driven)

**Analog:** `frontend/src/components/LocalNetworkBanner.tsx`

**Imports pattern** (lines 1-2):
```typescript
import React from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'
```

**Props interface pattern** — follow LocalNetworkBanner's typed props, with an `onKeepOpen` and `onDismiss` callback pair:
```typescript
// From LocalNetworkBanner.tsx lines 4-14
interface LocalNetworkBannerProps {
  visible: boolean
  // ...fields per feature...
  onDismiss?: () => void
  className?: string
}
```

**Core render pattern** — conditional return null, role="status" or role="alert", BEM class naming, XMarkIcon dismiss button:
```typescript
// From LocalNetworkBanner.tsx lines 28-53
export function LocalNetworkBanner({ visible, ...props }: LocalNetworkBannerProps): React.ReactElement | null {
  if (!visible) return null
  return (
    <div className={`local-network-banner${className ? ' ' + className : ''}`} role="status">
      <span className="local-network-banner__icon">{'\u26a0'}</span>
      <span className="local-network-banner__message">...</span>
      {onDismiss && (
        <button
          type="button"
          className="local-network-banner__dismiss"
          aria-label="Dismiss local network notification"
          onClick={onDismiss}
        >
          <XMarkIcon style={{ width: 16, height: 16 }} />
        </button>
      )}
    </div>
  )
}
```

**Fixed-position toast pattern** — the toast is mounted at App level and needs `position: fixed`. The existing CSS anchor:
```css
/* From style.css lines 993-1004 — use as structural template for exit-toast */
.update-banner {
  background: #16161e;
  border: 1px solid #292e42;
  border-left: 3px solid #7aa2f7;
  border-radius: 4px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
```

**Multiple-items pattern** — ExitToast renders one card per sessionId entry. Follows the same `Record<string, ExitState>` iteration pattern used in App.tsx for `webEnabled`:
```typescript
// From App.tsx lines 59-60 (analog for per-session map state)
const [webEnabled, setWebEnabled] = useState<Record<string, boolean>>({})
```

---

### `frontend/src/components/ExitCountdownBanner.tsx` (component, event-driven)

**Analog:** `frontend/src/components/UpdateBanner.tsx`

**Imports pattern** (lines 1-2 of UpdateBanner.tsx):
```typescript
import React from 'react'
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'
// ExitCountdownBanner does NOT need BrowserOpenURL — replace with no import from runtime
```

**Interface + export pattern** (lines 4-14, UpdateBanner.tsx):
```typescript
export interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseURL: string
}

interface UpdateBannerProps {
  update: UpdateInfo
  onDismiss: () => void
  className?: string
}

export function UpdateBanner({ update, onDismiss, className }: UpdateBannerProps): React.ReactElement {
```

**Core render pattern** — single banner div with BEM classes, flex layout, action button, dismiss button (lines 16-50, UpdateBanner.tsx):
```typescript
export function UpdateBanner({ update, onDismiss, className }: UpdateBannerProps): React.ReactElement {
  return (
    <div
      className={`update-banner${className ? ' ' + className : ''}`}
      role="alert"
      aria-live="polite"
    >
      <span className="update-banner__message">...</span>
      <div className="update-banner__actions">
        <button type="button" className="update-banner__btn--download" onClick={...}>
          Download Update
        </button>
        <button
          type="button"
          className="update-banner__btn--dismiss"
          aria-label="Dismiss update notification"
          onClick={onDismiss}
        >
          Dismiss
        </button>
      </div>
    </div>
  )
}
```

**Placement:** ExitCountdownBanner is rendered inside the `terminal-wrapper` div in App.tsx (between `<TerminalPanel>` and `<StatusBar>`), so it appears within the session content area.

---

### `frontend/src/components/__tests__/ExitToast.test.tsx` (test)

**Analog:** `frontend/src/components/__tests__/UpdateBanner.test.tsx`

**Full test file structure** (lines 1-98, UpdateBanner.test.tsx):
```typescript
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { UpdateBanner } from '../UpdateBanner'
import type { UpdateInfo } from '../UpdateBanner'

// Mock Wails runtime if needed
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}))

function renderUpdateBanner(update: UpdateInfo, onDismiss: () => void, className?: string) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(UpdateBanner, { update, onDismiss, className }))
  })
  return { container, root }
}

describe('UpdateBanner', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root?.unmount()
    container?.remove()
    vi.clearAllMocks()
  })

  it('renders update version information', () => {
    ;({ container, root } = renderUpdateBanner(mockUpdate, vi.fn()))
    expect(container.textContent).toContain('1.0.0')
  })

  it('calls onDismiss when Dismiss button clicked', () => {
    const onDismiss = vi.fn()
    ;({ container, root } = renderUpdateBanner(mockUpdate, onDismiss))
    const dismissBtn = container.querySelector('.update-banner__btn--dismiss') as HTMLButtonElement
    flushSync(() => { dismissBtn.click() })
    expect(onDismiss).toHaveBeenCalledOnce()
  })
  // ...
})
```

**Key pattern:** `createRoot` + `flushSync` for all renders; `afterEach` unmounts and removes; `vi.fn()` for callbacks; `flushSync(() => { btn.click() })` for interaction tests.

---

### `frontend/src/components/__tests__/ExitCountdownBanner.test.tsx` (test)

**Analog:** `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx`

**Render helper pattern** (lines 19-35, LocalNetworkBanner.test.tsx):
```typescript
function renderBanner(props: Partial<LocalNetworkBannerProps> & { visible: boolean; onOpenURL: (url: string) => void }) {
  const fullProps: LocalNetworkBannerProps = {
    tailscaleConnected: false,
    tailscaleInstalled: false,
    tailscaleBinaryFound: false,
    tailscaleDaemonUp: false,
    platformHint: '',
    ...props,
  }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(LocalNetworkBanner, fullProps))
  })
  return { container, root }
}
```

Adapt: `renderBanner` for `ExitCountdownBanner` with default props `{ countdown: 5, cancelled: false, exitCode: 0, ... }`.

---

### `frontend/src/components/__tests__/App.exit.test.tsx` (test)

**Analog:** `frontend/src/components/__tests__/App.wiring.test.tsx`

**Source inspection pattern** (entire file, App.wiring.test.tsx lines 1-64):
```typescript
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

// Source inspection tests — verify wiring contract without mounting full component tree.
describe('App.tsx remote-sessions wiring (52-03-02)', () => {
  it('defines REMOTE_SESSIONS_TAB constant', () => {
    expect(raw).toContain('REMOTE_SESSIONS_TAB')
  })
  it('imports GetRemoteSessions from wailsjs binding', () => {
    expect(raw).toContain('GetRemoteSessions')
  })
  // ... 12 more `expect(raw).toContain(...)` assertions
})
```

**Key insight:** Source inspection tests (`?raw` import) verify App.tsx wiring without needing Wails runtime mocks. Use this for App.exit.test.tsx assertions like: `expect(raw).toContain("session:exit")`, `expect(raw).toContain('sessionExits')`, `expect(raw).toContain('countdownTimers')`, etc.

---

### `frontend/src/App.tsx` (modify — add exit event state + subscription)

**Analog:** self — existing `session:status` event subscription (lines 239-245) and banner dismiss pattern (lines 118-131).

**Event subscription pattern** (lines 240-244, App.tsx):
```typescript
// Subscribe to live session:status events from the Go backend.
const offStatus = EventsOn(
  'session:status',
  (data: { sessionId: string; status: string }) => {
    setSessionStatuses((prev) => ({ ...prev, [data.sessionId]: data.status }))
  },
)
```

**Cleanup pattern** — all `off*` unsubscribers collected in the useEffect cleanup (lines 313-322, App.tsx):
```typescript
return () => {
  offStatus()
  offHealth()
  offDaemonError()
  cancelTrayFocus()
  if (upgradePollerRef.current !== null) {
    clearInterval(upgradePollerRef.current)
    upgradePollerRef.current = null
  }
}
```

**Banner dismiss with exit animation** (lines 118-131, App.tsx):
```typescript
const handleDismissLocalBanner = useCallback(() => {
  setLocalBannerExiting(true)
  setTimeout(() => {
    setLocalBannerDismissed(true)
    setLocalBannerExiting(false)
  }, 200)
}, [])
```

**Ref for interval handle** (lines 86-87, App.tsx):
```typescript
const upgradePollerRef = useRef<ReturnType<typeof setInterval> | null>(null)
```
Copy: `const countdownTimers = useRef<Record<string, ReturnType<typeof setInterval>>>({})` for per-session countdown intervals.

**handleCloseTab cleanup pattern** (lines 384-412, App.tsx) — all cleanup state keys removed by session ID:
```typescript
const handleCloseTab = useCallback(async (id: string) => {
  // ... web serving disable ...
  try {
    await KillSession(id)
  } catch (err) {
    console.warn('[App] KillSession failed:', err)
  }
  setTabs((prev) => { /* remove tab, activate adjacent */ })
  setSessionStatuses((prev) => { const n = { ...prev }; delete n[id]; return n })
  setFontSizes((prev) => { const n = { ...prev }; delete n[id]; return n })
  setQrSessionId((prev) => (prev === id ? null : prev))
}, [activeId, webEnabled])
```
Add: `delete n[id]` on `sessionExits` state in this same handler.

**Wails event emit pattern** (app.go lines 213-218):
```go
if a.ctx != nil && a.ctx.Value("frontend") != nil {
    runtime.EventsEmit(a.ctx, "session:status", map[string]string{
        "sessionId": sessionID,
        "status":    s,
    })
}
```

---

### `frontend/src/components/TabBar.tsx` (modify — add countdown indicator + exiting class)

**Analog:** self — existing `.tab__status` span pattern (lines 104-107, TabBar.tsx):
```typescript
<span
  className={`tab__status tab__status--${sessionStatuses?.[tab.sessionId] || 'running'}`}
  title={sessionStatuses?.[tab.sessionId] || 'running'}
/>
```

**Tab class composition pattern** (lines 100-103, TabBar.tsx):
```typescript
<div
  key={tab.id}
  className={`tab${tab.id === activeId ? ' tab--active' : ''}`}
  onClick={() => onSelect(tab.id)}
>
```
Add `${exitingIds?.has(tab.sessionId) ? ' tab--exiting' : ''}` to class string.

**New prop pattern** — add `exitCountdowns?: Record<string, number>` to `TabBarProps` interface (line 11), alongside existing `sessionStatuses?: Record<string, string>`.

---

### `frontend/src/style.css` (modify — add toast + tab countdown CSS)

**Analog:** self — `.banner-exit` animation pattern (lines 1571-1577) and `.update-banner` structure (lines 993-1056).

**Banner-exit animation pattern** (lines 1571-1577, style.css):
```css
.banner-exit {
  opacity: 0;
  max-height: 0;
  overflow: hidden;
  padding-top: 0;
  padding-bottom: 0;
  transition: opacity 150ms ease, max-height 200ms ease, padding 200ms ease;
}
```

**Tab transition pattern** (lines 112-113, style.css — existing `.tab` block):
```css
transition: background-color 0.1s;
```
Add `.tab--exiting { opacity: 0.5; transition: opacity 150ms ease; }` adjacent to `.tab--active`.

**Tab status badge pattern** (lines 880-897, style.css):
```css
.tab__status {
  width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0;
}
.tab__status--running { background: #9ece6a; }
.tab__status--idle    { background: #7aa2f7; }
.tab__status--waiting { background: #e0af68; }
.tab__status--errored { background: #f7768e; }
```
Add `.tab__countdown` adjacent as a text badge (font-size: 11px, tabular-nums, color: #9ece6a).

**Update-banner button pattern** (lines 1026-1056) — use for `.exit-toast__keep-open` (primary) and `.exit-toast__dismiss` (secondary) button styles.

---

### `internal/daemon/engine.go` (modify — add AutoCloseSession setting + natural exit detection)

**Analog:** self — `startMinimized` field pattern (lines 35, 108-113, 119-122, 331-343).

**daemonSettings struct pattern** (lines 64-68, engine.go):
```go
type daemonSettings struct {
    CLIPaths       map[string]string `json:"cliPaths,omitempty"`
    StartMinimized bool              `json:"startMinimized,omitempty"`
}
```
Add: `AutoCloseSession *bool `json:"autoCloseSession,omitempty"`` — use pointer to distinguish absent (default true) from explicit false.

**loadSettingsFromDisk pattern** (lines 83-114, engine.go):
```go
func (e *SessionEngine) loadSettingsFromDisk(dir string) {
    data, err := os.ReadFile(settingsPath(dir))
    if err != nil { return }
    var s daemonSettings
    if json.Unmarshal(data, &s) != nil { return }
    e.mu.Lock()
    // ... populate e.cliPaths ...
    e.startMinimized = s.StartMinimized
    e.mu.Unlock()
}
```

**saveSettingsToDisk pattern** (lines 118-128, engine.go):
```go
func (e *SessionEngine) saveSettingsToDisk() {
    s := daemonSettings{
        CLIPaths:       e.cliPaths,
        StartMinimized: e.startMinimized,
    }
    data, err := json.Marshal(s)
    if err != nil { return }
    _ = os.WriteFile(settingsPath(e.configDir), data, 0600)
}
```

**GetStartMinimized / SetStartMinimized getter/setter pattern** (lines 331-343, engine.go):
```go
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
```

**status.Watch goroutine spawn pattern** (lines 195-203, engine.go) — for the natural exit watcher goroutine:
```go
go status.Watch(hub, id, cli, func(sid string, s status.SessionStatus) {
    e.statusMu.Lock()
    e.sessionStatuses[sid] = s
    e.statusMu.Unlock()
    if onStatus != nil {
        onStatus(sid, s)
    }
})
```
Mirror structure for the exit watcher: `go func() { <-hub.Done(); /* capture exit code, call onExit */ }()`.

**hub.Done() signal** — from `internal/status/detector.go` lines 231-232:
```go
case <-hub.Done():
    return
```
`hub.Done()` is a `<-chan struct{}` that closes when PTY Read returns EOF. This is the correct exit signal (D-07).

---

### `internal/daemon/types.go` (modify — add ExitCode to SessionInfo)

**Analog:** self — existing `SessionInfo` struct (lines 4-14, types.go):
```go
type SessionInfo struct {
    ID          string `json:"id"`
    CLI         string `json:"cli"`
    Name        string `json:"name"`
    State       string `json:"state"`
    Status      string `json:"status"`
    CreatedAt   string `json:"createdAt"`
    Hostname    string `json:"hostname"`
    WebEnabled  bool   `json:"webEnabled"`
    ViewerCount int    `json:"viewerCount"`
}
```
Add: `ExitCode *int `json:"exitCode,omitempty"`` — nil when running, integer when stopped.

---

### `internal/daemon/api.go` (modify — add auto-close-session routes)

**Analog:** self — `handleGetStartMinimized` / `handleSetStartMinimized` pair (lines 343-357, api.go):
```go
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
```

**Route registration pattern** (lines 53-54, api.go):
```go
a.mux.HandleFunc("GET /settings/start-minimized", a.handleGetStartMinimized)
a.mux.HandleFunc("PATCH /settings/start-minimized", a.handleSetStartMinimized)
```
Add:
```go
a.mux.HandleFunc("GET /settings/auto-close-session", a.handleGetAutoCloseSession)
a.mux.HandleFunc("PATCH /settings/auto-close-session", a.handleSetAutoCloseSession)
```

---

### `internal/daemon/client.go` (modify — add GetAutoCloseSession / SetAutoCloseSession)

**Analog:** self — `GetStartMinimized` / `SetStartMinimized` pair (lines 111-124, client.go):
```go
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

**doJSON helper** (lines 214-250, client.go) — shared by all client methods. No changes needed; new methods use it directly.

---

### `app.go` (modify — add GetAutoCloseSession / SetAutoCloseSession Wails bindings + pollSessionStatus exit detection)

**Analog:** self — `GetStartMinimized` / `SetStartMinimized` Wails-bound pair (lines 329-346, app.go):
```go
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
        return fmt.Errorf("daemon not connected")
    }
    return a.client.SetStartMinimized(val)
}
```

**pollSessionStatus / EventsEmit pattern** (lines 202-225, app.go):
```go
func (a *App) pollSessionStatus(sessionID string) {
    var last string
    deadline := time.Now().Add(60 * time.Second)
    for time.Now().Before(deadline) {
        s, err := a.client.GetSessionStatus(sessionID)
        if err != nil {
            return
        }
        if s != last {
            last = s
            if a.ctx != nil && a.ctx.Value("frontend") != nil {
                runtime.EventsEmit(a.ctx, "session:status", map[string]string{
                    "sessionId": sessionID,
                    "status":    s,
                })
            }
            switch s {
            case string(status.StatusErrored), string(status.StatusRunning):
                return
            }
        }
        time.Sleep(500 * time.Millisecond)
    }
}
```

**KillSession EventsEmit pattern** (lines 259-276, app.go) — the `session:exit` event follows this exact shape:
```go
if a.ctx != nil && a.ctx.Value("frontend") != nil {
    runtime.EventsEmit(a.ctx, "session:status", map[string]string{
        "sessionId": id,
        "status":    string(status.StatusErrored),
    })
}
```

**startTrayPoller goroutine+ticker pattern** (lines 590-605, app.go) — for any background poller goroutine added to app.go:
```go
go func() {
    a.refreshTrayState()
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            a.refreshTrayState()
        case <-ctx.Done():
            return
        }
    }
}()
```

---

## Shared Patterns

### Wails Event Subscription (Frontend)
**Source:** `frontend/src/App.tsx` lines 240-244
**Apply to:** App.tsx new `session:exit` subscription
```typescript
const offStatus = EventsOn(
  'session:status',
  (data: { sessionId: string; status: string }) => {
    setSessionStatuses((prev) => ({ ...prev, [data.sessionId]: data.status }))
  },
)
// Return offStatus() in useEffect cleanup
```

### Wails Event Emission (Backend)
**Source:** `app.go` lines 212-217
**Apply to:** `app.go` new `emitExitEvent` helper
```go
if a.ctx != nil && a.ctx.Value("frontend") != nil {
    runtime.EventsEmit(a.ctx, "session:status", map[string]string{
        "sessionId": sessionID,
        "status":    s,
    })
}
```

### Banner Exit Animation
**Source:** `frontend/src/style.css` lines 1571-1577
**Apply to:** ExitToast, ExitCountdownBanner CSS classes
```css
.banner-exit {
  opacity: 0;
  max-height: 0;
  overflow: hidden;
  padding-top: 0;
  padding-bottom: 0;
  transition: opacity 150ms ease, max-height 200ms ease, padding 200ms ease;
}
```

### Settings Boolean Persist Pattern (Go)
**Source:** `internal/daemon/engine.go` lines 64-68, 108-109, 119-122, 331-343
**Apply to:** `AutoCloseSession` field in `daemonSettings`, engine getter/setter
- Use `*bool` pointer (not bare `bool`) to distinguish absent-from-file (default true) from stored-false.
- Getter returns `true` when pointer is nil (absent = enabled).
- Setter stores pointer, saves to disk under e.mu.Lock().

### Settings HTTP Route Pattern (Go)
**Source:** `internal/daemon/api.go` lines 343-357
**Apply to:** `handleGetAutoCloseSession` / `handleSetAutoCloseSession`
- GET returns `writeJSON(w, http.StatusOK, map[string]bool{...})`
- PATCH decodes body into anonymous struct, calls engine setter, returns `w.WriteHeader(http.StatusNoContent)`

### Settings Client Pattern (Go)
**Source:** `internal/daemon/client.go` lines 111-124
**Apply to:** `GetAutoCloseSession` / `SetAutoCloseSession` on DaemonClient
- GET: `doJSON(http.MethodGet, "/settings/auto-close-session", nil, &resp)` → `resp["autoCloseSession"]`
- PATCH: `doJSON(http.MethodPatch, "/settings/auto-close-session", map[string]bool{...}, nil)`

### Vitest Test Scaffold
**Source:** `frontend/src/components/__tests__/UpdateBanner.test.tsx` lines 1-30
**Apply to:** ExitToast.test.tsx, ExitCountdownBanner.test.tsx
```typescript
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'

// render helper → createRoot + flushSync
// afterEach → root?.unmount(); container?.remove(); vi.clearAllMocks()
// interactions → flushSync(() => { btn.click() })
```

### Source Inspection Test Pattern
**Source:** `frontend/src/components/__tests__/App.wiring.test.tsx` lines 1-12
**Apply to:** App.exit.test.tsx
```typescript
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'
// expect(raw).toContain('session:exit')
// expect(raw).toContain('sessionExits')
// etc.
```

---

## No Analog Found

All files have close analogs in the existing codebase. No files require falling back to RESEARCH.md patterns alone.

---

## Metadata

**Analog search scope:** `frontend/src/`, `internal/daemon/`, `internal/status/`, `app.go`
**Files read:** 14 source files + 2 test files
**Pattern extraction date:** 2026-04-19
