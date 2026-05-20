import React, { useEffect, useState } from 'react'
import type { SessionInfo } from '../wailsjs/go/main/App'
import { IssueCapabilities, GetLocalNetworkPassword } from '../wailsjs/go/main/App'
import { ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'
import { SessionSharePanel } from './SessionSharePanel'

export interface DaemonManagerPanelProps {
  sessions: SessionInfo[]
  sessionStatuses: Record<string, string>
  webServerRunning: boolean
  webServerMode?: 'tailscale' | 'local' | null
  webEnabled: Record<string, boolean>
  onKill: (id: string) => void
  onToggleWeb: (id: string) => void
  /**
   * Phase 120-04 UI-01 — open (or focus) the FileBrowserTab for a session.
   * Provided by App.tsx's handleOpenFileBrowser callback.
   */
  onOpenFileBrowser: (sessionId: string, sessionName: string) => void
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
  webServerMode,
  webEnabled,
  onKill,
  onToggleWeb,
  onOpenFileBrowser,
}: DaemonManagerPanelProps): React.ReactElement {
  // Per-session capability URLs + join codes issued by the daemon on toggle-on.
  // Populated reactively as webEnabled transitions true; cleared when false.
  const [sessionShares, setSessionShares] = useState<Record<string, SessionShare>>({})

  // P-3: Show LAN Basic Auth password inline on this panel when running in
  // local-network-fallback mode. The same value is in Settings, but a fresh
  // user grabbing a share link from here shouldn't have to navigate away to
  // see what password the visitor needs to type.
  const [lanPassword, setLanPassword] = useState('')
  const [lanPasswordCopied, setLanPasswordCopied] = useState(false)
  useEffect(() => {
    if (webServerMode === 'local' && webServerRunning) {
      GetLocalNetworkPassword().then(setLanPassword).catch(() => setLanPassword(''))
    } else {
      setLanPassword('')
    }
  }, [webServerMode, webServerRunning])

  async function handleCopyLanPassword(): Promise<void> {
    if (!lanPassword) return
    try {
      await ClipboardSetText(lanPassword)
      setLanPasswordCopied(true)
      setTimeout(() => setLanPasswordCopied(false), 1500)
    } catch {
      // ClipboardSetText failure — no user-visible action; password remains visible
    }
  }

  // P-2: When the web server restarts (off → on transition), the daemon's
  // in-memory JoinCodeManager is wiped. Any join codes we cached in
  // sessionShares are now invalid; the displayed QR encodes a code the
  // server no longer recognises. Clear our cache on this transition so
  // the reconcile effect below refetches fresh capabilities.
  useEffect(() => {
    if (!webServerRunning) return
    setSessionShares({})
  }, [webServerRunning])

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
  }, [sessions, webEnabled, webServerRunning])

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
      {webServerMode === 'local' && webServerRunning && lanPassword && (
        <div
          className="daemon-panel__lan-creds"
          style={{
            padding: '8px 12px',
            margin: '0 0 8px 0',
            background: '#1e2030',
            border: '1px solid #3b4261',
            borderRadius: 4,
            fontSize: 12,
            color: '#a9b1d6',
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            flexWrap: 'wrap',
          }}
        >
          <span style={{ fontWeight: 600 }}>LAN Basic Auth:</span>
          <span>username — leave blank or anything</span>
          <span>·</span>
          <span>password — </span>
          <code
            style={{
              background: '#16161e',
              padding: '2px 6px',
              borderRadius: 3,
              fontFamily: 'inherit',
              userSelect: 'all',
            }}
          >
            {lanPassword}
          </code>
          <button
            type="button"
            onClick={() => void handleCopyLanPassword()}
            style={{
              border: '1px solid #3b4261',
              background: 'transparent',
              color: '#a9b1d6',
              padding: '2px 8px',
              borderRadius: 3,
              cursor: 'pointer',
              fontSize: 11,
            }}
          >
            {lanPasswordCopied ? 'Copied!' : 'Copy'}
          </button>
        </div>
      )}
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
                  className="daemon-panel__btn daemon-panel__btn--browse"
                  onClick={() => onOpenFileBrowser(s.id, s.name || s.cli)}
                  title="Open the file browser for this session"
                  data-testid={`daemon-panel-browse-${s.id}`}
                >
                  Browse files
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
