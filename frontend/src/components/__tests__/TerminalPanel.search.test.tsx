/**
 * Phase 94 SRC-01 / SRC-02 / SRC-04 — TerminalPanel SearchAddon integration.
 *
 * Source-inspection invariants:
 *   - imports SearchAddon and isXtermFocused
 *   - declares searchAddonRef + onDidChangeResults disposable + debounce timer refs
 *   - hot-swap useEffect dep array includes pluginConfig?.search
 *   - window keydown listener uses isXtermFocused() for T-94-03 mitigation
 *   - findNext / findPrevious are NEVER called with `decorations:` option
 *     (SRC-04 theme.selectionBackground invariant — leaves the addon to use
 *     xterm theme.selectionBackground automatically across all 138 themes)
 *   - FindBar is rendered conditionally on findBarOpen && pluginConfig?.search
 *   - handleSearchClose clears decorations + cancels debounce
 *   - 100ms setTimeout debounce wired
 *
 * The runtime path (mounting xterm, firing key events) requires a real
 * <canvas> + WebGL context which jsdom does not provide; source-inspection
 * is the deterministic gate. Runtime verification lives in 94-VERIFICATION.md.
 */
import { describe, it, expect } from 'vitest'
import src from '../TerminalPanel.tsx?raw'

describe('Phase 94 SRC-01..04: TerminalPanel SearchAddon + FindBar integration', () => {
  it('imports SearchAddon from @xterm/addon-search', () => {
    expect(src).toMatch(/import\s+\{\s*SearchAddon\s*\}\s+from\s+['"]@xterm\/addon-search['"]/)
  })

  it('imports isXtermFocused from ../lib/isXtermFocused', () => {
    expect(src).toMatch(/import\s+\{\s*isXtermFocused\s*\}\s+from\s+['"]\.\.\/lib\/isXtermFocused['"]/)
  })

  it('imports FindBar from ./FindBar/FindBar', () => {
    expect(src).toMatch(/from\s+['"]\.\/FindBar\/FindBar['"]/)
  })

  it('imports SetPluginSettings from the wails App binding', () => {
    expect(src).toMatch(/import\s+\{\s*SetPluginSettings\s*\}\s+from\s+['"]\.\.\/wailsjs\/go\/main\/App['"]/)
  })

  it('declares searchAddonRef (≥4 references — declaration, attach, detach, cleanup)', () => {
    const matches = src.match(/searchAddonRef/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(4)
  })

  it('declares searchResultsDisposableRef for the onDidChangeResults subscription', () => {
    expect(src).toContain('searchResultsDisposableRef')
  })

  it('declares debounceTimerRef (Pattern 4 — 100ms input debounce)', () => {
    expect(src).toContain('debounceTimerRef')
  })

  it('subscribes to onDidChangeResults when SearchAddon attaches', () => {
    expect(src).toContain('onDidChangeResults(')
  })

  it('hot-swap useEffect dep array includes pluginConfig?.search', () => {
    // Must include all three of webgl, clipboard, search keys (Pitfall #1).
    expect(src).toMatch(/\[\s*pluginConfig\?\.webgl\s*,\s*pluginConfig\?\.clipboard\s*,\s*pluginConfig\?\.search/)
  })

  it('window keydown listener attached with addEventListener', () => {
    expect(src).toContain("window.addEventListener('keydown'")
  })

  it('Cmd-F handler uses isXtermFocused(containerRef.current) gate (T-94-03 mitigation)', () => {
    expect(src).toContain('isXtermFocused(containerRef.current)')
  })

  it('Cmd-F handler distinguishes Mac (metaKey) vs other (ctrlKey)', () => {
    expect(src).toContain('e.metaKey')
    expect(src).toContain('e.ctrlKey')
  })

  it('Cmd-F handler calls preventDefault before opening find bar', () => {
    expect(src).toContain('e.preventDefault()')
  })

  it('does NOT customize SearchAddon decoration colors (SRC-04 theme.selectionBackground invariant)', () => {
    // Plan 94-05 reconciliation: SearchAddon._fireResults gates the
    // onDidChangeResults event on !!opts.decorations — the empty-object form
    // (`decorations: {}`) is required so the match-count callback fires
    // (SRC-02). The SRC-04 invariant FORBIDS per-theme color overrides; with
    // none set, xterm core's selection (theme.selectionBackground) owns the
    // active-match highlight across all 138 themes. See FindBar.themeMatrix.
    for (const key of [
      'matchBackground',
      'activeMatchBackground',
      'matchBorder',
      'activeMatchBorder',
      'matchOverviewRuler',
      'activeMatchColorOverviewRuler',
    ]) {
      expect(src).not.toContain(key)
    }
    // The empty-decorations form IS expected — it's what makes onDidChangeResults fire.
    expect(src).toMatch(/decorations:\s*\{\s*\}/)
  })

  it('renders <FindBar> conditionally on (findBarOpen || findBarExiting) && pluginConfig?.search', () => {
    // Phase 94 WR-01 / SC-4 widened the guard so the bar stays in the DOM
    // during the 200ms exit transition (TerminalPanel.search.exit.test.tsx
    // asserts the timing). Match the new compound guard across whitespace.
    expect(src).toMatch(
      /\(\s*findBarOpen\s*\|\|\s*findBarExiting\s*\)\s*&&\s*pluginConfig\?\.search\s*&&\s*\(?\s*<FindBar/,
    )
  })

  it('calls SetPluginSettings on toggle change (persistence — Pattern 3)', () => {
    expect(src).toContain('SetPluginSettings(')
  })

  it('100ms setTimeout debounce is wired in handleSearchQueryChange', () => {
    expect(src).toMatch(/setTimeout\([\s\S]+?,\s*100\s*\)/)
  })

  it('handleSearchClose clears decorations AND cancels the debounce timer', () => {
    expect(src).toContain('clearDecorations()')
    expect(src).toContain('clearTimeout(debounceTimerRef.current)')
  })

  it('handleSearchClose returns focus to the xterm helper textarea', () => {
    expect(src).toContain('.xterm-helper-textarea')
  })

  it('onDidChangeResults subscription is disposed on hot-swap-off and unmount', () => {
    // Both call sites: hot-swap-off arm + mount cleanup. ≥2 occurrences of
    // searchResultsDisposableRef.current.dispose().
    const matches = src.match(/searchResultsDisposableRef\.current\.dispose\(\)/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(2)
  })

  it('SearchAddon disposed on hot-swap-off and unmount', () => {
    const matches = src.match(/searchAddonRef\.current\.dispose\(\)/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(2)
  })

  it('SearchAddon constructor is called WITHOUT options (default highlightLimit=1000)', () => {
    // RESEARCH §"SearchAddon API Contract" — accept default highlightLimit=1000.
    expect(src).toMatch(/new SearchAddon\(\)/)
  })
})
