/**
 * ChatDaySeparator — renders a day-label row in the chat virtualizer list.
 *
 * Layout: centered date label flanked by horizontal rule lines on each side.
 * The row height is 28px (non-scale value — legibility floor for 11px label).
 *
 * CRITICAL: This component does NOT set position:sticky. The parent
 * ChatPanel virtualizer applies sticky positioning to the wrapper div so
 * that the active day separator floats at the top of the scroll container.
 * Applying sticky here would conflict with the virtualizer's transform-based
 * absolute positioning on non-active rows (RESEARCH Pitfall 1).
 *
 * Date formatting: formatDaySeparator returns "Today" / "Yesterday" /
 * locale-short (e.g. "Mon, Jun 23") — see helper below.
 */
import React from 'react'

// ── Date formatting helper ─────────────────────────────────────────────────

/**
 * Format a UNIX millisecond timestamp into a day separator label.
 *
 * Returns:
 *   "Today"     — if the timestamp falls within the current calendar day (local time)
 *   "Yesterday" — if the timestamp falls within the previous calendar day
 *   locale-short — e.g. "Mon, Jun 23" for any earlier date
 *
 * Exported for unit tests.
 */
export function formatDaySeparator(tsMs: number): string {
  const date = new Date(tsMs)

  // Midnight of today (local time)
  const today = new Date()
  today.setHours(0, 0, 0, 0)

  // Midnight of yesterday (local time)
  const yesterday = new Date(today.getTime() - 86_400_000)

  // Midnight of the message's date (local time)
  const msgDay = new Date(date.getFullYear(), date.getMonth(), date.getDate())

  if (msgDay.getTime() === today.getTime()) return 'Today'
  if (msgDay.getTime() === yesterday.getTime()) return 'Yesterday'

  return new Intl.DateTimeFormat('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
  }).format(date)
}

// ── Component ──────────────────────────────────────────────────────────────

export interface ChatDaySeparatorProps {
  /** Pre-formatted label, e.g. "Today", "Yesterday", "Mon, Jun 23". */
  label: string
}

/**
 * Presentational day separator row. Receives a pre-formatted label string.
 * Sticky positioning is NOT applied here — see file JSDoc.
 */
export function ChatDaySeparator({ label }: ChatDaySeparatorProps) {
  return (
    <div className="chat-day-sep" aria-label={`Messages from ${label}`}>
      <hr className="chat-day-sep__rule" aria-hidden="true" />
      <span className="chat-day-sep__label">{label}</span>
      <hr className="chat-day-sep__rule" aria-hidden="true" />
    </div>
  )
}

export default ChatDaySeparator
