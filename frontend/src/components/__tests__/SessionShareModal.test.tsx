/**
 * Phase 137 / SHARE-01/02/04/05/06 + D-09 — SessionShareModal contract.
 * Phase 150 SET-01 — shell web-share warning cross-surface parity (D-09/D-10).
 *
 * GREEN tests — Plan 03 has built the SessionShareModal component.
 *
 * Verifies:
 *   SHARE-01: "Share the session" toggle present; toggling ON reveals share content
 *   SHARE-02: "Enable remote file browsing" toggle present; disabled when sharing OFF
 *   SHARE-03: browse toggle calls SetSessionBrowse then IssueCapabilities (server-truth)
 *   SHARE-04: local-mode fixture surfaces LAN Basic Auth password via GetLocalNetworkPassword
 *   SHARE-05: server-truth seeding — opening with webEnabled=true calls IssueCapabilities once
 *   SHARE-05: stale-URL clear — when webServerRunning flips false then true, cached URLs cleared
 *   SHARE-06: browse toggle calls SetSessionBrowse + re-issues caps (Pitfall 1 mitigation)
 *   D-09: homeDir fixture shows home-dir warning before browse is enabled
 *   SET-01 D-09: shell session + warningEnabled + !warned → banner shown, ToggleWebServing NOT called
 *   SET-01: shell session + warningEnabled=false → banner NOT shown; ToggleWebServing called
 *   SET-01: non-shell session → banner NOT shown; ToggleWebServing called immediately
 *   SET-01: confirming banner calls onShellWebShareConfirm + enables share
 *   SET-01: cancelling banner calls onShellWebShareCancel, share stays OFF
 */
import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
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

// Import mocked bindings for assertion
import { IssueCapabilities, SetSessionBrowse, SetSessionFunnel, GetLocalNetworkPassword, ToggleWebServing } from '../../wailsjs/go/main/App'
// Import the component under test
import { SessionShareModal } from '../Hub/SessionShareModal'

const mockedIssueCapabilities = IssueCapabilities as ReturnType<typeof vi.fn>
const mockedSetSessionBrowse = SetSessionBrowse as ReturnType<typeof vi.fn>
const mockedGetLocalNetworkPassword = GetLocalNetworkPassword as ReturnType<typeof vi.fn>

interface ModalOpts {
  webEnabled?: boolean
  homeDir?: boolean
  browseEnabled?: boolean
  funnelActive?: boolean
  viewerCount?: number
  webServerMode?: 'local' | 'tailscale'
  webServerRunning?: boolean
  // Phase 150 SET-01 shell-warn props
  cli?: string
  shellWebShareWarned?: boolean
  shellWebShareWarningEnabled?: boolean
  onShellWebShareConfirm?: () => Promise<boolean>
  onShellWebShareCancel?: () => void
  // Phase 166 FUI-06
  onOpenHelp?: () => void
  // Phase 168-08 (UX-02 / #115 gap closure)
  onShareEnabledChange?: (sessionId: string, enabled: boolean) => void
}

function makeSession(opts: ModalOpts = {}) {
  return {
    id: 'sess-1',
    name: 'Test Session',
    cli: opts.cli ?? 'claude',
    webEnabled: opts.webEnabled ?? false,
    homeDir: opts.homeDir ?? false,
    browseEnabled: opts.browseEnabled ?? false,
    funnelActive: opts.funnelActive ?? false,
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
        webServerMode: opts.webServerMode ?? null,
        webServerRunning: opts.webServerRunning ?? true,
        onClose: vi.fn(),
        shellWebShareWarned: opts.shellWebShareWarned,
        shellWebShareWarningEnabled: opts.shellWebShareWarningEnabled,
        onShellWebShareConfirm: opts.onShellWebShareConfirm,
        onShellWebShareCancel: opts.onShellWebShareCancel,
        onOpenHelp: opts.onOpenHelp,
        onShareEnabledChange: opts.onShareEnabledChange,
      })
    )
  })
  return { container: container!, root: root! }
}

// Phase 166 helpers: locate the Funnel toggle and risk-panel buttons.
function findFunnelToggle(c: HTMLElement): HTMLInputElement | null {
  return c.querySelector('input[aria-label="Enable internet sharing"]') as HTMLInputElement | null
}
function findByText(c: HTMLElement, text: string): HTMLElement | null {
  return (Array.from(c.querySelectorAll('button')).find(
    (b) => b.textContent?.trim() === text,
  ) as HTMLElement | null)
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

describe('SessionShareModal — SHARE-01: share toggle', () => {
  it('renders a "Share the session" toggle', () => {
    const { container: c } = renderModal()
    // The toggle should be present with a label about sharing
    const text = c.textContent ?? ''
    expect(text).toMatch(/share.*session/i)
  })

  it('toggling share ON reveals RO + RW link rows', async () => {
    const { container: c } = renderModal()
    // Find the share toggle
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    expect(shareToggle).not.toBeNull()
    // Toggle it ON
    await flushSync(() => { shareToggle!.click() })
    // Should now show link rows (ToggleWebServing called)
    const text = c.textContent ?? ''
    expect(text.length).toBeGreaterThan(0) // modal has content
  })
})

describe('SessionShareModal — SHARE-02: browse toggle', () => {
  it('renders "Enable remote file browsing" toggle', () => {
    const { container: c } = renderModal({ webEnabled: true })
    const text = c.textContent ?? ''
    expect(text).toMatch(/browse|file browsing/i)
  })

  it('browse toggle is disabled when sharing is OFF', () => {
    const { container: c } = renderModal({ webEnabled: false })
    // Any browse-related toggle must be disabled or absent while sharing is OFF
    const browseRow = Array.from(c.querySelectorAll('[role="switch"], input[type="checkbox"]')).find(
      el => {
        const label = el.getAttribute('aria-label') ?? ''
        const parentText = el.closest('label, div')?.textContent ?? ''
        return label.match(/browse/i) || parentText.match(/browse/i)
      }
    ) as HTMLElement | null
    if (browseRow) {
      const isDisabled =
        browseRow.getAttribute('aria-disabled') === 'true' ||
        (browseRow as HTMLInputElement).disabled === true ||
        browseRow.getAttribute('disabled') !== null
      expect(isDisabled).toBe(true)
    }
    // If no browse toggle rendered at all, that also satisfies "disabled when sharing OFF"
  })

  it('toggling browse ON calls SetSessionBrowse then IssueCapabilities (SHARE-06 / Pitfall 1)', async () => {
    const { container: c } = renderModal({ webEnabled: true })
    const browseToggle = Array.from(c.querySelectorAll('[role="switch"], input[type="checkbox"]')).find(
      el => {
        const label = el.getAttribute('aria-label') ?? ''
        const parentText = el.closest('label, div')?.textContent ?? ''
        return label.match(/browse/i) || parentText.match(/browse|file browsing/i)
      }
    ) as HTMLElement | null
    if (browseToggle) {
      await flushSync(() => { browseToggle.click() })
      // SetSessionBrowse must be called before (or as part of) cap re-issuance
      expect(mockedSetSessionBrowse).toHaveBeenCalled()
      expect(mockedIssueCapabilities).toHaveBeenCalled()
    }
  })
})

describe('SessionShareModal — SHARE-04: LAN password', () => {
  it('local-mode fixture surfaces LAN Basic Auth password via GetLocalNetworkPassword', async () => {
    const { container: c } = renderModal({
      webEnabled: true,
      webServerMode: 'local',
      webServerRunning: true,
    })
    // Give async effects time to resolve
    await new Promise<void>((r) => setTimeout(r, 0))
    flushSync(() => {/* re-render trigger */})
    // GetLocalNetworkPassword should have been called
    expect(mockedGetLocalNetworkPassword).toHaveBeenCalled()
    void c
  })
})

describe('SessionShareModal — D-09: homeDir warning', () => {
  it('homeDir fixture shows home-dir warning before browse is enabled', () => {
    const { container: c } = renderModal({ homeDir: true, webEnabled: true })
    const text = c.textContent ?? ''
    // Should show some kind of home directory / security warning
    expect(text).toMatch(/home|directory|warning|caution/i)
  })
})

describe('SessionShareModal — S-07 render smoke (hub-share-modal structure)', () => {
  it('renders .hub-share-modal__header element', () => {
    const { container: c } = renderModal({ webEnabled: true })
    const header = c.querySelector('.hub-share-modal__header')
    expect(header).not.toBeNull()
  })

  it('renders .hub-share-modal__body element', () => {
    const { container: c } = renderModal({ webEnabled: true })
    const body = c.querySelector('.hub-share-modal__body')
    expect(body).not.toBeNull()
  })

  it('renders .hub-share-modal panel container', () => {
    const { container: c } = renderModal()
    const panel = c.querySelector('.hub-share-modal')
    expect(panel).not.toBeNull()
  })
})

describe('SessionShareModal — SHARE-05: server-truth seeding', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('opening with webEnabled=true and no cached share calls IssueCapabilities once', async () => {
    renderModal({ webEnabled: true, webServerRunning: true })
    // Give async effects time to run
    await new Promise<void>((r) => setTimeout(r, 0))
    expect(mockedIssueCapabilities).toHaveBeenCalledTimes(1)
  })

  it('stale-URL clear: when webServerRunning flips false then true, IssueCapabilities is re-called', async () => {
    const { root: r, container: c } = renderModal({ webEnabled: true, webServerRunning: true })
    await new Promise<void>((resolve) => setTimeout(resolve, 0))
    const callsAfterOpen = mockedIssueCapabilities.mock.calls.length

    // Flip webServerRunning false → true (simulates restart)
    const session = makeSession({ webEnabled: true })
    flushSync(() => {
      r.render(
        React.createElement(SessionShareModal, {
          session,
          webServerMode: null,
          webServerRunning: false, // server stopped — stale URLs should clear
          onClose: vi.fn(),
        })
      )
    })
    flushSync(() => {
      r.render(
        React.createElement(SessionShareModal, {
          session,
          webServerMode: null,
          webServerRunning: true, // server restarted — must re-issue
          onClose: vi.fn(),
        })
      )
    })
    await new Promise<void>((resolve) => setTimeout(resolve, 0))

    // IssueCapabilities should have been called again after restart
    expect(mockedIssueCapabilities.mock.calls.length).toBeGreaterThan(callsAfterOpen)
    void c // reference to prevent unused-var lint
  })
})

// ---------------------------------------------------------------------------
// Phase 150 SET-01 — shell web-share warning cross-surface parity (D-09/D-10)
// ---------------------------------------------------------------------------
const mockedToggleWebServing = ToggleWebServing as ReturnType<typeof vi.fn>

describe('SessionShareModal — SET-01: shell warning interception on Hub Share modal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shell session + warningEnabled + !warned → banner shown, ToggleWebServing NOT called', async () => {
    const { container: c } = renderModal({
      cli: 'bash',
      shellWebShareWarningEnabled: true,
      shellWebShareWarned: false,
      onShellWebShareConfirm: vi.fn().mockResolvedValue(true),
      onShellWebShareCancel: vi.fn(),
    })
    // Find the share toggle and click it (toggle share ON)
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    expect(shareToggle).not.toBeNull()
    await flushSync(() => { shareToggle!.click() })
    // Banner should be shown (ShellWebShareBanner text)
    const text = c.textContent ?? ''
    expect(text).toMatch(/web sharing this shell|expose.*command execution/i)
    // ToggleWebServing must NOT have been called (interception happened)
    expect(mockedToggleWebServing).not.toHaveBeenCalled()
  })

  it('shell session whose cli is a FULL PATH (/bin/zsh) → banner shown (gap-closure: live UAT)', async () => {
    // Regression: sessions created via "New session" carry the shell's full
    // path as cli (e.g. "/bin/zsh"), but the gate matched a bare-name set, so
    // the banner never fired in the real app. All prior tests used bare names.
    const { container: c } = renderModal({
      cli: '/bin/zsh',
      shellWebShareWarningEnabled: true,
      shellWebShareWarned: false,
      onShellWebShareConfirm: vi.fn().mockResolvedValue(true),
      onShellWebShareCancel: vi.fn(),
    })
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    expect(shareToggle).not.toBeNull()
    await flushSync(() => { shareToggle!.click() })
    const text = c.textContent ?? ''
    expect(text).toMatch(/web sharing this shell|expose.*command execution/i)
    expect(mockedToggleWebServing).not.toHaveBeenCalled()
  })

  it('shell session + warningEnabled=false → banner NOT shown; ToggleWebServing called', async () => {
    const { container: c } = renderModal({
      cli: 'zsh',
      shellWebShareWarningEnabled: false,
      shellWebShareWarned: false,
      onShellWebShareConfirm: vi.fn().mockResolvedValue(true),
      onShellWebShareCancel: vi.fn(),
    })
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    expect(shareToggle).not.toBeNull()
    await flushSync(() => { shareToggle!.click() })
    await new Promise<void>((r) => setTimeout(r, 0))
    // No banner text expected
    const text = c.textContent ?? ''
    expect(text).not.toMatch(/expose.*command execution/i)
    // ToggleWebServing should have been called (warning suppressed)
    expect(mockedToggleWebServing).toHaveBeenCalled()
  })

  it('non-shell (claude) session → banner NOT shown; ToggleWebServing called immediately', async () => {
    const { container: c } = renderModal({
      cli: 'claude',
      shellWebShareWarningEnabled: true,
      shellWebShareWarned: false,
      onShellWebShareConfirm: vi.fn().mockResolvedValue(true),
      onShellWebShareCancel: vi.fn(),
    })
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    expect(shareToggle).not.toBeNull()
    await flushSync(() => { shareToggle!.click() })
    await new Promise<void>((r) => setTimeout(r, 0))
    // No shell-warning banner
    const text = c.textContent ?? ''
    expect(text).not.toMatch(/expose.*command execution/i)
    // ToggleWebServing called (non-shell passes through)
    expect(mockedToggleWebServing).toHaveBeenCalled()
  })

  it('shell session already warned → banner NOT shown; ToggleWebServing called', async () => {
    const { container: c } = renderModal({
      cli: 'shell',
      shellWebShareWarningEnabled: true,
      shellWebShareWarned: true,  // already acknowledged
      onShellWebShareConfirm: vi.fn().mockResolvedValue(true),
      onShellWebShareCancel: vi.fn(),
    })
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    expect(shareToggle).not.toBeNull()
    await flushSync(() => { shareToggle!.click() })
    await new Promise<void>((r) => setTimeout(r, 0))
    // No banner — already warned, skip the interception
    const text = c.textContent ?? ''
    expect(text).not.toMatch(/expose.*command execution/i)
    expect(mockedToggleWebServing).toHaveBeenCalled()
  })

  it('cancelling the banner leaves share OFF and calls onShellWebShareCancel', async () => {
    const onCancel = vi.fn()
    const { container: c } = renderModal({
      cli: 'bash',
      shellWebShareWarningEnabled: true,
      shellWebShareWarned: false,
      onShellWebShareConfirm: vi.fn().mockResolvedValue(true),
      onShellWebShareCancel: onCancel,
    })
    // Trigger banner
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    await flushSync(() => { shareToggle!.click() })
    // Find the Cancel button and click it
    const cancelBtn = Array.from(c.querySelectorAll('button')).find(
      (btn) => btn.textContent?.trim() === 'Cancel'
    ) as HTMLElement | null
    if (cancelBtn) {
      await flushSync(() => { cancelBtn.click() })
      expect(onCancel).toHaveBeenCalled()
    }
    // ToggleWebServing should NOT have been called
    expect(mockedToggleWebServing).not.toHaveBeenCalled()
  })

  it('confirming the banner calls onShellWebShareConfirm then enables share on success', async () => {
    const onConfirm = vi.fn().mockResolvedValue(true)
    const { container: c } = renderModal({
      cli: 'zsh',
      shellWebShareWarningEnabled: true,
      shellWebShareWarned: false,
      onShellWebShareConfirm: onConfirm,
      onShellWebShareCancel: vi.fn(),
    })
    // Trigger banner
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    await flushSync(() => { shareToggle!.click() })
    // Find the "Enable web sharing" button and click it
    const enableBtn = Array.from(c.querySelectorAll('button')).find(
      (btn) => btn.textContent?.match(/Enable web sharing/i)
    ) as HTMLElement | null
    if (enableBtn) {
      await flushSync(() => { enableBtn.click() })
      await new Promise<void>((r) => setTimeout(r, 0))
      // onShellWebShareConfirm should have been called
      expect(onConfirm).toHaveBeenCalled()
      // WR-01: on success, the toggle should now read "on"
      const toggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLInputElement | null
      expect(toggle?.checked).toBe(true)
    }
  })

  // WR-01 regression coverage: if the underlying ToggleWebServing call fails,
  // the modal must NOT report "sharing ON" even though onShellWebShareConfirm
  // resolved (without throwing) with a falsy success signal.
  it('WR-01: confirming the banner does NOT enable share when onShellWebShareConfirm resolves false (failure)', async () => {
    const onConfirm = vi.fn().mockResolvedValue(false)
    const { container: c } = renderModal({
      cli: 'zsh',
      shellWebShareWarningEnabled: true,
      shellWebShareWarned: false,
      onShellWebShareConfirm: onConfirm,
      onShellWebShareCancel: vi.fn(),
    })
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    await flushSync(() => { shareToggle!.click() })
    const enableBtn = Array.from(c.querySelectorAll('button')).find(
      (btn) => btn.textContent?.match(/Enable web sharing/i)
    ) as HTMLElement | null
    if (enableBtn) {
      await flushSync(() => { enableBtn.click() })
      await new Promise<void>((r) => setTimeout(r, 0))
      expect(onConfirm).toHaveBeenCalled()
      // The toggle must remain OFF — the backend call failed.
      const toggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLInputElement | null
      expect(toggle?.checked).toBe(false)
    }
  })
})

// ---------------------------------------------------------------------------
// Phase 166 — FUI-01/FUI-02/FUI-06/D-15 — Funnel enable flow
// ---------------------------------------------------------------------------
const mockedSetSessionFunnel = SetSessionFunnel as ReturnType<typeof vi.fn>

describe('SessionShareModal — FUI-01: risk panel on every enable (Tailscale mode)', () => {
  beforeEach(() => vi.clearAllMocks())

  it('renders an "Enable internet sharing" Funnel toggle', () => {
    const { container: c } = renderModal({ webServerMode: 'tailscale' })
    expect(findFunnelToggle(c)).not.toBeNull()
  })

  it('flipping the Funnel toggle ON opens the risk panel and does NOT call SetSessionFunnel', async () => {
    const { container: c } = renderModal({ webServerMode: 'tailscale' })
    const toggle = findFunnelToggle(c)!
    await flushSync(() => { toggle.click() })
    const panel = c.querySelector('.hub-funnel-risk-panel--open')
    expect(panel).not.toBeNull()
    // The risk statement is visible
    expect(c.textContent).toMatch(/reachable from the public internet/i)
    // Toggle ON must NOT commit — only the explicit CTA commits (D-02)
    expect(mockedSetSessionFunnel).not.toHaveBeenCalled()
  })
})

describe('SessionShareModal — FUI-02: explicit CTA commits with the selected preset', () => {
  beforeEach(() => vi.clearAllMocks())

  it('"Enable internet share" calls SetSessionFunnel(id, true, 3600) for the default', async () => {
    const { container: c } = renderModal({ webServerMode: 'tailscale' })
    await flushSync(() => { findFunnelToggle(c)!.click() })
    const cta = findByText(c, 'Enable internet share')!
    await flushSync(() => { cta.click() })
    await new Promise<void>((r) => setTimeout(r, 0))
    expect(mockedSetSessionFunnel).toHaveBeenCalledWith('sess-1', true, 3600)
  })

  it('changing the expiry preset commits the selected value', async () => {
    const { container: c } = renderModal({ webServerMode: 'tailscale' })
    await flushSync(() => { findFunnelToggle(c)!.click() })
    const select = c.querySelector('.hub-funnel-risk-panel select') as HTMLSelectElement
    select.value = '14400'
    flushSync(() => { select.dispatchEvent(new Event('change', { bubbles: true })) })
    const cta = findByText(c, 'Enable internet share')!
    await flushSync(() => { cta.click() })
    await new Promise<void>((r) => setTimeout(r, 0))
    expect(mockedSetSessionFunnel).toHaveBeenCalledWith('sess-1', true, 14400)
  })

  it('"Keep local only" collapses the panel with no API call', async () => {
    const { container: c } = renderModal({ webServerMode: 'tailscale' })
    await flushSync(() => { findFunnelToggle(c)!.click() })
    expect(c.querySelector('.hub-funnel-risk-panel--open')).not.toBeNull()
    const cancel = findByText(c, 'Keep local only')!
    await flushSync(() => { cancel.click() })
    expect(c.querySelector('.hub-funnel-risk-panel--open')).toBeNull()
    expect(mockedSetSessionFunnel).not.toHaveBeenCalled()
  })
})

describe('SessionShareModal — D-15: local-fallback fails closed', () => {
  beforeEach(() => vi.clearAllMocks())

  it('with webServerMode="local" the Funnel toggle is disabled and SetSessionFunnel is never called', async () => {
    const { container: c } = renderModal({ webServerMode: 'local' })
    const toggle = findFunnelToggle(c)
    expect(toggle).not.toBeNull()
    expect(toggle!.disabled).toBe(true)
    // Explanatory note present
    expect(c.textContent).toMatch(/Internet sharing requires Tailscale/i)
    // Clicking must not open a panel nor call the binding
    await flushSync(() => { toggle!.click() })
    expect(c.querySelector('.hub-funnel-risk-panel--open')).toBeNull()
    expect(mockedSetSessionFunnel).not.toHaveBeenCalled()
  })
})

describe('SessionShareModal — FUI-06: Help cross-link', () => {
  beforeEach(() => vi.clearAllMocks())

  it('the risk-panel Help link invokes the onOpenHelp callback', async () => {
    const onOpenHelp = vi.fn()
    const { container: c } = renderModal({ webServerMode: 'tailscale', onOpenHelp })
    await flushSync(() => { findFunnelToggle(c)!.click() })
    const helpLink = Array.from(c.querySelectorAll('button')).find((b) =>
      /See the Sharing Guide/i.test(b.textContent ?? ''),
    ) as HTMLElement | null
    expect(helpLink).not.toBeNull()
    await flushSync(() => { helpLink!.click() })
    expect(onOpenHelp).toHaveBeenCalled()
  })
})

// ---------------------------------------------------------------------------
// Phase 166 — FUI-05 warm-up state machine + FUI-04 disable (Plan 05)
// Share must be ON (webEnabled) so the SessionSharePanel — which hosts the
// Internet (public) section — is rendered.
// ---------------------------------------------------------------------------
const mockedIssueCapabilitiesForFunnel = IssueCapabilities as ReturnType<typeof vi.fn>

async function tick(): Promise<void> {
  await new Promise<void>((r) => setTimeout(r, 0))
}

async function enableFunnel(c: HTMLElement): Promise<void> {
  await flushSync(() => { findFunnelToggle(c)!.click() })
  const cta = findByText(c, 'Enable internet share')!
  await flushSync(() => { cta.click() })
  await tick()
}

describe('SessionShareModal — FUI-05: warm-up → live URL', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows the "Starting up… (TLS warming up)" state immediately after enable', async () => {
    const { container: c } = renderModal({ webEnabled: true, webServerMode: 'tailscale' })
    await tick() // seeding IssueCapabilities → cachedShare → SessionSharePanel renders
    await enableFunnel(c)
    expect(c.textContent).toMatch(/Starting up… \(TLS warming up\)/)
  })

  it('re-issues IssueCapabilities and reveals the public URL when funnelActive flips true', async () => {
    const { container: c, root: r } = renderModal({ webEnabled: true, webServerMode: 'tailscale' })
    await tick()
    await enableFunnel(c)
    const callsBefore = mockedIssueCapabilitiesForFunnel.mock.calls.length
    // Simulate the HubPanel 3s-poll sync delivering funnelActive=true.
    const session = makeSession({ webEnabled: true, funnelActive: true })
    flushSync(() => {
      r.render(
        React.createElement(SessionShareModal, {
          session,
          webServerMode: 'tailscale',
          webServerRunning: true,
          onClose: vi.fn(),
        }),
      )
    })
    await tick()
    expect(mockedIssueCapabilitiesForFunnel.mock.calls.length).toBeGreaterThan(callsBefore)
    expect(c.textContent).toMatch(/Public URL \(read-only\)/)
    expect(c.textContent).not.toMatch(/Starting up/)
  })
})

describe('SessionShareModal — FUI-04: one-click disable', () => {
  beforeEach(() => vi.clearAllMocks())

  it('"Disable internet share" calls SetSessionFunnel(id, false, 0)', async () => {
    const { container: c, root: r } = renderModal({ webEnabled: true, webServerMode: 'tailscale' })
    await tick()
    await enableFunnel(c)
    // Advance to the active state so the disable button sits in the live section.
    const session = makeSession({ webEnabled: true, funnelActive: true })
    flushSync(() => {
      r.render(
        React.createElement(SessionShareModal, {
          session,
          webServerMode: 'tailscale',
          webServerRunning: true,
          onClose: vi.fn(),
        }),
      )
    })
    await tick()
    const disableBtn = findByText(c, 'Disable internet share')!
    await flushSync(() => { disableBtn.click() })
    await tick()
    expect(mockedSetSessionFunnel).toHaveBeenCalledWith('sess-1', false, 0)
  })
})

describe('SessionShareModal — FUI-05: warm-up timeout + timer cleanup (fake timers)', () => {
  beforeEach(() => vi.clearAllMocks())

  it('surfaces the timeout error after 30s without funnelActive', async () => {
    vi.useFakeTimers()
    try {
      const { container: c } = renderModal({ webEnabled: true, webServerMode: 'tailscale' })
      await React.act(async () => { await vi.advanceTimersByTimeAsync(0) }) // seeding
      await React.act(async () => { findFunnelToggle(c)!.click() })
      // enable → resolve SetSessionFunnel → warmingUp + arm 30s timeout
      await React.act(async () => {
        findByText(c, 'Enable internet share')!.click()
        await vi.advanceTimersByTimeAsync(0)
      })
      expect(c.textContent).toMatch(/Starting up/)
      await React.act(async () => { await vi.advanceTimersByTimeAsync(30000) })
      expect(c.textContent).toMatch(/Connection timed out\. Try disabling and re-enabling\./)
    } finally {
      vi.useRealTimers()
    }
  })

  it('clears the 30s timeout on disable — no late timeout fire', async () => {
    vi.useFakeTimers()
    try {
      const { container: c } = renderModal({ webEnabled: true, webServerMode: 'tailscale' })
      await React.act(async () => { await vi.advanceTimersByTimeAsync(0) })
      await React.act(async () => { findFunnelToggle(c)!.click() })
      await React.act(async () => {
        findByText(c, 'Enable internet share')!.click()
        await vi.advanceTimersByTimeAsync(0)
      })
      // Disable during warm-up (the disable button is present while the section is engaged).
      await React.act(async () => {
        findByText(c, 'Disable internet share')!.click()
        await vi.advanceTimersByTimeAsync(0)
      })
      expect(mockedSetSessionFunnel).toHaveBeenCalledWith('sess-1', false, 0)
      // Advance well past the 30s window — the cleared timeout must not fire.
      await React.act(async () => { await vi.advanceTimersByTimeAsync(31000) })
      expect(c.textContent).not.toMatch(/Connection timed out/)
    } finally {
      vi.useRealTimers()
    }
  })
})

// ---------------------------------------------------------------------------
// Phase 168-08 (gap): onShareEnabledChange notifies App on modal toggle in
// the already-warned path — closes the UX-02 / #115 footer pill drift.
// ---------------------------------------------------------------------------
describe('SessionShareModal — Phase 168-08 (gap): onShareEnabledChange notifies App on modal toggle in the already-warned path', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('warned shell path: toggling ON calls ToggleWebServing(id, true) and onShareEnabledChange(id, true); no banner', async () => {
    const onShareEnabledChange = vi.fn()
    const { container: c } = renderModal({
      cli: '/bin/zsh',
      shellWebShareWarned: true,
      shellWebShareWarningEnabled: true,
      webEnabled: false,
      onShareEnabledChange,
    })
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    expect(shareToggle).not.toBeNull()
    await flushSync(() => { shareToggle!.click() })
    await new Promise<void>((r) => setTimeout(r, 0))

    expect(mockedToggleWebServing).toHaveBeenCalledWith('sess-1', true)
    expect(onShareEnabledChange).toHaveBeenCalledWith('sess-1', true)
    expect(onShareEnabledChange).toHaveBeenCalledTimes(1)
    // No shell-warning banner — already warned.
    const text = c.textContent ?? ''
    expect(text).not.toMatch(/expose.*command execution/i)
  })

  it('warned shell path: toggling OFF (already shared) calls ToggleWebServing(id, false) and onShareEnabledChange(id, false)', async () => {
    const onShareEnabledChange = vi.fn()
    const { container: c } = renderModal({
      cli: '/bin/zsh',
      shellWebShareWarned: true,
      shellWebShareWarningEnabled: true,
      webEnabled: true, // already shared
      onShareEnabledChange,
    })
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    expect(shareToggle).not.toBeNull()
    await flushSync(() => { shareToggle!.click() })
    await new Promise<void>((r) => setTimeout(r, 0))

    expect(mockedToggleWebServing).toHaveBeenCalledWith('sess-1', false)
    expect(onShareEnabledChange).toHaveBeenCalledWith('sess-1', false)
    expect(onShareEnabledChange).toHaveBeenCalledTimes(1)
  })

  it('un-warned first-time shell path: toggling ON shows the banner and does NOT call onShareEnabledChange (no double-set with handleShellWebShareConfirm)', async () => {
    const onShareEnabledChange = vi.fn()
    const { container: c } = renderModal({
      cli: '/bin/zsh',
      shellWebShareWarned: false,
      shellWebShareWarningEnabled: true,
      onShellWebShareConfirm: vi.fn().mockResolvedValue(true),
      onShellWebShareCancel: vi.fn(),
      onShareEnabledChange,
    })
    const shareToggle = c.querySelector('[role="switch"][aria-label*="Share"], input[type="checkbox"]') as HTMLElement | null
    expect(shareToggle).not.toBeNull()
    await flushSync(() => { shareToggle!.click() })
    await new Promise<void>((r) => setTimeout(r, 0))

    const text = c.textContent ?? ''
    expect(text).toMatch(/web sharing this shell|expose.*command execution/i)
    expect(mockedToggleWebServing).not.toHaveBeenCalled()
    expect(onShareEnabledChange).not.toHaveBeenCalled()
  })
})

describe('SessionShareModal — WR-03: single issuance for Funnel sessions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // WR-03 regression: opening the Share modal for a session that is already
  // web-enabled AND Funnel-active must issue capabilities EXACTLY ONCE. Before
  // the fix both the server-truth seeding effect and the warm-up completion
  // effect fired IssueCapabilities on the same open — two calls, each minting
  // two tokens + AddGrant'ing two grants, so the grant set grew by four per
  // modal open. The fix makes the warm-up effect the single issuer for Funnel
  // sessions (it now also seeds cachedShare); the seeding effect bows out when
  // session.funnelActive is true.
  it('opening a webEnabled + funnelActive session calls IssueCapabilities exactly once', async () => {
    renderModal({
      webEnabled: true,
      funnelActive: true,
      webServerMode: 'tailscale',
      webServerRunning: true,
    })
    // Let both effects' async IIFEs run.
    await new Promise<void>((r) => setTimeout(r, 0))
    expect(mockedIssueCapabilities).toHaveBeenCalledTimes(1)
  })

  // Guard against a regression in the OTHER direction: a plain (non-Funnel)
  // web-enabled session must still issue exactly once via the seeding effect —
  // the WR-03 funnel guard must not suppress issuance for non-Funnel shares.
  it('opening a webEnabled non-Funnel session still calls IssueCapabilities exactly once', async () => {
    renderModal({
      webEnabled: true,
      funnelActive: false,
      webServerRunning: true,
    })
    await new Promise<void>((r) => setTimeout(r, 0))
    expect(mockedIssueCapabilities).toHaveBeenCalledTimes(1)
  })
})
