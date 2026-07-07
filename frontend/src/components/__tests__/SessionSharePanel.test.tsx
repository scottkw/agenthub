/**
 * Phase 137 / SHARE-06 — SessionSharePanel simplified contract (post-CAP-05).
 *
 * The CAP-05 two-gate tests (owner-write prop, viewer opt-in state, confirm dialog)
 * are retired in Phase 137 — the write link is now always shown when writeURL/writeCode
 * are provided (no viewer opt-in gate). The browseEnabled prop controls scope text only.
 *
 * Surviving behaviors:
 *   - Write link row always rendered when writeURL/writeCode provided (no gate)
 *   - Scope text reflects browseEnabled prop: "Watch only" vs "Watch + browse files"
 *   - QR/clipboard actions work (SHARE-04 — unchanged)
 *   - Read code renders as plain text (unchanged)
 *   - "Join code:" label present (unchanged)
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { SessionSharePanel } from '../SessionSharePanel'

// Mock Wails runtime modules used by SessionSharePanel
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  GetCapabilityQRCode: vi.fn().mockResolvedValue(''),
}))

// Import mocked runtime bindings for assertion (Phase 166 FUI-05)
import { ClipboardSetText, BrowserOpenURL } from '../../wailsjs/wailsjs/runtime/runtime'

interface RenderOpts {
  browseEnabled?: boolean
  writeURL?: string
  writeCode?: string
  // Phase 166 FUI-04/FUI-05 Internet section
  funnelActive?: boolean
  funnelUrl?: string | null
  warmingUp?: boolean
  warmupTimedOut?: boolean
  onDisableFunnel?: () => void
  // Phase 170 / FNL-08 — reusable public read join code
  publicReadCode?: string | null
  // Phase 171 / FNL-09 — Danger section (public write consent gate)
  onGateConfirm?: (expirySeconds: number) => void
  writeGateUrl?: string | null
  writeGateCode?: string | null
  writeGateExpiresAt?: number | null
  writeGateUsed?: boolean
  onDisableGateWrite?: () => void
}

function renderPanel(opts: RenderOpts = {}) {
  const c = document.createElement('div')
  document.body.appendChild(c)
  const r = createRoot(c)
  flushSync(() => {
    r.render(
      React.createElement(SessionSharePanel, {
        sessionId: 'sess-1',
        readURL: 'https://example.com/r',
        writeURL: opts.writeURL ?? 'https://example.com/w',
        readCode: 'read-code',
        writeCode: opts.writeCode ?? 'write-code',
        browseEnabled: opts.browseEnabled ?? false,
        funnelActive: opts.funnelActive,
        funnelUrl: opts.funnelUrl,
        warmingUp: opts.warmingUp,
        warmupTimedOut: opts.warmupTimedOut,
        onDisableFunnel: opts.onDisableFunnel,
        publicReadCode: opts.publicReadCode,
        onGateConfirm: opts.onGateConfirm,
        writeGateUrl: opts.writeGateUrl,
        writeGateCode: opts.writeGateCode,
        writeGateExpiresAt: opts.writeGateExpiresAt,
        writeGateUsed: opts.writeGateUsed,
        onDisableGateWrite: opts.onDisableGateWrite,
      })
    )
  })
  return { container: c, root: r }
}

describe('SessionSharePanel — simplified write link (post-CAP-05)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

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

  it('write link row always rendered when writeURL/writeCode are provided (no gate)', () => {
    ;({ container, root } = renderPanel({
      writeURL: 'https://example.com/w?cap=WRITE_TOKEN_ALWAYS_VISIBLE',
    }))
    // The write URL must be in the DOM immediately — no opt-in required
    expect(container!.innerHTML).toContain('WRITE_TOKEN_ALWAYS_VISIBLE')
  })

  it('write code always rendered when writeCode is provided (no gate)', () => {
    ;({ container, root } = renderPanel({ writeCode: 'explicit-write-code' }))
    expect(container!.textContent).toContain('explicit-write-code')
  })

  it('no locked placeholder row present (CAP-05 two-gate removed)', () => {
    ;({ container, root } = renderPanel())
    // The old two-gate surfaced a locked placeholder when gates were closed.
    // Post-CAP-05 there should be no locked/hidden placeholder.
    const lockedRow = container!.querySelector('[data-testid="full-access-link-locked"]')
    expect(lockedRow).toBeNull()
  })
})

describe('SessionSharePanel — scope text (browseEnabled prop)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

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

  it('browseEnabled=false shows "watch only" scope text (no file access)', () => {
    ;({ container, root } = renderPanel({ browseEnabled: false }))
    const text = container!.textContent ?? ''
    // Should state that view-only has no file access
    expect(text).toMatch(/cannot send input or browse files|watch only|no file access/i)
  })

  it('browseEnabled=true reflects file browsing in scope text', () => {
    ;({ container, root } = renderPanel({ browseEnabled: true }))
    const text = container!.textContent ?? ''
    // Should reference browsing or file access in scope
    expect(text).toMatch(/browse|file access|full control/i)
  })
})

describe('SessionSharePanel — join code text display (unchanged)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

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

  it('renders the read code as plain text immediately (no user click required)', () => {
    ;({ container, root } = renderPanel())
    expect(container!.textContent).toContain('read-code')
    expect(container!.textContent).toContain('Join code:')
  })

  it('renders the write code as plain text (no gate in post-CAP-05 panel)', () => {
    ;({ container, root } = renderPanel({ writeCode: 'write-code' }))
    expect(container!.textContent).toContain('write-code')
  })
})

// v3.5 UAT relabel (#24): scope clarity assertions (unchanged from prior version).
describe('SessionSharePanel — link scope clarity (v3.5 relabel)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

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

  it('states the View-Only scope: session view only, no file access', () => {
    ;({ container, root } = renderPanel())
    expect(container!.textContent).toContain('cannot send input or browse files')
  })

  it('states the Full Access scope: full session control plus file browsing', () => {
    ;({ container, root } = renderPanel())
    expect(container!.textContent).toContain('Full control of the live session')
  })
})

// ---------------------------------------------------------------------------
// Phase 166 — FUI-04/FUI-05 — Internet (public) section
// ---------------------------------------------------------------------------
const mockedClipboardSetText = ClipboardSetText as ReturnType<typeof vi.fn>
const mockedBrowserOpenURL = BrowserOpenURL as ReturnType<typeof vi.fn>

describe('SessionSharePanel — FUI-05 Internet (public) section', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

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

  it('does not render the Internet section when funnel is not engaged', () => {
    ;({ container, root } = renderPanel())
    expect(container!.querySelector('.hub-share-internet-section')).toBeNull()
  })

  it('warmingUp renders the "Starting up… (TLS warming up)" state and no live URL actions', () => {
    ;({ container, root } = renderPanel({ warmingUp: true }))
    const section = container!.querySelector('.hub-share-internet-section')
    expect(section).not.toBeNull()
    expect(container!.textContent).toMatch(/Starting up… \(TLS warming up\)/)
    // No live "Copy URL" action while warming up
    const copyUrl = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Copy URL',
    )
    expect(copyUrl).toBeUndefined()
  })

  // FNL-08 fix (M-46): the public URL a guest opens must be the reusable,
  // share-lifetime /join entry point (https://host/join?code=<publicReadCode>),
  // NOT the ephemeral per-session capability link (funnelUrl = readUrl =
  // https://host/sessions/{id}?cap=<rTok>). A cap link is grant-bound and 401s
  // "capability required" once the grant rotates on a warm-up re-issue or the
  // daemon restarts — see 170-UAT.md gap.
  it('funnelActive renders the reusable /join entry URL (not the raw cap link) with Copy URL / Open / QR actions', () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=RO_TOKEN',
      publicReadCode: 'PUB1234',
    }))
    expect(container!.textContent).toMatch(/Public URL \(read-only\)/)
    expect(container!.innerHTML).toContain('https://sess.tail-scale.ts.net/join?code=PUB1234')
    // The raw capability token and the /sessions/{id} viewer path must NEVER be
    // handed to the public guest — that is exactly what dead-ends on 401.
    expect(container!.innerHTML).not.toContain('RO_TOKEN')
    expect(container!.innerHTML).not.toContain('/sessions/abc123')
    const labels = Array.from(container!.querySelectorAll('button')).map((b) => b.textContent?.trim())
    expect(labels).toContain('Copy URL')
    expect(labels.some((l) => l === 'Open')).toBe(true)
  })

  it('Copy URL copies the reusable /join entry URL (ClipboardSetText, not the cap link)', async () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=RO_TOKEN',
      publicReadCode: 'PUB1234',
    }))
    const copyBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Copy URL',
    ) as HTMLElement
    await flushSync(() => copyBtn.click())
    expect(mockedClipboardSetText).toHaveBeenCalledWith('https://sess.tail-scale.ts.net/join?code=PUB1234')
  })

  it('Open opens the reusable /join entry URL (BrowserOpenURL, not the cap link)', async () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=RO_TOKEN',
      publicReadCode: 'PUB1234',
    }))
    const openBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Open' && b.getAttribute('aria-label')?.includes('public'),
    ) as HTMLElement
    await flushSync(() => openBtn.click())
    expect(mockedBrowserOpenURL).toHaveBeenCalledWith('https://sess.tail-scale.ts.net/join?code=PUB1234')
  })

  it('does NOT render a public write link in the Internet section (D-12)', () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
      writeURL: 'https://example.com/w?cap=SECRET_WRITE',
    }))
    const section = container!.querySelector('.hub-share-internet-section')!
    // The write cap token must never appear inside the Internet (public) section.
    expect(section.innerHTML).not.toContain('SECRET_WRITE')
  })

  it('clicking "Disable internet share" invokes onDisableFunnel (single click, no confirm)', async () => {
    const onDisableFunnel = vi.fn()
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
      onDisableFunnel,
    }))
    const disableBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Disable internet share',
    ) as HTMLElement
    expect(disableBtn).not.toBeUndefined()
    await flushSync(() => disableBtn.click())
    expect(onDisableFunnel).toHaveBeenCalledTimes(1)
  })

  it('warmupTimedOut renders the timeout error copy', () => {
    ;({ container, root } = renderPanel({ warmupTimedOut: true }))
    expect(container!.querySelector('.hub-share-internet-section__error')).not.toBeNull()
    expect(container!.textContent).toMatch(/Connection timed out\. Try disabling and re-enabling\./)
  })
})

// ---------------------------------------------------------------------------
// Phase 170 / FNL-08 — reusable public join code row
// ---------------------------------------------------------------------------
describe('SessionSharePanel — FNL-08 reusable public join code', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

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

  it('renders the reusable public join code row with its reusability label when publicReadCode is present', () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
      publicReadCode: 'PUB1234',
    }))
    expect(container!.textContent).toContain('Public join code (reusable):')
    const codeEls = Array.from(container!.querySelectorAll('[data-testid="join-code-text"]'))
    expect(codeEls.some((el) => el.textContent === 'PUB1234')).toBe(true)
  })

  it('does NOT render a public code row when publicReadCode is absent (negative case)', () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      funnelUrl: 'https://sess.tail-scale.ts.net/',
    }))
    // The public URL row still renders...
    expect(container!.textContent).toMatch(/Public URL \(read-only\)/)
    // ...but no reusable-code label/row appears.
    expect(container!.textContent).not.toContain('Public join code (reusable):')
  })
})

// ---------------------------------------------------------------------------
// Phase 171 / FNL-09 — Danger section: public write consent gate
// ---------------------------------------------------------------------------
describe('SessionSharePanel — FNL-09 Danger section (public write consent gate)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

  afterEach(() => {
    if (root) {
      flushSync(() => root!.unmount())
      root = undefined
    }
    if (container) {
      container.remove()
      container = undefined
    }
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  function holdBtn(c: HTMLElement): HTMLButtonElement {
    return c.querySelector('.hub-funnel-write-gate__hold-btn') as HTMLButtonElement
  }

  it('does not render the Danger section when funnel is not engaged', () => {
    ;({ container, root } = renderPanel())
    expect(container!.querySelector('.hub-funnel-write-gate')).toBeNull()
  })

  it('renders the Danger section (heading + warning) when funnel is engaged', () => {
    ;({ container, root } = renderPanel({ funnelActive: true }))
    const gate = container!.querySelector('.hub-funnel-write-gate')
    expect(gate).not.toBeNull()
    expect(container!.textContent).toContain('PUBLIC WRITE ACCESS — COMMAND EXECUTION')
    expect(container!.textContent).toContain('You are exposing a terminal to the internet')
  })

  it('consent-copy compliance (SPEC Prohibition #4): warning body literally contains "command execution" and "anyone with the link"', () => {
    ;({ container, root } = renderPanel({ funnelActive: true }))
    const body = container!.querySelector('.hub-funnel-write-gate__warning-body')
    expect(body).not.toBeNull()
    const text = (body!.textContent ?? '').toLowerCase()
    expect(text).toContain('command execution')
    expect(text).toContain('anyone with the link')
  })

  it('R1: releasing the hold before 3s issues nothing — zero onGateConfirm calls, fill resets to 0%, label reverts', () => {
    vi.useFakeTimers()
    const onGateConfirm = vi.fn()
    ;({ container, root } = renderPanel({ funnelActive: true, onGateConfirm }))
    const btn = holdBtn(container!)
    act(() => {
      btn.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerId: 1 }))
    })
    act(() => { vi.advanceTimersByTime(1000) })
    act(() => {
      btn.dispatchEvent(new PointerEvent('pointerup', { bubbles: true, pointerId: 1 }))
    })
    expect(onGateConfirm).not.toHaveBeenCalled()
    const fill = container!.querySelector('.hub-funnel-write-gate__hold-fill') as HTMLElement
    expect(fill.style.width).toBe('0%')
    expect(container!.textContent).toContain('Hold 3s to confirm')
  })

  it('R1: completing the ≥3s hold fires exactly one onGateConfirm(expirySeconds) and reveals the result block', () => {
    vi.useFakeTimers()
    const onGateConfirm = vi.fn()
    ;({ container, root } = renderPanel({ funnelActive: true, onGateConfirm }))
    const btn = holdBtn(container!)
    act(() => {
      btn.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerId: 1 }))
    })
    act(() => { vi.advanceTimersByTime(3000) })
    expect(onGateConfirm).toHaveBeenCalledTimes(1)
    expect(onGateConfirm).toHaveBeenCalledWith(900) // D-11 default: 15 minutes
  })

  it('keyboard equivalent: Space/Enter keydown drives the same hold; early keyup issues nothing', () => {
    vi.useFakeTimers()
    const onGateConfirm = vi.fn()
    ;({ container, root } = renderPanel({ funnelActive: true, onGateConfirm }))
    const btn = holdBtn(container!)
    act(() => {
      btn.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true }))
    })
    act(() => { vi.advanceTimersByTime(1000) })
    act(() => {
      btn.dispatchEvent(new KeyboardEvent('keyup', { key: ' ', bubbles: true }))
    })
    expect(onGateConfirm).not.toHaveBeenCalled()
  })

  it('keyboard equivalent: holding Space/Enter for ≥3s fires exactly one onGateConfirm', () => {
    vi.useFakeTimers()
    const onGateConfirm = vi.fn()
    ;({ container, root } = renderPanel({ funnelActive: true, onGateConfirm }))
    const btn = holdBtn(container!)
    act(() => {
      btn.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    })
    act(() => { vi.advanceTimersByTime(3000) })
    expect(onGateConfirm).toHaveBeenCalledTimes(1)
  })

  it('warm-up gating: the hold control is disabled (aria-disabled) until funnelActive && !warmingUp', () => {
    ;({ container, root } = renderPanel({ funnelActive: false, warmingUp: true }))
    const btn = holdBtn(container!)
    expect(btn.disabled).toBe(true)
    expect(btn.getAttribute('aria-disabled')).toBe('true')
    expect(container!.textContent).toContain('Waiting for internet share to finish starting up…')
  })

  it('warm-up gating: the hold control is enabled once funnelActive && !warmingUp', () => {
    ;({ container, root } = renderPanel({ funnelActive: true, warmingUp: false }))
    const btn = holdBtn(container!)
    expect(btn.disabled).toBe(false)
    expect(btn.getAttribute('aria-disabled')).toBe('false')
  })

  it('renders the post-gate result block: public write URL + single-use write code + countdown + disable button', () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      writeGateUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=WRITE_TOKEN',
      writeGateCode: 'WGATE-CODE',
      writeGateExpiresAt: Math.floor(Date.now() / 1000) + 895,
    }))
    expect(container!.textContent).toContain('Public write URL:')
    expect(container!.innerHTML).toContain('WRITE_TOKEN')
    expect(container!.textContent).toContain('Single-use write code:')
    const codeEls = Array.from(container!.querySelectorAll('[data-testid="join-code-text"]'))
    expect(codeEls.some((el) => el.textContent === 'WGATE-CODE')).toBe(true)
    expect(container!.textContent).toMatch(/Expires in \d+:\d{2}/)
    const disableBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Disable public write',
    )
    expect(disableBtn).not.toBeUndefined()
    // Once the gate is confirmed, the hold control and expiry select disappear.
    expect(container!.querySelector('.hub-funnel-write-gate__hold-btn')).toBeNull()
  })

  it('clicking "Disable public write" invokes onDisableGateWrite (single click, no confirm)', async () => {
    const onDisableGateWrite = vi.fn()
    ;({ container, root } = renderPanel({
      funnelActive: true,
      writeGateUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=WRITE_TOKEN',
      writeGateCode: 'WGATE-CODE',
      onDisableGateWrite,
    }))
    const disableBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Disable public write',
    ) as HTMLElement
    await flushSync(() => disableBtn.click())
    expect(onDisableGateWrite).toHaveBeenCalledTimes(1)
  })

  it('used state: collapses the URL/code rows to "Write code used — one writer connected" while keeping countdown + disable', () => {
    ;({ container, root } = renderPanel({
      funnelActive: true,
      writeGateUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=WRITE_TOKEN',
      writeGateCode: 'WGATE-CODE',
      writeGateExpiresAt: Math.floor(Date.now() / 1000) + 500,
      writeGateUsed: true,
    }))
    expect(container!.textContent).toContain('Write code used — one writer connected')
    expect(container!.innerHTML).not.toContain('WRITE_TOKEN')
    expect(container!.textContent).not.toContain('Single-use write code:')
    // Countdown and disable button remain visible.
    expect(container!.textContent).toMatch(/Expires in \d+:\d{2}/)
    const disableBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Disable public write',
    )
    expect(disableBtn).not.toBeUndefined()
  })
})
