import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'
import raw from '../../components/SettingsTab.tsx?raw'

const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

describe('TRAY-01: Start-minimized toggle — Wails bindings', () => {
    it('imports GetStartMinimized from Wails bindings', () => {
        expect(raw).toContain('GetStartMinimized')
    })

    it('imports SetStartMinimized from Wails bindings', () => {
        expect(raw).toContain('SetStartMinimized')
    })
})

describe('TRAY-01: Start-minimized toggle — state variables', () => {
    it('has startMinimized state variable', () => {
        expect(raw).toContain('startMinimized')
        expect(raw).toContain('setStartMinimized')
    })

    it('has toggleLoaded state variable for flash prevention', () => {
        expect(raw).toContain('toggleLoaded')
        expect(raw).toContain('setToggleLoaded')
    })

    it('has toggleSaving state variable', () => {
        expect(raw).toContain('toggleSaving')
        expect(raw).toContain('setToggleSaving')
    })

    it('has toggleError state variable', () => {
        expect(raw).toContain('toggleError')
        expect(raw).toContain('setToggleError')
    })
})

describe('TRAY-01: Start-minimized toggle — mount behavior', () => {
    it('calls GetStartMinimized() in useEffect on mount', () => {
        expect(raw).toContain('GetStartMinimized()')
    })

    it('sets toggleLoaded true after GetStartMinimized resolves', () => {
        expect(raw).toContain('setToggleLoaded(true)')
    })
})

describe('TRAY-01: Start-minimized toggle — handler', () => {
    it('has handleToggleMinimized async function', () => {
        expect(raw).toContain('async function handleToggleMinimized')
    })

    it('calls await SetStartMinimized before setStartMinimized (non-optimistic)', () => {
        // Find the positions of both calls to verify ordering.
        const awaitPos = raw.indexOf('await SetStartMinimized(next)')
        const setPos = raw.indexOf('setStartMinimized(next)')
        expect(awaitPos).toBeGreaterThan(-1)
        expect(setPos).toBeGreaterThan(-1)
        expect(awaitPos).toBeLessThan(setPos)
    })
})

describe('TRAY-01: Start-minimized toggle — JSX structure', () => {
    it('renders Behavior section heading', () => {
        // Phase 104: header carries id="settings-behavior" anchor for the jump-bar.
        expect(raw).toMatch(/<h3[^>]*>Behavior<\/h3>/)
    })

    it('has "Start minimized to system tray" toggle label text', () => {
        expect(raw).toContain('Start minimized to system tray')
    })

    it('has toggle-row CSS class on label', () => {
        expect(raw).toContain('settings-panel__toggle-row')
    })

    it('has toggle-track CSS class', () => {
        expect(raw).toContain('settings-panel__toggle-track')
    })

    it('has toggle-thumb CSS class', () => {
        expect(raw).toContain('settings-panel__toggle-thumb')
    })

    it('has toggle-input class on hidden checkbox', () => {
        expect(raw).toContain('settings-panel__toggle-input')
    })

    it('has toggle description text', () => {
        expect(raw).toContain('When enabled, AgentHub launches with the window hidden.')
    })

    it('has toggle-row--checked modifier for checked state', () => {
        expect(raw).toContain('settings-panel__toggle-row--checked')
    })

    it('has toggle-label class', () => {
        expect(raw).toContain('settings-panel__toggle-label')
    })
})

describe('TRAY-01: Start-minimized toggle — CSS rules', () => {
    it('has .settings-panel__toggle-track with correct dimensions', () => {
        expect(cssRaw).toContain('.settings-panel__toggle-track')
        expect(cssRaw).toContain('width: 36px')
        expect(cssRaw).toContain('height: 20px')
    })

    it('has .settings-panel__toggle-row--checked modifier', () => {
        expect(cssRaw).toContain('.settings-panel__toggle-row--checked')
    })

    it('has .settings-panel__toggle-label class', () => {
        expect(cssRaw).toContain('.settings-panel__toggle-label')
    })
})
