import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'
import settingsRaw from '../../components/SettingsTab.tsx?raw'
import pluginsRaw from '../../components/PluginsSection.tsx?raw'

const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

// Phase 104 Settings Hyperlinked Index — anchor IDs on section headers.
// SETUI-01/02 require deep-linkable anchors so the JumpBar and Search can
// scroll to a specific section by hash.

describe('Phase 104 SETUI-01: section header anchor IDs', () => {
  const sections = [
    { label: 'Terminal Plugins', id: 'settings-plugins', file: 'PluginsSection.tsx', raw: pluginsRaw },
    { label: 'Behavior', id: 'settings-behavior', file: 'SettingsTab.tsx', raw: settingsRaw },
    { label: 'Session Behavior', id: 'settings-session-behavior', file: 'SettingsTab.tsx', raw: settingsRaw },
    { label: 'Appearance', id: 'settings-appearance', file: 'SettingsTab.tsx', raw: settingsRaw },
    { label: 'Web Server', id: 'settings-web-server', file: 'SettingsTab.tsx', raw: settingsRaw },
    { label: 'Security', id: 'settings-security', file: 'SettingsTab.tsx', raw: settingsRaw },
    { label: 'Paths', id: 'settings-paths', file: 'SettingsTab.tsx', raw: settingsRaw },
  ]

  for (const s of sections) {
    it(`${s.file}: h3 "${s.label}" has id="${s.id}"`, () => {
      // Look for <h3 id="settings-foo"> with the matching label text.
      const re = new RegExp(`<h3[^>]*id="${s.id}"[^>]*>${s.label}<\\/h3>`)
      expect(s.raw).toMatch(re)
    })
  }
})

describe('Phase 104 SETUI-01: scroll-margin-top on section headers', () => {
  it('settings-panel__body h3 sets scroll-margin-top to clear the sticky jump-bar', () => {
    // Find the rule block for .settings-panel__body h3 and assert scroll-margin-top is present.
    const ruleMatch = cssRaw.match(/\.settings-panel__body\s+h3\s*\{[^}]*\}/)
    expect(ruleMatch).not.toBeNull()
    expect(ruleMatch![0]).toMatch(/scroll-margin-top:\s*\d+px/)
  })
})

describe('Phase 104 SETUI-01: SettingsJumpBar component', () => {
  it('exports a SettingsJumpBar function component', async () => {
    const mod = await import('../../components/SettingsJumpBar')
    expect(typeof mod.SettingsJumpBar).toBe('function')
  })

  it('renders 7 anchor links pointing at the section IDs', async () => {
    const React = await import('react')
    const { createRoot } = await import('react-dom/client')
    const { flushSync } = await import('react-dom')
    const { SettingsJumpBar } = await import('../../components/SettingsJumpBar')

    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    flushSync(() => {
      root.render(React.createElement(SettingsJumpBar))
    })

    const expectedTargets = [
      '#settings-plugins',
      '#settings-behavior',
      '#settings-session-behavior',
      '#settings-appearance',
      '#settings-web-server',
      '#settings-security',
      '#settings-paths',
    ]
    const links = container.querySelectorAll('a[href^="#settings-"]')
    expect(links.length).toBe(7)
    const hrefs = Array.from(links).map((a) => a.getAttribute('href'))
    for (const t of expectedTargets) {
      expect(hrefs).toContain(t)
    }

    root.unmount()
    container.remove()
  })

  it('JumpBar uses sticky positioning', () => {
    // The bar must remain visible while scrolling; assert CSS has a sticky rule.
    expect(cssRaw).toMatch(/\.settings-jump-bar\s*\{[^}]*position:\s*sticky[^}]*\}/)
  })
})

describe('Phase 104 SETUI-03: SettingsSearch component', () => {
  it('exports a SettingsSearch function component', async () => {
    const mod = await import('../../components/SettingsSearch')
    expect(typeof mod.SettingsSearch).toBe('function')
  })

  it('renders an input with a search-friendly placeholder', async () => {
    const React = await import('react')
    const { createRoot } = await import('react-dom/client')
    const { flushSync } = await import('react-dom')
    const { SettingsSearch } = await import('../../components/SettingsSearch')

    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    flushSync(() => {
      root.render(React.createElement(SettingsSearch))
    })

    const input = container.querySelector('input[type="text"], input[type="search"]') as HTMLInputElement | null
    expect(input).not.toBeNull()
    expect(input!.placeholder.toLowerCase()).toContain('search')

    root.unmount()
    container.remove()
  })

  function setReactInputValue(input: HTMLInputElement, value: string): void {
    // React 19 uses a native-value tracker; we must invoke the
    // prototype setter so its onChange handler observes the change.
    const proto = Object.getPrototypeOf(input)
    const desc = Object.getOwnPropertyDescriptor(proto, 'value')
    desc?.set?.call(input, value)
    input.dispatchEvent(new Event('input', { bubbles: true }))
  }

  it('filters labels by substring (case-insensitive) and shows matches as clickable results', async () => {
    const React = await import('react')
    const { createRoot } = await import('react-dom/client')
    const { flushSync } = await import('react-dom')
    const { SettingsSearch } = await import('../../components/SettingsSearch')

    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    flushSync(() => {
      root.render(React.createElement(SettingsSearch))
    })

    const input = container.querySelector('input') as HTMLInputElement
    flushSync(() => {
      setReactInputValue(input, 'sess')
    })

    // Expect a results list with at least one entry mentioning "Session Behavior".
    const results = container.querySelectorAll('.settings-search__result')
    expect(results.length).toBeGreaterThan(0)
    const texts = Array.from(results).map((r) => (r.textContent || '').toLowerCase())
    const hasSession = texts.some((t) => t.includes('session'))
    expect(hasSession).toBe(true)

    root.unmount()
    container.remove()
  })

  it('search results expose the target section anchor', async () => {
    const React = await import('react')
    const { createRoot } = await import('react-dom/client')
    const { flushSync } = await import('react-dom')
    const { SettingsSearch } = await import('../../components/SettingsSearch')

    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    flushSync(() => {
      root.render(React.createElement(SettingsSearch))
    })

    const input = container.querySelector('input') as HTMLInputElement
    flushSync(() => {
      setReactInputValue(input, 'appearance')
    })

    const results = container.querySelectorAll('.settings-search__result')
    expect(results.length).toBeGreaterThan(0)
    // Each result must carry a data-target attribute pointing at the anchor id.
    const targets = Array.from(results).map((r) => r.getAttribute('data-target'))
    expect(targets).toContain('settings-appearance')

    root.unmount()
    container.remove()
  })
})

describe('Phase 104 SETUI-01: JumpBar + Search are mounted in SettingsTab', () => {
  it('SettingsTab imports SettingsJumpBar', () => {
    expect(settingsRaw).toMatch(/from\s+['"]\.\/SettingsJumpBar['"]/)
  })

  it('SettingsTab imports SettingsSearch', () => {
    expect(settingsRaw).toMatch(/from\s+['"]\.\/SettingsSearch['"]/)
  })

  it('SettingsTab renders <SettingsJumpBar /> in the body', () => {
    expect(settingsRaw).toMatch(/<SettingsJumpBar\s*\/>/)
  })

  it('SettingsTab renders <SettingsSearch /> in the body', () => {
    expect(settingsRaw).toMatch(/<SettingsSearch\s*\/>/)
  })
})
