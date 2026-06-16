import React, { useState } from 'react'
import { GetCapabilityQRCode } from '../wailsjs/go/main/App'
import { BrowserOpenURL, ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'

// CodeDisplay renders a join code with a small Copy affordance. Used for both
// the read code and the write code in the share panel. The code is displayed
// as plain text so the owner can read it aloud or transcribe it to the joiner.
function CodeDisplay({
  label,
  code,
}: {
  label: string
  code: string
}): React.ReactElement {
  const [copied, setCopied] = useState(false)
  async function handleCopyCode(): Promise<void> {
    try {
      await ClipboardSetText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // clipboard failure — code remains visible for manual copy
    }
  }
  return (
    <div
      className="session-share-panel__code-display"
      style={{ display: 'flex', alignItems: 'center', gap: 8, margin: '4px 0 8px', fontSize: 13 }}
    >
      <span style={{ color: '#9aa5ce' }}>{label}</span>
      <code
        style={{
          fontFamily: 'monospace',
          fontWeight: 700,
          letterSpacing: '0.05em',
          background: '#16161e',
          padding: '2px 8px',
          borderRadius: 3,
          userSelect: 'all',
          color: '#c0caf5',
        }}
        data-testid="join-code-text"
      >
        {code}
      </code>
      <button
        type="button"
        className="daemon-panel__btn"
        onClick={() => void handleCopyCode()}
        aria-label={`Copy ${label.toLowerCase()} to clipboard`}
        style={{ fontSize: 11, padding: '2px 8px' }}
      >
        {copied ? 'Copied!' : 'Copy'}
      </button>
    </div>
  )
}

interface SessionSharePanelProps {
  sessionId: string
  readURL: string
  writeURL: string
  readCode: string
  writeCode: string
  /**
   * Phase 124 / CAP-04: true when the owner has enabled file writes for this
   * session. Gates the "Allow file editing" viewer opt-in (Surface 2).
   * When false, the opt-in row is disabled (aria-disabled, opacity 0.6).
   */
  ownerWriteEnabled?: boolean
}

/**
 * Per-session share panel (Phase 87 UI-SPEC Surface 1).
 *
 * Renders two link rows (Read-Only Link + Full Access Link) with Copy/Open/QR
 * action buttons. The QR encodes the join-code exchange URL
 * (https://host/join?code=<code>) — NOT the raw capability token (D-09).
 *
 * Only one QR is visible per session at a time: activating one QR implicitly
 * hides the other.
 *
 * Phase 124 / CAP-05 two-gate model (WR-01 fix): the Full Access Link
 * (files.write URL + token) is ONLY surfaced when BOTH conditions hold:
 *   1. Owner-side write toggle is ON  (ownerWriteEnabled prop, Surface 1)
 *   2. Viewer has confirmed "Allow file editing" opt-in (Surface 2)
 * When either gate is off, the Full Access Link row shows a locked placeholder
 * instead of the write URL/token, preventing disclosure of the files.write cap
 * before explicit per-share consent.  The read-only token is unaffected.
 */
export function SessionSharePanel({
  sessionId,
  readURL,
  writeURL,
  readCode,
  writeCode,
  ownerWriteEnabled = false,
}: SessionSharePanelProps): React.ReactElement {
  const [readCopied, setReadCopied] = useState(false)
  const [writeCopied, setWriteCopied] = useState(false)
  const [showReadQR, setShowReadQR] = useState(false)
  const [showWriteQR, setShowWriteQR] = useState(false)
  const [readQRb64, setReadQRb64] = useState<string | null>(null)
  const [writeQRb64, setWriteQRb64] = useState<string | null>(null)
  const [qrError, setQrError] = useState<string | null>(null)

  // Phase 124 / CAP-05: "Allow file editing" viewer opt-in state.
  // Default OFF per T-124-14 (elevation of privilege if ON by default).
  const [allowFileEditing, setAllowFileEditing] = useState(false)
  // Inline confirmation state — shown when user toggles ON.
  const [showWriteConfirm, setShowWriteConfirm] = useState(false)

  // Phase 124 / CAP-05 two-gate (WR-01): the files.write link is only surfaced
  // after BOTH owner enables writes AND viewer confirms opt-in. Changing either
  // gate to false immediately hides the link (no stale token visible).
  const surfaceWriteLink = ownerWriteEnabled && allowFileEditing

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

  function joinURLFor(capURL: string, code: string): string {
    // QR encodes the join-code exchange URL (D-09), NOT the capability token.
    // Photographing/screenshotting a QR is worthless after 5 minutes or first
    // exchange — encoding the cap directly would defeat that mitigation.
    const u = new URL(capURL)
    return `${u.protocol}//${u.host}/join?code=${code}`
  }

  // Phase 124 / CAP-05: toggle handler for "Allow file editing" viewer opt-in.
  // Default OFF (T-124-14). ON is gated on ownerWriteEnabled (T-124-15).
  // Toggling ON opens inline confirmation before activating.
  function handleWriteOptinToggle(): void {
    if (!ownerWriteEnabled) return // disabled guard (T-124-15)
    if (allowFileEditing) {
      // Toggle OFF: revert immediately. QR hides because surfaceWriteLink becomes false.
      setAllowFileEditing(false)
      setShowWriteConfirm(false)
      // Collapse write QR when opt-in is revoked so stale QR is not left visible.
      setShowWriteQR(false)
      setWriteQRb64(null)
    } else {
      // Toggle ON: show inline confirmation before granting.
      setShowWriteConfirm(true)
    }
  }

  function handleWriteOptinConfirm(): void {
    setAllowFileEditing(true)
    setShowWriteConfirm(false)
  }

  function handleWriteOptinCancel(): void {
    setAllowFileEditing(false)
    setShowWriteConfirm(false)
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

    // Write QR: only reachable when surfaceWriteLink is true (button is not rendered otherwise).
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
      <p className="session-share-panel__scope" style={{ margin: '2px 0 10px', fontSize: 12, color: '#9aa5ce', lineHeight: 1.4 }}>
        Watch the live session only — cannot send input or browse files.
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

      {/* Phase 124 / CAP-05: "Allow file editing" viewer opt-in row.
          Renders ABOVE the Full Access Link row per UI-SPEC Surface 2.
          Default OFF (T-124-14). Disabled unless owner write is on (T-124-15).
          WR-01: confirming this opt-in is REQUIRED before the Full Access Link
          (files.write token) is surfaced — it is the second consent gate. */}
      <div
        className="session-share-panel__write-optin"
        aria-disabled={!ownerWriteEnabled}
        style={!ownerWriteEnabled ? { opacity: 0.6, pointerEvents: 'none' } : undefined}
      >
        <label
          className={`settings-panel__toggle-row${allowFileEditing ? ' settings-panel__toggle-row--checked' : ''}`}
          style={{ cursor: ownerWriteEnabled ? 'pointer' : 'not-allowed' }}
        >
          <input
            type="checkbox"
            className="settings-panel__toggle-input"
            role="switch"
            aria-checked={allowFileEditing}
            aria-label="Allow file editing"
            checked={allowFileEditing}
            disabled={!ownerWriteEnabled}
            onChange={handleWriteOptinToggle}
          />
          <span className="settings-panel__toggle-track">
            <span className="settings-panel__toggle-thumb" />
          </span>
          <span className="settings-panel__toggle-label">Allow file editing</span>
        </label>
        {showWriteConfirm && (
          <div className="session-share-panel__write-confirm">
            <p className="session-share-panel__write-confirm-body">
              This will allow the recipient to create, edit, delete, rename, and upload files in this session&apos;s working directory.
            </p>
            <div className="session-share-panel__write-confirm-actions">
              <button
                type="button"
                className="daemon-panel__btn"
                onClick={handleWriteOptinConfirm}
              >
                Confirm
              </button>
              <button
                type="button"
                className="daemon-panel__btn"
                onClick={handleWriteOptinCancel}
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>

      <p className="session-share-panel__scope" style={{ margin: '2px 0 6px', fontSize: 12, color: '#9aa5ce', lineHeight: 1.4 }}>
        Full control of the live session (send input) plus file browsing. Use the toggle above to also allow file editing.
      </p>
      {/* Phase 124 / CAP-05 two-gate (WR-01): Full Access Link row.
          The files.write URL and token are ONLY surfaced when surfaceWriteLink
          is true (owner enabled writes AND viewer confirmed opt-in). When either
          gate is off a locked placeholder is shown so the UI still communicates
          that a full-access link exists without disclosing the token. */}
      {surfaceWriteLink ? (
        <>
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
        </>
      ) : (
        <div className="session-share-panel__link-row session-share-panel__link-row--locked" data-testid="full-access-link-locked">
          <span className="session-share-panel__label">Full Access Link</span>
          <span className="session-share-panel__url session-share-panel__url--locked">
            {ownerWriteEnabled
              ? 'Enable “Allow file editing” above to generate the Full Access link'
              : 'Enable file writes to generate the Full Access link'}
          </span>
        </div>
      )}
      {surfaceWriteLink && showWriteQR && writeQRb64 && (
        <img
          className="session-share-panel__qr"
          src={`data:image/png;base64,${writeQRb64}`}
          width={200}
          height={200}
          alt="QR code for full-access share link"
        />
      )}

      {qrError && <p className="session-share-panel__error">{qrError}</p>}
    </div>
  )
}
