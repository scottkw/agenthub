/**
 * @vitest-environment jsdom
 *
 * Phase 120-04 Task 2 — FileBrowserTab singleton-tab-id contract.
 *
 * Pure helper test — pins the per-session find-or-add tab id format consumed
 * by App.tsx's handleOpenFileBrowser. The full find-or-add UI interaction is
 * exercised by Plan 05 e2e; here we lock the id-generation primitive in place
 * so a refactor cannot silently break the singleton invariant.
 */
import { describe, it, expect } from 'vitest'
import { fileBrowserTabId } from '../../FileBrowserTab'

describe('fileBrowserTabId', () => {
  it('produces __files__<sessionId>', () => {
    expect(fileBrowserTabId('abc')).toBe('__files__abc')
  })

  it('two calls with the same sessionId produce identical ids', () => {
    const a = fileBrowserTabId('sess-1')
    const b = fileBrowserTabId('sess-1')
    expect(a).toBe(b)
  })

  it('different sessionIds produce different ids', () => {
    expect(fileBrowserTabId('a')).not.toBe(fileBrowserTabId('b'))
  })

  it('id namespace does not collide with other App.tsx singleton tabs', () => {
    const id = fileBrowserTabId('any')
    expect(id.startsWith('__welcome__')).toBe(false)
    expect(id.startsWith('__settings__')).toBe(false)
    expect(id.startsWith('__daemon_manager__')).toBe(false)
    expect(id.startsWith('__remote_sessions__')).toBe(false)
    expect(id.startsWith('__files__')).toBe(true)
  })

  it('preserves the sessionId portion after the prefix', () => {
    const sid = 'b3d4-7f2c-2026'
    const id = fileBrowserTabId(sid)
    expect(id.slice('__files__'.length)).toBe(sid)
  })
})
