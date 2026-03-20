import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App', () => {
  // UILAY-02: App.tsx imports and renders <StatusBar> inside terminal-wrapper
  describe('StatusBar integration (UILAY-02)', () => {
    it('imports StatusBar from components/StatusBar', () => {
      expect(raw).toContain("import { StatusBar } from './components/StatusBar'")
    })

    it('renders <StatusBar inside terminal-wrapper', () => {
      // Both TerminalPanel and StatusBar must appear inside the terminal-wrapper block
      const wrapperBlock = raw.slice(raw.indexOf('terminal-wrapper'))
      expect(wrapperBlock).toContain('<StatusBar')
    })

    it('passes required props to StatusBar', () => {
      expect(raw).toContain('webServerRunning={webServerRunning}')
      expect(raw).toContain('webEnabled={!!webEnabled[tab.sessionId]}')
      expect(raw).toContain('sessionURL={sessionURLs[tab.sessionId]}')
    })
  })

  // UILAY-03: Old web-serving-bar overlay has been removed from App.tsx
  describe('old overlay removed (UILAY-03)', () => {
    it('does not contain web-serving-bar class', () => {
      expect(raw).not.toContain('web-serving-bar')
    })

    it('does not contain web-toggle-btn class', () => {
      expect(raw).not.toContain('web-toggle-btn')
    })

    it('does not contain web-session-url class', () => {
      expect(raw).not.toContain('web-session-url')
    })

    it('does not contain copy-token-btn class', () => {
      expect(raw).not.toContain('copy-token-btn')
    })
  })

  describe('per-tab font size state', () => {
    it('declares fontSizes state as Record<string, number>', () => {
      expect(raw).toContain('Record<string, number>')
      expect(raw).toContain('fontSizes')
    })

    it('defines DEFAULT_FONT_SIZE = 14', () => {
      expect(raw).toContain('DEFAULT_FONT_SIZE = 14')
    })

    it('defines handleFontSizeChange callback', () => {
      expect(raw).toContain('handleFontSizeChange')
    })

    it('clamps font size between 6 and 32', () => {
      expect(raw).toContain('Math.max(6')
      expect(raw).toContain('Math.min(32')
    })

    it('passes fontSize prop to TerminalPanel', () => {
      expect(raw).toContain('fontSize=')
    })

    it('passes onFontSizeChange prop to TerminalPanel', () => {
      expect(raw).toContain('onFontSizeChange=')
    })

    it('cleans up fontSizes entry on tab close', () => {
      expect(raw).toContain('setFontSizes')
      // Cleanup pattern: delete n[id] in handleCloseTab
      const closeSection = raw.slice(raw.indexOf('handleCloseTab'))
      expect(closeSection).toContain('setFontSizes')
    })
  })
})
