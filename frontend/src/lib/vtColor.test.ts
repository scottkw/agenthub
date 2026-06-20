// Phase 139 Plan 01 — RED tests for resolveColor (CARD-05).
//
// These tests are RED until Plan 04 creates frontend/src/lib/vtColor.ts
// and exports the resolveColor function.
//
// The tests lock the resolveColor contract that Plan 04 must satisfy:
//   - '' + isFg=true  → theme.foreground
//   - '' + isFg=false → undefined
//   - 'ansi:2'        → theme.green (ANSI index 2 maps to ITheme.green)
//   - 'ansi:16'       → theme.extendedAnsi[0] (extended color index 0)
//   - '#abcdef'       → '#abcdef' (hex passthrough, unchanged)

import { describe, it, expect } from 'vitest'
import type { ITheme } from '@xterm/xterm'
// RED: vtColor.ts does not exist yet — will fail to resolve until Plan 04.
import { resolveColor } from '../lib/vtColor'

// Minimal ITheme fixture covering the cases under test.
const theme: ITheme = {
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
  extendedAnsi: [
    '#111111', // index 16 (extendedAnsi[0])
    '#222222', // index 17
    '#333333', // index 18
  ],
}

describe('resolveColor', () => {
  it('empty string + isFg=true → theme.foreground', () => {
    expect(resolveColor('', theme, true)).toBe(theme.foreground)
  })

  it('undefined + isFg=true → theme.foreground', () => {
    expect(resolveColor(undefined, theme, true)).toBe(theme.foreground)
  })

  it('empty string + isFg=false → undefined (no default background)', () => {
    expect(resolveColor('', theme, false)).toBeUndefined()
  })

  it('undefined + isFg=false → undefined', () => {
    expect(resolveColor(undefined, theme, false)).toBeUndefined()
  })

  it('ansi:2 → theme.green (ANSI index 2 = green)', () => {
    expect(resolveColor('ansi:2', theme, true)).toBe(theme.green)
  })

  it('ansi:0 → theme.black (ANSI index 0 = black)', () => {
    expect(resolveColor('ansi:0', theme, true)).toBe(theme.black)
  })

  it('ansi:1 → theme.red (ANSI index 1 = red)', () => {
    expect(resolveColor('ansi:1', theme, true)).toBe(theme.red)
  })

  it('ansi:8 → theme.brightBlack (ANSI index 8 = brightBlack)', () => {
    expect(resolveColor('ansi:8', theme, true)).toBe(theme.brightBlack)
  })

  it('ansi:15 → theme.brightWhite (ANSI index 15 = brightWhite)', () => {
    expect(resolveColor('ansi:15', theme, true)).toBe(theme.brightWhite)
  })

  it('ansi:16 → theme.extendedAnsi[0] (extended color index 0)', () => {
    expect(resolveColor('ansi:16', theme, true)).toBe(theme.extendedAnsi![0])
  })

  it('#abcdef → #abcdef (hex passthrough unchanged)', () => {
    expect(resolveColor('#abcdef', theme, true)).toBe('#abcdef')
  })

  it('#123456 passthrough works as background too', () => {
    expect(resolveColor('#123456', theme, false)).toBe('#123456')
  })
})
