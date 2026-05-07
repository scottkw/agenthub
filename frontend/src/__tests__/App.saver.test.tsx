import { describe, it, expect } from 'vitest'
import raw from '../App.tsx?raw'
import terminalPanelRaw from '../components/TerminalPanel.tsx?raw'
import tabBarRaw from '../components/TabBar.tsx?raw'

describe('Phase 97 SER-01: App.tsx saver-registry round-trip — Plan 97-03 implements', () => {
  it('App.tsx imports stripAnsi from ./lib/stripAnsi', () => {
    expect.fail('RED scaffold — Plan 97-03 implements App.tsx import { stripAnsi } from "./lib/stripAnsi"')
  })
  it('App.tsx imports sanitizeFilename from ./lib/sanitizeFilename', () => {
    expect.fail('RED scaffold — Plan 97-03 implements App.tsx import { sanitizeFilename } from "./lib/sanitizeFilename"')
  })
  it('App.tsx imports SaveTerminalSession from wailsjs binding', () => {
    expect.fail('RED scaffold — Plan 97-03 implements App.tsx import { SaveTerminalSession } from "./wailsjs/go/main/App"')
  })
  it('App.tsx declares saver-registry state (serializerRegistry or equivalent Map<sessionId, () => string>)', () => {
    expect.fail('RED scaffold — Plan 97-03 implements serializerRegistry useState (97-RESEARCH §Pattern 2: Saver Registry)')
  })
  it('App.tsx declares handleRegisterSaver useCallback', () => {
    expect.fail('RED scaffold — Plan 97-03 implements handleRegisterSaver(sessionId, fn|null)')
  })
  it('App.tsx declares handleRequestSave useCallback that calls SaveTerminalSession after stripAnsi + sanitizeFilename', () => {
    expect.fail('RED scaffold — Plan 97-03 implements handleRequestSave(tabId) → registry lookup → stripAnsi → sanitizeFilename → SaveTerminalSession')
  })
  it('handleRequestSave shows a banner when registry is empty (Serialize OFF)', () => {
    expect.fail('RED scaffold — Plan 97-03 implements no-saver-registered banner branch (97-RESEARCH §Whether the Save menu item appears when Serialize is toggled OFF)')
  })
  it('TerminalPanel.tsx accepts onRegisterSaver?: (sessionId, fn) => void prop', () => {
    // This assertion uses terminalPanelRaw — ensures Plan 97-04 wires through.
    expect.fail('RED scaffold — Plan 97-04 implements onRegisterSaver prop on TerminalPanel')
  })
  it('TabBar.tsx accepts onRequestSave?: (tabId: string) => void prop', () => {
    // This assertion uses tabBarRaw — ensures Plan 97-04 wires through.
    expect.fail('RED scaffold — Plan 97-04 implements onRequestSave prop on TabBar')
  })
})

// Satisfy the linter: these imports are used by the describe block above.
void raw
void terminalPanelRaw
void tabBarRaw
