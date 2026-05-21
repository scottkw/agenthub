import React from 'react'

export interface RemoteSession {
  id: string
  name: string
  cliType: string
  status: string
  url: string
}

export interface RemotePeerSessions {
  hostname: string
  sessions: RemoteSession[]
}

export interface RemoteSessionsPanelProps {
  peers: RemotePeerSessions[]
  loading: boolean
  onOpen: (url: string) => void
  /**
   * Phase 122-03 — opens the file-browser tab for a remote session via the
   * paste-join-code flow (App.tsx handles the cap-cache + modal trigger).
   */
  onBrowseFiles: (sessionId: string, sessionName: string) => void
}

export function RemoteSessionsPanel({
  peers,
  loading,
  onOpen,
  onBrowseFiles,
}: RemoteSessionsPanelProps): React.ReactElement {
  if (loading && peers.length === 0) {
    return (
      <div className="remote-panel">
        <div className="remote-panel__loading">
          <div className="remote-panel__spinner" />
          Probing peers...
        </div>
      </div>
    )
  }
  if (!loading && peers.length === 0) {
    return (
      <div className="remote-panel">
        <div className="remote-panel__empty">
          <div className="remote-panel__empty-title">No remote peers found</div>
          <div className="remote-panel__empty-body">No tailnet peers are running AgentHub.</div>
        </div>
      </div>
    )
  }
  return (
    <div className="remote-panel">
      {peers.map((peer) => (
        <div key={peer.hostname} className="remote-panel__peer">
          <div className="remote-panel__peer-header">{peer.hostname}</div>
          <div className="remote-panel__peer-meta">Shows web-enabled sessions only</div>
          <div className="remote-panel__session-list">
            {peer.sessions.map((s) => (
              <div key={s.id} className="remote-panel__session-row">
                <span
                  className={`remote-panel__status remote-panel__status--${s.status}`}
                  title={s.status}
                />
                <span className="remote-panel__name">{s.name}</span>
                <span className="remote-panel__cli">{s.cliType}</span>
                <div className="remote-panel__actions">
                  <button
                    className="remote-panel__btn remote-panel__btn--open"
                    onClick={() => onOpen(s.url)}
                    title="Open in browser"
                    aria-label={`Open ${s.name} in browser`}
                  >
                    Open Session
                  </button>
                  <button
                    className="remote-panel__btn remote-panel__btn--browse"
                    onClick={() => onBrowseFiles(s.id, s.name)}
                    title="Browse files"
                    aria-label={`Browse files on ${s.name}`}
                  >
                    Browse files
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
