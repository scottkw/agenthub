// Phase 139 / CARD-05 — ITheme color-resolution utility.
//
// resolveColor maps StyledSpan color values (from the VT cell grid) through
// the active xterm ITheme. This keeps MiniPreview and HubBriefingModal
// colorblind-safe: colors come from the agent's OWN output, not app-level
// status encoding. Plan 04.

import type { ITheme } from '@xterm/xterm'

// ANSI 0..15 palette slots in ITheme order.
// Index 0 = black, 1 = red, ..., 7 = white, 8 = brightBlack, ..., 15 = brightWhite.
export const ANSI_THEME_KEYS: (keyof ITheme)[] = [
  'black', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan', 'white',
  'brightBlack', 'brightRed', 'brightGreen', 'brightYellow',
  'brightBlue', 'brightMagenta', 'brightCyan', 'brightWhite',
]

/**
 * resolveColor — map a StyledSpan color value to an inline-style color string.
 *
 * @param val   - color from StyledSpan: "ansi:N", "#rrggbb", "" or undefined
 * @param theme - active xterm ITheme (from App.tsx → HubPanel → component)
 * @param isFg  - true for foreground (empty → theme.foreground), false for background (empty → undefined)
 * @returns CSS color string or undefined (React omits undefined inline style props)
 */
export function resolveColor(
  val: string | undefined,
  theme: ITheme,
  isFg: boolean,
): string | undefined {
  if (!val) return isFg ? (theme.foreground ?? undefined) : undefined
  if (val.startsWith('ansi:')) {
    const idx = parseInt(val.slice(5), 10)
    if (idx < 16) return theme[ANSI_THEME_KEYS[idx]] as string | undefined
    return theme.extendedAnsi?.[idx - 16] ?? undefined
  }
  // "#rrggbb" passthrough
  return val
}
