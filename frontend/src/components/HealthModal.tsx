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
  onOpenURL: (url: string) => void
  onAutoInstall?: () => void
  installProgress: string[]
  installStatus: 'idle' | 'running' | 'success' | 'error'
  installError?: string
  webServerRunning?: boolean
}

const MACOS_INSTALL_CMD = 'brew install --cask tailscale-app'
const MACOS_DOWNLOAD_URL = 'https://tailscale.com/download/macos'
const LINUX_INSTALL_CMD = 'curl -fsSL https://tailscale.com/install.sh | sh'
const LINUX_DOWNLOAD_URL = 'https://tailscale.com/download/linux'
const WINDOWS_INSTALL_CMD = 'winget install Tailscale.Tailscale'
const WINDOWS_DOWNLOAD_URL = 'https://tailscale.com/download/windows'

function CopyableCommand({ command }: { command: string }): React.ReactElement {
  const [copied, setCopied] = React.useState(false)
  const handleCopy = () => {
    void navigator.clipboard.writeText(command).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }
  return (
    <div className="health-modal__copy-row">
      <code className="health-modal__code health-modal__code--block">{command}</code>
      <button
        className={copied ? 'health-modal__btn--copy health-modal__btn--copy--active' : 'health-modal__btn--copy'}
        onClick={handleCopy}
      >
        {copied ? 'Copied!' : 'Copy'}
      </button>
    </div>
  )
}

function NotInstalledPanel({
  platform,
  onOpenURL,
  onAutoInstall,
  installProgress,
  installStatus,
  installError: _installError,
}: {
  platform: string
  onOpenURL: (url: string) => void
  onAutoInstall?: () => void
  installProgress: string[]
  installStatus: 'idle' | 'running' | 'success' | 'error'
  installError?: string
}): React.ReactElement {
  return (
    <div className="health-modal__panel">
      <p className="health-modal__title">Tailscale is not installed or not running.</p>
      {platform === 'darwin' && (
        <>
          <p className="health-modal__text">Install with Homebrew:</p>
          <CopyableCommand command={MACOS_INSTALL_CMD} />
          <p className="health-modal__text" style={{ marginTop: '8px' }}>
            <a
              className="health-modal__download-link"
              onClick={() => onOpenURL(MACOS_DOWNLOAD_URL)}
            >
              Download for Mac
            </a>
          </p>
          {onAutoInstall && (
            <p className="health-modal__text" style={{ marginTop: '8px' }}>
              <button
                className={
                  installStatus === 'running'
                    ? 'health-modal__btn--auto-install health-modal__btn--auto-install--running'
                    : 'health-modal__btn--auto-install'
                }
                onClick={onAutoInstall}
                disabled={installStatus === 'running'}
              >
                {installStatus === 'running' ? 'Installing...' : 'Try Auto-Install'}
              </button>
            </p>
          )}
          {installStatus !== 'idle' && (
            <pre
              className={
                installStatus === 'success'
                  ? 'health-modal__install-output health-modal__install-output--success'
                  : installStatus === 'error'
                  ? 'health-modal__install-output health-modal__install-output--error'
                  : 'health-modal__install-output'
              }
            >
              {installProgress.join('\n')}
            </pre>
          )}
          {installStatus === 'success' && (
            <p className="health-modal__text" style={{ marginTop: '8px' }}>
              Next: open Tailscale from your menu bar and sign in, then click Check Again.
            </p>
          )}
          {installStatus === 'error' && (
            <p className="health-modal__text" style={{ marginTop: '8px' }}>
              Auto-install failed. Use the manual command above, or download directly.
            </p>
          )}
        </>
      )}
      {platform === 'linux' && (
        <>
          <p className="health-modal__text">Install with the official script:</p>
          <CopyableCommand command={LINUX_INSTALL_CMD} />
          <p className="health-modal__text" style={{ marginTop: '8px' }}>
            <a
              className="health-modal__download-link"
              onClick={() => onOpenURL(LINUX_DOWNLOAD_URL)}
            >
              Download for Linux
            </a>
          </p>
        </>
      )}
      {platform === 'windows' && (
        <>
          <p className="health-modal__text">Install with winget:</p>
          <CopyableCommand command={WINDOWS_INSTALL_CMD} />
          <p className="health-modal__text" style={{ marginTop: '8px' }}>
            <a
              className="health-modal__download-link"
              onClick={() => onOpenURL(WINDOWS_DOWNLOAD_URL)}
            >
              Download for Windows
            </a>
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
  onOpenURL,
}: {
  platform: string
  onCheckAgain: () => void
  onOpenURL: (url: string) => void
}): React.ReactElement {
  return (
    <>
      <div className="health-modal__panel">
        <p className="health-modal__title">
          Tailscale is connected but HTTPS certificates are not enabled.
        </p>
        <ol className="health-modal__steps">
          <li className="health-modal__step">
            <span className="health-modal__step-number">1</span>
            <span>
              Go to the Tailscale admin console:{' '}
              <a
                className="health-modal__download-link"
                onClick={() => onOpenURL('https://login.tailscale.com/admin/dns')}
              >
                DNS settings
              </a>
            </span>
          </li>
          <li className="health-modal__step">
            <span className="health-modal__step-number">2</span>
            <span>Enable MagicDNS if it is not already enabled.</span>
          </li>
          <li className="health-modal__step">
            <span className="health-modal__step-number">3</span>
            <span>Under HTTPS Certificates, click Enable HTTPS.</span>
          </li>
          <li className="health-modal__step">
            <span className="health-modal__step-number">4</span>
            <span>
              Acknowledge the Certificate Transparency disclosure — your device hostname will
              appear in public CT logs.
            </span>
          </li>
          <li className="health-modal__step">
            <span className="health-modal__step-number">5</span>
            <span>
              Return here and click Check Again — AgentHub will detect certs automatically.
            </span>
          </li>
        </ol>
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
  onOpenURL,
  onAutoInstall,
  installProgress,
  installStatus,
  installError,
  webServerRunning,
}: HealthModalProps): React.ReactElement | null {
  if (health === null) return null

  // When web server is running (even in local mode), don't block the UI.
  // The nudge banner handles the "suggest Tailscale" messaging.
  if (webServerRunning) return null

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
          {!isInstalled && (
            <NotInstalledPanel
              platform={platform}
              onOpenURL={onOpenURL}
              onAutoInstall={onAutoInstall}
              installProgress={installProgress}
              installStatus={installStatus}
              installError={installError}
            />
          )}
          {isInstalled && !isConnected && <NotConnectedPanel platform={platform} />}
          {isInstalled && isConnected && !hasCerts && (
            <NoCertsPanel platform={platform} onCheckAgain={onCheckAgain} onOpenURL={onOpenURL} />
          )}
        </div>
      </div>
    </div>
  )
}
