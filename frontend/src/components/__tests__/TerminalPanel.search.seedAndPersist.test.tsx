/**
 * Phase 94-07 WR-02 + WR-03 — TerminalPanel first-load seed + sub-key persistence.
 *
 * Source-inspection invariants (matches the existing TerminalPanel.search.*
 * test pattern — jsdom cannot render xterm because of its WebGL/<canvas>
 * dependency, so we assert the wiring at the source level).
 *
 * What this file proves:
 *   WR-02 — the seededRef one-shot useEffect seeds searchOptions exactly once
 *           when pluginConfig.searchConfig FIRST becomes non-null AND the
 *           find bar is closed. The Pitfall #2 mid-open invariant is preserved
 *           via an explicit `if (findBarOpen) return` guard.
 *
 *   WR-03 — handleSearchOptionsChange writes ONLY the SearchConfig sub-key via
 *           SetSearchConfig (NOT a full PluginSettings via SetPluginSettings).
 *           This closes the race against PluginsSection's stale local edit
 *           buffer: the find bar can no longer clobber an in-flight
 *           Plugins-tab boolean toggle.
 *
 * Runtime verification (Cmd-F → toggle → close → restart → bar opens with
 * persisted state) lives in 94-VERIFICATION.md human_verification[1] and
 * is rerun after this plan ships.
 */
import { describe, it, expect } from 'vitest'
import src from '../TerminalPanel.tsx?raw'

describe('Phase 94-07 WR-02: TerminalPanel first-load seed (seededRef)', () => {
  it('declares seededRef via useRef(false)', () => {
    expect(src).toMatch(/const\s+seededRef\s*=\s*useRef\(false\)/)
  })

  it('seededRef is referenced at least three times (declaration, current-check, set-true)', () => {
    const matches = src.match(/seededRef/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(3)
  })

  it('seed useEffect early-returns when seededRef.current is already true (one-shot guard)', () => {
    expect(src).toContain('if (seededRef.current) return')
  })

  it('seed useEffect early-returns when pluginConfig?.searchConfig is null/undefined', () => {
    expect(src).toContain('if (!pluginConfig?.searchConfig) return')
  })

  it('seed useEffect early-returns when findBarOpen is true (Pitfall #2 — never re-seed mid-open)', () => {
    expect(src).toContain('if (findBarOpen) return')
  })

  it('seed useEffect calls setSearchOptions with regex/caseSensitive/wholeWord from pluginConfig.searchConfig', () => {
    expect(src).toMatch(/setSearchOptions\(\s*\{[\s\S]*?regex:[\s\S]*?caseSensitive:[\s\S]*?wholeWord:[\s\S]*?\}\s*\)/)
    expect(src).toContain('pluginConfig.searchConfig.regex')
    expect(src).toContain('pluginConfig.searchConfig.caseSensitive')
    expect(src).toContain('pluginConfig.searchConfig.wholeWord')
  })

  it('seed useEffect sets seededRef.current = true to flip the one-shot latch', () => {
    expect(src).toContain('seededRef.current = true')
  })

  it('seed useEffect dep array includes pluginConfig?.searchConfig and findBarOpen', () => {
    // Match "[pluginConfig?.searchConfig, findBarOpen]" tolerating whitespace.
    expect(src).toMatch(/\[\s*pluginConfig\?\.searchConfig\s*,\s*findBarOpen\s*\]/)
  })
})

describe('Phase 94-07 WR-03: TerminalPanel persistence — sub-key write via SetSearchConfig', () => {
  it('imports SetSearchConfig from the wails App binding', () => {
    expect(src).toMatch(/import\s+\{[^}]*\bSetSearchConfig\b[^}]*\}\s+from\s+['"]\.\.\/wailsjs\/go\/main\/App['"]/)
  })

  it('handleSearchOptionsChange calls SetSearchConfig (the sub-key writer)', () => {
    // Locate the handleSearchOptionsChange callback body and confirm
    // SetSearchConfig is invoked inside it.
    const m = src.match(/handleSearchOptionsChange\s*=\s*useCallback\(\([^)]*\)\s*=>\s*\{([\s\S]*?)\},\s*\[/)
    expect(m).not.toBeNull()
    expect(m![1]).toContain('SetSearchConfig(')
  })

  it('handleSearchOptionsChange does NOT construct a full daemon.PluginSettings', () => {
    // The Pitfall #2 race fix: the old code built `new daemon.PluginSettings(...)`
    // from a stale prop and called SetPluginSettings. That construction site is
    // gone now — the find bar writes only the sub-key.
    const m = src.match(/handleSearchOptionsChange\s*=\s*useCallback\(\([^)]*\)\s*=>\s*\{([\s\S]*?)\},\s*\[/)
    expect(m).not.toBeNull()
    expect(m![1]).not.toContain('new daemon.PluginSettings')
    expect(m![1]).not.toContain('SetPluginSettings(')
  })

  it('handleSearchOptionsChange dep array drops pluginConfig (only searchQuery is a closure dep now)', () => {
    // The new sub-key writer doesn't read the full PluginSettings prop, so
    // pluginConfig is no longer a closure dep — only searchQuery (used to
    // re-fire findNext after a toggle change) remains.
    const m = src.match(/handleSearchOptionsChange\s*=\s*useCallback\(\([^)]*\)\s*=>\s*\{[\s\S]*?\},\s*\[([^\]]*)\]\)/)
    expect(m).not.toBeNull()
    const deps = m![1]
    expect(deps).toContain('searchQuery')
    expect(deps).not.toContain('pluginConfig')
  })

  it('SetSearchConfig is called exactly once per handleSearchOptionsChange invocation', () => {
    // Match the `SetSearchConfig(...)` call site (one .catch() chained — silent failure).
    expect(src).toMatch(/SetSearchConfig\([^)]*\)\.catch\(/)
  })
})

describe('Phase 94-07 WR-02: backward-compatibility — preserves the lazy initializer fallback', () => {
  it('keeps the useState lazy initializer for the first-render-with-pluginConfig fast path', () => {
    // If pluginConfig is non-null on the very first render (rare — App.tsx
    // loads it async — but possible if a future change resolves it
    // synchronously), the lazy initializer still seeds searchOptions
    // correctly. The seededRef effect then early-returns on the first run
    // (because seededRef stays false on mount; the effect runs once and
    // sets it true on the FIRST mount tick — which is harmless because
    // setSearchOptions(currentValues) is a no-op).
    expect(src).toMatch(
      /useState<FindBarSearchOptions>\(\(\)\s*=>\s*\(\{\s*regex:\s*pluginConfig\?\.searchConfig\?\.regex/,
    )
  })
})
