/**
 * Phase 124 CAP-06 — HomeDirWriteWarning render tests.
 *
 * Colorblind-safe contract (release-blocking):
 *   - ⚠ glyph MUST be present in rendered output.
 *   - Literal word "Warning:" MUST be present in rendered output.
 *   - Verbatim body copy (from 124-UI-SPEC §"GUI Surface 3") must match.
 *   - onDismiss fires when the dismiss button is clicked.
 *   - Banner is NOT timer-auto-dismissed (standing caution).
 *
 * Verification performed at source level per the colorblind rule
 * (glyph + text, not color).
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { HomeDirWriteWarning } from '../HomeDirWriteWarning'

function renderBanner(onDismiss: () => void) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(HomeDirWriteWarning, { onDismiss }))
  })
  return { container, root }
}

describe('HomeDirWriteWarning', () => {
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

  it('renders the ⚠ glyph (colorblind-safe non-color signal)', () => {
    ;({ container, root } = renderBanner(vi.fn()))
    expect(container.textContent).toContain('⚠')
  })

  it('renders literal "Warning:" text token (colorblind-safe non-color signal)', () => {
    ;({ container, root } = renderBanner(vi.fn()))
    expect(container.textContent).toContain('Warning:')
  })

  it('renders verbatim heading "Warning: writes can affect your home directory"', () => {
    ;({ container, root } = renderBanner(vi.fn()))
    expect(container.textContent).toContain('Warning: writes can affect your home directory')
  })

  it('renders verbatim CAP-06 body copy (dotfiles/SSH/shell config)', () => {
    ;({ container, root } = renderBanner(vi.fn()))
    expect(container.textContent).toContain(
      "This session's working directory is your home folder."
    )
    expect(container.textContent).toContain('dotfiles, SSH keys, and shell config')
    expect(container.textContent).toContain('(~/.zshrc, ~/.ssh, ~/.claude)')
    expect(container.textContent).toContain('Protected system files are always blocked.')
  })

  it('has role="status" and aria-live="polite" (accessibility contract)', () => {
    ;({ container, root } = renderBanner(vi.fn()))
    const statusEl = container.querySelector('[role="status"]')
    expect(statusEl).not.toBeNull()
    expect(statusEl?.getAttribute('aria-live')).toBe('polite')
  })

  it('dismiss button fires onDismiss when clicked', () => {
    const onDismiss = vi.fn()
    ;({ container, root } = renderBanner(onDismiss))
    const btn = container.querySelector(
      '[aria-label="Dismiss notification"]'
    ) as HTMLButtonElement | null
    expect(btn).not.toBeNull()
    flushSync(() => {
      btn!.click()
    })
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('banner uses webgl-recovery-banner--home-write-warning modifier class', () => {
    ;({ container, root } = renderBanner(vi.fn()))
    const banner = container.querySelector('.webgl-recovery-banner--home-write-warning')
    expect(banner).not.toBeNull()
  })

  it('is NOT auto-dismissed on a timer (standing caution)', () => {
    vi.useFakeTimers()
    const onDismiss = vi.fn()
    ;({ container, root } = renderBanner(onDismiss))
    vi.advanceTimersByTime(60000)
    expect(onDismiss).not.toHaveBeenCalled()
    vi.useRealTimers()
  })
})
