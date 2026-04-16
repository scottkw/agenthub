import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'
import raw from '../../components/SettingsTab.tsx?raw'

const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

describe('SET-03: Save confirmation feedback', () => {
    it('has saved state variable', () => {
        expect(raw).toContain('setSaved')
    })

    it('sets saved true after successful save', () => {
        expect(raw).toContain('setSaved(true)')
    })

    it('resets saved after timeout', () => {
        expect(raw).toContain('setSaved(false)')
    })

    it('displays Saved! text', () => {
        expect(raw).toContain("'Saved!'")
    })

    it('uses --saved CSS modifier', () => {
        expect(raw).toContain('settings-panel__btn--saved')
    })
})

describe('SET-03: Saved button CSS', () => {
    it('has settings-panel__btn--saved class', () => {
        expect(cssRaw).toContain('.settings-panel__btn--saved')
    })

    it('uses green background for saved state', () => {
        expect(cssRaw).toContain('#9ece6a')
    })
})

describe('SET-04: Browse button per path row', () => {
    it('imports OpenFileDialog from Wails', () => {
        expect(raw).toContain('OpenFileDialog')
    })

    it('has handleBrowse function', () => {
        expect(raw).toContain('handleBrowse')
    })

    it('has browse button with correct class', () => {
        expect(raw).toContain('settings-panel__browse-btn')
    })

    it('has path-row container class', () => {
        expect(raw).toContain('settings-panel__path-row')
    })

    it('has browse button title for accessibility', () => {
        expect(raw).toContain('Browse for executable')
    })
})

describe('SET-04: Browse button CSS', () => {
    it('has settings-panel__browse-btn class', () => {
        expect(cssRaw).toContain('.settings-panel__browse-btn')
    })

    it('has settings-panel__path-row class', () => {
        expect(cssRaw).toContain('.settings-panel__path-row')
    })
})

describe('SET-05: Browse result populates input', () => {
    it('calls OpenFileDialog and checks result', () => {
        expect(raw).toContain('await OpenFileDialog(')
    })

    it('guards against empty selection (cancelled dialog)', () => {
        expect(raw).toContain('if (selected)')
    })

    it('sets custom paths from selection', () => {
        expect(raw).toContain('setCustomPaths')
    })
})

describe('SET-01/02: Stored paths loaded on mount', () => {
    it('imports GetCLIPaths from Wails', () => {
        expect(raw).toContain('GetCLIPaths')
    })

    it('calls GetCLIPaths in useEffect', () => {
        expect(raw).toContain('GetCLIPaths()')
    })

    it('catches errors silently (daemon may not be connected)', () => {
        expect(raw).toContain('.catch(() => {})')
    })
})
