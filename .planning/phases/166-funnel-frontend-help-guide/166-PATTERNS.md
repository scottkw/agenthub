# Phase 166: Funnel Frontend + Help Guide — Pattern Map

**Mapped:** 2026-06-30
**Files analyzed:** 8 (7 modified + 1 new)
**Analogs found:** 8 / 8

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/wailsjs/go/main/App.d.ts` | config/stub | request-response | itself (existing stub pattern) | self-extension |
| `frontend/src/wailsjs/go/main/App.js` | config/stub | request-response | itself (existing stub pattern) | self-extension |
| `frontend/src/components/Hub/SessionShareModal.tsx` | component | request-response | itself — `SetSessionBrowse` toggle pattern (lines 232–256) | self-extension |
| `frontend/src/components/SessionSharePanel.tsx` | component | request-response | itself — `CodeDisplay` + QR pattern (lines 1–57, 99–175) | self-extension |
| `frontend/src/components/Hub/SessionCard.tsx` | component | event-driven | `STATUS_CONFIG` badge (lines 36–61) + `BellAlertIcon` attn badge | self-extension |
| `frontend/src/components/TabBar.tsx` | component | event-driven | `sessionStatuses` per-session prop pattern (lines 52, 224–228) | self-extension |
| `frontend/src/components/HelpTab.tsx` + `HelpSectionNav.tsx` | component | transform | itself — `SECTION_META` / `SECTIONS` arrays (lines 61–65 / 10–14) | self-extension |
| `frontend/src/content/help/sharing-guide.md` | content | — | `frontend/src/content/help/chat.md` | role-match |
| `frontend/src/style.css` | config | — | existing `--hub-attn-*` token group | role-match |

---

## Pattern Assignments

---

### Wave 0: `frontend/src/wailsjs/go/main/App.d.ts` (stub extension)

**Analog:** the file itself — existing hand-authored stub

**Existing `SessionInfo` interface pattern** (lines 6–24):
```ts
export interface SessionInfo {
  id: string
  cli: string
  name: string
  state: string
  status: string
  createdAt: string
  hostname: string
  webEnabled: boolean
  viewerCount: number
  exitCode?: number
  duration?: number
  /** Phase 124 / CAP-06: ... */
  homeDir: boolean
  /** Phase 137 / SHARE-03: ... */
  browseEnabled: boolean
  /** Phase 131 / GRID-02: ... */
  workDir: string
}
```

**Add after `browseEnabled`** — follow JSDoc + phase-tag comment style:
```ts
/** Phase 165 / FNL-01: true when Tailscale Funnel is active. NOT omitempty — false must serialize. */
funnelActive: boolean
```

**Existing function export pattern** (lines 201–207):
```ts
export function IssueCapabilities(sessionID: string): Promise<IssueCapabilitiesResponse>
export function GetCapabilityQRCode(joinURL: string): Promise<string>
export function SetSessionBrowse(sessionID: string, enabled: boolean): Promise<void>
```

**Add at end** — match phase-tag comment + typed-args style:
```ts
// Phase 165 / FNL-01 — Funnel on/off; expiresIn=0 means no auto-expiry (FNL-07).
export function SetSessionFunnel(sessionID: string, enabled: boolean, expiresIn: number): Promise<void>
```

---

### Wave 0: `frontend/src/wailsjs/go/main/App.js` (stub extension)

**Analog:** the file itself — existing hand-authored JS stub

**Existing Wails Call pattern** (lines 100–129):
```js
export const GetCapabilityQRCode     = (joinURL)            => Call('main.App.GetCapabilityQRCode', [joinURL])
export const SetSessionBrowse        = (sessionID, enabled) => Call('main.App.SetSessionBrowse', [sessionID, enabled])
```

**Add at end** — match positional-arg arrow style:
```js
// Phase 165 / FNL-01 — Funnel enable/disable with auto-expiry (expiresIn=0 = no expiry).
export const SetSessionFunnel = (sessionID, enabled, expiresIn) => Call('main.App.SetSessionFunnel', [sessionID, enabled, expiresIn])
```

---

### Wave 1: `frontend/src/components/Hub/SessionShareModal.tsx` (Funnel toggle + risk panel)

**Analog:** the file itself — `SetSessionBrowse` toggle handler (lines 232–256) and the `ShellWebShareBanner` inline-conditional pattern (lines 302–317)

**Imports pattern** (lines 1–14) — add `SetSessionFunnel` to the wailsjs import block:
```ts
import {
  IssueCapabilities,
  ToggleWebServing,
  SetSessionBrowse,
  SetSessionFunnel,           // Phase 166 / FNL-01 (add here)
  GetLocalNetworkPassword,
} from '../../wailsjs/go/main/App'
```

**`ShareSession` interface extension** (lines 18–25) — add `funnelActive`:
```ts
interface ShareSession {
  id: string
  name: string
  cli: string
  webEnabled: boolean
  homeDir: boolean
  browseEnabled: boolean
  funnelActive: boolean        // Phase 166 / FNL-01 — seeds Internet section state
}
```

**`SetSessionBrowse` toggle handler pattern to mirror for `SetSessionFunnel`** (lines 232–256):
```ts
async function handleBrowseToggle(): Promise<void> {
  if (!shareEnabled) return
  const next = !browseEnabled
  try {
    await SetSessionBrowse(session.id, next)
    setBrowseEnabled(next)
    if (shareEnabled) {
      try {
        const resp = await IssueCapabilities(session.id)
        setCachedShare({ readURL: resp.readUrl, writeURL: resp.writeUrl, ... })
      } catch {
        setCachedShare(null)
      }
    }
  } catch {
    // revert — state unchanged
  }
}
```

**Funnel toggle → risk panel pattern** — follows `pendingShellShare` conditional inline block (lines 302–317):
```ts
// Existing pattern: toggle intercept → banner shown inline
{pendingShellShare && (
  <ShellWebShareBanner
    variant="block"
    onConfirm={async () => { setPendingShellShare(false); ... }}
    onCancel={() => { setPendingShellShare(false); ... }}
  />
)}
```
The risk panel uses the same conditional-render-inside-modal-body pattern:
```ts
{riskPanelOpen && (
  <div className="hub-funnel-risk-panel">
    {/* warning block, expiry select, help link, action buttons */}
  </div>
)}
```

**Disabled toggle pattern** (lines 357–380) — `webServerMode` gate for local-fallback disable:
```tsx
<div
  aria-disabled={!shareEnabled}
  style={!shareEnabled ? { opacity: 0.6, pointerEvents: 'none' } : undefined}
>
  <label
    className={`settings-panel__toggle-row${browseEnabled ? ' settings-panel__toggle-row--checked' : ''}`}
    style={{ cursor: shareEnabled ? 'pointer' : 'not-allowed' }}
  >
    <input
      type="checkbox"
      className="settings-panel__toggle-input"
      role="switch"
      aria-checked={browseEnabled}
      disabled={!shareEnabled}
      ...
    />
```
Apply same pattern for the Funnel toggle when `webServerMode !== 'tailscale'`:
```tsx
<div
  aria-disabled={webServerMode !== 'tailscale'}
  style={webServerMode !== 'tailscale' ? { opacity: 0.6, pointerEvents: 'none' } : undefined}
>
  {/* Funnel toggle label — aria-disabled, disabled attribute */}
  {webServerMode !== 'tailscale' && (
    <p className="..." style={{ fontSize: 12.5, color: 'var(--hub-text-muted)' }}>
      Internet sharing requires Tailscale
    </p>
  )}
</div>
```

**Animation / motion guard pattern** (lines 81–84):
```ts
const prefersReducedMotion =
  typeof window !== 'undefined' &&
  typeof window.matchMedia === 'function' &&
  window.matchMedia('(prefers-reduced-motion: reduce)').matches
```
Apply the same inline `matchMedia` check as the guard for the risk panel's CSS transition; no separate hook needed.

**`IssueCapabilities` re-issue after Funnel enable** (lines 183–199 seeding effect):
```ts
let cancelled = false
void (async () => {
  try {
    const resp = await IssueCapabilities(session.id)
    if (cancelled) return
    setCachedShare({
      readURL: resp.readUrl,
      writeURL: resp.writeUrl,
      readCode: resp.readCode,
      writeCode: resp.writeCode,
    })
  } catch {
    setCachedShare(null)
  }
})()
return () => { cancelled = true }
```
The Funnel warm-up confirm handler calls `IssueCapabilities` the same way after `funnelActive` flips true.

---

### Wave 2: `frontend/src/components/SessionSharePanel.tsx` (Internet public section)

**Analog:** the file itself — `CodeDisplay` and QR toggle pattern (lines 1–57, 115–175)

**Import pattern** (lines 1–3) — already correct; add `SetSessionFunnel` if disable button lives here:
```ts
import { GetCapabilityQRCode } from '../wailsjs/go/main/App'
import { BrowserOpenURL, ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'
```

**Props interface extension** (lines 70–83):
```ts
interface SessionSharePanelProps {
  sessionId: string
  readURL: string
  writeURL: string
  readCode: string
  writeCode: string
  browseEnabled?: boolean
  // Phase 166 / FUI-04/05 additions:
  funnelActive?: boolean        // true when Funnel is active
  funnelUrl?: string            // Funnel-base readURL (from re-issued IssueCapabilities)
  warmingUp?: boolean           // true immediately after enable, before funnelActive flips
  onDisableFunnel?: () => void  // calls SetSessionFunnel(id, false, 0) from parent
}
```

**URL copy pattern** (lines 115–124) — `ClipboardSetText` + 1500 ms reset:
```ts
async function handleCopy(url: string, setter: (v: boolean) => void): Promise<void> {
  try {
    await ClipboardSetText(url)
  } catch {
    return
  }
  setter(true)
  setTimeout(() => setter(false), 1500)
}
```
The Funnel URL "Copy URL" button uses this exact handler. Do NOT use `navigator.clipboard`.

**QR toggle pattern** (lines 134–173) — `GetCapabilityQRCode(joinURLFor(url, code))`:
```ts
async function handleToggleQR(which: 'read' | 'write'): Promise<void> {
  // ...
  try {
    const b64 = await GetCapabilityQRCode(joinURLFor(readURL, readCode))
    setReadQRb64(b64)
  } catch {
    setQrError('QR unavailable — tap to retry')
    return
  }
  setShowReadQR(true)
}
```
The Funnel URL section reuses this pattern. The Funnel URL is a direct URL (not a join-code exchange URL), so it may be passed directly to `GetCapabilityQRCode` — confirm with the warm-up URL derivation from `IssueCapabilities`.

**Section divider pattern** — the Internet section sits below existing rows with:
```tsx
<div className="hub-share-internet-section">
  <span className="hub-share-internet-section__heading">Internet (public)</span>
  {warmingUp && (
    <p className="hub-share-internet-section__warmup">Starting up… (TLS warming up)</p>
  )}
  {!warmingUp && funnelActive && funnelUrl && (
    <>
      {/* URL row: copy + open + QR — same action button pattern as lines 180-202 */}
      <div className="session-share-panel__link-row">
        <span className="session-share-panel__label">Public URL (read-only):</span>
        <span className="session-share-panel__url" title={funnelUrl}>{funnelUrl}</span>
        <div className="session-share-panel__actions">
          <button className="daemon-panel__btn" onClick={...}>Copy URL</button>
          <button className="daemon-panel__btn" onClick={() => BrowserOpenURL(funnelUrl)}>Open</button>
          <button className="daemon-panel__btn" onClick={...}>QR</button>
        </div>
      </div>
      {/* QR display: same img pattern as lines 211-219 */}
      {showFunnelQR && funnelQRb64 && (
        <img src={`data:image/png;base64,${funnelQRb64}`} width={200} height={200} alt="QR code for public URL" />
      )}
      <button className="hub-share-internet-section__disable" onClick={onDisableFunnel}>
        Disable internet share
      </button>
    </>
  )}
  {warmupTimedOut && (
    <p className="hub-share-internet-section__error">Connection timed out. Try disabling and re-enabling.</p>
  )}
</div>
```

---

### Wave 2: `frontend/src/components/Hub/SessionCard.tsx` (internet badge)

**Analog:** the file itself — `STATUS_CONFIG` colorblind-safe badge pattern (lines 36–61) and the `BellAlertIcon` attn badge pattern

**Existing colorblind-safe comment + icon pattern** (lines 36–50):
```ts
// COLORBLIND-SAFE: every status has unique icon shape + text label; color is reinforcement only.
// Hex values are authoritative source of truth — verify at source, not by eye (user is colorblind).
const STATUS_CONFIG: Record<HubStatus, { Icon: ...; label: string; spin: boolean }> = {
  /* COLORBLIND-SAFE: status dot dark hex #3b82f6 (running) — reinforcement only; ArrowPathIcon carries the state */
  running: { Icon: ArrowPathIcon, label: 'Running', spin: true },
  ...
}
```

**`GlobeAltIcon` is already imported** (line 13) — no new import needed.

**New badge inline render** — follows the existing session status + attn badge placement in the card header row:
```tsx
{/* Phase 166 / FUI-03 — internet exposure badge.
    COLORBLIND-SAFE: GlobeAltIcon shape + "INTERNET" text carry state; color is reinforcement only.
    Dark hex #43ddb2 / light #0d7a5c — verify at source, NOT by eye (user is colorblind). */}
{session.funnelActive && (
  <span className="hub-internet-badge">
    <GlobeAltIcon className="hub-internet-badge__icon" aria-hidden="true" />
    <span className="hub-internet-badge__label">INTERNET</span>
  </span>
)}
```

---

### Wave 2: `frontend/src/components/TabBar.tsx` (tab internet icon)

**Analog:** the file itself — `sessionStatuses` prop pattern (lines 52, 224–228)

**Existing per-session prop pattern** (lines 52, 224–228):
```ts
// Prop declaration:
sessionStatuses?: Record<string, string>

// Usage in render:
<span
  className={`tab__status tab__status--${sessionStatuses?.[tab.sessionId] || 'running'}`}
  title={sessionStatuses?.[tab.sessionId] || 'running'}
/>
```

**New prop** — add parallel to `sessionStatuses`:
```ts
funnelActiveSessions?: Record<string, boolean>
```

**App.tsx derivation** (rides existing `hubSessions` state, App.tsx:984 poll):
```ts
const funnelActiveSessions = hubSessions.reduce<Record<string, boolean>>(
  (acc, s) => ({ ...acc, [s.id]: s.funnelActive }),
  {}
)
// pass as: <TabBar funnelActiveSessions={funnelActiveSessions} ... />
```

**New globe icon render** — add inside the tab render block after the `badgeClass` span (line 228), before the tab name:
```tsx
{/* Phase 166 / FUI-03 — tab internet exposure indicator.
    COLORBLIND-SAFE: GlobeAltIcon shape carries state; color is reinforcement only.
    Text label is in aria-label + title — not rendered visually (D-09). */}
{funnelActiveSessions?.[tab.sessionId] && (
  <GlobeAltIcon
    className="tab__internet-icon"
    aria-label="Internet exposed"
    title="Internet exposed"
    aria-hidden={false}
  />
)}
```
`GlobeAltIcon` must be imported from `@heroicons/react/24/outline` (not yet imported in TabBar.tsx — add import).

---

### Wave 3: `frontend/src/components/HelpTab.tsx` (SECTION_META addition)

**Analog:** the file itself — `SECTION_META` array (lines 61–65) and `?raw` markdown import (lines 12–14)

**Existing import pattern** (lines 12–14):
```ts
import gettingStartedMd from '../content/help/getting-started.md?raw'
import chatMd from '../content/help/chat.md?raw'
import faqMd from '../content/help/faq.md?raw'
```
**Add**:
```ts
import sharingMd from '../content/help/sharing-guide.md?raw'
```

**Existing SECTION_META array** (lines 61–65):
```ts
const SECTION_META: ReadonlyArray<{ id: string; label: string; markdown: string }> = [
  { id: 'help-getting-started', label: 'Getting Started', markdown: gettingStartedMd },
  { id: 'help-chat', label: 'Chat', markdown: chatMd },
  { id: 'help-faq', label: 'Frequently Asked Questions', markdown: faqMd },
]
```
**After `help-chat`, before `help-faq`**:
```ts
const SECTION_META: ReadonlyArray<{ id: string; label: string; markdown: string }> = [
  { id: 'help-getting-started', label: 'Getting Started', markdown: gettingStartedMd },
  { id: 'help-chat', label: 'Chat', markdown: chatMd },
  { id: 'help-sharing', label: 'Sharing Outside Your Tailnet', markdown: sharingMd },
  { id: 'help-faq', label: 'Frequently Asked Questions', markdown: faqMd },
]
```

---

### Wave 3: `frontend/src/components/HelpSectionNav.tsx` (SECTIONS addition)

**Analog:** the file itself — `SECTIONS` const array (lines 10–14)

**Existing SECTIONS array** (lines 10–14):
```ts
export const SECTIONS = [
  { id: 'help-getting-started', label: 'Getting Started' },
  { id: 'help-chat', label: 'Chat' },
  { id: 'help-faq', label: 'Frequently Asked Questions' },
] as const
```
**After `help-chat`, before `help-faq`** — must stay in sync with `SECTION_META` above:
```ts
export const SECTIONS = [
  { id: 'help-getting-started', label: 'Getting Started' },
  { id: 'help-chat', label: 'Chat' },
  { id: 'help-sharing', label: 'Sharing Outside Your Tailnet' },
  { id: 'help-faq', label: 'Frequently Asked Questions' },
] as const
```

---

### Wave 3: `frontend/src/content/help/sharing-guide.md` (new file)

**Analog:** `frontend/src/content/help/chat.md` — established prose style, heading hierarchy, and feature-specific concrete steps

**Content pattern** (from chat.md):
- Opens with a single sentence or short paragraph defining the feature's purpose
- Uses `###` for sub-topics, not `##` (the section `##` heading is provided by HelpContent.tsx)
- Concrete step-by-step instructions, not conceptual/abstract descriptions
- Copy-pasteable code blocks where relevant (e.g., the ACL JSON block)
- External links in standard markdown syntax (`[text](url)`) — `HelpContent.tsx` auto-converts them to `BrowserOpenURL` buttons with `.help-content__external-link` class

**Required article structure** (from CONTEXT.md D-17/D-18 and UI-SPEC Copywriting Contract):
```markdown
## Sharing Outside Your Tailnet

Intro paragraph — the two paths.

### Option 1 — Tailscale Funnel (public internet)

Step-by-step for the Funnel flow from the Share modal.
Note the join code gate and expiry.

### Option 2 — Device Share + ACL (contained)

Steps: share device → tag:agenthub → ACL grant:

​```json
{
  "action": "accept",
  "src": ["autogroup:shared"],
  "dst": ["tag:agenthub:tcp:7443"]
}
​```

Wildcard-default gotcha callout.
External links to Tailscale docs.
```

---

### Wave 0 + all waves: `frontend/src/style.css` (new tokens + component CSS)

**Analog:** existing `--hub-attn-*` token group (at line ~4644) and the `.settings-panel__toggle-row` pattern

**New token insertion point** — adjacent to the `--hub-attn-*` group, in both dark and light `:root` blocks:
```css
/* COLORBLIND-SAFE: internet badge dark hex #43ddb2 — reinforcement only; GlobeAltIcon shape + "INTERNET" text carry state */
--hub-internet-badge-bg: rgba(67, 221, 178, 0.15);
--hub-internet-badge-text: #43ddb2;
```
Light mode (inside `[data-ui-theme="light"]`):
```css
/* COLORBLIND-SAFE: internet badge light hex #0d7a5c — WCAG AA ~5.2:1 on white; icon + text carry state */
--hub-internet-badge-bg: rgba(13, 122, 92, 0.13);
--hub-internet-badge-text: #0d7a5c;
```

**New component classes** — follow BEM `hub-*__*` naming throughout; add at end of file or adjacent to related component blocks:
```css
/* Phase 166 / FUI-03 — internet exposure badge (SessionCard) */
.hub-internet-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;                          /* xs spacing */
  background: var(--hub-internet-badge-bg);
  border-radius: var(--hub-radius-pill);
  padding: 2px 8px;
}
.hub-internet-badge__icon {
  width: 12px; height: 12px;
  color: var(--hub-internet-badge-text);
}
.hub-internet-badge__label {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--hub-internet-badge-text);
}

/* Phase 166 / FUI-03 — tab internet icon (TabBar) */
.tab__internet-icon {
  width: 14px; height: 14px;
  color: var(--hub-internet-badge-text);
  flex-shrink: 0;
}

/* Phase 166 / FUI-01 — inline risk panel (SessionShareModal) */
.hub-funnel-risk-panel {
  background: var(--hub-surface-elevated);
  border: 1px solid var(--hub-border);
  border-radius: var(--hub-radius-sm);   /* 8px */
  padding: 16px;                          /* md */
  overflow: hidden;
  max-height: 0;
}
.hub-funnel-risk-panel--open {
  max-height: 400px;
}
/* Motion guard — mirrors existing project pattern */
@media (prefers-reduced-motion: no-preference) {
  .hub-funnel-risk-panel {
    transition: max-height 200ms ease-out, opacity 150ms ease;
  }
}
@media (prefers-reduced-motion: reduce) {
  .hub-funnel-risk-panel { transition: none; }
}
.hub-funnel-risk-panel__warning {
  border-left: 3px solid var(--hub-warning);
  padding-left: 8px;
  margin-bottom: 12px;
}
.hub-funnel-risk-panel__icon {
  color: var(--hub-warning);
  width: 16px; height: 16px;
  vertical-align: middle;
}
.hub-funnel-risk-panel__text {
  font-size: var(--hub-font-size-base);  /* 14px */
  color: var(--hub-text-secondary);
  line-height: 1.5;
}
.hub-funnel-risk-panel__expiry {
  display: flex;
  align-items: center;
  gap: 8px;                              /* sm */
  margin: 8px 0;
}
.hub-funnel-risk-panel__help-link {
  font-size: var(--hub-font-size-sm);   /* 12.5px */
  color: var(--hub-text-muted);
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}
.hub-funnel-risk-panel__help-link:hover {
  color: var(--hub-accent);
}
.hub-funnel-risk-panel__actions {
  display: flex;
  gap: 8px;                              /* sm */
  margin-top: 12px;
}

/* Phase 166 / FUI-04/05 — internet public section (SessionSharePanel) */
.hub-share-internet-section {
  margin-top: 16px;                      /* md */
  padding-top: 16px;
  border-top: 1px solid var(--hub-border);
}
.hub-share-internet-section__heading {
  font-size: var(--hub-font-size-sm);   /* 12.5px */
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--hub-text-muted);
  margin-bottom: 8px;
}
.hub-share-internet-section__warmup {
  font-size: var(--hub-font-size-sm);
  font-style: italic;
  color: var(--hub-text-muted);
}
.hub-share-internet-section__disable {
  margin-top: 8px;
  /* ghost/outlined style — neutral, not destructive */
  font-size: var(--hub-font-size-base);
  color: var(--hub-text-muted);
  background: none;
  border: 1px solid var(--hub-border);
  border-radius: var(--hub-radius-sm);
  cursor: pointer;
  padding: 4px 12px;
}
.hub-share-internet-section__disable:focus-visible {
  outline: 2px solid var(--hub-accent);
}
.hub-share-internet-section__error {
  font-size: var(--hub-font-size-sm);
  color: var(--hub-destructive);
  margin-top: 6px;
}
```

---

## Shared Patterns

### Wails RPC call + re-issue caps (auth + data flow)
**Source:** `SessionShareModal.tsx` lines 232–256 (`handleBrowseToggle`) and lines 183–199 (seeding effect)
**Apply to:** `handleFunnelEnable` in SessionShareModal, `onDisableFunnel` in SessionSharePanel
```ts
try {
  await SetSessionFunnel(session.id, true, expirySeconds)
  // Then re-issue caps after funnelActive flips (see warm-up polling)
} catch {
  // revert local UI state; show error inline
}
```

### ClipboardSetText copy pattern
**Source:** `SessionSharePanel.tsx` lines 115–124
**Apply to:** Funnel URL "Copy URL" button in hub-share-internet-section
```ts
await ClipboardSetText(url)
setter(true)
setTimeout(() => setter(false), 1500)
```
Never use `navigator.clipboard` — Wails runtime provides `ClipboardSetText`.

### Disabled toggle with opacity + pointer-events
**Source:** `SessionShareModal.tsx` lines 357–380 (browse toggle disabled when sharing OFF)
**Apply to:** Funnel toggle when `webServerMode !== 'tailscale'`
```tsx
<div
  aria-disabled={condition}
  style={condition ? { opacity: 0.6, pointerEvents: 'none' } : undefined}
>
```

### Colorblind-safe badge comment convention
**Source:** `SessionCard.tsx` lines 36–61
**Apply to:** every new color hex constant in Phase 166
```ts
/* COLORBLIND-SAFE: <description> dark hex #XXXXXX — reinforcement only; <shape/text> carries the state */
```

### Cancelled async effect pattern
**Source:** `SessionShareModal.tsx` lines 156–172, 183–199
**Apply to:** warm-up `IssueCapabilities` call after Funnel enable; 30 s timeout `useRef`
```ts
let cancelled = false
void (async () => {
  try { ... } catch { ... }
  if (cancelled) return
  // update state
})()
return () => { cancelled = true }
```

### `?raw` markdown import + SECTION_META entry
**Source:** `HelpTab.tsx` lines 12–14, 61–65; `HelpSectionNav.tsx` lines 10–14
**Apply to:** new `sharing-guide.md`
```ts
import sharingMd from '../content/help/sharing-guide.md?raw'
// in SECTION_META:
{ id: 'help-sharing', label: 'Sharing Outside Your Tailnet', markdown: sharingMd }
// in HelpSectionNav.SECTIONS:
{ id: 'help-sharing', label: 'Sharing Outside Your Tailnet' }
```
Both arrays must be updated in the same commit (Pitfall 5 from RESEARCH.md).

---

## No Analog Found

None. All files have close analogs within the codebase.

---

## Metadata

**Analog search scope:** `frontend/src/components/`, `frontend/src/wailsjs/go/main/`, `frontend/src/content/help/`, `frontend/src/style.css`
**Files read:** 9 (SessionShareModal, SessionSharePanel, SessionCard, TabBar, HelpTab, HelpSectionNav, App.d.ts, App.js, chat.md)
**Pattern extraction date:** 2026-06-30
