# Phase 98: Progress Addon (P2 — Cuttable) — Pattern Map

**Mapped:** 2026-05-08
**Files analyzed:** 19 (new + modified)
**Analogs found:** 19 / 19 (all paths have a strong precedent in Phases 92–97)

## Overview

Every Phase 98 artifact has a precedent in an already-shipping addon phase (93/94/95/96/97). The dominant analog is **Phase 97 (SerializeAddon)** — same shape: vendored addon, hot-swap useEffect arm, App-level registry keyed by sessionId, italic caption under the toggle row, OFF-path negative regression test. Two NEW shapes Phase 97 didn't have:

1. **Cross-tier RPC for tray icon byte swap** (PRG-03) — extends `(*App).updateTray` infrastructure already shipped in Phase 82 via a new `(*App).SetTrayProgress(quartile int) error` RPC. The closest cross-tier RPC analog is **Phase 97 `(*App).SaveTerminalSession`** (single Wails RPC that fans out to file I/O); the closest tray-byte-swap analog is **the existing `connected → trayIconBytes / trayIconErrorBytes` selector in `tray.go:89-122`**.
2. **Per-tab visual indicator on `TabBar.tsx`** — the `.tab__progress` underline parallels the existing `.tab__status` colored dot (lines 881-900 of `style.css`) and the `.tab--active` border-bottom (line 126).

No file in Phase 98 is genuinely without a precedent.

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/TerminalPanel.tsx` (extend) | component | event-driven | `frontend/src/components/TerminalPanel.tsx` (existing Phase 97 SER-01 hot-swap arm at lines 518-538) | exact (self-extension) |
| `frontend/src/components/TabBar.tsx` (extend) | component | request-response (props in, DOM out) | `frontend/src/components/TabBar.tsx` lines 105-145 (existing `.tab__status` per-tab indicator) | exact (self-extension) |
| `frontend/src/components/PluginsSection.tsx` (extend) | component | request-response | `frontend/src/components/PluginsSection.tsx` lines 138-140 (Serialize toggle with italic caption) | exact (self-extension) |
| `frontend/src/App.tsx` (extend) | component | event-driven (registry) | `frontend/src/App.tsx` lines 100-208 (Phase 97 saver registry + handleRegisterSaver/handleRequestSave) | exact (self-extension) |
| `frontend/src/lib/aggregateProgress.ts` | utility | transform | (none — first pure stats helper in `frontend/src/lib/`); shape parallels `frontend/src/lib/openLink.ts` | role-match |
| `frontend/src/lib/__tests__/aggregateProgress.test.ts` | test | transform | `frontend/src/lib/__tests__/openLink.test.ts` (similar pure-helper vitest pattern) | role-match |
| `frontend/src/components/__tests__/TerminalPanel.test.tsx` (extend) | test | event-driven | existing TerminalPanel.test.tsx serialize-arm cases (Phase 97) | exact (self-extension) |
| `frontend/src/components/__tests__/TabBar.test.tsx` (extend) | test | DOM render | existing TabBar.test.tsx (Phase 97 save-menu cases) | exact (self-extension) |
| `frontend/src/components/__tests__/PluginsSection.test.tsx` (extend) | test | DOM render | existing PluginsSection.test.tsx (Phase 96/97 italic-caption cases) | exact (self-extension) |
| `internal/release/no_progress_when_off_test.go` | test | static-grep regression | `internal/release/no_autosave_test.go` (Phase 97 SER-03 pattern, 195 lines, three tests) | exact |
| `web/vendor/xterm/addons/addon-progress.js` | static asset | file I/O (build-time copy) | `web/vendor/xterm/addons/addon-serialize.js` (vendored UMD copy) | exact |
| `web/vendor/xterm/VERSION` (extend) | config | append-only | existing 9-line manifest | exact (self-extension) |
| `web/embed.go` (extend) | config | compile-time embed | `web/embed.go` line 11 (existing addon-serialize embed) | exact (self-extension) |
| `web/terminal.html` (extend) | config | static HTML | `web/terminal.html` line 51 (existing addon-serialize.js script tag) | exact (self-extension) |
| `web/assets/terminal.js` (extend) | component | event-driven (IIFE) | `web/assets/terminal.js` lines 259-272 (Phase 97 SerializeAddon construction) | exact (self-extension) |
| `internal/webserver/vendor_drift_test.go` (extend) | test | static parse | `internal/webserver/vendor_drift_test.go` line 34 (current min-count guard = 9) | exact (self-extension) |
| `app.go` (extend) | controller (Wails RPC) | request-response | `app.go` lines 839-871 (Phase 97 `SaveTerminalSession`) + lines 921-959 (`startTrayPoller` + `refreshTrayState` + `updateTray`) | exact (self-extension) |
| `tray.go` / `tray_linux.go` / `tray_windows.go` (extend) | platform middleware | request-response | existing `connected ? trayIconBytes : trayIconErrorBytes` selector at `tray.go:89-122`, `tray_linux.go:404-430`, `tray_windows.go:507-539` | exact (self-extension) |
| `assets/tray_icon_progress_{25,50,75,100}.png` | static asset | file I/O (compile-time embed) | `assets/tray_icon.png` + `assets/tray_icon_error.png` (existing 18×18 base + error icons) | exact |
| `frontend/src/wailsjs/go/main/App.{d.ts,js}` (extend, generated) | binding | RPC stub | `App.d.ts` line 146 + `App.js` line 89 (existing `SaveTerminalSession` stub) | exact (self-extension) |
| `frontend/src/style.css` (extend) | config (CSS) | DOM render | `frontend/src/style.css` lines 123-127 (`.tab--active` border-bottom) + lines 881-900 (`.tab__status` indicator) | exact (self-extension) |
| `frontend/e2e/progress.spec.ts` | test (e2e) | DOM event-driven | `frontend/e2e/web-links-live-toggle.spec.ts` (74-line Playwright test.skip scaffold) | exact |

---

## Pattern Assignments

### 1. `frontend/src/components/TerminalPanel.tsx` (extend) — Hot-Swap Addon Arm

**Analog:** `frontend/src/components/TerminalPanel.tsx` lines 518-538 (Phase 97 Serialize hot-swap)
**Differs how:** New arm gates on `pluginConfig?.progress` instead of `pluginConfig?.serialize`; instead of registering a saver closure (`onRegisterSaver(sessionId, () => string)`), it subscribes to `progressAddon.onChange` and forwards `IProgressState` events via `onProgressChange(sessionId, state)`. On detach, must explicitly emit `{state:0, value:0}` to clear the registry (Pitfall #7 stuck-progress) — Phase 97 had no equivalent because `null` saver is the natural cleanup; here the registry uses `state:0` as the delete signal.

**Imports pattern** (lines 1-21 — add one):
```typescript
import { SerializeAddon } from '@xterm/addon-serialize'
// ADD:
import { ProgressAddon, type IProgressState } from '@xterm/addon-progress'
```

**Ref declaration pattern** (line 121, mirror):
```typescript
// Phase 97 SER-01: SerializeAddon ref. Construction is in the HOT-SWAP
// useEffect (NOT mount) — Serialize is a pure buffer-walker with no
// buffer-state implications, so it can be attached/detached at runtime.
const serializeAddonRef = useRef<SerializeAddon | null>(null)
// ADD parallel:
const progressAddonRef = useRef<ProgressAddon | null>(null)
const progressOnChangeDisposable = useRef<{ dispose(): void } | null>(null)
```

**Hot-swap arm pattern** (lines 524-537 — copy structure):
```typescript
if (pluginConfig?.serialize) {
  if (!serializeAddonRef.current) {
    const serializeAddon = new SerializeAddon()
    term.loadAddon(serializeAddon)
    serializeAddonRef.current = serializeAddon
    onRegisterSaver?.(sessionId, () => serializeAddon.serialize({ excludeModes: true }))
  }
} else {
  if (serializeAddonRef.current) {
    serializeAddonRef.current.dispose()
    serializeAddonRef.current = null
    onRegisterSaver?.(sessionId, null) // Pitfall #6 — flush stale closure
  }
}
```

**Dep-array pattern** (line 538 — must add `pluginConfig?.progress` and `onProgressChange`):
```typescript
}, [pluginConfig?.webgl, pluginConfig?.clipboard, pluginConfig?.search,
    pluginConfig?.webLinks, pluginConfig?.serialize, onWebGLContextLost,
    onRegisterSaver, sessionId])
// ADD: pluginConfig?.progress, onProgressChange
```

**Mount-cleanup pattern** (lines 314-321 — mirror):
```typescript
// Phase 97 SER-01: dispose serializeAddon AND flush the saver registry
// entry on unmount (Pitfall #6 — leaving a stale closure behind would
// mean handleRequestSave invokes a disposed addon).
if (serializeAddonRef.current) {
  serializeAddonRef.current.dispose()
  serializeAddonRef.current = null
}
onRegisterSaver?.(sessionId, null)
// ADD parallel for progressAddonRef + progressOnChangeDisposable + emit {state:0,value:0}.
```

---

### 2. `frontend/src/App.tsx` (extend) — Cross-Session Registry + Debounced RPC

**Analog:** `frontend/src/App.tsx` lines 100-208 (Phase 97 saver registry)
**Differs how:** Saver registry stored a `() => string` closure per session; progress registry stores a struct `IProgressState` per session. Saver was consumed on-demand from a context menu; progress is consumed on every event to compute an aggregate, so it lives in a `useRef<Map<...>>` (mutable, no re-render) plus a `useState<Record<sessionId, number>>` for the per-tab UI prop. New addition: a 200ms `setTimeout` debounce (Pattern 4 in RESEARCH) before the Wails RPC `SetTrayProgress(quartile)` — no Phase 97 equivalent.

**Saver registry pattern** (lines 100-106, mirror shape):
```typescript
// Phase 97 SER-01: saver registry. TerminalPanel registers a closure
// that returns the addon's serialize() output keyed by sessionId; App
// uses it when handleRequestSave fires from the TabBar context menu.
// Cleared on unmount via TerminalPanel's useEffect cleanup (Pitfall #6).
const [serializerRegistry, setSerializerRegistry] = useState<
  Record<string, (() => string) | null>
>({})
```

**Saver register-callback pattern** (lines 169-177):
```typescript
const handleRegisterSaver = useCallback(
  (sessionId: string, fn: (() => string) | null) => {
    setSerializerRegistry((prev) => ({ ...prev, [sessionId]: fn }))
  },
  []
)
```

**Progress equivalent (NEW, modeled on the above):**
```typescript
// Phase 98 PRG-02/PRG-03 — progress registry mirrors Phase 97 saver registry.
// useRef Map (mutable, no re-render on .set/.delete) for aggregation source;
// useState Record for the per-tab UI prop (drives <TabBar tabProgress=...>).
const progressRegistry = useRef(new Map<string, IProgressState>())
const [tabProgress, setTabProgress] = useState<Record<string, number>>({})
const trayDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
const lastDispatchedQuartileRef = useRef<number>(-1)

const handleProgressChange = useCallback(
  (sessionId: string, state: IProgressState) => {
    if (state.state === 1) {
      progressRegistry.current.set(sessionId, state)
      setTabProgress((prev) => ({ ...prev, [sessionId]: state.value }))
    } else {
      progressRegistry.current.delete(sessionId)
      setTabProgress((prev) => {
        const { [sessionId]: _, ...rest } = prev
        return rest
      })
    }
    const quartile = aggregateProgress(progressRegistry.current)
    if (trayDebounceRef.current) clearTimeout(trayDebounceRef.current)
    trayDebounceRef.current = setTimeout(() => {
      if (lastDispatchedQuartileRef.current === quartile) return
      lastDispatchedQuartileRef.current = quartile
      void SetTrayProgress(quartile)
    }, 200)
  },
  []
)
```

**TerminalPanel prop wiring pattern** (lines 995-1004 — mirror `onRegisterSaver`):
```typescript
<TerminalPanel
  sessionId={tab.sessionId}
  isActive={isActive}
  /* ... */
  onRegisterSaver={handleRegisterSaver}
  /* ADD: */
  onProgressChange={handleProgressChange}
/>
```

**TabBar prop wiring pattern** (lines 888-903 — add `tabProgress`):
```typescript
<TabBar
  tabs={tabs}
  activeId={activeId}
  /* ... */
  sessionStatuses={sessionStatuses}
  /* ADD: */
  tabProgress={tabProgress}
/>
```

---

### 3. `frontend/src/components/TabBar.tsx` (extend) — Per-Tab Progress Underline

**Analog:** `frontend/src/components/TabBar.tsx` lines 105-145 (existing per-tab `.tab__status` colored dot + tab class composition)
**Differs how:** `.tab__status` is a fixed-width colored dot read from `sessionStatuses[sessionId]`; `.tab__progress` is a full-width underline scaled by `transform: scaleX(value/100)` based on `tabProgress[sessionId]`. The existing `tab--exiting` modifier shows the conditional-class precedent; Phase 98 doesn't add a class modifier — the underline is unconditionally rendered with `scaleX(0)` when no progress (so a single CSS transition handles enter/exit smoothly).

**Tab render pattern** (lines 108-156 — add new `<div>` before `.tab__close` button):
```tsx
<div
  key={tab.id}
  className={`tab${tab.id === activeId ? ' tab--active' : ''}${exitCountdowns?.[tab.sessionId] ? ' tab--exiting' : ''}`}
  onClick={() => onSelect(tab.id)}
>
  <span
    className={`tab__status tab__status--${sessionStatuses?.[tab.sessionId] || 'running'}`}
    title={sessionStatuses?.[tab.sessionId] || 'running'}
  />
  {/* ... .tab__name ... .tab__countdown ... .tab__close ... */}
  {/* ADD before closing </div>: */}
  <div
    className="tab__progress"
    style={{ transform: `scaleX(${(tabProgress?.[tab.sessionId] ?? 0) / 100})` }}
    data-testid={`tab-progress-${tab.id}`}
  />
</div>
```

**Props extension pattern** (lines 11-26 — add `tabProgress`):
```typescript
interface TabBarProps {
  tabs: Tab[]
  activeId: string | null
  /* ... */
  sessionStatuses?: Record<string, string>
  exitCountdowns?: Record<string, number>
  /* ADD: */
  tabProgress?: Record<string, number>
}
```

---

### 4. `frontend/src/components/PluginsSection.tsx` (extend) — Italic v3.3-Flip Caption

**Analog:** `frontend/src/components/PluginsSection.tsx` lines 138-140 (Serialize toggle with secrets caption)
**Differs how:** Existing `renderRow` already supports an optional `caption` arg that renders with `settings-panel__description--italic` class. The Progress toggle row already exists at lines 143-144 *without* a caption; the change is to add the v3.3-flip caption string as the third arg.

**Existing helper signature** (line 78-83):
```typescript
function renderRow(
  key: PluginBooleanKey,
  label: string,
  description: string,
  caption?: string,
): React.ReactElement {
```

**Existing italic caption call site** (lines 138-140 — the precedent):
```typescript
{renderRow('serialize', 'Save terminal as text',
  'Right-click a tab to export the visible scrollback as a text file.',
  'Saved files include any secrets, tokens, or sensitive data printed in the session.')}
```

**Existing Progress row** (lines 143-144 — needs caption added):
```typescript
{renderRow('progress', 'Progress (OSC 9;4)',
  'Show a per-tab progress underline when the running CLI emits OSC 9;4 progress updates.')}
// ADD third arg: 'Default OFF in v3.2 — flips ON in v3.3 after field validation.'
```

**Italic CSS class application** (line 108-110 of PluginsSection.tsx — already present, no CSS change needed):
```typescript
{caption && (
  <p className="settings-panel__description settings-panel__description--italic">
    {caption}
  </p>
)}
```

---

### 5. `frontend/src/lib/aggregateProgress.ts` (NEW) — Pure Stats Helper

**Analog:** `frontend/src/lib/openLink.ts` (other pure helper in lib/, single exported function, vitest-tested)
**Differs how:** `openLink` invokes `BrowserOpenURL` with platform branching; `aggregateProgress` is pure math with no side effects — even simpler. Closest *shape* is the `aggregateProgress` excerpt embedded directly in 98-RESEARCH.md §"Pattern 3" — copy verbatim.

**Verbatim shape (RESEARCH Pattern 3, lines 514-532):**
```typescript
import type { IProgressState } from '@xterm/addon-progress'

export function aggregateProgress(
  registry: Map<string, IProgressState>
): 0 | 1 | 2 | 3 | 4 {
  const values: number[] = []
  for (const s of registry.values()) {
    if (s.state === 1) values.push(s.value)
  }
  if (values.length === 0) return 0
  const mean = values.reduce((a, b) => a + b, 0) / values.length
  if (mean <= 0) return 0
  if (mean <= 25) return 1
  if (mean <= 50) return 2
  if (mean <= 75) return 3
  return 4
}
```

---

### 6. `frontend/src/style.css` (extend) — `.tab__progress` Rule

**Analog:** `frontend/src/style.css` lines 123-127 (`.tab--active` border-bottom — same TokyoNight #7aa2f7 accent) + lines 881-900 (`.tab__status` per-tab indicator)
**Differs how:** `.tab--active` is a static 2px border-bottom; `.tab__progress` is a scaled-via-transform underline that uses the same color but is positioned absolutely so multiple tabs can have it simultaneously. `.tab` must gain `position: relative` for the absolute positioning to anchor (no existing rule sets this — Pattern 5 in RESEARCH calls it out).

**Existing accent precedent** (lines 123-127):
```css
.tab--active {
  background-color: #1a1b26;
  color: #c0caf5;
  border-bottom: 2px solid #7aa2f7;
}
```

**New rules (verbatim from RESEARCH Pattern 5, lines 575-593):**
```css
.tab {
  /* ... existing rules ... */
  position: relative; /* ADD — anchor for .tab__progress */
}

.tab__progress {
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 2px;
  background: #7aa2f7; /* TokyoNight accent — same as .tab--active border-bottom */
  transform: scaleX(0);
  transform-origin: left;
  transition: transform 200ms ease-out;
  pointer-events: none;
}
```

---

### 7. `app.go` (extend) — `SetTrayProgress` Wails RPC + State Field

**Analog:** `app.go` lines 839-871 (Phase 97 `SaveTerminalSession` — single Wails RPC method with input validation, error wrapping); `app.go` lines 941-959 (`refreshTrayState` — existing tray-state recompute entry point)
**Differs how:** `SaveTerminalSession` performs file I/O via Wails dialog runtime; `SetTrayProgress` mutates a Go-side state field (`a.lastTrayQuartile`) and calls `refreshTrayState()` to fan out to the platform-specific `updateTray`. Idempotency check (Pitfall #3 transition guard) — return early if quartile unchanged. New struct field `lastTrayQuartile int` on `App` mirrors the existing `trayInit bool` field at line 58.

**Existing RPC method shape** (lines 846-871):
```go
func (a *App) SaveTerminalSession(defaultDir, defaultName, content string) error {
    if defaultDir == "" {
        if home, err := os.UserHomeDir(); err == nil {
            defaultDir = home
        }
    }
    path, err := a.saveFileDialogFunc(a.ctx, runtime.SaveDialogOptions{ /* ... */ })
    if err != nil {
        return fmt.Errorf("SaveTerminalSession: dialog: %w", err)
    }
    if path == "" {
        return nil
    }
    if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
        return fmt.Errorf("SaveTerminalSession: write: %w", err)
    }
    return nil
}
```

**Existing tray-state struct field** (line 58):
```go
type App struct {
    ctx       context.Context
    client    *daemon.DaemonClient
    trayInit  bool                 // true once initTray has been called
    /* ... */
    // ADD: lastTrayQuartile int   // Phase 98 PRG-03 — last applied tray progress quartile [0..4], -1 = unset
}
```

**Existing tray-refresh entry point** (lines 943-959):
```go
func (a *App) refreshTrayState() {
    if !a.trayInit {
        return
    }
    if a.client == nil {
        a.updateTray(nil, false)
        return
    }
    connected := a.client.Health() == nil
    var sessions []SessionInfo
    if connected {
        sessions = a.ListSessions()
    }
    a.updateTray(sessions, connected)
}
```

**New method (RESEARCH Pattern 6 — verbatim shape, lines 600-614):**
```go
func (a *App) SetTrayProgress(quartile int) error {
    if !a.trayInit {
        return nil
    }
    if quartile < 0 || quartile > 4 {
        return fmt.Errorf("SetTrayProgress: quartile out of range [0,4]: %d", quartile)
    }
    if a.lastTrayQuartile == quartile {
        return nil
    }
    a.lastTrayQuartile = quartile
    a.refreshTrayState()
    return nil
}
```

---

### 8. `tray.go` / `tray_linux.go` / `tray_windows.go` (extend) — Quartile Byte Selector

**Analog:** `tray.go` lines 89-122 (existing darwin `updateTray` with `connected` byte branch); `tray_linux.go` lines 404-430 (Linux iconPixmap branch); `tray_windows.go` lines 507-539 (Windows hIcon branch)
**Differs how:** Existing branch is binary (`connected ? trayIconBytes : trayIconErrorBytes`); new logic adds a 4-way branch on `a.lastTrayQuartile` ONLY when `connected=true` (error precedence — Pitfall #8). Add 4 new `//go:embed` directives mirroring lines 29-33 of `tray.go` (`trayIconBytes` + `trayIconErrorBytes`). Windows path requires an extra `createIconFromPNG` call per quartile to build HICON handles up-front (line 392 pattern), Linux requires `makePixmap` per call site, darwin just hands the byte slice to cgo.

**Existing darwin embed pattern** (lines 29-33):
```go
//go:embed assets/tray_icon.png
var trayIconBytes []byte

//go:embed assets/tray_icon_error.png
var trayIconErrorBytes []byte
```

**Existing darwin update branch** (lines 89-97):
```go
func (a *App) updateTray(sessions []SessionInfo, connected bool) {
    if connected {
        ptr := unsafe.Pointer(&trayIconBytes[0])
        C.updateTrayIcon(ptr, C.int(len(trayIconBytes)))
    } else {
        ptr := unsafe.Pointer(&trayIconErrorBytes[0])
        C.updateTrayIcon(ptr, C.int(len(trayIconErrorBytes)))
    }
    /* ... tooltip + session menu ... */
}
```

**Existing Linux update branch** (lines 410-416):
```go
tray.mu.Lock()
if connected {
    tray.iconPixmap = makePixmap(trayIconBytes)
} else {
    tray.iconPixmap = makePixmap(trayIconErrorBytes)
}
```

**Existing Windows update branch** (lines 517-522):
```go
wt.mu.Lock()
defer wt.mu.Unlock()
if connected {
    wt.nid.HIcon = wt.hIcon
} else {
    wt.nid.HIcon = wt.hIconErr
}
```

**Existing Windows HICON pre-creation pattern** (lines 392-406 — replicate per quartile):
```go
wt.hIcon, err = createIconFromPNG(trayIconBytes)
if err != nil {
    log.Printf("system tray: failed to create normal icon: %v; tray icon disabled", err)
    wt.disabled = true
    close(wt.ready)
    return
}
wt.hIconErr, err = createIconFromPNG(trayIconErrorBytes)
```

**New helper (RESEARCH Pattern 6, lines 632-643) to insert into all three files:**
```go
func (a *App) trayIconBytesForState(connected bool) []byte {
    if !connected {
        return trayIconErrorBytes
    }
    switch a.lastTrayQuartile {
    case 1: return trayIconProgress25Bytes
    case 2: return trayIconProgress50Bytes
    case 3: return trayIconProgress75Bytes
    case 4: return trayIconProgress100Bytes
    default: return trayIconBytes
    }
}
```

Then existing `updateTray` body changes one line: `bytes := a.trayIconBytesForState(connected)` and feed `bytes` into `C.updateTrayIcon` / `makePixmap` / `wt.nid.HIcon` (Windows needs HICON cache lookup — the 4 new HICONs created at `initTray` time).

---

### 9. `web/embed.go` (extend) — `//go:embed` Directive

**Analog:** `web/embed.go` line 11 (existing `vendor/xterm/addons/addon-image.js vendor/xterm/addons/addon-serialize.js`)
**Differs how:** Append `vendor/xterm/addons/addon-progress.js` to the same `//go:embed` directive that already lists `addon-image.js` and `addon-serialize.js`.

**Existing embed pattern** (full file, 12 lines):
```go
package web

import "embed"

//go:embed dashboard.html terminal.html join.html
//go:embed assets/terminal.js assets/terminal.css
//go:embed assets/dashboard.js assets/dashboard.css
//go:embed assets/join.js assets/join.css
//go:embed vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION
//go:embed vendor/xterm/addons/addon-webgl.js vendor/xterm/addons/addon-unicode11.js vendor/xterm/addons/addon-clipboard.js vendor/xterm/addons/addon-search.js vendor/xterm/addons/addon-web-links.js
//go:embed vendor/xterm/addons/addon-image.js vendor/xterm/addons/addon-serialize.js
var WebFS embed.FS
```

**Change:** Append ` vendor/xterm/addons/addon-progress.js` to the line 11 directive (or add a new `//go:embed` line — both are equivalent per Go spec).

---

### 10. `web/terminal.html` (extend) — Script Tag Include

**Analog:** `web/terminal.html` line 51 (existing `<script src="/assets/xterm/addons/addon-serialize.js"></script>`)
**Differs how:** Add a new `<script>` line after addon-serialize. Note that `web/embed.go` mounts `web/vendor/xterm/` at the URL prefix `/assets/xterm/` (verified by lines 43-51 — the served URL is `/assets/xterm/...` while the source path is `web/vendor/xterm/...`).

**Existing script-tag include pattern** (lines 43-51):
```html
<script src="/assets/xterm/xterm.js"></script>
<script src="/assets/xterm/addon-fit.js"></script>
<script src="/assets/xterm/addons/addon-webgl.js"></script>
<script src="/assets/xterm/addons/addon-unicode11.js"></script>
<script src="/assets/xterm/addons/addon-clipboard.js"></script>
<script src="/assets/xterm/addons/addon-search.js"></script>
<script src="/assets/xterm/addons/addon-web-links.js"></script>
<script src="/assets/xterm/addons/addon-image.js"></script>
<script src="/assets/xterm/addons/addon-serialize.js"></script>
<!-- ADD: <script src="/assets/xterm/addons/addon-progress.js"></script> -->
```

---

### 11. `web/assets/terminal.js` (extend) — IIFE Web-Parity Construction

**Analog:** `web/assets/terminal.js` lines 259-272 (Phase 97 SerializeAddon construction)
**Differs how:** Serialize addon was vendored-but-inert (no UI consumer on web). Progress addon is vendored AND used: subscribe to `onChange`, mutate a thin `<div id="progress-underline">` at the top of the page (web has no tab strip). UMD global is `ProgressAddon.ProgressAddon` (mirrors `SerializeAddon.SerializeAddon` and `ImageAddon.ImageAddon` patterns).

**Existing serialize construction pattern** (lines 267-272):
```javascript
if (pluginConfig.serialize) {
  try {
    var serializeAddon = new SerializeAddon.SerializeAddon();
    term.loadAddon(serializeAddon);
  } catch (e) { /* addon UMD may not be present — silent */ }
}
```

**New progress construction (RESEARCH §"Web parity construction", lines 849-864):**
```javascript
if (pluginConfig.progress) {
  try {
    var progressAddon = new ProgressAddon.ProgressAddon();
    term.loadAddon(progressAddon);
    progressAddon.onChange(function (state) {
      var bar = document.getElementById('progress-underline');
      if (!bar) return;
      if (state.state === 1) {
        bar.style.transform = 'scaleX(' + (state.value / 100) + ')';
      } else {
        bar.style.transform = 'scaleX(0)';
      }
    });
  } catch (e) { /* addon UMD may not be present — silent */ }
}
```

---

### 12. `internal/webserver/vendor_drift_test.go` (extend) — Min-Count Bump

**Analog:** `internal/webserver/vendor_drift_test.go` line 34 (existing min-count guard `< 9`)
**Differs how:** Trivial bump: `9 → 10`. The regex at line 18 already matches every `@xterm/addon-*` package, so no regex change. Update the error-message string to mention `addon-progress`.

**Existing min-count check** (lines 34-36):
```go
if len(pnpmVersions) < 9 {
    t.Fatalf("failed to parse at least 9 @xterm/* packages (xterm, addon-fit, addon-webgl, addon-unicode11, addon-clipboard, addon-search, addon-web-links, addon-image, addon-serialize) from pnpm-lock.yaml: found %v (Phase 95 SRC-95-06 — addon-web-links joined the manifest; Phase 96 IMG-03 — addon-image joined the manifest; Phase 97 SER-03 — addon-serialize joined the manifest)", pnpmVersions)
}
```

**Change:** `< 9` → `< 10`; append ", addon-progress" to package list and append "; Phase 98 PRG-04 — addon-progress joined the manifest" to provenance trail.

---

### 13. `internal/release/no_progress_when_off_test.go` (NEW) — OFF-Path Negative Regression

**Analog:** `internal/release/no_autosave_test.go` (Phase 97 SER-03, 195 lines, three tests)
**Differs how:** SER-03 tested forbidden auto-save patterns (no scheduled invocation, no auto-save settings field, exactly one `Save*Session*` method in app.go). PRG-OFF tests three things: (a) no `setInterval.*[Pp]rogress` polling (Pitfall #6); (b) every `new ProgressAddon(` is preceded by `pluginConfig?.progress` guard within the same useEffect; (c) every `SetTrayProgress(` callsite is reachable only when `tabProgress` is in scope. Mirror the `filepath.WalkDir` + `regexp.MustCompile` + skip-list scaffold verbatim.

**Existing scaffold pattern** (lines 29-44 — copy structure):
```go
forbidden := []struct {
    re   *regexp.Regexp
    desc string
}{
    {regexp.MustCompile(`setInterval\([^)]*[Ss]eriali[zs]e`), "setInterval scheduling serialize()"},
    {regexp.MustCompile(`setTimeout\([^,]*[Ss]eriali[zs]e[^,]*,\s*[0-9]{4,}`), "setTimeout long-delay scheduled serialize()"},
    /* ... etc ... */
}
```

**Existing skip-list pattern** (lines 47-60):
```go
skipDirs := map[string]bool{
    ".git": true, "node_modules": true,
    "frontend/node_modules": true, "build": true, "dist": true,
    "vendor": true, "internal/release": true,
    ".planning": true, "frontend/src/wailsjs": true,
    "screenshots": true, ".claude": true, ".claire": true,
}
```

**Existing walk + scan pattern** (lines 69-105):
```go
err = filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
    /* skip dirs / extension filter / read file / regex match → record violation */
})
if len(violations) > 0 {
    t.Errorf("SER-03 invariant violated — auto-save patterns detected:\n  %s", strings.Join(violations, "\n  "))
}
```

**New PRG-OFF forbidden patterns:**
```go
{regexp.MustCompile(`setInterval\([^)]*[Pp]rogress`), "setInterval polling progress (Pitfall #6 — addon is event-driven)"},
{regexp.MustCompile(`setTimeout\([^,]*[Pp]rogress[^,]*,\s*[0-9]{4,}`), "setTimeout long-delay progress polling"},
// Optional: assert "new ProgressAddon" callsites are gated. Lighter touch than a full AST walk:
// just confirm any source line containing "new ProgressAddon(" sits in a file that ALSO contains
// "pluginConfig?.progress" or "pluginConfig.progress" within the same file.
```

---

### 14. `frontend/e2e/progress.spec.ts` (NEW) — Playwright E2E Scaffold

**Analog:** `frontend/e2e/web-links-live-toggle.spec.ts` (74-line `test.skip` scaffold)
**Differs how:** web-links spec scaffolds documented walk-throughs but stays skipped pending Playwright pluming for real sessions; progress spec follows the same pattern — scaffolded `test.skip` blocks documenting the OSC 9;4 fixture path. Real test would: (1) navigate to a session URL with `progress: true`, (2) `term.write('\x1b]9;4;1;47\x07')` via page.evaluate, (3) assert `transform: matrix(0.47, ...)` style on `.tab__progress` (or `#progress-underline` for web).

**Existing scaffold shape** (lines 22-40):
```typescript
import { test } from '@playwright/test';

test.describe('Phase 95 — web-links live toggle (LNK-05/SC-5)', () => {
  test.skip(
    'toggle webLinks=false disposes addon; ...',
    async () => {
      // 1. Navigate to a Tailscale-served session URL with webLinks=true.
      // 2. Echo `https://example.com\r` into the terminal via term.write().
      /* ... documented walk ... */
      throw new Error('test.skip — see test body for the documented walk');
    }
  );
});
```

---

### 15. `assets/tray_icon_progress_{25,50,75,100}.png` (NEW) — 18×18 PNGs

**Analog:** `assets/tray_icon.png` + `assets/tray_icon_error.png` (existing 18×18 base + error icons referenced by `//go:embed` in `tray.go:29-33`, `tray_linux.go:18-22`, `tray_windows.go:20-24`)
**Differs how:** Same dimensions, same TokyoNight palette, with a horizontal fill bar at the bottom proportional to the quartile (25/50/75/100% width). Designer-supplied or generated via a `build/gen_progress_icons.go` script (RESEARCH Open Question #1). Either way the four PNGs land in `assets/` to be picked up by the existing per-OS `//go:embed` directives.

**Existing asset path precedent:**
```
assets/tray_icon.png         # 18×18 PNG, base TokyoNight icon
assets/tray_icon_error.png   # 18×18 PNG, red-tinted disconnect icon
```

---

### 16. `frontend/src/wailsjs/go/main/App.{d.ts,js}` (extend, generated) — RPC Stub

**Analog:** `App.d.ts` line 146 + `App.js` line 89 (existing `SaveTerminalSession` RPC stub)
**Differs how:** Wails generates these from the bound Go method signature. After adding `(*App).SetTrayProgress(quartile int) error` to `app.go`, running `wails dev` (or `wails generate module`) regenerates these files. Plan should call out the regeneration step explicitly so reviewers see the diff.

**Existing TS declaration pattern** (line 146):
```typescript
// Save terminal session (Phase 97 SER-01).
export function SaveTerminalSession(defaultDir: string, defaultName: string, content: string): Promise<void>
// ADD: Set tray progress quartile (Phase 98 PRG-03).
//      export function SetTrayProgress(quartile: number): Promise<void>
```

**Existing JS stub pattern** (line 89):
```javascript
// Save terminal session (Phase 97 SER-01).
export const SaveTerminalSession = (defaultDir, defaultName, content) => Call('main.App.SaveTerminalSession', [defaultDir, defaultName, content])
// ADD: export const SetTrayProgress = (quartile) => Call('main.App.SetTrayProgress', [quartile])
```

---

## Shared Patterns

### Hot-Swap Addon Lifecycle (Pitfall #1: dep array specificity)

**Source:** `frontend/src/components/TerminalPanel.tsx` lines 351-538 (single useEffect with specific-key deps; webgl + clipboard + search + webLinks + serialize all branch independently within it)
**Apply to:** All TerminalPanel addon arms; Phase 98 adds the progress arm to this same useEffect.

The dep-array invariant — never include the whole `pluginConfig` object — is established. New deps must be specific keys: `pluginConfig?.progress, onProgressChange`. Mount/unmount cleanup at lines 314-321 is the parallel destruction site for the on-detach `{state:0,value:0}` emit.

### Italic Caption Under Toggle (settings-panel__description--italic)

**Source:** `frontend/src/components/PluginsSection.tsx` lines 78-114 (`renderRow` with optional `caption?: string` 4th arg → `<p className="settings-panel__description settings-panel__description--italic">{caption}</p>`)
**Apply to:** Phase 98 progress toggle row (line 143-144 of PluginsSection — add caption as 3rd arg to `renderRow`). No CSS change needed — class is already styled.

### Vendoring Discipline (WEB-01)

**Source:** Phase 93 introduced; Phase 94/95/96/97 followed verbatim. Pipeline: `pnpm add` → `cp node_modules/@xterm/addon-X/lib/addon-X.js web/vendor/xterm/addons/` → append `@xterm/addon-X@<v>` to `web/vendor/xterm/VERSION` → bump `vendor_drift_test.go` min-count → add to `web/embed.go //go:embed` → add `<script>` to `web/terminal.html` → optionally consume in `web/assets/terminal.js`.
**Apply to:** Phase 98 addon-progress vendoring (every step of the pipeline).

### Cross-Tier RPC (Frontend → Wails → Go)

**Source:** `(*App).SaveTerminalSession` (app.go:846), wired to `frontend/src/wailsjs/go/main/App.{d.ts,js}` auto-generated stubs.
**Apply to:** Phase 98 `(*App).SetTrayProgress`. Same pattern: Go method with input validation + error wrap; auto-generated TS/JS stubs imported and `void`-called from App.tsx.

### Negative Regression Test (release/no_*_test.go)

**Source:** `internal/release/no_autosave_test.go` (Phase 97 SER-03 — three tests in one file: forbidden source-grep, forbidden settings field, exact-method-count assertion)
**Apply to:** Phase 98 `internal/release/no_progress_when_off_test.go`. Mirror the `filepath.WalkDir` + `regexp.MustCompile` + skip-list scaffold verbatim. Difference: PRG-OFF gates check that `new ProgressAddon` is preceded by `pluginConfig?.progress` (the OFF-path-zero-side-effects invariant) rather than ban any pattern outright.

### Tray Icon Byte Selector (cross-platform)

**Source:** `tray.go:89-122`, `tray_linux.go:404-430`, `tray_windows.go:507-539` — existing 2-way branch on `connected` bool maps to platform-specific update path.
**Apply to:** Phase 98 5-way branch on `connected && lastTrayQuartile`. Insert helper `(a *App) trayIconBytesForState(connected bool) []byte` in each platform file (or in a shared `tray_common.go` if Linux/darwin can share — Windows always needs its own HICON pre-cache so probably not worth a shared file). Error precedence (Pitfall #8): `connected=false` always returns error bytes regardless of quartile.

### Frontend Debounce Idiom (200ms)

**Source:** Phase 94 BannerStack 200ms slide animation precedent; Pattern 4 in 98-RESEARCH.md (lines 538-555) — `useRef<setTimeout>` pattern, no lodash dep.
**Apply to:** Phase 98 App.tsx `scheduleSetTrayProgress(quartile)` debounce before `void SetTrayProgress(quartile)` Wails call. Cleanup on unmount: `clearTimeout(trayDebounceRef.current)` in a returned-from-useEffect cleanup function.

### Per-Tab DOM Element (mirror `.tab__status`)

**Source:** `frontend/src/components/TabBar.tsx` lines 114-117 + `frontend/src/style.css` lines 881-900 — the existing `.tab__status` colored dot is keyed by `sessionId` and read from a `Record<string, string>` prop.
**Apply to:** New `.tab__progress` element keyed by `sessionId` and read from `tabProgress?: Record<string, number>` prop. Same render pattern, same prop shape, different child element + CSS rule.

---

## No Analog Found

No file in Phase 98 lacks a precedent. Every artifact has an analog in either Phase 97 (SerializeAddon, the closest), an earlier vendoring phase (93–96), or a long-shipped infrastructure file (Phase 82 tray, Phase 92 plugin settings). Two artifacts come closest to "novel":

| File | Closest Analog | Why It's Still A Match |
|------|----------------|------------------------|
| `frontend/src/lib/aggregateProgress.ts` | `frontend/src/lib/openLink.ts` | Same role (pure helper in `lib/`) and project conventions; the math itself is RESEARCH-supplied verbatim Pattern 3, so there's nothing to invent. |
| `(*App).SetTrayProgress` Wails RPC | `(*App).SaveTerminalSession` (Phase 97) for the RPC plumbing + existing `connected` byte-swap branch in `tray.go:89-97` for the byte selection | The combination is new but each half is a verbatim mirror. |

---

## Metadata

**Analog search scope:** `/Users/ken/dev/agenthub/{frontend/src,internal,web,assets,app.go,tray*.go}`
**Files scanned:** ~30 (all referenced from RESEARCH.md §"Recommended Project Structure" + Sources)
**Planning consumption note:** Every "Pattern Assignments" subsection above gives the planner a concrete file path, line range, and either an excerpt from the existing analog OR a verbatim RESEARCH-supplied excerpt. The planner does not need to re-discover any pattern — copy-paste directly into plan action sections.

**Pattern extraction date:** 2026-05-08
