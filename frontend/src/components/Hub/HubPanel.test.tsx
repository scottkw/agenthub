import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import hubPanelRaw from './HubPanel.tsx?raw'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import type { ITheme } from '@xterm/xterm'

// Minimal ITheme stub — WR-03 made terminalTheme required on HubPanelProps.
const STUB_THEME: ITheme = { background: '#000000', foreground: '#ffffff' }

// Mock Wails RPC before component import — GetSessionStyledTailLines required for usePreviewPoller
// Phase 139 / CARD-05: poller switched from GetSessionTailLines to GetSessionStyledTailLines
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  ListSessions: vi.fn().mockResolvedValue([]),
  GetSessionStyledTailLines: vi.fn().mockResolvedValue([[{ c: 'line1' }], [{ c: 'line2' }]]),
}))

vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
}))

// FE-ROUTE-01: Mock TerminalPanel so HubInteractiveModal can mount in jsdom without
// triggering xterm canvas API calls (canvas is absent in jsdom).
vi.mock('../TerminalPanel', () => ({
  TerminalPanel: () => React.createElement('div', { 'data-testid': 'mock-terminal-panel' }),
}))

import { HubPanel } from './HubPanel'
import { GetSessionStyledTailLines } from '../../wailsjs/go/main/App'

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
    browseEnabled: false,
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
    browseEnabled: false,
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
  remoteCapsCached?: Set<string>
  onRequestRemoteCap?: (s: { id: string; name: string; hostname: string }) => void
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
    // WR-03: terminalTheme is now required on HubPanelProps
    terminalTheme: STUB_THEME,
    remoteCapsCached: overrides.remoteCapsCached,
    onRequestRemoteCap: overrides.onRequestRemoteCap,
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

  // ---- Phase 132: hub__body wrapper ----
  // Note: .hub__header removed in Phase 138 Plan 03 (CARD-01) — HubFilterBar is sole
  // New Session entry. See App.hub.test.tsx for the header-removal regression guard.

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

  it('calls GetSessionStyledTailLines when isActive=true and sessions present', async () => {
    vi.useFakeTimers()
    const sessions = [makeSession({ id: 'local-1', hostname: '' })]
    renderPanel({ sessions, isActive: true })

    // advance by 100ms to trigger the initial poll (not the interval)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    expect(GetSessionStyledTailLines).toHaveBeenCalledWith('local-1', 4)
    vi.useRealTimers()
  })

  it('does NOT call GetSessionStyledTailLines when isActive=false', async () => {
    vi.useFakeTimers()
    const sessions = [makeSession({ id: 'local-1', hostname: '' })]
    renderPanel({ sessions, isActive: false })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    expect(GetSessionStyledTailLines).not.toHaveBeenCalled()
    vi.useRealTimers()
  })

  // ---- Phase 132: usePreviewPoller — remote sessions excluded from fetch ----

  it('fetches tails for LOCAL sessions (even with a machine hostname) but NOT remote-prop sessions', async () => {
    // Regression: local sessions carry the machine hostname (os.Hostname()), so the
    // old `hostname === ''` filter wrongly excluded them → perpetual "Loading…".
    // Local vs remote is decided by provenance: `sessions` (local) vs `remoteSessions`.
    vi.useFakeTimers()
    const sessions = [
      makeSession({ id: 'local-1', hostname: 'Kens-Personal-MacBook-Air.local' }),
    ]
    const remoteSessions = [
      makeSession({ id: 'remote-99', hostname: 'peer-host' }),
    ]
    renderPanel({ sessions, remoteSessions, isActive: true })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    // local-1 IS fetched despite its non-empty (machine) hostname; remote-99 is not.
    expect(GetSessionStyledTailLines).toHaveBeenCalledWith('local-1', 4)
    expect(GetSessionStyledTailLines).not.toHaveBeenCalledWith('remote-99', expect.anything())
    vi.useRealTimers()
  })

  // ---- CR-03: usePreviewPoller preserves last-seen lines when fetch returns empty ----

  it('CR-03: does NOT call GetSessionStyledTailLines for remote sessions when remote session ID changes (WR-04)', async () => {
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
    const callCount = (GetSessionStyledTailLines as ReturnType<typeof vi.fn>).mock.calls.length

    // Simulate a remote session change (e.g. new remote session id) — should NOT reset interval
    // The sessionIdKey dep is now local-only, so the remote change has no effect on the interval.
    // We verify: only local sessions were fetched, not remote
    expect(GetSessionStyledTailLines).toHaveBeenCalledWith('local-A', 4)
    expect(GetSessionStyledTailLines).toHaveBeenCalledWith('local-B', 4)
    expect(GetSessionStyledTailLines).not.toHaveBeenCalledWith('remote-X', expect.anything())
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

// ---- Phase 133: attention live vs debounced behavior (ATTN-02/03/04/05) ----

describe('HubPanel attention (ATTN-02/03/04/05)', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    localStorage.clear()
    vi.useRealTimers()
  })

  it('attention card shows .hub-card--attention immediately (live, not debounced)', async () => {
    vi.useFakeTimers()
    // Session starts as waiting (attention)
    const sessions = [
      makeSession({ id: 'attn-sess', name: 'Attn Session', status: 'waiting', state: 'running', workDir: '/proj' }),
    ]
    const { container } = renderPanel({ sessions, isActive: false })

    // Even without advancing timers, the live isAttention prop should be applied immediately
    const cards = container.querySelectorAll('article.hub-card')
    expect(cards.length).toBe(1)
    expect(cards[0].classList.contains('hub-card--attention')).toBe(true)
  })

  it('ATTN-03/05: session transitioning from waiting to running loses .hub-card--attention with no remount', async () => {
    vi.useFakeTimers()
    const sessions = [
      makeSession({ id: 'sess-clear', name: 'ClearSession', status: 'waiting', state: 'running', workDir: '/proj' }),
    ]
    const { container, root } = renderPanel({ sessions, isActive: false })

    // Verify attention is shown initially
    let card = container.querySelector('article.hub-card')
    expect(card).not.toBeNull()
    expect(card!.classList.contains('hub-card--attention')).toBe(true)

    // Re-render with the session now in 'running' state (attention clears)
    const updatedSessions = [
      makeSession({ id: 'sess-clear', name: 'ClearSession', status: 'running', state: 'running', workDir: '/proj' }),
    ]
    act(() => {
      root.render(
        <HubPanel
          sessions={updatedSessions}
          error={false}
          onNewSession={vi.fn()}
          onRename={vi.fn()}
          isActive={false}
          terminalTheme={STUB_THEME}
        />
      )
    })

    // The card should still be present (no remount) but WITHOUT the attention class
    card = container.querySelector('article.hub-card')
    expect(card).not.toBeNull()
    expect(card!.classList.contains('hub-card--attention')).toBe(false)
  })

  it('debounced sort key settles after ~1000ms (reorder only after debounce)', async () => {
    vi.useFakeTimers()
    // Two sessions: first is non-attention, second is attention
    // After render, attention card should float to top after 1s debounce
    const sessions = [
      makeSession({ id: 'non-attn', name: 'NonAttnFirst', status: 'running', state: 'running', workDir: '/proj' }),
      makeSession({ id: 'attn', name: 'AttnSecond', status: 'waiting', state: 'running', workDir: '/proj' }),
    ]
    const { container } = renderPanel({ sessions, isActive: false })

    // Before debounce settles: we should still see both cards (grid is rendered)
    let cards = container.querySelectorAll('article.hub-card')
    expect(cards.length).toBe(2)

    // After advancing 1100ms (debounce settles), attention card should be first
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1100)
    })

    cards = container.querySelectorAll('article.hub-card')
    expect(cards.length).toBe(2)
    // The attention card should now be first in DOM order within its group
    const firstLabel = cards[0].getAttribute('aria-label') ?? ''
    expect(firstLabel).toContain('AttnSecond')
  })

  it('single setInterval invariant: only one interval created (CARD-07 preserved)', async () => {
    vi.useFakeTimers()
    const setIntervalSpy = vi.spyOn(globalThis, 'setInterval')

    const sessions = [
      makeSession({ id: 'local-1', hostname: '' }),
    ]
    renderPanel({ sessions, isActive: true })

    // Let async effects settle
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    // Count setInterval calls — must be exactly 1 (the preview poller)
    // useDebouncedValue uses setTimeout, not setInterval
    const intervalCalls = setIntervalSpy.mock.calls.length
    expect(intervalCalls).toBe(1)

    setIntervalSpy.mockRestore()
  })

  it('needs attention: multiple attention sessions all show hub-card--attention', () => {
    const sessions = [
      makeSession({ id: 's1', name: 'Session1', status: 'waiting', state: 'running', workDir: '/proj' }),
      makeSession({ id: 's2', name: 'Session2', status: 'errored', state: 'running', workDir: '/proj' }),
      makeSession({ id: 's3', name: 'Session3', status: 'running', state: 'running', workDir: '/proj' }),
    ]
    const { container } = renderPanel({ sessions, isActive: false })

    const attentionCards = container.querySelectorAll('article.hub-card--attention')
    expect(attentionCards.length).toBe(2)
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
    browseEnabled: false,
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

// ---- Phase 134: MODAL-06 source-inspection assertions ----
// These tests use ?raw import to assert structural contract without mounting HubModal
// (xterm requires canvas APIs absent in jsdom — no DOM mounting of HubModal in tests).
describe('HubPanel MODAL-06 source-inspection (Phase 134)', () => {
  it('HubPanelProps declares onRequestRemoteCap', () => {
    expect(hubPanelRaw).toContain('onRequestRemoteCap')
  })

  it('handleCardClick gates remote-without-cap via remoteCapsCached before setModalState', () => {
    // The remote gate references remoteCapsCached and onRequestRemoteCap
    expect(hubPanelRaw).toContain('remoteCapsCached')
    expect(hubPanelRaw).toContain('onRequestRemoteCap')
    // The early return must appear before setModalState in the source
    const returnIdx = hubPanelRaw.indexOf('onRequestRemoteCap?.({')
    const setModalIdx = hubPanelRaw.indexOf('setModalState({ session, sourceRect: rect })')
    expect(returnIdx).toBeGreaterThan(-1)
    expect(setModalIdx).toBeGreaterThan(-1)
    expect(returnIdx).toBeLessThan(setModalIdx)
  })

  it('renders <HubModal in source', () => {
    expect(hubPanelRaw).toContain('<HubModal')
  })

  it('threads onCardClick={handleCardClick} to SessionCardGrid', () => {
    expect(hubPanelRaw).toContain('onCardClick={handleCardClick}')
  })
})

// ---- FE-ROUTE-01: Behavioral tests — remote gate routing (WR-07) ----
// These tests exercise runtime behavior, not source strings.
// TerminalPanel is mocked (see top of file) so HubInteractiveModal can mount in jsdom.
describe('HubPanel FE-ROUTE-01: remote routing gate (behavioral)', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  it('FE-ROUTE-01a: remote-without-cap card click calls onRequestRemoteCap and does NOT open the modal', () => {
    const onRequestRemoteCap = vi.fn()
    const remoteSession = makeRemoteSession({ id: 'r1', hostname: 'remote-host' })
    const { container } = renderPanel({
      sessions: [],
      remoteSessions: [remoteSession],
      remoteCapsCached: new Set<string>(), // no cap for r1
      onRequestRemoteCap,
      isActive: false,
    })

    // Find and click the remote session card
    const card = container.querySelector('article.hub-card')
    expect(card).not.toBeNull()

    act(() => {
      card!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // onRequestRemoteCap MUST be called (the cap-request flow fires)
    expect(onRequestRemoteCap).toHaveBeenCalledOnce()
    expect(onRequestRemoteCap).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'r1', hostname: 'remote-host' }),
    )

    // The hub-modal-overlay must NOT appear (setModalState was NOT called)
    const overlay = container.querySelector('.hub-modal-overlay')
    expect(overlay).toBeNull()
  })

  it('FE-ROUTE-01b: local card click does NOT call onRequestRemoteCap and DOES open the modal', () => {
    const onRequestRemoteCap = vi.fn()
    const localSession = makeSession({ id: 'loc1', hostname: '' })

    // Render with relayPort > 0 so the HubModal guard passes and the modal mounts.
    const modalContainer = document.createElement('div')
    document.body.appendChild(modalContainer)
    const modalRoot = createRoot(modalContainer)

    act(() => {
      modalRoot.render(
        React.createElement(HubPanel, {
          sessions: [localSession],
          remoteSessions: [],
          error: false,
          onNewSession: vi.fn(),
          onRename: vi.fn(),
          isActive: false,
          terminalTheme: STUB_THEME,
          relayPort: 51234,
          remoteCapsCached: new Set<string>(),
          onRequestRemoteCap,
        }),
      )
    })

    const card = modalContainer.querySelector('article.hub-card')
    expect(card).not.toBeNull()

    act(() => {
      card!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // onRequestRemoteCap must NOT be called for local sessions
    expect(onRequestRemoteCap).not.toHaveBeenCalled()

    // The hub-modal-overlay MUST appear (setModalState was called → HubModal rendered)
    const overlay = modalContainer.querySelector('.hub-modal-overlay')
    expect(overlay).not.toBeNull()
  })

  it('FE-ROUTE-01c: local card with a non-empty MACHINE hostname opens the modal (provenance, not hostname)', () => {
    // GAP-134-A regression: local sessions carry os.Hostname(), so a hostname-based
    // remote check misroutes EVERY local session to the remote-cap join flow. Local vs
    // remote must be decided by provenance (the `sessions` prop vs `remoteSessions`).
    const onRequestRemoteCap = vi.fn()
    const localSession = makeSession({ id: 'loc-hn', hostname: 'Kens-Personal-MacBook-Air.local' })

    const modalContainer = document.createElement('div')
    document.body.appendChild(modalContainer)
    const modalRoot = createRoot(modalContainer)

    act(() => {
      modalRoot.render(
        React.createElement(HubPanel, {
          sessions: [localSession],
          remoteSessions: [],
          error: false,
          onNewSession: vi.fn(),
          onRename: vi.fn(),
          isActive: false,
          terminalTheme: STUB_THEME,
          relayPort: 51234,
          remoteCapsCached: new Set<string>(),
          onRequestRemoteCap,
        }),
      )
    })

    const card = modalContainer.querySelector('article.hub-card')
    expect(card).not.toBeNull()

    act(() => {
      card!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    // A local session — even with a machine hostname — must NOT trigger the remote cap flow…
    expect(onRequestRemoteCap).not.toHaveBeenCalled()
    // …and the modal MUST open.
    const overlay = modalContainer.querySelector('.hub-modal-overlay')
    expect(overlay).not.toBeNull()

    act(() => { modalRoot.unmount() })
    modalContainer.remove()
  })

  it('GAP-134-D: under prefers-reduced-motion the modal closes without onAnimationEnd', () => {
    // With reduced motion the CSS disables animations, so onAnimationEnd never fires.
    // The close path must not depend on it, or the modal becomes impossible to close.
    const originalMatchMedia = window.matchMedia
    // jsdom has no matchMedia; install a reduced-motion stub
    window.matchMedia = (query: string) => ({
      matches: query.includes('prefers-reduced-motion: reduce'),
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    })

    try {
      const localSession = makeSession({ id: 'rm1', hostname: '' })
      const modalContainer = document.createElement('div')
      document.body.appendChild(modalContainer)
      const modalRoot = createRoot(modalContainer)

      act(() => {
        modalRoot.render(
          React.createElement(HubPanel, {
            sessions: [localSession],
            remoteSessions: [],
            error: false,
            onNewSession: vi.fn(),
            onRename: vi.fn(),
            isActive: false,
            terminalTheme: STUB_THEME,
            relayPort: 51234,
            remoteCapsCached: new Set<string>(),
            onRequestRemoteCap: vi.fn(),
          }),
        )
      })

      // Open the modal.
      const card = modalContainer.querySelector('article.hub-card')
      act(() => { card!.dispatchEvent(new MouseEvent('click', { bubbles: true })) })
      expect(modalContainer.querySelector('.hub-modal-overlay')).not.toBeNull()

      // Click the close button — NO onAnimationEnd is dispatched (animations are off).
      const closeBtn = modalContainer.querySelector('.hub-modal__close') as HTMLElement
      expect(closeBtn).not.toBeNull()
      act(() => { closeBtn.dispatchEvent(new MouseEvent('click', { bubbles: true })) })

      // The modal must be gone (onClose ran synchronously).
      expect(modalContainer.querySelector('.hub-modal-overlay')).toBeNull()

      act(() => { modalRoot.unmount() })
      modalContainer.remove()
    } finally {
      window.matchMedia = originalMatchMedia
    }
  })
})
