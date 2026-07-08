import React from 'react'
import { ShareLinkCard } from './ShareLinkCard'

// ---------------------------------------------------------------------------
// Phase 173 / SM-04 + SM-06 — TailnetTab: private tailnet-only link tier.
// Extracted verbatim (behavior unchanged, D-08) from SessionSharePanel.tsx's
// Read-Only (:438) and Full Access (:482) rows, swapping the hand-laid rows
// for the reusable ShareLinkCard (SM-06). No public-write UI lives here —
// see the SM-04 negative wall-off test in TailnetTab.test.tsx.
// ---------------------------------------------------------------------------

export interface TailnetTabProps {
  readURL: string
  writeURL: string
  readCode: string
  writeCode: string
  /**
   * Phase 137 / D-11: optional display hint for scope text only — no gating
   * logic. true → "Watch + browse files" / "…plus file browsing and editing."
   * false → "Watch only — no file access" / "…Use the toggle above to also
   * allow file editing."
   */
  browseEnabled?: boolean
}

/**
 * Renders the two private tailnet-only link cards: Read-Only Link and Full
 * Access Link. Each ShareLinkCard carries its own scope description directly
 * beneath it (SM-06). No public-write / command-execution markup appears in
 * this tab (SM-04).
 */
export function TailnetTab({
  readURL,
  writeURL,
  readCode,
  writeCode,
  browseEnabled = false,
}: TailnetTabProps): React.ReactElement {
  return (
    <div className="session-share-panel__tab session-share-panel__tab--tailnet">
      <ShareLinkCard
        title="Read-Only Link"
        url={readURL}
        code={readCode}
        description={
          browseEnabled
            ? 'Watch the live session and browse files read-only — cannot send input.'
            : 'Watch the live session only — cannot send input or browse files.'
        }
      />
      <ShareLinkCard
        title="Full Access Link"
        url={writeURL}
        code={writeCode}
        description={
          browseEnabled
            ? 'Full control of the live session (send input) plus file browsing and editing.'
            : 'Full control of the live session (send input) plus file browsing. Use the toggle above to also allow file editing.'
        }
      />
    </div>
  )
}
