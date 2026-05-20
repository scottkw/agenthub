import React, { useEffect, useState } from 'react'
import { ArrowPathIcon } from '@heroicons/react/24/outline'
import type { BreadcrumbSegment } from '../../lib/filesTypes'

export interface BreadcrumbBarProps {
  /**
   * Path segments from session cwd to the current directory.
   * First segment is ALWAYS the session cwd root (rendered as non-clickable
   * "session/" text). Last segment is the current directory (rendered as
   * text marked as the current page. Middle segments are clickable.
   */
  segments: BreadcrumbSegment[]
  /** ISO timestamp of last successful list; null while no list has loaded. */
  refreshedAt: string | null
  /** Click handler for a middle segment. Receives that segment's pathFromCwd. */
  onNavigateTo: (pathFromCwd: string) => void
  /** Click handler for the refresh icon. */
  onRefresh: () => void
}

/**
 * Render a human-readable "refreshed N{s|m} ago" string.
 *
 * - <5s   → "just now"
 * - 5..59s → "Last refreshed Ns ago"
 * - ≥60s   → "Last refreshed Nm ago"
 *
 * Returns "" when refreshedAt is null (caller renders nothing).
 */
function formatRefreshedAt(refreshedAt: string | null, now: number): string {
  if (refreshedAt === null) return ''
  const then = Date.parse(refreshedAt)
  if (Number.isNaN(then)) return ''
  const deltaMs = Math.max(0, now - then)
  const deltaSec = Math.floor(deltaMs / 1000)
  if (deltaSec < 5) return 'just now'
  if (deltaSec < 60) return `Last refreshed ${deltaSec}s ago`
  const deltaMin = Math.floor(deltaSec / 60)
  return `Last refreshed ${deltaMin}m ago`
}

/**
 * Internal hook: keep the "refreshed Ns ago" text live by ticking every 5s.
 *
 * RESEARCH Pitfall 4: don't recompute every render — drive updates via a
 * setInterval keyed on the refreshedAt prop so the component only re-renders
 * when the displayed delta would change visibly.
 */
function useRefreshedText(refreshedAt: string | null): string {
  const [now, setNow] = useState<number>(() => Date.now())
  useEffect(() => {
    if (refreshedAt === null) return
    setNow(Date.now()) // immediate update on prop change
    const id = window.setInterval(() => setNow(Date.now()), 5000)
    return () => window.clearInterval(id)
  }, [refreshedAt])
  return formatRefreshedAt(refreshedAt, now)
}

export function BreadcrumbBar({
  segments,
  refreshedAt,
  onNavigateTo,
  onRefresh,
}: BreadcrumbBarProps): React.ReactElement {
  const refreshedText = useRefreshedText(refreshedAt)
  const lastIndex = segments.length - 1

  return (
    <nav
      className="file-browser__breadcrumb"
      aria-label="Path"
      data-testid="file-browser-breadcrumb"
    >
      <ol className="file-browser__breadcrumb-list">
        {segments.map((seg, idx) => {
          const isRoot = idx === 0
          const isCurrent = idx === lastIndex
          const testId = `file-browser-breadcrumb-segment-${idx}`
          // Root (index 0) — always non-clickable plain text.
          // Last segment (the current directory) — text only, aria-current=page.
          // Middle segments — button that navigates on click.
          if (isRoot) {
            return (
              <li key={idx} className="file-browser__breadcrumb-item">
                <span
                  className="file-browser__breadcrumb-root"
                  data-testid={testId}
                >
                  {seg.name}/
                </span>
              </li>
            )
          }
          if (isCurrent) {
            return (
              <li key={idx} className="file-browser__breadcrumb-item">
                <span
                  className="file-browser__breadcrumb-current"
                  aria-current="page"
                  data-testid={testId}
                >
                  {seg.name}
                </span>
              </li>
            )
          }
          return (
            <li key={idx} className="file-browser__breadcrumb-item">
              <button
                type="button"
                className="file-browser__breadcrumb-segment"
                data-testid={testId}
                onClick={() => onNavigateTo(seg.pathFromCwd)}
              >
                {seg.name}
              </button>
            </li>
          )
        })}
      </ol>
      <div className="file-browser__breadcrumb-actions">
        {refreshedText !== '' && (
          <span className="file-browser__breadcrumb-refreshed">
            {refreshedText}
          </span>
        )}
        <button
          type="button"
          className="file-browser__breadcrumb-refresh"
          aria-label="Refresh directory listing"
          data-testid="file-browser-refresh"
          onClick={onRefresh}
        >
          <ArrowPathIcon
            className="file-browser__breadcrumb-refresh-icon"
            aria-hidden="true"
            width={14}
            height={14}
          />
        </button>
      </div>
    </nav>
  )
}
