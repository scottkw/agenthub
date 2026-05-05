// Phase 94 Wave 0 RED scaffold — Plan 94-04 will implement.
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md
//      ## Per-Task Verification Map (row 04-perf wave 3)
// RESEARCH §"Pitfall #10 — Cancel-on-close"; UI-SPEC §"Closing the Find Bar"
import { describe, it, expect } from 'vitest'

describe('FindBar — cancel-on-close (SRC-03) — Plan 94-04 implementation', () => {
  it('Closing find bar clears decorations + cancels pending debounce', () => {
    expect.fail(
      'RED scaffold — Plan 94-04 source-inspects TerminalPanel.tsx?raw and asserts ' +
        'clearTimeout(debounceTimerRef.current) AND searchAddonRef.current?.clearDecorations() ' +
        'in the handleSearchClose path. See 94-VALIDATION.md row 04-perf wave 3.',
    )
  })
})
