/**
 * useChatUnreadListeners — Phase 160 / NOTIF-01 Part B
 *
 * Opens one lightweight read-only relay WS subscription per BACKGROUNDED
 * session (modal closed) and accrues unread chat counts per session.
 *
 * When a session's modal is open, ChatPanel already holds the WS and tracks
 * unread via its own internal state. This hook only subscribes to sessions
 * that are NOT currently open in the modal, preventing double-counting.
 *
 * Design choices:
 * - useRef (not useState) for per-session UnreadState accumulation so that
 *   incoming MsgChat frames do not trigger re-renders of this component.
 * - A stable sessionIdKey dep (sessions.map(s => s.id).join(',')) prevents
 *   effect re-runs on array reference churn (PATTERNS.md §1).
 * - onOutput is passed as an explicit no-op — RelayClientCallbacks.onOutput
 *   is required (not optional) in the type definition (Pitfall 4).
 */

import { useEffect, useRef } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { RelayClient } from '../../lib/relayClient'
import { accrueUnread } from './ChatPanel'
import type { UnreadState } from './ChatPanel'

export function useChatUnreadListeners(
  sessions: SessionInfo[],
  relayPort: number,
  openModalSessionId: string | null,
  isActive: boolean,
  onUnreadChange: (sessionId: string, count: number, hasMention: boolean) => void,
): void {
  // Per-session unread state accumulation. useRef avoids re-renders on every
  // incoming MsgChat frame (Pitfall 5 — only onUnreadChange propagates state up).
  const unreadRefs = useRef<Map<string, UnreadState>>(new Map())

  // Stable dep key prevents effect re-runs on array reference churn.
  const sessionIdKey = sessions.map((s) => s.id).join(',')

  useEffect(() => {
    // Idle-tab gate: no connections when the window is inactive or there are no
    // sessions to subscribe to (Pitfall 2 / isActive guard from PATTERNS.md).
    if (!isActive || sessions.length === 0) return

    const clients: RelayClient[] = []

    for (const session of sessions) {
      // Exclude the currently-open modal session (double-count guard / Pitfall 1).
      if (session.id === openModalSessionId) continue
      // Gate on valid relay port (ws://127.0.0.1:0 fails immediately / Pitfall 2).
      if (relayPort <= 0) continue

      const client = new RelayClient(relayPort, session.id, {
        // onOutput is REQUIRED by RelayClientCallbacks — pass explicit no-op.
        // Background badge listener never renders PTY output (Pitfall 4).
        onOutput: () => {},
        onChat: (message) => {
          const prev = unreadRefs.current.get(session.id) ?? { count: 0, hasMention: false }
          // 'local' = desktop owner identity (always "local" for Hub desktop).
          const next = accrueUnread(prev, message, 'local')
          unreadRefs.current.set(session.id, next)
          onUnreadChange(session.id, next.count, next.hasMention)
        },
      })

      clients.push(client)
    }

    // Cleanup: close every RelayClient opened by this effect run so that
    // removed sessions and unmounts do not leave stale WS connections (Pitfall 3 /
    // T-160-02 DoS mitigation).
    return () => {
      for (const client of clients) {
        client.close()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionIdKey, isActive])
}
