import React, { useEffect } from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'

/**
 * SessionEndedBanner — Phase 175-06 (BUG-02 / #125).
 *
 * Rendered on the guest (remote viewer) path when the RelayClient WebSocket
 * closes because the owner ended/stopped the shared session — the server's
 * write pump calls conn.Close(StatusNormalClosure, "session ended") on
 * hub.Done() (both internal/webserver/server.go and internal/relay/server.go).
 * Without this banner a remote viewer just sees a frozen, silently dead
 * terminal with zero explanation.
 *
 * Mirrors WebGLRecoveryBanner's accessible-banner pattern: role="status" +
 * aria-live="polite" + fixed copy + dismiss button.
 *
 * Per the phase's locked decision this is a SINGLE generic notice — do NOT
 * build differentiated "owner ended" vs "network error" messaging, and do
 * NOT wire any auto-reconnect to this banner (RESEARCH anti-patterns): a
 * "session ended" close means the session is genuinely gone.
 *
 * Colorblind-safe (MEMORY: user is colorblind): the message is conveyed via
 * text + an accessible status role, never via color alone.
 *
 * Security (T-175-06-02): `reason` is the raw WebSocket CloseEvent.reason —
 * untrusted text from the wire. It is accepted here ONLY for logging/
 * branching (never rendered into the DOM, never passed to
 * dangerouslySetInnerHTML). The visible copy is always the fixed message
 * below, regardless of what `reason` contains.
 */
export interface SessionEndedBannerProps {
  onDismiss: () => void
  /**
   * Raw WebSocket CloseEvent.reason, if any. Used only for logging/
   * branching — see the Security note above. Never rendered.
   */
  reason?: string
  className?: string
}

export function SessionEndedBanner({
  onDismiss,
  reason,
  className,
}: SessionEndedBannerProps): React.ReactElement {
  useEffect(() => {
    if (reason) {
      // Logging only (T-175-06-02) — the fixed message below is the only
      // text ever rendered into the DOM for this banner.
      console.debug(`[SessionEndedBanner] close reason: ${reason}`)
    }
  }, [reason])

  const cls = ['session-ended-banner', className].filter(Boolean).join(' ')

  return (
    <div className={cls} role="status" aria-live="polite">
      <span className="session-ended-banner__message">
        Session ended — the owner stopped this session.
      </span>
      <button
        type="button"
        className="session-ended-banner__dismiss"
        aria-label="Dismiss notification"
        onClick={onDismiss}
      >
        <XMarkIcon style={{ width: 16, height: 16 }} aria-hidden="true" />
      </button>
    </div>
  )
}
