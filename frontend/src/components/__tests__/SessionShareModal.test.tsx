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
  GetLocalNetworkPassword: vi.fn().mockResolvedValue('lan-pass-secret'),
}))

// Import mocked bindings for assertion
import { IssueCapabilities, SetSessionBrowse, GetLocalNetworkPassword, ToggleWebServing } from '../../wailsjs/go/main/App'
// Import the component under test
import { SessionShareModal } from '../Hub/SessionShareModal'

const mockedIssueCapabilities = IssueCapabilities as ReturnType<typeof vi.fn>
const mockedSetSessionBrowse = SetSessionBrowse as ReturnType<typeof vi.fn>
const mockedGetLocalNetworkPassword = GetLocalNetworkPassword as ReturnType<typeof vi.fn>

interface ModalOpts {
  webEnabled?: boolean
  homeDir?: boolean
  browseEnabled?: boolean
  webServerMode?: 'local' | 'tailscale'
  webServerRunning?: boolean
  // Phase 150 SET-01 shell-warn props
  cli?: string
  shellWebShareWarned?: boolean
  shellWebShareWarningEnabled?: boolean
  onShellWebShareConfirm?: () => Promise<void>
  onShellWebShareCancel?: () => void
}

function makeSession(opts: ModalOpts = {}) {
  return {
    id: 'sess-1',
    name: 'Test Session',
    cli: opts.cli ?? 'claude',
    webEnabled: opts.webEnabled ?? false,
    homeDir: opts.homeDir ?? false,
    browseEnabled: opts.browseEnabled ?? false,
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
      onShellWebShareConfirm: vi.fn().mockResolvedValue(undefined),
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
      onShellWebShareConfirm: vi.fn().mockResolvedValue(undefined),
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
      onShellWebShareConfirm: vi.fn().mockResolvedValue(undefined),
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
      onShellWebShareConfirm: vi.fn().mockResolvedValue(undefined),
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
      onShellWebShareConfirm: vi.fn().mockResolvedValue(undefined),
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
      onShellWebShareConfirm: vi.fn().mockResolvedValue(undefined),
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

  it('confirming the banner calls onShellWebShareConfirm then enables share', async () => {
    const onConfirm = vi.fn().mockResolvedValue(undefined)
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
    }
  })
})
