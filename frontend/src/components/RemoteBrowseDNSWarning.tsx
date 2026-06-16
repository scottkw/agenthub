import React from 'react'

interface RemoteBrowseDNSWarningProps {
  connected: boolean
  acceptDns?: boolean
  className?: string
}

/**
 * RemoteBrowseDNSWarning shows a proactive actionable banner when the local
 * Tailscale node is connected but accept-dns is disabled (acceptDns === false).
 *
 * When acceptDns is false, MagicDNS hostname resolution fails for remote peers,
 * blocking remote file browse. This banner warns the user BEFORE they attempt
 * a remote browse — naming the fix so they can act without hitting an opaque 502.
 *
 * Renders only when connected === true && acceptDns === false.
 * Returns null otherwise (including when acceptDns is undefined, i.e. daemon
 * prefs were unavailable — safe default: no spurious warning).
 *
 * DNS-03: proactive UX surface for accept-dns=false condition.
 */
export function RemoteBrowseDNSWarning({
  connected,
  acceptDns,
  className,
}: RemoteBrowseDNSWarningProps): React.ReactElement | null {
  // Only warn when Tailscale is connected AND accept-dns is explicitly false.
  // acceptDns === undefined means prefs unavailable; do not warn spuriously.
  if (!(connected === true && acceptDns === false)) {
    return null
  }

  return (
    <div
      className={`local-network-banner${className ? ' ' + className : ''}`}
      role="status"
    >
      <span className="local-network-banner__icon">{'⚠'}</span>
      <span className="local-network-banner__message">
        Enable Tailscale DNS (accept-dns) to browse remote sessions
      </span>
      <span className="local-network-banner__sub">
        Tailscale DNS (accept-dns) is disabled. MagicDNS hostnames for remote
        peers cannot be resolved. Enable accept-dns in your Tailscale client to
        browse remote file sessions.
      </span>
    </div>
  )
}
