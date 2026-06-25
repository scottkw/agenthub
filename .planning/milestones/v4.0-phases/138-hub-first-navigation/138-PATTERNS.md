# Phase 138: Hub-First Navigation - Pattern Map

**Mapped:** 2026-06-20
**Files analyzed:** 10 modified files + 3 deleted files
**Analogs found:** 10 / 10

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/Sidebar.tsx` | component | event-driven | `frontend/src/components/Sidebar.tsx` (self — edit) | exact |
| `frontend/src/components/Hub/HubPanel.tsx` | component | request-response | `frontend/src/components/Hub/HubPanel.tsx` (self — edit) | exact |
| `frontend/src/components/Hub/SessionCard.tsx` | component | event-driven | `frontend/src/components/Hub/SessionCard.tsx` (self — edit) | exact |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | component | transform | `frontend/src/components/Hub/SessionCardGrid.tsx` (self — edit) | exact |
| `frontend/src/lib/remoteSession.ts` | utility | transform | `frontend/src/lib/remoteAdapter.ts` | exact |
| `frontend/src/lib/remoteAdapter.ts` | utility | transform | `frontend/src/lib/remoteAdapter.ts` (self — edit) | exact |
| `frontend/src/App.tsx` | provider | event-driven | `frontend/src/App.tsx` (self — edit) | exact |
| `frontend/src/components/TabBar.tsx` | component | event-driven | `frontend/src/components/TabBar.tsx` (self — edit) | exact |
| `frontend/src/components/__tests__/Sidebar.test.tsx` | test | request-response | `frontend/src/components/__tests__/Sidebar.test.tsx` (self — rewrite) | exact |
| `frontend/src/components/__tests__/App.hub.test.tsx` | test | request-response | `frontend/src/components/__tests__/App.hub.test.tsx` (self — edit) | exact |
| `frontend/src/components/__tests__/style.hub.test.ts` | test | request-response | `frontend/src/components/__tests__/style.hub.test.ts` (self — edit) | exact |
| `frontend/src/components/__tests__/SessionCard.share.test.tsx` | test | request-response | `frontend/src/components/__tests__/SessionCard.share.test.tsx` (extend) | exact |
| *(deleted)* `frontend/src/components/DaemonManagerPanel.tsx` | component | CRUD | n/a — deletion | n/a |
| *(deleted)* `frontend/src/components/RemoteSessionsPanel.tsx` | component | CRUD | n/a — deletion (types migrated first) | n/a |

---

## Pattern Assignments

### `frontend/src/components/Sidebar.tsx` (component, event-driven — EDIT)

**Change:** Remove `onOpenRemoteSessions`, `onOpenDaemonManager`, `onAdd` props and their corresponding
buttons (Remote, Sessions, New Session). Collapse sidebar to exactly Home / Hub / Settings (3 items).

**Imports pattern** (lines 1-10 — current state to trim):
```tsx
import React, { useState } from 'react'
import {
  Bars3Icon,
  ServerStackIcon,    // REMOVE — Sessions icon
  HomeIcon,
  GlobeAltIcon,      // REMOVE — Remote icon
  PlusIcon,          // REMOVE — New Session icon
  Cog6ToothIcon,
  Squares2X2Icon,    // KEEP — Hub icon
} from '@heroicons/react/24/outline'
```

**After trim — imports block:**
```tsx
import React, { useState } from 'react'
import {
  Bars3Icon,
  HomeIcon,
  Cog6ToothIcon,
  Squares2X2Icon,
} from '@heroicons/react/24/outline'
```

**Props interface pattern** (lines 14-22 — current, with deletions marked):
```tsx
interface SidebarProps {
  onHome: () => void
  onOpenRemoteSessions: () => void   // REMOVE
  onOpenDaemonManager: () => void    // REMOVE
  onAdd: () => void                  // REMOVE
  onSettings: () => void
  onOpenHub: () => void
  activePanel?: string
}
```

**Active-state button pattern** (lines 86-92 — authoritative pattern for new Hub item — KEEP):
```tsx
<button
  className={`sidebar__item${activePanel === '__hub__' ? ' sidebar__item--active' : ''}`}
  onClick={onOpenHub}
  aria-label="Hub"
>
  <Squares2X2Icon className="sidebar__icon" />
  {!collapsed && <span className="sidebar__label">Hub</span>}
</button>
```

**Buttons to REMOVE** (lines 67-101):
- `<button aria-label="Remote">` block (GlobeAltIcon)
- `<button aria-label="Sessions">` block (ServerStackIcon)
- `<button aria-label="New Session">` block (PlusIcon)

**Sidebar item count after:** 3 items (Home, Hub, Settings) — the toggle button and Settings remain
unchanged; only the three middle items are deleted.

---

### `frontend/src/components/Hub/HubPanel.tsx` (component, request-response — EDIT)

**Change:** Remove `.hub__header` block (CARD-01); thread `remoteCapsCached` down to
`SessionCardGrid` and `SessionCard` as `connectedRemoteIds`; thread `isRemote` provenance
prop; add `remotePeers` unreachable-hint rendering.

**hub__header DELETE block** (lines 462-467 — exact text to remove):
```tsx
{/* Header strip — UI-SPEC Layout Contract */}
<div className="hub__header">
  <span className="hub__title">Hub</span>
  <button className="hub__new-session-btn" type="button" onClick={onNewSession}>
    New session
  </button>
</div>
```

**remoteIdSet pattern** (lines 302-305 — authoritative provenance source, KEEP):
```tsx
// GAP-134-A: local vs remote is decided by PROVENANCE (which prop the session came
// from), NOT by hostname — local sessions carry the machine's os.Hostname(), so a
// hostname check misclassifies every local session as remote.
const remoteIdSet = React.useMemo(
  () => new Set((remoteSessions ?? []).map((s) => s.id)),
  [remoteSessions],
)
```

**Connected prop threading pattern** (copy pattern from `attentionIds` threading at lines 332-336):
```tsx
// Mirror attentionIds threading — derive boolean Set from remoteCapsCached
// and thread to SessionCardGrid as connectedRemoteIds
const connectedRemoteIds = React.useMemo(
  () => remoteCapsCached ?? new Set<string>(),
  [remoteCapsCached],
)
```

**SessionCardGrid invocation to extend** (lines 443-455 — add `connectedRemoteIds` + `remoteIdSet`):
```tsx
<SessionCardGrid
  sessions={visibleSessions}
  onRename={onRename}
  onOpenSession={onOpenSession}
  onCardClick={handleCardClick}
  onShare={handleShare}
  groupDefs={groupDefs.length > 0 ? groupDefs : undefined}
  previewTails={previewTails}
  onAssignGroup={handleAssignGroup}
  attentionIds={attentionIds}
  debouncedSortKey={debouncedSortKey}
  // NEW — thread connection state and provenance:
  connectedRemoteIds={connectedRemoteIds}
  remoteIdSet={remoteIdSet}
  onKill={...}            // NEW — wired to handleCloseTab from props
  onOpenInBrowser={...}  // NEW — wired to handleOpenRemoteSession from props
  onBrowseFiles={...}    // NEW — wired to handleBrowseFilesRemote from props
/>
```

**Peer unreachable hint pattern** (UI-SPEC §Per-peer hint — new, below HubFilterBar):
```tsx
{/* Per-peer unreachable/empty hint — renders only when relevant peers exist */}
{(remotePeers ?? [])
  .filter((p) => !p.reachable || p.sessions.length === 0)
  .map((p) => (
    <p key={p.hostname} className="hub__peer-hint">
      {!p.reachable
        ? `${p.hostname} is unreachable`
        : `${p.hostname} has no shared sessions`}
    </p>
  ))}
```

**New props to add to HubPanelProps interface:**
```tsx
/** Phase 138 — Kill handler threaded to card overflow menu */
onKill?: (sessionId: string) => void
/** Phase 138 — Open remote session in system browser */
onOpenInBrowser?: (url: string) => void
/** Phase 138 — Browse remote session files (join-code flow) */
onBrowseFiles?: (sessionId: string, sessionName: string) => void
/** Phase 138 — remotePeers raw data for unreachable-peer hints */
remotePeers?: RemotePeerSessions[]
```

---

### `frontend/src/components/Hub/SessionCard.tsx` (component, event-driven — EDIT)

**Change:** Add `isRemote`, `isConnected`, `onKill`, `onOpenInBrowser`, `onBrowseFiles` props;
fix provenance-based `isRemote` derivation; add CARD-03 connection chip; add CARD-04 overflow
menu items; extend `cardAriaLabel`.

**Imports — icons to ADD** (extend existing import block at lines 1-16):
```tsx
import {
  // ... existing imports ...
  LinkIcon,                    // CARD-03: "Connected" state shape signal
  ArrowTopRightOnSquareIcon,   // CARD-04: "Open in browser" menu item icon
} from '@heroicons/react/24/outline'
```

**Props interface additions** (after existing `onShare` prop at line 113):
```tsx
/**
 * Phase 138 / CARD-02: explicit provenance flag — true when session came from
 * remoteSessions prop (not local sessions). Replaces hostname-based isLocal heuristic.
 * Derived in HubPanel from remoteIdSet.has(session.id).
 */
isRemote?: boolean
/**
 * Phase 138 / CARD-03: true when remoteCapsCached.has(session.id) — user has
 * exchanged a join code. Never exposes the token itself (T-122-03-01).
 */
isConnected?: boolean
/** Phase 138 / CARD-04: Kill session — wired to handleCloseTab in App. */
onKill?: (sessionId: string) => void
/** Phase 138 / CARD-04: Open remote session in system browser. */
onOpenInBrowser?: (url: string) => void
/** Phase 138 / CARD-04: Browse remote files (join-code cap flow). */
onBrowseFiles?: (sessionId: string, sessionName: string) => void
```

**isRemote / isLocal derivation CHANGE** (line 164 — current):
```tsx
// CURRENT (hostname-based — unreliable):
const isLocal = !hostname || hostname === ''
const originText = isLocal ? 'Local' : hostname

// REPLACE WITH (provenance-based — CARD-02 fix):
const isLocal = !isRemote  // derive from explicit prop, not hostname
const originText = isLocal ? 'Local' : hostname
```

**cardAriaLabel EXTENSION** (line 180 — extend to include connection state):
```tsx
// CURRENT:
const cardAriaLabel = `${name}, ${displayLabel}, ${cli}, ${originText}${isAttention ? ', needs attention' : ''}`

// EXTENDED (append connection state for remote cards):
const cardAriaLabel = `${name}, ${displayLabel}, ${cli}, ${originText}${isAttention ? ', needs attention' : ''}${isRemote ? (isConnected ? ', connected' : ', available') : ''}`
```

**card-click guard pattern** (lines 248-257 — authoritative, COPY for all new clickable children):
```tsx
onClick={(e) => {
  // Defense-in-depth: verify click did not originate from controlled children
  const target = e.target as HTMLElement
  if (target.closest('.hub-card__open')) return
  if (target.closest('.hub-card__share')) return  // D-12/Pitfall 6
  if (target.closest('.hub-card__menu-btn')) return
  if (target.closest('.hub-card__menu')) return   // covers Kill/Open-browser/Browse
  if (target.closest('.InlineSessionName input')) return
  if (isDragging) return
  onCardClick?.(session, (e.currentTarget as HTMLElement).getBoundingClientRect())
}}
```

**New inline child e.stopPropagation() pattern** (copy from Share button at line 421):
```tsx
// All new interactive children MUST use this pattern:
onClick={(e) => { e.stopPropagation(); /* action */ }}
```

**Overflow menu additions** (after line 320 — inside `{menuOpen && (...)}` block):
```tsx
{/* Remote-only actions */}
{isRemote && (
  <>
    <hr className="hub-card__menu-divider" />
    <button
      type="button"
      className="hub-card__menu-item"
      role="menuitem"
      onClick={(e) => { e.stopPropagation(); onOpenInBrowser?.(session.url ?? ''); setMenuOpen(false) }}
    >
      Open in browser
    </button>
    <button
      type="button"
      className="hub-card__menu-item"
      role="menuitem"
      onClick={(e) => { e.stopPropagation(); onBrowseFiles?.(id, name); setMenuOpen(false) }}
    >
      Browse files
    </button>
  </>
)}
{/* Kill session — all live sessions */}
{session.state !== 'stopped' && (
  <>
    <hr className="hub-card__menu-divider" />
    {/* Two-step inline confirm pattern (UI-SPEC §Kill) */}
    <KillConfirmItem onKill={() => { onKill?.(id); setMenuOpen(false) }} />
  </>
)}
```

**KillConfirmItem inline two-step pattern** (new local component in SessionCard.tsx):
```tsx
// Two-step destructive confirm — label flips on first click, second click confirms.
// No modal — inline within the overflow menu (UI-SPEC Claude's Discretion choice).
function KillConfirmItem({ onKill }: { onKill: () => void }) {
  const [confirming, setConfirming] = useState(false)
  return (
    <button
      type="button"
      className={`hub-card__menu-item hub-card__menu-item--destructive`}
      role="menuitem"
      onClick={(e) => {
        e.stopPropagation()
        if (!confirming) { setConfirming(true); return }
        onKill()
      }}
    >
      {confirming ? (
        <span>
          <span>Confirm kill</span>
          <span className="hub-card__menu-item-sub">This will stop the session</span>
        </span>
      ) : 'Kill session'}
    </button>
  )
}
```

**CARD-03 connection chip** (new ROW 2b, after the ROW 2 origin block at lines 354-370):
```tsx
{/* ROW 2b: connection indicator — remote cards only (CARD-03)
    COLORBLIND-SAFE: LinkIcon (connected) + GlobeAltIcon (available) carry the state;
    color is reinforcement only. Hex source: --hub-accent, --hub-text-muted. */}
{isRemote && (
  <div className="hub-card__row2b">
    <span className={`hub-card__conn${isConnected ? ' hub-card__conn--connected' : ''}`}>
      {isConnected ? (
        <><LinkIcon className="hub-card__conn-icon" aria-hidden="true" /><span>Connected</span></>
      ) : (
        <><GlobeAltIcon className="hub-card__conn-icon" aria-hidden="true" /><span>Available</span></>
      )}
    </span>
  </div>
)}
```

**STATUS_CONFIG colorblind-safe pattern** (lines 26-52 — authoritative pattern to replicate for CARD-03):
```tsx
// COLORBLIND-SAFE pattern: every state has unique icon shape + text label;
// color (hex) is reinforcement only. Verify at source (hex constants), not by eye.
// CARD-03 mirrors this: LinkIcon+text vs GlobeAltIcon+text; color via CSS custom property.
```

---

### `frontend/src/components/Hub/SessionCardGrid.tsx` (component, transform — EDIT)

**Change:** Thread `connectedRemoteIds`, `remoteIdSet`, `onKill`, `onOpenInBrowser`,
`onBrowseFiles` props down to each `SessionCard` in both render paths.

**Props interface additions** (after `onShare` at line 150):
```tsx
/** Phase 138 — Set of session IDs with a cached cap (isConnected signal) */
connectedRemoteIds?: Set<string>
/** Phase 138 — Set of remote session IDs (provenance-based isRemote signal) */
remoteIdSet?: Set<string>
/** Phase 138 — Kill handler (threaded from HubPanel → App.handleCloseTab) */
onKill?: (sessionId: string) => void
/** Phase 138 — Open remote session in browser (threaded from HubPanel) */
onOpenInBrowser?: (url: string) => void
/** Phase 138 — Browse remote files (threaded from HubPanel) */
onBrowseFiles?: (sessionId: string, sessionName: string) => void
```

**SessionCard invocation pattern** (lines 245-255 and 289-299 — BOTH render paths must be updated
identically; copy from existing `attentionIds` threading pattern at line 254):
```tsx
<SessionCard
  session={s}
  onRename={onRename}
  onOpenSession={onOpenSession}
  onCardClick={onCardClick}
  onShare={onShare}
  previewLines={previewTails?.get(s.id)}
  groupDefs={groupDefs}
  onAssignGroup={onAssignGroup}
  isAttention={attentionIds?.has(s.id)}
  // NEW — Phase 138 props:
  isRemote={remoteIdSet?.has(s.id)}
  isConnected={connectedRemoteIds?.has(s.id)}
  onKill={onKill}
  onOpenInBrowser={onOpenInBrowser}
  onBrowseFiles={onBrowseFiles}
/>
```

**Both render paths note:** Lines 239-256 (named-group path) and 283-299 (workDir-group path) are
structurally identical `<SessionCard>` invocations — update both; they share the same `attentionIds?.has(s.id)`
pattern that new props follow exactly.

---

### `frontend/src/lib/remoteSession.ts` (utility, transform — EDIT: add type exports)

**Change:** Add the `RemoteSession` and `RemotePeerSessions` interface exports that currently live in
`RemoteSessionsPanel.tsx` (lines 1-16 of that file). This relocates the types BEFORE the panel
is deleted.

**Types to migrate from `RemoteSessionsPanel.tsx` lines 3-16:**
```tsx
export interface RemoteSession {
  id: string
  name: string
  cliType: string
  status: string
  url: string
}

export interface RemotePeerSessions {
  hostname: string
  /** Phase 130 — true when the peer responded to the metadata probe; false when unreachable. */
  reachable: boolean
  sessions: RemoteSession[]
}
```

**Add above the existing `RemoteSessionWithHost` interface in `remoteSession.ts`** (currently line 13).
The file already imports from `RemoteSessionsPanel` — after the types are added here, remove that import.

---

### `frontend/src/lib/remoteAdapter.ts` (utility, transform — EDIT: update import)

**Change:** Update the import source from `RemoteSessionsPanel` to `remoteSession`.

**Current import** (line 2):
```tsx
import type { RemotePeerSessions, RemoteSession } from '../components/RemoteSessionsPanel'
```

**After type relocation:**
```tsx
import type { RemotePeerSessions, RemoteSession } from './remoteSession'
```

**Rest of the file is unchanged** — `adaptRemoteSession` and `adaptAllRemoteSessions` are
already the canonical implementations (lines 5-29).

---

### `frontend/src/App.tsx` (provider, event-driven — EDIT)

**Change:** Remove `DAEMON_MANAGER_TAB`, `REMOTE_SESSIONS_TAB`, dead handlers, dead render
branches, half of the remote poll guard, and the three deleted props from `<Sidebar>`.
Also wire new `onKill`, `onOpenInBrowser`, `onBrowseFiles` props into `<HubPanel>`.

**Consts to REMOVE** (lines 88-89):
```tsx
const DAEMON_MANAGER_TAB: Tab = { id: '__daemon_manager__', ... }  // REMOVE
const REMOTE_SESSIONS_TAB: Tab = { id: '__remote_sessions__', ... } // REMOVE
// HUB_TAB at line 92 — KEEP
```

**handleAddTab** (line 737 — verify no other caller before removing):
```tsx
// Current:
const handleAddTab = useCallback(() => {
  setShowNewSessionModal(true)
}, [])
// REMOVE — if grep confirms no other caller besides the deleted sidebar onAdd prop.
// Hub uses onNewSession → setShowNewSessionModal(true) directly. Run:
// grep -n "handleAddTab" frontend/src/App.tsx
```

**handleCloseTab** (lines 749-783 — KEEP, expose as onKill to HubPanel):
```tsx
// EXISTING — already handles kill + tab cleanup. Wire to HubPanel:
// In <HubPanel>:  onKill={(id) => void handleCloseTab(id)}
// (mirrors the existing DaemonManagerPanel wiring at line 1364)
onKill={(id) => void handleCloseTab(id)}
```

**handleOpenRemoteSession** (line 1062 — KEEP, wire to HubPanel):
```tsx
const handleOpenRemoteSession = useCallback((url: string) => {
  BrowserOpenURL(url)
}, [])
// Wire to HubPanel: onOpenInBrowser={handleOpenRemoteSession}
```

**handleBrowseFilesRemote** (lines 1069-1104 — KEEP, wire to HubPanel):
```tsx
// Wire to HubPanel: onBrowseFiles={handleBrowseFilesRemote}
// Existing signature: (sessionId: string, sessionName: string) => Promise<void>
```

**handleOpenDaemonManager** (line 992 — REMOVE), **handleOpenRemoteSessions** (line 1141 — REMOVE):
```tsx
const handleOpenDaemonManager = useCallback(() => { ... })  // REMOVE
const handleOpenRemoteSessions = useCallback(() => { ... }) // REMOVE
```

**DaemonManager poll REMOVE** (lines 891-911 — remove entire effect):
```tsx
// Poll sessions when the daemon-manager panel tab is active.
useEffect(() => {
  ...
  const isDaemonManagerActive = activeId === DAEMON_MANAGER_TAB.id
  ...
}, [activeId])   // REMOVE this entire effect
```

**Remote poll — partial EDIT** (line 946 — remove only the REMOTE_SESSIONS_TAB half):
```tsx
// CURRENT:
if (activeId !== REMOTE_SESSIONS_TAB.id && activeId !== HUB_TAB.id) return

// AFTER (keep HUB_TAB guard; Hub still needs remote data):
if (activeId !== HUB_TAB.id) return
```

**DaemonManagerPanel render branch REMOVE** (lines 1357-1369):
```tsx
{activeId === DAEMON_MANAGER_TAB.id && (
  <DaemonManagerPanel ... />
)}   // REMOVE ENTIRE BLOCK
```

**RemoteSessionsPanel render branch REMOVE** (lines 1478-1486):
```tsx
{activeId === REMOTE_SESSIONS_TAB.id && (
  <RemoteSessionsPanel ... />
)}   // REMOVE ENTIRE BLOCK
```

**Tab-type filter lists — EDIT** (lines 1517 and 1556):
```tsx
// Line 1517 — remove 'daemon-manager' and 'remote-sessions' from the filter:
// CURRENT:
tabs.filter((t) => t.type !== 'welcome' && t.type !== 'daemon-manager' && t.type !== 'remote-sessions' && t.type !== 'settings' && t.type !== 'hub')
// AFTER:
tabs.filter((t) => t.type !== 'welcome' && t.type !== 'settings' && t.type !== 'hub')

// Line 1556 — same removal:
// CURRENT:
if (tab.type === 'welcome' || tab.type === 'daemon-manager' || tab.type === 'remote-sessions' || ...)
// AFTER:
if (tab.type === 'welcome' || tab.type === 'settings' || tab.type === 'file-browser' || tab.type === 'hub') return null
```

**Sidebar wiring EDIT** (lines 1324-1332 — remove three deleted props):
```tsx
// CURRENT:
<Sidebar
  onHome={handleHome}
  onOpenRemoteSessions={handleOpenRemoteSessions}   // REMOVE
  onOpenDaemonManager={handleOpenDaemonManager}     // REMOVE
  onAdd={handleAddTab}                              // REMOVE
  onSettings={handleOpenSettings}
  onOpenHub={handleOpenHub}
  activePanel={activeId ?? undefined}
/>

// AFTER:
<Sidebar
  onHome={handleHome}
  onSettings={handleOpenSettings}
  onOpenHub={handleOpenHub}
  activePanel={activeId ?? undefined}
/>
```

**HubPanel wiring ADDITIONS** (after existing props at line 1384):
```tsx
<HubPanel
  ...existing props...
  onKill={(id) => void handleCloseTab(id)}
  onOpenInBrowser={handleOpenRemoteSession}
  onBrowseFiles={handleBrowseFilesRemote}
  remotePeers={remotePeers}
/>
```

---

### `frontend/src/components/TabBar.tsx` (component, event-driven — EDIT)

**Change:** Remove `'daemon-manager'` and `'remote-sessions'` from the `Tab.type` union (line 8).

**Current:**
```tsx
type?: 'terminal' | 'welcome' | 'daemon-manager' | 'remote-sessions' | 'settings' | 'file-browser' | 'hub'
```

**After:**
```tsx
type?: 'terminal' | 'welcome' | 'settings' | 'file-browser' | 'hub'
```

---

### `frontend/src/components/__tests__/Sidebar.test.tsx` (test, request-response — REWRITE)

**Change (Wave 0):** Remove assertions for Sessions, New Session, Remote buttons.
Update `items.length` assertions from 6 → 3. Update `renderSidebar` helper to remove
deleted props.

**renderSidebar helper — updated** (lines 13-29 — remove three deleted props):
```tsx
// CURRENT defaultProps includes:
onOpenDaemonManager: vi.fn(),  // REMOVE
onOpenRemoteSessions: vi.fn(), // REMOVE
onAdd: vi.fn(),                // REMOVE

// AFTER: only Home/Hub/Settings props remain:
function renderSidebar(overrides: Partial<Parameters<typeof Sidebar>[0]> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const defaultProps = {
    onSettings: vi.fn(),
    onHome: vi.fn(),
    onOpenHub: vi.fn(),
  }
  act(() => { root.render(<Sidebar {...defaultProps} {...overrides} />) })
  return { container, root, ...defaultProps }
}
```

**Tests to REMOVE** (they assert the deleted items):
- Line 59-64: `'renders a Sessions sidebar__item button'`
- Line 66-73: `'renders "New Session" label...'`
- Line 220-230: `'Sessions button (aria-label="Sessions") contains an SVG element'`

**Tests to ADD** (assert new 3-item structure):
```tsx
// Follow established describe/it structure in the file:
it('renders exactly 3 sidebar__item buttons (Home, Hub, Settings)', () => {
  ;({ container, root } = renderSidebar())
  const items = container.querySelectorAll('button.sidebar__item')
  expect(items.length).toBe(3)
})

it('does NOT render a Sessions button', () => {
  ;({ container, root } = renderSidebar())
  expect(container.querySelector('button[aria-label="Sessions"]')).toBeNull()
})

it('does NOT render a Remote button', () => {
  ;({ container, root } = renderSidebar())
  expect(container.querySelector('button[aria-label="Remote"]')).toBeNull()
})

it('does NOT render a New Session button', () => {
  ;({ container, root } = renderSidebar())
  expect(container.querySelector('button[aria-label="New Session"]')).toBeNull()
})
```

**items.length assertions to update:**
- Line 200-202: `expect(items.length).toBe(6)` → `expect(items.length).toBe(3)`
- Line 297-298: `expect(expandedIcons.length).toBeGreaterThanOrEqual(7)` → `toBeGreaterThanOrEqual(4)` (toggle + 3 items)
- Line 297-298 comment update: `1 toggle + 3 nav items = 4 sidebar__icon SVGs total`

---

### `frontend/src/components/__tests__/App.hub.test.tsx` (test, request-response — EDIT)

**Change (Wave 0):** Remove tests that assert DAEMON_MANAGER_TAB / REMOTE_SESSIONS_TAB.
Add tests asserting those are absent. Add test for no `.hub__header` in HubPanel source.
Add tests for new HubPanel props (`onKill`, `onOpenInBrowser`, `onBrowseFiles`).

**Source-inspection test pattern** (lines 1-3 — authoritative pattern for this file):
```tsx
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'
// All assertions are raw.toContain() / raw.not.toContain()
```

**Tests to ADD following this pattern:**
```tsx
describe('NAV-03/04: DaemonManagerPanel and RemoteSessionsPanel are removed', () => {
  it('does not define DAEMON_MANAGER_TAB constant', () => {
    expect(raw).not.toContain('DAEMON_MANAGER_TAB')
  })
  it('does not define REMOTE_SESSIONS_TAB constant', () => {
    expect(raw).not.toContain('REMOTE_SESSIONS_TAB')
  })
  it('does not render DaemonManagerPanel', () => {
    expect(raw).not.toContain('DaemonManagerPanel')
  })
  it('does not render RemoteSessionsPanel', () => {
    expect(raw).not.toContain('RemoteSessionsPanel')
  })
})

describe('CARD-01: HubPanel receives onKill, onOpenInBrowser, onBrowseFiles', () => {
  it('wires onKill to handleCloseTab', () => {
    expect(raw).toContain('onKill=')
  })
  it('wires onOpenInBrowser to handleOpenRemoteSession', () => {
    expect(raw).toContain('onOpenInBrowser=')
  })
  it('wires onBrowseFiles to handleBrowseFilesRemote', () => {
    expect(raw).toContain('onBrowseFiles=')
  })
})
```

---

### `frontend/src/components/__tests__/style.hub.test.ts` (test, request-response — EDIT)

**Change (Wave 0):** Remove `.hub__header` assertions; add CARD-03/CARD-04 CSS assertions.

**Tests to REMOVE** (assert deleted CSS classes):
```tsx
// Any test asserting `.hub__header`, `.hub__title`, `.hub__new-session-btn`
expect(cssRaw).toContain('.hub__header')   // REMOVE — deleted CSS
```

**Tests to ADD:**
```tsx
describe('CARD-03: Connection indicator CSS (hub-card__conn)', () => {
  it('defines .hub-card__conn class', () => {
    expect(cssRaw).toContain('.hub-card__conn')
  })
  it('defines .hub-card__conn--connected modifier', () => {
    expect(cssRaw).toContain('.hub-card__conn--connected')
  })
  it('defines .hub-card__conn-icon class', () => {
    expect(cssRaw).toContain('.hub-card__conn-icon')
  })
})

describe('CARD-04: Preserved grid CSS (anti-regression)', () => {
  it('preserves .hub__card-row grid definition', () => {
    expect(cssRaw).toContain('.hub__card-row')
  })
  it('preserves .hub-card--attention class', () => {
    expect(cssRaw).toContain('.hub-card--attention')
  })
  it('preserves hub-card min-width 240px constraint', () => {
    expect(cssRaw).toContain('240px')
  })
})

describe('CARD-04: Kill menu item CSS', () => {
  it('defines .hub-card__menu-item--destructive modifier', () => {
    expect(cssRaw).toContain('.hub-card__menu-item--destructive')
  })
  it('destructive color uses --hub-destructive custom property', () => {
    expect(cssRaw).toContain('--hub-destructive')
  })
})
```

---

### `frontend/src/components/__tests__/SessionCard.share.test.tsx` (test, request-response — EXTEND)

**Change:** Add CARD-02 origin + CARD-03 connection indicator tests and CARD-04 Kill/remote menu tests,
following the existing fixture + `renderCard` + `flushSync` pattern in the file.

**Existing fixture pattern** (lines 30-75 — authoritative; extend by adding `isRemote`/`isConnected` variants):
```tsx
// The file already defines localSession and remoteSession fixtures.
// Add to the remoteSession fixture or create new fixtures for connected/available states:
const connectedRemoteSession: SessionInfo = {
  ...remoteSession,
  id: 'sess-3',
  name: 'Connected Remote',
}
// Render with isRemote + isConnected props via renderCard:
renderCard({ session: connectedRemoteSession, isRemote: true, isConnected: true })
```

**renderCard function signature to extend** (line 61-75 — add new props to RenderOpts):
```tsx
interface RenderOpts {
  session?: SessionInfo
  onShare?: (session: SessionInfo) => void
  onCardClick?: () => void
  // NEW:
  isRemote?: boolean
  isConnected?: boolean
  onKill?: (sessionId: string) => void
  onOpenInBrowser?: (url: string) => void
  onBrowseFiles?: (sessionId: string, sessionName: string) => void
}
```

**Tests to ADD (follow describe/it structure of the file):**
```tsx
describe('CARD-02: Local vs remote origin indicator', () => {
  it('local card renders ComputerDesktopIcon and "Local" text', () => {
    const { container } = renderCard({ session: localSession, isRemote: false })
    const origin = container.querySelector('.hub-card__origin')
    expect(origin?.textContent).toContain('Local')
    expect(origin?.querySelector('svg')).not.toBeNull()
  })
  it('remote card renders GlobeAltIcon and hostname text', () => {
    const { container } = renderCard({ session: remoteSession, isRemote: true })
    const origin = container.querySelector('.hub-card__origin')
    expect(origin?.textContent).toContain('remote.host')
    expect(origin?.querySelector('svg')).not.toBeNull()
  })
})

describe('CARD-03: Connection indicator (colorblind-safe)', () => {
  it('connected remote card renders .hub-card__conn--connected and "Connected" text', () => {
    const { container } = renderCard({ session: remoteSession, isRemote: true, isConnected: true })
    const chip = container.querySelector('.hub-card__conn--connected')
    expect(chip).not.toBeNull()
    expect(chip?.textContent).toContain('Connected')
    expect(chip?.querySelector('svg')).not.toBeNull() // LinkIcon
  })
  it('available remote card renders .hub-card__conn without --connected and "Available" text', () => {
    const { container } = renderCard({ session: remoteSession, isRemote: true, isConnected: false })
    const chip = container.querySelector('.hub-card__conn')
    expect(chip).not.toBeNull()
    expect(chip?.textContent).toContain('Available')
    expect(chip?.classList.contains('hub-card__conn--connected')).toBe(false)
  })
  it('connection chip is absent on local cards', () => {
    const { container } = renderCard({ session: localSession, isRemote: false })
    expect(container.querySelector('.hub-card__conn')).toBeNull()
  })
  it('connected card aria-label includes ", connected"', () => {
    const { container } = renderCard({ session: remoteSession, isRemote: true, isConnected: true })
    const article = container.querySelector('article')
    expect(article?.getAttribute('aria-label')).toContain(', connected')
  })
})

describe('CARD-04: Kill menu item guard (stopPropagation)', () => {
  it('Kill option appears in overflow menu for live local sessions', () => {
    const onKill = vi.fn()
    const { container } = renderCard({ session: localSession, onKill })
    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    act(() => { menuBtn.click() })
    expect(container.textContent).toContain('Kill session')
  })
  it('Kill confirm does not trigger card-click modal', () => {
    const onCardClick = vi.fn()
    const onKill = vi.fn()
    const { container } = renderCard({ session: localSession, onKill, onCardClick })
    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    act(() => { menuBtn.click() })
    const killBtn = Array.from(container.querySelectorAll('.hub-card__menu-item')).find(
      (el) => el.textContent?.includes('Kill session')
    ) as HTMLButtonElement
    flushSync(() => { killBtn?.click() })
    expect(onCardClick).not.toHaveBeenCalled()
  })
})
```

---

## Shared Patterns

### Pattern A: Colorblind-Safe Icon + Text (applies to ALL new indicators)

**Source:** `frontend/src/components/Hub/SessionCard.tsx` lines 26-52 (STATUS_CONFIG) and lines 357-369 (origin row)
**Apply to:** CARD-02 origin row (provenance fix), CARD-03 connection chip, CARD-04 Kill menu item
```tsx
// Authoritative precedent — EVERY state carries icon shape + text label:
{isLocal ? (
  <><ComputerDesktopIcon className="hub-card__origin-icon" aria-hidden="true" /><span>Local</span></>
) : (
  <><GlobeAltIcon className="hub-card__origin-icon" aria-hidden="true" /><span>{hostname}</span></>
)}
// Color is CSS custom property reinforcement only. Hex constants:
// --hub-accent: #7aa2f7 (dark) / #3d6fe8 (light)
// --hub-text-muted: verify in style.css
// VERIFY AT SOURCE — do not verify by eye (owner is colorblind, MEMORY.md user_colorblind)
```

### Pattern B: e.stopPropagation() Guard for Card Click (applies to ALL new interactive children)

**Source:** `frontend/src/components/Hub/SessionCard.tsx` lines 248-257 (article onClick guard) and line 421 (Share button)
**Apply to:** Kill button, Open-in-browser item, Browse-files item in overflow menu
```tsx
// Step 1: add .closest() guard in article onClick:
if (target.closest('.hub-card__menu')) return  // already present — covers new menu items

// Step 2: stopPropagation on each new interactive child:
onClick={(e) => { e.stopPropagation(); /* action */ }}
```

### Pattern C: Provenance-Based isRemote (applies to CARD-02/03 and prop threading)

**Source:** `frontend/src/components/Hub/HubPanel.tsx` lines 302-305
**Apply to:** HubPanel → SessionCardGrid → SessionCard prop chain
```tsx
const remoteIdSet = React.useMemo(
  () => new Set((remoteSessions ?? []).map((s) => s.id)),
  [remoteSessions],
)
// Use: isRemote={remoteIdSet.has(s.id)} — never derive from hostname
```

### Pattern D: Set-Threading Prop Pattern (applies to connectedRemoteIds threading)

**Source:** `frontend/src/components/Hub/SessionCardGrid.tsx` lines 143-145 (`attentionIds` prop) and lines 254 (`attentionIds?.has(s.id)`)
**Apply to:** `connectedRemoteIds` and `remoteIdSet` prop threading in SessionCardGrid
```tsx
// Props declaration pattern (mirrors attentionIds):
connectedRemoteIds?: Set<string>
// Usage pattern (mirrors attentionIds?.has(s.id)):
isConnected={connectedRemoteIds?.has(s.id)}
```

### Pattern E: Source-Inspection Test Pattern (applies to App.hub.test.tsx additions)

**Source:** `frontend/src/components/__tests__/App.hub.test.tsx` lines 1-3 and all describe blocks
**Apply to:** All new App.hub.test.tsx assertions
```tsx
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'
// Pattern: raw.toContain('literal string') or raw.not.toContain(...)
// No mounting, no mocks — pure source inspection
```

### Pattern F: CSS Source Test Pattern (applies to style.hub.test.ts additions)

**Source:** `frontend/src/components/__tests__/style.hub.test.ts` lines 1-10 and all assertions
**Apply to:** All new style.hub.test.ts CSS assertions
```tsx
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')
// Pattern: expect(cssRaw).toContain('.class-name')
// Verifies CSS classes exist at source; JSDOM has no layout engine
```

### Pattern G: CSS Custom Property Convention (applies to CARD-03 chip CSS)

**Source:** `frontend/src/components/__tests__/style.hub.test.ts` lines 44-71 (token assertions)
**Apply to:** New `.hub-card__conn` CSS block in `frontend/src/style.css`
```css
/* UI-SPEC exact CSS for CARD-03 connection chip: */
.hub-card__conn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0 8px;
  height: 20px;
  border-radius: 10px;
  border: 1px solid currentColor;
  font-size: 12px;
  font-family: system-ui, -apple-system, "Segoe UI", sans-serif;
  color: var(--hub-text-muted);       /* COLORBLIND-SAFE: available state; text+icon carry the signal */
}
.hub-card__conn--connected {
  color: var(--hub-accent);           /* COLORBLIND-SAFE: reinforcement only; LinkIcon+text carry the signal */
}
.hub-card__conn-icon {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
}
.hub-card__menu-item--destructive {
  color: var(--hub-destructive);      /* COLORBLIND-SAFE: reinforcement; "Kill session" text carries the signal */
}
```

---

## CSS File Changes (style.css)

**File:** `frontend/src/style.css`
**Changes:**
1. ADD new classes: `.hub-card__conn`, `.hub-card__conn--connected`, `.hub-card__conn-icon`, `.hub-card__menu-item--destructive`, `.hub-card__menu-divider` (if not already present), `.hub__peer-hint`, `.hub-card__menu-item-sub`
2. REMOVE or comment out: `.hub__header`, `.hub__title`, `.hub__new-session-btn` (dead after CARD-01 deletion)
3. PRESERVE (must not touch): `.hub__card-row`, `.hub-card`, `.hub-card--attention`, `.hub-card__open`, `.hub-card__share`, `.hub-card__menu-item`, `@media (prefers-reduced-motion: reduce)` attention guard

**Locate hub__header rules in style.css:**
```bash
grep -n "hub__header\|hub__title\|hub__new-session-btn" frontend/src/style.css
```

---

## No Analog Found

All files in scope have direct analogs (they are self-edits of existing files). There are no
genuinely novel file types in this phase. The connection chip (CARD-03) and Kill two-step pattern
are new behaviors, but they directly extend existing patterns (STATUS_CONFIG colorblind pattern,
existing overflow menu) with no structural novelty.

---

## Metadata

**Analog search scope:** `frontend/src/components/`, `frontend/src/components/Hub/`,
`frontend/src/lib/`, `frontend/src/components/__tests__/`, `frontend/src/App.tsx`
**Files scanned:** 14 source files read in full or via targeted range reads
**Pattern extraction date:** 2026-06-20

**Execution order implied by patterns:**

1. (Wave 0, before any implementation) Update `Sidebar.test.tsx`, `App.hub.test.tsx`,
   `style.hub.test.ts` to RED-fail on new assertions
2. Relocate `RemoteSession`/`RemotePeerSessions` types → `lib/remoteSession.ts`; update
   `remoteAdapter.ts` import → run `tsc --noEmit`
3. Thread `isRemote`, `isConnected`, `onKill`, `onOpenInBrowser`, `onBrowseFiles` props through
   HubPanel → SessionCardGrid → SessionCard (CARD-02/03/04)
4. Add CARD-03 CSS to `style.css`; add Kill/menu CSS
5. Remove `.hub__header` block from HubPanel (CARD-01)
6. Remove Remote/Sessions/New Session from Sidebar (NAV-02/03/04/05); clean up App.tsx
7. Delete `DaemonManagerPanel.tsx` and `RemoteSessionsPanel.tsx`
8. Remove `Tab.type` union members from TabBar
9. Clean dead CSS (`.hub__header`, `.hub__title`, `.hub__new-session-btn`)
10. Full `npm test` + `tsc --noEmit` gate before `/gsd:verify-work`
