/**
 * Phase 168-06 (FIX-02, #117) — "Disconnect all viewers" button in SessionShareModal.
 *
 * Verifies:
 *   - The button renders only when session.viewerCount > 0.
 *   - Clicking it calls the bound DisconnectViewers(session.id) and does NOT call
 *     ToggleWebServing / revoke the share cap (D-06).
 *   - On RPC failure the inline error "Couldn't disconnect viewers — try again."
 *     shows via the existing error-text pattern.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'

// Mock Wails runtime + bindings (must be before component import)
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))

vi.mock('../../wailsjs/go/main/App', () => ({
  GetCapabilityQRCode: vi.fn().mockResolvedValue(''),
  IssueCapabilities: vi.fn().mockResolvedValue({
    readUrl: 'https://example.com/r?cap=READ_TOKEN',
    writeUrl: 'https://example.com/w?cap=WRITE_TOKEN',
    readCode: 'rc',
    writeCode: 'wc',
    homeDir: false,
  }),
  ToggleWebServing: vi.fn().mockResolvedValue(undefined),
  SetSessionBrowse: vi.fn().mockResolvedValue(undefined),
  SetSessionFunnel: vi.fn().mockResolvedValue(undefined),
  GetLocalNetworkPassword: vi.fn().mockResolvedValue('lan-pass-secret'),
  DisconnectViewers: vi.fn().mockResolvedValue(undefined),
}))

import { ToggleWebServing, DisconnectViewers } from '../../wailsjs/go/main/App'
import { SessionShareModal } from '../Hub/SessionShareModal'

const mockedToggleWebServing = ToggleWebServing as ReturnType<typeof vi.fn>
const mockedDisconnectViewers = DisconnectViewers as ReturnType<typeof vi.fn>

interface ModalOpts {
  webEnabled?: boolean
  viewerCount?: number
}

function makeSession(opts: ModalOpts = {}) {
  return {
    id: 'sess-1',
    name: 'Test Session',
    cli: 'claude',
    webEnabled: opts.webEnabled ?? true,
    homeDir: false,
    browseEnabled: false,
    funnelActive: false,
    viewerCount: opts.viewerCount ?? 0,
  }
}

let container: HTMLElement | undefined
let root: Root | undefined

function renderModal(opts: ModalOpts = {}) {
  container = document.createElement('div')
  document.body.appendChild(container)
  root = createRoot(container)
  const session = makeSession(opts)
  flushSync(() => {
    root!.render(
      React.createElement(SessionShareModal, {
        session,
        webServerMode: 'tailscale',
        webServerRunning: true,
        onClose: vi.fn(),
      }),
    )
  })
  return { container: container!, root: root! }
}

function findDisconnectButton(c: HTMLElement): HTMLButtonElement | null {
  return (
    Array.from(c.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Disconnect all viewers',
    ) ?? null
  ) as HTMLButtonElement | null
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

describe('SessionShareModal — "Disconnect all viewers" (Phase 168-06 FIX-02)', () => {
  it('does NOT render the button when viewerCount is 0', () => {
    const { container: c } = renderModal({ viewerCount: 0 })
    expect(findDisconnectButton(c)).toBeNull()
  })

  it('renders the button when viewerCount > 0', () => {
    const { container: c } = renderModal({ viewerCount: 2 })
    expect(findDisconnectButton(c)).not.toBeNull()
  })

  it('clicking the button calls DisconnectViewers(session.id) and NOT ToggleWebServing', async () => {
    const { container: c } = renderModal({ viewerCount: 1 })
    const btn = findDisconnectButton(c)
    expect(btn).not.toBeNull()

    await flushSync(() => { btn!.click() })
    // Allow the async handler to resolve (mirrors SessionShareModal.test.tsx's
    // setTimeout(resolve, 0) pattern for post-click async state updates).
    await new Promise<void>((resolve) => setTimeout(resolve, 0))

    expect(mockedDisconnectViewers).toHaveBeenCalledWith('sess-1')
    expect(mockedToggleWebServing).not.toHaveBeenCalled()
  })

  it('shows the inline error on RPC failure and does not throw', async () => {
    mockedDisconnectViewers.mockRejectedValueOnce(new Error('boom'))
    const { container: c } = renderModal({ viewerCount: 1 })
    const btn = findDisconnectButton(c)
    expect(btn).not.toBeNull()

    await flushSync(() => { btn!.click() })
    await new Promise<void>((resolve) => setTimeout(resolve, 0))

    const text = c.textContent ?? ''
    expect(text).toContain("Couldn't disconnect viewers — try again.")
  })

  it('reusable ghost/outline class is used, not a destructive style', () => {
    const { container: c } = renderModal({ viewerCount: 1 })
    const btn = findDisconnectButton(c)
    expect(btn?.className).toContain('hub-share-internet-section__disable')
    expect(btn?.className).not.toContain('destructive')
  })
})
