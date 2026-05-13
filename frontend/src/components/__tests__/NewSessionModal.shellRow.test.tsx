/**
 * SHELL-10 test contract — UI-SPEC §4 (six assertions).
 * Locks the single static Shell row behavior introduced in Phase 107-03.
 */
import { describe, it, expect, afterEach, vi, beforeEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { act } from 'react-dom/test-utils'
import { NewSessionModal } from '../NewSessionModal'
import * as AppMock from '../../wailsjs/go/main/App'

// Mock Wails bindings — GetShellPath must resolve to the daemon-resolved path.
vi.mock('../../wailsjs/go/main/App', () => ({
  OpenDirectoryDialog: vi.fn().mockResolvedValue(''),
  GetShellPath: vi.fn().mockResolvedValue('/bin/zsh'),
  SetShellPath: vi.fn().mockResolvedValue(undefined),
}))

interface RenderOpts {
  clis?: Array<{ Name: string; Path: string; DisplayName?: string }>
  shells?: Array<{ name: string; displayName: string; path: string; argv: string[] }>
  shellsLoading?: boolean
  onConfirm?: (cli: string, workDir: string, args: string[]) => void
  onClose?: () => void
}

function renderModal(opts: RenderOpts = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const props: any = {
    isOpen: true,
    clis: opts.clis ?? [
      { Name: 'claude', Path: '/usr/local/bin/claude', DisplayName: 'Claude Code' },
    ],
    shells: opts.shells ?? [],
    shellsLoading: opts.shellsLoading ?? false,
    onConfirm: opts.onConfirm ?? vi.fn(),
    onClose: opts.onClose ?? vi.fn(),
  }
  flushSync(() => {
    root.render(React.createElement(NewSessionModal, props))
  })
  return { container, root }
}

describe('SHELL-10: Single Shell row in NewSessionModal', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(AppMock.GetShellPath).mockResolvedValue('/bin/zsh')
  })

  afterEach(() => {
    root?.unmount()
    container?.remove()
    localStorage.clear()
  })

  // Assertion 1: Exactly ONE shell button regardless of shells prop length.
  it('renders exactly ONE shell button regardless of shells prop length (empty)', () => {
    ;({ container, root } = renderModal({ shells: [] }))
    const shellBtns = container.querySelectorAll('.new-session-modal__agent-btn--shell')
    expect(shellBtns.length).toBe(1)
  })

  it('renders exactly ONE shell button regardless of shells prop length (2 shells)', () => {
    ;({ container, root } = renderModal({
      shells: [
        { name: 'zsh', displayName: 'zsh', path: '/bin/zsh', argv: ['-i'] },
        { name: 'bash', displayName: 'bash', path: '/bin/bash', argv: ['-i'] },
      ],
    }))
    const shellBtns = container.querySelectorAll('.new-session-modal__agent-btn--shell')
    expect(shellBtns.length).toBe(1)
  })

  it('renders exactly ONE shell button regardless of shells prop length (3 shells)', () => {
    ;({ container, root } = renderModal({
      shells: [
        { name: 'zsh', displayName: 'zsh', path: '/bin/zsh', argv: ['-i'] },
        { name: 'bash', displayName: 'bash', path: '/bin/bash', argv: ['-i'] },
        { name: 'fish', displayName: 'fish', path: '/usr/bin/fish', argv: ['-i'] },
      ],
    }))
    const shellBtns = container.querySelectorAll('.new-session-modal__agent-btn--shell')
    expect(shellBtns.length).toBe(1)
  })

  // Assertion 2: aria-pressed=true when selectedAgent === 'shell'.
  it('shell button has aria-pressed=true when shell is selected', () => {
    ;({ container, root } = renderModal({ clis: [] }))
    // With no clis, the modal defaults to selecting nothing or the first clis entry.
    // Click the shell button to select it.
    const shellBtn = container.querySelector('.new-session-modal__agent-btn--shell') as HTMLButtonElement
    expect(shellBtn).not.toBeNull()
    flushSync(() => shellBtn.click())
    expect(shellBtn.getAttribute('aria-pressed')).toBe('true')
  })

  // Assertion 3: Button contains "Shell" span and detail span with resolved path.
  it('button contains a span with text "Shell" and a detail span with resolved path', async () => {
    ;({ container, root } = renderModal())
    // Wait for GetShellPath() promise to resolve.
    await act(async () => {
      await Promise.resolve()
    })
    const shellBtn = container.querySelector('.new-session-modal__agent-btn--shell')
    expect(shellBtn).not.toBeNull()
    // Label span.
    const spans = shellBtn!.querySelectorAll('span')
    const labelSpan = Array.from(spans).find((s) => s.textContent === 'Shell')
    expect(labelSpan).not.toBeUndefined()
    // Detail span.
    const detailSpan = shellBtn!.querySelector('.new-session-modal__agent-btn__detail')
    expect(detailSpan).not.toBeNull()
    expect(detailSpan!.textContent).toBe('/bin/zsh')
  })

  // Assertion 4: No "Loading shells…" skeleton even when shellsLoading=true.
  it('does NOT render Loading shells skeleton when shellsLoading=true', () => {
    ;({ container, root } = renderModal({ shells: [], shellsLoading: true }))
    expect(container.textContent).not.toContain('Loading shells')
  })

  // Assertion 5: Clicking the row then confirming calls onConfirm with bare 'shell'.
  it('clicking the row then confirming calls onConfirm with bare "shell" (no prefix)', () => {
    const onConfirm = vi.fn()
    ;({ container, root } = renderModal({ clis: [], onConfirm }))
    const shellBtn = container.querySelector('.new-session-modal__agent-btn--shell') as HTMLButtonElement
    flushSync(() => shellBtn.click())
    const createBtn = container.querySelector('.new-session-modal__btn--create') as HTMLButtonElement
    flushSync(() => createBtn.click())
    expect(onConfirm).toHaveBeenCalledWith('shell', expect.any(String), [])
    // Confirm: NOT prefixed with 'shell:' or any other prefix.
    const firstArg = onConfirm.mock.calls[0][0] as string
    expect(firstArg).toBe('shell')
    expect(firstArg.startsWith('shell:')).toBe(false)
  })

  // Assertion 6: Existing AI CLI rows are unchanged.
  it('existing AI CLI rows still render and are clickable', () => {
    const onConfirm = vi.fn()
    ;({ container, root } = renderModal({
      clis: [{ Name: 'claude', Path: '/usr/local/bin/claude', DisplayName: 'Claude Code' }],
      onConfirm,
    }))
    const claudeBtn = container.querySelector(
      '.new-session-modal__agent-btn:not(.new-session-modal__agent-btn--shell)'
    ) as HTMLButtonElement
    expect(claudeBtn).not.toBeNull()
    expect(claudeBtn.textContent).toContain('Claude Code')
    flushSync(() => claudeBtn.click())
    expect(claudeBtn.getAttribute('aria-pressed')).toBe('true')
  })
})
