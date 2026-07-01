import { describe, it, expect, vi, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'

// Mock Wails RPC before component import
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  ListSessions: vi.fn().mockResolvedValue([]),
}))

import { SessionCard } from './SessionCard'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import type { HubGroupDef } from '../../lib/hubGroups'
import { daemon } from '../../wailsjs/go/models'

// Helper to convert plain strings to StyledSpan[][] for previewLines tests.
function makePreviewLines(lines: string[]): daemon.StyledSpan[][] {
  return lines.map((line) => [{ c: line }])
}

function makeSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'sess-1',
    cli: 'claude',
    name: 'Test Session',
    state: 'running',
    status: 'running',
    createdAt: new Date(Date.now() - 7454000).toISOString(), // ~2h 4m ago
    // IN-03: local sessions have hostname: '' in production; remote sessions have a
    // non-empty peer hostname. Default to '' (local) so time/uptime tests reflect
    // the real local-session code path. Tests needing a remote session override explicitly.
    hostname: '',
    webEnabled: false,
    viewerCount: 0,
    homeDir: false,
    browseEnabled: false,
    funnelActive: false,
    workDir: '/home/user/project',
    ...overrides,
  }
}

function makeGroupDefs(): HubGroupDef[] {
  return [
    { id: 'group-a', name: 'Group A', memberKeys: ['Test Session:::/home/user/project'] },
    { id: 'group-b', name: 'Group B', memberKeys: [] },
  ]
}

function renderCard(session: SessionInfo, onRename?: (id: string, name: string) => void) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<SessionCard session={session} onRename={onRename} />)
  })
  return { container, root }
}

function renderCardWithAttention(session: SessionInfo, isAttention: boolean) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<SessionCard session={session} isAttention={isAttention} />)
  })
  return { container, root }
}

describe('SessionCard', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  // ---- Status rendering: icon + text label for every status ----

  it('running: renders icon with aria-label "Running" AND visible text label "Running"', () => {
    const { container } = renderCard(makeSession({ state: 'running', status: 'running' }))
    const icon = container.querySelector('[aria-label="Running"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Running')
  })

  it('idle: renders icon with aria-label "Idle" AND visible text label "Idle"', () => {
    const { container } = renderCard(makeSession({ state: 'idle', status: 'idle' }))
    const icon = container.querySelector('[aria-label="Idle"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Idle')
  })

  it('waiting: renders icon with aria-label "Needs input" AND visible text label "Needs input"', () => {
    const { container } = renderCard(makeSession({ state: 'waiting', status: 'waiting' }))
    const icon = container.querySelector('[aria-label="Needs input"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Needs input')
  })

  it('errored: renders icon with aria-label "Error" AND visible text label "Error"', () => {
    const { container } = renderCard(makeSession({ state: 'errored', status: 'errored' }))
    const icon = container.querySelector('[aria-label="Error"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Error')
  })

  it('stopped exit-0: renders icon with aria-label "Done" AND visible text label "Done"', () => {
    const exitCode = 0
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 3600 }))
    const icon = container.querySelector('[aria-label="Done"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Done')
  })

  it('stopped exit-nonzero: renders icon with aria-label containing "Exited" AND visible text label containing "Exited"', () => {
    const exitCode = 1
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 3600 }))
    const icon = container.querySelector('[aria-label*="Exited"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Exited')
  })

  // ---- Spin modifier on running only ----

  it('running card icon has hub-card__status-icon--spin class', () => {
    const { container } = renderCard(makeSession({ state: 'running', status: 'running' }))
    const spinIcon = container.querySelector('.hub-card__status-icon--spin')
    expect(spinIcon).not.toBeNull()
  })

  it('idle card icon does NOT have hub-card__status-icon--spin class', () => {
    const { container } = renderCard(makeSession({ state: 'idle', status: 'idle' }))
    const spinIcon = container.querySelector('.hub-card__status-icon--spin')
    expect(spinIcon).toBeNull()
  })

  // ---- Dimming ----

  it('stopped exit-0 card gets hub-card--dim class', () => {
    const exitCode = 0
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 120 }))
    const card = container.querySelector('.hub-card--dim')
    expect(card).not.toBeNull()
  })

  it('stopped non-zero exit card does NOT get hub-card--dim class', () => {
    const exitCode = 1
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 120 }))
    const card = container.querySelector('.hub-card--dim')
    expect(card).toBeNull()
  })

  it('stopped non-zero exit card shows exit-code chip with "Exited {code}"', () => {
    const exitCode = 2
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 120 }))
    const chip = container.querySelector('.hub-card__exit-chip')
    expect(chip).not.toBeNull()
    expect(chip!.textContent).toContain('2')
  })

  it('stopped exit-0 card does NOT show exit-code chip', () => {
    const exitCode = 0
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 120 }))
    const chip = container.querySelector('.hub-card__exit-chip')
    expect(chip).toBeNull()
  })

  // ---- Viewer count ----

  it('viewerCount === 0 hides the viewer row', () => {
    const { container } = renderCard(makeSession({ viewerCount: 0 }))
    const viewers = container.querySelector('.hub-card__viewers')
    expect(viewers).toBeNull()
  })

  it('viewerCount === 1 shows "1 viewer" (singular)', () => {
    const { container } = renderCard(makeSession({ viewerCount: 1 }))
    const viewers = container.querySelector('.hub-card__viewers')
    expect(viewers).not.toBeNull()
    expect(viewers!.textContent).toContain('1 viewer')
    expect(viewers!.textContent).not.toContain('1 viewers')
  })

  it('viewerCount === 3 shows "3 viewers" (plural)', () => {
    const { container } = renderCard(makeSession({ viewerCount: 3 }))
    const viewers = container.querySelector('.hub-card__viewers')
    expect(viewers).not.toBeNull()
    expect(viewers!.textContent).toContain('3 viewers')
  })

  // ---- Origin marker ----

  it('local session (empty hostname) shows "Local" with ComputerDesktopIcon', () => {
    const { container } = renderCard(makeSession({ hostname: '' }))
    const origin = container.querySelector('.hub-card__origin')
    expect(origin).not.toBeNull()
    expect(origin!.textContent).toContain('Local')
  })

  it('remote session shows peer hostname with GlobeAltIcon', () => {
    const { container } = renderCard(makeSession({ hostname: 'remote-peer.tail' }))
    const origin = container.querySelector('.hub-card__origin')
    expect(origin).not.toBeNull()
    expect(origin!.textContent).toContain('remote-peer.tail')
  })

  // ---- CLI badge ----

  it('renders CLI badge with cli name as text', () => {
    const { container } = renderCard(makeSession({ cli: 'claude' }))
    // WR-03/CR-01: badge now uses hub-card__badge (Hub text-chip) not tab__agent-badge dot
    const badge = container.querySelector('.hub-card__badge')
    expect(badge).not.toBeNull()
    // CLI name should be visible text
    expect(badge!.textContent).toContain('claude')
  })

  // ---- Uptime / duration ----

  it('running session shows uptime string (not "Ran")', () => {
    const { container } = renderCard(makeSession({ state: 'running', status: 'running' }))
    // CR-01: class name corrected from hub-card__time to hub-card__uptime (matches CSS)
    const timeRow = container.querySelector('.hub-card__uptime')
    expect(timeRow).not.toBeNull()
    expect(timeRow!.textContent).not.toContain('Ran')
  })

  it('stopped session shows "Ran ..." duration', () => {
    const { container } = renderCard(
      makeSession({ state: 'stopped', exitCode: 0, duration: 3600 })
    )
    // CR-01: class name corrected from hub-card__time to hub-card__uptime (matches CSS)
    const timeRow = container.querySelector('.hub-card__uptime')
    expect(timeRow).not.toBeNull()
    expect(timeRow!.textContent).toContain('Ran')
  })

  // ---- Aria-label on card ----

  it('card has aria-label containing name, status, cli, and origin', () => {
    const { container } = renderCard(makeSession({ hostname: '' }))
    const card = container.querySelector('.hub-card')
    expect(card).not.toBeNull()
    const label = card!.getAttribute('aria-label') ?? ''
    expect(label).toContain('Test Session')
    expect(label).toContain('claude')
    expect(label).toContain('Local')
  })

  it('card has tabIndex={0}', () => {
    const { container } = renderCard(makeSession())
    const card = container.querySelector('.hub-card')
    expect(card).not.toBeNull()
    expect((card as HTMLElement).tabIndex).toBe(0)
  })

  // ---- Open button (Phase 131 UAT follow-up: re-attach to a running session) ----

  function renderCardWithOpen(
    session: SessionInfo,
    onOpenSession?: (id: string, name: string, cli: string) => void,
  ) {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(<SessionCard session={session} onOpenSession={onOpenSession} />)
    })
    return { container, root }
  }

  it('renders an Open button for a live (running) session and calls onOpenSession with id/name/cli', () => {
    const onOpen = vi.fn()
    const { container } = renderCardWithOpen(
      makeSession({ id: 'sess-9', name: 'My Shell', cli: '/bin/zsh', state: 'running' }),
      onOpen,
    )
    const btn = container.querySelector('.hub-card__open') as HTMLButtonElement | null
    expect(btn).not.toBeNull()
    expect(btn!.textContent).toContain('Open')
    act(() => { btn!.click() })
    expect(onOpen).toHaveBeenCalledTimes(1)
    expect(onOpen).toHaveBeenCalledWith('sess-9', 'My Shell', '/bin/zsh')
  })

  it('does NOT render the Open button for a stopped session (no live PTY to attach)', () => {
    const { container } = renderCardWithOpen(
      makeSession({ state: 'stopped', status: 'stopped', exitCode: 0 }),
      vi.fn(),
    )
    expect(container.querySelector('.hub-card__open')).toBeNull()
  })

  it('does NOT render the Open button when onOpenSession is not provided', () => {
    const { container } = renderCard(makeSession({ state: 'running' }))
    expect(container.querySelector('.hub-card__open')).toBeNull()
  })

  // ---- PHASE 132 REGRESSION GUARD: Open button still fires onOpenSession ----

  it('REGRESSION(Phase 131): Open button fires onOpenSession — guard against Phase 132 breakage', () => {
    const onOpen = vi.fn()
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard
          session={makeSession({ id: 'r-sess', name: 'Regression', cli: 'claude', state: 'running' })}
          onOpenSession={onOpen}
          previewLines={makePreviewLines(['line1', 'line2'])}
          groupDefs={makeGroupDefs()}
          onAssignGroup={vi.fn()}
        />
      )
    })
    const btn = container.querySelector('.hub-card__open') as HTMLButtonElement | null
    expect(btn).not.toBeNull()
    act(() => { btn!.click() })
    expect(onOpen).toHaveBeenCalledWith('r-sess', 'Regression', 'claude')
  })

  // ---- ROW 6: MiniPreview (CARD-07) ----

  it('renders .hub-card__preview pane (ROW 6) when previewLines is undefined (loading state)', () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(<SessionCard session={makeSession()} previewLines={undefined} />)
    })
    const preview = container.querySelector('.hub-card__preview')
    expect(preview).not.toBeNull()
    // Loading state
    const loadingEl = container.querySelector('.hub-card__preview--loading')
    expect(loadingEl).not.toBeNull()
    expect(loadingEl!.textContent).toContain('Loading')
  })

  it('renders .hub-card__preview pane with lines when previewLines has content', () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(<SessionCard session={makeSession()} previewLines={makePreviewLines(['hello', 'world'])} />)
    })
    const preview = container.querySelector('.hub-card__preview')
    expect(preview).not.toBeNull()
    expect(preview!.textContent).toContain('hello')
    expect(preview!.textContent).toContain('world')
  })

  it('renders .hub-card__preview in empty state when previewLines is []', () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(<SessionCard session={makeSession()} previewLines={[]} />)
    })
    const emptyEl = container.querySelector('.hub-card__preview--empty')
    expect(emptyEl).not.toBeNull()
    expect(emptyEl!.textContent).toContain('No output yet')
  })

  it('renders ROW 6 preview even when previewLines prop is omitted', () => {
    // previewLines omitted → undefined → loading state
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(<SessionCard session={makeSession()} />)
    })
    // MiniPreview always renders; undefined = loading
    const preview = container.querySelector('.hub-card__preview')
    expect(preview).not.toBeNull()
  })

  // ---- Drag source (GROUP-02) ----

  it('article has draggable="true" attribute', () => {
    const { container } = renderCard(makeSession())
    const article = container.querySelector('article.hub-card')
    expect(article).not.toBeNull()
    expect(article!.getAttribute('draggable')).toBe('true')
  })

  it('dragStart sets text/plain to memberKey(name, workDir)', () => {
    const session = makeSession({ name: 'My Session', workDir: '/home/user/proj' })
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(<SessionCard session={session} />)
    })
    const article = container.querySelector('article.hub-card') as HTMLElement
    expect(article).not.toBeNull()

    // Simulate dragStart with a mock dataTransfer
    const dataTransfer: Record<string, string> = {}
    const dragEvent = new Event('dragstart', { bubbles: true }) as Event & { dataTransfer: DataTransfer }
    Object.defineProperty(dragEvent, 'dataTransfer', {
      value: {
        setData: (key: string, val: string) => { dataTransfer[key] = val },
        effectAllowed: 'none' as DataTransfer['effectAllowed'],
      },
      writable: true,
    })

    act(() => {
      article.dispatchEvent(dragEvent)
    })

    // memberKey('My Session', '/home/user/proj') = 'My Session:::/home/user/proj'
    expect(dataTransfer['text/plain']).toBe('My Session:::/home/user/proj')
  })

  it('renders .hub-card__drag-handle element', () => {
    const { container } = renderCard(makeSession())
    const handle = container.querySelector('.hub-card__drag-handle')
    expect(handle).not.toBeNull()
  })

  // ---- Overflow group menu (GROUP-02) ----

  it('renders .hub-card__menu-btn (overflow menu trigger) button', () => {
    const { container } = renderCard(makeSession())
    const menuBtn = container.querySelector('.hub-card__menu-btn')
    expect(menuBtn).not.toBeNull()
    expect(menuBtn!.tagName).toBe('BUTTON')
  })

  it('menu button has aria-haspopup="menu"', () => {
    const { container } = renderCard(makeSession())
    const menuBtn = container.querySelector('.hub-card__menu-btn')
    expect(menuBtn).not.toBeNull()
    expect(menuBtn!.getAttribute('aria-haspopup')).toBe('menu')
  })

  it('menu button has aria-expanded="false" initially', () => {
    const { container } = renderCard(makeSession())
    const menuBtn = container.querySelector('.hub-card__menu-btn')
    expect(menuBtn).not.toBeNull()
    expect(menuBtn!.getAttribute('aria-expanded')).toBe('false')
  })

  it('clicking menu button opens .hub-card__menu (role="menu") and sets aria-expanded="true"', () => {
    const session = makeSession()
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard session={session} groupDefs={makeGroupDefs()} onAssignGroup={vi.fn()} />
      )
    })

    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    expect(menuBtn).not.toBeNull()

    // Menu is closed initially
    expect(container.querySelector('[role="menu"]')).toBeNull()

    act(() => { menuBtn.click() })

    // Menu is now open
    const menu = container.querySelector('[role="menu"]')
    expect(menu).not.toBeNull()
    expect(menuBtn.getAttribute('aria-expanded')).toBe('true')
  })

  it('menu shows group defs as menu items when groupDefs provided', () => {
    const session = makeSession()
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard session={session} groupDefs={makeGroupDefs()} onAssignGroup={vi.fn()} />
      )
    })

    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    act(() => { menuBtn.click() })

    const menu = container.querySelector('[role="menu"]')
    expect(menu).not.toBeNull()
    expect(menu!.textContent).toContain('Group A')
    expect(menu!.textContent).toContain('Group B')
    expect(menu!.textContent).toContain('Other (default)')
  })

  it('selecting a group sub-item fires onAssignGroup(memberKey, groupId)', () => {
    const onAssign = vi.fn()
    const session = makeSession({ name: 'Test Session', workDir: '/home/user/project' })
    const groupDefs = makeGroupDefs()
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard session={session} groupDefs={groupDefs} onAssignGroup={onAssign} />
      )
    })

    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    act(() => { menuBtn.click() })

    // Find the "Group B" menu item (not in memberKeys)
    const menuItems = container.querySelectorAll('[role="menuitem"]')
    const groupBItem = Array.from(menuItems).find((el) => el.textContent?.trim() === 'Group B')
    expect(groupBItem).not.toBeNull()

    act(() => { (groupBItem as HTMLElement).click() })

    expect(onAssign).toHaveBeenCalledWith(
      'Test Session:::/home/user/project',
      'group-b',
    )
  })

  it('selecting "Other (default)" fires onAssignGroup(memberKey, "__other__")', () => {
    const onAssign = vi.fn()
    const session = makeSession({ name: 'Test Session', workDir: '/home/user/project' })
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard session={session} groupDefs={makeGroupDefs()} onAssignGroup={onAssign} />
      )
    })

    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    act(() => { menuBtn.click() })

    const menuItems = container.querySelectorAll('[role="menuitem"]')
    const otherItem = Array.from(menuItems).find((el) =>
      el.textContent?.includes('Other (default)')
    )
    expect(otherItem).not.toBeNull()

    act(() => { (otherItem as HTMLElement).click() })

    expect(onAssign).toHaveBeenCalledWith(
      'Test Session:::/home/user/project',
      '__other__',
    )
  })

  it('WR-01: shows "Other (default)" only when session IS in a named group (acts as remove)', () => {
    // session is in 'Group A' (memberKeys includes its key) — Other (default) is visible
    const session = makeSession({ name: 'Test Session', workDir: '/home/user/project' })
    const groupDefsWithMember: HubGroupDef[] = [
      { id: 'group-a', name: 'Group A', memberKeys: ['Test Session:::/home/user/project'] },
    ]
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard session={session} groupDefs={groupDefsWithMember} onAssignGroup={vi.fn()} />
      )
    })

    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    act(() => { menuBtn.click() })

    const menu = container.querySelector('[role="menu"]')
    expect(menu).not.toBeNull()
    // "Other (default)" is shown (moves session back to Other — combined with old "Remove from group")
    expect(menu!.textContent).toContain('Other (default)')
    // Old "Remove from group" section is gone — replaced by "Other (default)" above
    expect(menu!.textContent).not.toContain('Remove from group')
  })

  it('WR-01: does NOT show "Other (default)" when session is already ungrouped', () => {
    // session's memberKey is NOT in any groupDef.memberKeys → isInNamedGroup = false
    const session = makeSession({ name: 'Unmatched Session', workDir: '/other/path' })
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard session={session} groupDefs={makeGroupDefs()} onAssignGroup={vi.fn()} />
      )
    })

    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    act(() => { menuBtn.click() })

    const menu = container.querySelector('[role="menu"]')
    expect(menu).not.toBeNull()
    // Session is already in "Other" — showing this is a no-op; hidden per WR-01
    expect(menu!.textContent).not.toContain('Other (default)')
    expect(menu!.textContent).not.toContain('Remove from group')
  })

  it('pressing Escape when menu is open closes the menu', () => {
    const session = makeSession()
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard session={session} groupDefs={makeGroupDefs()} onAssignGroup={vi.fn()} />
      )
    })

    const menuBtn = container.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    act(() => { menuBtn.click() })
    expect(container.querySelector('[role="menu"]')).not.toBeNull()

    act(() => {
      const escEvent = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })
      document.dispatchEvent(escEvent)
    })

    expect(container.querySelector('[role="menu"]')).toBeNull()
  })

  // ---- STATUS_CONFIG usage (source-level check via grep — verified in acceptance_criteria)
  // ---- COLORBLIND-SAFE comments (source-level check via grep — verified in acceptance_criteria)

  // ---- Phase 166 / FUI-03: internet exposure badge ----

  it('FUI-03: funnelActive=true renders .hub-internet-badge with "INTERNET" text', () => {
    const { container } = renderCard(makeSession({ funnelActive: true }))
    const badge = container.querySelector('.hub-internet-badge')
    expect(badge).not.toBeNull()
    expect(badge!.textContent).toContain('INTERNET')
  })

  it('FUI-03: funnelActive=true badge contains .hub-internet-badge__label with "INTERNET"', () => {
    const { container } = renderCard(makeSession({ funnelActive: true }))
    const label = container.querySelector('.hub-internet-badge__label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toBe('INTERNET')
  })

  it('FUI-03: funnelActive=true badge contains .hub-internet-badge__icon (GlobeAltIcon, aria-hidden)', () => {
    const { container } = renderCard(makeSession({ funnelActive: true }))
    const icon = container.querySelector('.hub-internet-badge__icon')
    expect(icon).not.toBeNull()
    expect(icon!.getAttribute('aria-hidden')).toBe('true')
  })

  it('FUI-03: funnelActive=false renders no .hub-internet-badge', () => {
    const { container } = renderCard(makeSession({ funnelActive: false }))
    const badge = container.querySelector('.hub-internet-badge')
    expect(badge).toBeNull()
  })

  it('FUI-03 COLORBLIND-SAFE: SessionCard.tsx source contains COLORBLIND-SAFE comment for the internet badge', () => {
    expect(rawSessionCard).toContain('COLORBLIND-SAFE')
    expect(rawSessionCard).toContain('hub-internet-badge')
  })
})

// ---- Phase 134 Plan 02: source-inspection assertions (TDD RED) ----
// These tests import SessionCard.tsx as raw text and assert the click contract
// is present at source level — no DOM mounting required.

import rawSessionCard from './SessionCard.tsx?raw'

describe('SessionCard Phase 134 source-inspection (onCardClick + stopPropagation)', () => {
  it('declares onCardClick prop in SessionCardProps', () => {
    expect(rawSessionCard).toContain('onCardClick')
  })

  it('article onClick body calls getBoundingClientRect', () => {
    expect(rawSessionCard).toContain('getBoundingClientRect')
  })

  it('article onClick body guards against .hub-card__open clicks', () => {
    expect(rawSessionCard).toContain("closest('.hub-card__open')")
  })

  it('article onClick body guards against .hub-card__menu-btn clicks', () => {
    expect(rawSessionCard).toContain("closest('.hub-card__menu-btn')")
  })

  it('Open button onClick calls e.stopPropagation()', () => {
    // The Open button handler must include stopPropagation BEFORE the onOpenSession call
    const openBtnIdx = rawSessionCard.indexOf('hub-card__open')
    expect(openBtnIdx).toBeGreaterThan(-1)
    // The next onClick after the hub-card__open class declaration should include stopPropagation
    const afterOpenClass = rawSessionCard.slice(openBtnIdx)
    const firstOnClickIdx = afterOpenClass.indexOf('onClick')
    expect(firstOnClickIdx).toBeGreaterThan(-1)
    const onClickHandler = afterOpenClass.slice(firstOnClickIdx, firstOnClickIdx + 200)
    expect(onClickHandler).toContain('e.stopPropagation()')
  })

  it('menu button onClick calls e.stopPropagation()', () => {
    // The menu button handler must include stopPropagation
    const menuBtnIdx = rawSessionCard.indexOf('hub-card__menu-btn')
    expect(menuBtnIdx).toBeGreaterThan(-1)
    const afterMenuClass = rawSessionCard.slice(menuBtnIdx)
    const firstOnClickIdx = afterMenuClass.indexOf('onClick')
    expect(firstOnClickIdx).toBeGreaterThan(-1)
    const onClickHandler = afterMenuClass.slice(firstOnClickIdx, firstOnClickIdx + 100)
    expect(onClickHandler).toContain('e.stopPropagation()')
  })

  it('article onKeyDown fires onCardClick on Enter', () => {
    expect(rawSessionCard).toContain('onKeyDown')
    expect(rawSessionCard).toContain("'Enter'")
  })
})

describe('SessionCard attention (ATTN-01)', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  it('isAttention=true: card article has .hub-card--attention class', () => {
    const { container } = renderCardWithAttention(
      makeSession({ state: 'waiting', status: 'waiting' }),
      true,
    )
    const card = container.querySelector('.hub-card--attention')
    expect(card).not.toBeNull()
  })

  it('isAttention=true: renders .hub-card__attn-icon element with aria-label "Needs attention"', () => {
    const { container } = renderCardWithAttention(
      makeSession({ state: 'waiting', status: 'waiting' }),
      true,
    )
    const attnIcon = container.querySelector('.hub-card__attn-icon')
    expect(attnIcon).not.toBeNull()
    expect(attnIcon!.getAttribute('aria-label')).toBe('Needs attention')
  })

  it('isAttention=true: card article aria-label ends with ", needs attention"', () => {
    const { container } = renderCardWithAttention(
      makeSession({ state: 'waiting', status: 'waiting' }),
      true,
    )
    const card = container.querySelector('.hub-card')
    expect(card).not.toBeNull()
    const label = card!.getAttribute('aria-label') ?? ''
    expect(label).toMatch(/, needs attention$/)
  })

  it('isAttention=false: no .hub-card--attention, no .hub-card__attn-icon, aria-label does NOT contain "needs attention"', () => {
    const { container } = renderCardWithAttention(
      makeSession({ state: 'running', status: 'running' }),
      false,
    )
    expect(container.querySelector('.hub-card--attention')).toBeNull()
    expect(container.querySelector('.hub-card__attn-icon')).toBeNull()
    const card = container.querySelector('.hub-card')
    const label = card!.getAttribute('aria-label') ?? ''
    expect(label).not.toContain('needs attention')
  })

  it('REGRESSION(Phase 131/132): isAttention=true still renders .hub-card__status-indicator and Open button for a live session', () => {
    const onOpen = vi.fn()
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard
          session={makeSession({ state: 'waiting', status: 'waiting' })}
          isAttention={true}
          onOpenSession={onOpen}
        />,
      )
    })
    // Status indicator preserved
    expect(container.querySelector('.hub-card__status-indicator')).not.toBeNull()
    // Open button preserved for live session
    expect(container.querySelector('.hub-card__open')).not.toBeNull()
  })

  // ---- NOTIF-01 / D-10: ChatBadge on SessionCard ----

  it('NOTIF-01: unreadCount=0 renders no chat badge (count-0 = no DOM node)', () => {
    const { container } = renderCard(makeSession())
    // Default: no unreadCount prop → badge absent
    const badge = container.querySelector('.chat-badge')
    expect(badge).toBeNull()
  })

  it('NOTIF-01: unreadCount=3 renders chat badge with count text', () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard session={makeSession()} unreadCount={3} hasChatMention={false} />
      )
    })
    const badge = container.querySelector('.chat-badge')
    expect(badge).not.toBeNull()
    expect(badge!.textContent).toBe('3')
  })

  it('D-10: hasChatMention=true renders chat-badge--mention with @ glyph (COLORBLIND-SAFE: shape signal)', () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    act(() => {
      root.render(
        <SessionCard session={makeSession()} unreadCount={2} hasChatMention={true} />
      )
    })
    const badge = container.querySelector('.chat-badge--mention')
    expect(badge).not.toBeNull()
    expect(badge!.textContent).toBe('@')
  })
})
