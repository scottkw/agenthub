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

describe('TS-01: Install guidance with copy and download', () => {
  it('shows brew install command for macOS', () => {
    expect(raw).toContain('brew install --cask tailscale-app')
  })
  it('shows winget command for Windows', () => {
    expect(raw).toContain('winget install Tailscale.Tailscale')
  })
  it('shows curl install command for Linux', () => {
    expect(raw).toContain('curl -fsSL https://tailscale.com/install.sh | sh')
  })
  it('includes copy-to-clipboard handler', () => {
    expect(raw).toContain('navigator.clipboard.writeText')
  })
  it('uses onOpenURL prop not BrowserOpenURL directly', () => {
    expect(raw).not.toContain('BrowserOpenURL')
    expect(raw).toContain('onOpenURL')
  })
  it('has download links for all platforms', () => {
    expect(raw).toContain('tailscale.com/download/macos')
    expect(raw).toContain('tailscale.com/download/linux')
    expect(raw).toContain('tailscale.com/download/windows')
  })
  it('includes CopyableCommand component', () => {
    expect(raw).toContain('CopyableCommand')
  })
})

describe('TS-02: Auto-install button', () => {
  it('gates auto-install to darwin platform', () => {
    expect(raw).toContain("platform === 'darwin'")
  })
  it('uses onAutoInstall prop', () => {
    expect(raw).toContain('onAutoInstall')
  })
  it('has install progress output area', () => {
    expect(raw).toContain('health-modal__install-output')
  })
  it('has running state class', () => {
    expect(raw).toContain('health-modal__btn--auto-install--running')
  })
})

describe('TS-03: NoCerts next steps guide', () => {
  it('includes MagicDNS enable step', () => {
    expect(raw).toContain('MagicDNS')
  })
  it('links to admin DNS page', () => {
    expect(raw).toContain('login.tailscale.com/admin/dns')
  })
  it('has numbered step structure', () => {
    expect(raw).toContain('health-modal__steps')
    expect(raw).toContain('health-modal__step-number')
  })
  it('preserves CT disclosure', () => {
    expect(raw).toContain('Certificate Transparency')
  })
  it('preserves Check Again button', () => {
    expect(raw).toContain('Check Again')
  })
})
