import { describe, it, expect } from 'vitest'
import raw from '../../components/SplashScreen.tsx?raw'

describe('SplashScreen (BRND-02)', () => {
  it('exports SplashScreen function component', () => {
    expect(raw).toContain('export function SplashScreen')
  })

  it('accepts done prop of type boolean', () => {
    expect(raw).toContain('done: boolean')
  })

  it('renders img with /agenthub-title-logo.png src', () => {
    expect(raw).toContain('/agenthub-title-logo.png')
  })

  it('uses position fixed overlay with z-index 9999', () => {
    expect(raw).toContain('zIndex: 9999')
  })

  it('uses dark background matching app theme', () => {
    expect(raw).toContain('#1a1b26')
  })

  it('sets opacity to 0 when done is true (fade-out)', () => {
    expect(raw).toContain('opacity: done ? 0 : 1')
  })

  it('uses CSS transition for fade-out', () => {
    expect(raw).toContain('opacity 0.3s ease')
  })

  it('sets pointerEvents none to avoid blocking interaction', () => {
    expect(raw).toContain("pointerEvents: 'none'")
  })

  it('hides static splash element on mount', () => {
    expect(raw).toContain("getElementById('splash-static')")
  })

  it('returns null when not visible (unmounts after fade)', () => {
    expect(raw).toContain('if (!visible) return null')
  })

  it('delays unmount by 300ms after done becomes true', () => {
    expect(raw).toContain('setTimeout(() => setVisible(false), 300)')
  })
})
