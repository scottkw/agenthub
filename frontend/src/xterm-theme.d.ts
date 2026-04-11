// Type declaration for xterm-theme (no @types package available).
// The module exports a plain object mapping theme name strings to ITheme objects.
import type { ITheme } from '@xterm/xterm'

declare module 'xterm-theme' {
  const themes: Record<string, ITheme>
  export = themes
}
