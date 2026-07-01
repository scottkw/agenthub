import React, { useState, useRef, useEffect, useCallback } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import type { AdaptedRemoteSessionInfo } from '../../lib/remoteAdapter'
import { GetSessionStyledTailLines } from '../../wailsjs/go/main/App'
import type { ITheme } from '@xterm/xterm'
import { daemon } from '../../wailsjs/go/models'
import { ExclamationCircleIcon } from '@heroicons/react/24/outline'
import { HubFilterBar } from './HubFilterBar'
import type { HubFilter } from './HubFilterBar'
import { SessionCardGrid } from './SessionCardGrid'
import { HubEmptyState } from './HubEmptyState'
import { computeCounts, computeGlobalCounts } from '../../lib/hubGroupCounts'
import type { GroupCounts } from '../../lib/hubGroupCounts'
import { HubModal } from './HubModal'
import { SessionShareModal } from './SessionShareModal'
import { useChatUnreadListeners } from './useChatUnreadListeners'
// WR-01: deriveHubStatus extracted to shared util (was triplicated across SessionCard/HubFilterBar/HubPanel)
// ATTN-01/04: isAttentionStatus is the single canonical attention predicate; used for live set + debounced sort key
import { deriveHubStatus, isAttentionStatus } from '../../lib/hubStatus'
import {
  memberKey,
  type HubGroupDef,
} from '../../lib/hubGroups'

type PluginSettings = daemon.PluginSettings

/**
 * filterSessions — apply status filter + case-insensitive substring search.
 *
 * @param sessions  - full session list (from prop)
 * @param filter    - active HubFilter key ('all' means no filter)
 * @param search    - trimmed search string ('' means no search)
 * @returns filtered subset, preserving input order
 */
export function filterSessions(
  sessions: SessionInfo[],
  filter: HubFilter,
  search: string,
): SessionInfo[] {
  const needle = search.toLowerCase()
  return sessions.filter((s) => {
    // Status filter
    if (filter !== 'all') {
      const status = deriveHubStatus(s)
      if (status !== filter) return false
    }

    // Search filter — case-insensitive substring on name, cli, and hostname
    if (needle) {
      const inName = s.name.toLowerCase().includes(needle)
      const inCli = s.cli.toLowerCase().includes(needle)
      const inHost = s.hostname.toLowerCase().includes(needle)
      if (!inName && !inCli && !inHost) return false
    }

    return true
  })
}

// ---- usePreviewPoller ----

// CARD-07: single shared 3s poller drives all card previews — NOT per-card intervals
// T-132-12: stable `sessionIdKey` dep prevents polling storm on array reference churn
// Phase 139 / CARD-05: tails are now StyledSpan[][] instead of string[] (VT cell grid with color+bold)
function usePreviewPoller(
  sessions: SessionInfo[],
  isActive: boolean,
): Map<string, daemon.StyledSpan[][]> {
  const [tails, setTails] = useState<Map<string, daemon.StyledSpan[][]>>(new Map())

  // Caller passes LOCAL sessions only (see call site). Local sessions carry the
  // machine's own hostname (os.Hostname()), so a hostname check is NOT a valid
  // local/remote discriminator — provenance (which prop) is. Dep key = all ids passed.
  const sessionIdKey = sessions.map((s) => s.id).join(',')

  useEffect(() => {
    if (!isActive || sessions.length === 0) return
    let cancelled = false

    async function poll() {
      // All passed sessions are local (caller filters by provenance). Remote sessions
      // have no tail API and are excluded by the caller, not by hostname.
      const localSessions = sessions
      if (localSessions.length === 0) return
      const results = await Promise.all(
        localSessions.map((s) =>
          // Fetch 12 tail lines (clamped [1..20]) so the card mini-preview shows real
          // session content above the fixed footer/input region of TUI agents like Claude,
          // not just the bottom 4 footer lines.
          GetSessionStyledTailLines(s.id, 12).catch(() => [] as daemon.StyledSpan[][])
        )
      )
      if (!cancelled) {
        // CR-03: merge results into the previous map instead of replacing it wholesale.
        // When a session is stopped/killed, GetSessionStyledTailLines returns [] (the hub has
        // been removed). Replacing the whole map would flip the preview back to "No output
        // yet". Instead, only overwrite an entry when we received real lines OR when there
        // is no prior value for that session (first fetch). This preserves the last-seen
        // snapshot for sessions that return empty due to error or cleanup.
        setTails((prev) => {
          const next = new Map(prev)
          localSessions.forEach((s, i) => {
            const lines = results[i]
            if (lines.length > 0 || !prev.has(s.id)) {
              next.set(s.id, lines)
            }
          })
          return next
        })
      }
    }

    void poll()
    const interval = setInterval(() => void poll(), 3000)
    return () => { cancelled = true; clearInterval(interval) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionIdKey, isActive])

  return tails
}

/* ATTN-04: debounce hook — controls reorder ORDER only, never card content */
/* Uses useRef + setTimeout — does NOT add a second periodic timer; the single interval is in usePreviewPoller */
function useDebouncedValue<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = React.useState<T>(value)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (timerRef.current !== null) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => setDebouncedValue(value), delay)
    return () => { if (timerRef.current !== null) clearTimeout(timerRef.current) }
  }, [value, delay])

  return debouncedValue
}

// ---- Props ----

// WR-04: DEFAULT_FONT_SIZE used in the modal render to avoid magic number 14.
// Kept module-local (not re-exported from App) to preserve App.tsx's constant.
const DEFAULT_FONT_SIZE = 14

export interface HubPanelProps {
  /** Session list polled by App.tsx (Plan 05 wires the polling). */
  sessions: SessionInfo[]
  /** True when the last ListSessions() call threw — triggers error state. */
  error: boolean
  /** Opens the NewSessionModal (handled by App.tsx). */
  onNewSession: () => void
  /** Fired when the user commits an inline rename. */
  onRename: (id: string, name: string) => void
  /** Opens (or focuses) the session's terminal tab. Phase 131 UAT follow-up. */
  onOpenSession?: (sessionId: string, name: string, cli: string) => void
  /** Phase 132 / GRID-07 — remote sessions from App.tsx (already adapted to SessionInfo[]) */
  remoteSessions?: SessionInfo[]
  /** Phase 132 / CARD-07 — true when Hub tab is active (gates usePreviewPoller interval) */
  isActive?: boolean
  /** Phase 134 — relay port for mounting TerminalPanel inside the interactive modal */
  relayPort?: number
  /** Phase 134 — xterm.js theme passed to HubInteractiveModal (WR-03: required — App always supplies a non-null theme). */
  terminalTheme: ITheme
  /** Phase 134 — plugin config passed to HubInteractiveModal */
  pluginConfig?: PluginSettings | null
  /** Phase 134 — MODAL-06: cap set for remote sessions; checked before opening modal */
  remoteCapsCached?: Set<string>
  /** Phase 134 — MODAL-06: triggers RemoteJoinCodeModal flow for uncapped remote sessions */
  onRequestRemoteCap?: (session: { id: string; name: string; hostname: string }) => void
  /** Phase 134 — registers a callback that HubPanel will call when a cap is acquired for a pending session */
  onRegisterCapAcquired?: (fn: (sessionId: string) => void) => void
  /** Phase 134 — registers a callback that HubPanel will call when a cap request is cancelled (WR-01) */
  onRegisterCapCancelled?: (fn: () => void) => void
  /** Phase 134 — WR-04: per-session font sizes (keyed by session id). Falls back to DEFAULT_FONT_SIZE. */
  fontSizes?: Record<string, number>
  /** Phase 134 — WR-04: font size change callback; receives (delta) for the active modal session */
  onFontSizeChange?: (sessionId: string, delta: number) => void
  /** Phase 137 / SHARE-04: web server mode ('local' | 'tailscale' | null) for LAN-password display */
  webServerMode?: 'tailscale' | 'local' | null
  /** Phase 137 / SHARE-05: true when the web server is running; triggers restart-clear in SessionShareModal */
  webServerRunning?: boolean
  /** Phase 138 — Kill handler threaded to card overflow menu */
  onKill?: (sessionId: string) => void
  /** Phase 138 — Open remote session in system browser (Phase 146: receives session object for cap exchange) */
  onOpenInBrowser?: (session: AdaptedRemoteSessionInfo) => void
  /** Phase 138 — Browse remote session files (join-code flow) */
  onBrowseFiles?: (sessionId: string, sessionName: string) => void
  /** Phase 138 — remotePeers raw data for unreachable-peer hints */
  remotePeers?: import('../../lib/remoteSession').RemotePeerSessions[]
  /** POL-05 — active group filter driven by Sidebar selection (state lifted to App.tsx) */
  activeGroupId?: string | null
  /** POL-05 — group definitions for per-card Move-to-group menu (state lifted to App.tsx) */
  groupDefs?: HubGroupDef[]
  /** POL-05 — drag-drop / per-card assign callback (routed through App.tsx) */
  onDropOnGroup?: (groupId: string, mKey: string) => void
  /** POL-05 — counts emitted upward so Sidebar can display live running/total per group */
  onGroupCountsChange?: (
    counts: Record<string, GroupCounts>,
    global: GroupCounts
  ) => void
  // Phase 150 SET-01 — shell warning cross-surface parity (D-09/D-10).
  // Threaded from App.tsx (single warned authority) into SessionShareModal.
  /** True when the user has acknowledged the shell web-share warning on this machine */
  shellWebShareWarned?: boolean
  /** True (default) when the shell web-share warning is enabled */
  shellWebShareWarningEnabled?: boolean
  /** Confirm callback from App.tsx race-mitigation handler */
  onShellWebShareConfirm?: () => Promise<void>
  /** Cancel callback from App.tsx */
  onShellWebShareCancel?: () => void
  // Phase 166 FUI-06 — opens the in-app Help tab at the Sharing Guide section.
  // Threaded from App.tsx into the SessionShareModal risk-panel cross-link.
  onOpenHelp?: () => void
}

// POL-05: SIDEBAR_COLLAPSED_KEY removed — hub-group-sidebar-collapsed localStorage key no longer used.

// ---- Component ----

/**
 * HubPanel — top-level Hub surface.
 *
 * Owns filter + search state; applies filtering; composes:
 *   - HubFilterBar  (sticky; owns searchRef; passes state + callbacks; sole New session entry)
 *   - .hub__body
 *       → .hub__grid-scroll (full width — POL-05: GroupSidebar removed; group nav in main Sidebar)
 *           → error state     when error=true
 *           → no-sessions     when sessions.length === 0
 *           → no-matches      when filtered.length === 0 but sessions.length > 0
 *           → SessionCardGrid otherwise
 *
 * Keyboard shortcut (GRID-05): '/' focuses the search input when no input is active.
 *
 * Error-state copy (UI-SPEC Copywriting Contract):
 *   Heading: "Couldn't load sessions"
 *   Body:    "Check that the daemon is running and try again."
 */
export function HubPanel({
  sessions,
  error,
  onNewSession,
  onRename,
  onOpenSession,
  remoteSessions,
  isActive,
  relayPort,
  terminalTheme,
  pluginConfig,
  remoteCapsCached,
  onRequestRemoteCap,
  onRegisterCapAcquired,
  onRegisterCapCancelled,
  fontSizes,
  onFontSizeChange,
  webServerMode,
  webServerRunning,
  onKill,
  onOpenInBrowser,
  onBrowseFiles,
  remotePeers,
  activeGroupId: activeGroupIdProp = null,
  groupDefs: groupDefsProp = [],
  onDropOnGroup: onDropOnGroupProp,
  onGroupCountsChange,
  shellWebShareWarned,
  shellWebShareWarningEnabled,
  onShellWebShareConfirm,
  onShellWebShareCancel,
  onOpenHelp,
}: HubPanelProps): React.ReactElement {
  const [activeFilter, setActiveFilter] = useState<HubFilter>('all')
  const [searchText, setSearchText] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)

  // Phase 134 — modal state: null = closed; non-null = modal open for this session+rect
  interface HubModalState {
    session: SessionInfo
    sourceRect: DOMRect
  }
  const [modalState, setModalState] = useState<HubModalState | null>(null)

  // Phase 137 — share modal state: null = closed; non-null = share modal open for this session
  const [shareModalSession, setShareModalSession] = useState<SessionInfo | null>(null)

  const handleShare = useCallback((session: SessionInfo) => {
    setShareModalSession(session)
  }, [])

  // Phase 166 (RESEARCH Pitfall 3): keep the Share modal's session prop in sync with the
  // 3s Hub poll. shareModalSession is a snapshot taken at open-time; without this effect a
  // funnelActive flip (Plan 05's warm-up completing on the daemon) never reaches the modal,
  // so the warm-up UX would hang forever. Early-return keeps it inert while the modal is
  // closed. Keyed on shareModalSession?.id (not the whole object) to avoid re-running from
  // its own setState.
  useEffect(() => {
    if (!shareModalSession) return
    const updated = sessions.find((s) => s.id === shareModalSession.id)
    if (updated && updated !== shareModalSession) {
      setShareModalSession(updated)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessions, shareModalSession?.id])

  // NOTIF-01: per-session unread counts lifted from HubInteractiveModal's local state.
  // Two sources feed this map: (a) open-modal ChatPanel via onUnreadChange prop threading,
  // and (b) backgrounded sessions via useChatUnreadListeners hook (Plan 160-01).
  // Pitfall 5: always use functional setState with new Map(prev) — mutating in place
  // does NOT trigger re-render since Map identity is unchanged.
  const [unreadMap, setUnreadMap] = useState<Map<string, { count: number; hasMention: boolean }>>(
    new Map(),
  )

  function handleUnreadChange(sessionId: string, count: number, hasMention: boolean) {
    setUnreadMap((prev) => {
      const m = new Map(prev)
      m.set(sessionId, { count, hasMention })
      return m
    })
  }

  // Phase 134 — MODAL-06: pending remote session awaiting cap acquisition
  const [pendingModalSessionId, setPendingModalSessionId] = useState<string | null>(null)
  // Capture the sourceRect at the time of card click so it's available for auto-open
  const pendingSourceRectRef = useRef<DOMRect | null>(null)

  // POL-05: groupDefs + activeGroupId are now props (state lifted to App.tsx).
  // Use prop aliases to keep the filtering logic below unchanged.
  const groupDefs = groupDefsProp
  const activeGroupId = activeGroupIdProp

  // '/' shortcut — focus search input when no input is focused (GRID-05)
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === '/' && (document.activeElement as HTMLElement)?.tagName !== 'INPUT') {
        e.preventDefault()
        searchRef.current?.focus()
        searchRef.current?.select()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  // Reset both filter and search (used by Clear-filter CTA)
  const handleClearFilter = useCallback(() => {
    setActiveFilter('all')
    setSearchText('')
  }, [])

  // Phase 132 / GRID-07: merge local + remote sessions into one unified grid
  const allSessions = [...sessions, ...(remoteSessions ?? [])]

  // GAP-134-A: local vs remote is decided by PROVENANCE (which prop the session came
  // from), NOT by hostname — local sessions carry the machine's os.Hostname(), so a
  // hostname check misclassifies every local session as remote (same rule the
  // usePreviewPoller comment above documents). This set drives the card-click cap gate
  // and the modal's remote-proxy seam.
  const remoteIdSet = React.useMemo(
    () => new Set((remoteSessions ?? []).map((s) => s.id)),
    [remoteSessions],
  )

  // Phase 138 / CARD-03: derive connected set from remoteCapsCached.
  // Mirror attentionIds threading — boolean Set threaded to SessionCardGrid as connectedRemoteIds.
  const connectedRemoteIds = React.useMemo(
    () => remoteCapsCached ?? new Set<string>(),
    [remoteCapsCached],
  )

  // Phase 132 / CARD-07: single shared 3s poller — LOCAL sessions only (the `sessions`
  // prop). Local sessions carry the machine hostname, so we must NOT use hostname to
  // decide local-vs-remote — provenance (this prop vs remoteSessions) is the discriminator.
  const localPreviewTails = usePreviewPoller(sessions, isActive ?? false)
  // NOTIF-01 Part B: lightweight WS listeners for sessions whose modal is currently closed.
  // modalState?.session.id is the exclusion id — prevents double-counting with the open modal's
  // ChatPanel subscription (ChatPanel's onUnreadChange already calls handleUnreadChange above).
  useChatUnreadListeners(
    sessions,
    relayPort ?? 0,
    modalState?.session.id ?? null,
    isActive ?? false,
    handleUnreadChange,
  )
  // Remote sessions have no tail API: seed them as empty ([] → "No output yet", not a
  // perpetual "Loading…" placeholder).
  const previewTails = React.useMemo(() => {
    const m = new Map<string, daemon.StyledSpan[][]>(localPreviewTails)
    for (const r of remoteSessions ?? []) {
      if (!m.has(r.id)) m.set(r.id, [] as daemon.StyledSpan[][])
    }
    return m
  }, [localPreviewTails, remoteSessions])

  /* ATTN-02: float-to-top reorder is within-group; debounce window 1000ms */
  /* Only POSITION is debounced — card content (isAttention) uses the LIVE set above */
  const attentionSortKey = allSessions
    .map((s) => `${s.id}:${isAttentionStatus(deriveHubStatus(s)) ? '1' : '0'}`)
    .join(',')
  const debouncedSortKey = useDebouncedValue(attentionSortKey, 1000)

  /* ATTN-01: live attention set — NOT debounced; card border/icon update immediately */
  /* IN-02: memoized on attentionSortKey (encodes all id:bit changes) so the Set keeps
     referential identity across preview-poll renders where attention membership is stable.
     This avoids unnecessary SessionCardGrid re-renders from always-new Set references. */
  const attentionIds = React.useMemo(
    () => new Set(allSessions.filter((s) => isAttentionStatus(deriveHubStatus(s))).map((s) => s.id)),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [attentionSortKey], // attentionSortKey encodes all id:bit changes
  )

  // Apply status filter + search to allSessions
  const filtered = filterSessions(allSessions, activeFilter, searchText)

  // Phase 132 / GRID-03: apply named-group filter when a group is selected
  // When activeGroupId is '__other__', show sessions not in any named group.
  // When activeGroupId is a group id, show sessions whose memberKey is in that group.
  // When activeGroupId is null, show all filtered sessions.
  let visibleSessions = filtered
  if (activeGroupId !== null && groupDefs.length > 0) {
    if (activeGroupId === '__other__') {
      const allMemberKeys = new Set(groupDefs.flatMap((g) => g.memberKeys))
      visibleSessions = filtered.filter((s) => !allMemberKeys.has(memberKey(s.name, s.workDir)))
    } else {
      const activeDef = groupDefs.find((g) => g.id === activeGroupId)
      if (activeDef) {
        const memberKeySet = new Set(activeDef.memberKeys)
        visibleSessions = filtered.filter((s) => memberKeySet.has(memberKey(s.name, s.workDir)))
      }
    }
  }

  // POL-05: Group CRUD callbacks now delegate to the prop callbacks (state lifted to App.tsx).
  // onDropOnGroupProp is the per-card drag path; handleAssignGroup wraps it for the card menu.
  const handleAssignGroup = useCallback((mKey: string, groupId: string) => {
    onDropOnGroupProp?.(groupId, mKey)
  }, [onDropOnGroupProp])

  // POL-05: Emit per-group + global counts upward via onGroupCountsChange whenever
  // allSessions or groupDefs changes. This avoids lifting allSessions to App.tsx.
  useEffect(() => {
    if (!onGroupCountsChange) return
    const counts: Record<string, { running: number; total: number; attention: number; waiting: number }> = {}
    for (const g of groupDefs) {
      counts[g.id] = computeCounts(allSessions, new Set(g.memberKeys))
    }
    const global = computeGlobalCounts(allSessions)
    onGroupCountsChange(counts, global)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allSessions.map(s => s.id + ':' + s.status).join(','), groupDefs.map(g => g.id + ':' + g.memberKeys.join(';')).join(','), onGroupCountsChange])

  // Phase 134 — MODAL-06: card click handler with remote cap gate
  const handleCardClick = useCallback((session: SessionInfo, rect: DOMRect) => {
    const isRemote = remoteIdSet.has(session.id) // GAP-134-A: provenance, not hostname
    if (isRemote && !remoteCapsCached?.has(session.id)) {
      // Store rect for later use when cap is acquired
      pendingSourceRectRef.current = rect
      setPendingModalSessionId(session.id)
      onRequestRemoteCap?.({ id: session.id, name: session.name, hostname: session.hostname })
      return
    }
    // NOTIF-01: reset unread badge to 0 when the user opens the modal for this session.
    // The user is now actively engaged with the session — badge must clear immediately.
    // Pitfall 5: new Map(prev) prevents stale-closure re-renders (mutating in place is a no-op).
    setUnreadMap((prev) => {
      const m = new Map(prev)
      m.delete(session.id)
      return m
    })
    setModalState({ session, sourceRect: rect })
  }, [remoteIdSet, remoteCapsCached, onRequestRemoteCap])

  // Phase 134 — MODAL-06: cap-acquired handler — called by App.tsx after successful join-code exchange
  const handleCapAcquired = useCallback((sessionId: string) => {
    if (sessionId !== pendingModalSessionId) return
    // Find the session object from all sessions
    const session = [...sessions, ...(remoteSessions ?? [])].find((s) => s.id === sessionId)
    if (!session) {
      setPendingModalSessionId(null)
      pendingSourceRectRef.current = null
      return
    }
    // Use stored rect or fall back to centered rect
    const sourceRect = pendingSourceRectRef.current ??
      new DOMRect(window.innerWidth / 2, window.innerHeight / 2, 0, 0)
    setModalState({ session, sourceRect })
    setPendingModalSessionId(null)
    pendingSourceRectRef.current = null
  }, [pendingModalSessionId, sessions, remoteSessions])

  // Phase 134 — register cap-acquired callback with App.tsx
  useEffect(() => {
    onRegisterCapAcquired?.(handleCapAcquired)
  }, [onRegisterCapAcquired, handleCapAcquired])

  // Phase 134 — WR-01: cancel callback — resets pending state when the join-code modal
  // is dismissed without completing the exchange (no cap acquired). App calls this from
  // RemoteJoinCodeModal onClose so a dismissed modal does not strand pendingModalSessionId.
  const handleCapCancelled = useCallback(() => {
    setPendingModalSessionId(null)
    pendingSourceRectRef.current = null
  }, [])

  // Phase 134 — register cap-cancelled callback with App.tsx
  useEffect(() => {
    onRegisterCapCancelled?.(handleCapCancelled)
  }, [onRegisterCapCancelled, handleCapCancelled])

  // ---- Determine which body to render ----
  let body: React.ReactNode

  if (error) {
    body = (
      <div className="hub__error-state">
        <ExclamationCircleIcon className="hub__error-icon" aria-hidden="true" />
        {/* UI-SPEC Copywriting Contract — error-state copy */}
        <h2 className="hub__error-heading">{"Couldn't load sessions"}</h2>
        <p className="hub__error-body">Check that the daemon is running and try again.</p>
      </div>
    )
  } else if (allSessions.length === 0) {
    body = <HubEmptyState variant="no-sessions" onNewSession={onNewSession} />
  } else if (visibleSessions.length === 0) {
    body = <HubEmptyState variant="no-matches" onClearFilter={handleClearFilter} />
  } else {
    body = (
      <SessionCardGrid
        sessions={visibleSessions}
        onRename={onRename}
        onOpenSession={onOpenSession}
        onCardClick={handleCardClick}
        onShare={handleShare}
        groupDefs={groupDefs.length > 0 ? groupDefs : undefined}
        previewTails={previewTails}
        previewTheme={terminalTheme}
        onAssignGroup={handleAssignGroup}
        attentionIds={attentionIds}
        debouncedSortKey={debouncedSortKey}
        connectedRemoteIds={connectedRemoteIds}
        remoteIdSet={remoteIdSet}
        onKill={onKill}
        onOpenInBrowser={onOpenInBrowser}
        onBrowseFiles={onBrowseFiles}
        unreadBySessionId={unreadMap}
      />
    )
  }

  return (
    <>
      <div className="hub">
        {/* CARD-01: Header strip removed — HubFilterBar's New Session button is the sole entry point */}
        <HubFilterBar
          sessions={allSessions}
          activeFilter={activeFilter}
          searchText={searchText}
          searchRef={searchRef}
          onFilterChange={setActiveFilter}
          onSearchChange={setSearchText}
          onNewSession={onNewSession}
        />

        {/* Per-peer unreachable/empty hint — renders only when relevant peers exist */}
        {(remotePeers ?? [])
          .filter((p) => !p.reachable || p.sessions.length === 0)
          .map((p) => (
            <p key={p.hostname} className="hub__peer-hint">
              {!p.reachable
                ? `${p.hostname} is unreachable`
                : `${p.hostname} has no shared sessions`}
            </p>
          ))}

        {/* POL-05: GroupSidebar removed — grid spans full width; group nav lives in main Sidebar */}
        <div className="hub__body">
          <div className="hub__grid-scroll">
            {body}
          </div>
        </div>
      </div>

      {/* Phase 134 — Hub modal: rendered outside .hub so overlay covers the full Hub surface.
          IN-01: guard relayPort > 0 mirrors the tab grid guard (App.tsx:1535) — avoids
          building ws://127.0.0.1:0/... on a transient 0 value.
          WR-03: terminalTheme is now required on HubPanelProps — the unsafe empty-object cast is removed.
          WR-04: real per-session fontSize + onFontSizeChange instead of hardcoded 14. */}
      {/* Phase 137 — SessionShareModal: rendered outside .hub so overlay covers the full Hub surface */}
      {shareModalSession && (
        <SessionShareModal
          session={shareModalSession}
          webServerMode={webServerMode}
          webServerRunning={webServerRunning}
          onClose={() => setShareModalSession(null)}
          shellWebShareWarned={shellWebShareWarned}
          shellWebShareWarningEnabled={shellWebShareWarningEnabled}
          onShellWebShareConfirm={onShellWebShareConfirm}
          onShellWebShareCancel={onShellWebShareCancel}
          onOpenHelp={onOpenHelp}
        />
      )}

      {modalState && relayPort !== undefined && relayPort > 0 && (() => {
        // Compute isRemote at render time (same rule as handleCardClick).
        // GAP-134-A: provenance (remoteIdSet), NOT hostname — local sessions carry the
        // machine hostname. This discriminator routes remote sessions through the daemon
        // WS proxy seam added in Plan 07 (RelayClient opts.remote → /api/relay/remote/{id}/ws).
        const isRemote = remoteIdSet.has(modalState.session.id)
        return (
          <HubModal
            session={modalState.session}
            sourceRect={modalState.sourceRect}
            relayPort={relayPort}
            fontSize={fontSizes?.[modalState.session.id] ?? DEFAULT_FONT_SIZE}
            theme={terminalTheme}
            pluginConfig={pluginConfig}
            remote={isRemote}
            onFontSizeChange={onFontSizeChange ? (delta) => onFontSizeChange(modalState.session.id, delta) : undefined}
            onClose={() => setModalState(null)}
            onUnreadChange={handleUnreadChange}
          />
        )
      })()}
    </>
  )
}
