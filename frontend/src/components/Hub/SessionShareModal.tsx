import React, { useState, useEffect, useRef, useCallback } from 'react'
import { XMarkIcon } from '@heroicons/react/24/outline'
import {
  IssueCapabilities,
  ToggleWebServing,
  SetSessionBrowse,
  GetLocalNetworkPassword,
} from '../../wailsjs/go/main/App'
import { SessionSharePanel } from '../SessionSharePanel'
import { HomeDirWriteWarning } from '../HomeDirWriteWarning'
import { ShellWebShareBanner } from '../ShellWebShareBanner'

// Phase 150 SET-01 — must match App.tsx:89 and engine.go:isShellSession()
// Identifies shell sessions that require the web-share security warning.
const SHELL_CLIS = new Set(['shell', 'bash', 'zsh', 'pwsh', 'powershell'])

// ---- Types ----

interface ShareSession {
  id: string
  name: string
  cli: string          // Phase 150 SET-01: needed to check SHELL_CLIS gate
  webEnabled: boolean
  homeDir: boolean
  browseEnabled: boolean
}

interface CachedShare {
  readURL: string
  writeURL: string
  readCode: string
  writeCode: string
}

export interface SessionShareModalProps {
  session: ShareSession
  webServerMode?: 'tailscale' | 'local' | null
  webServerRunning?: boolean
  onClose: () => void
  // Phase 150 SET-01 — shell warning cross-surface parity (D-09/D-10).
  // Shared authority from App.tsx — do NOT fork into local state.
  shellWebShareWarned?: boolean
  shellWebShareWarningEnabled?: boolean
  onShellWebShareConfirm?: () => Promise<void>
  onShellWebShareCancel?: () => void
}

/**
 * SessionShareModal — per-card Share modal for the Hub (Phase 137 / SHARE-01/02/04/05/06).
 *
 * Two-toggle design (D-10):
 *   1. "Share the session" — calls ToggleWebServing; on ON, issues caps + reveals SessionSharePanel
 *   2. "Enable remote file browsing" — disabled when sharing OFF; calls SetSessionBrowse then
 *      re-issues caps so the RO/RW URLs reflect the new browse perm matrix (D-03/D-04).
 *
 * Lifecycle (SHARE-05):
 *   - Seeds shareEnabled from session.webEnabled and browseEnabled from session.browseEnabled on open.
 *   - If webEnabled=true on open and no cached share, IssueCapabilities is called once.
 *   - When webServerRunning transitions false→true (restart), cached URLs are cleared so stale caps
 *     are not displayed. The seeding effect then re-issues on next render.
 *
 * Security: LockClosedIcon on remote peer cards (D-13, implemented in SessionCard — this modal is
 * only reachable for local cards). HomeDir warning (D-09) rendered before browse toggle.
 *
 * Phase machine: entering → open → exiting (fade in/out). prefers-reduced-motion: skip animation.
 * Focus return: returns focus to the Share button that opened the modal on unmount.
 * No inert background trap (leaner than HubModal — no terminal inside).
 */
export function SessionShareModal({
  session,
  webServerMode,
  webServerRunning,
  onClose,
  shellWebShareWarned,
  shellWebShareWarningEnabled,
  onShellWebShareConfirm,
  onShellWebShareCancel,
}: SessionShareModalProps): React.ReactElement {
  // ---- Animation phase machine (entering → open → exiting) ----
  // Same pattern as HubModal but without grow animation (no sourceRect / transformOrigin).
  // prefers-reduced-motion: skip animated phases.
  const prefersReducedMotion =
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches

  const [phase, setPhase] = useState<'entering' | 'open' | 'exiting'>(
    prefersReducedMotion ? 'open' : 'entering',
  )

  const handleClose = useCallback(() => {
    if (prefersReducedMotion) {
      onClose()
      return
    }
    setPhase('exiting')
  }, [prefersReducedMotion, onClose])

  // ---- Focus return on unmount (MODAL-02) ----
  const openerFocusRef = useRef<HTMLElement | null>(null)
  useEffect(() => {
    openerFocusRef.current = document.activeElement as HTMLElement
    return () => {
      openerFocusRef.current?.focus()
    }
  }, [])

  // ---- Escape key (dialog-scoped via onKeyDown, same as HubModal pattern) ----
  // The Escape handler is on the dialog role="dialog" element (see JSX below).

  // ---- Share state ----
  // Seeds from server truth (session.webEnabled); overridden by user toggle.
  const [shareEnabled, setShareEnabled] = useState(session.webEnabled)
  // Phase 150 SET-01 (D-09): true when the shell-warning banner is shown
  // and awaiting user confirm/cancel. Pending = ToggleWebServing intercepted.
  const [pendingShellShare, setPendingShellShare] = useState(false)
  // Seeds from session.browseEnabled (Plan 02 SessionInfo field).
  const [browseEnabled, setBrowseEnabled] = useState(session.browseEnabled)
  // Cached capability URLs + codes (cleared on server restart, re-issued on seeding).
  const [cachedShare, setCachedShare] = useState<CachedShare | null>(null)
  // Whether the home-dir warning has been dismissed by the user this session.
  const [homeDirDismissed, setHomeDirDismissed] = useState(false)
  // Ref mirror of cachedShare for the seeding effect to read without depending on it
  // (avoids adding cachedShare to seeding effect deps, which interacts poorly with flushSync).
  const cachedShareRef = useRef<CachedShare | null>(null)
  cachedShareRef.current = cachedShare

  // ---- LAN password (SHARE-04) ----
  const [lanPassword, setLanPassword] = useState('')
  useEffect(() => {
    if (webServerMode === 'local' && webServerRunning) {
      GetLocalNetworkPassword().then(setLanPassword).catch(() => setLanPassword(''))
    } else {
      setLanPassword('')
    }
  }, [webServerMode, webServerRunning])

  // ---- Server restart clears stale cache + re-seeds (SHARE-05) ----
  // P-2 (mirrors DaemonManagerPanel): when webServerRunning transitions false→true,
  // the daemon's JoinCodeManager is wiped. Clear cached caps and directly re-issue
  // caps in the same effect (no second effect / extra render cycle needed).
  const prevWebServerRunning = useRef<boolean | undefined>(undefined)
  // Ref mirrors of shareEnabled and session.id for use in effects without adding
  // them as restart-effect deps (would cause unintended re-runs on non-restart changes).
  const shareEnabledRef = useRef(shareEnabled)
  shareEnabledRef.current = shareEnabled
  const sessionIdRef = useRef(session.id)
  sessionIdRef.current = session.id
  useEffect(() => {
    const isRestart = prevWebServerRunning.current === false && webServerRunning === true
    prevWebServerRunning.current = webServerRunning
    if (!isRestart) return
    setCachedShare(null)
    // Directly re-issue caps if sharing is on (same effect — avoids an extra render cycle
    // that React 18's scheduler may defer past the test's setTimeout(0) fence).
    if (!shareEnabledRef.current) return
    let cancelled = false
    void (async () => {
      try {
        const resp = await IssueCapabilities(sessionIdRef.current)
        if (cancelled) return
        setCachedShare({
          readURL: resp.readUrl,
          writeURL: resp.writeUrl,
          readCode: resp.readCode,
          writeCode: resp.writeCode,
        })
      } catch {
        // IssueCapabilities failed — cachedShare remains null; seeding effect will retry.
      }
    })()
    return () => { cancelled = true }
  }, [webServerRunning])

  // ---- Server-truth seeding on open (SHARE-05) ----
  // Runs when shareEnabled or session.id changes.
  // Uses cachedShareRef (not state) to avoid adding cachedShare as a dep,
  // which would create a problematic dep that interacts poorly with flushSync.
  // Restart re-seeding is handled directly in the restart-clear effect above.
  useEffect(() => {
    if (!shareEnabled) return
    if (cachedShareRef.current !== null) return
    let cancelled = false
    void (async () => {
      try {
        const resp = await IssueCapabilities(session.id)
        if (cancelled) return
        setCachedShare({
          readURL: resp.readUrl,
          writeURL: resp.writeUrl,
          readCode: resp.readCode,
          writeCode: resp.writeCode,
        })
      } catch {
        // IssueCapabilities failed — leave cachedShare null; user can toggle off/on to retry.
      }
    })()
    return () => { cancelled = true }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [shareEnabled, session.id])

  // ---- Toggle handlers ----

  // "Share the session" toggle handler.
  async function handleShareToggle(): Promise<void> {
    const next = !shareEnabled
    // Phase 150 SET-01 (D-09): intercept ON-toggles for shell sessions when
    // the warning is enabled and the user hasn't yet acknowledged it on this
    // machine. AI-CLI toggles, shell OFF-toggles, and shell ON-toggles with
    // warned=true fall through unchanged. Single warned authority from App.tsx
    // (pitfall 4 — do NOT fork into local state).
    if (next && SHELL_CLIS.has(session.cli) && shellWebShareWarningEnabled && !shellWebShareWarned) {
      setPendingShellShare(true)
      return
    }
    try {
      await ToggleWebServing(session.id, next)
      setShareEnabled(next)
      if (!next) {
        // Sharing turned OFF — clear cached share to avoid stale display.
        setCachedShare(null)
      }
      // If turning ON, the seeding effect above will issue caps on next render.
    } catch {
      // ToggleWebServing failed — revert (shareEnabled unchanged).
    }
  }

  // "Enable remote file browsing" toggle handler.
  // Disabled when sharing is OFF (see JSX below).
  // Pitfall 1: MUST re-issue caps after SetSessionBrowse when sharing ON,
  // or the cached RO URL still lacks files.read (stale cap).
  async function handleBrowseToggle(): Promise<void> {
    if (!shareEnabled) return // guard — should not be clickable, but defense-in-depth
    const next = !browseEnabled
    try {
      await SetSessionBrowse(session.id, next)
      setBrowseEnabled(next)
      // Re-issue caps so URLs reflect the new browse perm matrix (D-03/D-04).
      if (shareEnabled) {
        try {
          const resp = await IssueCapabilities(session.id)
          setCachedShare({
            readURL: resp.readUrl,
            writeURL: resp.writeUrl,
            readCode: resp.readCode,
            writeCode: resp.writeCode,
          })
        } catch {
          // IssueCapabilities failed — clear stale cache so seeding effect refetches.
          setCachedShare(null)
        }
      }
    } catch {
      // SetSessionBrowse failed — revert (browseEnabled unchanged).
    }
  }

  // ---- Render ----
  return (
    <div
      className={`hub-modal-overlay hub-modal-overlay--${phase}`}
      onClick={handleClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={`Share ${session.name}`}
        className={`hub-share-modal hub-share-modal--${phase}`}
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.preventDefault()
            e.stopPropagation()
            handleClose()
          }
        }}
        onAnimationEnd={() => {
          if (phase === 'entering') setPhase('open')
          if (phase === 'exiting') onClose()
        }}
      >
        {/* ---- Header ---- */}
        <div className="hub-share-modal__header">
          <span className="hub-share-modal__title">Share — {session.name}</span>
          <button
            type="button"
            className="hub-modal__close"
            aria-label="Close share modal"
            onClick={handleClose}
          >
            <XMarkIcon aria-hidden="true" />
          </button>
        </div>

        {/* ---- Body ---- */}
        <div className="hub-share-modal__body">
          {/* Phase 150 SET-01 (D-09): shell web-share warning banner — shown when
              the user toggles share ON for a shell session and warningEnabled + !warned.
              Reuses ShellWebShareBanner (D-10 cross-surface parity).
              onConfirm calls the App.tsx race-mitigation confirm handler (sets warned
              synchronously) then enables share; onCancel clears the pending state. */}
          {pendingShellShare && (
            <ShellWebShareBanner
              sessionName={session.name}
              onConfirm={async () => {
                setPendingShellShare(false)
                await onShellWebShareConfirm?.()
                setShareEnabled(true)
                // seeding effect will issue caps on next render
              }}
              onCancel={() => {
                setPendingShellShare(false)
                onShellWebShareCancel?.()
              }}
            />
          )}
          {/* SHARE-01: "Share the session" toggle */}
          <label
            className={`settings-panel__toggle-row${shareEnabled ? ' settings-panel__toggle-row--checked' : ''}`}
            style={{ cursor: 'pointer' }}
          >
            <input
              type="checkbox"
              className="settings-panel__toggle-input"
              role="switch"
              aria-checked={shareEnabled}
              aria-label="Share the session"
              checked={shareEnabled}
              onChange={() => void handleShareToggle()}
            />
            <span className="settings-panel__toggle-track">
              <span className="settings-panel__toggle-thumb" />
            </span>
            <span className="settings-panel__toggle-label">Share the session</span>
          </label>

          {/* LAN password (SHARE-04): shown in local mode */}
          {webServerMode === 'local' && webServerRunning && lanPassword && (
            <div className="hub-share-modal__lan-creds" style={{ margin: '8px 0', fontSize: 12 }}>
              <span style={{ fontWeight: 600 }}>LAN password:</span>{' '}
              <code>
                {lanPassword}
              </code>
            </div>
          )}

          {/* HomeDirWriteWarning (D-09): rendered when session.homeDir is true */}
          {session.homeDir && !homeDirDismissed && (
            <HomeDirWriteWarning
              onDismiss={() => setHomeDirDismissed(true)}
            />
          )}

          {/* SHARE-02: "Enable remote file browsing" toggle.
              Disabled/no-op when sharing is OFF (Pitfall 1 guard). */}
          <div
            aria-disabled={!shareEnabled}
            style={!shareEnabled ? { opacity: 0.6, pointerEvents: 'none' } : undefined}
          >
            <label
              className={`settings-panel__toggle-row${browseEnabled ? ' settings-panel__toggle-row--checked' : ''}`}
              style={{ cursor: shareEnabled ? 'pointer' : 'not-allowed' }}
            >
              <input
                type="checkbox"
                className="settings-panel__toggle-input"
                role="switch"
                aria-checked={browseEnabled}
                aria-label="Enable remote file browsing"
                checked={browseEnabled}
                disabled={!shareEnabled}
                onChange={() => void handleBrowseToggle()}
              />
              <span className="settings-panel__toggle-track">
                <span className="settings-panel__toggle-thumb" />
              </span>
              <span className="settings-panel__toggle-label">Enable remote file browsing</span>
            </label>
          </div>

          {/* Share panel: shown only when sharing is ON and caps are available.
              D-11: simplified SessionSharePanel (CAP-05 two-gate stripped). */}
          {shareEnabled && cachedShare && (
            <SessionSharePanel
              sessionId={session.id}
              readURL={cachedShare.readURL}
              writeURL={cachedShare.writeURL}
              readCode={cachedShare.readCode}
              writeCode={cachedShare.writeCode}
              browseEnabled={browseEnabled}
            />
          )}
        </div>
      </div>
    </div>
  )
}
