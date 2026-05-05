// Phase 94 Wave 0 RED scaffold — Plan 94-03 will implement.
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md
//      ## Per-Task Verification Map (row 03-findbar wave 2)
// RESEARCH §"Persistence — daemon SearchConfig round-trip"; UI-SPEC §"Persisting Per-Flag Defaults"
import { describe, it, expect } from 'vitest'

describe('FindBar — persistence (SRC-02) — Plan 94-03 implementation', () => {
  it('regex/case/wholeWord toggles round-trip to daemon SearchConfig via SetPluginSettings', () => {
    expect.fail(
      'RED scaffold — Plan 94-03 wires toggle changes through SetPluginSettings ' +
        '(adds searchConfig{regex,caseSensitive,wholeWord} to PluginSettings). ' +
        'See 94-VALIDATION.md row 03-findbar wave 2; depends on Plan 94-02 SearchConfig daemon struct.',
    )
  })
})
