import React, { useState } from 'react'
import { GetCapabilityQRCode } from '../../wailsjs/go/main/App'
import { BrowserOpenURL, ClipboardSetText } from '../../wailsjs/wailsjs/runtime/runtime'
import { CodeDisplay } from './shared'

// ---------------------------------------------------------------------------
// Phase 173 / SM-06 — ShareLinkCard: the one reusable link row (title ·
// truncated URL · Copy/Open/QR · join code · scope description directly
// beneath). Replaces the four ad-hoc hand-laid link rows in
// SessionSharePanel.tsx (Read-Only, Full Access, Internet public, and the
// public-write gate result row) with a single component so every tier
// renders an identical layout.
//
// QR target: joinURLFor(url, code) — the join-code EXCHANGE url
// (https://host/join?code=<code>), never the raw capability token
// (Information Disclosure mitigation, T-173-02). Reused verbatim from
// SessionSharePanel.tsx's own joinURLFor helper.
// ---------------------------------------------------------------------------

/**
 * QR encodes the join-code exchange URL (D-09/T-173-02), NOT the capability
 * token. Photographing/screenshotting a QR is worthless after the code
 * expires or is redeemed — encoding the cap directly would defeat that
 * mitigation. Reused verbatim from SessionSharePanel.tsx.
 */
function joinURLFor(capURL: string, code: string): string {
  const u = new URL(capURL)
  return `${u.protocol}//${u.host}/join?code=${code}`
}

export interface ShareLinkCardProps {
  /** Card title, e.g. "Read-Only Link" / "Full Access Link" / "Public URL (read-only)". */
  title: string
  /** The full URL — rendered CSS-truncated with the full value in title= (D-06). */
  url: string
  /** The join code rendered via the shared CodeDisplay. */
  code: string
  /** Scope description text, rendered directly beneath the link (SM-06 — fixes orphaned scope paragraphs). */
  description: string
  /** Label passed to CodeDisplay (defaults to "Join code:"). */
  codeLabel?: string
}

/**
 * One reusable link card: title · truncated URL (full URL in title=) ·
 * Copy/Open/QR actions · join code (shared CodeDisplay) · scope description
 * directly beneath. Each instance owns independent copied/QR-open/QR-b64/
 * QR-error state — a deliberate, small encapsulation improvement over the
 * single shared qrError slot the old ad-hoc rows used (RESEARCH A1).
 */
export function ShareLinkCard({
  title,
  url,
  code,
  description,
  codeLabel = 'Join code:',
}: ShareLinkCardProps): React.ReactElement {
  const [copied, setCopied] = useState(false)
  const [qrOpen, setQrOpen] = useState(false)
  const [qrB64, setQrB64] = useState<string | null>(null)
  const [qrError, setQrError] = useState<string | null>(null)

  async function handleCopy(): Promise<void> {
    try {
      await ClipboardSetText(url)
    } catch {
      // ClipboardSetText failed — no user-visible action needed.
      return
    }
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  async function handleToggleQR(): Promise<void> {
    if (qrOpen) {
      setQrOpen(false)
      return
    }
    setQrError(null)
    if (!qrB64) {
      try {
        const b64 = await GetCapabilityQRCode(joinURLFor(url, code))
        setQrB64(b64)
      } catch {
        setQrError('QR unavailable — tap to retry')
        return
      }
    }
    setQrOpen(true)
  }

  return (
    <div className="share-linkcard">
      <div className="share-linkcard__top">
        <span className="share-linkcard__title">{title}</span>
        <span className="share-linkcard__url" title={url}>{url}</span>
        <div className="share-linkcard__actions">
          <button
            type="button"
            className="daemon-panel__btn"
            onClick={() => void handleCopy()}
            aria-label={`Copy ${title.toLowerCase()} to clipboard`}
          >
            {copied ? 'Copied!' : 'Copy'}
          </button>
          <button
            type="button"
            className="daemon-panel__btn"
            onClick={() => BrowserOpenURL(url)}
            aria-label={`Open ${title.toLowerCase()} in browser`}
          >
            Open
          </button>
          <button
            type="button"
            className="daemon-panel__btn"
            onClick={() => void handleToggleQR()}
            aria-label={qrOpen ? `Hide ${title.toLowerCase()} QR code` : `Show ${title.toLowerCase()} QR code`}
          >
            {qrOpen ? 'Hide QR' : 'QR'}
          </button>
        </div>
      </div>
      <div className="share-linkcard__join">
        <CodeDisplay label={codeLabel} code={code} />
      </div>
      <p className="share-linkcard__desc">{description}</p>
      {qrOpen && qrB64 && (
        <img
          className="session-share-panel__qr"
          src={`data:image/png;base64,${qrB64}`}
          width={200}
          height={200}
          alt={`QR code for ${title.toLowerCase()}`}
        />
      )}
      {qrError && <p className="session-share-panel__error">{qrError}</p>}
    </div>
  )
}
