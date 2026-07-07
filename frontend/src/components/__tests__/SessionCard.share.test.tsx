/**
 * Phase 137 / D-12/D-13 — SessionCard Share button contract.
 *
 * GREEN tests — Plan 03 has added the Share button + onShare prop to SessionCard.
 *
 * Verifies:
 *   - Share button renders on a local card with accessible label "Share <name>"
 *   - Clicking Share fires onShare and does NOT bubble to the card click handler
 *     (stopPropagation required — Pitfall 6 in RESEARCH.md)
 *   - On a remote card (hostname: "remote.host"), the Share button is disabled,
 *     has aria-label/title "Only the session owner can share", and renders a
 *     lock icon affordance (D-13 colorblind-safe: shape+text, not color alone)
 *
 * Phase 138 / CARD-02/03/04 — Origin indicator, connection indicator, Kill + remote affordances.
 *
 * These tests are RED against the current SessionCard — the new props (isRemote,
 * isConnected, onKill, onOpenInBrowser, onBrowseFiles) will be added in Plans 02/03.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'

// Mock Wails bindings needed transitively by SessionCard imports
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  GetSessionTailLines: vi.fn().mockResolvedValue([]),
}))

// Import the component under test.
import { SessionCard } from '../Hub/SessionCard'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// Minimal SessionInfo-shaped fixture used by the card tests.
const localSession: SessionInfo = {
  id: 'sess-1',
  name: 'My Session',
  cli: 'claude',
  state: 'running',
  status: 'idle',
  createdAt: '2026-01-01T00:00:00Z',
  hostname: '', // local: empty hostname
  webEnabled: false,
  viewerCount: 0,
  homeDir: false,
  browseEnabled: false,
  funnelActive: false,
  funnelWriteActive: false,
  workDir: '/home/user',
}

const remoteSession: SessionInfo = {
  ...localSession,
  id: 'sess-2',
  name: 'Remote Session',
  hostname: 'remote.host', // non-local: D-13 gate
}

// Phase 138 / CARD-03: connected remote fixture (same data, rendered with isRemote + isConnected)
const connectedRemoteSession: SessionInfo = {
  ...remoteSession,
  id: 'sess-3',
  name: 'Connected Remote',
}

// Phase 138 / CARD-02..04: extended RenderOpts with new affordance props.
// These props will be forwarded to SessionCard once Plans 02/03 add them to SessionCardProps.
interface RenderOpts {
  session?: SessionInfo
  onShare?: (session: SessionInfo) => void
  onCardClick?: () => void
  // Phase 138 / CARD-02: explicit provenance flag (replaces hostname-based isLocal heuristic)
  isRemote?: boolean
  // Phase 138 / CARD-03: true when remoteCapsCached.has(session.id)
  isConnected?: boolean
  // Phase 138 / CARD-04: kill and remote affordance handlers
  onKill?: (sessionId: string) => void
  // Phase 146: signature changed from (url: string) to (session) for cap-exchange flow
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  onOpenInBrowser?: (session: any) => void
  onBrowseFiles?: (sessionId: string, sessionName: string) => void
  // Phase 131/134 re-attach affordance — local-only after CARD-04 gating (WR-01)
  onOpenSession?: (id: string, name: string, cli: string) => void
}

let container: HTMLElement | undefined
let root: Root | undefined

function renderCard(opts: RenderOpts = {}) {
  container = document.createElement('div')
  document.body.appendChild(container)
  root = createRoot(container)
  const session = opts.session ?? localSession
  flushSync(() => {
    root!.render(
      React.createElement(SessionCard, {
        session,
        onShare: opts.onShare,
        onCardClick: opts.onCardClick ? (_s: SessionInfo, _rect: DOMRect) => opts.onCardClick!() : undefined,
        // Phase 138 props — forwarded when Plans 02/03 add them to SessionCardProps:
        isRemote: opts.isRemote,
        isConnected: opts.isConnected,
        onKill: opts.onKill,
        onOpenInBrowser: opts.onOpenInBrowser,
        onBrowseFiles: opts.onBrowseFiles,
        onOpenSession: opts.onOpenSession,
      } as Parameters<typeof SessionCard>[0])
    )
  })
  return { container: container!, root: root! }
}

afterEach(() => {
  if (root) {
    flushSync(() => root!.unmount())
    root = undefined
  }
  if (container) {
    container.remove()
    container = undefined
  }
  vi.clearAllMocks()
})

describe('SessionCard — Share button (D-12/D-13)', () => {
  it('renders a Share button on a local card with accessible label "Share <name>"', () => {
    const { container: c } = renderCard()
    const shareBtn = c.querySelector('button[aria-label="Share My Session"], button.hub-card__share') as HTMLButtonElement | null
    expect(shareBtn).not.toBeNull()
    // Must have accessible label containing the session name
    const label = shareBtn!.getAttribute('aria-label') ?? shareBtn!.textContent ?? ''
    expect(label).toContain('My Session')
  })

  it('clicking Share fires onShare and does NOT fire the card click handler (stopPropagation)', () => {
    const onShare = vi.fn()
    const onCardClick = vi.fn()
    const { container: c } = renderCard({ onShare, onCardClick })
    const shareBtn = c.querySelector('button[aria-label="Share My Session"], button.hub-card__share') as HTMLButtonElement | null
    expect(shareBtn).not.toBeNull()
    flushSync(() => {
      shareBtn!.click()
    })
    expect(onShare).toHaveBeenCalledOnce()
    expect(onCardClick).not.toHaveBeenCalled()
  })

  it('on a remote card: Share button is disabled with "Only the session owner can share" label (D-13 colorblind-safe)', () => {
    const { container: c } = renderCard({ session: remoteSession })
    // The Share button should exist but be disabled
    const shareBtn = c.querySelector('button.hub-card__share') as HTMLButtonElement | null
    expect(shareBtn).not.toBeNull()
    expect(shareBtn!.disabled).toBe(true)
    // D-13: non-color signal — text/aria label carries disabled meaning
    const label = shareBtn!.getAttribute('aria-label') ?? ''
    expect(label).toBe('Only the session owner can share')
    const title = shareBtn!.getAttribute('title') ?? ''
    expect(title).toContain('Only the session owner can share')
  })

  it('on a remote card: a lock icon affordance is present (D-13 shape signal)', () => {
    const { container: c } = renderCard({ session: remoteSession })
    // The lock icon carries the disabled state as a shape signal (colorblind-safe)
    const lockIcon = c.querySelector('.hub-card__share-lock')
    expect(lockIcon).not.toBeNull()
  })
})

// Phase 138 / CARD-02: origin indicator (provenance-based, not hostname-based).
// RED against current SessionCard — isRemote prop not yet accepted; Plans 02/03 add it.
describe('CARD-02: Local vs remote origin indicator', () => {
  it('local card renders ComputerDesktopIcon and "Local" text', () => {
    const { container: c } = renderCard({ session: localSession, isRemote: false })
    const origin = c.querySelector('.hub-card__origin')
    expect(origin?.textContent).toContain('Local')
    expect(origin?.querySelector('svg')).not.toBeNull()
  })
  it('remote card renders GlobeAltIcon and hostname text', () => {
    const { container: c } = renderCard({ session: remoteSession, isRemote: true })
    const origin = c.querySelector('.hub-card__origin')
    expect(origin?.textContent).toContain('remote.host')
    expect(origin?.querySelector('svg')).not.toBeNull()
  })
})

// Phase 138 / CARD-03: connection indicator (colorblind-safe — icon + text carry the state).
// RED against current SessionCard — isConnected prop not yet accepted; Plans 02/03 add it.
// COLORBLIND-SAFE: LinkIcon (connected) + GlobeAltIcon (available); color is reinforcement only.
describe('CARD-03: Connection indicator (colorblind-safe)', () => {
  it('connected remote card renders .hub-card__conn--connected and "Connected" text', () => {
    const { container: c } = renderCard({ session: connectedRemoteSession, isRemote: true, isConnected: true })
    const chip = c.querySelector('.hub-card__conn--connected')
    expect(chip).not.toBeNull()
    expect(chip?.textContent).toContain('Connected')
    expect(chip?.querySelector('svg')).not.toBeNull() // LinkIcon shape signal
  })
  it('available remote card renders .hub-card__conn without --connected and "Available" text', () => {
    const { container: c } = renderCard({ session: remoteSession, isRemote: true, isConnected: false })
    const chip = c.querySelector('.hub-card__conn')
    expect(chip).not.toBeNull()
    expect(chip?.textContent).toContain('Available')
    expect(chip?.classList.contains('hub-card__conn--connected')).toBe(false)
  })
  it('connection chip is absent on local cards', () => {
    const { container: c } = renderCard({ session: localSession, isRemote: false })
    expect(c.querySelector('.hub-card__conn')).toBeNull()
  })
  it('connected card aria-label includes ", connected"', () => {
    const { container: c } = renderCard({ session: connectedRemoteSession, isRemote: true, isConnected: true })
    const article = c.querySelector('article')
    expect(article?.getAttribute('aria-label')).toContain(', connected')
  })
})

// Phase 138 / CARD-04: Kill + remote affordances in the overflow menu.
// RED against current SessionCard — onKill/onOpenInBrowser/onBrowseFiles not yet accepted; Plans 02/03 add them.
// Kill uses a two-step inline confirm (UI-SPEC Claude's Discretion choice — no modal).
describe('CARD-04: Kill menu item guard (stopPropagation)', () => {
  it('Kill option appears in overflow menu for live local sessions', () => {
    const onKill = vi.fn()
    const { container: c } = renderCard({ session: localSession, onKill })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).toContain('Kill session')
  })
  it('Kill confirm does not trigger card-click modal (stopPropagation guard)', () => {
    const onCardClick = vi.fn()
    const onKill = vi.fn()
    const { container: c } = renderCard({ session: localSession, onKill, onCardClick })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    const killBtn = Array.from(c.querySelectorAll('.hub-card__menu-item')).find(
      (el) => el.textContent?.includes('Kill session')
    ) as HTMLButtonElement
    flushSync(() => { killBtn?.click() })
    expect(onCardClick).not.toHaveBeenCalled()
  })
  // CR-02: Kill kills only locally-owned daemon tabs; on a remote card it is a silent
  // no-op (the old Remote page offered no Kill). Hide it on remote rather than show a
  // dead two-step destructive action.
  it('Kill option is HIDDEN on remote cards (CR-02 — no dead destructive affordance)', () => {
    const onKill = vi.fn()
    const { container: c } = renderCard({ session: remoteSession, isRemote: true, onKill })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).not.toContain('Kill session')
  })
})

// Phase 138 / CARD-04: Remote-only affordances (Open in browser, Browse files).
// RED against current SessionCard — remote menu items not yet implemented; Plans 02/03 add them.
describe('CARD-04: Remote affordances in overflow menu', () => {
  // FIX-03 RC-C (plan 09): D-17 opens an in-app tab, so the label reads "Open in
  // tab" (in-app glyph) — the stale "Open in browser" external-link wording is gone.
  it('remote card overflow menu contains "Open in tab" and NOT the stale "Open in browser"', () => {
    const onOpenInBrowser = vi.fn()
    const { container: c } = renderCard({ session: remoteSession, isRemote: true, onOpenInBrowser })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).toContain('Open in tab')
    expect(c.textContent).not.toContain('Open in browser')
  })
  it('remote card overflow menu contains "Browse files"', () => {
    const onBrowseFiles = vi.fn()
    const { container: c } = renderCard({ session: remoteSession, isRemote: true, onBrowseFiles })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).toContain('Browse files')
  })
  it('local card overflow menu does NOT contain "Open in tab"', () => {
    const { container: c } = renderCard({ session: localSession, isRemote: false })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).not.toContain('Open in tab')
  })
  it('local card overflow menu does NOT contain "Browse files"', () => {
    const { container: c } = renderCard({ session: localSession, isRemote: false })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).not.toContain('Browse files')
  })
  // CR-01 / Phase 146 FIX-03 (out-of-band): "Open in tab" is NOT gated on roJoinCode.
  // The button opens for any remote session — the modal guides the viewer to obtain a code.
  it('"Open in tab" is enabled without roJoinCode (D-03: modal replaces dead-end)', () => {
    const onOpenInBrowser = vi.fn()
    // No roJoinCode — out-of-band design: modal guides the viewer
    const remoteWithUrl = { ...remoteSession, url: 'https://remote.host/session/sess-2' } as SessionInfo
    const { container: c } = renderCard({ session: remoteWithUrl, isRemote: true, onOpenInBrowser })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    const openBtn = Array.from(c.querySelectorAll('.hub-card__menu-item')).find(
      (el) => el.textContent?.includes('Open in tab')
    ) as HTMLButtonElement
    // D-03: button must NOT be disabled — modal provides the path to obtain a code
    expect(openBtn?.disabled, '"Open in tab" must not be disabled (D-03 out-of-band)').toBe(false)
    flushSync(() => { openBtn?.click() })
    // Called with the session object so the handler can route to the modal
    expect(onOpenInBrowser).toHaveBeenCalledWith(expect.objectContaining({ id: 'sess-2' }))
  })
})

// Phase 138 / WR-01: the row-5 re-attach "Open" button attaches a LOCAL PTY; it must not
// render on remote cards (the old Remote page offered no local re-attach). Preserve it on
// local cards (Phase 131/134 re-attach affordance).
describe('CARD-04: re-attach Open button is local-only', () => {
  it('does NOT render the re-attach Open button on remote cards (WR-01)', () => {
    const onOpenSession = vi.fn()
    const { container: c } = renderCard({ session: remoteSession, isRemote: true, onOpenSession })
    expect(c.querySelector('button[aria-label="Open Remote Session"]')).toBeNull()
  })
  it('still renders the re-attach Open button on live local cards (regression guard)', () => {
    const onOpenSession = vi.fn()
    const { container: c } = renderCard({ session: localSession, isRemote: false, onOpenSession })
    expect(c.querySelector('button[aria-label="Open My Session"]')).not.toBeNull()
  })
})

// ---------------------------------------------------------------------------
// Phase 171 / FNL-09 — FULL ACCESS badge (R7 distinct indicator + coexistence)
// ---------------------------------------------------------------------------
describe('FNL-09: FULL ACCESS badge — distinct indicator + coexistence with INTERNET', () => {
  it('R7: renders .hub-fullaccess-badge with LockOpenIcon and literal label "FULL ACCESS" when funnelWriteActive', () => {
    const session: SessionInfo = { ...localSession, funnelWriteActive: true }
    const { container: c } = renderCard({ session })
    const badge = c.querySelector('.hub-fullaccess-badge')
    expect(badge).not.toBeNull()
    expect(badge!.querySelector('svg')).not.toBeNull()
    expect(badge!.querySelector('.hub-fullaccess-badge__label')?.textContent).toBe('FULL ACCESS')
    // Source-level distinctness from the read badge — different class, icon import name,
    // and label string (never color alone, per the colorblind-safe standing rule).
    expect(badge!.className).not.toBe('hub-internet-badge')
  })

  it('does not render .hub-fullaccess-badge when funnelWriteActive is false', () => {
    const session: SessionInfo = { ...localSession, funnelWriteActive: false }
    const { container: c } = renderCard({ session })
    expect(c.querySelector('.hub-fullaccess-badge')).toBeNull()
  })

  it('coexistence: both badges render (read INTERNET first, FULL ACCESS second) when both are active', () => {
    const session: SessionInfo = { ...localSession, funnelActive: true, funnelWriteActive: true }
    const { container: c } = renderCard({ session })
    const internetBadge = c.querySelector('.hub-internet-badge')
    const fullAccessBadge = c.querySelector('.hub-fullaccess-badge')
    expect(internetBadge).not.toBeNull()
    expect(fullAccessBadge).not.toBeNull()
    // DOM order: INTERNET badge precedes FULL ACCESS badge (read-then-write order, D-10).
    const badges = Array.from(c.querySelectorAll('.hub-internet-badge, .hub-fullaccess-badge'))
    expect(badges[0]).toBe(internetBadge)
    expect(badges[1]).toBe(fullAccessBadge)
  })

  it('RW teardown keeps the read badge: funnelWriteActive=false with funnelActive=true renders only the read badge', () => {
    const session: SessionInfo = { ...localSession, funnelActive: true, funnelWriteActive: false }
    const { container: c } = renderCard({ session })
    expect(c.querySelector('.hub-internet-badge')).not.toBeNull()
    expect(c.querySelector('.hub-fullaccess-badge')).toBeNull()
  })
})
