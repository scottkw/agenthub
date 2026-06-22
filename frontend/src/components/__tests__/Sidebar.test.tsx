import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { Sidebar } from '../Sidebar'
import { readFileSync } from 'fs'
import { resolve } from 'path'
import type { HubGroupDef } from '../../lib/hubGroups'

// CSS source for contract tests (jsdom has no layout engine — we inspect raw CSS text)
// This is the established project pattern from SettingsTab, WelcomeTab, TerminalPanel tests.
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

// Source files for POL-03 / POL-04 source-gate assertions
const terminalPanelRaw = readFileSync(
  resolve(__dirname, '../TerminalPanel.tsx'),
  'utf-8',
)
const hubFilterBarRaw = readFileSync(
  resolve(__dirname, '../Hub/HubFilterBar.tsx'),
  'utf-8',
)
const hubEmptyStateRaw = readFileSync(
  resolve(__dirname, '../Hub/HubEmptyState.tsx'),
  'utf-8',
)

// Helper: default group defs for POL-05 tests
function makeGroup(overrides: Partial<HubGroupDef> = {}): HubGroupDef {
  return {
    id: 'grp-1',
    name: 'Alpha',
    memberKeys: [],
    ...overrides,
  }
}

// Helper to render Sidebar with default no-op props
// Phase 138 / NAV-02..05: reduced to 3-item sidebar (Home, Hub, Settings only).
// onOpenDaemonManager, onOpenRemoteSessions, onAdd removed — those panels are deleted.
function renderSidebar(overrides: Partial<Parameters<typeof Sidebar>[0]> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const defaultProps = {
    onSettings: vi.fn(),
    onHome: vi.fn(),
    onOpenHub: vi.fn(),
    // POL-05 additions — required props with safe empty defaults
    groupDefs: [] as HubGroupDef[],
    activeGroupId: null as string | null,
    onGroupSelect: vi.fn(),
    onCreateGroup: vi.fn(),
    onDropOnGroup: vi.fn(),
    groupCounts: {} as Record<string, { running: number; total: number; attention: number; waiting: number }>,
    globalGroupCounts: { running: 0, total: 0, attention: 0, waiting: 0 },
  }
  act(() => {
    root.render(<Sidebar {...defaultProps} {...overrides} />)
  })
  return { container, root, ...defaultProps, ...(overrides as typeof defaultProps) }
}

describe('Sidebar component (SIDE-01)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders a <nav> element with class "sidebar"', () => {
    ;({ container, root } = renderSidebar())
    const nav = container.querySelector('nav.sidebar')
    expect(nav).not.toBeNull()
  })

  it('renders <nav> with aria-label="Main navigation"', () => {
    ;({ container, root } = renderSidebar())
    const nav = container.querySelector('nav')
    expect(nav).not.toBeNull()
    expect(nav!.getAttribute('aria-label')).toBe('Main navigation')
  })

  it('renders buttons with class sidebar__item for nav items', () => {
    ;({ container, root } = renderSidebar())
    const items = container.querySelectorAll('button.sidebar__item')
    expect(items.length).toBeGreaterThan(0)
  })

  // Phase 138 / NAV-02..05: exactly 3 items (Home, Hub, Settings)
  it('renders exactly 3 sidebar__item buttons (Home, Hub, Settings)', () => {
    ;({ container, root } = renderSidebar())
    const items = container.querySelectorAll('button.sidebar__item')
    expect(items.length).toBe(3)
  })

  // Phase 138 / NAV-03: Sessions panel removed — button must be absent
  it('does NOT render a Sessions button', () => {
    ;({ container, root } = renderSidebar())
    expect(container.querySelector('button[aria-label="Sessions"]')).toBeNull()
  })

  // Phase 138 / NAV-02: Remote panel removed — button must be absent
  it('does NOT render a Remote button', () => {
    ;({ container, root } = renderSidebar())
    expect(container.querySelector('button[aria-label="Remote"]')).toBeNull()
  })

  // Phase 138 / NAV-05: New Session sidebar item removed — creation lives in HubFilterBar
  it('does NOT render a New Session button', () => {
    ;({ container, root } = renderSidebar())
    expect(container.querySelector('button[aria-label="New Session"]')).toBeNull()
  })
})

describe('Sidebar toggle (SIDE-02)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    localStorage.clear()
  })

  it('renders with class "sidebar" and NOT "sidebar--collapsed" by default (expanded state)', () => {
    ;({ container, root } = renderSidebar())
    const nav = container.querySelector('nav')
    expect(nav).not.toBeNull()
    expect(nav!.classList.contains('sidebar')).toBe(true)
    expect(nav!.classList.contains('sidebar--collapsed')).toBe(false)
  })

  it('clicking the toggle button (.sidebar__toggle) adds sidebar--collapsed class', () => {
    ;({ container, root } = renderSidebar())
    const toggleBtn = container.querySelector('.sidebar__toggle') as HTMLButtonElement
    expect(toggleBtn).not.toBeNull()
    act(() => {
      toggleBtn.click()
    })
    const nav = container.querySelector('nav')
    expect(nav!.classList.contains('sidebar--collapsed')).toBe(true)
  })

  it('clicking toggle again removes sidebar--collapsed (back to expanded)', () => {
    ;({ container, root } = renderSidebar())
    const toggleBtn = container.querySelector('.sidebar__toggle') as HTMLButtonElement
    act(() => {
      toggleBtn.click()
    })
    act(() => {
      toggleBtn.click()
    })
    const nav = container.querySelector('nav')
    expect(nav!.classList.contains('sidebar--collapsed')).toBe(false)
  })
})

describe('Sidebar localStorage persistence (SIDE-03)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    localStorage.clear()
  })

  it('renders without sidebar--collapsed when localStorage has no sidebar-collapsed key (expanded default)', () => {
    expect(localStorage.getItem('sidebar-collapsed')).toBeNull()
    ;({ container, root } = renderSidebar())
    const nav = container.querySelector('nav')
    expect(nav!.classList.contains('sidebar--collapsed')).toBe(false)
  })

  it('renders with sidebar--collapsed when localStorage has sidebar-collapsed = "true"', () => {
    localStorage.setItem('sidebar-collapsed', 'true')
    ;({ container, root } = renderSidebar())
    const nav = container.querySelector('nav')
    expect(nav!.classList.contains('sidebar--collapsed')).toBe(true)
  })

  it('toggling the sidebar writes "true" to localStorage key sidebar-collapsed when collapsing', () => {
    ;({ container, root } = renderSidebar())
    const toggleBtn = container.querySelector('.sidebar__toggle') as HTMLButtonElement
    act(() => {
      toggleBtn.click()
    })
    expect(localStorage.getItem('sidebar-collapsed')).toBe('true')
  })

  it('toggling the sidebar writes "false" to localStorage key sidebar-collapsed when expanding', () => {
    localStorage.setItem('sidebar-collapsed', 'true')
    ;({ container, root } = renderSidebar())
    const toggleBtn = container.querySelector('.sidebar__toggle') as HTMLButtonElement
    act(() => {
      toggleBtn.click()
    })
    expect(localStorage.getItem('sidebar-collapsed')).toBe('false')
  })
})

describe('Sidebar icon centering precondition (SBR-01)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    localStorage.clear()
  })

  it('collapsed sidebar items contain only an SVG icon (no label span)', () => {
    ;({ container, root } = renderSidebar())
    const toggleBtn = container.querySelector('.sidebar__toggle') as HTMLButtonElement
    act(() => { toggleBtn.click() })
    const items = container.querySelectorAll('.sidebar__item')
    items.forEach((item) => {
      expect(item.querySelector('svg')).not.toBeNull()
      expect(item.querySelector('.sidebar__label')).toBeNull()
    })
  })

  // Phase 138 / NAV-02..05: 3 items remain in DOM when collapsed (was 6)
  it('all 3 sidebar items remain in DOM when collapsed', () => {
    ;({ container, root } = renderSidebar())
    const toggleBtn = container.querySelector('.sidebar__toggle') as HTMLButtonElement
    act(() => { toggleBtn.click() })
    const items = container.querySelectorAll('.sidebar__item')
    expect(items.length).toBe(3)
  })
})

describe('Sidebar icons (ICON-01, ICON-02)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders SVG elements for toggle and nav icons (querySelectorAll("svg").length >= 2)', () => {
    ;({ container, root } = renderSidebar())
    const svgs = container.querySelectorAll('svg')
    expect(svgs.length).toBeGreaterThanOrEqual(2)
  })
})

describe('Sidebar Hub item (HUB-01, Phase 131)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders a Hub button with aria-label="Hub"', () => {
    ;({ container, root } = renderSidebar())
    const hubBtn = container.querySelector('button[aria-label="Hub"]')
    expect(hubBtn).not.toBeNull()
    expect(hubBtn!.classList.contains('sidebar__item')).toBe(true)
  })

  it('Hub button fires onOpenHub when clicked', () => {
    const onOpenHub = vi.fn()
    ;({ container, root } = renderSidebar({ onOpenHub }))
    const hubBtn = container.querySelector('button[aria-label="Hub"]') as HTMLButtonElement
    expect(hubBtn).not.toBeNull()
    act(() => { hubBtn.click() })
    expect(onOpenHub).toHaveBeenCalledTimes(1)
  })

  it('Hub button does NOT have sidebar__item--active when activePanel is not __hub__', () => {
    ;({ container, root } = renderSidebar({ activePanel: '__settings__' }))
    const hubBtn = container.querySelector('button[aria-label="Hub"]')
    expect(hubBtn).not.toBeNull()
    expect(hubBtn!.classList.contains('sidebar__item--active')).toBe(false)
  })

  it('Hub button has sidebar__item--active when activePanel === "__hub__"', () => {
    ;({ container, root } = renderSidebar({ activePanel: '__hub__' }))
    const hubBtn = container.querySelector('button[aria-label="Hub"]')
    expect(hubBtn).not.toBeNull()
    expect(hubBtn!.classList.contains('sidebar__item--active')).toBe(true)
  })

  it('Hub button has sidebar__item--active only when active (not when other panel is active)', () => {
    // Settings panel active — Hub button must NOT be active
    ;({ container, root } = renderSidebar({ activePanel: '__settings__' }))
    const hubBtn = container.querySelector('button[aria-label="Hub"]')
    expect(hubBtn!.classList.contains('sidebar__item--active')).toBe(false)
  })
})

describe('Sidebar icon position stability (SBR-02)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    localStorage.clear()
  })

  it('all sidebar__icon elements exist in both expanded and collapsed states', () => {
    // Phase 138: 1 toggle + 3 nav items = 4 sidebar__icon SVGs total
    ;({ container, root } = renderSidebar())
    const expandedIcons = container.querySelectorAll('svg.sidebar__icon')
    expect(expandedIcons.length).toBeGreaterThanOrEqual(4)

    const toggleBtn = container.querySelector('.sidebar__toggle') as HTMLButtonElement
    act(() => { toggleBtn.click() })

    // After collapse, same icon count — icons are always in the DOM
    const collapsedIcons = container.querySelectorAll('svg.sidebar__icon')
    expect(collapsedIcons.length).toBe(expandedIcons.length)
  })

  it('sidebar__toggle contains exactly one sidebar__icon SVG', () => {
    ;({ container, root } = renderSidebar())
    // The toggle button uses the same .sidebar__icon class — it participates
    // in the unified icon-alignment system (14px margin, 24px center)
    const toggleIcons = container.querySelectorAll('.sidebar__toggle svg.sidebar__icon')
    expect(toggleIcons.length).toBe(1)
  })

  it('CSS contract: .sidebar__icon has margin: 0 14px (fixed 48px icon slot — SBR-02)', () => {
    // Verify the stylesheet declares the fixed-width icon slot.
    // Math: 14px left + 20px icon + 14px right = 48px slot.
    // Icon center = 14 + 10 = 24px in both expanded and collapsed states.
    expect(cssRaw).toMatch(/\.sidebar__icon\s*\{[^}]*margin:\s*0\s+14px/)
  })

  it('CSS contract: .sidebar--collapsed .sidebar__item justify-content override is removed (anti-regression)', () => {
    // The Phase 63 rule `.sidebar--collapsed .sidebar__item { justify-content: center }`
    // caused the 6px icon shift bug. It must NOT be present after the Phase 70 fix.
    // If this test fails, the override has been re-introduced.
    expect(cssRaw).not.toMatch(/\.sidebar--collapsed\s+\.sidebar__item\s*\{[^}]*justify-content\s*:\s*center/)
  })
})

// ============================================================
// NAV-05 positive render contract — 3 items with groups present (GAP-03)
// ============================================================

describe('NAV-05 positive render contract — 3 items with groups present (GAP-03)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    localStorage.clear()
  })

  it('still renders exactly 3 sidebar__item buttons when groupDefs are present (Home, Hub, Settings)', () => {
    // sidebar is expanded by default (no sidebar-collapsed in localStorage)
    // showGroupList = effectiveExpanded && groupDefs.length > 0 — group items render in ul.sidebar__group-list,
    // NOT as button.sidebar__item, so the top-level nav count must stay at exactly 3.
    const groupDefs = [
      makeGroup({ id: 'grp-1', name: 'Alpha' }),
      makeGroup({ id: 'grp-2', name: 'Beta' }),
    ]
    ;({ container, root } = renderSidebar({ groupDefs }))
    const items = container.querySelectorAll('button.sidebar__item')
    expect(items.length).toBe(3)
  })

  it('renders group entries as .sidebar__group-item elements (not as top-level nav items)', () => {
    const groupDefs = [
      makeGroup({ id: 'grp-1', name: 'Alpha' }),
      makeGroup({ id: 'grp-2', name: 'Beta' }),
    ]
    ;({ container, root } = renderSidebar({ groupDefs }))
    // Groups appear inside ul.sidebar__group-list as li.sidebar__group-item
    const groupItems = container.querySelectorAll('.sidebar__group-item')
    // 2 named groups + "All" sentinel = 3 total group items
    expect(groupItems.length).toBe(groupDefs.length + 1)
  })

  it('group item count matches groupDefs.length (named groups only, excluding "All")', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    ;({ container, root } = renderSidebar({ groupDefs }))
    // Named group items: one per groupDef. "All" is also a .sidebar__group-item but its
    // li.sidebar__group-item is also in the list; total = groupDefs.length + 1.
    // Here we assert the list renders groupDefs.length named items by checking total = N+1.
    const groupItems = container.querySelectorAll('.sidebar__group-item')
    expect(groupItems.length).toBe(groupDefs.length + 1) // +1 for "All"
  })

  it('no Sessions or Remote buttons appear regardless of groupDefs', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    ;({ container, root } = renderSidebar({ groupDefs }))
    expect(container.querySelector('button[aria-label="Sessions"]')).toBeNull()
    expect(container.querySelector('button[aria-label="Remote"]')).toBeNull()
  })
})

// ============================================================
// POL-05: Sidebar group sub-list (RED — fails until POL-05 lands)
// ============================================================

describe('Sidebar group sub-list (POL-05) — RED', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    localStorage.clear()
  })

  // (a) renders a group sub-list when groupDefs is non-empty
  it('POL-05 RED: renders ul.sidebar__group-list when groupDefs.length > 0', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    ;({ container, root } = renderSidebar({ groupDefs }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
  })

  // (a) "All" item is always present when groupDefs.length > 0
  it('POL-05 RED: renders an "All" item in the group sub-list', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    ;({ container, root } = renderSidebar({ groupDefs }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
    expect(list!.textContent).toContain('All')
  })

  // (a) one button per group plus one for "All"
  it('POL-05 RED: renders one group item button per groupDef plus an All item', () => {
    const groupDefs = [
      makeGroup({ id: 'grp-1', name: 'Alpha' }),
      makeGroup({ id: 'grp-2', name: 'Beta' }),
    ]
    ;({ container, root } = renderSidebar({ groupDefs }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
    // 1 All + 2 groups = 3 items
    const items = list!.querySelectorAll('li')
    expect(items.length).toBe(3)
  })

  // (b) aria-pressed reflects activeGroupId
  it('POL-05 RED: group item button has aria-pressed=true when activeGroupId matches', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    ;({ container, root } = renderSidebar({ groupDefs, activeGroupId: 'grp-1' }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
    // Find the button for Alpha
    const buttons = list!.querySelectorAll('button')
    const alphaBtn = Array.from(buttons).find(b => b.textContent?.includes('Alpha'))
    expect(alphaBtn).not.toBeUndefined()
    expect(alphaBtn!.getAttribute('aria-pressed')).toBe('true')
  })

  it('POL-05 RED: group item button has aria-pressed=false when not active', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    ;({ container, root } = renderSidebar({ groupDefs, activeGroupId: null }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
    const buttons = list!.querySelectorAll('button')
    const alphaBtn = Array.from(buttons).find(b => b.textContent?.includes('Alpha'))
    expect(alphaBtn).not.toBeUndefined()
    expect(alphaBtn!.getAttribute('aria-pressed')).toBe('false')
  })

  // (b) group item button has aria-label containing running/total sessions
  it('POL-05 RED: group item button has aria-label with running/total session count', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    const groupCounts = { 'grp-1': { running: 2, total: 3, attention: 0, waiting: 0 } }
    ;({ container, root } = renderSidebar({ groupDefs, groupCounts }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
    const buttons = list!.querySelectorAll('button')
    const alphaBtn = Array.from(buttons).find(b =>
      b.getAttribute('aria-label')?.includes('Alpha') || b.textContent?.includes('Alpha')
    )
    expect(alphaBtn).not.toBeUndefined()
    // aria-label should include session counts
    const ariaLabel = alphaBtn!.getAttribute('aria-label') ?? ''
    expect(ariaLabel).toMatch(/\d+\/\d+/)
  })

  // (c) clicking a group item calls onGroupSelect(g.id) AND onOpenHub
  it('POL-05 RED: clicking a group item calls onGroupSelect(id) and onOpenHub', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    const onGroupSelect = vi.fn()
    const onOpenHub = vi.fn()
    ;({ container, root } = renderSidebar({ groupDefs, onGroupSelect, onOpenHub }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
    const buttons = list!.querySelectorAll('button')
    const alphaBtn = Array.from(buttons).find(b => b.textContent?.includes('Alpha')) as HTMLButtonElement
    expect(alphaBtn).not.toBeUndefined()
    act(() => { alphaBtn.click() })
    expect(onGroupSelect).toHaveBeenCalledWith('grp-1')
    expect(onOpenHub).toHaveBeenCalled()
  })

  // (d) clicking "All" calls onGroupSelect(null)
  it('POL-05 RED: clicking "All" calls onGroupSelect(null)', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    const onGroupSelect = vi.fn()
    ;({ container, root } = renderSidebar({ groupDefs, onGroupSelect }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
    const buttons = list!.querySelectorAll('button')
    const allBtn = Array.from(buttons).find(b => b.textContent?.trim() === 'All') as HTMLButtonElement
    expect(allBtn).not.toBeUndefined()
    act(() => { allBtn.click() })
    expect(onGroupSelect).toHaveBeenCalledWith(null)
  })

  // (e) drop event on a group <li> reads dataTransfer and calls onDropOnGroup(g.id, key)
  it('POL-05 RED: drop on a group li calls onDropOnGroup(groupId, key)', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    const onDropOnGroup = vi.fn()
    ;({ container, root } = renderSidebar({ groupDefs, onDropOnGroup }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
    const items = list!.querySelectorAll('li')
    // Find Alpha item — second <li> after All
    const alphaLi = Array.from(items).find(li => li.textContent?.includes('Alpha')) as HTMLElement
    expect(alphaLi).not.toBeUndefined()

    const dropKey = 'MySession:::/home/user/project'
    act(() => {
      const dropEvent = new Event('drop', { bubbles: true, cancelable: true })
      Object.defineProperty(dropEvent, 'dataTransfer', {
        value: { getData: (type: string) => type === 'text/plain' ? dropKey : '' },
      })
      alphaLi.dispatchEvent(dropEvent)
    })

    expect(onDropOnGroup).toHaveBeenCalledWith('grp-1', dropKey)
  })

  // (f) dropping on "All" does NOT call onDropOnGroup (id===null guard)
  it('POL-05 RED: drop on "All" li does NOT call onDropOnGroup', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    const onDropOnGroup = vi.fn()
    ;({ container, root } = renderSidebar({ groupDefs, onDropOnGroup }))
    const list = container.querySelector('ul.sidebar__group-list')
    expect(list).not.toBeNull()
    const items = list!.querySelectorAll('li')
    // "All" is the first <li>
    const allLi = Array.from(items).find(li => li.textContent?.trim() === 'All') as HTMLElement
    expect(allLi).not.toBeUndefined()

    const dropKey = 'MySession:::/home/user/project'
    act(() => {
      const dropEvent = new Event('drop', { bubbles: true, cancelable: true })
      Object.defineProperty(dropEvent, 'dataTransfer', {
        value: { getData: (type: string) => type === 'text/plain' ? dropKey : '' },
      })
      allLi.dispatchEvent(dropEvent)
    })

    expect(onDropOnGroup).not.toHaveBeenCalled()
  })

  // (g) inline "New group" creation: clicking shows input; Enter with text calls onCreateGroup
  it('POL-05 RED: clicking new-group affordance reveals an input', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    ;({ container, root } = renderSidebar({ groupDefs }))
    // Find the "New group" button (any button with "New group" text near the sub-list)
    const newGroupBtn = Array.from(container.querySelectorAll('button')).find(
      b => b.textContent?.trim() === 'New group',
    ) as HTMLButtonElement
    expect(newGroupBtn).not.toBeUndefined()
    act(() => { newGroupBtn.click() })
    const input = container.querySelector('input')
    expect(input).not.toBeNull()
  })

  it('POL-05 RED: Enter with trimmed text calls onCreateGroup', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    const onCreateGroup = vi.fn()
    ;({ container, root } = renderSidebar({ groupDefs, onCreateGroup }))
    const newGroupBtn = Array.from(container.querySelectorAll('button')).find(
      b => b.textContent?.trim() === 'New group',
    ) as HTMLButtonElement
    expect(newGroupBtn).not.toBeUndefined()
    act(() => { newGroupBtn.click() })
    const input = container.querySelector('input') as HTMLInputElement
    expect(input).not.toBeNull()
    act(() => {
      Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')?.set?.call(input, 'My New Group')
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    act(() => {
      input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    })
    expect(onCreateGroup).toHaveBeenCalledWith('My New Group')
  })

  // Group sub-list is hidden when sidebar is collapsed
  it('POL-05 RED: group sub-list is not visible when sidebar is collapsed', () => {
    const groupDefs = [makeGroup({ id: 'grp-1', name: 'Alpha' })]
    ;({ container, root } = renderSidebar({ groupDefs }))
    const toggleBtn = container.querySelector('.sidebar__toggle') as HTMLButtonElement
    act(() => { toggleBtn.click() })
    // When collapsed, group sub-list should not be rendered or visible
    const list = container.querySelector('ul.sidebar__group-list')
    // Either absent from DOM or has display:none — either is valid; assert absence from DOM
    expect(list).toBeNull()
  })
})

// ============================================================
// CSS source-gate: POL-05 sidebar__group-* rules
// ============================================================

describe('CSS source-gate: POL-05 sidebar__group-* rules (RED)', () => {
  it('POL-05 RED: style.css contains .sidebar__group-list rule', () => {
    expect(cssRaw).toContain('.sidebar__group-list')
  })

  it('POL-05 RED: style.css contains .sidebar__group-item rule', () => {
    expect(cssRaw).toContain('.sidebar__group-item')
  })
})

// ============================================================
// Source-gate: POL-04 — pendingThemeRef in TerminalPanel.tsx (RED)
// ============================================================

describe('Source-gate: POL-04 pendingThemeRef in TerminalPanel.tsx (RED)', () => {
  it('POL-04 RED: TerminalPanel.tsx contains pendingThemeRef', () => {
    expect(terminalPanelRaw).toContain('pendingThemeRef')
  })

  it('POL-04 RED: fitTerminal appears after clearTextureAtlas in TerminalPanel.tsx', () => {
    // clearTextureAtlas must come before fitTerminal in the repaint path
    const clearIdx = terminalPanelRaw.indexOf('clearTextureAtlas')
    const fitIdx = terminalPanelRaw.indexOf('fitTerminal', clearIdx)
    expect(clearIdx).toBeGreaterThan(-1)
    expect(fitIdx).toBeGreaterThan(clearIdx)
  })
})

// ============================================================
// Source-gate: POL-03 — PlusIcon in HubFilterBar.tsx and HubEmptyState.tsx (RED)
// ============================================================

describe('Source-gate: POL-03 PlusIcon in HubFilterBar.tsx and HubEmptyState.tsx (RED)', () => {
  it('POL-03 RED: HubFilterBar.tsx contains PlusIcon', () => {
    expect(hubFilterBarRaw).toContain('PlusIcon')
  })

  it('POL-03 RED: HubEmptyState.tsx contains PlusIcon', () => {
    expect(hubEmptyStateRaw).toContain('PlusIcon')
  })
})
