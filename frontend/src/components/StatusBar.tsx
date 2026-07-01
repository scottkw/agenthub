import React from 'react'

export interface StatusBarProps {
  sessionId: string
  webServerRunning: boolean
  webEnabled: boolean
  onShareSession: () => void
}

/**
 * Per-tab status bar showing web-serving state for the current session.
 * Always rendered at 32px height regardless of state — no layout reflow on toggle.
 *
 * Phase 87 cleanup: the raw session URL and QR button were removed. Sharing
 * is the Hub card's job (SessionShareModal renders cap-bearing Read-Only
 * and Full Access links with Copy/Open/QR actions per link). The status bar
 * owns state display + a single entry point into the Share modal.
 *
 * Phase 168-05 (UX-02, D-13/D-14): the button no longer toggles web sharing
 * directly — one label ("Share Session") covers both the OFF and ON states,
 * and clicking it always opens the (App.tsx-lifted) Share modal for the
 * active session. Toggling/disabling only happens inside that modal, which
 * is now the single source of truth (eliminates the button↔modal state
 * drift that was #115).
 */
export function StatusBar({
  webServerRunning,
  webEnabled,
  onShareSession,
}: StatusBarProps): React.ReactElement {
  return (
    <div className="tab-status-bar">
      {!webServerRunning && (
        <span className="tab-status-bar__state tab-status-bar__state--inactive">
          WEB SERVER NOT RUNNING
        </span>
      )}

      {webServerRunning && !webEnabled && (
        <span className="tab-status-bar__state tab-status-bar__state--off">WEB OFF</span>
      )}

      {webServerRunning && webEnabled && (
        <span className="tab-status-bar__state tab-status-bar__state--on">WEB ON</span>
      )}

      {webServerRunning && (
        <button
          className="tab-status-bar__btn"
          onClick={onShareSession}
          title="Share this session"
        >
          Share Session
        </button>
      )}
    </div>
  )
}
