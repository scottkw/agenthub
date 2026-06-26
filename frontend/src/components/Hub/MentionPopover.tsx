/**
 * MentionPopover — @ autocomplete popover with @session pinned first.
 *
 * MENTION-01 / D-07:
 *   - @session is ALWAYS the first option, in its own "Agent" section.
 *   - @session is NEVER filtered away regardless of the `filter` prop.
 *   - Section 2 participants are filtered by `filter` (case-insensitive prefix/substring on alias).
 *
 * Keyboard navigation (Escape global listener is owned here; Arrow/Enter index tracking
 * is owned by the ChatPanel composer — see plan 154-06):
 *   - Escape fires onClose (global keydown listener, SessionCard pattern).
 *
 * Positioning: bottom-anchored to the composer wrapper. The parent is expected to
 * render this inside a `position:relative` wrapper; the popover uses
 * `position:absolute; bottom:100%` (see .mention-popover in style.css).
 * A `style`/`className` passthrough prop is accepted for parent-controlled positioning.
 *
 * Security (T-154-07 / T-154-08):
 *   - @session is structurally in its own section — a participant cannot alias as "@session"
 *     and be confused with the agent row (D-07 / T-154-07).
 *   - Aliases render as React text nodes (auto-escaped, never dangerouslySetInnerHTML) (T-154-08).
 */
import { useEffect } from 'react'
import { CommandLineIcon } from '@heroicons/react/24/outline'
import type { PresenceEntry } from '../../lib/relayClient'

/** Deterministic avatar hue from tailnetID — same algorithm as ChatMessage.tsx */
function tailnetIdToHue(tailnetID: string): number {
  let hash = 0
  for (let i = 0; i < tailnetID.length; i++) {
    hash = (hash * 31 + tailnetID.charCodeAt(i)) >>> 0
  }
  return hash % 360
}

export interface MentionPopoverProps {
  /** Live session participants. @session is always added above these. */
  participants: PresenceEntry[]
  /** Text typed after "@". Used to filter participants (NOT @session). */
  filter: string
  /**
   * Currently active index.
   * 0 = @session.
   * 1..N = filtered participant at index (activeIndex - 1).
   * Managed by the parent (ChatPanel composer, plan 154-06).
   */
  activeIndex: number
  /** Called with "@session" or "@{alias}" when an item is selected. */
  onSelect: (alias: string) => void
  /** Called when Escape is pressed. */
  onClose: () => void
  /** Optional style passthrough for parent-controlled positioning. */
  style?: React.CSSProperties
  /** Optional className passthrough. */
  className?: string
}

export function MentionPopover({
  participants,
  filter,
  activeIndex,
  onSelect,
  onClose,
  style,
  className,
}: MentionPopoverProps) {
  // Filter participants by prefix/substring on alias (case-insensitive).
  // @session is NOT in this list — it is always rendered first (D-07).
  const filtered = filter
    ? participants.filter((p) =>
        p.alias.toLowerCase().includes(filter.toLowerCase()),
      )
    : participants

  // Global Escape listener — closes the popover even when focus is in the textarea.
  // Pattern from SessionCard keyboard-dismissable dropdown (PATTERNS.md lines 303–313).
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  return (
    <div
      role="listbox"
      className={`mention-popover${className ? ` ${className}` : ''}`}
      style={style}
    >
      {/* ── Section 1: @session — ALWAYS first, ALWAYS shown, never filtered (D-07) ── */}
      <div className="mention-popover__section-label" aria-hidden="true">
        Agent
      </div>
      <div
        role="option"
        className={`mention-popover__item mention-popover__item--session${activeIndex === 0 ? ' mention-popover__item--active' : ''}`}
        aria-selected={activeIndex === 0}
        onClick={() => onSelect('@session')}
      >
        {/* CommandLineIcon: shape is the primary signal for the inject nature (D-07).
            Color (--hub-accent) is reinforcement only. */}
        <CommandLineIcon className="mention-popover__session-icon" aria-hidden="true" />
        <span className="mention-popover__alias">@session</span>
        <span className="mention-popover__desc">Inject into terminal</span>
      </div>

      {/* ── Divider between Agent section and participants ── */}
      <hr className="mention-popover__divider" />

      {/* ── Section 2: filtered live participants ── */}
      {filtered.map((p, i) => {
        // activeIndex 0 = @session; participant i has activeIndex = i + 1
        const participantActiveIndex = i + 1
        const isActive = activeIndex === participantActiveIndex
        const hue = tailnetIdToHue(p.tailnetID)
        return (
          <div
            key={p.personKey}
            role="option"
            className={`mention-popover__item mention-popover__item--participant${isActive ? ' mention-popover__item--active' : ''}`}
            aria-selected={isActive}
            onClick={() => onSelect(`@${p.alias}`)}
          >
            {/* Avatar: 20×20px circle with deterministic hue; alias initial */}
            <span
              className="mention-popover__avatar"
              style={{ background: `hsl(${hue}, 55%, 45%)` }}
              aria-hidden="true"
            >
              {p.alias[0]?.toUpperCase()}
            </span>
            <span className="mention-popover__alias">{p.alias}</span>
            <span className="mention-popover__tailnet-id">{p.tailnetID}</span>
          </div>
        )
      })}
    </div>
  )
}
