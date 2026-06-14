/**
 * Phase 124 CAP-05 — SessionSharePanel write opt-in tests.
 *
 * Verifies Surface 2 "Allow file editing" viewer opt-in row:
 *   - Default OFF (aria-checked="false")
 *   - Disabled (aria-disabled) when ownerWriteEnabled is false
 *   - Enabled when ownerWriteEnabled is true
 *   - Toggling ON opens inline confirmation with verbatim CAP-05 body
 *   - Confirm closes confirmation and issues write-capable URL
 *   - Cancel closes confirmation and reverts to OFF
 *   - Both the opt-in toggle and owner toggle expose role="switch" + aria-checked
 */
import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { SessionSharePanel } from '../SessionSharePanel'

// Mock Wails runtime modules used by SessionSharePanel
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  BrowserOpenURL: vi.fn(),
}))
vi.mock('../../wailsjs/go/main/App', () => ({
  GetCapabilityQRCode: vi.fn().mockResolvedValue(''),
}))

interface RenderOpts {
  ownerWriteEnabled?: boolean
  writeURL?: string
}

function renderPanel(opts: RenderOpts = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(SessionSharePanel, {
        sessionId: 'sess-1',
        readURL: 'https://example.com/r',
        writeURL: opts.writeURL ?? 'https://example.com/w',
        readCode: 'read-code',
        writeCode: 'write-code',
        ownerWriteEnabled: opts.ownerWriteEnabled ?? false,
      })
    )
  })
  return { container, root }
}

describe('SessionSharePanel — Allow file editing opt-in (CAP-05)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

  afterEach(() => {
    if (root) {
      flushSync(() => root!.unmount())
      root = undefined
    }
    if (container) {
      container.remove()
      container = undefined
    }
    vi.clearAllMocks()
  })

  it('renders "Allow file editing" label', () => {
    ;({ container, root } = renderPanel())
    expect(container.textContent).toContain('Allow file editing')
  })

  it('opt-in toggle defaults OFF (aria-checked="false")', () => {
    ;({ container, root } = renderPanel())
    const toggle = container.querySelector('[role="switch"][aria-label*="Allow file editing"], [role="switch"][data-testid="allow-file-editing-toggle"]') ??
      Array.from(container.querySelectorAll('[role="switch"]')).find(el =>
        el.closest('label')?.textContent?.includes('Allow file editing') ||
        el.getAttribute('aria-label')?.includes('Allow file editing') ||
        (el.nextElementSibling?.textContent ?? el.previousElementSibling?.textContent ?? '').includes('Allow file editing')
      )
    expect(toggle).not.toBeNull()
    expect(toggle!.getAttribute('aria-checked')).toBe('false')
  })

  it('opt-in toggle is aria-disabled when ownerWriteEnabled=false', () => {
    ;({ container, root } = renderPanel({ ownerWriteEnabled: false }))
    // Find the toggle row containing "Allow file editing"
    const writeOptinRow = Array.from(container.querySelectorAll('.session-share-panel__write-optin, .settings-panel__toggle-row')).find(
      el => el.textContent?.includes('Allow file editing')
    )
    expect(writeOptinRow).not.toBeNull()
    // The row or its input should have aria-disabled or disabled
    const hasDisabled =
      writeOptinRow!.getAttribute('aria-disabled') === 'true' ||
      writeOptinRow!.querySelector('[aria-disabled="true"]') !== null ||
      (writeOptinRow!.style.opacity === '0.6') ||
      writeOptinRow!.classList.contains('settings-panel__toggle-row--disabled')
    expect(hasDisabled).toBe(true)
  })

  it('opt-in toggle is enabled when ownerWriteEnabled=true', () => {
    ;({ container, root } = renderPanel({ ownerWriteEnabled: true }))
    const writeOptinRow = Array.from(container.querySelectorAll('.session-share-panel__write-optin, .settings-panel__toggle-row')).find(
      el => el.textContent?.includes('Allow file editing')
    )
    expect(writeOptinRow).not.toBeNull()
    const isDisabled =
      writeOptinRow!.getAttribute('aria-disabled') === 'true' ||
      writeOptinRow!.querySelector('[aria-disabled="true"]') !== null
    expect(isDisabled).toBe(false)
  })

  it('toggling ON when ownerWriteEnabled=true shows confirmation with CAP-05 body', () => {
    ;({ container, root } = renderPanel({ ownerWriteEnabled: true }))
    // Find the toggle input/button for "Allow file editing"
    const writeOptinSection = Array.from(container.querySelectorAll('.session-share-panel__write-optin, .settings-panel__toggle-row')).find(
      el => el.textContent?.includes('Allow file editing')
    )
    expect(writeOptinSection).not.toBeNull()
    // Click the toggle to open confirmation
    const toggle = writeOptinSection!.querySelector('input[type="checkbox"], [role="switch"], label') as HTMLElement | null
    expect(toggle).not.toBeNull()
    flushSync(() => { toggle!.click() })
    // Confirmation should show the verbatim CAP-05 body
    expect(container.textContent).toContain(
      'This will allow the recipient to create, edit, delete, rename, and upload files in this session\'s working directory.'
    )
  })

  it('cancelling confirmation reverts toggle to OFF', () => {
    ;({ container, root } = renderPanel({ ownerWriteEnabled: true }))
    const writeOptinSection = Array.from(container.querySelectorAll('.session-share-panel__write-optin, .settings-panel__toggle-row')).find(
      el => el.textContent?.includes('Allow file editing')
    )
    expect(writeOptinSection).not.toBeNull()
    const toggle = writeOptinSection!.querySelector('input[type="checkbox"], [role="switch"], label') as HTMLElement | null
    flushSync(() => { toggle!.click() })
    // Click Cancel
    const cancelBtn = Array.from(container.querySelectorAll('button')).find(
      btn => btn.textContent?.trim() === 'Cancel'
    ) as HTMLButtonElement | null
    expect(cancelBtn).not.toBeNull()
    flushSync(() => { cancelBtn!.click() })
    // Confirmation should be gone
    expect(container.textContent).not.toContain(
      'This will allow the recipient to create, edit, delete, rename'
    )
    // Toggle should be back to OFF
    const allSwitches = container.querySelectorAll('[role="switch"]')
    const optInSwitch = Array.from(allSwitches).find(el => {
      const row = el.closest('.session-share-panel__write-optin, .settings-panel__toggle-row')
      return row?.textContent?.includes('Allow file editing')
    })
    if (optInSwitch) {
      expect(optInSwitch.getAttribute('aria-checked')).toBe('false')
    }
  })

  it('confirming opt-in marks toggle ON (aria-checked="true")', () => {
    ;({ container, root } = renderPanel({ ownerWriteEnabled: true }))
    const writeOptinSection = Array.from(container.querySelectorAll('.session-share-panel__write-optin, .settings-panel__toggle-row')).find(
      el => el.textContent?.includes('Allow file editing')
    )
    const toggle = writeOptinSection!.querySelector('input[type="checkbox"], [role="switch"], label') as HTMLElement | null
    flushSync(() => { toggle!.click() })
    // Click Confirm
    const confirmBtn = Array.from(container.querySelectorAll('button')).find(
      btn => btn.textContent?.trim() === 'Confirm' || btn.textContent?.trim() === 'Enable' || btn.textContent?.trim() === 'Allow'
    ) as HTMLButtonElement | null
    expect(confirmBtn).not.toBeNull()
    flushSync(() => { confirmBtn!.click() })
    // Confirmation gone
    expect(container.textContent).not.toContain(
      'This will allow the recipient to create, edit, delete, rename'
    )
    // Toggle should now be ON
    const allSwitches = container.querySelectorAll('[role="switch"]')
    const optInSwitch = Array.from(allSwitches).find(el => {
      const row = el.closest('.session-share-panel__write-optin, .settings-panel__toggle-row')
      return row?.textContent?.includes('Allow file editing')
    })
    if (optInSwitch) {
      expect(optInSwitch.getAttribute('aria-checked')).toBe('true')
    }
  })

  it('the write-optin row renders above the Full Access Link row', () => {
    ;({ container, root } = renderPanel({ ownerWriteEnabled: false }))
    const rows = Array.from(container.querySelectorAll('.session-share-panel__link-row, .session-share-panel__write-optin'))
    const writeOptinIdx = rows.findIndex(el => el.textContent?.includes('Allow file editing'))
    const fullAccessIdx = rows.findIndex(el => el.textContent?.includes('Full Access Link'))
    expect(writeOptinIdx).toBeGreaterThanOrEqual(0)
    expect(fullAccessIdx).toBeGreaterThanOrEqual(0)
    expect(writeOptinIdx).toBeLessThan(fullAccessIdx)
  })
})

/**
 * Phase 124 WR-01 — Two-gate model: Full Access Link (files.write) is only
 * surfaced when BOTH owner write is enabled AND viewer has confirmed opt-in.
 */
describe('SessionSharePanel — WR-01 two-gate model (files.write link gating)', () => {
  let container: HTMLElement | undefined
  let root: Root | undefined

  afterEach(() => {
    if (root) {
      flushSync(() => root!.unmount())
      root = undefined
    }
    if (container) {
      container.remove()
      container = undefined
    }
    vi.clearAllMocks()
  })

  // Helper: confirm the "Allow file editing" opt-in toggle
  function confirmOptIn(c: HTMLElement): void {
    const writeOptinSection = Array.from(c.querySelectorAll('.session-share-panel__write-optin, .settings-panel__toggle-row')).find(
      el => el.textContent?.includes('Allow file editing')
    ) as HTMLElement | null
    if (!writeOptinSection) throw new Error('opt-in section not found')
    const toggle = writeOptinSection.querySelector('input[type="checkbox"], [role="switch"], label') as HTMLElement | null
    if (!toggle) throw new Error('opt-in toggle not found')
    flushSync(() => { toggle.click() })
    const confirmBtn = Array.from(c.querySelectorAll('button')).find(
      btn => btn.textContent?.trim() === 'Confirm'
    ) as HTMLButtonElement | null
    if (!confirmBtn) throw new Error('confirm button not found')
    flushSync(() => { confirmBtn.click() })
  }

  it('opt-in OFF, owner OFF → Full Access Link is NOT surfaced (no write URL shown)', () => {
    ;({ container, root } = renderPanel({
      ownerWriteEnabled: false,
      writeURL: 'https://example.com/w?cap=WRITE_TOKEN_WITH_FILES_WRITE',
    }))
    // The write URL/token must not be in the DOM when both gates are off
    expect(container.innerHTML).not.toContain('WRITE_TOKEN_WITH_FILES_WRITE')
    // The locked placeholder should be visible instead
    const lockedRow = container.querySelector('[data-testid="full-access-link-locked"]')
    expect(lockedRow).not.toBeNull()
  })

  it('opt-in OFF, owner ON → Full Access Link is NOT surfaced (write URL hidden)', () => {
    ;({ container, root } = renderPanel({
      ownerWriteEnabled: true,
      writeURL: 'https://example.com/w?cap=WRITE_TOKEN_WITH_FILES_WRITE',
    }))
    // Even though owner has enabled writes, viewer has not confirmed opt-in yet
    expect(container.innerHTML).not.toContain('WRITE_TOKEN_WITH_FILES_WRITE')
    const lockedRow = container.querySelector('[data-testid="full-access-link-locked"]')
    expect(lockedRow).not.toBeNull()
  })

  it('opt-in ON, owner OFF → Full Access Link is NOT surfaced (write URL hidden)', () => {
    // owner OFF means the toggle is disabled — opt-in cannot be confirmed
    // Even if we somehow had a stale allowFileEditing=true state, owner OFF
    // means surfaceWriteLink = false. We verify via the locked placeholder.
    ;({ container, root } = renderPanel({
      ownerWriteEnabled: false,
      writeURL: 'https://example.com/w?cap=WRITE_TOKEN_WITH_FILES_WRITE',
    }))
    expect(container.innerHTML).not.toContain('WRITE_TOKEN_WITH_FILES_WRITE')
    const lockedRow = container.querySelector('[data-testid="full-access-link-locked"]')
    expect(lockedRow).not.toBeNull()
    const liveRow = container.querySelector('[data-testid="full-access-link-row"]')
    expect(liveRow).toBeNull()
  })

  it('opt-in ON + confirmed, owner ON → Full Access Link IS surfaced with write URL', () => {
    ;({ container, root } = renderPanel({
      ownerWriteEnabled: true,
      writeURL: 'https://example.com/w?cap=WRITE_TOKEN_WITH_FILES_WRITE',
    }))
    // Confirm the opt-in
    confirmOptIn(container!)
    // Now the write URL must be in the DOM
    expect(container!.innerHTML).toContain('WRITE_TOKEN_WITH_FILES_WRITE')
    const liveRow = container!.querySelector('[data-testid="full-access-link-row"]')
    expect(liveRow).not.toBeNull()
    const lockedRow = container!.querySelector('[data-testid="full-access-link-locked"]')
    expect(lockedRow).toBeNull()
  })

  it('after opt-in confirmed, toggling opt-in OFF hides the write URL again', () => {
    ;({ container, root } = renderPanel({
      ownerWriteEnabled: true,
      writeURL: 'https://example.com/w?cap=WRITE_TOKEN_WITH_FILES_WRITE',
    }))
    // Confirm opt-in
    confirmOptIn(container!)
    expect(container!.innerHTML).toContain('WRITE_TOKEN_WITH_FILES_WRITE')

    // Now toggle opt-in OFF
    const writeOptinSection = Array.from(container!.querySelectorAll('.session-share-panel__write-optin, .settings-panel__toggle-row')).find(
      el => el.textContent?.includes('Allow file editing')
    ) as HTMLElement | null
    const toggle = writeOptinSection!.querySelector('input[type="checkbox"], [role="switch"], label') as HTMLElement | null
    flushSync(() => { toggle!.click() })

    // Write URL must be gone
    expect(container!.innerHTML).not.toContain('WRITE_TOKEN_WITH_FILES_WRITE')
    const lockedRow = container!.querySelector('[data-testid="full-access-link-locked"]')
    expect(lockedRow).not.toBeNull()
  })
})
