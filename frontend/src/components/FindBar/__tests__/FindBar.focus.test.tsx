// Phase 94 Wave 0 RED scaffold — Plan 94-03 will implement.
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md
//      ## Per-Task Verification Map (row 03-findbar wave 2)
// RESEARCH §"Pattern 2 — Focus-conditioned Cmd-F"; UI-SPEC §"Opening the Find Bar"
import { describe, it, expect } from 'vitest'

describe('FindBar — focus conditioning (SRC-01) — Plan 94-03 implementation', () => {
  it('Cmd-F preventDefault only when xtermRef.contains(document.activeElement)', () => {
    expect.fail(
      'RED scaffold — Plan 94-03 implements FindBar.tsx + TerminalPanel keydown handler. ' +
        'See 94-VALIDATION.md row 03-findbar wave 2; RESEARCH §"Pattern 2 — Focus-conditioned Cmd-F".',
    )
  })
})
