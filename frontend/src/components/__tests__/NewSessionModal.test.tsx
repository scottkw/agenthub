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

// Phase 101-02: NewSessionModal shell rows (SHELL-01 GUI half).
// Render tests use createRoot + flushSync, mirroring DaemonManagerPanel.test.tsx.

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

describe('Phase 101-02 NewSessionModal — shell rows (SHELL-01 GUI)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
    localStorage.clear()
  })

  it('renders one row per shell with Shell em-dash prefix', () => {
    const shells = [
      makeShell({ name: 'shell', displayName: 'system default', path: '/bin/zsh' }),
      makeShell({ name: 'bash', displayName: 'bash', path: '/bin/bash' }),
      makeShell({ name: 'zsh', displayName: 'zsh', path: '/bin/zsh' }),
    ]
    ;({ container, root } = renderModal({ shells }))
    const buttons = Array.from(container.querySelectorAll('.new-session-modal__agent-btn'))
    const shellButtons = buttons.filter((b) => (b.textContent || '').startsWith('Shell —'))
    expect(shellButtons.length).toBe(3)
    const labels = shellButtons.map((b) => (b.textContent || '').replace(/\s+/g, ' '))
    expect(labels.some((l) => l.includes('Shell — system default'))).toBe(true)
    expect(labels.some((l) => l.includes('Shell — bash'))).toBe(true)
    expect(labels.some((l) => l.includes('Shell — zsh'))).toBe(true)
  })

  it('system default shell row appears first', () => {
    const shells = [
      makeShell({ name: 'bash', displayName: 'bash', path: '/bin/bash' }),
      makeShell({ name: 'shell', displayName: 'system default', path: '/bin/zsh' }),
      makeShell({ name: 'zsh', displayName: 'zsh', path: '/bin/zsh' }),
    ]
    ;({ container, root } = renderModal({ shells }))
    const buttons = Array.from(container.querySelectorAll('.new-session-modal__agent-btn'))
    const shellButtons = buttons.filter((b) => (b.textContent || '').startsWith('Shell —'))
    expect(shellButtons[0].textContent).toContain('Shell — system default')
  })

  it('shell row shows resolved path as mono secondary line', () => {
    const shells = [makeShell({ name: 'bash', displayName: 'bash', path: '/bin/bash' })]
    ;({ container, root } = renderModal({ shells }))
    const detail = container.querySelector('.new-session-modal__agent-btn__detail')
    expect(detail).not.toBeNull()
    expect(detail!.textContent).toBe('/bin/bash')
  })

  it('selecting shell row applies shell cyan border modifier', () => {
    const shells = [makeShell({ name: 'bash', displayName: 'bash', path: '/bin/bash' })]
    ;({ container, root } = renderModal({ shells }))
    const buttons = Array.from(container.querySelectorAll('.new-session-modal__agent-btn'))
    const bashBtn = buttons.find((b) => (b.textContent || '').includes('Shell — bash')) as HTMLButtonElement
    expect(bashBtn).toBeDefined()
    flushSync(() => bashBtn.click())
    expect(bashBtn.getAttribute('aria-pressed')).toBe('true')
    expect(bashBtn.className).toContain('--selected-shell')
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

  it('loading skeleton renders when shellsLoading is true and shells is empty', () => {
    ;({ container, root } = renderModal({ shells: [], shellsLoading: true }))
    expect(container.textContent).toContain('Loading shells…')
  })

  it('no shell rows when shells prop is empty and not loading', () => {
    ;({ container, root } = renderModal({ shells: [], shellsLoading: false }))
    const buttons = Array.from(container.querySelectorAll('.new-session-modal__agent-btn'))
    const shellButtons = buttons.filter((b) => (b.textContent || '').startsWith('Shell —'))
    expect(shellButtons.length).toBe(0)
    expect(container.textContent || '').not.toContain('Loading shells')
  })

  it('args field disabled when shell selected', () => {
    const shells = [makeShell({ name: 'bash', displayName: 'bash', path: '/bin/bash' })]
    ;({ container, root } = renderModal({ shells }))
    const buttons = Array.from(container.querySelectorAll('.new-session-modal__agent-btn'))
    const bashBtn = buttons.find((b) => (b.textContent || '').includes('Shell — bash')) as HTMLButtonElement
    flushSync(() => bashBtn.click())
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

  it('args namespace key uses shell prefix when shell selected', () => {
    // Seed the AI-CLI-style key for bash, plus the shell-prefixed key.
    localStorage.setItem('agenthub:args:bash', 'AI-CLI-flag')
    localStorage.setItem('agenthub:args:shell:bash', '--login')
    const shells = [makeShell({ name: 'bash', displayName: 'bash', path: '/bin/bash' })]
    ;({ container, root } = renderModal({ shells }))
    // Source check: NewSessionModal must reference the shell namespace token.
    expect(raw).toContain('agenthub:args:shell:')
    // Behavior probe: after selecting bash shell, the modal must NOT pull the AI-CLI bash key.
    const buttons = Array.from(container.querySelectorAll('.new-session-modal__agent-btn'))
    const bashBtn = buttons.find((b) => (b.textContent || '').includes('Shell — bash')) as HTMLButtonElement
    flushSync(() => bashBtn.click())
    const argsInput = container.querySelector('.new-session-modal__args-input') as HTMLInputElement
    // Since shells disable the args field, the field's value (if any) must not be the AI-CLI value.
    expect(argsInput.value).not.toBe('AI-CLI-flag')
  })
})
