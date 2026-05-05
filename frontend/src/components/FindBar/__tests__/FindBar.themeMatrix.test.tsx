// Phase 94 Wave 0 RED scaffold — Plan 94-05 will implement.
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md
//      ## Per-Task Verification Map (row 05-web wave 4)
// RESEARCH §"Theme matrix"; UI-SPEC §"Theme-Aware Match Highlight"
import { describe, it, expect } from 'vitest'

describe('FindBar — theme matrix (SRC-04 138-theme invariant) — Plan 94-05 implementation', () => {
  it('Highlight contrast valid on TokyoNight + light theme snapshot', () => {
    expect.fail(
      'RED scaffold — Plan 94-05 verifies SearchAddon construction never passes `decorations:` ' +
        'on either desktop (TerminalPanel.tsx) or web (web/assets/terminal.js) — leaving it ' +
        'undefined keeps the 138-theme invariant via xterm theme.selectionBackground. ' +
        'See 94-VALIDATION.md row 05-web wave 4.',
    )
  })
})
