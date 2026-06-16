/* CARD-07: mini preview is plain text snapshot — NO xterm instance; polling interval 3s shared across all cards */
import React from 'react'

export interface MiniPreviewProps {
  /** Lines from usePreviewPoller. undefined = not yet fetched (loading). Empty array = no output. */
  lines: string[] | undefined
  /** When true (stopped-ok), preview gets dim opacity via parent card's CSS. */
  dimmed?: boolean
}

/**
 * MiniPreview — ROW 6 of SessionCard. Plain text snapshot of last 4 lines.
 * NEVER mounts an xterm instance. aria-hidden — decorative only.
 */
export function MiniPreview({ lines }: MiniPreviewProps): React.ReactElement {
  if (lines === undefined) {
    return (
      <div className="hub-card__preview hub-card__preview--loading" aria-hidden="true">
        <span className="hub-card__preview-line">Loading…</span>
      </div>
    )
  }
  if (lines.length === 0) {
    return (
      <div className="hub-card__preview hub-card__preview--empty" aria-hidden="true">
        <span className="hub-card__preview-line">No output yet</span>
      </div>
    )
  }
  return (
    <div className="hub-card__preview" aria-hidden="true">
      {lines.map((line, i) => (
        // key by index — order is stable within a snapshot
        <div key={i} className="hub-card__preview-line">{line || ' '}</div>
      ))}
    </div>
  )
}
