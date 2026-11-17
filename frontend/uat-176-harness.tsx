// Phase 176-03 UAT harness — isolated live mount of the REAL MiniPreview component
// (no Wails app shell) against the real style.css, inside a container sized like the
// real .hub-card (240-360px wide). Feeds StyledSpan[][] mirroring EXACTLY how the
// backend produces it (internal/daemon/engine.go GetSessionStyledTailLines): ONE
// StyledSpan PER CHARACTER CELL (not per style-run), derived from the #127 long-line
// repro command:
//   sh -c 'i=0; while true; do i=$((i+1)); printf "%s  live tick #%d\n" "$(date +%T)" "$i"; sleep 1; done'
// Purpose: reproduce (or disprove) the char-per-row stacking LIVE before any code
// change (D-01/D-02). Temporary; safe to delete; NOT committed.
import React from 'react'
import { createRoot } from 'react-dom/client'
import './src/style.css'
import { MiniPreview } from './src/components/Hub/MiniPreview'
import type { ITheme } from '@xterm/xterm'

interface StyledSpan {
  c: string
  fg?: string
  bg?: string
  b?: boolean
}

const testTheme: ITheme = {
  foreground: '#c0caf5',
  background: '#1a1b26',
  black: '#15161e',
  red: '#f7768e',
  green: '#9ece6a',
  yellow: '#e0af68',
  blue: '#7aa2f7',
  magenta: '#bb9af7',
  cyan: '#7dcfff',
  white: '#a9b1d6',
  brightBlack: '#414868',
  brightRed: '#f7768e',
  brightGreen: '#9ece6a',
  brightYellow: '#e0af68',
  brightBlue: '#7aa2f7',
  brightMagenta: '#bb9af7',
  brightCyan: '#7dcfff',
  brightWhite: '#c0caf5',
}

// Build one StyledSpan PER CHARACTER, exactly like engine.go's per-cell extraction
// (row = append(row, StyledSpan{Char: cell.Content, ...}) for x in 0..cols).
// A real terminal row is `cols` wide (commonly 80) — pad/truncate to 80 to mirror
// a real VT grid row rather than a raw unbounded string.
function lineToStyledSpans(text: string, cols = 80, colorEveryNth?: number): StyledSpan[] {
  const padded = text.length >= cols ? text.slice(0, cols) : text.padEnd(cols, ' ')
  const spans: StyledSpan[] = []
  for (let x = 0; x < cols; x++) {
    const ch = padded[x]
    const isColored = colorEveryNth != null && x > 0 && x % colorEveryNth === 0
    spans.push({ c: ch, fg: isColored ? 'ansi:2' : undefined, b: isColored })
  }
  return spans
}

// #127 repro data: long unbroken "live tick" lines from the repro command,
// each one padded/truncated to a full 80-col VT row (one StyledSpan per cell).
const rawLines = [
  '14:23:05  live tick #140',
  '14:23:06  live tick #141',
  '14:23:07  live tick #142 -- some extra trailing content to overflow the card width',
  '14:23:08  live tick #143',
]
const styledLines: StyledSpan[][] = rawLines.map((line, i) =>
  lineToStyledSpans(line, 80, i === 2 ? 7 : undefined)
)

function Harness(): React.ReactElement {
  return (
    // Mirror the real Hub card shell: .hub-card (240-360px wide) > .hub-card__preview.
    <div className="hub-card" data-agent="shell" style={{ width: 300 }}>
      <div className="hub-card__row1" style={{ padding: '4px 0' }}>
        <span className="hub-card__status-label">176-03 BUG-07 repro harness</span>
      </div>
      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      <MiniPreview lines={styledLines} theme={testTheme} />
    </div>
  )
}

createRoot(document.getElementById('root')!).render(<Harness />)
