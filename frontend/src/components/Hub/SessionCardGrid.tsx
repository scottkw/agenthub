import React from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { SessionCard } from './SessionCard'
import { memberKey, type HubGroupDef } from '../../lib/hubGroups'
import { deriveHubStatus, isAttentionStatus } from '../../lib/hubStatus'

// ---- Helpers ----

/**
 * Group sessions by their working directory.
 * Sessions with an empty workDir are keyed under ''.
 * NOTE: Does NOT sort — caller applies sortSessionsForDisplay after grouping,
 * gated on debouncedSortKey so position changes happen only after debounce settles.
 *
 * @returns A Map preserving insertion order — one entry per distinct workDir.
 */
export function groupByWorkDir(sessions: SessionInfo[]): Map<string, SessionInfo[]> {
  const groups = new Map<string, SessionInfo[]>()
  for (const s of sessions) {
    const key = s.workDir || ''
    const group = groups.get(key) ?? []
    group.push(s)
    groups.set(key, group)
  }
  return groups
}

/**
 * Group sessions by named group definitions.
 * Sessions are placed into the first matching HubGroupDef (matched by memberKey).
 * Unmatched sessions fall into the '__other__' bucket labelled "Other".
 * Named groups appear in definition order, Other last.
 * GROUP-04: membership key = "${session.name}:::${session.workDir}" — survives session-id churn.
 * NOTE: Does NOT sort — caller applies sortSessionsForDisplay after grouping,
 * gated on debouncedSortKey so position changes happen only after debounce settles.
 */
export function groupByNamedGroups(
  sessions: SessionInfo[],
  groupDefs: HubGroupDef[],
): Map<string, { label: string; sessions: SessionInfo[] }> {
  // Build result map: named groups in definition order, then "Other"
  const result = new Map<string, { label: string; sessions: SessionInfo[] }>()
  for (const g of groupDefs) {
    result.set(g.id, { label: g.name, sessions: [] })
  }
  result.set('__other__', { label: 'Other', sessions: [] })

  for (const s of sessions) {
    const key = memberKey(s.name, s.workDir)
    const matchingGroup = groupDefs.find((g) => g.memberKeys.includes(key))
    if (matchingGroup) {
      result.get(matchingGroup.id)!.sessions.push(s)
    } else {
      result.get('__other__')!.sessions.push(s)
    }
  }
  return result
}

/**
 * basename — extract the last non-empty path segment.
 * Works with both Unix ('/', \\) and Windows ('\\') separators.
 * Do NOT import node:path in frontend code.
 */
function basename(path: string): string {
  // Normalise separators, then split
  const segments = path.replace(/\\/g, '/').split('/')
  // Take the last non-empty segment
  for (let i = segments.length - 1; i >= 0; i--) {
    if (segments[i] !== '') return segments[i]
  }
  return path
}

/* ATTN-02: float-to-top sort within each group — stable; attention before non-attention */
export function sortSessionsForDisplay(sessions: SessionInfo[]): SessionInfo[] {
  return [...sessions].sort((a, b) => {
    const aAttn = isAttentionStatus(deriveHubStatus(a)) ? 0 : 1
    const bAttn = isAttentionStatus(deriveHubStatus(b)) ? 0 : 1
    return aAttn - bAttn
  })
}

/* ATTN-02: FLIP animation hook — measures positions before/after sort, animates with transform */
/* ATTN-02: reorder animation: FLIP pattern, 300ms ease; suppressed under prefers-reduced-motion */
function useFLIPAnimation(enabled: boolean) {
  const nodeMap = React.useRef<Map<string, HTMLElement>>(new Map())
  const prevPositions = React.useRef<Map<string, DOMRect>>(new Map())

  const registerNode = React.useCallback((id: string, el: HTMLElement | null) => {
    if (el) nodeMap.current.set(id, el)
    else nodeMap.current.delete(id)
  }, [])

  // WR-02: capture ALWAYS updates prevPositions regardless of prefers-reduced-motion,
  // so a mid-session preference change from reduce→no-preference doesn't animate
  // to stale/wrong positions. The animation guard stays in playFLIP only.
  const capturePositions = React.useCallback(() => {
    if (!enabled) return
    const snap = new Map<string, DOMRect>()
    for (const [id, el] of nodeMap.current) snap.set(id, el.getBoundingClientRect())
    prevPositions.current = snap
  }, [enabled])

  const playFLIP = React.useCallback(() => {
    if (!enabled) return
    /* ATTN-02: reorder animation: FLIP pattern, 300ms ease; suppressed under prefers-reduced-motion */
    if (typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    for (const [id, el] of nodeMap.current) {
      const prev = prevPositions.current.get(id)
      if (!prev) continue
      const next = el.getBoundingClientRect()
      const deltaY = prev.top - next.top
      if (Math.abs(deltaY) < 1) continue
      el.style.transform = `translateY(${deltaY}px)`
      el.style.transition = 'none'
      requestAnimationFrame(() => {
        el.style.transform = ''
        el.style.transition = 'transform 300ms ease'
        el.addEventListener('transitionend', () => { el.style.transition = '' }, { once: true })
      })
    }
  }, [enabled])

  return { registerNode, capturePositions, playFLIP }
}

// ---- Props ----

export interface SessionCardGridProps {
  /** Full list of sessions to group and render. */
  sessions: SessionInfo[]
  /** Passed through to each SessionCard for inline rename. */
  onRename: (id: string, name: string) => void
  /** Passed through to each SessionCard's Open button. Phase 131 UAT follow-up. */
  onOpenSession?: (sessionId: string, name: string, cli: string) => void
  /** Phase 132 — named group definitions; when non-empty overrides workDir grouping */
  groupDefs?: HubGroupDef[]
  /** Phase 132 — tail lines map from usePreviewPoller, keyed by session ID */
  previewTails?: Map<string, string[]>
  /** Phase 132 — fires when user assigns via card overflow menu or DnD */
  onAssignGroup?: (memberKey: string, groupId: string) => void
  /** ATTN-02: live attention set — NOT debounced; used for per-card isAttention prop */
  attentionIds?: Set<string>
  /** ATTN-04: debounced sort key — triggers reorder-position updates after 1s debounce */
  debouncedSortKey?: string
  /** Phase 134 — threaded to each SessionCard for modal open trigger */
  onCardClick?: (session: SessionInfo, rect: DOMRect) => void
  /** Phase 137 / D-12 — threaded to each SessionCard Share button */
  onShare?: (session: SessionInfo) => void
  /** Phase 138 — Set of session IDs with a cached cap (isConnected signal) */
  connectedRemoteIds?: Set<string>
  /** Phase 138 — Set of remote session IDs (provenance-based isRemote signal) */
  remoteIdSet?: Set<string>
  /** Phase 138 — Kill handler (threaded from HubPanel → App.handleCloseTab) */
  onKill?: (sessionId: string) => void
  /** Phase 138 — Open remote session in browser (threaded from HubPanel) */
  onOpenInBrowser?: (url: string) => void
  /** Phase 138 — Browse remote files (threaded from HubPanel) */
  onBrowseFiles?: (sessionId: string, sessionName: string) => void
}

// ---- Component ----

/**
 * SessionCardGrid — groups SessionCards by working directory with group headers.
 *
 * Layout per UI-SPEC Layout Contract:
 *   .hub__group
 *     .hub__group-header (h2, role="heading" aria-level=2)
 *       span[title={fullPath}]  ← basename shown; full path as tooltip
 *     .hub__card-row [role="list"]
 *       div [role="listitem"]
 *         SessionCard
 *
 * Phase 132 named-group branch:
 *   When groupDefs is non-empty, uses groupByNamedGroups instead of groupByWorkDir.
 *   Group headers show named group labels (in definition order), Other last (GROUP-04).
 *
 * GRID-02: groupByWorkDir drives the grouping logic when no named groups are defined.
 * Accessibility: role=list/listitem satisfies UI-SPEC §Accessibility rule 6.
 * No node:path import — basename helper is self-contained.
 */
export function SessionCardGrid({
  sessions,
  onRename,
  onOpenSession,
  groupDefs,
  previewTails,
  onAssignGroup,
  attentionIds,
  debouncedSortKey,
  onCardClick,
  onShare,
  connectedRemoteIds,
  remoteIdSet,
  onKill,
  onOpenInBrowser,
  onBrowseFiles,
}: SessionCardGridProps): React.ReactElement {
  /* ATTN-02: reorder animation FLIP 300ms ease; suppressed under prefers-reduced-motion */
  const { registerNode, capturePositions, playFLIP } = useFLIPAnimation(true)

  // CR-01 + WR-03: capture + play are driven by the SAME debouncedSortKey dependency.
  // Capture runs in the cleanup (before the next debouncedSortKey-triggered DOM mutation),
  // play runs after the DOM update. The standalone always-running capture effect is removed
  // so prevPositions is never overwritten by unrelated renders (preview polls, filter changes).
  React.useLayoutEffect(() => {
    playFLIP()
    return () => {
      capturePositions()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedSortKey])

  // WR-03: the RELATIVE sort order is derived only when debouncedSortKey changes, but the
  // SET of displayed sessions follows the live sessions prop (filter/search changes apply
  // immediately). Strategy: memoize a stable ID-order array on debouncedSortKey, then apply
  // that order to the live sessions. New/removed sessions (filter changes) take effect live;
  // only the attention-float reordering waits for the 1s debounce to settle.
  //   - card border/icon uses live attentionIds (immediate, from HubPanel)
  //   - card position within group uses the debounced sort order
  const sortedOrder = React.useMemo(
    () => sortSessionsForDisplay(sessions).map((s) => s.id),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [debouncedSortKey], // re-compute order only when debounced key changes
  )
  // Apply the stable order to the live sessions (handles filter/add/remove immediately)
  const sortedSessions = React.useMemo(() => {
    const idIndex = new Map(sortedOrder.map((id, i) => [id, i]))
    return [...sessions].sort((a, b) => {
      const ai = idIndex.get(a.id) ?? Infinity
      const bi = idIndex.get(b.id) ?? Infinity
      return ai - bi
    })
  }, [sessions, sortedOrder])

  // Phase 132: when groupDefs is non-empty, use named-group grouping; else fall back to workDir
  if (groupDefs && groupDefs.length > 0) {
    // Named-group render path (Phase 132) — sort THEN group (sort is debounce-gated above)
    const namedGroups = groupByNamedGroups(sortedSessions, groupDefs)

    return (
      <>
        {Array.from(namedGroups.entries()).map(([groupId, { label, sessions: groupSessions }]) => (
          <div key={groupId} className="hub__group">
            {/* Group header: 11px / 600 / uppercase — named group label */}
            <h2 className="hub__group-header" role="heading" aria-level={2}>
              <span>{label}</span>
            </h2>

            {/* Card grid — role=list per UI-SPEC §Accessibility rule 6 */}
            <div role="list" className="hub__card-row">
              {groupSessions.map((s) => (
                <div
                  role="listitem"
                  key={s.id}
                  ref={(el) => registerNode(s.id, el)}
                >
                  <SessionCard
                    session={s}
                    onRename={onRename}
                    onOpenSession={onOpenSession}
                    onCardClick={onCardClick}
                    onShare={onShare}
                    previewLines={previewTails?.get(s.id)}
                    groupDefs={groupDefs}
                    onAssignGroup={onAssignGroup}
                    isAttention={attentionIds?.has(s.id)}
                    isRemote={remoteIdSet?.has(s.id)}
                    isConnected={connectedRemoteIds?.has(s.id)}
                    onKill={onKill}
                    onOpenInBrowser={onOpenInBrowser}
                    onBrowseFiles={onBrowseFiles}
                  />
                </div>
              ))}
            </div>
          </div>
        ))}
      </>
    )
  }

  // Phase 131 fallback: workDir-based grouping — sort THEN group (sort is debounce-gated above)
  const groups = groupByWorkDir(sortedSessions)

  return (
    <>
      {Array.from(groups.entries()).map(([workDir, groupSessions]) => {
        const headerLabel = workDir ? basename(workDir) : 'Other'
        const titleValue = workDir || undefined

        return (
          <div key={workDir || '__other__'} className="hub__group">
            {/* Group header: 11px / 600 / uppercase — mirrors .remote-panel__peer-header */}
            <h2 className="hub__group-header" role="heading" aria-level={2}>
              <span title={titleValue}>{headerLabel}</span>
            </h2>

            {/* Card grid — role=list per UI-SPEC §Accessibility rule 6 */}
            <div role="list" className="hub__card-row">
              {groupSessions.map((s) => (
                <div
                  role="listitem"
                  key={s.id}
                  ref={(el) => registerNode(s.id, el)}
                >
                  <SessionCard
                    session={s}
                    onRename={onRename}
                    onOpenSession={onOpenSession}
                    onCardClick={onCardClick}
                    onShare={onShare}
                    previewLines={previewTails?.get(s.id)}
                    groupDefs={groupDefs}
                    onAssignGroup={onAssignGroup}
                    isAttention={attentionIds?.has(s.id)}
                    isRemote={remoteIdSet?.has(s.id)}
                    isConnected={connectedRemoteIds?.has(s.id)}
                    onKill={onKill}
                    onOpenInBrowser={onOpenInBrowser}
                    onBrowseFiles={onBrowseFiles}
                  />
                </div>
              ))}
            </div>
          </div>
        )
      })}
    </>
  )
}
