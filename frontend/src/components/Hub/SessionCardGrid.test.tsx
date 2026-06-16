import { describe, it, expect, vi, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// Mock Wails RPC before component import (required by SessionCard → InlineSessionName)
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  ListSessions: vi.fn().mockResolvedValue([]),
}))

vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
}))

import { SessionCardGrid, groupByNamedGroups, groupByWorkDir } from './SessionCardGrid'
import type { HubGroupDef } from '../../lib/hubGroups'

// ---- Helpers ----

function makeSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'sess-1',
    cli: 'claude',
    name: 'Test Session',
    state: 'running',
    status: 'running',
    createdAt: new Date().toISOString(),
    hostname: '',
    webEnabled: false,
    viewerCount: 0,
    homeDir: false,
    filesWrite: false,
    workDir: '/home/user/project',
    ...overrides,
  }
}

function makeGroupDefs(): HubGroupDef[] {
  return [
    {
      id: 'group-alpha',
      name: 'Alpha Team',
      memberKeys: ['SessionAlpha:::/home/user/alpha'],
    },
    {
      id: 'group-beta',
      name: 'Beta Team',
      memberKeys: ['SessionBeta:::/home/user/beta'],
    },
  ]
}

function renderGrid(
  sessions: SessionInfo[],
  onRename: (id: string, name: string) => void = vi.fn(),
  extra: { groupDefs?: HubGroupDef[]; previewTails?: Map<string, string[]>; onAssignGroup?: (mk: string, gid: string) => void } = {},
) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(
      <SessionCardGrid
        sessions={sessions}
        onRename={onRename}
        groupDefs={extra.groupDefs}
        previewTails={extra.previewTails}
        onAssignGroup={extra.onAssignGroup}
      />
    )
  })
  return { container, root }
}

describe('SessionCardGrid', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  // ---- Grouping: two distinct workDirs produce two groups ----

  it('renders two groups for two distinct workDirs', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/alpha' }),
      makeSession({ id: 'sess-2', workDir: '/home/user/beta' }),
    ]
    const { container } = renderGrid(sessions)
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(2)
  })

  it('renders sessions with the same workDir in a single group', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/alpha', name: 'Session A' }),
      makeSession({ id: 'sess-2', workDir: '/home/user/alpha', name: 'Session B' }),
    ]
    const { container } = renderGrid(sessions)
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(1)
    // Both listitems are in the single group
    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(2)
  })

  // ---- "Other" group for empty workDir ----

  it('groups sessions with empty workDir under an "Other" header', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '' }),
    ]
    const { container } = renderGrid(sessions)
    const headings = container.querySelectorAll('.hub__group-header')
    expect(headings.length).toBe(1)
    expect(headings[0].textContent).toContain('Other')
  })

  it('puts empty-workDir sessions in "Other" alongside named-workDir sessions', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/project' }),
      makeSession({ id: 'sess-2', workDir: '' }),
    ]
    const { container } = renderGrid(sessions)
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(2)
    const headingTexts = Array.from(container.querySelectorAll('.hub__group-header')).map(
      (h) => h.textContent,
    )
    expect(headingTexts.some((t) => t?.includes('Other'))).toBe(true)
  })

  // ---- Group header shows basename ----

  it('shows the basename of workDir in the group header', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/my-project' }),
    ]
    const { container } = renderGrid(sessions)
    const header = container.querySelector('.hub__group-header')
    expect(header).not.toBeNull()
    expect(header!.textContent).toContain('my-project')
  })

  it('uses the full workDir as the title tooltip on the group header span', () => {
    const workDir = '/home/user/my-project'
    const sessions = [makeSession({ id: 'sess-1', workDir })]
    const { container } = renderGrid(sessions)
    // The span inside the header should have the title attr set to the full path
    const titleEl = container.querySelector('.hub__group-header span[title]')
    expect(titleEl).not.toBeNull()
    expect(titleEl!.getAttribute('title')).toBe(workDir)
  })

  it('shows basename for nested paths (only last segment)', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/a/b/c/d/my-deep-project' }),
    ]
    const { container } = renderGrid(sessions)
    const header = container.querySelector('.hub__group-header')
    expect(header!.textContent).toContain('my-deep-project')
    expect(header!.textContent).not.toContain('/a/b/c/d/')
  })

  // ---- Per-session listitem structure ----

  it('renders one listitem per session', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/alpha' }),
      makeSession({ id: 'sess-2', workDir: '/home/user/alpha' }),
      makeSession({ id: 'sess-3', workDir: '/home/user/beta' }),
    ]
    const { container } = renderGrid(sessions)
    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(3)
  })

  it('each group card-row has role="list"', () => {
    const sessions = [makeSession({ id: 'sess-1', workDir: '/home/user/project' })]
    const { container } = renderGrid(sessions)
    const lists = container.querySelectorAll('[role="list"]')
    expect(lists.length).toBeGreaterThanOrEqual(1)
  })

  // ---- Group headers have correct heading semantics ----

  it('group headers use role="heading" aria-level=2 or h2 element', () => {
    const sessions = [makeSession({ id: 'sess-1', workDir: '/home/user/project' })]
    const { container } = renderGrid(sessions)
    // Accept either <h2> or element with role="heading" aria-level="2"
    const h2s = container.querySelectorAll('h2')
    const roleHeading2 = container.querySelectorAll('[role="heading"][aria-level="2"]')
    expect(h2s.length + roleHeading2.length).toBeGreaterThanOrEqual(1)
  })

  // ---- Empty sessions ----

  it('renders nothing (no groups) when sessions is empty', () => {
    const { container } = renderGrid([])
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(0)
  })

  // ---- Phase 132: groupByWorkDir fallback when groupDefs is empty/undefined ----

  it('falls back to groupByWorkDir when groupDefs is undefined', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/alpha' }),
      makeSession({ id: 'sess-2', workDir: '/home/user/beta' }),
    ]
    const { container } = renderGrid(sessions, vi.fn(), { groupDefs: undefined })
    // Should still show 2 groups (workDir-based)
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(2)
  })

  it('falls back to groupByWorkDir when groupDefs is an empty array', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/alpha', name: 'Alpha' }),
      makeSession({ id: 'sess-2', workDir: '/home/user/beta', name: 'Beta' }),
    ]
    const { container } = renderGrid(sessions, vi.fn(), { groupDefs: [] })
    // Empty groupDefs → workDir grouping → 2 groups
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(2)
    // Headers should be the basenames, not named group labels
    const headingTexts = Array.from(container.querySelectorAll('.hub__group-header')).map(
      (h) => h.textContent
    )
    expect(headingTexts.some((t) => t?.includes('alpha'))).toBe(true)
    expect(headingTexts.some((t) => t?.includes('beta'))).toBe(true)
  })

  // ---- Phase 132: named-group grouping (groupByNamedGroups) ----

  it('uses named group labels when groupDefs is non-empty', () => {
    const sessions = [
      makeSession({ id: 'sess-1', name: 'SessionAlpha', workDir: '/home/user/alpha' }),
      makeSession({ id: 'sess-2', name: 'SessionBeta', workDir: '/home/user/beta' }),
    ]
    const { container } = renderGrid(sessions, vi.fn(), { groupDefs: makeGroupDefs() })
    const headers = Array.from(container.querySelectorAll('.hub__group-header')).map(
      (h) => h.textContent
    )
    // Should show "Alpha Team" and "Beta Team" (definition-order group labels)
    expect(headers.some((t) => t?.includes('Alpha Team'))).toBe(true)
    expect(headers.some((t) => t?.includes('Beta Team'))).toBe(true)
  })

  it('renders named groups in definition order', () => {
    const sessions = [
      makeSession({ id: 'sess-1', name: 'SessionBeta', workDir: '/home/user/beta' }),
      makeSession({ id: 'sess-2', name: 'SessionAlpha', workDir: '/home/user/alpha' }),
    ]
    const { container } = renderGrid(sessions, vi.fn(), { groupDefs: makeGroupDefs() })
    const headers = Array.from(container.querySelectorAll('.hub__group-header')).map(
      (h) => h.textContent?.trim()
    )
    // Alpha Team is first in makeGroupDefs() → should appear before Beta Team
    const alphaIdx = headers.findIndex((t) => t?.includes('Alpha Team'))
    const betaIdx = headers.findIndex((t) => t?.includes('Beta Team'))
    expect(alphaIdx).not.toBe(-1)
    expect(betaIdx).not.toBe(-1)
    expect(alphaIdx).toBeLessThan(betaIdx)
  })

  it('session matching a named group renders under that group', () => {
    const sessions = [
      makeSession({ id: 'sess-1', name: 'SessionAlpha', workDir: '/home/user/alpha' }),
    ]
    const { container } = renderGrid(sessions, vi.fn(), { groupDefs: makeGroupDefs() })
    const headers = Array.from(container.querySelectorAll('.hub__group-header')).map(
      (h) => h.textContent
    )
    expect(headers.some((t) => t?.includes('Alpha Team'))).toBe(true)
    // Should NOT appear as workDir basename
    expect(headers.some((t) => t?.includes('alpha') && !t?.includes('Alpha Team'))).toBe(false)
  })

  it('unmatched session renders under "Other" group (GROUP-04)', () => {
    const sessions = [
      makeSession({ id: 'sess-unmatched', name: 'Unmatched', workDir: '/some/other/path' }),
    ]
    const { container } = renderGrid(sessions, vi.fn(), { groupDefs: makeGroupDefs() })
    const headers = Array.from(container.querySelectorAll('.hub__group-header')).map(
      (h) => h.textContent
    )
    expect(headers.some((t) => t?.includes('Other'))).toBe(true)
  })

  it('"Other" group appears last after named groups (GROUP-04)', () => {
    const sessions = [
      makeSession({ id: 'sess-1', name: 'SessionAlpha', workDir: '/home/user/alpha' }),
      makeSession({ id: 'sess-2', name: 'Unmatched', workDir: '/some/other/path' }),
    ]
    const { container } = renderGrid(sessions, vi.fn(), { groupDefs: makeGroupDefs() })
    const headers = Array.from(container.querySelectorAll('.hub__group-header')).map(
      (h) => h.textContent?.trim()
    )
    const alphaIdx = headers.findIndex((t) => t?.includes('Alpha Team'))
    const otherIdx = headers.findIndex((t) => t?.includes('Other'))
    expect(alphaIdx).not.toBe(-1)
    expect(otherIdx).not.toBe(-1)
    expect(otherIdx).toBeGreaterThan(alphaIdx)
  })

  it('session in a named group does NOT appear in Other', () => {
    const sessions = [
      makeSession({ id: 'sess-1', name: 'SessionAlpha', workDir: '/home/user/alpha' }),
    ]
    const { container } = renderGrid(sessions, vi.fn(), { groupDefs: makeGroupDefs() })
    // Find the "Other" group element
    const allGroups = Array.from(container.querySelectorAll('.hub__group'))
    const otherGroup = allGroups.find((g) => g.querySelector('.hub__group-header')?.textContent?.includes('Other'))
    if (otherGroup) {
      // Other group must have 0 listitems
      const listitems = otherGroup.querySelectorAll('[role="listitem"]')
      expect(listitems.length).toBe(0)
    }
    // If no "Other" group found, that's also acceptable (empty Other may be elided)
    // — the key test is that SessionAlpha appears under Alpha Team
    const alphaGroup = allGroups.find((g) => g.querySelector('.hub__group-header')?.textContent?.includes('Alpha Team'))
    expect(alphaGroup).not.toBeUndefined()
    expect(alphaGroup!.querySelectorAll('[role="listitem"]').length).toBe(1)
  })

  // ---- Phase 132: previewTails threading ----

  it('passes previewLines to each SessionCard from previewTails map', () => {
    // We verify threading by checking that previewTails data appears in the rendered card.
    // Since MiniPreview renders lines, the text should appear in the DOM.
    const sessions = [
      makeSession({ id: 'sess-preview', name: 'Preview Session', workDir: '/home/user/proj' }),
    ]
    const previewTails = new Map([
      ['sess-preview', ['output line 1', 'output line 2']],
    ])
    const { container } = renderGrid(sessions, vi.fn(), { previewTails })
    // MiniPreview renders lines as text in .hub-card__preview
    const preview = container.querySelector('.hub-card__preview')
    expect(preview).not.toBeNull()
    expect(preview!.textContent).toContain('output line 1')
    expect(preview!.textContent).toContain('output line 2')
  })

  it('renders MiniPreview in loading state when session id not in previewTails', () => {
    const sessions = [
      makeSession({ id: 'sess-no-preview', name: 'No Preview', workDir: '/home/user/proj' }),
    ]
    // previewTails doesn't contain this session → undefined → loading
    const previewTails = new Map<string, string[]>()
    const { container } = renderGrid(sessions, vi.fn(), { previewTails })
    // previewTails.get('sess-no-preview') === undefined → loading state in MiniPreview
    const loadingEl = container.querySelector('.hub-card__preview--loading')
    expect(loadingEl).not.toBeNull()
  })

  // ---- Phase 132: onAssignGroup threading ----

  it('threads onAssignGroup to each SessionCard (menu triggers callback)', () => {
    const onAssign = vi.fn()
    const sessions = [
      makeSession({ id: 'sess-1', name: 'Test Session', workDir: '/home/user/project' }),
    ]
    const groupDefs: HubGroupDef[] = [
      { id: 'group-x', name: 'Group X', memberKeys: [] },
    ]
    const { container } = renderGrid(sessions, vi.fn(), { groupDefs, onAssignGroup: onAssign })

    // Open the card menu
    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement | null
    expect(menuBtn).not.toBeNull()
    act(() => { menuBtn!.click() })

    // Click "Group X" menu item
    const menuItems = container.querySelectorAll('[role="menuitem"]')
    const groupXItem = Array.from(menuItems).find((el) => el.textContent?.trim() === 'Group X')
    expect(groupXItem).not.toBeNull()
    act(() => { (groupXItem as HTMLElement).click() })

    expect(onAssign).toHaveBeenCalledWith('Test Session:::/home/user/project', 'group-x')
  })

  // ---- groupByNamedGroups unit tests ----

  describe('groupByNamedGroups', () => {
    it('places session with matching memberKey in the correct named group', () => {
      const sessions = [
        makeSession({ id: 's1', name: 'SessionAlpha', workDir: '/home/user/alpha' }),
      ]
      const defs = makeGroupDefs()
      const result = groupByNamedGroups(sessions, defs)

      const alphaGroup = result.get('group-alpha')
      expect(alphaGroup).not.toBeUndefined()
      expect(alphaGroup!.label).toBe('Alpha Team')
      expect(alphaGroup!.sessions).toHaveLength(1)
      expect(alphaGroup!.sessions[0].id).toBe('s1')
    })

    it('places unmatched session in __other__ group', () => {
      const sessions = [
        makeSession({ id: 's1', name: 'Unmatched', workDir: '/some/other' }),
      ]
      const defs = makeGroupDefs()
      const result = groupByNamedGroups(sessions, defs)

      const otherGroup = result.get('__other__')
      expect(otherGroup).not.toBeUndefined()
      expect(otherGroup!.label).toBe('Other')
      expect(otherGroup!.sessions).toHaveLength(1)
    })

    it('named groups appear before __other__ in map iteration order', () => {
      const sessions = [
        makeSession({ id: 's1', name: 'Unmatched', workDir: '/none' }),
      ]
      const defs = makeGroupDefs()
      const result = groupByNamedGroups(sessions, defs)
      const keys = Array.from(result.keys())
      // group-alpha and group-beta come before __other__
      expect(keys.indexOf('group-alpha')).toBeLessThan(keys.indexOf('__other__'))
      expect(keys.indexOf('group-beta')).toBeLessThan(keys.indexOf('__other__'))
    })

    it('empty sessions returns empty sessions arrays in all groups', () => {
      const result = groupByNamedGroups([], makeGroupDefs())
      for (const [, group] of result) {
        expect(group.sessions).toHaveLength(0)
      }
    })

    it('session in no group goes to Other even when matching name but not workDir', () => {
      // The key is name:::workDir — if workDir differs, it's not matched
      const sessions = [
        makeSession({ id: 's1', name: 'SessionAlpha', workDir: '/different/path' }),
      ]
      const defs = makeGroupDefs() // alpha key is 'SessionAlpha:::/home/user/alpha'
      const result = groupByNamedGroups(sessions, defs)
      const otherGroup = result.get('__other__')
      expect(otherGroup!.sessions).toHaveLength(1)
      const alphaGroup = result.get('group-alpha')
      expect(alphaGroup!.sessions).toHaveLength(0)
    })
  })

  // ---- groupByWorkDir unit test (Phase 131 preserved) ----

  describe('groupByWorkDir', () => {
    it('groups sessions by workDir correctly', () => {
      const sessions = [
        makeSession({ id: 's1', workDir: '/alpha', name: 'A' }),
        makeSession({ id: 's2', workDir: '/alpha', name: 'B' }),
        makeSession({ id: 's3', workDir: '/beta', name: 'C' }),
      ]
      const result = groupByWorkDir(sessions)
      expect(result.get('/alpha')).toHaveLength(2)
      expect(result.get('/beta')).toHaveLength(1)
    })
  })
})
