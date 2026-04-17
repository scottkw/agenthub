# Phase 81: Banner Notifications - Pattern Map

**Mapped:** 2026-04-16
**Files analyzed:** 8 (5 modified, 1 new component, 1 new test, 1 modified test)
**Analogs found:** 8 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/LocalNetworkBanner.tsx` | component | request-response | self (existing) | exact — modify in place |
| `frontend/src/components/UpdateBanner.tsx` | component | event-driven | `WelcomeTab.tsx` §update-banner | exact — extract existing markup |
| `frontend/src/App.tsx` | provider/root | event-driven + request-response | self (existing) | exact — add state + BannerStack |
| `frontend/src/style.css` | config | — | self §local-network-banner | exact — extend existing BEM |
| `frontend/src/components/WelcomeTab.tsx` | component | request-response | self (existing) | exact — remove lifted state |
| `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` | test | — | self (existing) | exact — extend with dismiss tests |
| `frontend/src/components/__tests__/UpdateBanner.test.tsx` | test | — | `LocalNetworkBanner.test.tsx` | exact — same createRoot+flushSync pattern |
| `frontend/src/components/__tests__/App.test.tsx` | test | — | self (existing) | exact — add BannerStack describe block |

---

## Pattern Assignments

### `frontend/src/components/LocalNetworkBanner.tsx` (component, request-response)

**Analog:** self — existing file, modify in place

**Current props interface** (lines 3-11):
```typescript
interface LocalNetworkBannerProps {
  visible: boolean
  tailscaleConnected: boolean
  tailscaleInstalled: boolean
  tailscaleBinaryFound: boolean
  tailscaleDaemonUp: boolean
  platformHint: string
  onOpenURL: (url: string) => void
}
```

**New props to add** (per RESEARCH.md §Code Examples):
```typescript
interface LocalNetworkBannerProps {
  // ... all existing props above ...
  onDismiss?: () => void    // NEW — D-03, D-04
  className?: string         // NEW — D-05 exit animation passthrough
}
```

**Imports addition** (add to line 1):
```typescript
import { XMarkIcon } from '@heroicons/react/20/solid'
```

**Updated function signature pattern** (line 25):
```typescript
export function LocalNetworkBanner({
  visible, tailscaleConnected, tailscaleInstalled: _tailscaleInstalled,
  tailscaleBinaryFound, tailscaleDaemonUp, platformHint, onOpenURL,
  onDismiss, className   // ADD these two
}: LocalNetworkBannerProps): React.ReactElement | null {
```

**className passthrough on each return div** (apply to all 4 return branches, lines 30, 44, 58, 74):
```tsx
// Before:
<div className="local-network-banner" role="status">
// After:
<div className={`local-network-banner${className ? ' ' + className : ''}`} role="status">
```

**Dismiss X button** (add as last child inside each banner div, after existing CTA if present):
```tsx
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
```

**Existing CTA button pattern** (line 82-88 — install-tailscale branch already uses `margin-left: auto` via `local-network-banner__cta`; dismiss button also uses `margin-left: auto` for non-CTA branches):
```tsx
<button
  className="local-network-banner__cta"
  onClick={() => onOpenURL('https://tailscale.com/download')}
  aria-label="Install Tailscale — opens tailscale.com in browser"
>
  Install Tailscale
</button>
```

---

### `frontend/src/components/UpdateBanner.tsx` (component, event-driven) — NEW FILE

**Analog:** `frontend/src/components/WelcomeTab.tsx` §update-banner (lines 51-79)

**Complete source to extract and adapt** (WelcomeTab.tsx lines 1-3 for imports, lines 51-79 for markup):
```typescript
import React from 'react'
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'
// XMarkIcon NOT needed — UpdateBanner keeps the existing text "Dismiss" button (RESEARCH.md §Code Examples note)
```

**UpdateInfo interface** (extracted from WelcomeTab.tsx lines 5-9 — move to UpdateBanner.tsx or App.tsx):
```typescript
interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseURL: string
}
```

**Props interface** (new):
```typescript
interface UpdateBannerProps {
  update: UpdateInfo
  onDismiss: () => void
  className?: string    // D-05 exit animation passthrough
}
```

**Component body** (adapted from WelcomeTab.tsx lines 51-79):
```tsx
export function UpdateBanner({ update, onDismiss, className }: UpdateBannerProps): React.ReactElement {
  return (
    <div
      className={`update-banner${className ? ' ' + className : ''}`}
      role="alert"
      aria-live="polite"
    >
      <span className="update-banner__message">
        Update available:{' '}
        <span className="update-banner__version">{update.currentVersion}</span>
        {' '}<span className="update-banner__arrow">&rarr;</span>{' '}
        <span className="update-banner__version">{update.latestVersion}</span>
      </span>
      <div className="update-banner__actions">
        <button
          type="button"
          className="update-banner__btn--download"
          onClick={() => BrowserOpenURL(update.releaseURL)}
        >
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

Note: `onClick={onDismiss}` replaces the inline `onClick={() => setUpdate(null)}` from WelcomeTab (line 73). The dismiss button keeps the text label "Dismiss" (not an X icon) per the RESEARCH.md §Code Examples note.

---

### `frontend/src/App.tsx` (provider/root, event-driven + request-response)

**Analog:** self — existing file; add new state, import, and BannerStack JSX

**New import to add** (after line 34 `import { LocalNetworkBanner }`):
```typescript
import { UpdateBanner } from './components/UpdateBanner'
```

**UpdateInfo interface** (add near top of file, or import from UpdateBanner.tsx if exported):
```typescript
interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseURL: string
}
```

**New state declarations** (add after existing `[remotePeers, setRemotePeers]` at line 88, following the existing `useState` pattern):
```typescript
// Update notification state (lifted from WelcomeTab — Phase 81)
const [update, setUpdate] = useState<UpdateInfo | null>(null)
// Local network banner dismiss state (session-only, D-04)
const [localBannerDismissed, setLocalBannerDismissed] = useState(false)
const [localBannerExiting, setLocalBannerExiting] = useState(false)
const [updateExiting, setUpdateExiting] = useState(false)
```

**Existing useState pattern for reference** (App.tsx lines 50-88 — all UI state declared at top of App function):
```typescript
const [tabs, setTabs] = useState<Tab[]>([WELCOME_TAB])
const [activeId, setActiveId] = useState<string | null>(WELCOME_TAB.id)
// ...
const [remotePeers, setRemotePeers] = useState<RemotePeerSessions[]>([])
const [remoteLoading, setRemoteLoading] = useState(false)
```

**Update subscription useEffect** (add after existing `useEffect` blocks, copying pattern from WelcomeTab.tsx lines 21-37):
```typescript
// Lift update:available subscription from WelcomeTab (Phase 81)
useEffect(() => {
  GetLastUpdateInfo()
    .then((info) => { if (info) setUpdate(info) })
    .catch(() => {})
  const offUpdate = EventsOn('update:available', (info: UpdateInfo) => {
    setUpdate(info)
  })
  return () => { offUpdate() }
}, [])
```

**Existing EventsOn pattern** (App.tsx lines 213-219 — how events are subscribed and cleaned up):
```typescript
const offStatus = EventsOn(
  'session:status',
  (data: { sessionId: string; status: string }) => {
    setSessionStatuses((prev) => ({ ...prev, [data.sessionId]: data.status }))
  },
)
```

**Dismiss handlers** (add using useCallback, following `handleThemeChange` pattern at App.tsx lines 100-106):
```typescript
const handleDismissLocalBanner = useCallback(() => {
  setLocalBannerExiting(true)
  setTimeout(() => {
    setLocalBannerDismissed(true)
    setLocalBannerExiting(false)
  }, 200)
}, [])

const handleDismissUpdate = useCallback(() => {
  setUpdateExiting(true)
  setTimeout(() => {
    setUpdate(null)
    setUpdateExiting(false)
  }, 200)
}, [])
```

**Existing useCallback pattern** (App.tsx lines 100-106):
```typescript
const handleThemeChange = useCallback((name: string) => {
  localStorage.setItem(THEME_STORAGE_KEY, name)
  setTerminalThemeName(name)
  NotifyThemeChange().catch(err => console.warn('NotifyThemeChange failed:', err))
}, [])
```

**BannerStack JSX** (replace lines 582-592 in App.tsx — current `{webServerMode === 'local' && (<LocalNetworkBanner .../>)}`):
```tsx
{/* Banner stack — replaces inline LocalNetworkBanner (Phase 81 BAN-01/BAN-02) */}
{(webServerMode === 'local' && !localBannerDismissed || update) && (
  <div className="banner-stack">
    {webServerMode === 'local' && !localBannerDismissed && (
      <LocalNetworkBanner
        visible={true}
        tailscaleConnected={!!(tailscaleHealth?.connected && tailscaleHealth?.hasCerts && tailscaleHealth?.ip)}
        tailscaleInstalled={!!(tailscaleHealth?.installed || detectedCLIs.some(c => c.Name === 'tailscale'))}
        tailscaleBinaryFound={!!(tailscaleHealth?.binaryFound)}
        tailscaleDaemonUp={!!(tailscaleHealth?.daemonUp)}
        platformHint={tailscaleHealth?.platformHint || ''}
        onOpenURL={BrowserOpenURL}
        onDismiss={handleDismissLocalBanner}
        className={localBannerExiting ? 'banner-exit' : undefined}
      />
    )}
    {update && (
      <UpdateBanner
        update={update}
        onDismiss={handleDismissUpdate}
        className={updateExiting ? 'banner-exit' : undefined}
      />
    )}
  </div>
)}
```

**webServerMode reset for D-04** (add useEffect to reset dismissed state when mode transitions to 'local'):
```typescript
// Reset dismissed state when entering local mode (D-04: reappear if conditions change)
useEffect(() => {
  if (webServerMode === 'local') {
    setLocalBannerDismissed(false)
  }
}, [webServerMode])
```

**GetLastUpdateInfo import** (add to existing wailsjs import block at lines 8-25):
```typescript
import {
  // ... existing imports ...
  GetLastUpdateInfo,
} from './wailsjs/go/main/App'
```

---

### `frontend/src/style.css` (config)

**Analog:** self — existing `.local-network-banner` block (lines 1441-1485) for BEM naming; `.update-banner` block (lines 940-1005) for update button patterns

**New `.banner-stack` rule** (add after `.local-network-banner__cta:hover` at line 1484, before the `.app__row` block at line 1488):
```css
/* ─── Banner stack container (Phase 81 — BAN-01/BAN-02) ─────────── */
.banner-stack {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  max-height: calc(3 * 53px);   /* D-02: cap at 3 banners (~53px each) */
  overflow-y: auto;
}
```

**New `.banner-exit` rule** (add after `.banner-stack`, D-05 dismiss animation):
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

**New `.local-network-banner__dismiss` rule** (add after `.local-network-banner__cta:hover` at line 1484 — D-03):
```css
.local-network-banner__dismiss {
  margin-left: auto;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  color: #9aa5ce;
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  min-height: 24px;
  flex-shrink: 0;
}
.local-network-banner__dismiss:hover {
  color: #c0caf5;
  border-color: #3b4261;
}
.local-network-banner__dismiss:focus-visible {
  outline: 2px solid #7aa2f7;
  outline-offset: 2px;
}
```

**Existing `.local-network-banner__cta` pattern** (lines 1469-1485 — reference for button color tokens):
```css
.local-network-banner__cta {
  margin-left: auto;
  background: #7aa2f7;
  color: #1a1b26;
  border: none;
  border-radius: 4px;
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
}
```

**Note on `.update-banner` CSS** (lines 940-953): The existing update-banner CSS has `margin-bottom: 24px` which is correct for inline-in-WelcomeTab but may need to become `0` when it's in the BannerStack. The element will be a flex row child in `.banner-stack` instead of a block-level element inside `.welcome-tab__content`. Adjust `margin-bottom: 0` on `.update-banner` when used in the stack context, or override via `.banner-stack .update-banner { margin-bottom: 0; }`.

---

### `frontend/src/components/WelcomeTab.tsx` (component, request-response)

**Analog:** self — existing file; remove lifted state and update banner markup

**Imports to remove** (lines 2-3 — `GetLastUpdateInfo` and `EventsOn` move to App):
```typescript
// REMOVE:
import { GetVersion, GetLastUpdateInfo } from '../wailsjs/go/main/App'
import { EventsOn, BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'
// KEEP as:
import { GetVersion } from '../wailsjs/go/main/App'
// BrowserOpenURL is no longer needed in WelcomeTab after extraction
```

**State to remove** (lines 13 — `update` state moves to App):
```typescript
// REMOVE:
const [update, setUpdate] = useState<UpdateInfo | null>(null)
```

**useEffect to remove** (lines 21-37 — entire `update:available` subscription block):
```typescript
// REMOVE entire block:
useEffect(() => {
  GetLastUpdateInfo()...
  const offUpdate = EventsOn('update:available', ...)
  return () => { offUpdate() }
}, [])
```

**JSX to remove** (lines 51-79 — the `{update && (<div className="update-banner" ...>)}` block):
```tsx
// REMOVE:
{update && (
  <div className="update-banner" role="alert" aria-live="polite">
    ...
  </div>
)}
```

**UpdateInfo interface to remove** (lines 5-9 — moves to App.tsx or UpdateBanner.tsx):
```typescript
// REMOVE from WelcomeTab:
interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseURL: string
}
```

---

### `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` (test)

**Analog:** self — existing file; extend with dismiss button tests

**Existing test helper** (lines 17-33 — `renderBanner` function, update to include `onDismiss`):
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

**Interface update needed** (line 7-15 — add `onDismiss` and `className` to the test-local interface copy):
```typescript
interface LocalNetworkBannerProps {
  // ... existing ...
  onDismiss?: () => void   // ADD
  className?: string        // ADD
}
```

**New test patterns to add** (following existing `it(...)` pattern at lines 44-145):
```typescript
it('renders dismiss button when onDismiss is provided', () => {
  const onDismiss = vi.fn()
  ;({ container, root } = renderBanner({ visible: true, onOpenURL: vi.fn(), onDismiss }))
  const dismissBtn = container.querySelector('.local-network-banner__dismiss')
  expect(dismissBtn).not.toBeNull()
})

it('does not render dismiss button when onDismiss is not provided', () => {
  ;({ container, root } = renderBanner({ visible: true, onOpenURL: vi.fn() }))
  const dismissBtn = container.querySelector('.local-network-banner__dismiss')
  expect(dismissBtn).toBeNull()
})

it('calls onDismiss when dismiss button clicked', () => {
  const onDismiss = vi.fn()
  ;({ container, root } = renderBanner({ visible: true, onOpenURL: vi.fn(), onDismiss }))
  const dismissBtn = container.querySelector('.local-network-banner__dismiss') as HTMLButtonElement
  flushSync(() => { dismissBtn.click() })
  expect(onDismiss).toHaveBeenCalledOnce()
})

it('applies className when provided', () => {
  ;({ container, root } = renderBanner({ visible: true, onOpenURL: vi.fn(), className: 'banner-exit' }))
  const banner = container.querySelector('.local-network-banner')
  expect(banner?.classList.contains('banner-exit')).toBe(true)
})
```

---

### `frontend/src/components/__tests__/UpdateBanner.test.tsx` (test) — NEW FILE

**Analog:** `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` — copy the `createRoot + flushSync` behavioral test pattern exactly

**File structure to follow** (LocalNetworkBanner.test.tsx lines 1-5, 17-33, 36-42):
```typescript
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { UpdateBanner } from '../UpdateBanner'
```

**Test helper pattern** (adapted from LocalNetworkBanner.test.tsx lines 17-33):
```typescript
function renderUpdateBanner(update: UpdateInfo, onDismiss: () => void, className?: string) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(UpdateBanner, { update, onDismiss, className }))
  })
  return { container, root }
}
```

**Lifecycle pattern** (LocalNetworkBanner.test.tsx lines 37-42 — copy exactly):
```typescript
afterEach(() => {
  root?.unmount()
  container?.remove()
})
```

**Test assertions to include** (cover BAN-02 requirements):
```typescript
it('renders update version information', ...)
it('renders Download Update button', ...)
it('renders Dismiss button with correct aria-label', ...)
it('calls onDismiss when Dismiss button clicked', ...)
it('has role="alert" for accessibility', ...)
it('applies className when provided (for banner-exit animation)', ...)
it('calls BrowserOpenURL with releaseURL on download click', ...)  // mock BrowserOpenURL
```

---

### `frontend/src/components/__tests__/App.test.tsx` (test)

**Analog:** self — existing file; add BannerStack integration describe block

**Existing structural test pattern** (lines 1-5, App.test.tsx — `?raw` import + `toContain` assertions):
```typescript
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App', () => {
  describe('some feature', () => {
    it('does something', () => {
      expect(raw).toContain('some string')
    })
  })
})
```

**New describe block to add** (following existing pattern — covers BAN-01/BAN-02):
```typescript
describe('BannerStack integration (BAN-01, BAN-02)', () => {
  it('imports UpdateBanner component', () => {
    expect(raw).toContain("import { UpdateBanner } from './components/UpdateBanner'")
  })

  it('declares update state at App level', () => {
    expect(raw).toContain('useState<UpdateInfo | null>(null)')
  })

  it('declares localBannerDismissed state', () => {
    expect(raw).toContain('localBannerDismissed')
  })

  it('renders banner-stack div', () => {
    expect(raw).toContain('banner-stack')
  })

  it('renders LocalNetworkBanner inside banner-stack', () => {
    const stackBlock = raw.slice(raw.indexOf('banner-stack'))
    expect(stackBlock).toContain('<LocalNetworkBanner')
  })

  it('renders UpdateBanner inside banner-stack', () => {
    const stackBlock = raw.slice(raw.indexOf('banner-stack'))
    expect(stackBlock).toContain('<UpdateBanner')
  })

  it('subscribes to update:available event', () => {
    expect(raw).toContain("EventsOn('update:available'")
  })

  it('passes onDismiss to LocalNetworkBanner', () => {
    expect(raw).toContain('onDismiss={handleDismissLocalBanner}')
  })

  it('passes onDismiss to UpdateBanner', () => {
    expect(raw).toContain('onDismiss={handleDismissUpdate}')
  })
})
```

**WelcomeTab.test.tsx update** — remove the `describe('update banner (UPD-02, UPD-03)', ...)` block (lines 62-124). These tests move to `UpdateBanner.test.tsx`. Also update the WelcomeTab import assertion at line 24: the test currently checks `import { GetVersion, GetLastUpdateInfo }` — after extraction it will be `import { GetVersion }` only.

---

## Shared Patterns

### BEM CSS Naming
**Source:** `frontend/src/style.css` lines 1441-1485 (`.local-network-banner`) and lines 940-1005 (`.update-banner`)
**Apply to:** All new CSS rules in style.css

The project uses strict BEM: `.block`, `.block__element`, `.block--modifier`. New CSS rules follow the same naming pattern. No CSS-in-JS, no Tailwind. Tokyo Night dark palette color tokens:
- Background: `#16161e`
- Border base: `#292e42`
- Border accent (hover): `#3b4261`
- Text primary: `#c0caf5`
- Text secondary: `#a9b1d6`
- Text muted: `#9aa5ce`
- Accent amber (warning): `#f59e0b`
- Accent blue (info): `#7aa2f7`
- Accent blue (hover): `#89b4fa`
- Focus ring: `#7aa2f7`

### useState at App Level
**Source:** `frontend/src/App.tsx` lines 50-88
**Apply to:** All new state declarations (`update`, `localBannerDismissed`, `localBannerExiting`, `updateExiting`)

Pattern: inline `useState<T>(initialValue)` at top of App function body, grouped with related state. Never use context for state that only flows downward to 1-2 components.

### useCallback for Event Handlers
**Source:** `frontend/src/App.tsx` lines 100-106 (`handleThemeChange`)
**Apply to:** `handleDismissLocalBanner`, `handleDismissUpdate`

```typescript
const handleDismissLocalBanner = useCallback(() => {
  // ...
}, [])
```

### EventsOn Subscription + Cleanup
**Source:** `frontend/src/App.tsx` lines 213-219, 221-231
**Apply to:** `update:available` subscription in App.tsx

```typescript
const offUpdate = EventsOn('update:available', (info: UpdateInfo) => {
  setUpdate(info)
})
// return from useEffect:
return () => { offUpdate() }
```

### Test: createRoot + flushSync Behavioral
**Source:** `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` lines 1-33
**Apply to:** `UpdateBanner.test.tsx` (copy the helper function and afterEach lifecycle exactly)

```typescript
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
// ...
const root = createRoot(container)
flushSync(() => { root.render(...) })
// afterEach: root?.unmount(); container?.remove()
```

### Test: ?raw Structural
**Source:** `frontend/src/components/__tests__/App.test.tsx` lines 1-5
**Apply to:** All new App.test.tsx assertions

```typescript
import raw from '../../App.tsx?raw'
expect(raw).toContain('some string')
```

---

## No Analog Found

All files have close analogs. No entries in this section.

---

## Metadata

**Analog search scope:** `frontend/src/components/`, `frontend/src/components/__tests__/`, `frontend/src/App.tsx`, `frontend/src/style.css`
**Files scanned:** 8 source files read directly
**Pattern extraction date:** 2026-04-16
