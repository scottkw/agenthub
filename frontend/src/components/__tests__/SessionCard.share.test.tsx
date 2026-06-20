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
  onOpenInBrowser?: (url: string) => void
  onBrowseFiles?: (sessionId: string, sessionName: string) => void
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
})

// Phase 138 / CARD-04: Remote-only affordances (Open in browser, Browse files).
// RED against current SessionCard — remote menu items not yet implemented; Plans 02/03 add them.
describe('CARD-04: Remote affordances in overflow menu', () => {
  it('remote card overflow menu contains "Open in browser"', () => {
    const onOpenInBrowser = vi.fn()
    const { container: c } = renderCard({ session: remoteSession, isRemote: true, onOpenInBrowser })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).toContain('Open in browser')
  })
  it('remote card overflow menu contains "Browse files"', () => {
    const onBrowseFiles = vi.fn()
    const { container: c } = renderCard({ session: remoteSession, isRemote: true, onBrowseFiles })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).toContain('Browse files')
  })
  it('local card overflow menu does NOT contain "Open in browser"', () => {
    const { container: c } = renderCard({ session: localSession, isRemote: false })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).not.toContain('Open in browser')
  })
  it('local card overflow menu does NOT contain "Browse files"', () => {
    const { container: c } = renderCard({ session: localSession, isRemote: false })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    expect(c.textContent).not.toContain('Browse files')
  })
})
