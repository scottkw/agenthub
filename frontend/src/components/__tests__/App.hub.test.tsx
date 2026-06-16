import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

// Source-inspection tests for App.tsx Hub integration (Phase 131 Plan 05).
// These verify the wiring contract without mounting the full Wails component tree.
// Pattern: mirrors App.nav.test.tsx (source inspection via ?raw import).

describe('HUB-TAB: HUB_TAB constant exists with correct id and type (HUB-01)', () => {
  it("defines HUB_TAB constant", () => {
    expect(raw).toContain('HUB_TAB')
  })

  it("HUB_TAB has id '__hub__'", () => {
    expect(raw).toContain("id: '__hub__'")
  })

  it("HUB_TAB has type 'hub'", () => {
    expect(raw).toContain("type: 'hub'")
  })

  it('HUB_TAB has name Hub', () => {
    expect(raw).toContain("name: 'Hub'")
  })
})

describe('HUB-WIRE: onOpenHub is wired to Sidebar (HUB-01)', () => {
  it('defines handleOpenHub callback', () => {
    expect(raw).toContain('handleOpenHub')
  })

  it('wires onOpenHub={handleOpenHub} to Sidebar', () => {
    expect(raw).toContain('onOpenHub={handleOpenHub}')
  })

  it('wires activePanel={activeId ?? undefined} to Sidebar', () => {
    expect(raw).toContain('activePanel={activeId ?? undefined}')
  })

  it('handleOpenHub uses find-or-create pattern checking t.type === hub', () => {
    expect(raw).toContain("t.type === 'hub'")
  })
})

describe('HUB-POLL: Hub poll is gated on HUB_TAB.id (T-131-10 DoS prevention)', () => {
  it('Hub poll early-returns when activeId !== HUB_TAB.id', () => {
    expect(raw).toContain('activeId !== HUB_TAB.id')
  })

  it('Hub poll uses 3s interval (3000ms)', () => {
    // The Hub poll must use 3000ms interval (same as daemon-manager poll)
    // We verify both the HUB_TAB.id gate and the 3000 interval are present
    expect(raw).toContain('3000')
    expect(raw).toContain('HUB_TAB.id')
  })

  it('Hub poll calls ListSessions', () => {
    expect(raw).toContain('ListSessions')
  })

  it('Hub poll sets hubSessions and hubError', () => {
    expect(raw).toContain('setHubSessions')
    expect(raw).toContain('setHubError')
  })

  it('Hub poll early-returns in web mode', () => {
    // The poll must check mode === web before polling (Phase 120-06 gate)
    expect(raw).toContain("mode === 'web'")
  })
})

describe('HUB-RENDER: HubPanel is rendered with correct props (HUB-01)', () => {
  it('imports HubPanel from Hub/HubPanel', () => {
    expect(raw).toContain("from './components/Hub/HubPanel'")
  })

  it('renders HubPanel when activeId === HUB_TAB.id', () => {
    expect(raw).toContain('activeId === HUB_TAB.id')
  })

  it('passes sessions={hubSessions} to HubPanel', () => {
    expect(raw).toContain('sessions={hubSessions}')
  })

  it('passes error={hubError} to HubPanel', () => {
    expect(raw).toContain('error={hubError}')
  })

  it('HubPanel onNewSession calls setShowNewSessionModal(true)', () => {
    expect(raw).toContain('setShowNewSessionModal(true)')
  })

  it('passes onRename={handleRenameTab} to HubPanel', () => {
    expect(raw).toContain('onRename={handleRenameTab}')
  })
})

describe('HUB-02: Hub coexists with Sessions panel (terminal-exclusion sites)', () => {
  it("'hub' is excluded from the daemonError empty-filter (first terminal-exclusion site)", () => {
    // The daemonError check filters out non-terminal tabs to decide whether to show the error.
    // 'hub' must be excluded so the Hub tab doesn't trigger daemon-error display.
    const daemonErrorIdx = raw.indexOf('daemonError && tabs.filter')
    expect(daemonErrorIdx).toBeGreaterThan(-1)
    const surroundingBlock = raw.slice(daemonErrorIdx, daemonErrorIdx + 300)
    expect(surroundingBlock).toContain("t.type !== 'hub'")
  })

  it("'hub' is excluded from the terminal map (second terminal-exclusion site)", () => {
    // The terminal map skips non-terminal tabs via early-return.
    // 'hub' must be in that exclusion list so it doesn't try to render a TerminalPanel.
    const terminalMapIdx = raw.indexOf("tab.type === 'welcome' || tab.type === 'daemon-manager'")
    expect(terminalMapIdx).toBeGreaterThan(-1)
    const surroundingBlock = raw.slice(terminalMapIdx, terminalMapIdx + 300)
    expect(surroundingBlock).toContain("tab.type === 'hub'")
  })

  it('daemon-manager gate is untouched (uses DAEMON_MANAGER_TAB.id, not hub id)', () => {
    // HUB-02 coexistence: the daemon-manager panel gate must NOT reference __hub__
    const dmGateIdx = raw.indexOf('activeId === DAEMON_MANAGER_TAB.id')
    expect(dmGateIdx).toBeGreaterThan(-1)
  })

  it("'hub' appears in the daemonError terminal-exclusion filter (t.type !== 'hub')", () => {
    // First exclusion site uses t.type !== 'hub'
    const matches1 = raw.match(/t\.type !== 'hub'/g)
    expect(matches1).not.toBeNull()
    expect(matches1!.length).toBeGreaterThanOrEqual(1)
  })

  it("'hub' appears in the terminal map exclusion (tab.type === 'hub')", () => {
    // Second exclusion site uses tab.type === 'hub' in the early-return guard
    const matches2 = raw.match(/tab\.type === 'hub'/g)
    expect(matches2).not.toBeNull()
    expect(matches2!.length).toBeGreaterThanOrEqual(1)
  })
})

describe('HUB-REATTACH: open/focus a running session terminal tab (Phase 131 UAT follow-up)', () => {
  it('defines handleOpenSessionTab callback', () => {
    expect(raw).toContain('handleOpenSessionTab')
  })

  it('handleOpenSessionTab focuses an existing tab by sessionId (find-or-create)', () => {
    const idx = raw.indexOf('const handleOpenSessionTab')
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 500)
    // focuses existing tab when one already exists for this session id
    expect(block).toContain('tabs.find((t) => t.id === sessionId)')
    expect(block).toContain('setActiveId(existing.id)')
    // otherwise creates a terminal tab (no `type` → terminal) keyed by sessionId
    expect(block).toContain('sessionId,')
    expect(block).toContain('setActiveId(newTab.id)')
  })

  it('wires onOpenSession={handleOpenSessionTab} to HubPanel', () => {
    const idx = raw.indexOf('<HubPanel')
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 300)
    expect(block).toContain('onOpenSession={handleOpenSessionTab}')
  })

  it('wires onOpenSession={handleOpenSessionTab} to DaemonManagerPanel (cross-surface parity)', () => {
    const idx = raw.indexOf('<DaemonManagerPanel')
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 700)
    expect(block).toContain('onOpenSession={handleOpenSessionTab}')
  })
})
