import React from 'react'

interface LocalNetworkBannerProps {
  visible: boolean
  tailscaleConnected: boolean
  tailscaleInstalled: boolean
  onOpenURL: (url: string) => void
}

/**
 * LocalNetworkBanner shows a persistent nudge when the web server is running in
 * local network mode (not Tailscale). Three states:
 *
 * 1. tailscaleConnected=true — Tailscale healthy, server upgrading automatically
 * 2. tailscaleInstalled=true — Tailscale binary found but daemon not connected;
 *    user needs to start Tailscale, not install it
 * 3. neither — Tailscale not found; show install CTA
 *
 * Rendered above the sidebar+content row (never inside terminal-container).
 * Visible only when webServerMode === 'local'.
 */
export function LocalNetworkBanner({ visible, tailscaleConnected, tailscaleInstalled, onOpenURL }: LocalNetworkBannerProps): React.ReactElement | null {
  if (!visible) return null

  if (tailscaleConnected) {
    return (
      <div className="local-network-banner" role="status">
        <span className="local-network-banner__icon">{'\u26a0'}</span>
        <span className="local-network-banner__message">
          Local network mode active &mdash; upgrading to Tailscale&hellip;
        </span>
        <span className="local-network-banner__sub">
          Tailscale detected. Web server is restarting with end-to-end encryption.
        </span>
      </div>
    )
  }

  if (tailscaleInstalled) {
    return (
      <div className="local-network-banner" role="status">
        <span className="local-network-banner__icon">{'\u26a0'}</span>
        <span className="local-network-banner__message">
          Local network mode active &mdash; your sessions are accessible on your LAN.
        </span>
        <span className="local-network-banner__sub">
          Tailscale is installed but not connected. Start Tailscale for end-to-end encrypted remote access.
        </span>
      </div>
    )
  }

  return (
    <div className="local-network-banner" role="status">
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
    </div>
  )
}
