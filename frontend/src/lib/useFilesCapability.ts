// Phase 120-02 Task 4 — capability-detection hook.
//
// Probes /api/files/list at cwd ('.') and resolves the 4-state outcome the
// FileBrowserTab uses to dispatch its empty/error/list views:
//
//   'unknown'       — client unavailable (e.g. no session yet)
//   'present'       — probe returned 200; full browser may render
//   'denied'        — viewer cap is missing files.read (UI shows upgrade hint)
//   'probe-failed'  — non-permission error (network, 401, 404, etc.); UI shows retry
//
// AbortController cancels the in-flight fetch when the tab unmounts mid-probe
// (RESEARCH Pitfall 3 — race between navigation and async resolution).

import { useCallback, useEffect, useState } from 'react'
import { FilesApiError, type FilesApiClient } from './filesApi'

export type CapabilityState = 'unknown' | 'present' | 'denied' | 'probe-failed'

export function useFilesCapability(
  client: FilesApiClient | null,
  sessionId: string,
): { state: CapabilityState; retry: () => void } {
  const [state, setState] = useState<CapabilityState>('unknown')
  // Bump this counter to force re-running the probe effect.
  const [retryNonce, setRetryNonce] = useState(0)

  const retry = useCallback(() => {
    setRetryNonce((n) => n + 1)
  }, [])

  useEffect(() => {
    if (client === null) {
      setState('unknown')
      return
    }

    let cancelled = false
    setState('unknown')

    void (async () => {
      try {
        await client.listFiles(sessionId, '.')
        if (!cancelled) setState('present')
      } catch (err) {
        if (cancelled) return
        if (err instanceof FilesApiError && err.isMissingFilesReadPerm()) {
          setState('denied')
        } else {
          setState('probe-failed')
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [client, sessionId, retryNonce])

  return { state, retry }
}
