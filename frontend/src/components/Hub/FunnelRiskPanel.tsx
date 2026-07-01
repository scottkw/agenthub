import React from 'react'
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

/**
 * FunnelRiskPanel — inline risk-acknowledgment panel for the Hub Share modal (Phase 166).
 *
 * Presentational only: it renders the mandated risk copy (FUI-01), an auto-expiry
 * selector (FUI-02), a Help cross-link (FUI-06), and the two action buttons. It performs
 * NO Wails calls — the SetSessionFunnel commit happens in SessionShareModal on the
 * explicit "Enable internet share" CTA (D-02). This two-step gesture (toggle → panel →
 * CTA) is the only path that exposes a session to the public internet, shown on every
 * enable with no don't-show-again (FUI-01 / D-03).
 *
 * Tab order (per UI-SPEC Interaction Contract): expiry select → help link →
 * Keep local only → Enable internet share. DOM order matches.
 *
 * The expand transition is CSS-driven (.hub-funnel-risk-panel--open, max-height) and
 * already guarded by prefers-reduced-motion in style.css.
 */

// D-05/D-07: fixed preset enum. 0 is the "no auto-expiry" sentinel (Until I disable).
// The UI cannot submit an arbitrary integer (RESEARCH V5 / T-166-05).
const EXPIRY_OPTIONS: ReadonlyArray<{ value: number; label: string }> = [
  { value: 1800, label: '30 minutes' },
  { value: 3600, label: '1 hour' },
  { value: 14400, label: '4 hours' },
  { value: 28800, label: '8 hours' },
  { value: 0, label: 'Until I disable' },
]

const RISK_STATEMENT =
  'This makes the session reachable from the public internet. The join code is the only access gate for its lifetime. Prefer short-lived, read-only shares.'

export interface FunnelRiskPanelProps {
  /** Drives the .hub-funnel-risk-panel--open expand modifier. */
  open: boolean
  /** Currently-selected auto-expiry preset (seconds; 0 = no expiry). */
  expirySeconds: number
  onExpiryChange: (seconds: number) => void
  onEnable: () => void
  onCancel: () => void
  onOpenHelp: () => void
}

export function FunnelRiskPanel({
  open,
  expirySeconds,
  onExpiryChange,
  onEnable,
  onCancel,
  onOpenHelp,
}: FunnelRiskPanelProps): React.ReactElement {
  return (
    <div
      className={`hub-funnel-risk-panel${open ? ' hub-funnel-risk-panel--open' : ''}`}
      role="group"
      aria-label="Internet sharing risk acknowledgment"
    >
      <div className="hub-funnel-risk-panel__warning">
        <ExclamationTriangleIcon
          className="hub-funnel-risk-panel__icon"
          aria-hidden="true"
        />
        <span className="hub-funnel-risk-panel__text">{RISK_STATEMENT}</span>
      </div>

      <label className="hub-funnel-risk-panel__expiry">
        <span>Auto-expire:</span>
        <select
          value={String(expirySeconds)}
          onChange={(e) => onExpiryChange(Number(e.target.value))}
          aria-label="Auto-expire"
        >
          {EXPIRY_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </label>

      <button
        type="button"
        className="hub-funnel-risk-panel__help-link"
        onClick={onOpenHelp}
      >
        Want tighter containment? See the Sharing Guide →
      </button>

      <div className="hub-funnel-risk-panel__actions">
        <button type="button" className="hub-share-internet-section__disable" onClick={onCancel}>
          Keep local only
        </button>
        <button
          type="button"
          className="link-confirm-popover__btn link-confirm-popover__btn--continue"
          onClick={onEnable}
        >
          Enable internet share
        </button>
      </div>
    </div>
  )
}
