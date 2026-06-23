import { describe, it, expect } from 'vitest'
import { agentBadgeModifier } from './agentBadge'

describe('agentBadgeModifier', () => {
  it("returns 'claude' for cli='claude'", () => {
    expect(agentBadgeModifier('claude')).toBe('claude')
  })

  it("returns 'opencode' for cli='opencode'", () => {
    expect(agentBadgeModifier('opencode')).toBe('opencode')
  })

  it("returns 'codex' for cli='codex'", () => {
    expect(agentBadgeModifier('codex')).toBe('codex')
  })

  it("returns 'gemini' for cli='gemini'", () => {
    expect(agentBadgeModifier('gemini')).toBe('gemini')
  })

  it("returns 'cursor' for cli='cursor'", () => {
    expect(agentBadgeModifier('cursor')).toBe('cursor')
  })

  it("returns 'aider' for cli='aider'", () => {
    expect(agentBadgeModifier('aider')).toBe('aider')
  })

  it("returns 'agy' for cli='agy'", () => {
    expect(agentBadgeModifier('agy')).toBe('agy')
  })

  it("returns 'shell' for cli='shell'", () => {
    expect(agentBadgeModifier('shell')).toBe('shell')
  })

  it("returns 'shell' for cli='bash' (shell variant)", () => {
    expect(agentBadgeModifier('bash')).toBe('shell')
  })

  it("returns 'shell' for cli='zsh' (shell variant)", () => {
    expect(agentBadgeModifier('zsh')).toBe('shell')
  })

  it("returns 'shell' for cli='pwsh' (shell variant)", () => {
    expect(agentBadgeModifier('pwsh')).toBe('shell')
  })

  it("returns 'shell' for cli='powershell' (shell variant)", () => {
    expect(agentBadgeModifier('powershell')).toBe('shell')
  })

  it('returns null for an unknown cli string (fallback)', () => {
    expect(agentBadgeModifier('totally-unknown')).toBeNull()
  })

  it('returns null for an empty string', () => {
    expect(agentBadgeModifier('')).toBeNull()
  })
})
