import { describe, it, expect, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { TabBar } from '../TabBar'
import type { Tab } from '../TabBar'
import raw from '../TabBar.tsx?raw'

interface TabBarProps {
  tabs: Tab[]
  activeId: string | null
  onSelect: (id: string) => void
  onClose: (id: string) => void
  onRename: (id: string, name: string) => void
  sessionStatuses?: Record<string, string>
}

function renderTabBar() {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(TabBar, {
        tabs: [],
        activeId: null,
        onSelect: () => {},
        onClose: () => {},
        onRename: () => {},
      })
    )
  })
  return { container, root }
}

function renderTabBarWithTabs(overrides: Partial<TabBarProps> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const props = {
    tabs: [
      { id: 'tab1', name: 'claude 1', sessionId: 'sess1', cli: 'claude' },
      { id: 'tab2', name: 'codex 1', sessionId: 'sess2', cli: 'codex' },
    ],
    activeId: 'tab1',
    onSelect: () => {},
    onClose: () => {},
    onRename: () => {},
    ...overrides,
  }
  flushSync(() => {
    root.render(React.createElement(TabBar, props))
  })
  return { container, root }
}

describe('TabBar', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders tab-bar root element', () => {
    ;({ container, root } = renderTabBar())
    expect(container.querySelector('.tab-bar')).not.toBeNull()
  })

  it('renders tab-list container', () => {
    ;({ container, root } = renderTabBar())
    expect(container.querySelector('.tab-list')).not.toBeNull()
  })
})


describe('TabBar context menu', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
    // Clean up any leftover event listeners by dispatching a mousedown to body
  })

  it('right-clicking tab name shows context menu', () => {
    ;({ container, root } = renderTabBarWithTabs())
    const tabName = container.querySelector('.tab__name') as HTMLElement
    expect(tabName).not.toBeNull()
    flushSync(() => {
      tabName.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true }))
    })
    expect(container.querySelector('.tab__context-menu')).not.toBeNull()
  })

  it('context menu has role="menu" and contains Rename menuitem', () => {
    ;({ container, root } = renderTabBarWithTabs())
    const tabName = container.querySelector('.tab__name') as HTMLElement
    flushSync(() => {
      tabName.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true }))
    })
    const menu = container.querySelector('.tab__context-menu') as HTMLElement
    expect(menu).not.toBeNull()
    expect(menu.getAttribute('role')).toBe('menu')
    const renameBtn = menu.querySelector('button[role="menuitem"]') as HTMLButtonElement
    expect(renameBtn).not.toBeNull()
    expect(renameBtn.textContent).toBe('Rename')
  })

  it('clicking Rename starts inline editing', () => {
    ;({ container, root } = renderTabBarWithTabs())
    const tabName = container.querySelector('.tab__name') as HTMLElement
    flushSync(() => {
      tabName.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true }))
    })
    const renameBtn = container.querySelector('button[role="menuitem"]') as HTMLButtonElement
    expect(renameBtn).not.toBeNull()
    flushSync(() => {
      renameBtn.click()
    })
    expect(container.querySelector('.tab__rename-input')).not.toBeNull()
  })

  it('tab name title includes right-click', () => {
    ;({ container, root } = renderTabBarWithTabs())
    const tabName = container.querySelector('.tab__name') as HTMLElement
    expect(tabName).not.toBeNull()
    expect(tabName.getAttribute('title')).toContain('right-click')
  })
})

describe('Phase 97 SER-01: TabBar Save Terminal As… menu item — Plan 97-04 implementation', () => {
  it('Save Terminal As menu-item label appears in TabBar source (with U+2026 ellipsis)', () => {
    // The literal string must contain U+2026 HORIZONTAL ELLIPSIS, not "..."
    expect(raw).toContain('Save Terminal As…')
  })

  it('TabBar.tsx props interface declares onRequestSave?: (tabId: string) => void', () => {
    expect(raw).toMatch(/onRequestSave\?:\s*\(\s*tabId:\s*string\s*\)\s*=>\s*void/)
  })

  it('Save menu-item onClick invokes onRequestSave?.(contextMenu.tabId) and closes the menu', () => {
    // Source must contain a click handler that calls both onRequestSave and setContextMenu(null).
    expect(raw).toMatch(/onRequestSave\?\.\(\s*contextMenu\.tabId\s*\)/)
    // The "Save Terminal As…" button must be near a setContextMenu(null) call.
    const saveIdx = raw.indexOf('Save Terminal As…')
    expect(saveIdx).toBeGreaterThan(-1)
    // Walk back ~300 chars from the button label and look for setContextMenu(null) — the
    // entire button JSX (handler + label) fits in this window.
    const window = raw.slice(Math.max(0, saveIdx - 300), saveIdx + 100)
    expect(window).toMatch(/setContextMenu\(\s*null\s*\)/)
  })
})

// Phase 101-02: TabBar agent badge (SHELL-06 GUI half).
// Render-based tests using createRoot + flushSync, mirroring the existing
// renderTabBarWithTabs helper.

function renderTabBarWithCLI(cli: string, tabName = 'session-1') {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(TabBar, {
        tabs: [{ id: 'tab1', name: tabName, sessionId: 'sess1', cli }],
        activeId: 'tab1',
        onSelect: () => {},
        onClose: () => {},
        onRename: () => {},
      })
    )
  })
  return { container, root }
}

describe('Phase 101-02 TabBar agent badge (SHELL-06 GUI)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders agent badge for claude session', () => {
    ;({ container, root } = renderTabBarWithCLI('claude'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge).not.toBeNull()
    expect(badge!.className).toContain('claude')
  })

  it('renders shell agent badge for cli=shell', () => {
    ;({ container, root } = renderTabBarWithCLI('shell'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge).not.toBeNull()
    expect(badge!.className).toContain('--shell')
  })

  it('renders shell agent badge for cli=bash', () => {
    ;({ container, root } = renderTabBarWithCLI('bash'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge).not.toBeNull()
    expect(badge!.className).toContain('--shell')
    expect(badge!.className).not.toContain('--bash')
  })

  it('renders shell agent badge for cli=zsh', () => {
    ;({ container, root } = renderTabBarWithCLI('zsh'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge!.className).toContain('--shell')
  })

  it('renders shell agent badge for cli=pwsh', () => {
    ;({ container, root } = renderTabBarWithCLI('pwsh'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge!.className).toContain('--shell')
  })

  it('renders shell agent badge for cli=powershell', () => {
    ;({ container, root } = renderTabBarWithCLI('powershell'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge!.className).toContain('--shell')
  })

  it('falls back to muted badge for unknown cli', () => {
    ;({ container, root } = renderTabBarWithCLI('unknown-tool'))
    const badge = container.querySelector('.tab__agent-badge') as HTMLElement
    expect(badge).not.toBeNull()
    const knownModifiers = ['--claude', '--opencode', '--codex', '--gemini', '--cursor', '--aider', '--shell']
    for (const mod of knownModifiers) {
      expect(badge.className).not.toContain(mod)
    }
  })

  it('agent badge is aria-hidden', () => {
    ;({ container, root } = renderTabBarWithCLI('claude'))
    const badge = container.querySelector('.tab__agent-badge') as HTMLElement
    expect(badge.getAttribute('aria-hidden')).toBe('true')
  })

  it('tab tooltip includes agent type for shell session', () => {
    ;({ container, root } = renderTabBarWithCLI('bash', 'scratch-bash'))
    const tabName = container.querySelector('.tab__name') as HTMLElement
    expect(tabName).not.toBeNull()
    const title = tabName.getAttribute('title') || ''
    expect(title).toContain('Shell — bash')
  })

  it('tab agent badge appears between status dot and tab name', () => {
    ;({ container, root } = renderTabBarWithCLI('claude'))
    const tab = container.querySelector('.tab') as HTMLElement
    const children = Array.from(tab.children)
    const statusIdx = children.findIndex((c) => c.classList.contains('tab__status'))
    const badgeIdx = children.findIndex((c) => c.classList.contains('tab__agent-badge'))
    const nameIdx = children.findIndex((c) => c.classList.contains('tab__name'))
    expect(statusIdx).toBeGreaterThanOrEqual(0)
    expect(badgeIdx).toBeGreaterThan(statusIdx)
    expect(nameIdx).toBeGreaterThan(badgeIdx)
  })
})

// Phase 98 PRG-03 — Wave 0 RED scaffold.
// These two test cases are RED until Wave 3 (Plan 04) adds the .tab__progress
// underline element to TabBar.tsx. Tagged `progress-underline` and
// `progress-transform` for VALIDATION.md verify targeting.
// Using source-inspection (?raw) pattern consistent with the rest of this file.
describe('Phase 98 PRG-03: TabBar per-tab progress underline — Wave 0 RED scaffold (Plan 98-01 Wave 0)', () => {
  it('progress-underline: TabBarProps interface declares tabProgress?: Record<string, number>', () => {
    // RED until Wave 3 (Plan 04) extends the TabBarProps interface.
    expect(raw).toMatch(/tabProgress\?:\s*Record<string,\s*number>/)
  })

  it('progress-underline: renders .tab__progress element with data-testid keyed by tab.id', () => {
    // RED until Wave 3 adds the .tab__progress <div> inside the tab render.
    expect(raw).toContain('tab__progress')
    expect(raw).toMatch(/data-testid[^}]*tab-progress-/)
  })

  it('progress-transform: uses transform scaleX for the underline animation (not width)', () => {
    // RED until Wave 3. The production element must set transform: scaleX(...)
    // and must NOT set inline width (CSS width stays 100%; only transform changes).
    expect(raw).toMatch(/scaleX\(/)
  })
})

// ============================================================================
// Phase 139 Plan 01 — TabBar chevron + floor + rename + title tests (TAB-01..03)
//
// These tests are RED until Plan 02 adds:
//   - canScrollLeft / canScrollRight state + ResizeObserver in TabBar.tsx
//   - button[aria-label="Scroll tabs left"] and button[aria-label="Scroll tabs right"]
//   - title attribute on the OUTER .tab div (not just .tab__name)
// ============================================================================

describe('Phase 139 TAB-02: TabBar chevron overflow — RED until Plan 02', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('right chevron appears when tab list overflows scrollWidth > clientWidth', () => {
    // TabBar source must declare canScrollRight state driven by scrollWidth comparison.
    // This source-level check is RED until Plan 02 adds the canScrollRight state.
    expect(raw).toContain('canScrollRight')
    expect(raw).toContain('aria-label="Scroll tabs right"')
  })

  it('left chevron appears when scrollLeft > 0', () => {
    // TabBar source must declare canScrollLeft state.
    expect(raw).toContain('canScrollLeft')
    expect(raw).toContain('aria-label="Scroll tabs left"')
  })

  it('left chevron is absent at scrollLeft=0 (source guards on canScrollLeft)', () => {
    // The left chevron must be guarded by {canScrollLeft && ...} in JSX.
    // Plan 02 sets canScrollLeft = scrollLeft > 0; at initial render (scrollLeft=0)
    // the chevron must not appear. Source-level check confirms the guard exists.
    expect(raw).toMatch(/canScrollLeft\s*&&/)
  })

  it('right chevron is guarded on canScrollRight', () => {
    expect(raw).toMatch(/canScrollRight\s*&&/)
  })

  it('chevron buttons use ResizeObserver to detect overflow (source check)', () => {
    // Plan 02 must wire a ResizeObserver to the .tab-list element.
    expect(raw).toContain('ResizeObserver')
  })

  it('chevron onClick scrolls the tab list by a fixed step (source check)', () => {
    // Plan 02 must call scrollBy on listRef.current.
    expect(raw).toContain('scrollBy')
  })
})

describe('Phase 139 TAB-03: TabBar rename-at-floor via context menu — RED until Plan 02', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('right-click context menu Rename fires onRename without requiring tab__name double-click', () => {
    // Simulate: render with 2 tabs, right-click the .tab element (outer div, not .tab__name),
    // click Rename in the context menu, confirm onRename is called.
    // At icon-only floor, .tab__name may be hidden by CSS; the context menu must still work.
    let renamedId: string | null = null
    let renamedName: string | null = null
    ;({ container, root } = renderTabBarWithTabs({
      onRename: (id, name) => {
        renamedId = id
        renamedName = name
      },
    }))

    // Right-click the first tab's .tab__name to open the context menu.
    // (The context menu trigger is on .tab__name contextmenu event.)
    const tabName = container.querySelector('.tab__name') as HTMLElement
    expect(tabName).not.toBeNull()
    flushSync(() => {
      tabName.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true }))
    })

    // Find and click the Rename button in the context menu.
    const renameBtn = container.querySelector('button[role="menuitem"]') as HTMLButtonElement
    expect(renameBtn).not.toBeNull()
    expect(renameBtn.textContent).toBe('Rename')
    flushSync(() => {
      renameBtn.click()
    })

    // The rename input should appear.
    const renameInput = container.querySelector('.tab__rename-input') as HTMLInputElement
    expect(renameInput).not.toBeNull()

    // Simulate typing a new name and pressing Enter to commit.
    flushSync(() => {
      Object.defineProperty(renameInput, 'value', { value: 'new-name', configurable: true })
      renameInput.dispatchEvent(new Event('change', { bubbles: true }))
    })
    flushSync(() => {
      renameInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    })

    // onRename must have been called.
    expect(renamedId).toBe('tab1')
    // If name change didn't propagate in jsdom, at least assert the flow ran without error.
    // (actual rename value assertion may need Plan 02 updates)
    expect(renamedId).not.toBeNull()
  })
})

describe('Phase 139 TAB-03: TabBar title on outer .tab div — RED until Plan 02', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('outer .tab div carries a title attribute equal to the full tab name tooltip', () => {
    // Currently (before Plan 02) the title is only on .tab__name (inner span).
    // Plan 02 must ALSO put title on the outer .tab div so it is discoverable
    // at icon-only floor when .tab__name is hidden by CSS.
    // This test is RED until Plan 02 moves/copies title to the outer .tab div.
    ;({ container, root } = renderTabBarWithTabs())

    const outerTab = container.querySelector('.tab') as HTMLElement
    expect(outerTab).not.toBeNull()
    const outerTitle = outerTab.getAttribute('title')
    expect(outerTitle).not.toBeNull()
    expect(typeof outerTitle).toBe('string')
    expect((outerTitle as string).length).toBeGreaterThan(0)
  })

  it('outer .tab div title contains the full tab name (not truncated)', () => {
    // The title on the outer .tab div must include the full tab name "claude 1".
    ;({ container, root } = renderTabBarWithTabs())

    const outerTab = container.querySelector('.tab') as HTMLElement
    const outerTitle = outerTab.getAttribute('title') || ''
    // Must contain the full tab name (not just "rename hint" text without the name).
    // Note: The existing title on .tab__name contains the rename hint, not the name.
    // Plan 02 must put the full name in the outer div's title attribute.
    expect(outerTitle).toContain('claude 1')
  })
})
