/**
 * Phase 101-03 SHELL-08 — ShellWebShareBanner render tests.
 *
 * Verbatim copy locked by 101-UI-SPEC §Web-share security banner copy:
 *   Heading:        "Web sharing this shell will expose arbitrary command execution."
 *   Body line 1:    "You are about to share 'SESSIONNAME'. Anyone on your tailnet who can
 *                   reach the daemon will be able to type commands as your user account."
 *   Body line 2:    "Read-only viewers cannot type, but commands you run remain visible to them."
 *   Primary CTA:    "Enable web sharing"
 *   Confirming:     "Enabling…" (U+2026 ellipsis)
 *   Secondary CTA:  "Cancel"
 *   Dismiss aria:   "Dismiss security warning"
 *
 * Differs from PluginToggleBanner: role="alert" + aria-live="assertive" (action-blocking,
 * not informational). Focus moves to Cancel on mount (safe action per QuitConfirmModal precedent).
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync, act } from 'react-dom'
import { ShellWebShareBanner } from '../ShellWebShareBanner'

interface Props {
  sessionName: string
  onConfirm: () => void
  onCancel: () => void
}

function render(props: Props) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(ShellWebShareBanner, props))
  })
  return { container, root }
}

describe('ShellWebShareBanner', () => {
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

  it('renders heading verbatim', () => {
    ;({ container, root } = render({
      sessionName: 'scratch-bash',
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    }))
    expect(container.textContent).toContain(
      'Web sharing this shell will expose arbitrary command execution.'
    )
  })

  it('renders body with sessionName interpolated and both body sentences', () => {
    ;({ container, root } = render({
      sessionName: 'scratch-bash',
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    }))
    expect(container.textContent).toContain(
      "You are about to share 'scratch-bash'. Anyone on your tailnet who can reach the daemon will be able to type commands as your user account."
    )
    expect(container.textContent).toContain(
      'Read-only viewers cannot type, but commands you run remain visible to them.'
    )
  })

  it("primary CTA reads 'Enable web sharing' verbatim", () => {
    ;({ container, root } = render({
      sessionName: 'scratch-bash',
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    }))
    const buttons = Array.from(container.querySelectorAll('button'))
    const primary = buttons.find((b) => b.textContent === 'Enable web sharing')
    expect(primary).toBeTruthy()
  })

  it("secondary CTA reads 'Cancel' verbatim", () => {
    ;({ container, root } = render({
      sessionName: 'scratch-bash',
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    }))
    const buttons = Array.from(container.querySelectorAll('button'))
    const secondary = buttons.find((b) => b.textContent === 'Cancel')
    expect(secondary).toBeTruthy()
  })

  it("dismiss button aria-label is 'Dismiss security warning'", () => {
    ;({ container, root } = render({
      sessionName: 'scratch-bash',
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    }))
    const dismiss = container.querySelector('[aria-label="Dismiss security warning"]')
    expect(dismiss).not.toBeNull()
  })

  it('root has role="alert" and aria-live="assertive" (NOT polite — differs from PluginToggleBanner)', () => {
    ;({ container, root } = render({
      sessionName: 'scratch-bash',
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    }))
    const el = container.querySelector('[role="alert"]')
    expect(el).not.toBeNull()
    expect(el?.getAttribute('aria-live')).toBe('assertive')
  })

  it('focuses Cancel button on mount (safe action per QuitConfirmModal precedent)', () => {
    ;({ container, root } = render({
      sessionName: 'scratch-bash',
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    }))
    expect(document.activeElement?.textContent).toBe('Cancel')
  })

  it('Esc keydown fires onCancel and not onConfirm', () => {
    const onCancel = vi.fn()
    const onConfirm = vi.fn()
    ;({ container, root } = render({ sessionName: 's', onConfirm, onCancel }))
    flushSync(() => {
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    })
    expect(onCancel).toHaveBeenCalledTimes(1)
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('clicking Cancel fires onCancel and not onConfirm', () => {
    const onCancel = vi.fn()
    const onConfirm = vi.fn()
    ;({ container, root } = render({ sessionName: 's', onConfirm, onCancel }))
    const buttons = Array.from(container.querySelectorAll('button'))
    const cancel = buttons.find((b) => b.textContent === 'Cancel') as HTMLButtonElement
    flushSync(() => cancel.click())
    expect(onCancel).toHaveBeenCalledTimes(1)
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('clicking dismiss (×) fires onCancel', () => {
    const onCancel = vi.fn()
    ;({ container, root } = render({ sessionName: 's', onConfirm: vi.fn(), onCancel }))
    const dismiss = container.querySelector(
      '[aria-label="Dismiss security warning"]'
    ) as HTMLButtonElement
    flushSync(() => dismiss.click())
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it("clicking 'Enable web sharing' fires onConfirm", () => {
    const onConfirm = vi.fn()
    ;({ container, root } = render({ sessionName: 's', onConfirm, onCancel: vi.fn() }))
    const buttons = Array.from(container.querySelectorAll('button'))
    const primary = buttons.find((b) => b.textContent === 'Enable web sharing') as HTMLButtonElement
    flushSync(() => primary.click())
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it("after confirm click, primary shows 'Enabling…' and aria-busy=true; both buttons disabled", () => {
    const onConfirm = vi.fn()
    ;({ container, root } = render({ sessionName: 's', onConfirm, onCancel: vi.fn() }))
    const buttons = () => Array.from(container!.querySelectorAll('button'))
    const primary = buttons().find((b) => b.textContent === 'Enable web sharing') as HTMLButtonElement
    flushSync(() => primary.click())

    // After click, primary button text changes to "Enabling…".
    const primaryAfter = buttons().find((b) =>
      b.textContent === 'Enabling…' || b.textContent === 'Enabling…'
    ) as HTMLButtonElement | undefined
    expect(primaryAfter).toBeTruthy()
    expect(primaryAfter!.disabled).toBe(true)

    // Cancel button is also disabled while enabling.
    const cancelAfter = buttons().find((b) => b.textContent === 'Cancel') as HTMLButtonElement | undefined
    expect(cancelAfter).toBeTruthy()
    expect(cancelAfter!.disabled).toBe(true)

    // Root has aria-busy="true".
    const rootEl = container!.querySelector('[role="alert"]') as HTMLElement
    expect(rootEl.getAttribute('aria-busy')).toBe('true')
  })

  it('BEM class includes both webgl-recovery-banner and webgl-recovery-banner--shell-warning', () => {
    ;({ container, root } = render({
      sessionName: 's',
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    }))
    const rootEl = container.firstElementChild as HTMLElement
    expect(rootEl.className).toContain('webgl-recovery-banner')
    expect(rootEl.className).toContain('webgl-recovery-banner--shell-warning')
  })
})

// Silence unused-import lint in case act() is not invoked above; keep import for any future expansion.
void act
