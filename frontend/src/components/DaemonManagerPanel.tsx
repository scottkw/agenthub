import React from 'react'
import type { SessionInfo } from '../wailsjs/go/main/App'

export interface DaemonManagerPanelProps {
  sessions: SessionInfo[]
  sessionStatuses: Record<string, string>
  webServerRunning: boolean
  webEnabled: Record<string, boolean>
  onKill: (id: string) => void
  onToggleWeb: (id: string) => void
}

export function DaemonManagerPanel({
  sessions,
  sessionStatuses,
  webServerRunning,
  webEnabled,
  onKill,
  onToggleWeb,
}: DaemonManagerPanelProps): React.ReactElement {
  if (sessions.length === 0) {
    return (
      <div className="daemon-panel">
        <div className="daemon-panel__empty">No active sessions</div>
      </div>
    )
  }

  return (
    <div className="daemon-panel">
      <div className="daemon-panel__header">
        <h2 className="daemon-panel__title">Sessions</h2>
        <span className="daemon-panel__count">{sessions.length} active</span>
      </div>
      <div className="daemon-panel__list">
        {sessions.map((s) => {
          const status = sessionStatuses[s.id] || s.state || 'running'
          const isWebOn = !!webEnabled[s.id]
          return (
            <div key={s.id} className="daemon-panel__session-row">
              <span
                className={`daemon-panel__status daemon-panel__status--${status}`}
                title={status}
              />
              <span className="daemon-panel__name">{s.name}</span>
              <span className="daemon-panel__cli">{s.cli}</span>
              <div className="daemon-panel__actions">
                <button
                  className="daemon-panel__btn daemon-panel__btn--web"
                  onClick={() => onToggleWeb(s.id)}
                  disabled={!webServerRunning}
                  title={
                    !webServerRunning
                      ? 'Web server not running'
                      : isWebOn
                        ? 'Disable web sharing'
                        : 'Enable web sharing'
                  }
                >
                  {isWebOn ? 'Web Off' : 'Web On'}
                </button>
                <button
                  className="daemon-panel__btn daemon-panel__btn--kill"
                  onClick={() => onKill(s.id)}
                  title="Kill session"
                >
                  Kill
                </button>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
