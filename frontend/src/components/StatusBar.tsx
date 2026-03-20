import React from 'react'

export interface StatusBarProps {
  sessionId: string
  webServerRunning: boolean
  webEnabled: boolean
  sessionURL: string | undefined
  onToggleWeb: () => void
  onShowQR: () => void
}

/**
 * Per-tab status bar showing web-serving state for the current session.
 * Always rendered at 32px height regardless of state — no layout reflow on toggle.
 */
export function StatusBar({
  webServerRunning,
  webEnabled,
  sessionURL,
  onToggleWeb,
  onShowQR,
}: StatusBarProps): React.ReactElement {
  return (
    <div className="tab-status-bar">
      {!webServerRunning && (
        <span className="tab-status-bar__state tab-status-bar__state--inactive">
          WEB SERVER NOT RUNNING
        </span>
      )}

      {webServerRunning && !webEnabled && (
        <>
          <span className="tab-status-bar__state tab-status-bar__state--off">WEB OFF</span>
          <button
            className="tab-status-bar__btn"
            onClick={onToggleWeb}
            title="Enable web sharing for this session"
          >
            Enable Web
          </button>
        </>
      )}

      {webServerRunning && webEnabled && (
        <>
          <span className="tab-status-bar__state tab-status-bar__state--on">WEB ON</span>
          <a
            className="tab-status-bar__url"
            href={sessionURL}
            target="_blank"
            rel="noreferrer"
            title="Open session in browser"
          >
            {sessionURL}
          </a>
          <button
            className="tab-status-bar__btn"
            onClick={onToggleWeb}
            title="Disable web sharing for this session"
          >
            Disable Web
          </button>
          <button
            className="tab-status-bar__btn"
            onClick={onShowQR}
            title="Show QR code for this session"
          >
            QR
          </button>
        </>
      )}
    </div>
  )
}
