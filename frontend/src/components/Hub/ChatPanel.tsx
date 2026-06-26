/**
 * ChatPanel.tsx — Phase 154-05 RED STUB
 *
 * All exports are stubs that throw. Tests must fail in RED phase.
 * Replace with full implementation in GREEN phase.
 */
import type React from 'react'
import type { ChatMessage } from '../../lib/relayClient'

export type VirtualItem =
  | { type: 'message'; message: ChatMessage; isConsecutive: boolean }
  | { type: 'separator'; label: string; isoDate: string }

export interface UnreadState {
  count: number
  hasMention: boolean
}

export interface ChatPanelProps {
  sessionId: string
  relayPort: number
  open: boolean
  currentUserTailnetID?: string
  onUnreadChange?: (count: number, hasMention: boolean) => void
}

// RED STUB — not implemented

export function buildItems(_messages: ChatMessage[]): VirtualItem[] {
  throw new Error('buildItems: not implemented')
}

export function mergeWithDedup(
  _current: ChatMessage[],
  _incoming: ChatMessage[],
  _seenIds: Set<string>,
): ChatMessage[] {
  throw new Error('mergeWithDedup: not implemented')
}

export function accrueUnread(
  _prev: UnreadState,
  _message: ChatMessage,
  _currentUserTailnetID: string,
): UnreadState {
  throw new Error('accrueUnread: not implemented')
}

export function getRowStyle(
  _isActiveSeparator: boolean,
  _start: number,
): React.CSSProperties {
  throw new Error('getRowStyle: not implemented')
}

export async function loadChatHistory(
  _relayPort: number,
  _sessionId: string,
): Promise<ChatMessage[]> {
  throw new Error('loadChatHistory: not implemented')
}

export function ChatPanel(_props: ChatPanelProps): React.ReactElement {
  throw new Error('ChatPanel: not implemented')
}

export default ChatPanel
