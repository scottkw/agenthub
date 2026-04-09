import React from 'react'

interface LocalNetworkBannerProps {
  visible: boolean
  onOpenURL: (url: string) => void
}

/**
 * LocalNetworkBanner shows a persistent nudge when the web server is running in
 * local network mode (not Tailscale). Recommends installing Tailscale for
 * end-to-end encrypted remote access.
 *
 * Rendered above the sidebar+content row (never inside terminal-container).
 * Visible only when webServerMode === 'local'.
 */
export function LocalNetworkBanner({ visible, onOpenURL }: LocalNetworkBannerProps): React.ReactElement | null {
  if (!visible) return null

  return (
    <div className="local-network-banner" role="status">
      <span className="local-network-banner__icon">{'\u26a0'}</span>
      <span className="local-network-banner__message">
        Local network mode active — your sessions are accessible on your LAN.
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
