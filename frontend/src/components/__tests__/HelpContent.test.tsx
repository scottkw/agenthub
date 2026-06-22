// Phase 147-01: HelpContent test stubs (RED until Plan 03).
//
// Source-gates that HelpContent.tsx imports react-markdown and uses Markdown
// rendering. Also tests that external links call BrowserOpenURL (not <a href>),
// and that HelpContent contains no raw <a href literals. WILL fail until
// HelpContent.tsx is implemented in Plan 03 — intended RED state for Wave 0.

import { describe, it, expect, vi, afterEach } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// Mock the Wails runtime so tests that mount HelpContent can run in jsdom
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}))

// ============================================================
// Source-gate: HelpContent.tsx imports react-markdown (RED until Plan 03)
// ============================================================

describe('HelpContent source gate: react-markdown import (Phase 147)', () => {
  it('HelpContent.tsx imports from react-markdown', () => {
    // File does not exist yet — this test will fail with ENOENT in RED state
    const src = readFileSync(resolve(__dirname, '../HelpContent.tsx'), 'utf-8')
    expect(src).toContain('react-markdown')
  })

  it('HelpContent.tsx imports BrowserOpenURL from Wails runtime', () => {
    const src = readFileSync(resolve(__dirname, '../HelpContent.tsx'), 'utf-8')
    expect(src).toContain('BrowserOpenURL')
  })

  it('HelpContent.tsx contains no raw <a href literal (external links are BrowserOpenURL buttons)', () => {
    const src = readFileSync(resolve(__dirname, '../HelpContent.tsx'), 'utf-8')
    // Must NOT contain a raw anchor href in JSX — links must go through BrowserOpenURL
    expect(src).not.toContain('<a href')
  })
})

// ============================================================
// Render helper
// ============================================================

// Phase 147-02: Component now exists — use static import (GREEN state).
// The try/catch require() pattern used in RED state is incompatible with
// Vitest vmForks pool's CJS resolver (which does not try .tsx extensions).
import { HelpContent } from '../HelpContent'

function renderHelpContent(markdown: string) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<HelpContent markdown={markdown} />)
  })
  return { container, root }
}

let container: HTMLElement
let root: ReturnType<typeof createRoot>

afterEach(() => {
  try {
    root.unmount()
  } catch {
    // Ignore — RED state may not have rendered anything
  }
  container?.remove()
  vi.clearAllMocks()
})

// ============================================================
// External link → BrowserOpenURL (RED until Plan 03)
// ============================================================

describe('HelpContent: external links call BrowserOpenURL (Phase 147)', () => {
  it('clicking an external-link button calls BrowserOpenURL with the URL', async () => {
    const { BrowserOpenURL } = await import('../../wailsjs/wailsjs/runtime/runtime')
    const mockBrowserOpenURL = vi.mocked(BrowserOpenURL)
    mockBrowserOpenURL.mockClear()

    const md = '[GitHub Issues](https://github.com/scottkw/agenthub/issues)'
    ;({ container, root } = renderHelpContent(md))

    // HelpContent renders Markdown <a> elements as <button> elements that call BrowserOpenURL
    const linkBtn = container.querySelector('.help-content__external-link') as HTMLButtonElement
    expect(linkBtn).not.toBeNull()

    act(() => { linkBtn.click() })

    expect(mockBrowserOpenURL).toHaveBeenCalledWith('https://github.com/scottkw/agenthub/issues')
  })

  it('rendered output contains no raw <a href> elements (all links are buttons)', () => {
    const md = '[GitHub Issues](https://github.com/scottkw/agenthub/issues)'
    ;({ container, root } = renderHelpContent(md))
    // Raw anchor tags must not appear — BrowserOpenURL buttons are used instead
    const anchors = container.querySelectorAll('a[href]')
    expect(anchors.length).toBe(0)
  })
})
