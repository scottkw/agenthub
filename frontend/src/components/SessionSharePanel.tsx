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
}: SessionSharePanelProps): React.ReactElement {
  const [readCopied, setReadCopied] = useState(false)
  const [writeCopied, setWriteCopied] = useState(false)
  const [showReadQR, setShowReadQR] = useState(false)
  const [showWriteQR, setShowWriteQR] = useState(false)
  const [readQRb64, setReadQRb64] = useState<string | null>(null)
  const [writeQRb64, setWriteQRb64] = useState<string | null>(null)
  const [qrError, setQrError] = useState<string | null>(null)

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
    </div>
  )
}
