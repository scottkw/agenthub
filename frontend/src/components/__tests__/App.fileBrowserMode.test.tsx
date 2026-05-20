// Phase 120-06 Task 2 — source-inspection coverage that App.tsx selects
// fbBaseURL + capToken correctly per detectMode().
//
// Mounting the real <App /> component would require stubbing ~30 wailsjs
// imports and the entire xterm runtime; the existing App.test.tsx /
// App.wiring.test.tsx files solve that by source-inspecting App.tsx?raw.
// We follow the same pattern so the test stays hermetic and resilient to
// downstream refactors of unrelated state.
//
// The assertions enforce the behaviour contract from 120-06-PLAN.md
// Task 2: the file mounts detectMode() once, gates init/retryInit/poll
// effects with `if (mode === 'web') return`, and selects fbBaseURL +
// capToken via mode at the file-browser tab render site.

import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App.tsx — Phase 120-06 web-mode wiring', () => {
  describe('module-level imports', () => {
    it('imports detectMode and readWebModeParams from lib/webMode', () => {
      expect(raw).toMatch(/import\s*\{[^}]*\bdetectMode\b[^}]*\breadWebModeParams\b[^}]*\}\s*from\s*['"]\.\/lib\/webMode['"]/)
    })

    it('imports useMemo from react (required to memoize the URL params)', () => {
      expect(raw).toMatch(/import\s*(?:React,\s*)?\{[^}]*\buseMemo\b[^}]*\}\s*from\s*['"]react['"]/)
    })
  })

  describe('mode capture is a single source of truth', () => {
    it('captures mode exactly once via const mode = detectMode()', () => {
      const matches = raw.match(/const\s+mode\s*=\s*detectMode\(\)/g) ?? []
      expect(matches.length, 'mode binding must appear exactly once').toBe(1)
    })

    it('memoizes the URL params once on first mount via useMemo', () => {
      expect(raw).toMatch(/const\s+webParams\s*=\s*useMemo\(\s*\(\)\s*=>\s*readWebModeParams\(\)/)
    })
  })

  describe('Wails RPC gating: web mode skips the Wails suite', () => {
    it('gates the init useEffect with an early-return for web mode', () => {
      // First mode-guard must appear before any GetDaemonError() call inside init.
      const initIdx = raw.indexOf("async function init()")
      expect(initIdx, 'init() declaration must be present').toBeGreaterThan(0)
      const initSlice = raw.slice(initIdx, initIdx + 2000)
      expect(initSlice).toMatch(/if\s*\(\s*mode\s*===\s*['"]web['"]\s*\)\s*\{/)
    })

    it('gates retryInit with mode === "web" return', () => {
      const retryIdx = raw.indexOf('const retryInit')
      expect(retryIdx, 'retryInit declaration must be present').toBeGreaterThan(0)
      const retrySlice = raw.slice(retryIdx, retryIdx + 800)
      expect(retrySlice).toMatch(/if\s*\(\s*mode\s*===\s*['"]web['"]\s*\)\s*return/)
    })

    it('has at least two web-mode early-return guards across init + retry + polls', () => {
      const guardCount = (raw.match(/if\s*\(\s*mode\s*===\s*['"]web['"]\s*\)\s*(?:return|\{)/g) ?? []).length
      expect(guardCount, 'at least 2 web-mode guards required (init + retryInit, optionally polls)').toBeGreaterThanOrEqual(2)
    })
  })

  describe('file-browser tab gate uses mode-aware fbBaseURL + capToken', () => {
    it('uses window.location.origin for fbBaseURL when in web mode', () => {
      expect(raw).toMatch(/window\.location\.origin/)
    })

    it('keeps the 127.0.0.1:relayPort branch for desktop mode', () => {
      expect(raw).toMatch(/http:\/\/127\.0\.0\.1:\$\{relayPort/)
    })

    it('selects capToken from webParams in web mode', () => {
      expect(raw).toMatch(/webParams\.capToken/)
    })

    it('passes isRemote={isWeb} (or equivalent mode-aware boolean) to FileBrowserTab', () => {
      // The PLAN spells this as isRemote={isWeb}; allow either literal pattern.
      expect(raw).toMatch(/isRemote\s*=\s*\{(?:isWeb|mode\s*===\s*['"]web['"])\}/)
    })

    it('passes capToken to FileBrowserTab from the mode-aware selection', () => {
      expect(raw).toMatch(/capToken\s*=\s*\{fbCapToken\}/)
    })
  })

  describe('web-mode auto-open of the file-browser tab', () => {
    it('mentions handleOpenFileBrowser in conjunction with webParams.sessionId', () => {
      // The effect that opens the tab on first mount must reference both.
      const handleOpenIdx = raw.indexOf('handleOpenFileBrowser')
      expect(handleOpenIdx).toBeGreaterThan(0)
      // The web-mode auto-open must reference webParams.sessionId somewhere in
      // the body to drive the find-or-add.
      expect(raw).toMatch(/webParams\.sessionId/)
    })
  })

  describe('parity-comment for remote-on-desktop deferral', () => {
    it('documents that remote-on-desktop browse is a v3.5 follow-on', () => {
      expect(raw).toMatch(/remote-on-desktop|v3\.5 follow-on/)
    })
  })
})
