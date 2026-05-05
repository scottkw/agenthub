// Phase 94 SRC-04 / SC-4 — theme.selectionBackground invariant across both
// surfaces (desktop FindBar.tsx + TerminalPanel.tsx, web web/assets/terminal.js).
//
// The invariant: NEITHER surface customizes SearchAddon decoration COLORS
// (matchBackground / activeMatchBackground / matchBorder / activeMatchBorder /
// matchOverviewRuler / activeMatchColorOverviewRuler). When those are unset,
// xterm core uses the active theme's `selectionBackground` to highlight the
// active match. Phase 65/71 established the 138-theme invariant that
// `selectionBackground` is always set, so this single source-inspection gate
// covers all 138 themes.
//
// IMPORTANT (Plan 94-05 reconciliation): both surfaces DO pass `decorations: {}`
// (empty object) to findNext/findPrevious. SearchAddon._fireResults gates the
// onDidChangeResults event on `!!opts.decorations` — without a truthy value,
// the match-count callback never fires (SRC-02 broken). Empty `decorations: {}`
// makes the event fire while leaving every per-theme color undefined, so xterm
// core's selection (theme.selectionBackground) still owns the active-match
// highlight. See `web/assets/terminal.js::searchOpts()` and the four call
// sites in `TerminalPanel.tsx` that thread `{...searchOptions, decorations: {}}`
// through findNext/findPrevious.
//
// References:
// - 94-RESEARCH.md ## SearchAddon API Contract
// - 94-UI-SPEC.md ## Color "Match highlight" row
// - ROADMAP Phase 94 SC-4
// - 94-05-SUMMARY.md ## Deviations (decorations reconciliation)

import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import terminalPanelSrc from '../../TerminalPanel.tsx?raw'
import findBarSrc from '../FindBar.tsx?raw'

// web/assets/terminal.js lives outside the frontend/ vite root, so `?raw`
// won't resolve it. Read it via Node fs at test time (Vitest runs in Node).
const webJsPath = resolve(__dirname, '../../../../../web/assets/terminal.js')
const webJsSrc = readFileSync(webJsPath, 'utf8')

const FORBIDDEN_DECORATION_COLORS = [
  'matchBackground',
  'activeMatchBackground',
  'matchBorder',
  'activeMatchBorder',
  'matchOverviewRuler',
  'activeMatchColorOverviewRuler',
] as const

describe('Phase 94 SRC-04 — theme.selectionBackground invariant across surfaces', () => {
  it('TerminalPanel.tsx does NOT customize SearchAddon decoration colors (138-theme contract)', () => {
    for (const key of FORBIDDEN_DECORATION_COLORS) {
      expect(terminalPanelSrc).not.toContain(key)
    }
  })

  it('FindBar.tsx is a controlled component — never references decoration colors', () => {
    for (const key of FORBIDDEN_DECORATION_COLORS) {
      expect(findBarSrc).not.toContain(key)
    }
  })

  it('web/assets/terminal.js does NOT customize SearchAddon decoration colors', () => {
    for (const key of FORBIDDEN_DECORATION_COLORS) {
      expect(webJsSrc).not.toContain(key)
    }
    // Smoke check that this assertion is meaningful — terminal.js MUST still
    // construct SearchAddon. Without this, the regex above would pass trivially
    // if SearchAddon were removed entirely.
    expect(webJsSrc).toMatch(/new SearchAddon\.SearchAddon\(/)
  })

  it('both surfaces DO pass `decorations: {}` so onDidChangeResults fires (SRC-02 reconciliation)', () => {
    // SearchAddon._fireResults gates fireResultsChanged on !!opts.decorations.
    // Without a truthy decorations field, the match-count callback never fires
    // (SRC-02 break). The empty-object form is the surgical fix that preserves
    // SRC-04 (no color overrides → theme.selectionBackground wins for active
    // match) while restoring SRC-02 (count event fires).
    expect(terminalPanelSrc).toMatch(/decorations:\s*\{\s*\}/)
    expect(webJsSrc).toMatch(/decorations:\s*\{\s*\}/)
  })

  it('both surfaces subscribe to onDidChangeResults (SRC-02 match count wiring)', () => {
    expect(terminalPanelSrc).toMatch(/onDidChangeResults/)
    expect(webJsSrc).toMatch(/onDidChangeResults/)
  })
})
