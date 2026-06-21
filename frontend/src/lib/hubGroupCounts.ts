/* hubGroupCounts.ts — shared count helpers extracted from GroupSidebar.tsx (POL-05) */
/* ATTN-06: attention is a superset of waiting (waiting | errored | stopped-err) */
import { memberKey } from './hubGroups'
import { deriveHubStatus, isAttentionStatus } from './hubStatus'
import type { SessionInfo } from '../wailsjs/go/main/App'

export interface GroupCounts {
  running: number
  total: number
  waiting: number
  attention: number  // ATTN-06: superset of waiting (waiting | errored | stopped-err)
}

export function computeCounts(sessions: SessionInfo[], memberKeys: Set<string>): GroupCounts {
  let running = 0
  let total = 0
  let waiting = 0
  let attention = 0
  for (const s of sessions) {
    const key = memberKey(s.name, s.workDir)
    if (!memberKeys.has(key)) continue
    total++
    const st = deriveHubStatus(s)
    if (st === 'running' || st === 'idle' || st === 'waiting') running++
    if (st === 'waiting') waiting++
    if (isAttentionStatus(st)) attention++
  }
  return { running, total, waiting, attention }
}

export function computeGlobalCounts(sessions: SessionInfo[]): GroupCounts {
  let running = 0
  let waiting = 0
  let attention = 0
  for (const s of sessions) {
    const st = deriveHubStatus(s)
    if (st === 'running' || st === 'idle' || st === 'waiting') running++
    if (st === 'waiting') waiting++
    if (isAttentionStatus(st)) attention++
  }
  return { running, total: sessions.length, waiting, attention }
}
