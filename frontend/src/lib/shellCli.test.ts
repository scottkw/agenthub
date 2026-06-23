/**
 * isShellCli — shell-session detection for the web-share warning gate
 * (Phase 150 SET-01 gap-closure, found in live UAT).
 *
 * Session `cli` may be a bare command ('zsh') OR a full path ('/bin/zsh',
 * 'C:\\...\\powershell.exe'), because shell sessions store the resolved shell
 * binary path. The original gate matched a bare-name Set directly, so real
 * shell sessions (always full paths) never triggered the banner.
 */
import { describe, it, expect } from 'vitest'
import { isShellCli } from './shellCli'

describe('isShellCli', () => {
  it('matches bare shell names', () => {
    expect(isShellCli('zsh')).toBe(true)
    expect(isShellCli('bash')).toBe(true)
    expect(isShellCli('shell')).toBe(true)
    expect(isShellCli('pwsh')).toBe(true)
    expect(isShellCli('powershell')).toBe(true)
  })

  it('matches full POSIX shell paths (the real-app case)', () => {
    expect(isShellCli('/bin/zsh')).toBe(true)
    expect(isShellCli('/bin/bash')).toBe(true)
    expect(isShellCli('/usr/local/bin/bash')).toBe(true)
  })

  it('matches Windows paths and strips .exe', () => {
    expect(isShellCli('C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe')).toBe(true)
    expect(isShellCli('pwsh.exe')).toBe(true)
  })

  it('is case-insensitive on the basename', () => {
    expect(isShellCli('/bin/ZSH')).toBe(true)
  })

  it('does NOT match non-shell agent CLIs', () => {
    expect(isShellCli('claude')).toBe(false)
    expect(isShellCli('codex')).toBe(false)
    expect(isShellCli('/usr/bin/claude')).toBe(false)
  })

  it('does NOT match shells outside the warning set', () => {
    expect(isShellCli('/bin/sh')).toBe(false)
    expect(isShellCli('fish')).toBe(false)
  })

  it('handles empty/undefined safely', () => {
    expect(isShellCli('')).toBe(false)
    expect(isShellCli(undefined)).toBe(false)
    expect(isShellCli(null)).toBe(false)
  })
})
