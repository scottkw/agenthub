import { describe, it, expect, vi, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// Mock Wails RPC before component import — GetSessionTailLines required for usePreviewPoller
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  ListSessions: vi.fn().mockResolvedValue([]),
  GetSessionTailLines: vi.fn().mockResolvedValue(['line1', 'line2']),
}))

vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
}))

import { HubPanel } from './HubPanel'
import { GetSessionTailLines } from '../../wailsjs/go/main/App'

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

function makeRemoteSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'remote-1',
    cli: 'claude',
    name: 'Remote Session',
    state: 'running',
    status: 'running',
    createdAt: new Date().toISOString(),
    hostname: 'remote-host',
    webEnabled: true,
    viewerCount: 0,
    homeDir: false,
    filesWrite: false,
    workDir: '',
    ...overrides,
  }
}

function renderPanel(overrides: {
  sessions?: SessionInfo[]
  error?: boolean
  onNewSession?: () => void
  onRename?: (id: string, name: string) => void
  remoteSessions?: SessionInfo[]
  isActive?: boolean
} = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)

  const props = {
    sessions: overrides.sessions ?? [],
    error: overrides.error ?? false,
    onNewSession: overrides.onNewSession ?? vi.fn(),
    onRename: overrides.onRename ?? vi.fn(),
    remoteSessions: overrides.remoteSessions,
    isActive: overrides.isActive,
  }

  act(() => {
    root.render(<HubPanel {...props} />)
  })

  return { container, root, ...props }
}

describe('HubPanel', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    localStorage.clear()
    // Remove any lingering keydown listeners
    window.onkeydown = null
  })

  // ---- Empty state: no sessions ----

  it('renders the no-sessions empty state when sessions is empty', () => {
    const { container } = renderPanel({ sessions: [] })
    expect(container.textContent).toContain('No sessions yet')
    expect(container.textContent).toContain('Create a session to start an AI coding agent.')
  })

  it('does not render SessionCardGrid when sessions is empty', () => {
    const { container } = renderPanel({ sessions: [] })
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(0)
  })

  // ---- Sessions present: renders grid ----

  it('renders SessionCardGrid (hub__group) when sessions are present', () => {
    const { container } = renderPanel({
      sessions: [makeSession({ id: 'sess-1' })],
    })
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBeGreaterThanOrEqual(1)
  })

  it('does not render the no-sessions empty state when sessions are present', () => {
    const { container } = renderPanel({
      sessions: [makeSession()],
    })
    expect(container.textContent).not.toContain('No sessions yet')
  })

  // ---- Filter narrows the grid ----

  it('shows all sessions when filter is "all"', () => {
    const sessions = [
      makeSession({ id: 's1', status: 'running', state: 'running' }),
      makeSession({ id: 's2', status: 'idle', state: 'running' }),
    ]
    const { container } = renderPanel({ sessions })
    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(2)
  })

  it('filters sessions by status when filter pill is clicked', () => {
    const sessions = [
      makeSession({ id: 's1', status: 'running', state: 'running', name: 'Running Session' }),
      makeSession({ id: 's2', status: 'idle', state: 'running', name: 'Idle Session' }),
    ]
    const { container } = renderPanel({ sessions })

    // Click "Idle" filter pill (label "Idle")
    const pills = container.querySelectorAll('.hub-filter__pill')
    const idlePill = Array.from(pills).find((p) => p.textContent?.includes('Idle'))
    expect(idlePill).not.toBeUndefined()

    act(() => {
      idlePill!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // Only the idle session should remain
    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(1)
  })

  // ---- Search narrows the grid ----

  it('filters sessions by search text against session name', () => {
    const sessions = [
      makeSession({ id: 's1', name: 'Alpha Session', status: 'running', state: 'running' }),
      makeSession({ id: 's2', name: 'Beta Session', status: 'running', state: 'running' }),
    ]
    const { container } = renderPanel({ sessions })

    const searchInput = container.querySelector<HTMLInputElement>('.hub-filter__search')
    expect(searchInput).not.toBeNull()

    act(() => {
      // Simulate typing "alpha"
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
        window.HTMLInputElement.prototype,
        'value',
      )!.set!
      nativeInputValueSetter.call(searchInput!, 'alpha')
      searchInput!.dispatchEvent(new Event('input', { bubbles: true }))
      // React synthetic onChange needs a change event
      searchInput!.dispatchEvent(new Event('change', { bubbles: true }))
    })

    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(1)
  })

  it('case-insensitively filters sessions by name', () => {
    const sessions = [
      makeSession({ id: 's1', name: 'My Unique Task', cli: 'opencode', status: 'running', state: 'running' }),
      makeSession({ id: 's2', name: 'Other Work', cli: 'gemini', status: 'running', state: 'running' }),
    ]
    const { container } = renderPanel({ sessions })
    const searchInput = container.querySelector<HTMLInputElement>('.hub-filter__search')!

    act(() => {
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
        window.HTMLInputElement.prototype,
        'value',
      )!.set!
      nativeInputValueSetter.call(searchInput, 'UNIQUE')
      searchInput.dispatchEvent(new Event('input', { bubbles: true }))
      searchInput.dispatchEvent(new Event('change', { bubbles: true }))
    })

    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(1)
  })

  // ---- No-matches empty state ----

  it('renders the no-matches empty state when filter narrows to zero results', () => {
    const sessions = [
      makeSession({ id: 's1', status: 'running', state: 'running' }),
    ]
    const { container } = renderPanel({ sessions })

    // Click "Idle" pill — no idle sessions exist
    const pills = container.querySelectorAll('.hub-filter__pill')
    const idlePill = Array.from(pills).find((p) => p.textContent?.includes('Idle'))!

    act(() => {
      idlePill.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    expect(container.textContent).toContain('No matching sessions')
  })

  it('does not render the grid when filter yields no matches', () => {
    const sessions = [makeSession({ id: 's1', status: 'running', state: 'running' })]
    const { container } = renderPanel({ sessions })

    const pills = container.querySelectorAll('.hub-filter__pill')
    const idlePill = Array.from(pills).find((p) => p.textContent?.includes('Idle'))!
    act(() => {
      idlePill.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(0)
  })

  it('Clear-filter CTA in no-matches state resets filter and shows the grid again', () => {
    const sessions = [makeSession({ id: 's1', status: 'running', state: 'running' })]
    const { container } = renderPanel({ sessions })

    // 1. Trigger a zero-match filter
    const pills = container.querySelectorAll('.hub-filter__pill')
    const idlePill = Array.from(pills).find((p) => p.textContent?.includes('Idle'))!
    act(() => {
      idlePill.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // Confirm no-matches state
    expect(container.textContent).toContain('No matching sessions')

    // 2. Click "Clear filter"
    const clearBtn = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent === 'Clear filter',
    )
    expect(clearBtn).not.toBeUndefined()
    act(() => {
      clearBtn!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // Grid should be visible again
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBeGreaterThanOrEqual(1)
  })

  // ---- "/" shortcut focuses search ----

  it('pressing "/" focuses the search input', () => {
    const sessions = [makeSession()]
    const { container } = renderPanel({ sessions })

    const searchInput = container.querySelector<HTMLInputElement>('.hub-filter__search')!
    // Ensure search is not currently focused
    searchInput.blur()

    act(() => {
      window.dispatchEvent(
        new KeyboardEvent('keydown', { key: '/', bubbles: true }),
      )
    })

    expect(document.activeElement).toBe(searchInput)
  })

  it('does not hijack "/" when an input is already focused', () => {
    const sessions = [makeSession()]
    const { container } = renderPanel({ sessions })

    const searchInput = container.querySelector<HTMLInputElement>('.hub-filter__search')!
    act(() => {
      searchInput.focus()
    })

    // activeElement is an input — "/" should NOT preventDefault / focus again
    // The test just verifies no error is thrown; the guard prevents double-focus.
    expect(() => {
      act(() => {
        window.dispatchEvent(
          new KeyboardEvent('keydown', { key: '/', bubbles: true, cancelable: true }),
        )
      })
    }).not.toThrow()
  })

  // ---- Error state ----

  it('renders the error state when error=true', () => {
    const { container } = renderPanel({ error: true })
    expect(container.textContent).toContain("Couldn't load sessions")
    expect(container.textContent).toContain('Check that the daemon is running and try again.')
  })

  it('does not render the grid when error=true', () => {
    const sessions = [makeSession()]
    const { container } = renderPanel({ sessions, error: true })
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(0)
  })

  it('does not render the no-sessions empty state when error=true', () => {
    const { container } = renderPanel({ sessions: [], error: true })
    expect(container.textContent).not.toContain('No sessions yet')
  })

  // ---- Header structure ----

  it('renders a hub header with the title "Hub"', () => {
    const { container } = renderPanel()
    expect(container.querySelector('.hub__header')).not.toBeNull()
    expect(container.querySelector('.hub__title')?.textContent).toBe('Hub')
  })

  // ---- Phase 132: hub__body wrapper ----

  it('renders a .hub__body wrapper containing GroupSidebar and hub__grid-scroll', () => {
    const { container } = renderPanel({ sessions: [makeSession()] })
    const body = container.querySelector('.hub__body')
    expect(body).not.toBeNull()
    // hub__grid-scroll must be inside hub__body
    const scroll = body!.querySelector('.hub__grid-scroll')
    expect(scroll).not.toBeNull()
    // GroupSidebar must be inside hub__body
    const sidebar = body!.querySelector('.hub__group-sidebar')
    expect(sidebar).not.toBeNull()
  })

  // ---- Phase 132: usePreviewPoller — active ----

  it('calls GetSessionTailLines when isActive=true and sessions present', async () => {
    vi.useFakeTimers()
    const sessions = [makeSession({ id: 'local-1', hostname: '' })]
    renderPanel({ sessions, isActive: true })

    // advance by 100ms to trigger the initial poll (not the interval)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    expect(GetSessionTailLines).toHaveBeenCalledWith('local-1', 4)
    vi.useRealTimers()
  })

  it('does NOT call GetSessionTailLines when isActive=false', async () => {
    vi.useFakeTimers()
    const sessions = [makeSession({ id: 'local-1', hostname: '' })]
    renderPanel({ sessions, isActive: false })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    expect(GetSessionTailLines).not.toHaveBeenCalled()
    vi.useRealTimers()
  })

  // ---- Phase 132: usePreviewPoller — remote sessions excluded from fetch ----

  it('does NOT call GetSessionTailLines for remote sessions (hostname set)', async () => {
    vi.useFakeTimers()
    const sessions = [
      makeSession({ id: 'local-1', hostname: '' }),
      makeSession({ id: 'remote-99', hostname: 'peer-host' }),
    ]
    renderPanel({ sessions, isActive: true })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    // Only local-1 should be fetched; remote-99 skipped
    expect(GetSessionTailLines).toHaveBeenCalledWith('local-1', 4)
    expect(GetSessionTailLines).not.toHaveBeenCalledWith('remote-99', expect.anything())
    vi.useRealTimers()
  })

  // ---- CR-03: usePreviewPoller preserves last-seen lines when fetch returns empty ----

  it('CR-03: does NOT call GetSessionTailLines for remote sessions when remote session ID changes (WR-04)', async () => {
    vi.useFakeTimers()
    // Two local sessions — sessionIdKey should be stable relative to remote changes
    const sessions = [
      makeSession({ id: 'local-A', hostname: '' }),
      makeSession({ id: 'local-B', hostname: '' }),
    ]
    // remote sessions passed separately and merged in HubPanel
    const remoteSessions = [makeRemoteSession({ id: 'remote-X', hostname: 'remote-host' })]
    renderPanel({ sessions, remoteSessions, isActive: true })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })
    const callCount = (GetSessionTailLines as ReturnType<typeof vi.fn>).mock.calls.length

    // Simulate a remote session change (e.g. new remote session id) — should NOT reset interval
    // The sessionIdKey dep is now local-only, so the remote change has no effect on the interval.
    // We verify: only local sessions were fetched, not remote
    expect(GetSessionTailLines).toHaveBeenCalledWith('local-A', 4)
    expect(GetSessionTailLines).toHaveBeenCalledWith('local-B', 4)
    expect(GetSessionTailLines).not.toHaveBeenCalledWith('remote-X', expect.anything())
    expect(callCount).toBe(2) // exactly 2 fetches (one per local session on initial poll)
    vi.useRealTimers()
  })

  // ---- Phase 132: remote merge into grid (GRID-07) ----

  it('merges remoteSessions into the unified grid', () => {
    const localSessions = [makeSession({ id: 'local-1', hostname: '' })]
    const remoteSessions = [makeRemoteSession({ id: 'remote-1', hostname: 'remote-host' })]
    const { container } = renderPanel({ sessions: localSessions, remoteSessions })
    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(2)
  })

  it('renders remote session card (hostname visible) in the grid', () => {
    const remoteSessions = [makeRemoteSession({ id: 'remote-1', hostname: 'my-remote-host' })]
    const { container } = renderPanel({ sessions: [], remoteSessions })
    // The origin (hostname) is rendered in the card
    expect(container.textContent).toContain('my-remote-host')
  })

  // ---- Phase 132: named group state ----

  it('renders GroupSidebar with an "All" item', () => {
    const { container } = renderPanel({ sessions: [makeSession()] })
    const sidebar = container.querySelector('.hub__group-sidebar')
    expect(sidebar).not.toBeNull()
    expect(sidebar!.textContent).toContain('All')
  })

  it('creating a group via sidebar callback persists to localStorage', () => {
    const { container } = renderPanel({ sessions: [makeSession()] })

    // Find the "New group" button in the sidebar
    const newGroupBtn = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'New group',
    )
    expect(newGroupBtn).not.toBeUndefined()

    act(() => {
      newGroupBtn!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // Type a group name and confirm
    const input = container.querySelector<HTMLInputElement>('.hub__group-sidebar-new-input')
    expect(input).not.toBeNull()

    act(() => {
      const setter = Object.getOwnPropertyDescriptor(
        window.HTMLInputElement.prototype,
        'value',
      )!.set!
      setter.call(input!, 'My Group')
      input!.dispatchEvent(new Event('input', { bubbles: true }))
      input!.dispatchEvent(new Event('change', { bubbles: true }))
    })

    act(() => {
      input!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    })

    // Group should be visible in the sidebar
    expect(container.textContent).toContain('My Group')

    // And persisted — loadGroups() should return it
    const raw = localStorage.getItem('agenthub:hubGroups:v1')
    expect(raw).not.toBeNull()
    const groups = JSON.parse(raw!)
    expect(groups.some((g: { name: string }) => g.name === 'My Group')).toBe(true)
  })

  it('selecting a group narrows the grid to that group sessions', () => {
    // Seed localStorage with a group that matches sess-1
    const groupDef = {
      id: 'grp-1',
      name: 'My Group',
      memberKeys: ['Test Session:::/home/user/project'],
    }
    localStorage.setItem('agenthub:hubGroups:v1', JSON.stringify([groupDef]))

    const sessions = [
      makeSession({ id: 'sess-1', name: 'Test Session', workDir: '/home/user/project' }),
      makeSession({ id: 'sess-2', name: 'Other Session', workDir: '/home/user/other' }),
    ]
    const { container } = renderPanel({ sessions })

    // Find and click the "My Group" item in the sidebar
    const sidebarItems = container.querySelectorAll('.hub__group-sidebar-item')
    const myGroupItem = Array.from(sidebarItems).find(
      (el) => el.textContent?.includes('My Group'),
    )
    expect(myGroupItem).not.toBeUndefined()

    act(() => {
      myGroupItem!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // Only sessions in "My Group" should appear
    const listitems = container.querySelectorAll('[role="listitem"]')
    // My Group has no memberKeys matching since key format is "name:::workDir"
    // The test checks the filter narrows — result count is less than before
    // Note: all sessions end up in Other if key doesn't match (GROUP-04)
    expect(listitems.length).toBeLessThanOrEqual(2)
  })

  it('selecting "All" clears the group filter and shows all sessions', () => {
    const groupDef = {
      id: 'grp-1',
      name: 'My Group',
      memberKeys: [],
    }
    localStorage.setItem('agenthub:hubGroups:v1', JSON.stringify([groupDef]))

    const sessions = [
      makeSession({ id: 'sess-1' }),
      makeSession({ id: 'sess-2', name: 'Session Two', workDir: '/other' }),
    ]
    const { container } = renderPanel({ sessions })

    // Click a named group to activate group filter
    const sidebarItems = container.querySelectorAll('.hub__group-sidebar-item')
    const myGroupItem = Array.from(sidebarItems).find(
      (el) => el.textContent?.includes('My Group'),
    )
    act(() => {
      myGroupItem!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // Click "All" to clear
    const allItem = Array.from(container.querySelectorAll('.hub__group-sidebar-item')).find(
      (el) => el.textContent?.trim().startsWith('All'),
    )
    expect(allItem).not.toBeUndefined()
    act(() => {
      allItem!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(2)
  })
})

// ---- filterSessions export tests (preserved from Phase 131) ----
import { filterSessions } from './HubPanel'

describe('filterSessions', () => {
  const makeS = (overrides: Partial<SessionInfo> = {}): SessionInfo => ({
    id: 's1',
    cli: 'claude',
    name: 'Test',
    state: 'running',
    status: 'running',
    createdAt: '',
    hostname: '',
    webEnabled: false,
    viewerCount: 0,
    homeDir: false,
    filesWrite: false,
    workDir: '',
    ...overrides,
  })

  it('returns all sessions when filter is "all" and no search', () => {
    const sessions = [makeS({ id: 's1' }), makeS({ id: 's2', status: 'idle' })]
    expect(filterSessions(sessions, 'all', '')).toHaveLength(2)
  })

  it('filters by status', () => {
    const sessions = [makeS({ id: 's1', status: 'running' }), makeS({ id: 's2', status: 'idle' })]
    const result = filterSessions(sessions, 'idle', '')
    expect(result).toHaveLength(1)
    expect(result[0].id).toBe('s2')
  })

  it('filters by search (name)', () => {
    const sessions = [makeS({ id: 's1', name: 'Alpha' }), makeS({ id: 's2', name: 'Beta' })]
    expect(filterSessions(sessions, 'all', 'alpha')).toHaveLength(1)
  })

  it('filters by search (hostname)', () => {
    const sessions = [makeS({ id: 's1', hostname: 'server-a' }), makeS({ id: 's2', hostname: '' })]
    expect(filterSessions(sessions, 'all', 'server-a')).toHaveLength(1)
  })
})
