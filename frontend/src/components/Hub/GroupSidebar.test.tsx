import { describe, it, expect, vi, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import { GroupSidebar } from './GroupSidebar'
import type { HubGroupDef } from '../../lib/hubGroups'
import type { SessionInfo } from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  ListSessions: vi.fn().mockResolvedValue([]),
}))

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

function makeGroup(overrides: Partial<HubGroupDef> = {}): HubGroupDef {
  return {
    id: 'group-1',
    name: 'Alpha',
    memberKeys: [],
    ...overrides,
  }
}

interface RenderGroupSidebarOptions {
  groupDefs?: HubGroupDef[]
  sessions?: SessionInfo[]
  activeGroupId?: string | null
  collapsed?: boolean
  onToggle?: () => void
  onGroupSelect?: (id: string | null) => void
  onCreateGroup?: (name: string) => void
  onDropOnGroup?: (groupId: string, memberKey: string) => void
}

function renderSidebar(opts: RenderGroupSidebarOptions = {}) {
  const {
    groupDefs = [],
    sessions = [],
    activeGroupId = null,
    collapsed = false,
    onToggle = vi.fn(),
    onGroupSelect = vi.fn(),
    onCreateGroup = vi.fn(),
    onDropOnGroup = vi.fn(),
  } = opts

  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(
      <GroupSidebar
        groupDefs={groupDefs}
        sessions={sessions}
        activeGroupId={activeGroupId}
        collapsed={collapsed}
        onToggle={onToggle}
        onGroupSelect={onGroupSelect}
        onCreateGroup={onCreateGroup}
        onDropOnGroup={onDropOnGroup}
      />
    )
  })
  return { container, root, onToggle, onGroupSelect, onCreateGroup, onDropOnGroup }
}

describe('GroupSidebar', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  // ---- All item + group items ----

  it('renders an "All" item at the top of the list', () => {
    const { container } = renderSidebar()
    const items = container.querySelectorAll('.hub__group-sidebar-item')
    expect(items.length).toBeGreaterThanOrEqual(1)
    expect(items[0].textContent).toContain('All')
  })

  it('renders one item per groupDef in addition to All', () => {
    const groups = [makeGroup({ id: 'g1', name: 'Alpha' }), makeGroup({ id: 'g2', name: 'Beta' })]
    const { container } = renderSidebar({ groupDefs: groups })
    const items = container.querySelectorAll('.hub__group-sidebar-item')
    // All + 2 groups
    expect(items.length).toBe(3)
    expect(items[1].textContent).toContain('Alpha')
    expect(items[2].textContent).toContain('Beta')
  })

  // ---- Counts ----

  it('each item shows running/total count', () => {
    const session = makeSession({ id: 's1', name: 'S1', workDir: '/x', state: 'running', status: 'running' })
    const groups = [makeGroup({ id: 'g1', name: 'Alpha', memberKeys: ['S1:::/x'] })]
    const { container } = renderSidebar({ groupDefs: groups, sessions: [session] })
    const countEls = container.querySelectorAll('.hub__group-sidebar-item__count')
    // At least one count element
    expect(countEls.length).toBeGreaterThanOrEqual(1)
    // All item: 1/1
    expect(countEls[0].textContent).toContain('1/1')
  })

  // ---- Needs-input badge ----

  it('renders a needs-input badge for a group with a waiting session', () => {
    const session = makeSession({ id: 's1', name: 'S1', workDir: '/x', state: 'waiting', status: 'waiting' })
    const groups = [makeGroup({ id: 'g1', name: 'Alpha', memberKeys: ['S1:::/x'] })]
    const { container } = renderSidebar({ groupDefs: groups, sessions: [session] })
    const badge = container.querySelector('.hub__group-sidebar-item__needs-input-badge')
    expect(badge).not.toBeNull()
  })

  it('needs-input badge contains a PauseCircleIcon (svg element)', () => {
    const session = makeSession({ id: 's1', name: 'S1', workDir: '/x', state: 'waiting', status: 'waiting' })
    const groups = [makeGroup({ id: 'g1', name: 'Alpha', memberKeys: ['S1:::/x'] })]
    const { container } = renderSidebar({ groupDefs: groups, sessions: [session] })
    const badge = container.querySelector('.hub__group-sidebar-item__needs-input-badge')
    expect(badge).not.toBeNull()
    // PauseCircleIcon renders as an SVG
    const svg = badge!.querySelector('svg')
    expect(svg).not.toBeNull()
  })

  it('needs-input badge aria-label says "1 session needs input" for count 1', () => {
    const session = makeSession({ id: 's1', name: 'S1', workDir: '/x', state: 'waiting', status: 'waiting' })
    const groups = [makeGroup({ id: 'g1', name: 'Alpha', memberKeys: ['S1:::/x'] })]
    const { container } = renderSidebar({ groupDefs: groups, sessions: [session] })
    const badge = container.querySelector('.hub__group-sidebar-item__needs-input-badge')
    expect(badge!.getAttribute('aria-label')).toBe('1 session needs input')
  })

  it('needs-input badge aria-label says "2 sessions need input" for count 2', () => {
    const s1 = makeSession({ id: 's1', name: 'S1', workDir: '/x', state: 'waiting', status: 'waiting' })
    const s2 = makeSession({ id: 's2', name: 'S2', workDir: '/x', state: 'waiting', status: 'waiting' })
    const groups = [makeGroup({ id: 'g1', name: 'Alpha', memberKeys: ['S1:::/x', 'S2:::/x'] })]
    const { container } = renderSidebar({ groupDefs: groups, sessions: [s1, s2] })
    const badge = container.querySelector('.hub__group-sidebar-item__needs-input-badge')
    expect(badge!.getAttribute('aria-label')).toBe('2 sessions need input')
  })

  it('does NOT render a needs-input badge when no waiting sessions in group', () => {
    const session = makeSession({ id: 's1', name: 'S1', workDir: '/x', state: 'running', status: 'running' })
    const groups = [makeGroup({ id: 'g1', name: 'Alpha', memberKeys: ['S1:::/x'] })]
    const { container } = renderSidebar({ groupDefs: groups, sessions: [session] })
    const badge = container.querySelector('.hub__group-sidebar-item__needs-input-badge')
    expect(badge).toBeNull()
  })

  // ---- Clicking items ----

  it('clicking a group item fires onGroupSelect with group id', () => {
    const groups = [makeGroup({ id: 'g1', name: 'Alpha' })]
    const { container, onGroupSelect } = renderSidebar({ groupDefs: groups })
    const items = container.querySelectorAll('.hub__group-sidebar-item')
    // Second item is Alpha
    act(() => { (items[1] as HTMLElement).click() })
    expect(onGroupSelect).toHaveBeenCalledWith('g1')
  })

  it('clicking "All" item fires onGroupSelect(null)', () => {
    const { container, onGroupSelect } = renderSidebar()
    const items = container.querySelectorAll('.hub__group-sidebar-item')
    act(() => { (items[0] as HTMLElement).click() })
    expect(onGroupSelect).toHaveBeenCalledWith(null)
  })

  // ---- Active state ----

  it('active group item has hub__group-sidebar-item--active class and aria-selected="true"', () => {
    const groups = [makeGroup({ id: 'g1', name: 'Alpha' })]
    const { container } = renderSidebar({ groupDefs: groups, activeGroupId: 'g1' })
    const items = container.querySelectorAll('.hub__group-sidebar-item')
    // Alpha item (index 1)
    expect(items[1].className).toContain('hub__group-sidebar-item--active')
    expect(items[1].getAttribute('aria-selected')).toBe('true')
  })

  it('non-active items have aria-selected="false"', () => {
    const groups = [makeGroup({ id: 'g1', name: 'Alpha' })]
    const { container } = renderSidebar({ groupDefs: groups, activeGroupId: null })
    const items = container.querySelectorAll('.hub__group-sidebar-item')
    expect(items[0].getAttribute('aria-selected')).toBe('true') // All is active when no group selected
    expect(items[1].getAttribute('aria-selected')).toBe('false')
  })

  // ---- Collapsed state ----

  it('collapsed=true adds hub__group-sidebar--collapsed class to root', () => {
    const { container } = renderSidebar({ collapsed: true })
    const sidebar = container.firstElementChild as HTMLElement
    expect(sidebar.className).toContain('hub__group-sidebar--collapsed')
  })

  it('collapsed=true hides group labels (does not render group name text)', () => {
    const groups = [makeGroup({ id: 'g1', name: 'Alpha' })]
    const { container } = renderSidebar({ groupDefs: groups, collapsed: true })
    // When collapsed, .hub__group-sidebar-item__name elements should not be present
    const nameEls = container.querySelectorAll('.hub__group-sidebar-item__name')
    expect(nameEls.length).toBe(0)
  })

  it('collapsed=true: toggle button aria-label is "Expand group sidebar"', () => {
    const { container } = renderSidebar({ collapsed: true })
    const toggleBtn = container.querySelector('.hub__group-sidebar-toggle')
    expect(toggleBtn).not.toBeNull()
    expect(toggleBtn!.getAttribute('aria-label')).toBe('Expand group sidebar')
  })

  it('collapsed=false: toggle button aria-label is "Collapse group sidebar"', () => {
    const { container } = renderSidebar({ collapsed: false })
    const toggleBtn = container.querySelector('.hub__group-sidebar-toggle')
    expect(toggleBtn!.getAttribute('aria-label')).toBe('Collapse group sidebar')
  })

  it('clicking toggle button fires onToggle', () => {
    const { container, onToggle } = renderSidebar({ collapsed: false })
    const toggleBtn = container.querySelector('.hub__group-sidebar-toggle') as HTMLButtonElement
    act(() => { toggleBtn.click() })
    expect(onToggle).toHaveBeenCalledTimes(1)
  })

  // ---- ARIA roles ----

  it('list has role="listbox"', () => {
    const { container } = renderSidebar()
    const list = container.querySelector('.hub__group-sidebar-list')
    expect(list).not.toBeNull()
    expect(list!.getAttribute('role')).toBe('listbox')
  })

  it('each item has role="option"', () => {
    const groups = [makeGroup({ id: 'g1', name: 'Alpha' })]
    const { container } = renderSidebar({ groupDefs: groups })
    const items = container.querySelectorAll('.hub__group-sidebar-item')
    items.forEach((item) => {
      expect(item.getAttribute('role')).toBe('option')
    })
  })

  // ---- Create flow ----

  it('clicking "New group" shows an inline input', () => {
    const { container } = renderSidebar()
    const newBtn = container.querySelector('.hub__group-sidebar-new') as HTMLButtonElement
    expect(newBtn).not.toBeNull()
    act(() => { newBtn.click() })
    const input = container.querySelector('input')
    expect(input).not.toBeNull()
  })

  it('Enter with non-empty name fires onCreateGroup and hides input', () => {
    const { container, onCreateGroup } = renderSidebar()
    const newBtn = container.querySelector('.hub__group-sidebar-new') as HTMLButtonElement
    act(() => { newBtn.click() })
    const input = container.querySelector('input') as HTMLInputElement
    act(() => {
      input.value = 'My Group'
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    act(() => {
      input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    })
    expect(onCreateGroup).toHaveBeenCalledWith('My Group')
  })

  it('Enter with empty/whitespace name does NOT fire onCreateGroup', () => {
    const { container, onCreateGroup } = renderSidebar()
    const newBtn = container.querySelector('.hub__group-sidebar-new') as HTMLButtonElement
    act(() => { newBtn.click() })
    const input = container.querySelector('input') as HTMLInputElement
    act(() => {
      input.value = '   '
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    act(() => {
      input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    })
    expect(onCreateGroup).not.toHaveBeenCalled()
  })

  it('Escape cancels creation and restores the "New group" button', () => {
    const { container, onCreateGroup } = renderSidebar()
    const newBtn = container.querySelector('.hub__group-sidebar-new') as HTMLButtonElement
    act(() => { newBtn.click() })
    const input = container.querySelector('input') as HTMLInputElement
    act(() => {
      input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    expect(onCreateGroup).not.toHaveBeenCalled()
    // Input should be gone, button should be back
    expect(container.querySelector('input')).toBeNull()
    expect(container.querySelector('.hub__group-sidebar-new')).not.toBeNull()
  })

  // ---- Drop target ----

  it('dropping on a group item calls onDropOnGroup with groupId and member key', () => {
    const groups = [makeGroup({ id: 'g1', name: 'Alpha' })]
    const { container, onDropOnGroup } = renderSidebar({ groupDefs: groups })
    const items = container.querySelectorAll('.hub__group-sidebar-item')
    // Alpha item (index 1)
    const alphaItem = items[1] as HTMLElement

    const dropData = 'MySession:::/home/user'
    act(() => {
      const dropEvent = new DragEvent('drop', { bubbles: true, cancelable: true })
      Object.defineProperty(dropEvent, 'dataTransfer', {
        value: {
          getData: (type: string) => type === 'text/plain' ? dropData : '',
        },
      })
      alphaItem.dispatchEvent(dropEvent)
    })

    expect(onDropOnGroup).toHaveBeenCalledWith('g1', dropData)
  })

  it('dragover on a group item sets drag-over class', () => {
    const groups = [makeGroup({ id: 'g1', name: 'Alpha' })]
    const { container } = renderSidebar({ groupDefs: groups })
    const items = container.querySelectorAll('.hub__group-sidebar-item')
    const alphaItem = items[1] as HTMLElement

    act(() => {
      const dragOverEvent = new DragEvent('dragover', { bubbles: true, cancelable: true })
      alphaItem.dispatchEvent(dragOverEvent)
    })

    // After dragover, the item should have a drag-over class
    expect(alphaItem.className).toContain('drag-over')
  })
})
