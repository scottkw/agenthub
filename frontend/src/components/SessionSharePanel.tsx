import React, { useState, useEffect, useRef } from 'react'
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'
import { GetCapabilityQRCode } from '../wailsjs/go/main/App'
import { BrowserOpenURL, ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'
import { CodeDisplay, HoldToConfirmButton } from './SessionShare/shared'

/** Formats a non-negative second count as "mm:ss" for the post-gate countdown. */
function formatCountdown(totalSeconds: number): string {
  const clamped = Math.max(0, totalSeconds)
  const mm = Math.floor(clamped / 60)
  const ss = clamped % 60
  return `${mm}:${String(ss).padStart(2, '0')}`
}

/**
 * Phase 137 / D-11: simplified props (CAP-05 two-gate stripped).
 *
 * The ownerWriteEnabled prop, the allowFileEditing / showWriteConfirm viewer
 * opt-in state, and the surfaceWriteLink gate are removed. The write link row
 * is always rendered when writeURL/writeCode are provided — the backend perm
 * matrix (D-03/D-04) is the authority, not a UI gate.
 *
 * browseEnabled is an optional display hint for the scope text only; it never
 * gates link visibility.
 */
interface SessionSharePanelProps {
  sessionId: string
  readURL: string
  writeURL: string
  readCode: string
  writeCode: string
  /**
   * Phase 137 / D-11: optional display hint for scope text.
   * true  → "Watch + browse files" scope text
   * false → "Watch only — no file access" scope text
   * No gating logic — write link is always shown when writeURL/writeCode present.
   */
  browseEnabled?: boolean
  // ---- Phase 166 FUI-04/FUI-05: Internet (public) Funnel section ----
  /** true once the daemon confirms the Funnel is live (session.funnelActive). */
  funnelActive?: boolean
  /** The read-only Funnel-base public URL (obtained by re-issuing caps after warm-up). */
  funnelUrl?: string | null
  /** true between enable and the funnelActive flip — shows the TLS warm-up state. */
  warmingUp?: boolean
  /** true when warm-up did not complete within 30s. */
  warmupTimedOut?: boolean
  /** One-click Funnel teardown — SetSessionFunnel(id, false, 0). No confirm dialog (D-13). */
  onDisableFunnel?: () => void
  /**
   * Phase 170 / FNL-08: the reusable, read-only, share-lifetime public join
   * code minted by the daemon alongside the Funnel read URL. `null`/absent
   * means no code is available yet (or the Funnel isn't live) — no row is
   * rendered in that case.
   */
  publicReadCode?: string | null

  // ---- Phase 171 / FNL-09: Danger section (public write consent gate) ----
  /**
   * Called exactly once, with the currently-selected expiry (seconds), when
   * the ≥3s hold-to-confirm gesture completes. Never called on early release
   * (R1). The panel owns the expiry <select> locally; the modal is the sole
   * RPC caller (SetSessionFunnelWrite).
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
   * remain visible). No live backend signal for this exists yet this phase
   * (see SUMMARY Deviations); this is a controlled prop for callers that can
   * detect it out-of-band.
   */
  writeGateUsed?: boolean
  /** One-click RW-only teardown (DisableSessionFunnelWrite) — no confirm dialog (asymmetric with the hold-in gate, D-13). */
  onDisableGateWrite?: () => void
}

/**
 * Per-session share panel (Phase 87 UI-SPEC Surface 1, simplified in Phase 137).
 *
 * Renders two link rows (Read-Only Link + Full Access Link) with Copy/Open/QR
 * action buttons. The QR encodes the join-code exchange URL
 * (https://host/join?code=<code>) — NOT the raw capability token (D-09).
 *
 * Only one QR is visible per session at a time: activating one QR implicitly
 * hides the other.
 *
 * Phase 137 / D-02/D-11: CAP-05 two-gate removed. The Full Access Link is
 * always rendered when writeURL/writeCode are provided — no viewer opt-in
 * required. The backend D-03/D-04 perm matrix is the access authority.
 */
export function SessionSharePanel({
  sessionId,
  readURL,
  writeURL,
  readCode,
  writeCode,
  browseEnabled = false,
  funnelActive = false,
  funnelUrl = null,
  warmingUp = false,
  warmupTimedOut = false,
  onDisableFunnel,
  publicReadCode = null,
  onGateConfirm,
  writeGateUrl = null,
  writeGateCode = null,
  writeGateExpiresAt = null,
  writeGateUsed = false,
  onDisableGateWrite,
}: SessionSharePanelProps): React.ReactElement {
  const [readCopied, setReadCopied] = useState(false)
  const [writeCopied, setWriteCopied] = useState(false)
  const [showReadQR, setShowReadQR] = useState(false)
  const [showWriteQR, setShowWriteQR] = useState(false)
  const [readQRb64, setReadQRb64] = useState<string | null>(null)
  const [writeQRb64, setWriteQRb64] = useState<string | null>(null)
  const [qrError, setQrError] = useState<string | null>(null)
  // Phase 166 FUI-05 — Internet (public) section local state.
  const [funnelCopied, setFunnelCopied] = useState(false)
  const [showFunnelQR, setShowFunnelQR] = useState(false)
  const [funnelQRb64, setFunnelQRb64] = useState<string | null>(null)
  const [funnelQRError, setFunnelQRError] = useState<string | null>(null)

  // ---- Phase 171 / FNL-09 — Danger section (public write consent gate) local state ----
  const [gateExpirySeconds, setGateExpirySeconds] = useState(900) // D-11 default: 15 minutes
  const [gateCopied, setGateCopied] = useState(false)
  const [showGateQR, setShowGateQR] = useState(false)
  const [gateQRb64, setGateQRb64] = useState<string | null>(null)
  const [gateQRError, setGateQRError] = useState<string | null>(null)
  // Live countdown tick (visual only — funnelWriteActive polling is the
  // authoritative collapse-to-Idle signal, owned by the modal).
  const [gateNowSec, setGateNowSec] = useState(() => Math.floor(Date.now() / 1000))
  useEffect(() => {
    if (writeGateExpiresAt == null) return
    const id = setInterval(() => setGateNowSec(Math.floor(Date.now() / 1000)), 1000)
    return () => clearInterval(id)
  }, [writeGateExpiresAt])
  // Focus management (UI-SPEC Focus Management): move focus to the Disable
  // button the moment the gate completes (writeGateUrl/writeGateCode go from
  // falsy to truthy).
  const disableGateBtnRef = useRef<HTMLButtonElement | null>(null)
  const hadGateResultRef = useRef(false)
  useEffect(() => {
    const hasResult = Boolean(writeGateUrl || writeGateCode)
    if (hasResult && !hadGateResultRef.current) {
      disableGateBtnRef.current?.focus()
    }
    hadGateResultRef.current = hasResult
  }, [writeGateUrl, writeGateCode])

  // The Internet section is present whenever the Funnel is engaged (warming, timed out,
  // or live). Kept out of the DOM entirely when Funnel is off to avoid an empty section.
  const funnelEngaged = warmingUp || warmupTimedOut || funnelActive

  // FNL-08 fix (M-46): the public URL a guest opens must be the reusable,
  // share-lifetime /join entry point (https://host/join?code=<publicReadCode>),
  // NOT funnelUrl. funnelUrl = readUrl = https://host/sessions/{id}?cap=<rTok>,
  // an ephemeral, grant-bound capability link that 401s "capability required"
  // once the grant rotates on a warm-up re-issue or the daemon restarts (grants
  // are in-memory). Derive the join URL from funnelUrl's origin — mirrors the
  // joinURLFor() pattern already used for the RO/Full QR path. When no reusable
  // code is present (degenerate non-Funnel path) fall back to the bare /join
  // page (State B, guest enters a code) rather than the broken cap link.
  const publicEntryUrl = ((): string | null => {
    if (!funnelUrl) return null
    try {
      const u = new URL(funnelUrl)
      const base = `${u.protocol}//${u.host}/join`
      return publicReadCode ? `${base}?code=${publicReadCode}` : base
    } catch {
      return funnelUrl
    }
  })()

  async function handleToggleFunnelQR(): Promise<void> {
    if (!publicEntryUrl) return
    if (showFunnelQR) {
      setShowFunnelQR(false)
      return
    }
    setFunnelQRError(null)
    if (!funnelQRb64) {
      try {
        // Encode the reusable /join entry URL (D-12) — a photographed QR keeps
        // working for the share lifetime, which is the intended public UX.
        const b64 = await GetCapabilityQRCode(publicEntryUrl)
        setFunnelQRb64(b64)
      } catch {
        setFunnelQRError('QR unavailable — tap to retry')
        return
      }
    }
    setShowFunnelQR(true)
  }

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

  // Phase 171 / FNL-09 — Danger section result-row QR toggle. Mirrors
  // handleToggleFunnelQR/handleToggleQR: encodes the join-code exchange URL
  // (D-09), never the raw capability token, so a photographed QR is worthless
  // after the single-use code is redeemed.
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

  function joinURLFor(capURL: string, code: string): string {
    // QR encodes the join-code exchange URL (D-09), NOT the capability token.
    // Photographing/screenshotting a QR is worthless after 5 minutes or first
    // exchange — encoding the cap directly would defeat that mitigation.
    const u = new URL(capURL)
    return `${u.protocol}//${u.host}/join?code=${code}`
  }

  async function handleToggleQR(which: 'read' | 'write'): Promise<void> {
    if (which === 'read') {
      if (showReadQR) {
        setShowReadQR(false)
        return
      }
      // Only one QR visible per session at a time (UI-SPEC Surface 1).
      setShowWriteQR(false)
      setQrError(null)
      if (!readQRb64) {
        try {
          const b64 = await GetCapabilityQRCode(joinURLFor(readURL, readCode))
          setReadQRb64(b64)
        } catch {
          setQrError('QR unavailable — tap to retry')
          return
        }
      }
      setShowReadQR(true)
      return
    }

    // Write QR: only reachable when write link is rendered.
    if (showWriteQR) {
      setShowWriteQR(false)
      return
    }
    setShowReadQR(false)
    setQrError(null)
    if (!writeQRb64) {
      try {
        const b64 = await GetCapabilityQRCode(joinURLFor(writeURL, writeCode))
        setWriteQRb64(b64)
      } catch {
        setQrError('QR unavailable — tap to retry')
        return
      }
    }
    setShowWriteQR(true)
  }

  return (
    <div className="session-share-panel" data-session-id={sessionId}>
      <div className="session-share-panel__link-row">
        <span className="session-share-panel__label">Read-Only Link</span>
        <span className="session-share-panel__url" title={readURL}>{readURL}</span>
        <div className="session-share-panel__actions">
          <button
            className="daemon-panel__btn"
            onClick={() => void handleCopy(readURL, setReadCopied)}
            aria-label="Copy read-only link to clipboard"
          >
            {readCopied ? 'Copied!' : 'Copy'}
          </button>
          <button
            className="daemon-panel__btn"
            onClick={() => BrowserOpenURL(readURL)}
            aria-label="Open read-only link in browser"
          >
            Open
          </button>
          <button
            className="daemon-panel__btn"
            onClick={() => void handleToggleQR('read')}
            aria-label={showReadQR ? 'Hide read-only QR code' : 'Show read-only QR code'}
          >
            {showReadQR ? 'Hide QR' : 'QR'}
          </button>
        </div>
      </div>
      <CodeDisplay label="Join code:" code={readCode} />
      {/* Phase 137 / D-11: scope text reflects browseEnabled prop only — no gating */}
      <p className="session-share-panel__scope" style={{ margin: '2px 0 10px', fontSize: 12, color: '#9aa5ce', lineHeight: 1.4 }}>
        {browseEnabled
          ? 'Watch the live session and browse files read-only — cannot send input.'
          : 'Watch the live session only — cannot send input or browse files.'}
      </p>
      {showReadQR && readQRb64 && (
        <img
          className="session-share-panel__qr"
          src={`data:image/png;base64,${readQRb64}`}
          width={200}
          height={200}
          alt="QR code for read-only share link"
        />
      )}

      {/* Phase 137 / D-02/D-11: Full Access Link always rendered when writeURL/writeCode present.
          CAP-05 two-gate removed — backend perm matrix (D-03/D-04) is the authority. */}
      <p className="session-share-panel__scope" style={{ margin: '2px 0 6px', fontSize: 12, color: '#9aa5ce', lineHeight: 1.4 }}>
        {browseEnabled
          ? 'Full control of the live session (send input) plus file browsing and editing.'
          : 'Full control of the live session (send input) plus file browsing. Use the toggle above to also allow file editing.'}
      </p>
      <div className="session-share-panel__link-row" data-testid="full-access-link-row">
        <span className="session-share-panel__label">Full Access Link</span>
        <span className="session-share-panel__url" title={writeURL}>{writeURL}</span>
        <div className="session-share-panel__actions">
          <button
            className="daemon-panel__btn"
            onClick={() => void handleCopy(writeURL, setWriteCopied)}
            aria-label="Copy full-access link to clipboard"
          >
            {writeCopied ? 'Copied!' : 'Copy'}
          </button>
          <button
            className="daemon-panel__btn"
            onClick={() => BrowserOpenURL(writeURL)}
            aria-label="Open full-access link in browser"
          >
            Open
          </button>
          <button
            className="daemon-panel__btn"
            onClick={() => void handleToggleQR('write')}
            aria-label={showWriteQR ? 'Hide full-access QR code' : 'Show full-access QR code'}
          >
            {showWriteQR ? 'Hide QR' : 'QR'}
          </button>
        </div>
      </div>
      <CodeDisplay label="Join code:" code={writeCode} />
      {showWriteQR && writeQRb64 && (
        <img
          className="session-share-panel__qr"
          src={`data:image/png;base64,${writeQRb64}`}
          width={200}
          height={200}
          alt="QR code for full-access share link"
        />
      )}

      {qrError && <p className="session-share-panel__error">{qrError}</p>}

      {/* Phase 166 FUI-04/FUI-05 — Internet (public) Funnel section. Only the read-only
          Funnel URL is ever shown here — never a public write link (D-12). */}
      {funnelEngaged && (
        <div className="hub-share-internet-section">
          <div className="hub-share-internet-section__heading">Internet (public)</div>

          {warmingUp && (
            <div className="hub-share-internet-section__warmup">Starting up… (TLS warming up)</div>
          )}

          {warmupTimedOut && (
            <p className="hub-share-internet-section__error">
              Connection timed out. Try disabling and re-enabling.
            </p>
          )}

          {funnelActive && publicEntryUrl && !warmingUp && (
            <>
              <div className="hub-share-internet-section__url-row">
                <span className="session-share-panel__label">Public URL (read-only):</span>
                <span className="session-share-panel__url" title={publicEntryUrl}>{publicEntryUrl}</span>
                <div className="session-share-panel__actions">
                  <button
                    type="button"
                    className="daemon-panel__btn"
                    onClick={() => void handleCopy(publicEntryUrl, setFunnelCopied)}
                    aria-label="Copy public internet URL to clipboard"
                  >
                    {funnelCopied ? 'Copied!' : 'Copy URL'}
                  </button>
                  <button
                    type="button"
                    className="daemon-panel__btn"
                    onClick={() => BrowserOpenURL(publicEntryUrl)}
                    aria-label="Open public internet URL in browser"
                  >
                    Open
                  </button>
                  <button
                    type="button"
                    className="daemon-panel__btn"
                    onClick={() => void handleToggleFunnelQR()}
                    aria-label={showFunnelQR ? 'Hide public URL QR code' : 'Show public URL QR code'}
                  >
                    {showFunnelQR ? 'Hide QR' : 'QR'}
                  </button>
                </div>
              </div>
              {publicReadCode && (
                <CodeDisplay label="Public join code (reusable):" code={publicReadCode} />
              )}
              {showFunnelQR && funnelQRb64 && (
                <img
                  className="session-share-panel__qr"
                  src={`data:image/png;base64,${funnelQRb64}`}
                  width={200}
                  height={200}
                  alt="QR code for public internet URL"
                />
              )}
              {funnelQRError && <p className="session-share-panel__error">{funnelQRError}</p>}
            </>
          )}

          <button
            type="button"
            className="hub-share-internet-section__disable"
            onClick={() => onDisableFunnel?.()}
          >
            Disable internet share
          </button>
        </div>
      )}

      {/* Phase 171 / FNL-09 — Danger section: public write consent gate.
          Physically separate block BELOW the read (Internet) section (D-06) —
          never nested inside it. Rendered under the same funnelEngaged gate as
          the read section (the RW gate is meaningless without an active/warming
          Funnel session), but the hold control itself stays disabled until
          funnelActive && !warmingUp (Interaction Contract). */}
      {funnelEngaged && (
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

          {!writeGateUrl && !writeGateCode && (
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
      )}
    </div>
  )
}
