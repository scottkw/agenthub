import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { Sidebar } from '../Sidebar'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// CSS source for contract tests (jsdom has no layout engine — we inspect raw CSS text)
// This is the established project pattern from SettingsTab, WelcomeTab, TerminalPanel tests.
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

// Helper to render Sidebar with default no-op props
function renderSidebar(overrides: Partial<Parameters<typeof Sidebar>[0]> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const defaultProps = {
    onOpenDaemonManager: vi.fn(),
    onOpenRemoteSessions: vi.fn(),
    onAdd: vi.fn(),
    onSettings: vi.fn(),
    onHome: vi.fn(),
    onOpenHub: vi.fn(),
  }
  act(() => {
    root.render(<Sidebar {...defaultProps} {...overrides} />)
  })
  return { container, root, ...defaultProps }
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

  it('renders a Sessions sidebar__item button', () => {
    ;({ container, root } = renderSidebar())
    const sessionsBtn = container.querySelector('button[aria-label="Sessions"]')
    expect(sessionsBtn).not.toBeNull()
    expect(sessionsBtn!.classList.contains('sidebar__item')).toBe(true)
  })

  it('renders "New Session" label and aria-label for the add button (UI-01)', () => {
    ;({ container, root } = renderSidebar())
    const addBtn = container.querySelector('button[aria-label="New Session"]')
    expect(addBtn).not.toBeNull()
    const label = addBtn!.querySelector('.sidebar__label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toBe('New Session')
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

  it('all 6 sidebar items remain in DOM when collapsed', () => {
    ;({ container, root } = renderSidebar())
    const toggleBtn = container.querySelector('.sidebar__toggle') as HTMLButtonElement
    act(() => { toggleBtn.click() })
    const items = container.querySelectorAll('.sidebar__item')
    expect(items.length).toBe(6)
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

  it('Sessions button (aria-label="Sessions") contains an SVG element (ServerStackIcon, not Unicode text)', () => {
    ;({ container, root } = renderSidebar())
    const sessionsBtn = container.querySelector('button[aria-label="Sessions"]')
    expect(sessionsBtn).not.toBeNull()
    const svg = sessionsBtn!.querySelector('svg')
    expect(svg).not.toBeNull()
    // Verify it uses SVG (Heroicon), not Unicode text characters
    const textContent = sessionsBtn!.textContent || ''
    // The button should not rely on Unicode symbols/emoji for the icon
    expect(textContent.trim()).not.toMatch(/^[\u2600-\u27BF\uD83C-\uDBFF\uDC00-\uDFFF]/)
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
    ;({ container, root } = renderSidebar({ activePanel: '__daemon_manager__' }))
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
    // Sessions panel active — Hub button must NOT be active
    ;({ container, root } = renderSidebar({ activePanel: '__daemon_manager__' }))
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
    // 1 toggle + 6 nav items = 7 sidebar__icon SVGs total (Hub added in Phase 131)
    ;({ container, root } = renderSidebar())
    const expandedIcons = container.querySelectorAll('svg.sidebar__icon')
    expect(expandedIcons.length).toBeGreaterThanOrEqual(7)

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
