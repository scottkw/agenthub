import React from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'

interface LocalNetworkBannerProps {
  visible: boolean
  tailscaleConnected: boolean
  tailscaleInstalled: boolean
  tailscaleBinaryFound: boolean
  tailscaleDaemonUp: boolean
  platformHint: string
  onOpenURL: (url: string) => void
  onDismiss?: () => void
  className?: string
}

/**
 * LocalNetworkBanner shows a persistent nudge when the web server is running in
 * local network mode (not Tailscale). Four states:
 *
 * 1. tailscaleConnected=true — Tailscale healthy, server upgrading automatically
 * 2. tailscaleDaemonUp=true — Daemon running but not connected to a network
 * 3. tailscaleBinaryFound=true — Binary found but daemon not running (text only, no buttons per D-06)
 * 4. none — Tailscale not found; show install CTA
 *
 * Rendered above the sidebar+content row (never inside terminal-container).
 * Visible only when webServerMode === 'local'.
 */
export function LocalNetworkBanner({ visible, tailscaleConnected, tailscaleInstalled: _tailscaleInstalled, tailscaleBinaryFound, tailscaleDaemonUp, platformHint, onOpenURL, onDismiss, className }: LocalNetworkBannerProps): React.ReactElement | null {
  if (!visible) return null

  if (tailscaleConnected) {
    return (
      <div className={`local-network-banner${className ? ' ' + className : ''}`} role="status">
        <span className="local-network-banner__icon">{'\u26a0'}</span>
        <span className="local-network-banner__message">
          Local network mode active &mdash; upgrading to Tailscale&hellip;
        </span>
        <span className="local-network-banner__sub">
          Tailscale detected. Web server is restarting with end-to-end encryption.
        </span>
        {onDismiss && (
          <button
            type="button"
            className="local-network-banner__dismiss"
            aria-label="Dismiss local network notification"
            onClick={onDismiss}
          >
            <XMarkIcon style={{ width: 16, height: 16 }} />
          </button>
        )}
      </div>
    )
  }

  if (tailscaleDaemonUp) {
    return (
      <div className={`local-network-banner${className ? ' ' + className : ''}`} role="status">
        <span className="local-network-banner__icon">{'\u26a0'}</span>
        <span className="local-network-banner__message">
          Local network mode active &mdash; Tailscale is not connected.
        </span>
        <span className="local-network-banner__sub">
          Tailscale daemon is running but not connected to a network. Connect Tailscale for end-to-end encrypted remote access.
        </span>
        {onDismiss && (
          <button
            type="button"
            className="local-network-banner__dismiss"
            aria-label="Dismiss local network notification"
            onClick={onDismiss}
          >
            <XMarkIcon style={{ width: 16, height: 16 }} />
          </button>
        )}
      </div>
    )
  }

  if (tailscaleBinaryFound) {
    return (
      <div className={`local-network-banner${className ? ' ' + className : ''}`} role="status">
        <span className="local-network-banner__icon">{'\u26a0'}</span>
        <span className="local-network-banner__message">
          Local network mode active &mdash; Tailscale daemon not running.
        </span>
        <span className="local-network-banner__sub">
          {platformHint === 'darwin' && 'Open Tailscale from Applications or the menu bar.'}
          {platformHint === 'linux' && 'Run: sudo systemctl start tailscaled'}
          {platformHint === 'windows' && 'Open Tailscale from the Start menu or system tray.'}
          {!['darwin', 'linux', 'windows'].includes(platformHint) && 'Start the Tailscale daemon.'}
        </span>
        {onDismiss && (
          <button
            type="button"
            className="local-network-banner__dismiss"
            aria-label="Dismiss local network notification"
            onClick={onDismiss}
          >
            <XMarkIcon style={{ width: 16, height: 16 }} />
          </button>
        )}
      </div>
    )
  }

  return (
    <div className={`local-network-banner${className ? ' ' + className : ''}`} role="status">
      <span className="local-network-banner__icon">{'\u26a0'}</span>
      <span className="local-network-banner__message">
        Local network mode active &mdash; your sessions are accessible on your LAN.
      </span>
      <span className="local-network-banner__sub">
        Install Tailscale for end-to-end encrypted remote access.
      </span>
      <button
        className="local-network-banner__cta"
        onClick={() => onOpenURL('https://tailscale.com/download')}
        aria-label="Install Tailscale — opens tailscale.com in browser"
      >
        Install Tailscale
      </button>
      {onDismiss && (
        <button
          type="button"
          className="local-network-banner__dismiss"
          aria-label="Dismiss local network notification"
          onClick={onDismiss}
        >
          <XMarkIcon style={{ width: 16, height: 16 }} />
        </button>
      )}
    </div>
  )
}
