import { describe, it, expect } from 'vitest'
import raw from '../HealthModal.tsx?raw'

describe('HealthModal', () => {
  describe('component structure', () => {
    it('exports HealthModal function', () => {
      expect(raw).toContain('export function HealthModal')
    })

    it('returns null when health is null (loading guard)', () => {
      expect(raw).toContain('health === null')
    })

    it('returns null when all health flags are true (healthy guard)', () => {
      expect(raw).toContain('isInstalled && isConnected && hasCerts')
    })

    it('renders health-modal-overlay wrapper', () => {
      expect(raw).toContain('health-modal-overlay')
    })

    it('renders modal heading "Tailscale Setup Required"', () => {
      expect(raw).toContain('Tailscale Setup Required')
    })
  })

  describe('three-state panels (HEALTH-04)', () => {
    it('renders NotInstalledPanel when !isInstalled', () => {
      expect(raw).toContain('!isInstalled')
      expect(raw).toContain('NotInstalledPanel')
    })

    it('renders NotConnectedPanel when installed but !connected', () => {
      expect(raw).toContain('isInstalled && !isConnected')
      expect(raw).toContain('NotConnectedPanel')
    })

    it('renders NoCertsPanel when connected but !hasCerts', () => {
      expect(raw).toContain('isInstalled && isConnected && !hasCerts')
      expect(raw).toContain('NoCertsPanel')
    })
  })

  describe('platform-specific instructions (HEALTH-05)', () => {
    it('checks platform === darwin for macOS instructions', () => {
      expect(raw).toContain("platform === 'darwin'")
    })

    it('checks platform === linux for Linux instructions', () => {
      expect(raw).toContain("platform === 'linux'")
    })

    it('checks platform === windows for Windows instructions', () => {
      expect(raw).toContain("platform === 'windows'")
    })

    it('contains macOS menu bar instruction', () => {
      expect(raw).toContain('menu bar')
    })

    it('contains Linux CLI install command', () => {
      expect(raw).toContain('curl -fsSL https://tailscale.com/install.sh')
    })

    it('contains Windows system tray instruction', () => {
      expect(raw).toContain('system tray')
    })
  })

  describe('NoCertsPanel details', () => {
    it('includes CT disclosure text (TLS-04)', () => {
      expect(raw).toContain('Certificate Transparency')
    })

    it('includes onCheckAgain callback', () => {
      expect(raw).toContain('onCheckAgain')
    })

    it('renders Check Again button', () => {
      expect(raw).toContain('Check Again')
    })

    it('includes cert enablement instructions', () => {
      expect(raw).toContain('tailscale.com/admin')
    })
  })
})
