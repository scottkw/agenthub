// Phase 94 Wave 0 RED scaffold — Plan 94-03 will implement.
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md
//      ## Per-Task Verification Map (row 03-findbar wave 2)
// RESEARCH §"Theme-aware highlight via theme.selectionBackground"; UI-SPEC §"Animation Contract"
import { describe, it, expect } from 'vitest'

describe('FindBar — visual contract (SRC-04) — Plan 94-03 implementation', () => {
  it('Slide-in animation 200ms; selectionBackground used (no decorations option passed)', () => {
    expect.fail(
      'RED scaffold — Plan 94-03 verifies (a) CSS slide-in transition is 200ms ' +
        '(`grep -c "transition.*200ms" frontend/src/components/FindBar/style.css`) AND ' +
        '(b) SearchAddon is constructed without `decorations:` option ' +
        '(source-inspect TerminalPanel.tsx?raw — leaving decorations undefined makes the addon use ' +
        'xterm built-in selectionBackground; works automatically across all 138 themes). ' +
        'See 94-VALIDATION.md row 03-findbar wave 2.',
    )
  })
})
