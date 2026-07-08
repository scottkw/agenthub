import React, { useState, useEffect, useRef } from 'react'
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'
import { GetCapabilityQRCode } from '../../wailsjs/go/main/App'
import { BrowserOpenURL, ClipboardSetText } from '../../wailsjs/wailsjs/runtime/runtime'
import { CodeDisplay, HoldToConfirmButton } from './shared'

// ---------------------------------------------------------------------------
// Phase 173 / SM-04 — InternetFullAccessTab: the ENTIRE public-write /
// command-execution flow, walled off inside this one tab. Extracted verbatim
// (behavior unchanged, D-08) from SessionSharePanel.tsx's hub-funnel-write-gate
// block (:603-724) — danger explainer, fixed-enum expiry select,
// HoldToConfirmButton, armed-summary result, live countdown, and the
// focus-management effect. No tailnet/read-only link UI appears here.
// ---------------------------------------------------------------------------

/** Formats a non-negative second count as "mm:ss" for the post-gate countdown. */
function formatCountdown(totalSeconds: number): string {
  const clamped = Math.max(0, totalSeconds)
  const mm = Math.floor(clamped / 60)
  const ss = clamped % 60
  return `${mm}:${String(ss).padStart(2, '0')}`
}

/**
 * QR encodes the join-code exchange URL (D-09/T-173-02), NOT the capability
 * token. Photographing/screenshotting a QR is worthless after the single-use
 * code is redeemed. Reused verbatim from SessionSharePanel.tsx.
 */
function joinURLFor(capURL: string, code: string): string {
  const u = new URL(capURL)
  return `${u.protocol}//${u.host}/join?code=${code}`
}

export interface InternetFullAccessTabProps {
  /** true once the daemon confirms the Funnel is live (session.funnelActive). */
  funnelActive?: boolean
  /** true between enable and the funnelActive flip — shows the TLS warm-up state. */
  warmingUp?: boolean
  /**
   * Called exactly once, with the currently-selected expiry (seconds), when
   * the ≥3s hold-to-confirm gesture completes. Never called on early release
   * (R1). This tab owns the expiry <select> locally; the shell/modal is the
   * sole RPC caller (SetSessionFunnelWrite).
   */
  onGateConfirm?: (expirySeconds: number) => void
  /** The public write URL once SetSessionFunnelWrite resolves (null/absent = gate not yet confirmed). */
  writeGateUrl?: string | null
  /** The single-use write join code once SetSessionFunnelWrite resolves. */
  writeGateCode?: string | null
  /** UNIX-seconds expiry of the write grant/code — drives the live "Expires in mm:ss" countdown. */
  writeGateExpiresAt?: number | null
  /**
   * true once a guest has redeemed the write code — collapses the URL/code
   * rows to "Write code used — one writer connected" (countdown + disable
   * remain visible). No live backend signal for this exists yet this phase;
   * this is a controlled prop for callers that can detect it out-of-band.
   */
  writeGateUsed?: boolean
  /** One-click RW-only teardown (DisableSessionFunnelWrite) — no confirm dialog (asymmetric with the hold-in gate, D-13). */
  onDisableGateWrite?: () => void
}

/**
 * Renders the entire public-write consent gate as three sequenced states —
 * Idle (danger explainer + "Enable public write access…" button) → Gate open
 * (expiry select + HoldToConfirmButton, disabled until funnelActive &&
 * !warmingUp, + Cancel) → Armed (write URL / single-use code / live
 * countdown / Disable public write). The danger explainer itself is always
 * visible across all three states. Focus moves to the Disable button the
 * instant the gate arms (focus-management effect preserved verbatim from
 * SessionSharePanel.tsx).
 */
export function InternetFullAccessTab({
  funnelActive = false,
  warmingUp = false,
  onGateConfirm,
  writeGateUrl = null,
  writeGateCode = null,
  writeGateExpiresAt = null,
  writeGateUsed = false,
  onDisableGateWrite,
}: InternetFullAccessTabProps): React.ReactElement {
  // Idle -> Gate-open local state (new in the tab decomposition — reduces
  // accidental exposure by requiring a deliberate click before the
  // hold-to-confirm control is even reachable).
  const [gateOpen, setGateOpen] = useState(false)
  const [gateExpirySeconds, setGateExpirySeconds] = useState(900) // D-11 default: 15 minutes
  const [gateCopied, setGateCopied] = useState(false)
  const [showGateQR, setShowGateQR] = useState(false)
  const [gateQRb64, setGateQRb64] = useState<string | null>(null)
  const [gateQRError, setGateQRError] = useState<string | null>(null)
  // Live countdown tick (visual only — funnelWriteActive polling is the
  // authoritative collapse-to-Idle signal, owned by the shell/modal).
  const [gateNowSec, setGateNowSec] = useState(() => Math.floor(Date.now() / 1000))
  useEffect(() => {
    if (writeGateExpiresAt == null) return
    const id = setInterval(() => setGateNowSec(Math.floor(Date.now() / 1000)), 1000)
    return () => clearInterval(id)
  }, [writeGateExpiresAt])
  // Focus management (UI-SPEC Focus Management): move focus to the Disable
  // button the moment the gate completes (writeGateUrl/writeGateCode go from
  // falsy to truthy). Symmetrically, reset back to Idle once a disable
  // clears the result (writeGateUrl/writeGateCode go from truthy to falsy) —
  // re-entering the danger tab always starts from Idle, never a stale
  // gate-open form.
  const disableGateBtnRef = useRef<HTMLButtonElement | null>(null)
  const hadGateResultRef = useRef(false)
  useEffect(() => {
    const hasResult = Boolean(writeGateUrl || writeGateCode)
    if (hasResult && !hadGateResultRef.current) {
      disableGateBtnRef.current?.focus()
    }
    if (!hasResult && hadGateResultRef.current) {
      setGateOpen(false)
    }
    hadGateResultRef.current = hasResult
  }, [writeGateUrl, writeGateCode])

  async function handleCopy(url: string, setter: (v: boolean) => void): Promise<void> {
    try {
      await ClipboardSetText(url)
    } catch {
      // ClipboardSetText failed — no user-visible action needed.
      return
    }
    setter(true)
    setTimeout(() => setter(false), 1500)
  }

  // Result-row QR toggle. Mirrors ShareLinkCard/SessionSharePanel: encodes
  // the join-code exchange URL (D-09), never the raw capability token, so a
  // photographed QR is worthless after the single-use code is redeemed.
  async function handleToggleGateQR(): Promise<void> {
    if (!writeGateUrl || !writeGateCode) return
    if (showGateQR) {
      setShowGateQR(false)
      return
    }
    setGateQRError(null)
    if (!gateQRb64) {
      try {
        const b64 = await GetCapabilityQRCode(joinURLFor(writeGateUrl, writeGateCode))
        setGateQRb64(b64)
      } catch {
        setGateQRError('QR unavailable — tap to retry')
        return
      }
    }
    setShowGateQR(true)
  }

  return (
    <div className="session-share-panel__tab session-share-panel__tab--internet-fa">
      <div className="hub-funnel-write-gate">
        <div className="hub-funnel-write-gate__heading">PUBLIC WRITE ACCESS — COMMAND EXECUTION</div>

        <div className="hub-funnel-write-gate__warning">
          <ExclamationTriangleIcon className="hub-funnel-write-gate__icon" aria-hidden="true" />
          <div className="hub-funnel-write-gate__warning-heading">⚠ You are exposing a terminal to the internet</div>
          <p className="hub-funnel-write-gate__warning-body">
            Anyone with the link and code gets full command execution on this machine, running
            as your account, until you disable it or it expires (max 1 hour). A leaked link =
            remote code execution.
          </p>
        </div>

        {!writeGateUrl && !writeGateCode && !gateOpen && (
          <button
            type="button"
            className="hub-funnel-write-gate__enable"
            onClick={() => setGateOpen(true)}
          >
            Enable public write access…
          </button>
        )}

        {!writeGateUrl && !writeGateCode && gateOpen && (
          <>
            <label className="hub-funnel-write-gate__expiry">
              <span>Expires:</span>
              <select
                value={String(gateExpirySeconds)}
                onChange={(e) => setGateExpirySeconds(Number(e.target.value))}
                aria-label="Expires"
              >
                <option value={900}>15 minutes</option>
                <option value={1800}>30 minutes</option>
                <option value={3600}>1 hour</option>
              </select>
            </label>

            {!(funnelActive && !warmingUp) && (
              <p className="hub-share-internet-section__warmup">
                Waiting for internet share to finish starting up…
              </p>
            )}

            <HoldToConfirmButton
              disabled={!(funnelActive && !warmingUp)}
              onConfirm={() => onGateConfirm?.(gateExpirySeconds)}
            />

            <button
              type="button"
              className="daemon-panel__btn"
              data-testid="write-gate-cancel"
              onClick={() => setGateOpen(false)}
            >
              Cancel
            </button>
          </>
        )}

        {(writeGateUrl || writeGateCode) && (
          <div className="hub-funnel-write-gate__result">
            {!writeGateUsed ? (
              <>
                <div className="session-share-panel__link-row" data-testid="write-gate-url-row">
                  <span className="session-share-panel__label">Public write URL:</span>
                  <span className="session-share-panel__url" title={writeGateUrl ?? ''}>{writeGateUrl}</span>
                  <div className="session-share-panel__actions">
                    <button
                      type="button"
                      className="daemon-panel__btn"
                      onClick={() => void handleCopy(writeGateUrl ?? '', setGateCopied)}
                      aria-label="Copy public write link to clipboard"
                    >
                      {gateCopied ? 'Copied!' : 'Copy'}
                    </button>
                    <button
                      type="button"
                      className="daemon-panel__btn"
                      onClick={() => writeGateUrl && BrowserOpenURL(writeGateUrl)}
                      aria-label="Open public write link in browser"
                    >
                      Open
                    </button>
                    <button
                      type="button"
                      className="daemon-panel__btn"
                      onClick={() => void handleToggleGateQR()}
                      aria-label={showGateQR ? 'Hide public write QR code' : 'Show public write QR code'}
                    >
                      {showGateQR ? 'Hide QR' : 'QR'}
                    </button>
                  </div>
                </div>
                {writeGateCode && (
                  <CodeDisplay label="Single-use write code:" code={writeGateCode} />
                )}
                {showGateQR && gateQRb64 && (
                  <img
                    className="session-share-panel__qr"
                    src={`data:image/png;base64,${gateQRb64}`}
                    width={200}
                    height={200}
                    alt="QR code for public write link"
                  />
                )}
                {gateQRError && <p className="session-share-panel__error">{gateQRError}</p>}
              </>
            ) : (
              <div className="hub-funnel-write-gate__used">Write code used — one writer connected</div>
            )}

            {writeGateExpiresAt != null && (
              <div
                className={`hub-funnel-write-gate__countdown${
                  writeGateExpiresAt - gateNowSec < 60 ? ' hub-funnel-write-gate__countdown--urgent' : ''
                }`}
              >
                Expires in {formatCountdown(writeGateExpiresAt - gateNowSec)}
              </div>
            )}

            <button
              type="button"
              ref={disableGateBtnRef}
              className="hub-funnel-write-gate__disable"
              onClick={() => onDisableGateWrite?.()}
            >
              Disable public write
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
