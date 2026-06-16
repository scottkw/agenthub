import React from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { SessionCard } from './SessionCard'

// ---- Helpers ----

/**
 * Group sessions by their working directory.
 * Sessions with an empty workDir are keyed under ''.
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

// ---- Props ----

export interface SessionCardGridProps {
  /** Full list of sessions to group and render. */
  sessions: SessionInfo[]
  /** Passed through to each SessionCard for inline rename. */
  onRename: (id: string, name: string) => void
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
 * Group key: s.workDir || '' — empty string maps to the "Other" bucket.
 * Group header label: basename(workDir) or "Other" when key is ''.
 *
 * GRID-02: groupByWorkDir drives the grouping logic.
 * Accessibility: role=list/listitem satisfies UI-SPEC §Accessibility rule 6.
 * No node:path import — basename helper is self-contained.
 */
export function SessionCardGrid({
  sessions,
  onRename,
}: SessionCardGridProps): React.ReactElement {
  const groups = groupByWorkDir(sessions)

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
                <div role="listitem" key={s.id}>
                  <SessionCard session={s} onRename={onRename} />
                </div>
              ))}
            </div>
          </div>
        )
      })}
    </>
  )
}
