// Phase 125-03 Task 1 — useFilesWrite hook.
//
// Manages the write side of file operations for the editor and write affordances.
// Shape: { write, del, rename, mkdir, upload, isSaving, saveError }
//
// `write` + three-state save (idle/saving/saved ~1.5s) are fully implemented here.
// `del`, `rename`, `mkdir`, `upload` are stub signatures for Plan 04/05 — declared
// so consumers compile with the full return type.
//
// Save state machine (EDIT-05/06/08):
//   idle   → write() called → isSaving=true
//   saving → 200           → isSaving=false, savedSnapshot updated, Saved ~1.5s
//   saving → 412           → isConflict=true (buffer NOT cleared — T-125-08 locked)
//   saving → other error   → saveError set (inline non-takeover — EDIT-06 copy)
//
// The `etag` passed to write() is the value returned by readFileText() verbatim
// (PATTERNS §If-Match echo contract). The client NEVER derives it from mtime+size
// directly — the server ETag header is the single source of truth (RESEARCH Open Q6).

import { useCallback, useRef, useState } from 'react'
import { FilesApiError, REMOTE_PEER_OUTDATED_MESSAGE, type FilesApiClient } from './filesApi'

/** ~1.5s transient for the "Saved" state (EDIT-06, mirrors Settings save transient). */
const SAVED_TIMEOUT = 1500

/** Three-state save indicator state. Idle = no indicator shown. */
export type SaveState = 'idle' | 'saving' | 'saved'

/**
 * Discriminated outcome from write(). Callers MUST branch on this value
 * rather than reading the async isConflict state after await (WR-02: stale
 * closure bug — React state updates are async; reading isConflict immediately
 * after await still sees the pre-write value).
 */
export type WriteOutcome = 'saved' | 'conflict' | 'error' | 'peer-outdated'

export interface UseFilesWriteResult {
  /**
   * Trigger a PUT /api/files/write with If-Match. Sets isSaving during the request.
   * Returns a discriminated outcome so callers can branch synchronously (WR-02).
   */
  write: (path: string, content: string, etag?: string) => Promise<WriteOutcome>
  /** Delete a file or directory (stub — Plan 04). */
  del: (path: string) => Promise<void>
  /** Rename/move a file (stub — Plan 04). */
  rename: (oldPath: string, newPath: string) => Promise<void>
  /** Create a directory (stub — Plan 04). */
  mkdir: (path: string) => Promise<void>
  /** Upload a file via XHR with progress. Pass overwrite=true to skip collision check (WR-07). */
  upload: (dir: string, file: File, onProgress?: (pct: number) => void, overwrite?: boolean) => Promise<void>
  /** True while a save PUT is in flight. */
  isSaving: boolean
  /** Three-state save indicator: idle / saving / saved. */
  saveState: SaveState
  /**
   * Non-null when the last save produced a non-412 error.
   * Copy verbatim: "Couldn't save the file. Your changes are still here — try again."
   * Cleared on next successful save.
   */
  saveError: string | null
  /**
   * True when the last save attempt hit a 412 Precondition Failed.
   * The editor buffer is NEVER cleared in this state — T-125-08 locked decision.
   * Cleared when the user resolves the conflict (force-overwrite / save-as / discard).
   */
  isConflict: boolean
  /** Clear the conflict flag (called after the user resolves the ConflictModal). */
  clearConflict: () => void
  /** Clear the save error (called on retry). */
  clearSaveError: () => void
}

export function useFilesWrite(
  client: FilesApiClient | null,
  sessionId: string,
): UseFilesWriteResult {
  const [isSaving, setIsSaving] = useState<boolean>(false)
  const [saveState, setSaveState] = useState<SaveState>('idle')
  const [saveError, setSaveError] = useState<string | null>(null)
  const [isConflict, setIsConflict] = useState<boolean>(false)

  // Timer ref for the ~1.5s "Saved" transient — cancelled if a new save starts.
  const savedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const clearSavedTimer = useCallback(() => {
    if (savedTimerRef.current !== null) {
      clearTimeout(savedTimerRef.current)
      savedTimerRef.current = null
    }
  }, [])

  const write = useCallback(
    async (path: string, content: string, etag?: string): Promise<WriteOutcome> => {
      if (client === null) return 'error'

      // Clear prior transient and error state.
      clearSavedTimer()
      setSaveError(null)
      setIsConflict(false)
      setIsSaving(true)
      setSaveState('saving')

      try {
        // Send the raw CM6 buffer as octet-stream; no CRLF→LF re-encode (T-125-09).
        // If-Match echoes the ETag verbatim from readFileText (EDIT-05 echo contract).
        await client.writeFile(sessionId, path, content, etag)

        // Success — show "Saved" for ~1.5s then return to idle.
        setIsSaving(false)
        setSaveState('saved')
        savedTimerRef.current = setTimeout(() => {
          setSaveState('idle')
          savedTimerRef.current = null
        }, SAVED_TIMEOUT)
        // WR-02: return discriminated outcome so callers don't read stale isConflict state.
        return 'saved'
      } catch (err) {
        setIsSaving(false)

        if (err instanceof FilesApiError && err.isConflict()) {
          // 412 — another process modified the file. Buffer is NOT cleared (T-125-08).
          // Signal the parent to open ConflictModal.
          setIsConflict(true)
          setSaveState('idle')
          return 'conflict'
        }

        // RMW-04: upstream 405 = remote peer is running v3.4 (no write routes).
        // Buffer is NOT cleared (T-125-08 locked). Returns distinct 'peer-outdated'
        // outcome so callers can surface the verbatim version-gate message.
        if (err instanceof FilesApiError && err.isMethodNotAllowed()) {
          setSaveError(REMOTE_PEER_OUTDATED_MESSAGE)
          setSaveState('idle')
          return 'peer-outdated'
        }

        // Non-412 / non-405 error — show inline save error (EDIT-06 verbatim copy).
        setSaveError("Couldn't save the file. Your changes are still here — try again.")
        setSaveState('idle')
        return 'error'
      }
    },
    [client, sessionId, clearSavedTimer],
  )

  // ─── Plan 04: del / rename / mkdir real implementations ─────────────────────

  /**
   * Delete a file or directory. The server handles recursive directory delete
   * within the os.Root sandbox. The caller is responsible for showing the
   * DeleteConfirmModal (with file count for directories) before calling this.
   *
   * Throws FilesApiError on server errors (404 / 403 / 500).
   */
  const del = useCallback(
    async (path: string): Promise<void> => {
      if (client === null) return
      await client.del(sessionId, path)
    },
    [client, sessionId],
  )

  /**
   * Rename a file/directory, or move it across directories (a rename with a
   * different parent path). Cross-directory move calls rename with both the
   * oldRel and newRel paths — the server validates BOTH via validateAndClean
   * (T-125-10, Phase 123 FSW-02).
   *
   * Throws FilesApiError(409) on name collision (fs.ErrExist → 409).
   */
  const rename = useCallback(
    async (oldPath: string, newPath: string): Promise<void> => {
      if (client === null) return
      await client.rename(sessionId, oldPath, newPath)
    },
    [client, sessionId],
  )

  /**
   * Create a directory (mkdir). Throws FilesApiError(409) on collision.
   */
  const mkdir = useCallback(
    async (path: string): Promise<void> => {
      if (client === null) return
      await client.mkdir(sessionId, path)
    },
    [client, sessionId],
  )

  // ─── Plan 05: upload — XHR with per-file progress (EDIT-10) ─────────────────

  /**
   * Upload a single file via XHR to the current directory.
   *
   * Uses XMLHttpRequest for per-file upload.onprogress events — fetch has no
   * upload-progress API (PATTERNS §Upload is the exception to fetchOrThrow).
   *
   * Throws FilesApiError(409) on collision (caller shows CollisionConfirmModal),
   * FilesApiError(413) on over-cap (caller surfaces the skip message).
   */
  const upload = useCallback(
    async (dir: string, file: File, onProgress?: (pct: number) => void, overwrite?: boolean): Promise<void> => {
      if (client === null) return
      await client.uploadFile(sessionId, dir, file, onProgress, overwrite)
    },
    [client, sessionId],
  )

  const clearConflict = useCallback(() => {
    setIsConflict(false)
  }, [])

  const clearSaveError = useCallback(() => {
    setSaveError(null)
  }, [])

  return {
    write,
    del,
    rename,
    mkdir,
    upload,
    isSaving,
    saveState,
    saveError,
    isConflict,
    clearConflict,
    clearSaveError,
  }
}
