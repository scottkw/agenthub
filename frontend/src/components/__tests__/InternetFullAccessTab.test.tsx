/**
 * Phase 173 / SM-04 — InternetFullAccessTab tests.
 *
 * Redistributed from SessionSharePanel.test.tsx's FNL-09 Danger-section
 * assertions, extended for the new Idle -> Gate-open -> Armed state machine
 * introduced by this tab's decomposition (danger explainer always visible;
 * "Enable public write access…" reveals the fixed-enum expiry select +
 * HoldToConfirmButton + Cancel).
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { InternetFullAccessTab } from '../SessionShare/InternetFullAccessTab'

vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  GetCapabilityQRCode: vi.fn().mockResolvedValue(''),
}))

interface RenderOpts {
  funnelActive?: boolean
  warmingUp?: boolean
  onGateConfirm?: (expirySeconds: number) => void
  writeGateUrl?: string | null
  writeGateCode?: string | null
  writeGateExpiresAt?: number | null
  writeGateUsed?: boolean
  onDisableGateWrite?: () => void
}

function renderTab(opts: RenderOpts = {}) {
  const c = document.createElement('div')
  document.body.appendChild(c)
  const r = createRoot(c)
  flushSync(() => {
    r.render(
      React.createElement(InternetFullAccessTab, {
        funnelActive: opts.funnelActive,
        warmingUp: opts.warmingUp,
        onGateConfirm: opts.onGateConfirm,
        writeGateUrl: opts.writeGateUrl,
        writeGateCode: opts.writeGateCode,
        writeGateExpiresAt: opts.writeGateExpiresAt,
        writeGateUsed: opts.writeGateUsed,
        onDisableGateWrite: opts.onDisableGateWrite,
      }),
    )
  })
  return { container: c, root: r }
}

function openGate(container: HTMLElement): void {
  const enableBtn = Array.from(container.querySelectorAll('button')).find(
    (b) => b.textContent?.trim() === 'Enable public write access…',
  ) as HTMLElement
  act(() => enableBtn.click())
}

function holdBtn(c: HTMLElement): HTMLButtonElement {
  return c.querySelector('.hub-funnel-write-gate__hold-btn') as HTMLButtonElement
}

describe('InternetFullAccessTab', () => {
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

  it('Idle: renders the danger explainer + "Enable public write access…" button; no hold control yet', () => {
    ;({ container, root } = renderTab())
    const gate = container!.querySelector('.hub-funnel-write-gate')
    expect(gate).not.toBeNull()
    expect(container!.textContent).toContain('PUBLIC WRITE ACCESS — COMMAND EXECUTION')
    expect(container!.textContent).toContain('You are exposing a terminal to the internet')
    const enableBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Enable public write access…',
    )
    expect(enableBtn).not.toBeUndefined()
    expect(holdBtn(container!)).toBeNull()
  })

  it('consent-copy compliance: warning body literally contains "command execution" and "anyone with the link"', () => {
    ;({ container, root } = renderTab())
    const body = container!.querySelector('.hub-funnel-write-gate__warning-body')
    expect(body).not.toBeNull()
    const text = (body!.textContent ?? '').toLowerCase()
    expect(text).toContain('command execution')
    expect(text).toContain('anyone with the link')
  })

  it('Gate open: clicking "Enable public write access…" reveals the expiry select + HoldToConfirmButton + Cancel', () => {
    ;({ container, root } = renderTab({ funnelActive: true }))
    openGate(container!)
    expect(holdBtn(container!)).not.toBeNull()
    const select = container!.querySelector('select[aria-label="Expires"]')
    expect(select).not.toBeNull()
    const cancelBtn = container!.querySelector('[data-testid="write-gate-cancel"]')
    expect(cancelBtn).not.toBeNull()
  })

  it('Cancel returns to Idle (hold control + select disappear, Enable button returns)', () => {
    ;({ container, root } = renderTab({ funnelActive: true }))
    openGate(container!)
    const cancelBtn = container!.querySelector('[data-testid="write-gate-cancel"]') as HTMLElement
    act(() => cancelBtn.click())
    expect(holdBtn(container!)).toBeNull()
    const enableBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Enable public write access…',
    )
    expect(enableBtn).not.toBeUndefined()
  })

  it('R1: releasing the hold before 3s issues nothing — zero onGateConfirm calls, fill resets to 0%', () => {
    vi.useFakeTimers()
    const onGateConfirm = vi.fn()
    ;({ container, root } = renderTab({ funnelActive: true, onGateConfirm }))
    openGate(container!)
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
  })

  it('R1: completing the ≥3s hold fires exactly one onGateConfirm(expirySeconds)', () => {
    vi.useFakeTimers()
    const onGateConfirm = vi.fn()
    ;({ container, root } = renderTab({ funnelActive: true, onGateConfirm }))
    openGate(container!)
    const btn = holdBtn(container!)
    act(() => {
      btn.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerId: 1 }))
    })
    act(() => { vi.advanceTimersByTime(3000) })
    expect(onGateConfirm).toHaveBeenCalledTimes(1)
    expect(onGateConfirm).toHaveBeenCalledWith(900) // D-11 default: 15 minutes
  })

  it('warm-up gating: the hold control is disabled until funnelActive && !warmingUp', () => {
    ;({ container, root } = renderTab({ funnelActive: false, warmingUp: true }))
    openGate(container!)
    const btn = holdBtn(container!)
    expect(btn.disabled).toBe(true)
    expect(btn.getAttribute('aria-disabled')).toBe('true')
    expect(container!.textContent).toContain('Waiting for internet share to finish starting up…')
  })

  it('warm-up gating: the hold control is enabled once funnelActive && !warmingUp', () => {
    ;({ container, root } = renderTab({ funnelActive: true, warmingUp: false }))
    openGate(container!)
    const btn = holdBtn(container!)
    expect(btn.disabled).toBe(false)
    expect(btn.getAttribute('aria-disabled')).toBe('false')
  })

  it('Armed: renders write URL + single-use write code + countdown + Disable public write; hold control gone', () => {
    ;({ container, root } = renderTab({
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
    expect(holdBtn(container!)).toBeNull()
  })

  it('focus management: focus moves to the Disable-public-write button the instant the gate arms', () => {
    ;({ container, root } = renderTab({ funnelActive: true }))
    const disableBtn = (): HTMLElement | undefined =>
      Array.from(container!.querySelectorAll('button')).find(
        (b) => b.textContent?.trim() === 'Disable public write',
      )
    expect(disableBtn()).toBeUndefined()
    act(() => {
      root!.render(
        React.createElement(InternetFullAccessTab, {
          funnelActive: true,
          writeGateUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=WRITE_TOKEN',
          writeGateCode: 'WGATE-CODE',
          writeGateExpiresAt: Math.floor(Date.now() / 1000) + 895,
        }),
      )
    })
    expect(document.activeElement).toBe(disableBtn())
  })

  it('clicking "Disable public write" invokes onDisableGateWrite (single click, no confirm)', async () => {
    const onDisableGateWrite = vi.fn()
    ;({ container, root } = renderTab({
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
    ;({ container, root } = renderTab({
      funnelActive: true,
      writeGateUrl: 'https://sess.tail-scale.ts.net/sessions/abc123?cap=WRITE_TOKEN',
      writeGateCode: 'WGATE-CODE',
      writeGateExpiresAt: Math.floor(Date.now() / 1000) + 500,
      writeGateUsed: true,
    }))
    expect(container!.textContent).toContain('Write code used — one writer connected')
    expect(container!.innerHTML).not.toContain('WRITE_TOKEN')
    expect(container!.textContent).not.toContain('Single-use write code:')
    expect(container!.textContent).toMatch(/Expires in \d+:\d{2}/)
    const disableBtn = Array.from(container!.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Disable public write',
    )
    expect(disableBtn).not.toBeUndefined()
  })
})
