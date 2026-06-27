// frontend/src/components/__tests__/WelcomeTab.install.test.tsx
// Phase 156 — INSTALL-01 / INSTALL-02 source-gate test.
// Pattern: reads WelcomeTab.tsx as a raw string (no render), asserts exact install strings.
// Consistent with style.hub.test.ts, style.redesign.test.ts source-gate pattern.
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, it, expect } from 'vitest'

const SOURCE = readFileSync(
  join(__dirname, '../../components/WelcomeTab.tsx'),
  'utf-8'
)

describe('WelcomeTab — install string source gates (INSTALL-01, INSTALL-02)', () => {
  it('INSTALL-01: Linux curl URL points to raw.githubusercontent.com install.sh', () => {
    expect(SOURCE).toContain(
      'https://raw.githubusercontent.com/scottkw/agenthub/main/scripts/install.sh'
    )
  })

  it('INSTALL-01: Linux curl URL does NOT contain the broken agenthub.dev domain', () => {
    expect(SOURCE).not.toContain('agenthub.dev')
  })

  it('INSTALL-02: winget command uses correct package id scottkw.agenthub', () => {
    expect(SOURCE).toContain('winget install scottkw.agenthub')
  })

  it('INSTALL-02: winget command does NOT use bare agenthub id', () => {
    // "winget install agenthub" without the scottkw. prefix must be absent
    expect(SOURCE).not.toMatch(/winget install agenthub(?!\.)/u)
  })

  it('INSTALL-02: repo link points to scottkw/agenthub', () => {
    expect(SOURCE).toContain('github.com/scottkw/agenthub')
  })

  it('INSTALL-02: repo link does NOT contain the wrong agenthub-dev org', () => {
    expect(SOURCE).not.toContain('agenthub-dev/agenthub')
  })
})
