import React from 'react'

interface TailscaleHealth {
  installed: boolean
  connected: boolean
  hasCerts: boolean
  ip: string
  domain: string
}

interface HealthModalProps {
  health: TailscaleHealth | null
  platform: string // 'darwin' | 'linux' | 'windows'
  onCheckAgain: () => void
}

function NotInstalledPanel({ platform }: { platform: string }): React.ReactElement {
  return (
    <div className="health-modal__panel">
      <p className="health-modal__title">Tailscale is not installed or not running.</p>
      {platform === 'darwin' && (
        <>
          <p className="health-modal__text">
            Install Tailscale from the Mac App Store or tailscale.com/download.
          </p>
          <p className="health-modal__text">
            Once installed, look for the Tailscale icon in your menu bar and sign in.
          </p>
        </>
      )}
      {platform === 'linux' && (
        <>
          <p className="health-modal__text">Install Tailscale with your package manager:</p>
          <code className="health-modal__code health-modal__code--block">
            curl -fsSL https://tailscale.com/install.sh | sh
          </code>
          <p className="health-modal__text">Then run:</p>
          <code className="health-modal__code">sudo tailscale up</code>
        </>
      )}
      {platform === 'windows' && (
        <>
          <p className="health-modal__text">
            Download and install Tailscale from tailscale.com/download.
          </p>
          <p className="health-modal__text">
            Once installed, find Tailscale in the system tray and sign in.
          </p>
        </>
      )}
    </div>
  )
}

function NotConnectedPanel({ platform }: { platform: string }): React.ReactElement {
  return (
    <div className="health-modal__panel">
      <p className="health-modal__title">
        Tailscale is installed but not connected to a tailnet.
      </p>
      {platform === 'darwin' && (
        <p className="health-modal__text">
          Click the Tailscale icon in your menu bar and select Connect.
        </p>
      )}
      {platform === 'linux' && (
        <>
          <p className="health-modal__text">Run:</p>
          <code className="health-modal__code">sudo tailscale up</code>
        </>
      )}
      {platform === 'windows' && (
        <p className="health-modal__text">
          Click the Tailscale icon in your system tray and select Connect.
        </p>
      )}
    </div>
  )
}

function NoCertsPanel({
  platform: _platform,
  onCheckAgain,
}: {
  platform: string
  onCheckAgain: () => void
}): React.ReactElement {
  return (
    <>
      <div className="health-modal__panel">
        <p className="health-modal__title">
          Tailscale is connected but HTTPS certificates are not enabled.
        </p>
        <p className="health-modal__text">1. Go to the Tailscale admin console: tailscale.com/admin</p>
        <p className="health-modal__text">2. Navigate to DNS &rarr; HTTPS Certificates</p>
        <p className="health-modal__text">3. Enable HTTPS and save</p>
        <div className="ct-disclosure">
          <p className="ct-disclosure__text">
            When you enable HTTPS, Tailscale will provision a Let&apos;s Encrypt certificate for
            your device&apos;s hostname. Your device hostname will be permanently visible in public
            Certificate Transparency logs. This is normal and expected for Let&apos;s Encrypt
            certificates.
          </p>
        </div>
      </div>
      <div className="health-modal__footer">
        <button className="health-modal__btn--check" onClick={onCheckAgain}>
          Check Again
        </button>
      </div>
    </>
  )
}

export function HealthModal({
  health,
  platform,
  onCheckAgain,
}: HealthModalProps): React.ReactElement | null {
  if (health === null) return null

  const isInstalled = health.installed
  const isConnected = health.connected
  const hasCerts = health.hasCerts

  if (isInstalled && isConnected && hasCerts) return null

  return (
    <div className="health-modal-overlay">
      <div className="health-modal">
        <div className="health-modal__header">
          <h2>Tailscale Setup Required</h2>
        </div>
        <div className="health-modal__body">
          {!isInstalled && <NotInstalledPanel platform={platform} />}
          {isInstalled && !isConnected && <NotConnectedPanel platform={platform} />}
          {isInstalled && isConnected && !hasCerts && (
            <NoCertsPanel platform={platform} onCheckAgain={onCheckAgain} />
          )}
        </div>
      </div>
    </div>
  )
}
