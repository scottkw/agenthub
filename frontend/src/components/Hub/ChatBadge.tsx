/**
 * ChatBadge — unread message count badge with a colorblind-safe @mention state.
 *
 * NOTIF-01 / D-10: The mention state is signalled by the "@" glyph replacing
 * the count (shape-based, not color-based). Color is reinforcement only.
 * Count=0 renders null (no DOM node).
 */
export function ChatBadge({ count, hasMention }: { count: number; hasMention: boolean }) {
  if (count === 0) return null

  const plural = count !== 1
  const ariaLabel = hasMention
    ? `${count} unread message${plural ? 's' : ''}, including a mention`
    : `${count} unread message${plural ? 's' : ''}`

  return (
    <span
      role="status"
      className={`chat-badge${hasMention ? ' chat-badge--mention' : ''}`}
      aria-label={ariaLabel}
    >
      {/* @ glyph replaces the count when hasMention is true.
          The glyph is the primary non-color signal for the mention state (D-10).
          Color (--hub-accent background) is reinforcement only. */}
      {hasMention ? '@' : count}
    </span>
  )
}
