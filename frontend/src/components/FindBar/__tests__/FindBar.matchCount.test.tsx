// Phase 94 Wave 0 RED scaffold — Plan 94-03 will implement.
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md
//      ## Per-Task Verification Map (row 03-findbar wave 2)
// RESEARCH §"Pattern 1 — onDidChangeResults match-count subscription"; UI-SPEC §"Match Count"
import { describe, it, expect } from 'vitest'

describe('FindBar — match count (SRC-02) — Plan 94-03 implementation', () => {
  it('onDidChangeResults updates "{i} of {n}" label', () => {
    expect.fail(
      'RED scaffold — Plan 94-03 wires SearchAddon.onDidChangeResults({resultIndex, resultCount}) ' +
        'to the FindBar count label. See 94-VALIDATION.md row 03-findbar wave 2.',
    )
  })

  it('renders "0 of 0" + --no-results modifier when query non-empty AND count zero', () => {
    expect.fail(
      'RED scaffold — Plan 94-03 implements zero-results visual treatment via ' +
        '.find-bar__count--no-results modifier. UI-SPEC §"Match Count > Zero Results".',
    )
  })
})
