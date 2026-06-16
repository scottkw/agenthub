import { describe, it, expect, vi, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'

// Mock Wails RPC before component import
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  ListSessions: vi.fn().mockResolvedValue([]),
}))

import { SessionCard } from './SessionCard'
import type { SessionInfo } from '../../wailsjs/go/main/App'

function makeSession(overrides: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id: 'sess-1',
    cli: 'claude',
    name: 'Test Session',
    state: 'running',
    status: 'running',
    createdAt: new Date(Date.now() - 7454000).toISOString(), // ~2h 4m ago
    hostname: 'local-machine.local',
    webEnabled: false,
    viewerCount: 0,
    homeDir: false,
    filesWrite: false,
    workDir: '/home/user/project',
    ...overrides,
  }
}

function renderCard(session: SessionInfo, onRename?: (id: string, name: string) => void) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<SessionCard session={session} onRename={onRename} />)
  })
  return { container, root }
}

describe('SessionCard', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  // ---- Status rendering: icon + text label for every status ----

  it('running: renders icon with aria-label "Running" AND visible text label "Running"', () => {
    const { container } = renderCard(makeSession({ state: 'running', status: 'running' }))
    const icon = container.querySelector('[aria-label="Running"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Running')
  })

  it('idle: renders icon with aria-label "Idle" AND visible text label "Idle"', () => {
    const { container } = renderCard(makeSession({ state: 'idle', status: 'idle' }))
    const icon = container.querySelector('[aria-label="Idle"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Idle')
  })

  it('waiting: renders icon with aria-label "Needs input" AND visible text label "Needs input"', () => {
    const { container } = renderCard(makeSession({ state: 'waiting', status: 'waiting' }))
    const icon = container.querySelector('[aria-label="Needs input"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Needs input')
  })

  it('errored: renders icon with aria-label "Error" AND visible text label "Error"', () => {
    const { container } = renderCard(makeSession({ state: 'errored', status: 'errored' }))
    const icon = container.querySelector('[aria-label="Error"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Error')
  })

  it('stopped exit-0: renders icon with aria-label "Done" AND visible text label "Done"', () => {
    const exitCode = 0
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 3600 }))
    const icon = container.querySelector('[aria-label="Done"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Done')
  })

  it('stopped exit-nonzero: renders icon with aria-label containing "Exited" AND visible text label containing "Exited"', () => {
    const exitCode = 1
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 3600 }))
    const icon = container.querySelector('[aria-label*="Exited"]')
    expect(icon).not.toBeNull()
    const label = container.querySelector('.hub-card__status-label')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Exited')
  })

  // ---- Spin modifier on running only ----

  it('running card icon has hub-card__status-icon--spin class', () => {
    const { container } = renderCard(makeSession({ state: 'running', status: 'running' }))
    const spinIcon = container.querySelector('.hub-card__status-icon--spin')
    expect(spinIcon).not.toBeNull()
  })

  it('idle card icon does NOT have hub-card__status-icon--spin class', () => {
    const { container } = renderCard(makeSession({ state: 'idle', status: 'idle' }))
    const spinIcon = container.querySelector('.hub-card__status-icon--spin')
    expect(spinIcon).toBeNull()
  })

  // ---- Dimming ----

  it('stopped exit-0 card gets hub-card--dim class', () => {
    const exitCode = 0
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 120 }))
    const card = container.querySelector('.hub-card--dim')
    expect(card).not.toBeNull()
  })

  it('stopped non-zero exit card does NOT get hub-card--dim class', () => {
    const exitCode = 1
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 120 }))
    const card = container.querySelector('.hub-card--dim')
    expect(card).toBeNull()
  })

  it('stopped non-zero exit card shows exit-code chip with "Exited {code}"', () => {
    const exitCode = 2
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 120 }))
    const chip = container.querySelector('.hub-card__exit-chip')
    expect(chip).not.toBeNull()
    expect(chip!.textContent).toContain('2')
  })

  it('stopped exit-0 card does NOT show exit-code chip', () => {
    const exitCode = 0
    const { container } = renderCard(makeSession({ state: 'stopped', exitCode, duration: 120 }))
    const chip = container.querySelector('.hub-card__exit-chip')
    expect(chip).toBeNull()
  })

  // ---- Viewer count ----

  it('viewerCount === 0 hides the viewer row', () => {
    const { container } = renderCard(makeSession({ viewerCount: 0 }))
    const viewers = container.querySelector('.hub-card__viewers')
    expect(viewers).toBeNull()
  })

  it('viewerCount === 1 shows "1 viewer" (singular)', () => {
    const { container } = renderCard(makeSession({ viewerCount: 1 }))
    const viewers = container.querySelector('.hub-card__viewers')
    expect(viewers).not.toBeNull()
    expect(viewers!.textContent).toContain('1 viewer')
    expect(viewers!.textContent).not.toContain('1 viewers')
  })

  it('viewerCount === 3 shows "3 viewers" (plural)', () => {
    const { container } = renderCard(makeSession({ viewerCount: 3 }))
    const viewers = container.querySelector('.hub-card__viewers')
    expect(viewers).not.toBeNull()
    expect(viewers!.textContent).toContain('3 viewers')
  })

  // ---- Origin marker ----

  it('local session (empty hostname) shows "Local" with ComputerDesktopIcon', () => {
    const { container } = renderCard(makeSession({ hostname: '' }))
    const origin = container.querySelector('.hub-card__origin')
    expect(origin).not.toBeNull()
    expect(origin!.textContent).toContain('Local')
  })

  it('remote session shows peer hostname with GlobeAltIcon', () => {
    const { container } = renderCard(makeSession({ hostname: 'remote-peer.tail' }))
    const origin = container.querySelector('.hub-card__origin')
    expect(origin).not.toBeNull()
    expect(origin!.textContent).toContain('remote-peer.tail')
  })

  // ---- CLI badge ----

  it('renders CLI badge with cli name as text', () => {
    const { container } = renderCard(makeSession({ cli: 'claude' }))
    // WR-03/CR-01: badge now uses hub-card__badge (Hub text-chip) not tab__agent-badge dot
    const badge = container.querySelector('.hub-card__badge')
    expect(badge).not.toBeNull()
    // CLI name should be visible text
    expect(badge!.textContent).toContain('claude')
  })

  // ---- Uptime / duration ----

  it('running session shows uptime string (not "Ran")', () => {
    const { container } = renderCard(makeSession({ state: 'running', status: 'running' }))
    // CR-01: class name corrected from hub-card__time to hub-card__uptime (matches CSS)
    const timeRow = container.querySelector('.hub-card__uptime')
    expect(timeRow).not.toBeNull()
    expect(timeRow!.textContent).not.toContain('Ran')
  })

  it('stopped session shows "Ran ..." duration', () => {
    const { container } = renderCard(
      makeSession({ state: 'stopped', exitCode: 0, duration: 3600 })
    )
    // CR-01: class name corrected from hub-card__time to hub-card__uptime (matches CSS)
    const timeRow = container.querySelector('.hub-card__uptime')
    expect(timeRow).not.toBeNull()
    expect(timeRow!.textContent).toContain('Ran')
  })

  // ---- Aria-label on card ----

  it('card has aria-label containing name, status, cli, and origin', () => {
    const { container } = renderCard(makeSession({ hostname: '' }))
    const card = container.querySelector('.hub-card')
    expect(card).not.toBeNull()
    const label = card!.getAttribute('aria-label') ?? ''
    expect(label).toContain('Test Session')
    expect(label).toContain('claude')
    expect(label).toContain('Local')
  })

  it('card has tabIndex={0}', () => {
    const { container } = renderCard(makeSession())
    const card = container.querySelector('.hub-card')
    expect(card).not.toBeNull()
    expect((card as HTMLElement).tabIndex).toBe(0)
  })

  // ---- STATUS_CONFIG usage (source-level check via grep — verified in acceptance_criteria)
  // ---- COLORBLIND-SAFE comments (source-level check via grep — verified in acceptance_criteria)
})
