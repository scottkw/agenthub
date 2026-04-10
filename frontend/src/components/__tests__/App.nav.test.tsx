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

describe('NAV-02: Remote sidebar button opens Remote Sessions panel', () => {
  it('wires onOpenRemoteSessions={handleOpenRemoteSessions} to Sidebar', () => {
    expect(raw).toContain('onOpenRemoteSessions={handleOpenRemoteSessions}')
  })

  it('find-or-create pattern checks t.type === remote-sessions', () => {
    expect(raw).toContain("t.type === 'remote-sessions'")
  })
})

describe('NAV-03: Sessions sidebar button opens Daemon Manager panel', () => {
  it('wires onOpenDaemonManager={handleOpenDaemonManager} to Sidebar', () => {
    expect(raw).toContain('onOpenDaemonManager={handleOpenDaemonManager}')
  })

  it('find-or-create pattern checks t.type === daemon-manager', () => {
    expect(raw).toContain("t.type === 'daemon-manager'")
  })
})

describe('NAV-04: New Session sidebar button opens new-session modal', () => {
  it('wires onAdd={handleAddTab} to Sidebar', () => {
    expect(raw).toContain('onAdd={handleAddTab}')
  })

  it('handleAddTab triggers setShowNewSessionModal(true)', () => {
    expect(raw).toContain('setShowNewSessionModal(true)')
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
