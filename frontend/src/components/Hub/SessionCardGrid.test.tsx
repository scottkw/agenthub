import { describe, it, expect, vi, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// Mock Wails RPC before component import (required by SessionCard → InlineSessionName)
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  ListSessions: vi.fn().mockResolvedValue([]),
}))

vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
}))

import { SessionCardGrid } from './SessionCardGrid'

// ---- Helpers ----

function makeSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'sess-1',
    cli: 'claude',
    name: 'Test Session',
    state: 'running',
    status: 'running',
    createdAt: new Date().toISOString(),
    hostname: '',
    webEnabled: false,
    viewerCount: 0,
    homeDir: false,
    filesWrite: false,
    workDir: '/home/user/project',
    ...overrides,
  }
}

function renderGrid(
  sessions: SessionInfo[],
  onRename: (id: string, name: string) => void = vi.fn(),
) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<SessionCardGrid sessions={sessions} onRename={onRename} />)
  })
  return { container, root }
}

describe('SessionCardGrid', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  // ---- Grouping: two distinct workDirs produce two groups ----

  it('renders two groups for two distinct workDirs', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/alpha' }),
      makeSession({ id: 'sess-2', workDir: '/home/user/beta' }),
    ]
    const { container } = renderGrid(sessions)
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(2)
  })

  it('renders sessions with the same workDir in a single group', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/alpha', name: 'Session A' }),
      makeSession({ id: 'sess-2', workDir: '/home/user/alpha', name: 'Session B' }),
    ]
    const { container } = renderGrid(sessions)
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(1)
    // Both listitems are in the single group
    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(2)
  })

  // ---- "Other" group for empty workDir ----

  it('groups sessions with empty workDir under an "Other" header', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '' }),
    ]
    const { container } = renderGrid(sessions)
    const headings = container.querySelectorAll('.hub__group-header')
    expect(headings.length).toBe(1)
    expect(headings[0].textContent).toContain('Other')
  })

  it('puts empty-workDir sessions in "Other" alongside named-workDir sessions', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/project' }),
      makeSession({ id: 'sess-2', workDir: '' }),
    ]
    const { container } = renderGrid(sessions)
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(2)
    const headingTexts = Array.from(container.querySelectorAll('.hub__group-header')).map(
      (h) => h.textContent,
    )
    expect(headingTexts.some((t) => t?.includes('Other'))).toBe(true)
  })

  // ---- Group header shows basename ----

  it('shows the basename of workDir in the group header', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/my-project' }),
    ]
    const { container } = renderGrid(sessions)
    const header = container.querySelector('.hub__group-header')
    expect(header).not.toBeNull()
    expect(header!.textContent).toContain('my-project')
  })

  it('uses the full workDir as the title tooltip on the group header span', () => {
    const workDir = '/home/user/my-project'
    const sessions = [makeSession({ id: 'sess-1', workDir })]
    const { container } = renderGrid(sessions)
    // The span inside the header should have the title attr set to the full path
    const titleEl = container.querySelector('.hub__group-header span[title]')
    expect(titleEl).not.toBeNull()
    expect(titleEl!.getAttribute('title')).toBe(workDir)
  })

  it('shows basename for nested paths (only last segment)', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/a/b/c/d/my-deep-project' }),
    ]
    const { container } = renderGrid(sessions)
    const header = container.querySelector('.hub__group-header')
    expect(header!.textContent).toContain('my-deep-project')
    expect(header!.textContent).not.toContain('/a/b/c/d/')
  })

  // ---- Per-session listitem structure ----

  it('renders one listitem per session', () => {
    const sessions = [
      makeSession({ id: 'sess-1', workDir: '/home/user/alpha' }),
      makeSession({ id: 'sess-2', workDir: '/home/user/alpha' }),
      makeSession({ id: 'sess-3', workDir: '/home/user/beta' }),
    ]
    const { container } = renderGrid(sessions)
    const listitems = container.querySelectorAll('[role="listitem"]')
    expect(listitems.length).toBe(3)
  })

  it('each group card-row has role="list"', () => {
    const sessions = [makeSession({ id: 'sess-1', workDir: '/home/user/project' })]
    const { container } = renderGrid(sessions)
    const lists = container.querySelectorAll('[role="list"]')
    expect(lists.length).toBeGreaterThanOrEqual(1)
  })

  // ---- Group headers have correct heading semantics ----

  it('group headers use role="heading" aria-level=2 or h2 element', () => {
    const sessions = [makeSession({ id: 'sess-1', workDir: '/home/user/project' })]
    const { container } = renderGrid(sessions)
    // Accept either <h2> or element with role="heading" aria-level="2"
    const h2s = container.querySelectorAll('h2')
    const roleHeading2 = container.querySelectorAll('[role="heading"][aria-level="2"]')
    expect(h2s.length + roleHeading2.length).toBeGreaterThanOrEqual(1)
  })

  // ---- Empty sessions ----

  it('renders nothing (no groups) when sessions is empty', () => {
    const { container } = renderGrid([])
    const groups = container.querySelectorAll('.hub__group')
    expect(groups.length).toBe(0)
  })
})
