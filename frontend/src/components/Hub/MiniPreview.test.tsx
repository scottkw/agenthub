// Phase 139 Plan 01 — MiniPreview tests updated for StyledSpan[][] prop (CARD-05).
//
// Existing tests are migrated to use StyledSpan[][] empty shapes.
// New tests assert:
//   - Colored spans render with inline style from the theme (FG color applied)
//   - Bold spans have fontWeight bold
//   - No .xterm element is mounted (CARD-07 hard constraint)
//
// New StyledSpan[][] tests are RED until Plan 04 updates MiniPreview.tsx
// to accept StyledSpan[][] instead of string[].

import { describe, it, expect, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import { MiniPreview } from './MiniPreview'
import type { ITheme } from '@xterm/xterm'

// Minimal ITheme fixture for color-resolution tests.
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// StyledSpan shape (mirrors the Go wire type and Plan 04 TS type).
interface StyledSpan {
  c: string
  fg?: string
  bg?: string
  b?: boolean
}

function renderPreviewStyled(lines: StyledSpan[][] | undefined, theme?: ITheme) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    // RED until Plan 04 changes MiniPreview props to StyledSpan[][] + theme.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    root.render(React.createElement(MiniPreview as any, { lines, theme }))
  })
  return { container, root }
}

// ---------------------------------------------------------------------------
// Existing state tests — migrated to StyledSpan[][] empty shapes.
// "undefined" and "[]" are unchanged semantically.
// ---------------------------------------------------------------------------

describe('MiniPreview', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  describe('loading state (lines === undefined)', () => {
    it('renders "Loading…" text', () => {
      const { container } = renderPreviewStyled(undefined)
      expect(container.textContent).toContain('Loading…')
    })

    it('applies hub-card__preview--loading modifier class', () => {
      const { container } = renderPreviewStyled(undefined)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).toContain('hub-card__preview--loading')
    })

    it('has aria-hidden="true" on the outer pane', () => {
      const { container } = renderPreviewStyled(undefined)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.getAttribute('aria-hidden')).toBe('true')
    })
  })

  describe('empty state (lines === [])', () => {
    it('renders "No output yet" text', () => {
      const { container } = renderPreviewStyled([])
      expect(container.textContent).toContain('No output yet')
    })

    it('applies hub-card__preview--empty modifier class', () => {
      const { container } = renderPreviewStyled([])
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).toContain('hub-card__preview--empty')
    })

    it('has aria-hidden="true" on the outer pane', () => {
      const { container } = renderPreviewStyled([])
      const pane = container.firstElementChild as HTMLElement
      expect(pane.getAttribute('aria-hidden')).toBe('true')
    })
  })

  // ---------------------------------------------------------------------------
  // StyledSpan rendering tests — RED until Plan 04 updates MiniPreview.tsx
  // ---------------------------------------------------------------------------

  describe('StyledSpan[][] rendering (CARD-05) — RED until Plan 04', () => {
    it('renders colored span with inline color from theme for FG=ansi:2 (green)', () => {
      // [[ {c:'g',fg:'ansi:2',b:true}, {c:'o'} ]] — one row, two spans
      const spans: StyledSpan[][] = [[
        { c: 'g', fg: 'ansi:2', b: true },
        { c: 'o' },
      ]]
      const { container } = renderPreviewStyled(spans, testTheme)

      // The container must have a <span> whose inline style includes the green color.
      // theme.green is '#9ece6a' for this fixture.
      const allSpans = container.querySelectorAll('span')
      const coloredSpan = Array.from(allSpans).find((el) => {
        const style = (el as HTMLElement).style
        return style.color && style.color !== ''
      })
      expect(coloredSpan).not.toBeNull()

      // The colored span should have fontWeight bold (b: true).
      const boldSpan = Array.from(allSpans).find((el) => {
        const style = (el as HTMLElement).style
        return style.fontWeight === 'bold' || style.fontWeight === '700'
      })
      expect(boldSpan).not.toBeNull()
    })

    it('does NOT mount any .xterm element (CARD-07 hard constraint)', () => {
      const spans: StyledSpan[][] = [[{ c: 'h' }, { c: 'i' }]]
      const { container } = renderPreviewStyled(spans, testTheme)

      // MiniPreview must never import or mount Terminal from @xterm/xterm.
      const xtermEl = container.querySelector('.xterm')
      expect(xtermEl).toBeNull()
    })

    it('renders hub-card__preview-line rows for each StyledSpan row', () => {
      const spans: StyledSpan[][] = [
        [{ c: 'a' }, { c: 'b' }],
        [{ c: 'c' }, { c: 'd' }],
      ]
      const { container } = renderPreviewStyled(spans, testTheme)

      const rows = container.querySelectorAll('.hub-card__preview-line')
      expect(rows.length).toBeGreaterThanOrEqual(2)
    })

    it('has aria-hidden="true" on outer pane in data state', () => {
      const spans: StyledSpan[][] = [[{ c: 'x' }]]
      const { container } = renderPreviewStyled(spans, testTheme)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.getAttribute('aria-hidden')).toBe('true')
    })
  })
})
