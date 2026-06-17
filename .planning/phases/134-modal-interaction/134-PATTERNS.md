# Phase 134: Modal Interaction - Pattern Map

**Mapped:** 2026-06-17
**Files analyzed:** 7 (3 new, 4 modified)
**Analogs found:** 7 / 7

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `frontend/src/components/Hub/HubModal.tsx` | component (modal shell) | request-response | `frontend/src/components/RemoteJoinCodeModal.tsx` | role-match |
| `frontend/src/components/Hub/HubInteractiveModal.tsx` | component (terminal host) | request-response | `frontend/src/components/TerminalPanel.tsx` (mounted-as-is) | data-flow-match |
| `frontend/src/components/Hub/HubBriefingModal.tsx` | component (form + async submit) | request-response | `frontend/src/components/RemoteJoinCodeModal.tsx` | role-match |
| `frontend/src/components/Hub/SessionCard.tsx` | component (card) | event-driven | self | exact (modification) |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | component (grid) | event-driven | self | exact (modification) |
| `frontend/src/components/Hub/HubPanel.tsx` | component (panel / state owner) | CRUD + event-driven | self | exact (modification) |
| `frontend/src/style.css` | config (CSS) | — | Phase 133 `hub-attn-pulse` block | exact |

---

## Pattern Assignments

### `HubModal.tsx` (modal shell, request-response)

**Analog:** `frontend/src/components/RemoteJoinCodeModal.tsx`

**Imports pattern** (lines 15–16 of RemoteJoinCodeModal.tsx):
```typescript
import React, { useCallback, useEffect, useRef, useState } from 'react'
// Add for HubModal:
import { deriveHubStatus, isAttentionStatus } from '../../lib/hubStatus'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import type { ITheme } from '@xterm/xterm'
import { daemon } from '../../wailsjs/go/models'
import { XMarkIcon } from '@heroicons/react/24/outline'
// + status icons from STATUS_CONFIG (copy the import block from SessionCard.tsx lines 3-15)
type PluginSettings = daemon.PluginSettings
```

**Overlay + stopPropagation pattern** (RemoteJoinCodeModal.tsx lines 89–101):
```typescript
return (
  <div
    className="hub-modal-overlay"
    onClick={handleClose}           // click-outside closes
  >
    <div
      role="dialog"
      aria-modal="true"
      aria-label={isBriefing ? `Briefing: ${session.name} needs input` : `Session terminal: ${session.name}`}
      className={`hub-modal hub-modal--${isBriefing ? 'briefing' : 'interactive'} hub-modal--${phase}`}
      style={{ transformOrigin }}   // only permitted inline style
      onClick={(e) => e.stopPropagation()}  // prevent overlay close
      onAnimationEnd={() => {
        if (phase === 'entering') setPhase('open')
        if (phase === 'exiting') onClose()
      }}
    >
      {/* header + body */}
    </div>
  </div>
)
```

**Escape key handler pattern** (RemoteJoinCodeModal.tsx lines 62–68):
```typescript
useEffect(() => {
  function handleKeyDown(e: KeyboardEvent): void {
    if (e.key === 'Escape') {
      e.stopImmediatePropagation()  // prevent Hub card menu Escape from also firing (Pitfall 6)
      handleClose()
    }
  }
  document.addEventListener('keydown', handleKeyDown)
  return () => document.removeEventListener('keydown', handleKeyDown)
}, [])  // no onClose dep — capture handleClose in closure or use ref
```

**Focus-return on unmount pattern** (derived from RemoteJoinCodeModal focus-on-mount, lines 57–59):
```typescript
// Store originating card reference; return focus on unmount
const cardFocusRef = useRef<HTMLElement | null>(null)
useEffect(() => {
  return () => { cardFocusRef.current?.focus() }
}, [])
```

**Phase state for animation** (new — derived from UI-SPEC §"Grow/Shrink Animation"):
```typescript
const [phase, setPhase] = useState<'entering' | 'open' | 'exiting'>('entering')
// 'entering' → .hub-modal--enter class → hub-modal-grow animation (220ms)
// 'open'     → animation complete; TerminalPanel isActive can flip true
// 'exiting'  → .hub-modal--exit class → hub-modal-shrink animation (180ms) → onClose()
```

**transform-origin for grow animation** (Research.md Pattern 2):
```typescript
const transformOrigin = `${sourceRect.left + sourceRect.width / 2}px ${sourceRect.top + sourceRect.height / 2}px`
// Set as inline style on the .hub-modal panel element — the only permitted inline style
```

**Modal header strip pattern** (mirrors `.new-session-modal__header` at style.css lines 751–763):
```typescript
// In HubModal JSX — header strip (both interactive and briefing):
<div className="hub-modal__header">
  <Icon className="hub-modal__status-icon" aria-hidden="true" />
  <span className="hub-modal__session-name">{session.name}</span>
  <span className="hub-card__badge">{session.cli}</span>        {/* reuse existing badge class */}
  {isLocal ? <ComputerDesktopIcon aria-hidden="true" /> : <GlobeAltIcon aria-hidden="true" />}
  <span className="hub-modal__origin-text">{originText}</span>
  {isAttention && (
    <span className="hub-modal__attn-badge">
      <BellAlertIcon aria-hidden="true" />
      <span>Needs attention</span>                               {/* non-color cue — required */}
    </span>
  )}
  <span style={{ flex: 1 }} />
  <button
    type="button"
    className="hub-modal__close"
    aria-label="Close modal"
    onClick={handleClose}
  >
    <XMarkIcon aria-hidden="true" />
  </button>
</div>
```

---

### `HubInteractiveModal.tsx` (terminal host, request-response)

**Analog:** `frontend/src/components/TerminalPanel.tsx` (mounted as-is inside this component)

**Imports pattern:**
```typescript
import React from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import type { ITheme } from '@xterm/xterm'
import { daemon } from '../../wailsjs/go/models'
import { TerminalPanel } from '../TerminalPanel'
type PluginSettings = daemon.PluginSettings
```

**TerminalPanel props contract** (TerminalPanel.tsx lines 55–85 — the full interface):
```typescript
// Props HubInteractiveModal must pass through to TerminalPanel:
interface HubInteractiveModalProps {
  session: SessionInfo
  isOpen: boolean          // maps to TerminalPanel.isActive
  relayPort: number
  fontSize: number
  theme: ITheme
  pluginConfig?: PluginSettings | null
  onFontSizeChange?: (delta: number) => void
}

// Mount pattern:
<div className="hub-modal__body hub-modal__body--interactive">
  <TerminalPanel
    sessionId={session.id}
    isActive={isOpen}      // false during 'entering' phase; true once 'open'
    relayPort={relayPort}
    fontSize={fontSize}
    onFontSizeChange={onFontSizeChange ?? (() => {})}
    theme={theme}
    pluginConfig={pluginConfig}
    // optional callbacks not needed for modal use case — omit
  />
</div>
```

**isActive timing guard** (Research.md §Pitfall 1):
```typescript
// In HubModal: pass isActive={phase === 'open'} — NOT isActive={true} immediately.
// This prevents TerminalPanel's rAF loop from computing 0-column dimensions during
// the 220ms grow animation. TerminalPanel's ResizeObserver fires once dimensions stabilize.
```

---

### `HubBriefingModal.tsx` (form + async submit, request-response)

**Analog:** `frontend/src/components/RemoteJoinCodeModal.tsx`

**Imports pattern:**
```typescript
import React, { useCallback, useEffect, useRef, useState } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { GetSessionTailLines } from '../../wailsjs/go/main/App'
import { RelayClient } from '../../lib/relayClient'
```

**Async state pattern** (RemoteJoinCodeModal.tsx lines 51–54):
```typescript
const [tailLines, setTailLines] = useState<string[] | null>(null)  // null = loading
const [responseText, setResponseText] = useState('')
const [sending, setSending] = useState(false)
const [sendError, setSendError] = useState<string | null>(null)
const respondInputRef = useRef<HTMLTextAreaElement>(null)
```

**Focus-on-mount pattern** (RemoteJoinCodeModal.tsx lines 57–59):
```typescript
useEffect(() => {
  respondInputRef.current?.focus()  // UI-SPEC: briefing modal focuses respond input on open
}, [])
```

**Tail fetch pattern** (derived from HubPanel.tsx usePreviewPoller, lines 74–86):
```typescript
// One-shot fetch on mount with n=20 (larger than the 4-line card preview)
useEffect(() => {
  GetSessionTailLines(session.id, 20)
    .then((lines) => setTailLines(lines))
    .catch(() => setTailLines([]))  // empty array → "No recent output available." copy
}, [session.id])
```

**Submit handler pattern** (RemoteJoinCodeModal.tsx lines 72–85):
```typescript
const handleSend = useCallback(async () => {
  if (sending || responseText.trim() === '') return
  setSending(true)
  setSendError(null)
  try {
    await new Promise<void>((resolve, reject) => {
      const client = new RelayClient(relayPort, session.id, {
        onOutput: () => {},        // discard — sending only
        onOpen: () => {
          client.sendInput(responseText + '\n')
          setTimeout(() => { client.close(); resolve() }, 100)
        },
        onClose: () => {},
      })
      // Guard: reject after 5s if WS never opens
      setTimeout(() => reject(new Error('timeout')), 5000)
    })
    onClose()
  } catch (e) {
    setSendError('Failed to send. Close and try again.')
    setSending(false)
  }
}, [relayPort, session.id, responseText, sending, onClose])
```

**Submit disabled state pattern** (RemoteJoinCodeModal.tsx line 70):
```typescript
// Matches: submitDisabled = pending || code.trim().length === 0
const sendDisabled = sending || responseText.trim() === ''
```

**Textarea + Send button pattern** (mirrors `.new-session-modal__args-input` and `.new-session-modal__btn--create`):
```typescript
<div className="hub-modal__respond">
  <span className="hub-modal__respond-label">RESPOND</span>
  <textarea
    ref={respondInputRef}
    className="hub-modal__respond-input"
    placeholder="Type a response…"
    value={responseText}
    onChange={(e) => setResponseText(e.target.value)}
    onKeyDown={(e) => {
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        void handleSend()
      }
    }}
    disabled={sending}
    maxLength={4096}    // Security: V5 input validation guard
    rows={3}
  />
  <div className="hub-modal__respond-footer">
    <button
      type="button"
      className="hub-modal__close-btn"
      onClick={onClose}
      disabled={sending}
    >
      Close
    </button>
    <button
      type="button"
      className="hub-modal__send-btn"
      onClick={() => void handleSend()}
      disabled={sendDisabled}
    >
      {sending ? 'Sending…' : 'Send Response'}
    </button>
  </div>
</div>
```

**Error display pattern** (RemoteJoinCodeModal.tsx lines 140–149):
```typescript
{sendError !== null && (
  <p className="hub-modal__error-banner" role="alert">
    {sendError}
  </p>
)}
```

---

### `SessionCard.tsx` (modification — add `onCardClick` prop + click disambiguation)

**Analog:** self (lines 84–386)

**New prop addition** (insert into `SessionCardProps` interface at lines 84–102):
```typescript
export interface SessionCardProps {
  // ... existing props ...
  /**
   * Phase 134 — fires when card body is clicked (not Open/menu/drag-handle/rename input).
   * Receives the session and the card's bounding rect (used for grow animation origin).
   */
  onCardClick?: (session: SessionInfo, rect: DOMRect) => void
}
```

**article onClick + defense-in-depth** (modify the `<article>` element at lines 220–236):
```typescript
<article
  className={[
    'hub-card',
    hubStatus === 'stopped-ok' ? 'hub-card--dim' : '',
    isDragging ? 'hub-card--dragging' : '',
    isAttention ? 'hub-card--attention' : '',
  ].filter(Boolean).join(' ')}
  draggable="true"
  onDragStart={(e) => {
    e.dataTransfer.setData('text/plain', memberKeyForSession)
    e.dataTransfer.effectAllowed = 'move'
    setIsDragging(true)
  }}
  onDragEnd={() => setIsDragging(false)}
  onClick={(e) => {
    // Defense-in-depth: verify click did not originate from controlled children
    const target = e.target as HTMLElement
    if (target.closest('.hub-card__open')) return
    if (target.closest('.hub-card__menu-btn')) return
    if (target.closest('.hub-card__menu')) return
    if (target.closest('.InlineSessionName input')) return
    if (isDragging) return
    onCardClick?.(session, (e.currentTarget as HTMLElement).getBoundingClientRect())
  }}
  onKeyDown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onCardClick?.(session, (e.currentTarget as HTMLElement).getBoundingClientRect())
    }
  }}
  aria-label={cardAriaLabel}
  tabIndex={0}
>
```

**Open button `stopPropagation` fix** (modify lines 369–381 — CRITICAL, currently missing):
```typescript
{onOpenSession && session.state !== 'stopped' && (
  <div className="hub-card__row5">
    <button
      type="button"
      className="hub-card__open"
      onClick={(e) => { e.stopPropagation(); onOpenSession(id, name, cli) }}  // ADD stopPropagation
      aria-label={`Open ${name}`}
    >
      Open
    </button>
  </div>
)}
```

**Menu button already uses `onClick` without stopPropagation** — it must also get it (lines 247–257):
```typescript
<button
  ref={menuBtnRef}
  type="button"
  className="hub-card__menu-btn"
  aria-label={`Card options for ${name}`}
  aria-expanded={menuOpen}
  aria-haspopup="menu"
  onClick={(e) => { e.stopPropagation(); setMenuOpen((p) => !p) }}  // ADD stopPropagation
>
```

---

### `SessionCardGrid.tsx` (modification — thread `onCardClick` prop)

**Analog:** self (lines 130–298)

**New prop addition** (insert into `SessionCardGridProps` interface at lines 130–147):
```typescript
export interface SessionCardGridProps {
  // ... existing props ...
  /** Phase 134 — threaded to each SessionCard for modal open trigger */
  onCardClick?: (session: SessionInfo, rect: DOMRect) => void
}
```

**Thread to SessionCard** (both render paths — named groups at line 239 and workDir groups at line 283):
```typescript
<SessionCard
  session={s}
  onRename={onRename}
  onOpenSession={onOpenSession}
  onCardClick={onCardClick}          // ADD this prop
  previewLines={previewTails?.get(s.id)}
  groupDefs={groupDefs}
  onAssignGroup={onAssignGroup}
  isAttention={attentionIds?.has(s.id)}
/>
```

---

### `HubPanel.tsx` (modification — modal state + wiring)

**Analog:** self (lines 130–376)

**New props** (insert into `HubPanelProps` interface at lines 130–145):
```typescript
export interface HubPanelProps {
  // ... existing props ...
  /** Phase 134 — relay port for mounting TerminalPanel inside the interactive modal */
  relayPort?: number
  /** Phase 134 — xterm.js theme passed to HubInteractiveModal */
  terminalTheme?: ITheme
  /** Phase 134 — plugin config passed to HubInteractiveModal */
  pluginConfig?: PluginSettings | null
  /** Phase 134 — MODAL-06: cap set for remote sessions; checked before opening modal */
  remoteCapsCached?: Set<string>
  /** Phase 134 — MODAL-06: triggers RemoteJoinCodeModal flow for uncapped remote sessions */
  onRequestRemoteCap?: (session: { id: string; name: string; hostname: string }) => void
}
```

**Modal state** (insert after existing useState declarations at ~line 180):
```typescript
// Phase 134 — modal state: null = closed; non-null = modal open for this session+rect
interface HubModalState {
  session: SessionInfo
  sourceRect: DOMRect
}
const [modalState, setModalState] = useState<HubModalState | null>(null)
```

**handleCardClick callback** (follows the pattern of existing handleAssignGroup at lines 298–302):
```typescript
const handleCardClick = useCallback((session: SessionInfo, rect: DOMRect) => {
  // MODAL-06: remote session cap gate
  const isRemote = session.hostname && session.hostname !== ''
  if (isRemote && !remoteCapsCached?.has(session.id)) {
    onRequestRemoteCap?.({ id: session.id, name: session.name, hostname: session.hostname })
    return
  }
  setModalState({ session, sourceRect: rect })
}, [remoteCapsCached, onRequestRemoteCap])
```

**HubModal render** (append inside HubPanel return, after the main `<div className="hub">` body):
```typescript
{modalState && relayPort !== undefined && (
  <HubModal
    session={modalState.session}
    sourceRect={modalState.sourceRect}
    relayPort={relayPort}
    theme={terminalTheme ?? {}}
    pluginConfig={pluginConfig}
    onClose={() => setModalState(null)}
  />
)}
```

**Thread onCardClick to SessionCardGrid** (modify the SessionCardGrid call at lines 322–331):
```typescript
<SessionCardGrid
  sessions={visibleSessions}
  onRename={onRename}
  onOpenSession={onOpenSession}
  onCardClick={handleCardClick}        // ADD this prop
  groupDefs={groupDefs.length > 0 ? groupDefs : undefined}
  previewTails={previewTails}
  onAssignGroup={handleAssignGroup}
  attentionIds={attentionIds}
  debouncedSortKey={debouncedSortKey}
/>
```

---

### `style.css` (CSS additions — modal rules + animation keyframes)

**Analog:** Phase 133 `hub-attn-pulse` block (style.css lines 4926–4956)

**Overlay pattern** (mirrors `.new-session-overlay` at lines 731–739, but z-index 200):
```css
.hub-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;                           /* above Hub (0) + card menu (20); below app modals (1000) */
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.6);
}
```

**Modal panel pattern** (mirrors `.new-session-modal` structure at lines 740–750, but larger):
```css
.hub-modal {
  position: relative;
  width: min(1100px, calc(100vw - 48px));
  height: min(750px, calc(100vh - 64px));
  background: var(--hub-surface-elevated);
  border: 1px solid var(--hub-border);
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
```

**Animation keyframes — inside `@media (prefers-reduced-motion: no-preference)` guard** (mirrors Phase 133 hub-attn-pulse pattern at lines 4927–4930):
```css
/* hub-modal-grow/shrink keyframes declared at root scope — only fire under no-preference guard */
@keyframes hub-modal-grow {
  from { opacity: 0; transform: scale(0.05); }
  to   { opacity: 1; transform: scale(1); }
}
@keyframes hub-modal-shrink {
  from { opacity: 1; transform: scale(1); }
  to   { opacity: 0; transform: scale(0.05); }
}
@keyframes hub-modal-overlay-in {
  from { opacity: 0; }
  to   { opacity: 1; }
}
@keyframes hub-modal-overlay-out {
  from { opacity: 1; }
  to   { opacity: 0; }
}

/* Animation CLASS assignments — inside no-preference guard (template: lines 4927-4939) */
@media (prefers-reduced-motion: no-preference) {
  .hub-modal--entering {
    animation: hub-modal-grow 220ms cubic-bezier(0.25, 0.46, 0.45, 0.94) forwards;
  }
  .hub-modal--exiting {
    animation: hub-modal-shrink 180ms cubic-bezier(0.55, 0, 1, 0.45) forwards;
  }
  .hub-modal-overlay--entering {
    animation: hub-modal-overlay-in 220ms ease forwards;
  }
  .hub-modal-overlay--exiting {
    animation: hub-modal-overlay-out 180ms ease forwards;
  }
}
```

**Reduced-motion fallback — mandatory** (mirrors lines 4950–4956):
```css
/* Reduced-motion: modal appears/disappears instantly — no animation */
@media (prefers-reduced-motion: reduce) {
  .hub-modal {
    animation: none;
    transition: none;
    opacity: 1;
    transform: none;
  }
  .hub-modal-overlay {
    animation: none;
    transition: none;
    opacity: 1;
  }
}
```

**Input focus pattern** (mirrors `.new-session-modal__args-input:focus` at style.css line 921–923):
```css
.hub-modal__respond-input:focus {
  border-color: var(--hub-accent);
  outline: none;
}
```

**Send button pattern** (mirrors `.new-session-modal__btn--create` at style.css lines 963–977):
```css
.hub-modal__send-btn {
  background: var(--hub-accent);
  color: var(--hub-bg);
  font-weight: 600;
  font-size: 14px;
  padding: 8px 16px;
  border-radius: 4px;
  border: none;
  cursor: pointer;
}
.hub-modal__send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
```

---

## Shared Patterns

### Overlay + click-outside dismiss
**Source:** `frontend/src/components/RemoteJoinCodeModal.tsx` lines 89–101
**Apply to:** `HubModal.tsx`
```typescript
// Outer div: onClick={handleClose}
// Inner div: onClick={(e) => e.stopPropagation()}
```

### Escape key via document listener
**Source:** `frontend/src/components/RemoteJoinCodeModal.tsx` lines 62–68
**Apply to:** `HubModal.tsx`
```typescript
useEffect(() => {
  function handleKeyDown(e: KeyboardEvent): void {
    if (e.key === 'Escape') onClose()
  }
  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [onClose])
```
Note: HubModal should use `e.stopImmediatePropagation()` instead of plain close to prevent
the Hub card's own Escape handler from double-firing (Research.md §Pitfall 6).

### role="dialog" + aria-modal ARIA contract
**Source:** `frontend/src/components/RemoteJoinCodeModal.tsx` lines 96–99
**Apply to:** `HubModal.tsx`
```typescript
role="dialog"
aria-modal="true"
aria-label={...}   // HubModal uses aria-label (not aria-labelledby) per UI-SPEC copywriting contract
```

### `var(--hub-*)` token-only CSS
**Source:** `frontend/src/style.css` lines 4096–4194 (Hub token block)
**Apply to:** All CSS rules in the `.hub-modal*` namespace
No hardcoded hex values permitted in any `.hub-modal*` or `.hub-briefing*` rule.

### Submit disabled state
**Source:** `frontend/src/components/RemoteJoinCodeModal.tsx` line 70
**Apply to:** `HubBriefingModal.tsx` Send button
```typescript
const sendDisabled = sending || responseText.trim() === ''
// → disabled={sendDisabled} + opacity:0.5 / cursor:not-allowed via CSS
```

### Wails RPC mock in test setup
**Source:** `frontend/src/components/Hub/HubPanel.test.tsx` lines 7–9
**Apply to:** All new Hub test files (`HubModal.test.tsx`, `HubInteractiveModal.test.tsx`, `HubBriefingModal.test.tsx`)
```typescript
vi.mock('../../wailsjs/go/main/App', () => ({
  GetSessionTailLines: vi.fn().mockResolvedValue(['line1', 'line2']),
  // add others as needed
}))
```

### Source-inspection test approach (no xterm/JSDOM mounting)
**Source:** `frontend/src/components/__tests__/style.hub.test.ts` lines 1–9
**Source:** `frontend/src/components/__tests__/App.hub.test.tsx` lines 1–5
**Apply to:** All new component tests for Phase 134
```typescript
// For CSS assertions: readFileSync
import { readFileSync } from 'fs'
import { resolve } from 'path'
const cssRaw = readFileSync(resolve(__dirname, '../../../style.css'), 'utf-8')

// For component source assertions: ?raw import
import raw from '../HubModal.tsx?raw'
describe('HubModal', () => {
  it('uses role="dialog"', () => {
    expect(raw).toContain('role="dialog"')
  })
})
```

### makeSession test factory
**Source:** `frontend/src/components/Hub/HubPanel.test.tsx` lines 22–38
**Apply to:** All new Hub modal test files
```typescript
function makeSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'sess-1', cli: 'claude', name: 'Test Session',
    state: 'running', status: 'running',
    createdAt: new Date().toISOString(),
    hostname: '', webEnabled: false, viewerCount: 0,
    homeDir: false, filesWrite: false, workDir: '/home/user/project',
    ...overrides,
  }
}
```

---

## No Analog Found

No files in Phase 134 are entirely without analog. The grow/shrink animation is novel but closely templated on the Phase 133 `hub-attn-pulse` / reduced-motion pattern.

| File | Role | Data Flow | Note |
|---|---|---|---|
| (none) | — | — | All files have close analogs in the codebase |

---

## CSS Test Pattern (for new style.hub.modal.test.ts)

**Source:** `frontend/src/components/__tests__/style.hub.test.ts` lines 1–12 and lines 176–197 (reduced-motion contract test)

Planner should create `style.hub.modal.test.ts` (or append to `style.hub.test.ts`) using this block-finder pattern:

```typescript
// Block-finder helper pattern from style.hub.test.ts lines 118-127:
it('.hub-modal-overlay has position:fixed; inset:0; z-index:200', () => {
  const idx = cssRaw.indexOf('.hub-modal-overlay')
  expect(idx).toBeGreaterThan(-1)
  const blockEnd = cssRaw.indexOf('}', idx)
  const block = cssRaw.slice(idx, blockEnd)
  expect(block).toContain('position: fixed')
  expect(block).toContain('inset: 0')
  expect(block).toContain('z-index: 200')
})

it('hub-modal-grow keyframe is declared', () => {
  expect(cssRaw).toContain('@keyframes hub-modal-grow')
})

it('hub-modal animation is inside prefers-reduced-motion: no-preference guard', () => {
  const mediaIdx = cssRaw.indexOf('prefers-reduced-motion: no-preference')
  expect(mediaIdx).toBeGreaterThan(-1)
  const enterIdx = cssRaw.indexOf('hub-modal--entering')
  expect(enterIdx).toBeGreaterThan(mediaIdx)
})

it('prefers-reduced-motion: reduce block sets animation:none on .hub-modal', () => {
  const reduceIdx = cssRaw.lastIndexOf('prefers-reduced-motion: reduce')
  expect(reduceIdx).toBeGreaterThan(-1)
  const blockEnd = cssRaw.indexOf('}', reduceIdx + 50)
  const block = cssRaw.slice(reduceIdx, blockEnd + 50)
  expect(block).toContain('animation: none')
})
```

---

## Metadata

**Analog search scope:** `frontend/src/components/`, `frontend/src/components/Hub/`, `frontend/src/lib/`, `frontend/src/style.css`
**Files scanned:** 10 source files + 5 test files + style.css
**Key discoveries:**
1. `SessionCard.tsx` line 374: `onClick={() => onOpenSession(id, name, cli)}` — MISSING `stopPropagation`. Must be fixed in Phase 134.
2. `SessionCard.tsx` line 250: `onClick={() => setMenuOpen((p) => !p)}` on menu button — ALSO missing `stopPropagation`. Must be fixed.
3. No existing grow/shrink shared-element animation in the codebase — Phase 133 `hub-attn-pulse` is the closest template for the `@media` guard pattern.
4. Tests in `frontend/src/components/Hub/` directory (not `__tests__/`) — new Hub modal tests go in `frontend/src/components/Hub/` alongside the components.
5. `TerminalPanel` is never mounted in any test (xterm requires canvas APIs absent in jsdom) — all new modal tests must use source inspection (`?raw` import), not DOM rendering.
**Pattern extraction date:** 2026-06-17
