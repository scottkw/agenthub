/**
 * Phase 95 Plan 95-03 — LinkConfirmPopover render tests.
 *
 * Wave 0 RED scaffolds (Plan 95-01 Task 2) flipped GREEN by Plan 95-03.
 *
 * XSS gate (95-RESEARCH §"Anti-Patterns"): the popover MUST render
 * untrusted URL display text via React text-rendering (textContent),
 * NEVER innerHTML / dangerouslySetInnerHTML.
 *
 * Risk types align with src/lib/urlSafety.ts getRisk return values:
 *   - 'osc8'      — OSC 8 display-vs-href divergence (Plan B: scaffold
 *                   only; popover surface exists but the live wiring
 *                   ships in v3.3 — see 95-RESEARCH §"Wave 0 Spike Outcome")
 *   - 'idn'       — non-ASCII codepoint(s) detected in hostname
 *   - 'typosquat' — host within edit-distance 1 of a known brand
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync, act } from 'react-dom'
import { LinkConfirmPopover } from '../LinkConfirmPopover'

interface RenderOpts {
  url?: string
  risk?: 'osc8' | 'idn' | 'typosquat'
  x?: number
  y?: number
  onContinue?: () => void
  onCancel?: () => void
}

function renderPopover(opts: RenderOpts = {}) {
  const onContinue = opts.onContinue ?? vi.fn()
  const onCancel = opts.onCancel ?? vi.fn()
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(LinkConfirmPopover, {
        url: opts.url ?? 'https://example.com',
        risk: opts.risk ?? 'osc8',
        x: opts.x ?? 100,
        y: opts.y ?? 100,
        onContinue,
        onCancel,
      })
    )
  })
  return { container, root, onContinue, onCancel }
}

describe('LinkConfirmPopover — Plan 95-03', () => {
  let cleanups: Array<() => void> = []

  function track(rendered: { container: HTMLElement; root: Root }) {
    cleanups.push(() => {
      flushSync(() => rendered.root.unmount())
      rendered.container.remove()
    })
    return rendered
  }

  afterEach(() => {
    for (const fn of cleanups) {
      try {
        fn()
      } catch {
        /* swallow */
      }
    }
    cleanups = []
    // Ensure no leftover dialogs from earlier tests bleed into the document.
    document.querySelectorAll('.link-confirm-popover').forEach((el) => el.remove())
  })

  it('renders risk-specific copy for risk="osc8"', () => {
    track(renderPopover({ risk: 'osc8' }))
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog).not.toBeNull()
    expect(dialog.textContent).toMatch(/displays one address but points to another/i)
  })

  it('renders risk-specific copy for risk="idn"', () => {
    track(renderPopover({ risk: 'idn' }))
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.textContent).toMatch(/internationalized characters that can spoof/i)
  })

  it('renders risk-specific copy for risk="typosquat"', () => {
    track(renderPopover({ risk: 'typosquat' }))
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.textContent).toMatch(/known impersonation pattern/i)
  })

  it('Continue button calls onContinue', () => {
    const { onContinue } = track(renderPopover())
    const btn = document.querySelector('.link-confirm-popover__btn--continue') as HTMLButtonElement | null
    expect(btn).not.toBeNull()
    flushSync(() => {
      btn!.click()
    })
    expect(onContinue).toHaveBeenCalledTimes(1)
  })

  it('Cancel button calls onCancel', () => {
    const { onCancel } = track(renderPopover())
    const btn = document.querySelector('.link-confirm-popover__btn--cancel') as HTMLButtonElement | null
    expect(btn).not.toBeNull()
    flushSync(() => {
      btn!.click()
    })
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('URL is rendered via textContent (NEVER innerHTML)', () => {
    // Attempt to inject script-tag-like content via the url prop. React's
    // default rendering escapes; the DOM should contain the literal string,
    // NOT a <script> element.
    const malicious = '<script data-malicious>alert(1)</script>'
    track(renderPopover({ url: malicious }))
    const urlEl = document.querySelector('.link-confirm-popover__url')!
    expect(urlEl).not.toBeNull()
    // The malicious string is preserved as TEXT (escaped by React).
    expect(urlEl.textContent).toBe(malicious)
    // No actual <script> children injected anywhere in the DOM:
    expect(document.querySelectorAll('script[data-malicious]').length).toBe(0)
  })

  it('focus is trapped inside the popover while open (a11y)', () => {
    // The Cancel button must receive focus on mount so a focused link
    // followed by Enter cannot accidentally auto-Continue. This is the
    // first half of the focus-trap contract (mount focus); Esc/Cancel
    // must dismiss.
    track(renderPopover())
    const cancelBtn = document.querySelector(
      '.link-confirm-popover__btn--cancel'
    ) as HTMLButtonElement | null
    expect(cancelBtn).not.toBeNull()
    expect(document.activeElement).toBe(cancelBtn)
  })

  it('Escape key calls onCancel (parity with Cancel button)', () => {
    const { onCancel } = track(renderPopover())
    flushSync(() => {
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    })
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  // ─── Additional Plan 95-03 coverage beyond Wave-0 scaffolds ───

  it('renders as a dialog with aria-modal="true" and aria-labelledby wired to the heading', () => {
    track(renderPopover())
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog).not.toBeNull()
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    const labelId = dialog.getAttribute('aria-labelledby')!
    expect(labelId).toBeTruthy()
    const heading = document.getElementById(labelId)
    expect(heading).not.toBeNull()
    expect(heading!.textContent).toMatch(/confirm link destination/i)
  })

  it('respects prefers-reduced-motion: reduce (popover class still applied; @media handles animation)', () => {
    // jsdom does not apply CSS @media queries; we verify the class invariant
    // (the @media block in style.css keys off `.link-confirm-popover`).
    // Real-browser visual confirmation lives in dev-browser UAT (Plan 95-04).
    const originalMatchMedia = window.matchMedia
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query.includes('prefers-reduced-motion'),
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })) as unknown as typeof window.matchMedia
    try {
      track(renderPopover())
      const dialog = document.querySelector('[role="dialog"]')!
      expect(dialog.classList.contains('link-confirm-popover')).toBe(true)
    } finally {
      window.matchMedia = originalMatchMedia
    }
  })

  it('does NOT use dangerouslySetInnerHTML (XSS gate — source-level invariant)', () => {
    // This is a sibling assertion to the textContent test above. The
    // 95-VALIDATION grep gate `! grep -q dangerouslySetInnerHTML
    // src/components/LinkConfirmPopover.tsx` is the binding check; this
    // runtime test asserts the rendered URL element has no children other
    // than the text node (i.e. the URL was inserted via React's text path).
    track(renderPopover({ url: 'https://example.com/<b>x</b>' }))
    const urlEl = document.querySelector('.link-confirm-popover__url')!
    // Single child node: the text node carrying the literal URL.
    expect(urlEl.childNodes.length).toBe(1)
    expect(urlEl.childNodes[0].nodeType).toBe(Node.TEXT_NODE)
    expect(urlEl.querySelector('b')).toBeNull()
  })
})

// Suppress unused-import warning if `act` is not invoked in this file.
void act
