/**
 * ChatMessage — renders a single message row in the chat thread.
 *
 * Rendering modes:
 *   - First-in-group: avatar, alias header, tailnet ID, and timestamp.
 *   - Consecutive: omits the header row; applies left-indent body only (CHAT-02).
 *   - @mention-of-me: three independent, colorblind-safe signals (NOTIF-02, D-05):
 *       1. Left accent bar (3 px) via CSS ::before on .chat-msg--mention
 *       2. Background tint via .chat-msg--mention class
 *       3. @you chip inline after the alias glyph
 *   - Session-inject: system-style indicator below the body (D-06):
 *       a horizontal rule + CommandLineIcon + descriptive caption text.
 *
 * Markdown rendering: react-markdown v10 + remark-gfm + rehype-sanitize.
 * Raw HTML passthrough is disabled — react-markdown v10 does not enable it
 * by default, and rehype-sanitize strips unsafe elements and attributes
 * as belt-and-suspenders protection against stored unsafe content (SEC-03).
 * The raw-HTML rehype plugin is intentionally absent from this file's imports.
 *
 * Comment-text discipline (SEC-03): unsafe inline payload strings are
 * confined to the test file. This file describes mitigations conceptually.
 */
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeSanitize from 'rehype-sanitize'
import { CommandLineIcon } from '@heroicons/react/24/outline'
import type { ChatMessage as ChatMessageData } from '../../lib/relayClient'

// ── Helpers ────────────────────────────────────────────────────────────────

/**
 * Derive a hue angle (0–359) from a tailnet ID string via a polynomial hash.
 * Used as supplementary avatar color; alias text is always the primary identity signal.
 * Exported for unit tests.
 */
export function tailnetIdToHue(tailnetID: string): number {
  let hash = 0
  for (let i = 0; i < tailnetID.length; i++) {
    hash = (hash * 31 + tailnetID.charCodeAt(i)) >>> 0
  }
  return hash % 360
}

/**
 * Format a UNIX millisecond timestamp as HH:MM (24-hour clock, local time).
 * Exported for unit tests.
 */
export function formatHHMM(ts: number): string {
  const d = new Date(ts)
  const h = String(d.getHours()).padStart(2, '0')
  const m = String(d.getMinutes()).padStart(2, '0')
  return `${h}:${m}`
}

// ── Props ──────────────────────────────────────────────────────────────────

export interface ChatMessageProps {
  /** The message data object from the relay (mirrors Go ChatMessage struct). */
  message: ChatMessageData
  /** True when this is the first message in a consecutive run from the same author. */
  isFirstInGroup: boolean
  /** True when the current user's tailnet ID appears in message.mentions. */
  isMentionOfMe: boolean
}

// ── Component ──────────────────────────────────────────────────────────────

/**
 * Renders one ChatMessage row. See file JSDoc for rendering modes.
 */
export function ChatMessage({ message, isFirstInGroup, isMentionOfMe }: ChatMessageProps) {
  const { authorID, alias, ts, content, sessionInject } = message
  const hue = tailnetIdToHue(authorID)
  const isoTime = new Date(ts).toISOString()
  const hhmm = formatHHMM(ts)

  return (
    <div
      className={`chat-msg${isMentionOfMe ? ' chat-msg--mention' : ''}`}
      role="listitem"
      aria-label={isMentionOfMe ? `${alias} mentioned you: ${content}` : undefined}
    >
      {isFirstInGroup && (
        <div className="chat-msg__header">
          {/* Avatar: supplementary color identity from tailnet ID hue; initial is primary */}
          <div
            className="chat-msg__avatar"
            style={{ background: `hsl(${hue}, 55%, 45%)` }}
            aria-hidden="true"
          >
            {alias[0]?.toUpperCase() ?? '?'}
          </div>
          <span className="chat-msg__alias">{alias}</span>
          <span className="chat-msg__tailnet-id">({authorID})</span>
          {/* @you chip: glyph channel — visible even when color is indistinguishable (D-05) */}
          {isMentionOfMe && (
            <span className="chat-msg__you-chip" aria-hidden="true">@</span>
          )}
          <time className="chat-msg__time" title={isoTime}>{hhmm}</time>
        </div>
      )}

      {/* Body: react-markdown v10 disables raw HTML passthrough by default;
          rehype-sanitize strips unsafe elements/attributes as additional mitigation (SEC-03). */}
      <div className={`chat-msg__body${!isFirstInGroup ? ' chat-msg__body--consecutive' : ''}`}>
        <Markdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeSanitize]}>
          {content}
        </Markdown>
      </div>

      {/* Session-inject indicator (D-06): system row shown below the message body
          when the message was also injected into the terminal.
          CommandLineIcon shape is the primary signal; caption text is secondary. */}
      {sessionInject && (
        <div
          className="chat-msg__inject-indicator"
          aria-label={`Injected into terminal by ${alias}`}
        >
          <hr className="chat-msg__inject-rule" />
          <span className="chat-msg__inject-caption">
            <CommandLineIcon className="chat-msg__inject-icon" aria-hidden="true" />
            → injected into terminal
          </span>
        </div>
      )}
    </div>
  )
}

export default ChatMessage
