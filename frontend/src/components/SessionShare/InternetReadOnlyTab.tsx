import React from 'react'
import { BrowserOpenURL } from '../../wailsjs/wailsjs/runtime/runtime'
import { ShareLinkCard } from './ShareLinkCard'

// ---------------------------------------------------------------------------
// Phase 173 / SM-04 + SM-06 — InternetReadOnlyTab: the public, read-only
// Funnel link tier. Extracted verbatim (behavior unchanged, D-08) from
// SessionSharePanel.tsx's INTERNET (PUBLIC) block (:531-601), swapping the
// hand-laid URL row for the reusable ShareLinkCard (SM-06). No public-write /
// command-execution UI lives here — see the SM-04 negative wall-off test in
// InternetReadOnlyTab.test.tsx.
//
// RESEARCH Pitfall 3: warming/live sub-state stays keyed off
// funnelActive/warmingUp/warmupTimedOut — never coupled to the shell's
// funnelOn state.
// ---------------------------------------------------------------------------

export interface InternetReadOnlyTabProps {
  /** true once the daemon confirms the Funnel is live (session.funnelActive). */
  funnelActive?: boolean
  /** The read-only Funnel-base public URL (obtained by re-issuing caps after warm-up). */
  funnelUrl?: string | null
  /** true between enable and the funnelActive flip — shows the TLS warm-up state. */
  warmingUp?: boolean
  /** true when warm-up did not complete within 30s. */
  warmupTimedOut?: boolean
  /**
   * Phase 170 / FNL-08: the reusable, read-only, share-lifetime public join
   * code minted by the daemon alongside the Funnel read URL. `null`/absent
   * means no code is available yet — the URL row falls back to the bare
   * /join page (guest enters a code manually) rather than a broken cap link.
   */
  publicReadCode?: string | null
  /** One-click Funnel teardown — SetSessionFunnel(id, false, 0). No confirm dialog (D-13). */
  onDisableFunnel?: () => void
}

/**
 * Renders the public read-only Internet (Funnel) tier: the reusable /join
 * entry URL as a ShareLinkCard, the reusable public join code, warming-up /
 * timed-out sub-states, and a "Disable internet share" button. No
 * public-write markup appears in this tab (SM-04).
 */
export function InternetReadOnlyTab({
  funnelActive = false,
  funnelUrl = null,
  warmingUp = false,
  warmupTimedOut = false,
  publicReadCode = null,
  onDisableFunnel,
}: InternetReadOnlyTabProps): React.ReactElement {
  // FNL-08 fix (M-46): the public URL a guest opens must be the reusable,
  // share-lifetime /join entry point (https://host/join?code=<publicReadCode>),
  // NOT funnelUrl. funnelUrl = readUrl = https://host/sessions/{id}?cap=<rTok>,
  // an ephemeral, grant-bound capability link that 401s "capability required"
  // once the grant rotates on a warm-up re-issue or the daemon restarts.
  // Derive the join URL from funnelUrl's origin — mirrors the joinURLFor()
  // pattern used by ShareLinkCard/SessionSharePanel. When no reusable code is
  // present (degenerate path) fall back to the bare /join page rather than
  // the broken cap link.
  const publicEntryUrl = ((): string | null => {
    if (!funnelUrl) return null
    try {
      const u = new URL(funnelUrl)
      const base = `${u.protocol}//${u.host}/join`
      return publicReadCode ? `${base}?code=${publicReadCode}` : base
    } catch {
      return funnelUrl
    }
  })()

  return (
    <div className="session-share-panel__tab session-share-panel__tab--internet-ro">
      <div className="hub-share-internet-section">
        <div className="hub-share-internet-section__heading">Internet (public)</div>

        {warmingUp && (
          <div className="hub-share-internet-section__warmup">Starting up… (TLS warming up)</div>
        )}

        {warmupTimedOut && (
          <p className="hub-share-internet-section__error">
            Connection timed out. Try disabling and re-enabling.
          </p>
        )}

        {funnelActive &&
          publicEntryUrl &&
          !warmingUp &&
          (publicReadCode ? (
            <ShareLinkCard
              title="Public URL (read-only)"
              url={publicEntryUrl}
              code={publicReadCode}
              codeLabel="Public join code (reusable):"
              description="Anyone with this link or code can watch the live session over the public internet — read-only, no file access, no input."
            />
          ) : (
            // Degenerate path (D-08 preserved): no reusable code minted yet —
            // fall back to the bare /join page instead of a broken cap link.
            <div className="hub-share-internet-section__url-row">
              <span className="session-share-panel__label">Public URL (read-only):</span>
              <span className="session-share-panel__url" title={publicEntryUrl}>
                {publicEntryUrl}
              </span>
              <div className="session-share-panel__actions">
                <button
                  type="button"
                  className="daemon-panel__btn"
                  onClick={() => BrowserOpenURL(publicEntryUrl)}
                  aria-label="Open public internet URL in browser"
                >
                  Open
                </button>
              </div>
            </div>
          ))}

        <button
          type="button"
          className="hub-share-internet-section__disable"
          onClick={() => onDisableFunnel?.()}
        >
          Disable internet share
        </button>
      </div>
    </div>
  )
}
