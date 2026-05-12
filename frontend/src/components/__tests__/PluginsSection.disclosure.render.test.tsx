/**
 * Phase 99 Plan 06 — gap-closure render test for disclosure checkbox class hygiene.
 *
 * Root cause (see .planning/debug/99-disclosure-checkboxes-missing.md):
 * The 6 disclosure checkbox <input> elements in PluginsSection.tsx reused
 * className="settings-panel__toggle-input", a Phase-82 iOS-toggle-switch-anchor
 * class that style.css:586-592 hides globally (position:absolute; width:1px;
 * height:1px; opacity:0; pointer-events:none). The main plugin rows pair this
 * hidden input with visible track/thumb spans — the disclosure helpers did not,
 * so the checkboxes were invisible. This test enforces the post-fix invariant.
 *
 * Test strategy: jsdom + ReactDOM.createRoot (NOT @testing-library/react — not
 * in devDependencies; see TerminalPanel.web-links.test.tsx convention comment).
 * Wails-generated daemon.* constructors fail under jsdom, so both
 * wailsjs/go/main/App and wailsjs/go/models are mocked via vi.mock.
 *
 * This file is a peer of PluginsSection.disclosure.test.tsx (source-inspection),
 * NOT a replacement. The source-inspection file covers PUI-03 / PUI-04 structure;
 * this file covers real-DOM class hygiene — a test strategy gap that allowed
 * the invisible-checkbox regression to pass 33/33 source-inspection tests.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createRoot, type Root } from 'react-dom/client'
import { act } from 'react'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetPluginSettings: vi.fn(async () => ({
    webgl: true,
    unicode11: false,
    search: true,
    webLinks: true,
    image: false,
    serialize: false,
    clipboard: false,
    progress: false,
    searchConfig: { regex: false, caseSensitive: false, wholeWord: false },
    webLinksConfig: {
      modifier: 'platform',
      confirmOSC8: true,
      confirmIDN: true,
      confirmTyposquat: true,
    },
    imageConfig: { storageLimit: 16 },
  })),
  SetPluginSettings: vi.fn(async () => undefined),
  SetSearchConfig: vi.fn(async () => undefined),
  SetWebLinksConfig: vi.fn(async () => undefined),
  SetImageConfig: vi.fn(async () => undefined),
}))

vi.mock('../../wailsjs/go/models', () => {
  class Stub {
    constructor(o: Record<string, unknown> = {}) {
      Object.assign(this, o)
    }
  }
  return {
    daemon: {
      PluginSettings: Stub,
      SearchConfig: Stub,
      WebLinksConfig: Stub,
      ImageConfig: Stub,
    },
  }
})

import { PluginsSection } from '../PluginsSection'
import React from 'react'

describe('Phase 99 gap-closure: disclosure checkboxes render with class hygiene', () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(async () => {
    container = document.createElement('div')
    document.body.appendChild(container)
    await act(async () => {
      root = createRoot(container)
      root.render(React.createElement(PluginsSection))
    })
    // Allow GetPluginSettings promise to resolve and React to re-render
    // with pluginsLoaded === true
    await act(async () => {
      await new Promise<void>((r) => setTimeout(r, 0))
    })
    // Force-open all <details> elements so disclosure markup is accessible
    container.querySelectorAll('details').forEach((d) => d.setAttribute('open', ''))
    await act(async () => {})
  })

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders six <input type="checkbox"> elements inside .settings-panel__details blocks', async () => {
    const disclosureCheckboxes = container.querySelectorAll(
      '.settings-panel__details input[type="checkbox"]',
    )
    expect(disclosureCheckboxes.length).toBe(6)
  })

  it('NONE of the disclosure checkboxes carry the .settings-panel__toggle-input hidden-anchor class', async () => {
    // This is the load-bearing assertion.
    // EXPECTED TO FAIL on pre-patch source (6 inputs carry the hidden-toggle class)
    // EXPECTED TO PASS on post-patch source (0 inputs carry it)
    const disclosureCheckboxes = container.querySelectorAll(
      '.settings-panel__details input[type="checkbox"]',
    )
    const allClean = Array.from(disclosureCheckboxes).every(
      (el) => !el.classList.contains('settings-panel__toggle-input'),
    )
    expect(allClean).toBe(true)
  })

  it('the renderRow main toggles (outside .settings-panel__details) DO carry .settings-panel__toggle-input', async () => {
    // Differential guardrail: assert the iOS-pill anchor pattern is still in
    // use for the 8 main plugin rows. This prevents an over-zealous future
    // refactor from stripping the class from renderRow.
    const allToggleInputs = container.querySelectorAll(
      'input[type="checkbox"].settings-panel__toggle-input',
    )
    const mainRowOnly = Array.from(allToggleInputs).filter(
      (el) => !el.closest('.settings-panel__details'),
    )
    expect(mainRowOnly.length).toBe(8)
  })
})
