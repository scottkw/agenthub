import { describe, it, expect, afterEach, vi } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import raw from '../NewSessionModal.tsx?raw'
import { NewSessionModal } from '../NewSessionModal'

describe('NewSessionModal source inspection', () => {
  // SESS-01: Modal structure
  describe('SESS-01: modal (not dropdown)', () => {
    it('uses new-session-overlay class', () => {
      expect(raw).toContain('new-session-overlay')
    })
    it('uses new-session-modal class', () => {
      expect(raw).toContain('new-session-modal')
    })
    it('accepts onConfirm prop', () => {
      expect(raw).toContain('onConfirm')
    })
    it('accepts onClose prop', () => {
      expect(raw).toContain('onClose')
    })
  })

  // SESS-02: Agent picker
  describe('SESS-02: agent picker', () => {
    it('uses DetectedCLI type', () => {
      expect(raw).toContain('DetectedCLI')
    })
    it('renders DisplayName for each CLI', () => {
      expect(raw).toContain('DisplayName')
    })
    it('tracks selected agent state', () => {
      // Phase 101-02 renamed selectedCLI -> selectedAgent to accommodate the
      // "shell:NAME" prefix scheme for shell rows.
      expect(raw).toContain('selectedAgent')
    })
  })

  // SESS-03: Folder browser
  describe('SESS-03: native folder browser', () => {
    it('imports OpenDirectoryDialog', () => {
      expect(raw).toContain('OpenDirectoryDialog')
    })
    it('has Browse button text', () => {
      expect(raw).toContain('Browse')
    })
    it('tracks browseLoading state', () => {
      expect(raw).toContain('browseLoading')
    })
  })

  // SESS-04: Last-used folder memory
  describe('SESS-04: last-used folder persistence', () => {
    it('uses agenthub:lastWorkDir localStorage key', () => {
      expect(raw).toContain('agenthub:lastWorkDir')
    })
    it('reads from localStorage on open', () => {
      expect(raw).toContain('localStorage.getItem')
    })
    it('writes to localStorage after folder pick', () => {
      expect(raw).toContain('localStorage.setItem')
    })
  })
})

// ARGS-02: Args text field
describe('ARGS-02: args text field', () => {
  it('has args input class', () => {
    expect(raw).toContain('new-session-modal__args-input')
  })
  it('has placeholder with example flag', () => {
    expect(raw).toContain('e.g. --model claude-opus-4-5')
  })
  it('splits args with filter(Boolean) to avoid empty strings', () => {
    expect(raw).toContain('.filter(Boolean)')
  })
})

// ARGS-04: Per-agent args persistence
describe('ARGS-04: per-agent args persistence', () => {
  it('uses agenthub:args: localStorage key pattern', () => {
    expect(raw).toContain('agenthub:args:')
  })
  it('reads args from localStorage on agent change', () => {
    // handleSelectCLI reads localStorage for the newly selected agent
    expect(raw).toContain('handleSelectCLI')
  })
  it('persists args to localStorage on confirm', () => {
    // handleConfirm calls localStorage.setItem with ARGS_KEY.
    // Phase 101-02 renamed selectedCLI -> selectedAgent.
    expect(raw).toContain('ARGS_KEY(selectedAgent)')
  })
})

// ARGS-05: Clear args
describe('ARGS-05: clear args button', () => {
  it('has handleClearArgs function', () => {
    expect(raw).toContain('handleClearArgs')
  })
  it('removes localStorage key on clear', () => {
    expect(raw).toContain('localStorage.removeItem')
  })
  it('has accessible clear button', () => {
    expect(raw).toContain('aria-label="Clear arguments"')
  })
})

// Phase 107-03: NewSessionModal shell row — updated for SHELL-10 (single static row).
// The Phase 101-02 multi-row tests are superseded by NewSessionModal.shellRow.test.tsx.
// This block retains the render-helper and keeps passing tests (AI-CLI behavior unchanged).

interface DetectedShellFixture {
  name: string
  displayName: string
  path: string
  argv: string[]
}

function makeShell(overrides: Partial<DetectedShellFixture> = {}): DetectedShellFixture {
  return {
    name: 'bash',
    displayName: 'bash',
    path: '/bin/bash',
    argv: ['-i'],
    ...overrides,
  }
}

interface RenderOpts {
  clis?: Array<{ Name: string; Path: string; DisplayName?: string }>
  shells?: DetectedShellFixture[]
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

// Phase 107-03 SHELL-10: single static Shell row replaces the Phase 101 multi-row loop.
// Full contract in NewSessionModal.shellRow.test.tsx; these tests verify backwards-compat
// of the render helper and AI-CLI interactions that remain unchanged.
describe('Phase 101-02 / SHELL-10 NewSessionModal — shell row (collapsed single row)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
    localStorage.clear()
  })

  it('always renders exactly ONE shell button regardless of shells prop', () => {
    const shells = [
      makeShell({ name: 'shell', displayName: 'system default', path: '/bin/zsh' }),
      makeShell({ name: 'bash', displayName: 'bash', path: '/bin/bash' }),
      makeShell({ name: 'zsh', displayName: 'zsh', path: '/bin/zsh' }),
    ]
    ;({ container, root } = renderModal({ shells }))
    const shellBtns = container.querySelectorAll('.new-session-modal__agent-btn--shell')
    expect(shellBtns.length).toBe(1)
  })

  it('shell button label is "Shell" (no em-dash suffix)', () => {
    ;({ container, root } = renderModal({ shells: [] }))
    const shellBtn = container.querySelector('.new-session-modal__agent-btn--shell')
    expect(shellBtn).not.toBeNull()
    const labelSpan = shellBtn!.querySelector('span:first-child')
    expect(labelSpan?.textContent).toBe('Shell')
  })

  it('selecting shell row applies shell cyan border modifier', () => {
    ;({ container, root } = renderModal({ shells: [] }))
    const shellBtn = container.querySelector('.new-session-modal__agent-btn--shell') as HTMLButtonElement
    expect(shellBtn).not.toBeNull()
    flushSync(() => shellBtn.click())
    expect(shellBtn.getAttribute('aria-pressed')).toBe('true')
    expect(shellBtn.className).toContain('--selected-shell')
  })

  it('selecting AI CLI row applies accent blue border modifier not shell cyan', () => {
    ;({ container, root } = renderModal({
      clis: [{ Name: 'claude', Path: '/usr/local/bin/claude', DisplayName: 'Claude Code' }],
      shells: [makeShell({ name: 'bash', displayName: 'bash', path: '/bin/bash' })],
    }))
    const buttons = Array.from(container.querySelectorAll('.new-session-modal__agent-btn'))
    const claudeBtn = buttons.find((b) => (b.textContent || '').includes('Claude Code')) as HTMLButtonElement
    expect(claudeBtn).toBeDefined()
    flushSync(() => claudeBtn.click())
    expect(claudeBtn.getAttribute('aria-pressed')).toBe('true')
    expect(claudeBtn.className).toContain('--selected')
    expect(claudeBtn.className).not.toContain('--selected-shell')
  })

  it('no "Loading shells…" skeleton even when shellsLoading=true (SHELL-10 removes skeleton)', () => {
    ;({ container, root } = renderModal({ shells: [], shellsLoading: true }))
    expect(container.textContent).not.toContain('Loading shells')
  })

  it('shell row still present when shells prop is empty', () => {
    ;({ container, root } = renderModal({ shells: [], shellsLoading: false }))
    const shellBtn = container.querySelector('.new-session-modal__agent-btn--shell')
    expect(shellBtn).not.toBeNull()
  })

  it('args field disabled when shell selected', () => {
    ;({ container, root } = renderModal({ shells: [] }))
    const shellBtn = container.querySelector('.new-session-modal__agent-btn--shell') as HTMLButtonElement
    flushSync(() => shellBtn.click())
    const argsInput = container.querySelector('.new-session-modal__args-input') as HTMLInputElement
    expect(argsInput).not.toBeNull()
    expect(argsInput.disabled).toBe(true)
    expect(argsInput.placeholder).toBe('Arguments are not passed to shell sessions')
  })

  it('args field enabled when AI CLI selected', () => {
    ;({ container, root } = renderModal({
      clis: [{ Name: 'claude', Path: '/usr/local/bin/claude', DisplayName: 'Claude Code' }],
      shells: [makeShell({ name: 'bash', displayName: 'bash', path: '/bin/bash' })],
    }))
    const buttons = Array.from(container.querySelectorAll('.new-session-modal__agent-btn'))
    const claudeBtn = buttons.find((b) => (b.textContent || '').includes('Claude Code')) as HTMLButtonElement
    flushSync(() => claudeBtn.click())
    const argsInput = container.querySelector('.new-session-modal__args-input') as HTMLInputElement
    expect(argsInput.disabled).toBe(false)
  })

  it('shell namespace key (agenthub:args:shell:) still referenced in source', () => {
    // Source check: SHELL_ARGS_KEY namespace preserved for backwards compat with
    // any stored args from Phase 101 multi-row era.
    expect(raw).toContain('agenthub:args:shell:')
  })
})
