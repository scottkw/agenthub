import type { SessionInfo } from '../wailsjs/go/main/App'

/**
 * HubStatus — the canonical set of display statuses for Hub session cards and filters.
 * Used by SessionCard, HubFilterBar, and HubPanel to derive a consistent status from
 * the raw SessionInfo fields.
 *
 * WR-01: extracted from three duplicate implementations in SessionCard.tsx,
 * HubFilterBar.tsx, and HubPanel.tsx. Single source of truth.
 */
export type HubStatus = 'running' | 'idle' | 'waiting' | 'errored' | 'stopped-ok' | 'stopped-err'

/**
 * Derive the Hub display status from a SessionInfo.
 * - state === 'stopped' + exitCode non-zero → 'stopped-err'
 * - state === 'stopped' + exitCode 0 (or undefined) → 'stopped-ok'
 * - otherwise: return session.status cast to HubStatus (running/idle/waiting/errored)
 *
 * HubFilter extends HubStatus with 'all', so this function is safe to use
 * for filtering contexts — the returned value is always a valid HubFilter
 * (excluding 'all', which is the identity/no-filter sentinel).
 */
export function deriveHubStatus(s: SessionInfo): HubStatus {
  if (s.state === 'stopped') {
    return (s.exitCode ?? 0) !== 0 ? 'stopped-err' : 'stopped-ok'
  }
  return s.status as HubStatus
}

/* ATTN-01: canonical attention predicate — waiting, errored, or non-zero-exit sessions need attention */
export function isAttentionStatus(status: HubStatus): boolean {
  return status === 'waiting' || status === 'errored' || status === 'stopped-err'
}
