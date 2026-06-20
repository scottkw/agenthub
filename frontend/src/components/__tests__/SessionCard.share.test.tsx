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

interface RenderOpts {
  session?: SessionInfo
  onShare?: (session: SessionInfo) => void
  onCardClick?: () => void
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
        onCardClick: opts.onCardClick ? (s: SessionInfo, _rect: DOMRect) => opts.onCardClick!() : undefined,
      })
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
