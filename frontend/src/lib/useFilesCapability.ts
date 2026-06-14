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
//
// Phase 125-02 extends the hook to also resolve canWrite:
//   - Desktop (non-remote, auth-less daemon socket): derived from the
//     SessionInfo.filesWrite signal passed in as `filesWriteSignal`. Probing
//     the write route does NOT work here — the daemon socket is auth-less and
//     always returns 200 regardless of the owner toggle (RESEARCH Pitfall 2).
//   - Web-share (remote, capToken present): probe the write route via a HEAD
//     request; a 403 with body containing "files.write" means the cap lacks
//     files.write perm (isMissingFilesWritePerm). Any other response (200, 404,
//     etc.) is treated as "can write" — the server enforces the real gate.

import { useCallback, useEffect, useState } from 'react'
import { FilesApiError, type FilesApiClient } from './filesApi'

export type CapabilityState = 'unknown' | 'present' | 'denied' | 'probe-failed'

/**
 * 403 with body containing 'files.write' → viewer cap is missing the write perm.
 * Mirrors isMissingFilesReadPerm for the write route (requireFilesWrite body contract).
 */
function isMissingFilesWritePerm(err: unknown): boolean {
  if (!(err instanceof FilesApiError)) return false
  if (err.status !== 403) return false
  return err.bodyText.toLowerCase().includes('files.write')
}

export function useFilesCapability(
  client: FilesApiClient | null,
  sessionId: string,
  /**
   * Phase 125-02 canWrite source.
   *
   * - Pass `true`/`false` on the desktop surface: this is the
   *   SessionInfo.filesWrite signal from the daemon (auth-less socket — probe
   *   does not reflect the owner toggle, RESEARCH Pitfall 2).
   * - Pass `undefined` on web-share: the hook will probe the write route via
   *   the existing capToken in the client.
   * - Pass `null` when canWrite is not relevant for this surface.
   */
  filesWriteSignal?: boolean | null,
): { state: CapabilityState; retry: () => void; canWrite: boolean } {
  const [state, setState] = useState<CapabilityState>('unknown')
  const [canWrite, setCanWrite] = useState<boolean>(false)
  // Bump this counter to force re-running the probe effect.
  const [retryNonce, setRetryNonce] = useState(0)

  const retry = useCallback(() => {
    setRetryNonce((n) => n + 1)
  }, [])

  // ─── canRead probe (files.read) ───
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

  // ─── canWrite resolution ───
  //
  // Desktop (filesWriteSignal is boolean): derive directly from the daemon signal.
  // Web-share (filesWriteSignal is undefined): probe the write route.
  // filesWriteSignal is null: canWrite stays false (capability not relevant).
  useEffect(() => {
    // Desktop path: use the daemon-provided signal directly.
    if (filesWriteSignal === true) {
      setCanWrite(true)
      return
    }
    if (filesWriteSignal === false || filesWriteSignal === null) {
      setCanWrite(false)
      return
    }

    // Web-share probe path (filesWriteSignal is undefined):
    // HEAD the write route with a dummy path; map 403-with-files.write → denied.
    if (client === null) {
      setCanWrite(false)
      return
    }

    let cancelled = false

    void (async () => {
      try {
        // Probe: HEAD a non-existent path — the perm check runs before path
        // resolution, so a missing-perm 403 fires before 404.
        await client.probeWrite(sessionId, '.')
        if (!cancelled) setCanWrite(true)
      } catch (err) {
        if (cancelled) return
        if (isMissingFilesWritePerm(err)) {
          setCanWrite(false)
        } else {
          // Any other error (404, 405, network) means the cap IS permitted but
          // the specific operation failed — treat as canWrite=true so the UI
          // shows affordances (server is the real authority anyway, EDIT-12).
          setCanWrite(true)
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [client, sessionId, filesWriteSignal, retryNonce])

  return { state, retry, canWrite }
}
