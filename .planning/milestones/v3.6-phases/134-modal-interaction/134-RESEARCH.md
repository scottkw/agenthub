# Phase 134: Modal Interaction - Research

**Researched:** 2026-06-17
**Domain:** React modal overlay + xterm.js terminal embedding + shared-element animation
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Briefing modal data source:** Driven by real terminal tail (actual prompt the agent printed). Structured "agent suggests options" multi-select (#78) is deferred to #93 — agents don't emit that data today.
- **Remote modal interaction:** Reuses locked Phase 122 design (daemon proxy + join-code exchange). No new remote-access architecture (MODAL-06).
- **Re-attach Open button must be preserved:** Phase 131 added an "Open" button on Hub cards + Sessions rows (commit 08fc2be). The card-click→modal interaction must COEXIST with that button, not regress it (see memory `project_phase134_reattach_button`).
- **Colorblind-safe constraint** (user is colorblind): any status/affordance conveyed in the modal must carry non-color cues. Release-blocking; verify at source level (hex constants), not by eye. Full a11y validation is Phase 135.

### Claude's Discretion
All other implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

### Deferred Ideas (OUT OF SCOPE)
- Structured "agent suggests options" multi-select in briefing modal → deferred to issue #93.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MODAL-01 | Clicking a card expands it into a modal via a shared-element-style grow animation | UI-SPEC animation contract + CSS keyframe pattern from Phase 133 |
| MODAL-02 | Closing the modal shrinks it back into the originating card's position and restores focus | `sourceRect` bounding rect + `cardRef.current?.focus()` on unmount |
| MODAL-03 | For non-blocked sessions, the modal mounts a full interactive TerminalPanel + RelayClient | TerminalPanel props contract documented below |
| MODAL-04 | For `waiting`/needs-input sessions, modal opens a briefing view with terminal tail + respond affordance | `GetSessionTailLines` API + `RelayClient.sendInput()` for response delivery |
| MODAL-05 | Modal session is fully functional — resize, copy/paste, scrollback search all work | TerminalPanel's isActive + ResizeObserver + fitTerminal() contract |
| MODAL-06 | For a remote session requiring a cap, uses existing remote-open/join-code exchange path | `RemoteJoinCodeModal` + `remoteCapsCached` pattern from App.tsx |
</phase_requirements>

---

## Summary

Phase 134 adds the card→modal interaction to the Hub surface built in Phases 131–133. The core work is: (1) wiring an `onCardClick` prop through `SessionCard` → `SessionCardGrid` → `HubPanel` with correct click disambiguation vs the existing "Open" button; (2) building three new components — `HubModal` (shell), `HubInteractiveModal` (TerminalPanel host), `HubBriefingModal` (tail display + respond input); (3) the shared-element grow/shrink animation originating from the clicked card's bounding rect; and (4) plumbing the briefing modal's response submission through the existing `RelayClient.sendInput()` path.

All reused components (`TerminalPanel`, `RelayClient`, `RemoteJoinCodeModal`, `GetSessionTailLines`) are already present in the codebase and work correctly. The primary engineering risk is TerminalPanel lifecycle management in a transient modal (xterm.js Terminal and RelayClient are created on open, must be fully disposed on close) and ensuring `fitTerminal()` fires after the open animation completes rather than during it.

The UI-SPEC (already approved) is the authoritative design contract. All decisions below derive from it and from direct codebase inspection.

**Primary recommendation:** State-lift the modal open/close to `HubPanel` (it already owns sessions, relayPort context, and the join-code flow). Three new components in `frontend/src/components/Hub/`. CSS additions to `style.css` using only `var(--hub-*)` tokens.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Card click → modal trigger | Browser/Client | — | onClick on article element; needs DOM rect at click time |
| Modal open/close state | Frontend (HubPanel) | — | HubPanel already owns session list and relay context; hoisting higher (App.tsx) is unnecessary |
| Grow/shrink animation | Browser/Client (CSS + JS) | — | CSS keyframes + runtime transform-origin driven by card bounding rect |
| Terminal mounting (interactive modal) | Browser/Client (TerminalPanel) | — | TerminalPanel already owns xterm.js + RelayClient lifecycle |
| Terminal input relay | Browser/Client (RelayClient WS) | API/Backend (relay server) | RelayClient.sendInput() → relay WS → Go Hub.WriteInput() → PTY |
| Briefing tail data | API/Backend | Browser/Client (polling) | GetSessionTailLines Wails RPC already used by usePreviewPoller |
| Briefing response submission | Browser/Client → API/Backend | — | RelayClient.sendInput() on a fresh client scoped to the session |
| Remote cap gate | Frontend (modal gate logic) | API/Backend (ExchangeJoinCodeAtURL) | RemoteJoinCodeModal + existing remoteCapsCached pattern |
| Focus management | Browser/Client | — | cardRef.current?.focus() on modal unmount |
| Accessibility (focus trap) | Browser/Client | — | Phase 134 wires initial focus(); Phase 135 hardens trap loop |

---

## Standard Stack

### Core (all already installed — NO new packages)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/xterm` | existing | Terminal emulator | Already used by TerminalPanel — reused as-is |
| `@heroicons/react` | existing | Icons in modal header | Already used by SessionCard/HubPanel |
| React (built-in hooks) | 18.x | `useRef`, `useState`, `useEffect` | Modal state and lifecycle management |

No new npm packages are required. [VERIFIED: codebase inspection]

### Supporting (existing patterns to reuse)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `RelayClient` (internal) | n/a | WebSocket input relay | Briefing modal response submission; interactive modal mounts via TerminalPanel |
| `GetSessionTailLines` (Wails RPC) | n/a | Fetch recent terminal output | Briefing modal tail display; already used by usePreviewPoller |
| `RemoteJoinCodeModal` | n/a | Cap exchange gate for remote sessions | MODAL-06 — rendered before opening the interactive modal for remote sessions |

**Installation:** None required. [VERIFIED: codebase inspection]

---

## Package Legitimacy Audit

No external packages are introduced by this phase. All components are hand-authored.

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| (none) | — | — | — | — | — | — |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
User clicks hub-card article body
         │
         ▼
SessionCard.onCardClick(session, card.getBoundingClientRect())
         │
         ▼
HubPanel: stores { session, sourceRect } in modalState
         │
         ├── isAttentionStatus(deriveHubStatus(session))?
         │         │
         │    true (waiting/errored)     false (running/idle/stopped)
         │         │                          │
         │         ▼                          ▼
         │  HubBriefingModal           HubInteractiveModal
         │  - GetSessionTailLines()    - <TerminalPanel
         │  - tail text display          sessionId={…}
         │  - respond textarea           isActive={true}
         │  - Send → RelayClient         relayPort={…}
         │    .sendInput(text+\n)        theme={…}
         │                               pluginConfig={…} />
         │         │                          │
         └─────────┴──────────────────────────┘
                   │
              HubModal (shell)
              - overlay (position:fixed; inset:0; z-index:200)
              - modal panel (grow animation from sourceRect)
              - header strip (status icon, name, badge, close btn)
              - Escape / click-outside → close
              - on close: shrink animation → unmount
              - on unmount: cardRef.current?.focus()

Remote session path (MODAL-06):
  if (!remoteCapsCached.has(session.id)):
    show RemoteJoinCodeModal
    on cap exchange success → open HubInteractiveModal
```

### Recommended Project Structure

```
frontend/src/components/Hub/
├── HubModal.tsx                ← NEW: outer shell (overlay, animation, Escape/click-outside)
├── HubInteractiveModal.tsx     ← NEW: mounts TerminalPanel, session header strip
├── HubBriefingModal.tsx        ← NEW: tail display + respond input + Send button
├── HubPanel.tsx                ← MODIFIED: add modal state + onCardClick wiring
├── SessionCard.tsx             ← MODIFIED: add onCardClick prop + click disambiguation
├── SessionCardGrid.tsx         ← MODIFIED: thread onCardClick prop through
├── [existing Hub files]        ← UNCHANGED
```

### Pattern 1: TerminalPanel Mounting in HubInteractiveModal

TerminalPanel is the canonical way to host an xterm.js terminal. It must receive:
- `sessionId` — the session's id string
- `isActive` — `true` while modal is open; `false` when closed (triggers terminal blur)
- `relayPort` — the daemon relay port (available from App.tsx as `relayPort` state)
- `fontSize` — from a local default or the app-level font size state
- `onFontSizeChange` — no-op or wired if desired
- `theme` — `ITheme` from App.tsx's `terminalTheme`
- `pluginConfig` — `PluginSettings | null`

The TerminalPanel creates a fresh xterm.js Terminal + RelayClient on mount and disposes both on unmount. The modal's unmount is the disposal event — no explicit teardown code is needed in the modal.

**Critical: fitTerminal fires on transitionend, not on mount.** The modal uses a grow animation (220ms). TerminalPanel's `isActive` effect fires rAF loops to fit — but the container has zero or incorrect dimensions during the animation. The solution is to fire a resize after the `transitionend` event of the modal panel.

**Implementation pattern (from UI-SPEC):**
```typescript
// In HubInteractiveModal: after the modal is fully open
// HubModal calls this callback when open animation completes
function onAnimationComplete() {
  // TerminalPanel's ResizeObserver will fire automatically once
  // the container reaches its final size — BUT only if isActive=true.
  // No explicit fitTerminal call needed; ResizeObserver handles it.
}
```

[VERIFIED: codebase inspection of TerminalPanel.tsx lines 641-681 — ResizeObserver on container fires fitTerminal whenever dimensions change]

### Pattern 2: Grow/Shrink Animation via transform-origin + CSS keyframes

```typescript
// On card click: capture bounding rect
const rect = cardElement.getBoundingClientRect()
const transformOrigin = `${rect.left + rect.width / 2}px ${rect.top + rect.height / 2}px`

// Apply as inline style to the .hub-modal element
// CSS handles the actual animation via .hub-modal--enter / .hub-modal--exit classes
```

```css
/* Declared inside @media (prefers-reduced-motion: no-preference) */
@keyframes hub-modal-grow {
  from { opacity: 0; transform: scale(0.05); }
  to   { opacity: 1; transform: scale(1); }
}
@keyframes hub-modal-shrink {
  from { opacity: 1; transform: scale(1); }
  to   { opacity: 0; transform: scale(0.05); }
}
```

The `.hub-modal--enter` class triggers `hub-modal-grow` (220ms). On close, add `.hub-modal--exit`, listen for `transitionend`/`animationend`, then unmount. [VERIFIED: UI-SPEC §"Grow/Shrink Animation"; Phase 133 precedent in style.css lines 4927-4947]

### Pattern 3: Click Disambiguation (Open button vs. card body)

The article `onClick` fires for all clicks on the card, including on child buttons. The Open button must call `e.stopPropagation()` to prevent the modal from opening when the user clicks "Open". This is the same pattern already used by the `hub-card__menu-btn` and `InlineSessionName`.

```typescript
// SessionCard.tsx — existing article element gains onClick:
<article
  className="hub-card"
  onClick={(e) => {
    // Defense-in-depth: verify click did not originate from controlled children
    const target = e.target as HTMLElement
    if (target.closest('.hub-card__open')) return
    if (target.closest('.hub-card__menu-btn')) return
    if (target.closest('.hub-card__menu')) return
    if (target.closest('.InlineSessionName input')) return
    onCardClick?.(session, (e.currentTarget as HTMLElement).getBoundingClientRect())
  }}
  onKeyDown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onCardClick?.(session, (e.currentTarget as HTMLElement).getBoundingClientRect())
    }
  }}
  tabIndex={0}
  // ...existing props
>
  {/* ROW 5: Open button — must stopPropagation */}
  <button
    className="hub-card__open"
    onClick={(e) => { e.stopPropagation(); onOpenSession?.(id, name, cli) }}
  >
    Open
  </button>
```

[VERIFIED: codebase inspection of SessionCard.tsx — `onOpenSession` handler currently has no stopPropagation; this must be added]

### Pattern 4: Briefing Modal Response Submission

The briefing modal submits a text response by writing it to the session's PTY via RelayClient. The TerminalPanel is NOT mounted in the briefing modal — only the tail display and a textarea are shown. Input is sent via a standalone RelayClient created and closed for the submit operation:

```typescript
// In HubBriefingModal — send response
async function handleSend(text: string) {
  // RelayClient is wired directly: create, send, close.
  // The existing TerminalPanel pattern proves this is the correct mechanism.
  const client = new RelayClient(relayPort, session.id, {
    onOutput: () => {},  // discard output — we're only sending
    onOpen: () => {
      client.sendInput(text + '\n')
      // Close after input is queued — OPEN guard in sendInput handles race
      setTimeout(() => client.close(), 100)
    },
    onClose: () => {},
  })
}
```

Alternative: `RelayClient.sendInput()` already guards on `ws.readyState === WebSocket.OPEN`, so input sent before the WS is open is silently dropped. The `onOpen` callback approach is safer. [VERIFIED: relayClient.ts lines 119-123]

### Pattern 5: Remote Session Cap Gate (MODAL-06)

App.tsx already maintains `remoteCapsCached: Set<string>` and the `joinModalForSession` state that drives `RemoteJoinCodeModal`. The Hub modal needs to pass `remoteCapsCached` and `onExchange` down to HubPanel, OR HubPanel can call back to App.tsx to gate the modal open.

The cleanest integration: HubPanel receives two new props from App.tsx:
- `remoteCapsCached: Set<string>` — already available in App.tsx state
- `onRequestRemoteCap: (session: RemoteSessionRef) => void` — triggers the join-code flow

When `onCardClick` fires for a remote session:
1. Check `remoteCapsCached.has(session.id)` — if cached, open interactive modal immediately.
2. If not cached, call `onRequestRemoteCap(session)` — this sets App.tsx's `joinModalForSession`, which renders `RemoteJoinCodeModal`.
3. On successful exchange, `remoteCapsCached` gains the session id → `HubPanel` can auto-open the modal.

This keeps the join-code exchange in App.tsx (where `ExchangeJoinCodeAtURL` and `RegisterRemoteCap` are already wired) and avoids duplicating that logic in HubPanel. [VERIFIED: App.tsx lines 1059-1093]

### Pattern 6: Focus Return on Modal Close

```typescript
// HubModal component
const cardRef = useRef<HTMLElement | null>(null)

// On open: capture the card element reference
function openModal(session, rect, cardElement) {
  cardRef.current = cardElement
  // ...open
}

// On close: return focus
function closeModal() {
  // Play shrink animation, then on animationend:
  onClose()  // unmounts HubModal
}

// In HubModal useEffect cleanup or onClose:
useEffect(() => {
  return () => {
    cardRef.current?.focus()
  }
}, [])
```

[VERIFIED: UI-SPEC §"Focus Management"; SessionCard.tsx — article element already has `tabIndex={0}`]

### Anti-Patterns to Avoid

- **Mounting TerminalPanel before the modal animation completes:** The xterm.js terminal will compute zero columns/rows if the container has no dimensions. Set `isActive={false}` during animation and flip to `true` on `transitionend`. The ResizeObserver in TerminalPanel's `isActive` effect handles fitting once dimensions are available. Alternatively, mount with `isActive={true}` and rely on the 20-attempt rAF loop (TerminalPanel.tsx lines 648-667) — both work, but the explicit `transitionend` approach is cleaner.
- **Not calling `e.stopPropagation()` on the Open button:** The article `onClick` will fire for every click including the Open button without it. The current SessionCard code does NOT have stopPropagation on `onOpenSession` — this must be added in Phase 134.
- **Creating a new RelayClient inside HubInteractiveModal directly:** TerminalPanel already creates and manages its own RelayClient. Don't add a second client — mount TerminalPanel and let it manage the connection.
- **Inline hex colors in modal CSS:** All CSS must use `var(--hub-*)` tokens. The UI-SPEC locked this explicitly. No hardcoded hex values in `.hub-modal*` rules.
- **Missing `@media (prefers-reduced-motion: no-preference)` guards on keyframes:** Phase 133 established the pattern — all animation keyframes live at root scope, but `animation:` property assignments live inside the `no-preference` guard. See style.css lines 4927-4947 for the template.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| xterm.js terminal in modal | Custom terminal implementation | `TerminalPanel` as-is | Already handles RelayClient, addons, fit, resize, scrollback, copy/paste, search |
| Input relay to PTY | Custom WebSocket / Go RPC | `RelayClient.sendInput()` | Binary frame protocol already implemented; WS keep-alive, close, reconnect all handled |
| Terminal tail for briefing | Custom polling + rendering | `GetSessionTailLines(id, n)` Wails RPC | Already used by usePreviewPoller; returns `string[]` ready to display |
| Remote cap exchange | New remote auth flow | `RemoteJoinCodeModal` + `remoteCapsCached` pattern from App.tsx | Phase 122 implemented the full exchange + cap caching flow |
| Modal focus trap | Custom focus management | Minimal: focus first control on open; `Escape` closes; Phase 135 hardens | Phase 134 scope is initial focus only; full trap loop is Phase 135 |
| FLIP animation for modal grow | Custom position interpolation | CSS keyframes + runtime `transform-origin` | simpler than FLIP; consistent with Phase 133's animation approach |

---

## Existing Components: Deep Reference

### TerminalPanel Props Contract

```typescript
interface TerminalPanelProps {
  sessionId: string          // session id (creates new Terminal + RelayClient on change)
  isActive: boolean          // show/hide + triggers fit when true
  relayPort: number          // WS relay port: ws://127.0.0.1:{relayPort}/sessions/{id}/ws
  fontSize: number           // applied to term.options.fontSize
  onFontSizeChange: (delta: number) => void
  theme: ITheme              // xterm.js theme
  pluginConfig?: PluginSettings | null
  onWebGLContextLost?: (reason: 'context-loss' | 'software-rasterized') => void
  onRegisterSaver?: (sessionId: string, fn: (() => string) | null) => void
  onProgressChange?: (sessionId: string, state: IProgressState) => void
}
```

**Key behaviors:**
- Creates Terminal + RelayClient once per `sessionId` (mount useEffect keyed on `[sessionId]`)
- Full disposal (WebSocket closed, xterm disposed) on unmount
- `isActive=true` → rAF loop + ResizeObserver fires fitTerminal()
- `isActive=false` → terminal hidden via display:none on parent `.terminal-wrapper` (App.tsx pattern); TerminalPanel itself does NOT hide — the parent controls visibility
- RelayClient URL: `ws://127.0.0.1:${relayPort}/sessions/${sessionId}/ws`
- Input via `term.onData()` → `client.sendInput()` (MSG_INPUT binary frame)
- Resize via `term.onResize()` → `client.sendResize()` (MSG_RESIZE2 binary frame)

[VERIFIED: TerminalPanel.tsx lines 198-371]

### RelayClient API

```typescript
class RelayClient {
  constructor(port: number, sessionId: string, callbacks: RelayClientCallbacks)
  sendInput(text: string): void    // MSG_INPUT binary frame; guards ws.readyState === OPEN
  sendResize(cols: number, rows: number): void
  close(): void
}
interface RelayClientCallbacks {
  onOutput: (data: Uint8Array) => void
  onOpen?: () => void
  onClose?: () => void
}
```

[VERIFIED: relayClient.ts]

### GetSessionTailLines Wails RPC

```typescript
GetSessionTailLines(id: string, n: number): Promise<string[]>
// Returns last n lines of the session's terminal output (ANSI-stripped on the Go side)
// Already used by usePreviewPoller in HubPanel with n=4
// For briefing modal, use a larger n (e.g., 20) to show the full prompt context
```

[VERIFIED: App.d.ts line 63; HubPanel.tsx line 84]

### deriveHubStatus / isAttentionStatus

```typescript
// lib/hubStatus.ts
type HubStatus = 'running' | 'idle' | 'waiting' | 'errored' | 'stopped-ok' | 'stopped-err'
function deriveHubStatus(s: SessionInfo): HubStatus
function isAttentionStatus(status: HubStatus): boolean
// isAttentionStatus returns true for: 'waiting' | 'errored' | 'stopped-err'
// This drives modal type: true → briefing modal; false → interactive modal
```

[VERIFIED: hubStatus.ts]

### RemoteJoinCodeModal Props

```typescript
interface RemoteJoinCodeModalProps {
  remoteSession: { id: string; name: string; hostname: string }
  onExchange: (code: string) => Promise<void>  // rejects with error message on failure
  onClose: () => void
}
// Existing Escape handling, focus-on-mount, error display all built-in
```

[VERIFIED: RemoteJoinCodeModal.tsx]

### Hub CSS Token System (authoritative source of truth)

All modal CSS must use `var(--hub-*)` tokens. The tokens relevant to the modal are:

| Token | Dark Value | Light Value |
|-------|-----------|-------------|
| `--hub-surface-elevated` | `#1e2030` | `#ececf0` |
| `--hub-border` | `#292e42` | `#d1d1db` |
| `--hub-text-primary` | `#c0caf5` | `#1a1b26` |
| `--hub-text-secondary` | `#a9b1d6` | `#3a3b50` |
| `--hub-text-muted` | `#9aa5ce` | `#5c5d80` |
| `--hub-accent` | `#7aa2f7` | `#3d6fe8` |
| `--hub-preview-bg` | `#0d0e17` | `#e8e8f0` |
| `--hub-preview-text` | `#8b92b3` | `#6b6f8e` |
| `--hub-attn-border` | `#e0af68` | `#b45309` |
| `--hub-attn-icon-color` | `#e0af68` | `#b45309` |

[VERIFIED: style.css lines 4096-4194]

### Z-Index Layers

| Layer | z-index | Element |
|-------|---------|---------|
| Hub panel | 0 (flow) | `.hub` |
| Hub card menu | 20 | `.hub-card__menu` (position: absolute) |
| Hub modal overlay | 200 | `.hub-modal-overlay` (new) |
| App-level modals | 1000 | `.new-session-overlay`, `.qr-modal-overlay` |

[VERIFIED: style.css direct inspection]

### Phase 131 Open Button (must coexist)

```typescript
// SessionCard.tsx ROW 5 — currently:
{onOpenSession && session.state !== 'stopped' && (
  <div className="hub-card__row5">
    <button
      type="button"
      className="hub-card__open"
      onClick={() => onOpenSession(id, name, cli)}  // ← NO stopPropagation currently
      aria-label={`Open ${name}`}
    >
      Open
    </button>
  </div>
)}
```

Phase 134 must add `onClick={(e) => { e.stopPropagation(); onOpenSession(id, name, cli) }}` to this button. Without this, clicking "Open" will ALSO trigger the modal. [VERIFIED: SessionCard.tsx lines 368-381]

### usePreviewPoller and tail data flow

HubPanel already polls tails via `usePreviewPoller`:
- Calls `GetSessionTailLines(s.id, 4)` every 3 seconds for all local sessions
- Returns `Map<string, string[]>` keyed by session id
- Briefing modal can fetch its own one-shot tail with a larger `n` (e.g., 20 lines) since 4-line previews aren't enough for the agent's prompt

---

## Common Pitfalls

### Pitfall 1: TerminalPanel Fitted to Zero Dimensions During Animation
**What goes wrong:** TerminalPanel's isActive useEffect fires rAFs immediately on mount. During the 220ms grow animation, the modal panel is scaling from ~0 to full size. `fitTerminal()` may compute columns=2 (the minimum) if it fires before the container reaches final dimensions.
**Why it happens:** TerminalPanel's `isActive` effect starts the rAF loop as soon as `isActive=true`; the ResizeObserver also starts immediately.
**How to avoid:** Two safe options: (A) mount TerminalPanel with `isActive={false}`, then flip to `true` after the modal's open `animationend` event fires; OR (B) rely on TerminalPanel's 20-attempt rAF loop — by attempt ~13 (216ms at 60fps), the animation should be complete and dimensions stable. Option A is explicit and deterministic.
**Warning signs:** Terminal renders with very few columns (wrapping every few characters) when the modal first opens.

### Pitfall 2: Close Button Triggers Without stopPropagation on Open Button
**What goes wrong:** User clicks "Open" on a card → the card body `onClick` also fires → the Hub modal opens on top of the tab switch.
**Why it happens:** Event bubbling. The current Open button does NOT call `e.stopPropagation()`.
**How to avoid:** Add `e.stopPropagation()` to ALL non-modal click handlers on the card: Open button, menu button, drag handle, InlineSessionName input. The card body `onClick` also does defense-in-depth via `.closest()` checks.
**Warning signs:** Modal opens when user clicks "Open" or menu button.

### Pitfall 3: RelayClient Not Disposed on Modal Close
**What goes wrong:** User opens and closes the interactive modal multiple times — each open creates a new RelayClient (via TerminalPanel), but if the Terminal is not properly disposed, multiple WS connections accumulate to the same session.
**Why it happens:** TerminalPanel only disposes on React unmount. If the modal isn't truly unmounted (e.g., display:none pattern like App.tsx uses for tabs), the RelayClient persists.
**How to avoid:** The modal must UNMOUNT TerminalPanel on close (not just hide it). The grow/shrink animation plays, then the component is removed from the tree — TerminalPanel's useEffect cleanup closes the RelayClient.
**Warning signs:** Multiple simultaneous connections to the relay server for the same session; terminal output appears doubled.

### Pitfall 4: transform-origin in Fixed/Absolute Coordinates
**What goes wrong:** The grow animation originates from the wrong position, appearing to expand from (0,0) instead of the card center.
**Why it happens:** `getBoundingClientRect()` returns viewport-relative coordinates. `transform-origin` on a fixed-position element accepts viewport-relative pixel values directly (`position: fixed` elements use the viewport as their containing block). This is correct — no offset adjustment needed.
**How to avoid:** Use `transform-origin: ${rect.left + rect.width/2}px ${rect.top + rect.height/2}px` as an inline style on `.hub-modal`. No subtraction of scroll offsets needed for `position: fixed` elements.
**Warning signs:** Animation expands from top-left corner or from a point that doesn't correspond to the card position.

### Pitfall 5: Briefing Modal RelayClient Race on Fast Send
**What goes wrong:** User types a response and immediately clicks Send. If the RelayClient WS hasn't opened yet (handshake in flight), `sendInput()` is silently dropped.
**Why it happens:** `RelayClient.sendInput()` guards on `ws.readyState === WebSocket.OPEN` but does not queue input.
**How to avoid:** Send inside the `onOpen` callback, not directly after construction. Alternatively, the briefing modal can keep the RelayClient alive for the duration it's open (connect on mount, close on unmount) and send on the `onOpen` event if not yet open.
**Warning signs:** Send button appears to succeed but nothing happens in the session.

### Pitfall 6: Escape Key Conflict with Hub Card Menu
**What goes wrong:** The modal's Escape handler fires, but the Hub card's overflow menu Escape handler (in SessionCard.tsx) also fires, causing an unintended menu close on a non-open menu.
**Why it happens:** Both add `keydown` listeners to `document`. When the modal is open, the card is no longer focused, so `menuOpen` should be `false` — but the card's Escape handler runs unconditionally when `menuOpen` is true.
**How to avoid:** The modal's Escape listener should call `e.stopPropagation()` — or better, use `e.stopImmediatePropagation()` so no other `keydown` listeners fire. Check this in testing.

### Pitfall 7: HubPanel Props Threading — relayPort and theme
**What goes wrong:** HubInteractiveModal needs `relayPort`, `theme`, and `pluginConfig` from App.tsx, but HubPanel currently doesn't receive these.
**Why it happens:** HubPanel was designed without a modal in Phase 131-133. These props were only needed by TerminalPanel instances in App.tsx's tab map.
**How to avoid:** Add `relayPort?: number`, `terminalTheme?: ITheme`, and `pluginConfig?: PluginSettings | null` to `HubPanelProps`. App.tsx already passes `relayPort` to TerminalPanel instances — add the same to `<HubPanel>`.

---

## Code Examples

### Hub Modal Open State (in HubPanel)

```typescript
// In HubPanel.tsx — add modal state
interface HubModalState {
  session: SessionInfo
  sourceRect: DOMRect
}
const [modalState, setModalState] = useState<HubModalState | null>(null)

// Passed as onCardClick to SessionCardGrid → SessionCard
const handleCardClick = useCallback((session: SessionInfo, rect: DOMRect) => {
  // MODAL-06: remote session cap gate
  if (session.hostname && session.hostname !== '') {
    if (!remoteCapsCached?.has(session.id)) {
      onRequestRemoteCap?.({ id: session.id, name: session.name, hostname: session.hostname })
      return
    }
  }
  setModalState({ session, sourceRect: rect })
}, [remoteCapsCached, onRequestRemoteCap])
```

### HubModal Shell Component Structure

```typescript
// HubModal.tsx
interface HubModalProps {
  session: SessionInfo
  sourceRect: DOMRect
  isAttention: boolean
  relayPort: number
  theme: ITheme
  pluginConfig?: PluginSettings | null
  onClose: () => void
}

export function HubModal({ session, sourceRect, isAttention, relayPort, theme, pluginConfig, onClose }: HubModalProps) {
  const [phase, setPhase] = useState<'entering' | 'open' | 'exiting'>('entering')
  const cardFocusRef = useRef<HTMLElement | null>(null)

  const transformOrigin = `${sourceRect.left + sourceRect.width / 2}px ${sourceRect.top + sourceRect.height / 2}px`
  const hubStatus = deriveHubStatus(session)
  const isBriefing = isAttentionStatus(hubStatus)

  // Return focus to originating card on unmount
  useEffect(() => {
    return () => { cardFocusRef.current?.focus() }
  }, [])

  // Escape key handler
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') { e.stopImmediatePropagation(); handleClose() }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [])

  function handleClose() {
    setPhase('exiting')
    // onClose called after animation via onAnimationEnd
  }

  return (
    <div
      className={`hub-modal-overlay hub-modal-overlay--${phase}`}
      onClick={handleClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={isBriefing ? `Briefing: ${session.name} needs input` : `Session terminal: ${session.name}`}
        className={`hub-modal hub-modal--${isBriefing ? 'briefing' : 'interactive'} hub-modal--${phase}`}
        style={{ transformOrigin }}
        onClick={(e) => e.stopPropagation()}
        onAnimationEnd={() => {
          if (phase === 'entering') setPhase('open')
          if (phase === 'exiting') onClose()
        }}
      >
        {/* header strip */}
        {/* body: HubInteractiveModal or HubBriefingModal */}
      </div>
    </div>
  )
}
```

[ASSUMED: exact prop names for onRequestRemoteCap — final names at planner discretion]

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Dialog element (native) | `position: fixed` overlay div | Standard in this codebase | No `<dialog>` element — consistent with existing NewSessionModal, QuitConfirmModal patterns |
| Separate RelayClient in modal | TerminalPanel manages its own RelayClient | Phase 131+ | Modal mounts TerminalPanel as-is; no relay duplication |
| per-card xterm instance | Throttled tail snapshot for cards, live terminal only in modal | Phase 132 | Performance constraint locked; modal is the ONLY live terminal outside tabs |

**Deprecated/outdated:**
- Per-card live xterm: explicitly out of scope (REQUIREMENTS.md §"Out of Scope").

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The briefing modal's RelayClient approach (connect on mount, send in `onOpen`) is the correct mechanism for response submission — no Wails RPC exists for directly writing to a session's PTY | Standard Stack / Pattern 4 | If a `WriteToSession` RPC exists, it would be cleaner; but the RelayClient path is proven correct via TerminalPanel |
| A2 | Adding `relayPort`, `terminalTheme`, and `pluginConfig` props to HubPanel is the correct integration point (vs lifting modal state to App.tsx) | Pattern 5 / Architecture | App.tsx could own modal state; HubPanel ownership is cleaner but either is valid |
| A3 | The `onRequestRemoteCap` callback approach for MODAL-06 (HubPanel calls App.tsx which renders RemoteJoinCodeModal) is the correct split | Pattern 5 | Could alternatively pass the exchange function directly to HubPanel; current App.tsx state for join modal is at App level |

---

## Open Questions (RESOLVED)

1. **Auto-open modal after cap exchange success (MODAL-06)**
   - What we know: After `remoteCapsCached` gains the session id, HubPanel needs to open the interactive modal for that session.
   - What's unclear: Does App.tsx pass a "cap just acquired" signal, or does HubPanel re-attempt the modal open on `remoteCapsCached` changes?
   - Recommendation: After `handleModalExchange` succeeds in App.tsx, call a new `onCapAcquired(sessionId)` callback on HubPanel, which then calls `setModalState`. Alternatively, HubPanel can track a `pendingModalSessionId` and open it when `remoteCapsCached` contains it.
   - **RESOLVED (Plan 134-05):** HubPanel tracks `pendingModalSessionId` and registers `onRegisterCapAcquired`; App.tsx signals via `capAcquiredRef` + an intent discriminator after `handleModalExchange` succeeds. Remote-without-cap calls `onRequestRemoteCap` (never opens the modal directly).

2. **Font size source for the modal TerminalPanel**
   - What we know: App.tsx maintains `fontSizes[sessionId]` per-session font size state with `DEFAULT_FONT_SIZE` fallback.
   - What's unclear: Should the modal TerminalPanel use the session's tab font size (if a tab exists) or always use `DEFAULT_FONT_SIZE`?
   - Recommendation: Use `fontSizes[session.id] ?? DEFAULT_FONT_SIZE`. This requires `fontSizes` to be passed down to HubPanel (or use a default). For Phase 134, using `DEFAULT_FONT_SIZE` is acceptable and simpler.
   - **RESOLVED (Plan 134-05 T1):** Use `DEFAULT_FONT_SIZE` (value 14) for the modal TerminalPanel — simpler and sufficient for Phase 134; per-session font-size threading is out of scope.

---

## Environment Availability

Step 2.6: SKIPPED — no new external dependencies. All runtime components (relay server, Wails RPC, xterm.js, RelayClient) are already present and verified operational by Phases 131–133.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest (jsdom environment) |
| Config file | `frontend/vite.config.ts` — `test: { environment: 'jsdom', globals: true, setupFiles: ['./src/test-setup.ts'] }` |
| Quick run command | `cd frontend && pnpm test --run --reporter=verbose` |
| Full suite command | `cd frontend && pnpm test --run` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MODAL-01 | Card article onClick passes session + DOMRect to onCardClick prop | unit (source inspection) | `pnpm test --run HubModal` | ❌ Wave 0 |
| MODAL-01 | SessionCard.tsx: card body onClick calls onCardClick; Open/menu buttons do NOT | unit (source inspection) | `pnpm test --run SessionCard` | ✅ (SessionCard.test.tsx — extend) |
| MODAL-02 | HubModal: Escape key calls onClose; close button calls onClose; focus returns to originating card | unit (source inspection) | `pnpm test --run HubModal` | ❌ Wave 0 |
| MODAL-02 | shrink animation: hub-modal--exit class added on close | unit (source inspection) | `pnpm test --run HubModal` | ❌ Wave 0 |
| MODAL-03 | HubInteractiveModal mounts TerminalPanel with correct props | unit (source inspection) | `pnpm test --run HubInteractiveModal` | ❌ Wave 0 |
| MODAL-03 | isAttentionStatus(false) → HubInteractiveModal; isAttentionStatus(true) → HubBriefingModal | unit (source inspection) | `pnpm test --run HubModal` | ❌ Wave 0 |
| MODAL-04 | HubBriefingModal calls GetSessionTailLines and renders tail lines | unit (source inspection) | `pnpm test --run HubBriefingModal` | ❌ Wave 0 |
| MODAL-04 | HubBriefingModal Send button disabled when textarea empty; enabled when non-empty | unit (source inspection) | `pnpm test --run HubBriefingModal` | ❌ Wave 0 |
| MODAL-05 | HubInteractiveModal: TerminalPanel receives isActive=true while open | unit (source inspection) | `pnpm test --run HubInteractiveModal` | ❌ Wave 0 |
| MODAL-06 | Remote session without cap → onRequestRemoteCap called instead of opening modal | unit (source inspection) | `pnpm test --run HubPanel` | ✅ (HubPanel.test.tsx — extend) |
| A11Y-02 | Enter/Space on focused hub-card fires onCardClick | unit (source inspection) | `pnpm test --run SessionCard` | ✅ (extend) |
| CSS | .hub-modal-overlay has position:fixed; inset:0; z-index:200 | CSS assertion (style.css raw read) | `pnpm test --run style.hub` | ❌ Wave 0 |
| CSS | hub-modal-grow/shrink keyframes exist in style.css | CSS assertion | `pnpm test --run style.hub` | ❌ Wave 0 |
| CSS | prefers-reduced-motion guard on hub-modal animations | CSS assertion | `pnpm test --run style.hub` | ❌ Wave 0 |

**Established test pattern:** All Hub tests use source-inspection (`?raw` import or `readFileSync(style.css)`) — no JSDOM mounting of components with xterm.js (xterm requires canvas APIs absent in jsdom). New tests follow this same pattern.

**CSS test file:** The existing pattern is to add CSS assertions to a file named `style.hub.test.ts`. Phase 133 already has assertions in `frontend/src/components/__tests__/` — the new modal CSS tests go in `style.hub.test.ts` (already exists from Phase 133) or a new `style.hub.modal.test.ts`.

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm test --run --reporter=verbose 2>&1 | tail -20`
- **Per wave merge:** `cd frontend && pnpm test --run`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/components/Hub/HubModal.test.tsx` — source-inspection tests for MODAL-01, MODAL-02, MODAL-03 routing
- [ ] `frontend/src/components/Hub/HubInteractiveModal.test.tsx` — MODAL-03, MODAL-05
- [ ] `frontend/src/components/Hub/HubBriefingModal.test.tsx` — MODAL-04
- [ ] CSS assertions appended to existing `style.hub.test.ts` (or new `style.hub.modal.test.ts`) — overlay z-index, keyframe existence, reduced-motion guards

---

## Security Domain

> `security_enforcement` not explicitly set to false in config.json — treating as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Remote cap is already authenticated by Phase 122 |
| V3 Session Management | no | Session identity is the session id; no new session state |
| V4 Access Control | no | Cap check already enforced by Phase 122 join-code exchange |
| V5 Input Validation | yes | Briefing textarea input sent verbatim to PTY via RelayClient — no sanitization needed (terminal input is opaque bytes); BUT textarea maxlength should be set as a reasonable guard |
| V6 Cryptography | no | No cryptography in this phase |

### Known Threat Patterns for Modal + Terminal

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| XSS via session name in modal header | Tampering | React text content (auto-escaped) — `{session.name}` not `dangerouslySetInnerHTML` |
| Unbounded textarea input → PTY flood | DoS | Set `maxLength={4096}` on respond textarea |
| Tab index pollution (modal not trapping focus) | — | Phase 134 wires initial focus; Phase 135 hardens trap. Note: not a security issue, but a UX/a11y one |
| Modal z-index conflict (attacker-controlled overlay) | — | Not applicable in Wails desktop app context |

---

## Sources

### Primary (HIGH confidence)
- `frontend/src/components/TerminalPanel.tsx` — full props contract, lifecycle, RelayClient usage
- `frontend/src/lib/relayClient.ts` — binary framing protocol, sendInput/sendResize API
- `frontend/src/lib/hubStatus.ts` — deriveHubStatus, isAttentionStatus
- `frontend/src/components/Hub/SessionCard.tsx` — existing card structure, Open button, click handling
- `frontend/src/components/Hub/HubPanel.tsx` — modal state ownership point, usePreviewPoller
- `frontend/src/components/RemoteJoinCodeModal.tsx` — MODAL-06 reuse component
- `frontend/src/App.tsx` — relayPort state, remoteCapsCached, joinModalForSession, handleOpenSessionTab
- `frontend/src/style.css` — z-index layers, hub-attn-pulse pattern, reduced-motion template
- `.planning/phases/134-modal-interaction/134-UI-SPEC.md` — APPROVED design contract

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` — MODAL-01..06 requirement text
- `.planning/STATE.md` — locked decisions
- `.planning/phases/134-modal-interaction/134-CONTEXT.md` — locked CONTEXT decisions

### Tertiary (LOW confidence)
- None — all critical claims verified via codebase inspection.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all components verified in codebase, no new packages
- Architecture: HIGH — data flow traced through actual source files
- Pitfalls: HIGH — identified from direct source inspection of TerminalPanel lifecycle and SessionCard click handlers
- Animation pattern: HIGH — verified against Phase 133 CSS template and UI-SPEC

**Research date:** 2026-06-17
**Valid until:** 2026-07-17 (stable codebase; no upstream library changes expected)
