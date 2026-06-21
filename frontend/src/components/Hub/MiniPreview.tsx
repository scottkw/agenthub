/* CARD-07: mini preview is plain text snapshot — NO xterm instance; polling interval 3s shared across all cards */
// Phase 139 / CARD-05: prop type migrated from string[] to StyledSpan[][] + theme (ITheme).
// MUST NOT import or mount Terminal from @xterm/xterm (type-only import allowed).
import React from 'react'
import type { ITheme } from '@xterm/xterm'
import type { daemon } from '../../wailsjs/go/models'
import { resolveColor } from '../../lib/vtColor'

type StyledSpan = daemon.StyledSpan

export interface MiniPreviewProps {
  /** Styled lines from usePreviewPoller. undefined = not yet fetched (loading). Empty array = no output. */
  lines: StyledSpan[][] | undefined
  /** Active xterm ITheme — used to resolve ansi:N color codes through the theme palette. */
  theme: ITheme
  /** When true (stopped-ok), preview gets dim opacity via parent card's CSS. */
  dimmed?: boolean
}

/**
 * MiniPreview — ROW 6 of SessionCard. Styled cell grid snapshot of last 4 lines.
 * NEVER mounts an xterm instance (CARD-07 hard constraint). aria-hidden — decorative only.
 * Colors are mapped through ITheme via resolveColor (D-03 / D-04).
 */
export function MiniPreview({ lines, theme }: MiniPreviewProps): React.ReactElement {
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
      {lines.map((row, i) => (
        <div key={i} className="hub-card__preview-line">
          {row.length === 0 ? (
            // Empty row — preserve line height
            <span>{' '}</span>
          ) : (
            row.map((span, j) => (
              <span
                key={j}
                style={{
                  color: resolveColor(span.fg, theme, true),
                  background: resolveColor(span.bg, theme, false),
                  fontWeight: span.b ? 'bold' : undefined,
                }}
              >
                {span.c || ' '}
              </span>
            ))
          )}
        </div>
      ))}
    </div>
  )
}
