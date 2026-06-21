import React from 'react'
import { PlusIcon } from '@heroicons/react/24/outline'

// ---- Props ----

export interface HubEmptyStateProps {
  /**
   * 'no-sessions' — the session list is empty; offer to create one.
   * 'no-matches' — sessions exist but none match the current filter/search;
   *                offer to clear the filter.
   */
  variant: 'no-sessions' | 'no-matches'
  /** Fired when the user clicks "New session" (no-sessions variant only). */
  onNewSession?: () => void
  /** Fired when the user clicks "Clear filter" (no-matches variant only). */
  onClearFilter?: () => void
}

// ---- Component ----

/**
 * HubEmptyState — two-variant empty state component for the Hub grid.
 *
 * All copy is from the UI-SPEC Copywriting Contract:
 *   no-sessions: "No sessions yet" / "Create a session to start an AI coding agent." / "New session"
 *   no-matches:  "No matching sessions" / "Clear the filter or search to see all sessions." / "Clear filter"
 *
 * CSS class names:
 *   .hub__empty-state   — container
 *   .hub__empty-heading — heading (h2, 16px/600)
 *   .hub__empty-body    — body text (12px/400)
 *   .hub__empty-cta     — call-to-action button
 */
export function HubEmptyState({
  variant,
  onNewSession,
  onClearFilter,
}: HubEmptyStateProps): React.ReactElement {
  if (variant === 'no-sessions') {
    return (
      <div className="hub__empty-state">
        <h2 className="hub__empty-heading">No sessions yet</h2>
        <p className="hub__empty-body">Create a session to start an AI coding agent.</p>
        <button className="hub__empty-cta" onClick={onNewSession} type="button">
          <PlusIcon className="hub__empty-cta-icon" aria-hidden="true" />
          New session
        </button>
      </div>
    )
  }

  // variant === 'no-matches'
  return (
    <div className="hub__empty-state">
      <h2 className="hub__empty-heading">No matching sessions</h2>
      <p className="hub__empty-body">Clear the filter or search to see all sessions.</p>
      <button className="hub__empty-cta" onClick={onClearFilter} type="button">
        Clear filter
      </button>
    </div>
  )
}
