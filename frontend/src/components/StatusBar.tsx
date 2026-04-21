import React from 'react'

export interface StatusBarProps {
  sessionId: string
  webServerRunning: boolean
  webEnabled: boolean
  onToggleWeb: () => void
}

/**
 * Per-tab status bar showing web-serving state for the current session.
 * Always rendered at 32px height regardless of state — no layout reflow on toggle.
 *
 * Phase 87 cleanup: the raw session URL and QR button were removed. Sharing
 * is the Sessions tab's job (SessionSharePanel renders cap-bearing Read-Only
 * and Full Access links with Copy/Open/QR actions per link). The status bar
 * owns state display + toggle control only.
 */
export function StatusBar({
  webServerRunning,
  webEnabled,
  onToggleWeb,
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
          <span className="tab-status-bar__hint">
            Share links are on the Sessions tab
          </span>
          <button
            className="tab-status-bar__btn"
            onClick={onToggleWeb}
            title="Disable web sharing for this session"
          >
            Disable Web
          </button>
        </>
      )}
    </div>
  )
}
