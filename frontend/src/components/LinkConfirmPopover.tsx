import React, { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import type { RiskKind } from '../lib/urlSafety'

/**
 * LinkConfirmPopover — Phase 95 (LNK-03) — click-confirmation surface for
 * OSC 8 mismatch / IDN spoof / typosquat risks. Portal-rendered to escape
 * the terminal container's `overflow: hidden`.
 *
 * Trust model: `url` and `risk` are computed by Plan 95-04's TerminalPanel
 * handler from the addon-web-links callback. Both are validated upstream
 * (scheme allowlist + getRisk). This component renders them as TEXT
 * CONTENT only — never via the React unsafe-HTML escape hatch and never
 * via Element.innerHTML. The 95-VALIDATION grep gate is the binding
 * source-level invariant (see 95-RESEARCH §"Anti-Patterns").
 *
 * v3.2 status: the `osc8` branch ships in this component even though the
 * live wiring (display-vs-href divergence) is deferred to v3.3 per the
 * Wave 0 spike outcome (Plan B; see 95-RESEARCH §"Wave 0 Spike Outcome").
 * This means a v3.3 wiring-only PR can flip the slice GREEN without
 * re-touching presentation.
 */
export interface LinkConfirmPopoverProps {
  url: string
  risk: RiskKind
  /** event.clientX (page-fixed coords) */
  x: number
  /** event.clientY */
  y: number
  onContinue: () => void
  onCancel: () => void
}

/**
 * Risk-specific copy. String-only — RENDERED VIA REACT TEXT CONTENT.
 * The RiskKind type comes from urlSafety.ts; priority is osc8 → idn →
 * typosquat (see urlSafety.getRisk).
 */
const RISK_COPY: Record<RiskKind, string> = {
  osc8: 'This link displays one address but points to another. Verify the destination before continuing.',
  idn: 'This link contains internationalized characters that can spoof familiar domains.',
  typosquat: 'This domain matches a known impersonation pattern. Verify the spelling carefully.',
}

const TITLE_ID = 'link-confirm-title'

export function LinkConfirmPopover({
  url,
  risk,
  x,
  y,
  onContinue,
  onCancel,
}: LinkConfirmPopoverProps): React.ReactPortal | null {
  const popoverRef = useRef<HTMLDivElement>(null)
  const cancelButtonRef = useRef<HTMLButtonElement>(null)
  const [position, setPosition] = useState<{ left: number; top: number }>({ left: x, top: y })

  // Focus the Cancel button on mount (defensive — never auto-Continue).
  // A user holding Enter on a focused link must NOT be able to confirm
  // navigation by accident.
  useEffect(() => {
    cancelButtonRef.current?.focus()
  }, [])

  // Esc dismisses with onCancel — keyboard accessibility parity with the
  // Cancel button. Listener is attached at document level so it fires
  // regardless of which child has focus inside the popover.
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation()
        onCancel()
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onCancel])

  // Edge-clipping mitigation (95-RESEARCH Pitfall #4): after first paint,
  // measure the popover. If the click coords would push it past the right
  // or bottom viewport edge, flip the anchor (top→bottom or left→right).
  useLayoutEffect(() => {
    const node = popoverRef.current
    if (!node) return
    const rect = node.getBoundingClientRect()
    const margin = 8
    let nextLeft = x
    let nextTop = y
    if (x + rect.width + margin > window.innerWidth) {
      nextLeft = Math.max(margin, x - rect.width)
    }
    if (y + rect.height + margin > window.innerHeight) {
      nextTop = Math.max(margin, y - rect.height)
    }
    if (nextLeft !== position.left || nextTop !== position.top) {
      setPosition({ left: nextLeft, top: nextTop })
    }
    // Intentionally no dep on `position` — we only flip once per (x, y).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [x, y])

  return createPortal(
    <div
      ref={popoverRef}
      className="link-confirm-popover"
      role="dialog"
      aria-modal="true"
      aria-labelledby={TITLE_ID}
      style={{ position: 'fixed', left: position.left, top: position.top }}
    >
      <h3 id={TITLE_ID} className="link-confirm-popover__title">
        Confirm link destination
      </h3>
      <p className="link-confirm-popover__reason">{RISK_COPY[risk]}</p>
      <code className="link-confirm-popover__url">{url}</code>
      <div className="link-confirm-popover__actions">
        <button
          ref={cancelButtonRef}
          type="button"
          onClick={onCancel}
          className="link-confirm-popover__btn link-confirm-popover__btn--cancel"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={onContinue}
          className="link-confirm-popover__btn link-confirm-popover__btn--continue"
        >
          Continue to site
        </button>
      </div>
    </div>,
    document.body
  )
}
