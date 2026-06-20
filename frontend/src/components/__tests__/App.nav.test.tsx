import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

// Source-inspection tests for App.tsx sidebar navigation wiring (56-01).
// These verify the wiring contract without mounting the full component tree
// (which requires Wails runtime mocks).

describe('NAV-01: Home sidebar button opens Welcome tab', () => {
  it('defines handleHome callback', () => {
    expect(raw).toContain('handleHome')
  })

  it('wires onHome={handleHome} to Sidebar', () => {
    expect(raw).toContain('onHome={handleHome}')
  })

  it('find-or-create pattern checks t.type === welcome', () => {
    expect(raw).toContain("t.type === 'welcome'")
  })

  it('uses WELCOME_TAB constant', () => {
    expect(raw).toContain('WELCOME_TAB')
  })
})

// Phase 138 / NAV-02: Remote sidebar item removed.
// App.tsx must NOT contain Remote Sessions wiring anymore.
describe('NAV-02: Remote sidebar item is removed (Phase 138)', () => {
  it('does NOT wire onOpenRemoteSessions to Sidebar', () => {
    expect(raw).not.toContain('onOpenRemoteSessions=')
  })

  it('does NOT contain t.type === remote-sessions routing', () => {
    expect(raw).not.toContain("t.type === 'remote-sessions'")
  })
})

// Phase 138 / NAV-03: Sessions sidebar item removed (DaemonManagerPanel retired).
// App.tsx must NOT contain Sessions/Daemon Manager wiring anymore.
describe('NAV-03: Sessions sidebar item is removed (Phase 138)', () => {
  it('does NOT wire onOpenDaemonManager to Sidebar', () => {
    expect(raw).not.toContain('onOpenDaemonManager=')
  })

  it('does NOT contain t.type === daemon-manager routing', () => {
    expect(raw).not.toContain("t.type === 'daemon-manager'")
  })
})

// Phase 138 / NAV-04: New Session sidebar item removed — creation lives solely on Hub's HubFilterBar.
// App.tsx must NOT pass onAdd={handleAddTab} to Sidebar, and handleAddTab must be gone.
describe('NAV-04: New Session sidebar item is removed (Phase 138)', () => {
  it('does NOT wire onAdd={handleAddTab} to Sidebar', () => {
    expect(raw).not.toContain('onAdd={handleAddTab}')
  })

  it('does NOT define handleAddTab callback', () => {
    expect(raw).not.toContain('const handleAddTab')
  })
})

describe('NAV-05: Settings sidebar button opens Settings tab', () => {
  it('wires onSettings prop to Sidebar', () => {
    expect(raw).toContain('onSettings={')
  })

  it('find-or-create pattern checks t.type === settings', () => {
    expect(raw).toContain("t.type === 'settings'")
  })

  it('uses SETTINGS_TAB constant', () => {
    expect(raw).toContain('SETTINGS_TAB')
  })

  it('renders SettingsTab component', () => {
    expect(raw).toContain('<SettingsTab')
  })
})

describe('TAB-01: Tab bar has no action buttons', () => {
  it('TabBar JSX block does not receive onAdd prop', () => {
    // Extract the TabBar JSX block from the raw source
    const tabBarStart = raw.indexOf('<TabBar')
    const tabBarEnd = raw.indexOf('/>', tabBarStart)
    expect(tabBarStart).toBeGreaterThan(-1)
    const tabBarBlock = raw.slice(tabBarStart, tabBarEnd + 2)
    expect(tabBarBlock).not.toContain('onAdd')
  })

  it('TabBar JSX block does not receive onSettings prop', () => {
    const tabBarStart = raw.indexOf('<TabBar')
    const tabBarEnd = raw.indexOf('/>', tabBarStart)
    expect(tabBarStart).toBeGreaterThan(-1)
    const tabBarBlock = raw.slice(tabBarStart, tabBarEnd + 2)
    expect(tabBarBlock).not.toContain('onSettings')
  })

  it('TabBar JSX block does not receive onOpenDaemonManager prop', () => {
    const tabBarStart = raw.indexOf('<TabBar')
    const tabBarEnd = raw.indexOf('/>', tabBarStart)
    expect(tabBarStart).toBeGreaterThan(-1)
    const tabBarBlock = raw.slice(tabBarStart, tabBarEnd + 2)
    expect(tabBarBlock).not.toContain('onOpenDaemonManager')
  })
})
