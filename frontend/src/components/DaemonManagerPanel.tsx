import React, { useEffect, useState } from 'react'
import type { SessionInfo } from '../wailsjs/go/main/App'
import { IssueCapabilities } from '../wailsjs/go/main/App'
import { SessionSharePanel } from './SessionSharePanel'

export interface DaemonManagerPanelProps {
  sessions: SessionInfo[]
  sessionStatuses: Record<string, string>
  webServerRunning: boolean
  webEnabled: Record<string, boolean>
  onKill: (id: string) => void
  onToggleWeb: (id: string) => void
}

interface SessionShare {
  readURL: string
  writeURL: string
  readCode: string
  writeCode: string
}

export function DaemonManagerPanel({
  sessions,
  sessionStatuses,
  webServerRunning,
  webEnabled,
  onKill,
  onToggleWeb,
}: DaemonManagerPanelProps): React.ReactElement {
  // Per-session capability URLs + join codes issued by the daemon on toggle-on.
  // Populated reactively as webEnabled transitions true; cleared when false.
  const [sessionShares, setSessionShares] = useState<Record<string, SessionShare>>({})

  useEffect(() => {
    let cancelled = false

    async function reconcile(): Promise<void> {
      // For every session that is web-enabled but missing a share entry, fetch
      // capabilities from the daemon. For every share entry whose session is
      // no longer web-enabled (or no longer exists), drop it — T-87-07 (stale
      // URLs after toggle-off) is mitigated by this cleanup.
      const validIds = new Set(sessions.map((s) => s.id))

      // Drop stale shares (toggle-off or session removed).
      setSessionShares((prev) => {
        let changed = false
        const next: Record<string, SessionShare> = {}
        for (const [id, share] of Object.entries(prev)) {
          if (webEnabled[id] && validIds.has(id)) {
            next[id] = share
          } else {
            changed = true
          }
        }
        return changed ? next : prev
      })

      // Fetch shares for newly-enabled sessions.
      for (const s of sessions) {
        if (!webEnabled[s.id]) continue
        if (sessionShares[s.id]) continue
        try {
          const resp = await IssueCapabilities(s.id)
          if (cancelled) return
          setSessionShares((prev) => {
            if (prev[s.id]) return prev
            return {
              ...prev,
              [s.id]: {
                readURL: resp.readUrl,
                writeURL: resp.writeUrl,
                readCode: resp.readCode,
                writeCode: resp.writeCode,
              },
            }
          })
        } catch (err) {
          // Capability issuance failed — log but don't crash the panel. The
          // user can toggle off/on to retry; the web-toggle itself already
          // succeeded (daemon enabled the session).
          console.warn('[DaemonManagerPanel] IssueCapabilities failed for', s.id, err)
        }
      }
    }

    void reconcile()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessions, webEnabled])

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
          const share = sessionShares[s.id]
          return (
            <div key={s.id} className="daemon-panel__session-row">
              <span
                className={`daemon-panel__status daemon-panel__status--${status}`}
                title={status}
              />
              <span className="daemon-panel__name">{s.name}</span>
              <span className="daemon-panel__cli">{s.cli}</span>
              <span className="daemon-panel__hostname" title={s.hostname || ''}>
                {s.hostname || '\u2014'}
              </span>
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
              {isWebOn && share && (
                <SessionSharePanel
                  sessionId={s.id}
                  readURL={share.readURL}
                  writeURL={share.writeURL}
                  readCode={share.readCode}
                  writeCode={share.writeCode}
                />
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
