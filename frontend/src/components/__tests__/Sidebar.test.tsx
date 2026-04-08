import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { Sidebar } from '../Sidebar'

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
