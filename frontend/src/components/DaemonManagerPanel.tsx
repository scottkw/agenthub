import React, { useEffect, useState } from 'react'
import type { SessionInfo } from '../wailsjs/go/main/App'
import { IssueCapabilities, SetSessionFilesWrite, GetLocalNetworkPassword } from '../wailsjs/go/main/App'
import { ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'
import { SessionSharePanel } from './SessionSharePanel'
import { HomeDirWriteWarning } from './HomeDirWriteWarning'

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
  /** Phase 124 / CAP-06: true when session cwd equals EvalSymlinks($HOME) */
  homeDir: boolean
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

  // Phase 124 / CAP-04: per-session write enabled state (owner toggle, Surface 1).
  // Default OFF for every session (T-124-14; opt-in for all per CAP-08).
  const [sessionWrites, setSessionWrites] = useState<Record<string, boolean>>({})
  // Phase 124 / CAP-04: tracks in-flight toggle saves to disable during save.
  const [writeSaving, setWriteSaving] = useState<Record<string, boolean>>({})
  // Phase 124 / CAP-04: error messages per session for failed write toggle.
  const [writeError, setWriteError] = useState<Record<string, string>>({})
  // Phase 124 / CAP-06: per-session home-dir banner dismissal. Re-shows on re-enable.
  // Key = sessionId, value = the filesWrite state at which it was dismissed.
  const [homeDirDismissed, setHomeDirDismissed] = useState<Record<string, boolean>>({})

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

  // Phase 124 / CAP-04: Toggle the per-session owner write capability.
  // Calls SetSessionFilesWrite, then re-issues capabilities to refresh URLs.
  // Dismiss banner on re-enable so it re-shows when writes are turned back on.
  async function handleToggleFilesWrite(sessionId: string, enabled: boolean): Promise<void> {
    setWriteSaving((prev) => ({ ...prev, [sessionId]: true }))
    setWriteError((prev) => ({ ...prev, [sessionId]: '' }))
    try {
      await SetSessionFilesWrite(sessionId, enabled)
      setSessionWrites((prev) => ({ ...prev, [sessionId]: enabled }))
      // Re-dismiss banner state so it shows again when writes are re-enabled.
      if (enabled) {
        setHomeDirDismissed((prev) => ({ ...prev, [sessionId]: false }))
      }
      // Re-issue capabilities so URLs reflect the new write state.
      if (webEnabled[sessionId]) {
        try {
          const resp = await IssueCapabilities(sessionId)
          setSessionShares((prev) => ({
            ...prev,
            [sessionId]: {
              readURL: resp.readUrl,
              writeURL: resp.writeUrl,
              readCode: resp.readCode,
              writeCode: resp.writeCode,
              homeDir: resp.homeDir ?? false,
            },
          }))
        } catch (capErr) {
          // IN-01: IssueCapabilities failed after SetSessionFilesWrite succeeded.
          // The cached share entry now reflects the pre-toggle token (which may
          // incorrectly carry or lack files.write). Clear it so the reconcile
          // effect below refetches fresh capabilities on the next render cycle,
          // preventing stale URLs from being displayed.
          setSessionShares((prev) => {
            const next = { ...prev }
            delete next[sessionId]
            return next
          })
          setWriteError((prev) => ({
            ...prev,
            [sessionId]: 'Links may be stale — try toggling off and on to refresh.',
          }))
          console.warn('[DaemonManagerPanel] IssueCapabilities after write-toggle failed', capErr)
        }
      }
    } catch {
      setWriteError((prev) => ({
        ...prev,
        [sessionId]: "Couldn't save the write setting. Try again.",
      }))
    } finally {
      setWriteSaving((prev) => ({ ...prev, [sessionId]: false }))
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
                homeDir: resp.homeDir ?? false,
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
              {/* Phase 124 / CAP-04: owner "Enable file writes" toggle (Surface 1).
                  Renders for every session (not just web-on) so the owner can
                  pre-enable writes before sharing. Verbatim label from 124-UI-SPEC. */}
              <div className="daemon-panel__files-write">
                <label
                  className={`settings-panel__toggle-row${sessionWrites[s.id] ? ' settings-panel__toggle-row--checked' : ''}${writeSaving[s.id] ? ' settings-panel__toggle-row--saving' : ''}`}
                  style={writeSaving[s.id] ? { pointerEvents: 'none', opacity: 0.6 } : undefined}
                >
                  <input
                    type="checkbox"
                    className="settings-panel__toggle-input"
                    role="switch"
                    aria-checked={!!sessionWrites[s.id]}
                    aria-label="Enable file writes"
                    checked={!!sessionWrites[s.id]}
                    disabled={!!writeSaving[s.id]}
                    onChange={() => void handleToggleFilesWrite(s.id, !sessionWrites[s.id])}
                  />
                  <span className="settings-panel__toggle-track">
                    <span className="settings-panel__toggle-thumb" />
                  </span>
                  <span className="settings-panel__toggle-label">Enable file writes</span>
                </label>
                <p className="settings-panel__toggle-helptext" style={{ margin: '0 0 4px 46px', fontSize: 12, color: '#9aa5ce', lineHeight: 1.4 }}>
                  Lets this session create, edit, delete, rename, and upload files in its working directory. Off by default.
                </p>
                {writeError[s.id] && (
                  <p className="settings-panel__error" style={{ margin: '0 0 4px 46px', fontSize: 12, color: '#f7768e' }}>
                    {writeError[s.id]}
                  </p>
                )}
                {/* Phase 124 / CAP-06 / WR-04: home-dir write warning banner (Surface 3).
                    Source homeDir from SessionInfo.homeDir (the ListSessions-derived
                    field, same source of truth the TUI uses in internal/tui/files.go)
                    rather than share?.homeDir (per-capability response, can be stale
                    if IssueCapabilities fails or has not yet run). Both surfaces now
                    collapse to the single server-side source of truth documented in
                    engine.go:461-467 for cross-surface parity. */}
                {sessionWrites[s.id] && s.homeDir && !homeDirDismissed[s.id] && (
                  <HomeDirWriteWarning
                    onDismiss={() => setHomeDirDismissed((prev) => ({ ...prev, [s.id]: true }))}
                  />
                )}
              </div>
              {isWebOn && share && (
                <SessionSharePanel
                  sessionId={s.id}
                  readURL={share.readURL}
                  writeURL={share.writeURL}
                  readCode={share.readCode}
                  writeCode={share.writeCode}
                  ownerWriteEnabled={!!sessionWrites[s.id]}
                />
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
