// Phase 120-04 Task 2 — FileBrowserTab orchestrator.
//
// Top-level tab component composing breadcrumb + list + preview + status.
// Owns:
//   - directory state (path + entries + truncated + refreshedAt)
//   - preview state (PreviewState discriminated union)
//   - sort state (key + dir)
//   - filter state (active + value)
//   - capability state (via useFilesCapability)
//   - listError dispatch for non-permission failures
//
// Race protection: every async effect captures the requesting path at
// dispatch time and ignores resolutions for stale paths (RESEARCH Pitfall 3).
// AbortController cancels in-flight HTTP on cleanup.
//
// Capability denial (files.read missing) replaces the entire tab body with
// PermissionDeniedTakeover — breadcrumb / list / preview do not render.
//
// Document-level '/' filter activation: a keydown listener is mounted iff
// isActive, mirroring isXtermFocused-style gating.

// Phase 125-03: Editor + save flow + unsaved-changes guard + 412 conflict modal.
// All navigation triggers (file-switch, navigate-up, tab-close) route through
// guardThen() which opens UnsavedChangesModal when the buffer is dirty.
// NO beforeunload — Wails blocks it; guard is entirely React-level (EDIT-07).

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { FilesApiClient, FilesApiError, type FileEntry } from '../lib/filesApi'
import { useFilesCapability } from '../lib/useFilesCapability'
import { useFilesWrite } from '../lib/useFilesWrite'
import type { BreadcrumbSegment, PreviewState, SortKey, SortDir } from '../lib/filesTypes'
import { humanSize } from '../lib/humanSize'
import { BreadcrumbBar } from './FileBrowser/BreadcrumbBar'
import { FileListPane } from './FileBrowser/FileListPane'
import { StatusLine } from './FileBrowser/StatusLine'
import { PreviewPane } from './FileBrowser/PreviewPane'
import { PermissionDeniedTakeover } from './FileBrowser/PermissionDeniedTakeover'
import { EnableWebSharingTakeover } from './FileBrowser/EnableWebSharingTakeover'
import { NetworkErrorState } from './FileBrowser/NetworkErrorState'
import { EmptyDirectoryState } from './FileBrowser/EmptyDirectoryState'
import { sortEntries } from './FileBrowser/sortEntries'
import { Editor } from './Editor'
import { EditorHeader } from './FileBrowser/EditorHeader'
import { UnsavedChangesModal } from './FileBrowser/modals/UnsavedChangesModal'
import { ConflictModal } from './FileBrowser/modals/ConflictModal'
import { DeleteConfirmModal } from './FileBrowser/modals/DeleteConfirmModal'
import { CollisionConfirmModal } from './FileBrowser/modals/CollisionConfirmModal'
import { MoveToPickerModal } from './FileBrowser/modals/MoveToPickerModal'
import { InlineNameInput } from './FileBrowser/InlineNameInput'

export interface FileBrowserTabProps {
  sessionId: string
  sessionName: string
  /** True when this tab is currently active in App.tsx. Gates the '/' key listener. */
  isActive: boolean
  /** True when the session is being browsed via the webserver (remote/web-share). */
  isRemote: boolean
  /** Base URL — daemon-loopback for local, HTTPS for remote. */
  baseURL: string
  /** Capability token — present iff isRemote (web-share cap reuse per CONTEXT D-02). */
  capToken?: string
  /**
   * Phase 122-03 — optional path-prefix forwarded to FilesApiClient. Defaults
   * to `/api/files`. For the desktop GUI's remote-session path, App.tsx passes
   * `/api/files/remote/{sessionId}` so the local-daemon proxy route is hit
   * instead of a (cross-origin-blocked) direct fetch to the remote peer.
   */
  pathPrefix?: string
  /**
   * Phase 122-03 — optional callback invoked when the user clicks "Re-enter
   * join code" in the EnableWebSharingTakeover (rendered when isRemote and
   * the fetch returns 401). Required for the remote-session path; absent
   * for local-session browsing.
   */
  onReenterJoinCode?: () => void
}

/**
 * Build the per-session singleton tab id used in App.tsx find-or-add logic.
 * Exported so tests can pin the format without importing the rest of the file.
 */
export function fileBrowserTabId(sessionId: string): string {
  return `__files__${sessionId}`
}

/** Join two path segments, treating '.' as the cwd root. */
function joinPath(base: string, name: string): string {
  if (base === '.' || base === '') return name
  return `${base}/${name}`
}

/** Path of the parent directory. Returns '.' at root. */
function dirname(path: string): string {
  if (path === '.' || path === '') return '.'
  const idx = path.lastIndexOf('/')
  if (idx <= 0) return '.'
  return path.slice(0, idx)
}

/** Build breadcrumb segments from a cwd-relative path. */
function segmentsFor(path: string): BreadcrumbSegment[] {
  if (path === '.' || path === '') {
    return [{ name: 'session', pathFromCwd: '.' }]
  }
  const parts = path.split('/').filter(Boolean)
  const segs: BreadcrumbSegment[] = [{ name: 'session', pathFromCwd: '.' }]
  for (let i = 0; i < parts.length; i++) {
    segs.push({
      name: parts[i],
      pathFromCwd: parts.slice(0, i + 1).join('/'),
    })
  }
  return segs
}

/** True iff candidate is the same as current or a strict prefix of current. */
function isPrefixOrEqual(candidate: string, current: string): boolean {
  if (candidate === '.' || candidate === '') return true
  if (candidate === current) return true
  return current.startsWith(`${candidate}/`)
}

type ListError =
  | 'idle'
  | 'permission-denied'
  | 'not-found'
  | 'network-error'
  | 'not-authorized'
  | 'enable-web-sharing'

const MARKDOWN_EXT = /\.(md|markdown)$/i
const IMAGE_EXT = /\.(png|jpe?g|webp|gif|svg)$/i

export function FileBrowserTab({
  sessionId,
  sessionName: _sessionName,
  isActive,
  isRemote,
  baseURL,
  capToken,
  pathPrefix,
  onReenterJoinCode,
}: FileBrowserTabProps): React.ReactElement {
  // FilesApiClient construction — memoized so identity is stable across renders.
  const client = useMemo(
    () => new FilesApiClient({ baseURL, capToken, pathPrefix }),
    [baseURL, capToken, pathPrefix],
  )

  const { state: capState, retry: retryCapability, canWrite } = useFilesCapability(client, sessionId)

  // Phase 125-03/04: write hook — write/del/rename/mkdir/upload + save state.
  const {
    write: writeFile,
    del,
    rename,
    mkdir,
    isSaving,
    saveState,
    saveError,
    isConflict,
    clearConflict,
    clearSaveError,
  } = useFilesWrite(client, sessionId)

  // ─── Editor state (EDIT-02, EDIT-05/06/07/08) ───
  // `editingEntry` — the FileEntry being edited (null = preview/browse mode).
  // `editContent`  — current CM6 buffer content (updated via onSave/onDirty).
  // `editEtag`     — ETag echoed from readFileText; used as If-Match on save.
  // `editDirty`    — true when buffer ≠ saved snapshot.
  const [editingEntry, setEditingEntry] = useState<FileEntry | null>(null)
  const [editContent, setEditContent] = useState<string>('')
  const [editEtag, setEditEtag] = useState<string | undefined>(undefined)
  const [editDirty, setEditDirty] = useState<boolean>(false)

  // Unsaved-changes modal state (EDIT-07)
  const [unsavedModalOpen, setUnsavedModalOpen] = useState<boolean>(false)
  // Deferred action that will run after the unsaved-changes guard resolves.
  const pendingActionRef = useRef<(() => void) | null>(null)

  // Conflict modal state (EDIT-08) — opened by useFilesWrite when 412 fires.
  // isConflict from useFilesWrite drives the modal open state directly.

  // ─── Plan 04: write affordance state (EDIT-09) ────────────────────────────────
  // delete modal
  const [deleteTarget, setDeleteTarget] = useState<FileEntry | null>(null)
  const [deleteFileCount, setDeleteFileCount] = useState<number>(0)
  // collision modal — shown on 409 from rename/create/mkdir
  const [collisionModalOpen, setCollisionModalOpen] = useState<boolean>(false)
  const [collisionName, setCollisionName] = useState<string>('')
  // Pending retry function for collision (re-issue with force semantics)
  const collisionRetryRef = useRef<(() => Promise<void>) | null>(null)
  // move-to picker modal
  const [moveTarget, setMoveTarget] = useState<FileEntry | null>(null)
  // inline name input mode: null = hidden
  type InlineMode = 'create-file' | 'new-folder' | 'rename'
  const [inlineMode, setInlineMode] = useState<InlineMode | null>(null)
  const [inlineTarget, setInlineTarget] = useState<FileEntry | null>(null) // for rename
  // generic write operation error (e.g. 500 on delete/mkdir)
  const [writeOpError, setWriteOpError] = useState<string | null>(null)

  // ─── guardThen: wrap any navigation action through the unsaved-changes guard ───
  // All three navigation triggers (file-switch, navigate-up, tab-close) route here.
  // NO beforeunload — Wails blocks it; this is pure React-level guarding (EDIT-07).
  const guardThen = useCallback(
    (action: () => void) => {
      if (!editDirty || editingEntry === null) {
        action()
        return
      }
      // Buffer is dirty — open the modal and park the action.
      pendingActionRef.current = action
      setUnsavedModalOpen(true)
    },
    [editDirty, editingEntry],
  )

  // Directory state
  const [path, setPath] = useState<string>('.')
  const [entries, setEntries] = useState<FileEntry[]>([])
  const [truncated, setTruncated] = useState<boolean>(false)
  const [refreshedAt, setRefreshedAt] = useState<string | null>(null)
  const [listError, setListError] = useState<ListError>('idle')

  // Selection + preview
  const [selected, setSelected] = useState<string | null>(null)
  const [preview, setPreview] = useState<PreviewState>({ kind: 'idle' })

  // Sort + filter
  const [sortKey, setSortKey] = useState<SortKey>('name')
  const [sortDir, setSortDir] = useState<SortDir>('asc')
  const [filterActive, setFilterActive] = useState<boolean>(false)
  const [filterValue, setFilterValue] = useState<string>('')

  // Phase 120 UAT-1: resizable list/preview split. State is percent of the
  // body width occupied by the list pane; bounded to [20, 80] so neither
  // pane collapses entirely. Drag tracked via a ref + window-level mousemove
  // listener so the cursor can leave the divider mid-drag without losing
  // the grip.
  const [listWidthPct, setListWidthPct] = useState<number>(40)
  const bodyRef = useRef<HTMLDivElement | null>(null)
  const draggingRef = useRef<boolean>(false)
  const onDividerMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    draggingRef.current = true
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }, [])
  useEffect(() => {
    const onMove = (e: MouseEvent) => {
      if (!draggingRef.current || !bodyRef.current) return
      const rect = bodyRef.current.getBoundingClientRect()
      if (rect.width === 0) return
      const pct = ((e.clientX - rect.left) / rect.width) * 100
      const clamped = Math.max(20, Math.min(80, pct))
      setListWidthPct(clamped)
    }
    const onUp = () => {
      if (!draggingRef.current) return
      draggingRef.current = false
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
  }, [])

  // Refetch counter for the directory listing — bumped on Refresh / retry.
  const [dirNonce, setDirNonce] = useState<number>(0)

  const tabRef = useRef<HTMLDivElement | null>(null)

  // ─── Directory fetch effect ───
  useEffect(() => {
    if (capState !== 'present') return
    const requestPath = path
    const abort = new AbortController()
    let cancelled = false

    void (async () => {
      try {
        const resp = await client.listFiles(sessionId, requestPath)
        if (cancelled || abort.signal.aborted) return
        // Race protection — only commit if the path is still current.
        if (requestPath !== path) return
        setEntries(resp.entries)
        setTruncated(resp.truncated)
        setRefreshedAt(resp.refreshedAt ?? new Date().toISOString())
        setListError('idle')
      } catch (err) {
        if (cancelled || abort.signal.aborted) return
        if (requestPath !== path) return
        if (err instanceof FilesApiError) {
          if (err.isMissingFilesReadPerm()) {
            // Defer to the capability hook — re-run probe so the takeover
            // surfaces consistently across all error paths.
            retryCapability()
            return
          }
          if (err.isUnauthorized()) {
            // Phase 122-03 — when browsing a REMOTE session, a 401 from the
            // local-daemon proxy means the upstream cap is rejected (web-share
            // disabled remotely OR cap rotated). Surface the locked D-04
            // takeover; the user can re-enter the join code to recover.
            if (isRemote && onReenterJoinCode) {
              setListError('enable-web-sharing')
              return
            }
            setListError('not-authorized')
            return
          }
          if (err.isNotFound()) {
            setListError('not-found')
            return
          }
          if (err.isForbidden()) {
            setListError('permission-denied')
            return
          }
        }
        setListError('network-error')
      }
    })()

    return () => {
      cancelled = true
      abort.abort()
    }
  }, [client, sessionId, path, dirNonce, capState, retryCapability, isRemote, onReenterJoinCode])

  // Reset selection on path change so a stale row doesn't carry forward.
  // Also exit edit mode when navigating to a new directory.
  useEffect(() => {
    setSelected(null)
    setPreview({ kind: 'idle' })
    setFilterActive(false)
    setFilterValue('')
    setEditingEntry(null)
    setEditContent('')
    setEditDirty(false)
    setEditEtag(undefined)
  }, [path])

  // ─── Preview fetch effect ───
  useEffect(() => {
    if (selected === null) {
      setPreview({ kind: 'idle' })
      return
    }
    const entry = entries.find((e) => e.name === selected)
    if (!entry) return
    if (entry.isDir) {
      // Directories are not previewed; selecting a directory shows idle.
      setPreview({ kind: 'idle' })
      return
    }

    const requestPath = joinPath(path, selected)
    const requestSelected = selected
    const abort = new AbortController()
    let cancelled = false

    setPreview({ kind: 'loading' })

    void (async () => {
      try {
        const head = await client.headFile(sessionId, requestPath)
        if (cancelled || abort.signal.aborted) return
        if (requestSelected !== selected) return

        const { size, contentType } = head
        const downloadUrl = client.buildDownloadUrl(sessionId, requestPath)

        if (size === 0) {
          setPreview({ kind: 'empty', filename: requestSelected })
          return
        }

        const isMarkdown =
          contentType.startsWith('text/markdown') || MARKDOWN_EXT.test(requestSelected)
        const isImage =
          contentType.startsWith('image/') && IMAGE_EXT.test(requestSelected)
        const isText = contentType.startsWith('text/')

        if (isImage) {
          setPreview({
            kind: 'image',
            url: client.buildImageUrl(sessionId, requestPath),
            size,
            mtime: entry.mtime,
          })
          return
        }

        if (isMarkdown) {
          const body = await client.readFileText(sessionId, requestPath)
          if (cancelled || abort.signal.aborted) return
          if (requestSelected !== selected) return
          // Phase 125-03: capture ETag for If-Match on subsequent write (EDIT-05).
          setEditEtag(body.etag)
          setPreview({
            kind: 'markdown',
            text: body.text,
            size: body.size || size,
            mtime: entry.mtime,
          })
          return
        }

        if (isText) {
          const body = await client.readFileText(sessionId, requestPath)
          if (cancelled || abort.signal.aborted) return
          if (requestSelected !== selected) return
          // Phase 125-03: capture ETag for If-Match on subsequent write (EDIT-05).
          setEditEtag(body.etag)
          setPreview({
            kind: 'text',
            text: body.text,
            size: body.size || size,
            mtime: entry.mtime,
          })
          return
        }

        // Binary / unsupported content type — refuse with Download fallback.
        setPreview({
          kind: 'unsupported',
          filename: requestSelected,
          humanSize: humanSize(size),
          downloadUrl,
        })
      } catch (err) {
        if (cancelled || abort.signal.aborted) return
        if (requestSelected !== selected) return
        if (err instanceof FilesApiError) {
          if (err.isOverCap()) {
            const downloadUrl = client.buildDownloadUrl(sessionId, requestPath)
            setPreview({
              kind: 'over-cap',
              filename: requestSelected,
              humanSize: humanSize(entry.size || 0),
              downloadUrl,
            })
            return
          }
          if (err.isForbidden()) {
            setPreview({ kind: 'forbidden-file', filename: requestSelected })
            return
          }
          if (err.isNotFound()) {
            setPreview({
              kind: 'read-error',
              filename: requestSelected,
              message: 'not found',
              onRetry: () => setSelected((s) => (s ? `${s}` : s)),
            })
            return
          }
        }
        setPreview({
          kind: 'read-error',
          filename: requestSelected,
          message: 'network error',
          // Retry by toggling selected through the same string (React detects
          // the new reference and re-runs this effect — race-protected by the
          // requestSelected capture above).
          onRetry: () => {
            setSelected(null)
            setTimeout(() => setSelected(requestSelected), 0)
          },
        })
      }
    })()

    return () => {
      cancelled = true
      abort.abort()
    }
  }, [client, sessionId, path, selected, entries])

  // ─── Navigation (EDIT-07: all three triggers route through guardThen) ───
  const navigateInto = useCallback(
    (name: string) => {
      // Phase 120 WR-03 — UI-side defence-in-depth: reject any entry name that
      // could synthesise a sandbox-escape path on the wire. The server's
      // sandbox.ResolvePath is the actual security boundary (UI-05), but a
      // bug that ever reduced 'subdir/..' to 'subdir' server-side would
      // silently violate UI intent — never construct such a path here.
      // Disallow:
      //   - '.' / '..' (parent/self traversal)
      //   - '/' or '\\' embedded in the name (multi-segment hops the listing
      //     never legitimately returns; List returns one path component per
      //     entry).
      // Empty names are also rejected so joinPath cannot produce 'subdir/'.
      if (
        name === '' ||
        name === '.' ||
        name === '..' ||
        name.includes('/') ||
        name.includes('\\')
      ) {
        return
      }
      // Guard: if a dirty file is open, confirm before navigating into a directory.
      guardThen(() => setPath((p) => joinPath(p, name)))
    },
    [guardThen],
  )

  const navigateUp = useCallback(() => {
    // Guard: if a dirty file is open, confirm before navigating up.
    guardThen(() => setPath((p) => (p === '.' || p === '' ? p : dirname(p))))
  }, [guardThen])

  const navigateTo = useCallback(
    (segmentPath: string) => {
      // Defense-in-depth (UI-05): only accept '.' or a strict prefix of the
      // current path. The server sandbox is the actual security boundary.
      // Guard: if a dirty file is open, confirm before jumping via breadcrumb.
      guardThen(() =>
        setPath((p) => (isPrefixOrEqual(segmentPath, p) ? segmentPath : p)),
      )
    },
    [guardThen],
  )

  const refresh = useCallback(() => {
    setDirNonce((n) => n + 1)
  }, [])

  // ─── Plan 04: write affordance helpers (EDIT-09) ────────────────────────────

  /**
   * Count files recursively in a directory for the delete-confirm body copy.
   * Client-side listFiles walk — avoids a server change (RESEARCH Open Q3).
   * Returns a non-negative integer; errors treated as 0 (conservative display).
   */
  const countFilesRecursive = useCallback(
    async (dirPath: string): Promise<number> => {
      if (client === null) return 0
      let count = 0
      async function walk(p: string) {
        try {
          const resp = await client!.listFiles(sessionId, p)
          for (const e of resp.entries) {
            if (e.isDir) {
              await walk(p === '.' ? e.name : `${p}/${e.name}`)
            } else {
              count++
            }
          }
        } catch {
          // Ignore errors in recursive walk — show 0 on failure.
        }
      }
      await walk(dirPath)
      return count
    },
    [client, sessionId],
  )

  /** Open the delete confirm modal for a given entry. Counts files first for dirs. */
  const handleDeleteRequest = useCallback(
    async (entry: FileEntry) => {
      if (!canWrite) return
      if (entry.isDir) {
        const dirPath = joinPath(path, entry.name)
        const count = await countFilesRecursive(dirPath)
        setDeleteFileCount(count)
      }
      setDeleteTarget(entry)
    },
    [canWrite, path, countFilesRecursive],
  )

  /** Execute delete after confirmation. */
  const handleDeleteConfirm = useCallback(async () => {
    if (deleteTarget === null) return
    const targetPath = joinPath(path, deleteTarget.name)
    try {
      await del(targetPath)
      // Deselect if the deleted file was selected.
      if (selected === deleteTarget.name) {
        setSelected(null)
        setPreview({ kind: 'idle' })
      }
      setDeleteTarget(null)
      refresh()
    } catch {
      setWriteOpError("Couldn't complete that. Try again.")
      setDeleteTarget(null)
    }
  }, [deleteTarget, path, del, selected, refresh])

  /**
   * Open collision modal with a pending retry function.
   * The retryFn is called if the user chooses Replace.
   */
  const openCollisionModal = useCallback(
    (name: string, retryFn: () => Promise<void>) => {
      collisionRetryRef.current = retryFn
      setCollisionName(name)
      setCollisionModalOpen(true)
    },
    [],
  )

  /** Open inline name input for creating a new file. */
  const handleNewFile = useCallback(() => {
    if (!canWrite) return
    setInlineMode('create-file')
    setInlineTarget(null)
  }, [canWrite])

  /** Open inline name input for creating a new folder. */
  const handleNewFolder = useCallback(() => {
    if (!canWrite) return
    setInlineMode('new-folder')
    setInlineTarget(null)
  }, [canWrite])

  /** Handle inline name input commit (create-file / new-folder / rename). */
  const handleInlineCommit = useCallback(
    async (name: string) => {
      if (!inlineMode) return
      const trimmed = name.trim()
      if (!trimmed) {
        setInlineMode(null)
        return
      }
      try {
        if (inlineMode === 'create-file') {
          const newPath = joinPath(path, trimmed)
          // Create an empty file by writing an empty body (no If-Match = new file).
          await client?.writeFile(sessionId, newPath, '')
          setInlineMode(null)
          refresh()
        } else if (inlineMode === 'new-folder') {
          const newPath = joinPath(path, trimmed)
          await mkdir(newPath)
          setInlineMode(null)
          refresh()
        } else if (inlineMode === 'rename' && inlineTarget) {
          const oldPath = joinPath(path, inlineTarget.name)
          const newPath = joinPath(path, trimmed)
          await rename(oldPath, newPath)
          if (selected === inlineTarget.name) setSelected(trimmed)
          setInlineMode(null)
          setInlineTarget(null)
          refresh()
        }
      } catch (err) {
        if (err instanceof FilesApiError && err.isCollision()) {
          const finalName = trimmed
          const finalMode = inlineMode
          const finalTarget = inlineTarget
          openCollisionModal(trimmed, async () => {
            // Replace: re-issue with force semantics.
            // For create-file: PUT with If-Match='*' (overwrite existing).
            // For rename: we can't truly force a rename; just close and report.
            // (Server currently doesn't support a force-rename flag — no-op Replace.)
            if (finalMode === 'create-file') {
              const newPath = joinPath(path, finalName)
              await client?.writeFile(sessionId, newPath, '', '*')
              setInlineMode(null)
              refresh()
            } else if (finalMode === 'new-folder') {
              // mkdir collision: can't overwrite a directory — dismiss.
              setInlineMode(null)
            } else if (finalMode === 'rename' && finalTarget) {
              // rename collision: close inline input; user must choose a different name.
              setInlineMode(null)
              setInlineTarget(null)
            }
          })
        } else {
          setWriteOpError("Couldn't complete that. Try again.")
          setInlineMode(null)
          setInlineTarget(null)
        }
      }
    },
    [
      inlineMode, inlineTarget, path, client, sessionId, mkdir, rename,
      selected, refresh, openCollisionModal,
    ],
  )

  /** Open inline rename input for a row. */
  const handleRenameRequest = useCallback(
    (entry: FileEntry) => {
      if (!canWrite) return
      setInlineTarget(entry)
      setInlineMode('rename')
    },
    [canWrite],
  )

  /** Open the Move to… picker for a row. */
  const handleMoveRequest = useCallback(
    (entry: FileEntry) => {
      if (!canWrite) return
      setMoveTarget(entry)
    },
    [canWrite],
  )

  /** Execute move (cross-dir rename) after picker confirms. */
  const handleMoveConfirm = useCallback(
    async (destDir: string) => {
      if (moveTarget === null) return
      const oldPath = joinPath(path, moveTarget.name)
      const newPath = joinPath(destDir, moveTarget.name)
      try {
        await rename(oldPath, newPath)
        if (selected === moveTarget.name) {
          setSelected(null)
          setPreview({ kind: 'idle' })
        }
        setMoveTarget(null)
        refresh()
      } catch (err) {
        if (err instanceof FilesApiError && err.isCollision()) {
          const capturedTarget = moveTarget
          const capturedDestDir = destDir
          setMoveTarget(null)
          openCollisionModal(capturedTarget.name, async () => {
            // Replace: re-issue rename — server will overwrite existing.
            // NOTE: server rename with ErrExist; if server supports it, this retries.
            // For now, just report — server may not support force-rename.
            const old = joinPath(path, capturedTarget.name)
            const dest = joinPath(capturedDestDir, capturedTarget.name)
            await rename(old, dest)
            if (selected === capturedTarget.name) {
              setSelected(null)
              setPreview({ kind: 'idle' })
            }
            refresh()
          })
        } else {
          setWriteOpError("Couldn't complete that. Try again.")
          setMoveTarget(null)
        }
      }
    },
    [moveTarget, path, rename, selected, refresh, openCollisionModal],
  )

  // ─── Sort change ───
  const onSortChange = useCallback(
    (key: SortKey) => {
      setSortKey((cur) => {
        if (cur === key) {
          // Toggle direction on same column.
          setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
          return cur
        }
        // Default direction per column: name=asc, size=desc, modified=desc.
        setSortDir(key === 'name' ? 'asc' : 'desc')
        return key
      })
    },
    [],
  )

  // ─── Editor open handler (EDIT-02/03/12) ───
  // Called when the user clicks the Edit button in PreviewPane.
  // Captures the ETag from the already-loaded preview bytes — no re-fetch (RESEARCH anti-pattern).
  const handleEdit = useCallback(() => {
    if (selected === null) return
    const entry = entries.find((e) => e.name === selected)
    if (!entry || entry.isDir || entry.isBinary) return

    // Capture the preview text and ETag from the current preview state.
    let initialText = ''
    let etag: string | undefined = undefined
    if (preview.kind === 'text' || preview.kind === 'markdown') {
      initialText = preview.text
    }
    // ETag is stored on the preview state if we added it; fall back to undefined.
    // In the current architecture the preview fetch (readFileText) returns etag;
    // we thread it via a parallel editEtag state set in the preview effect.
    etag = editEtag

    setEditContent(initialText)
    setEditDirty(false)
    setEditingEntry(entry)
    // If the preview fetch stored the etag, it's already in editEtag state.
    // If not, we'll write with no If-Match (new-file semantics — safe fallback).
    void etag // suppress unused warning; it's read from editEtag in onSave
  }, [selected, entries, preview, editEtag])

  // ─── Save handler (EDIT-05) ───
  // Called by EditorHeader.onSave and by the Cmd-S keymap in Editor.tsx.
  const handleSave = useCallback(
    async (content?: string) => {
      if (editingEntry === null) return
      const filePath = joinPath(path, editingEntry.name)
      const body = content ?? editContent
      await writeFile(filePath, body, editEtag)
      // On success (no throw), update editContent so subsequent dirty checks are
      // against the new snapshot. The etag will be updated on the next read —
      // for now continue with the same etag (server will 412 on next conflict).
      if (!isConflict) {
        setEditContent(body)
        setEditDirty(false)
      }
    },
    [editingEntry, path, editContent, editEtag, writeFile, isConflict],
  )

  // ─── Filter helpers ───
  const onFilterActivate = useCallback(() => {
    setFilterActive(true)
  }, [])

  const onFilterDismiss = useCallback(() => {
    setFilterActive(false)
    setFilterValue('')
  }, [])

  // ─── Document-level '/' filter activation (UI-04) ───
  useEffect(() => {
    if (!isActive) return
    const onKey = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement) return
      if (!tabRef.current) return
      if (!tabRef.current.contains(e.target as Node)) return
      if (e.key === '/') {
        e.preventDefault()
        setFilterActive(true)
        return
      }
      if ((e.key === 'r' || e.key === 'R') && !filterActive) {
        e.preventDefault()
        refresh()
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [isActive, filterActive, refresh])

  // ─── Compute visible counts for the status line. ───
  const visibleCount = useMemo(() => {
    if (filterValue.length === 0) return entries.length
    const lower = filterValue.toLowerCase()
    return entries.filter((e) => e.name.toLowerCase().includes(lower)).length
  }, [entries, filterValue])

  // Sort the entries once for downstream display (FileListPane re-applies
  // filter+sort for consistency — the work is cached via its own useMemo).
  const sortedEntries = useMemo(
    () => sortEntries(entries, sortKey, sortDir),
    [entries, sortKey, sortDir],
  )

  // ─── Render ───

  if (capState === 'denied') {
    return (
      <section
        ref={tabRef}
        role="region"
        aria-label="File browser"
        className="file-browser"
        data-testid="file-browser-tab"
      >
        <PermissionDeniedTakeover />
      </section>
    )
  }

  if (capState === 'unknown' || capState === 'probe-failed') {
    if (capState === 'probe-failed') {
      return (
        <section
          ref={tabRef}
          role="region"
          aria-label="File browser"
          className="file-browser"
          data-testid="file-browser-tab"
        >
          <NetworkErrorState scope="directory" onRetry={retryCapability} />
        </section>
      )
    }
    // capState === 'unknown' — show a spinner while the initial probe runs.
    return (
      <section
        ref={tabRef}
        role="region"
        aria-label="File browser"
        className="file-browser"
        data-testid="file-browser-tab"
      >
        <div
          className="file-browser__preview--loading"
          data-testid="file-browser-cap-loading"
        >
          <div className="file-browser__spinner" aria-hidden="true" />
          <span>Loading…</span>
        </div>
      </section>
    )
  }

  // capState === 'present' — render the full browser.

  const downloadUrlForSelected =
    selected !== null && !entries.find((e) => e.name === selected)?.isDir
      ? client.buildDownloadUrl(sessionId, joinPath(path, selected))
      : null

  return (
    <section
      ref={tabRef}
      role="region"
      aria-label="File browser"
      className="file-browser"
      data-testid="file-browser-tab"
    >
      {listError === 'enable-web-sharing' && onReenterJoinCode ? (
        <EnableWebSharingTakeover onReenterJoinCode={onReenterJoinCode} />
      ) : listError === 'network-error' ? (
        <NetworkErrorState scope="directory" onRetry={refresh} />
      ) : listError === 'not-found' ? (
        <div
          className="file-browser__error"
          data-testid="file-browser-not-found"
          role="alert"
        >
          <h3 className="file-browser__error-heading">Directory not found</h3>
          <p className="file-browser__error-body">
            It may have been moved or deleted.
          </p>
          <button
            type="button"
            className="file-browser__btn file-browser__btn--secondary"
            onClick={() => navigateTo('.')}
          >
            Go to root
          </button>
        </div>
      ) : listError === 'not-authorized' ? (
        <div
          className="file-browser__error"
          data-testid="file-browser-not-authorized"
          role="alert"
        >
          <h3 className="file-browser__error-heading">Not authorized</h3>
          <p className="file-browser__error-body">
            Sign in or re-open this session to refresh credentials.
          </p>
        </div>
      ) : listError === 'permission-denied' ? (
        <div
          className="file-browser__error"
          data-testid="file-browser-dir-permission-denied"
          role="alert"
        >
          <h3 className="file-browser__error-heading">Permission denied</h3>
          <p className="file-browser__error-body">Cannot list {path}.</p>
        </div>
      ) : (
        <>
          <BreadcrumbBar
            segments={segmentsFor(path)}
            refreshedAt={refreshedAt}
            onNavigateTo={navigateTo}
            onRefresh={refresh}
            canWrite={canWrite}
            onNewFile={handleNewFile}
            onNewFolder={handleNewFolder}
          />
          <div className="file-browser__body" ref={bodyRef}>
            <div
              className="file-browser__list-container"
              style={{ width: `${listWidthPct}%` }}
            >
              {/* Phase 125-04: inline name input (create-file / new-folder / rename) */}
              {inlineMode !== null && inlineMode !== 'rename' && canWrite && (
                <InlineNameInput
                  mode={inlineMode}
                  onCommit={handleInlineCommit}
                  onCancel={() => { setInlineMode(null); setInlineTarget(null) }}
                />
              )}
              {sortedEntries.length === 0 && inlineMode === null ? (
                <EmptyDirectoryState relativePathFromCwd={path} />
              ) : (
                <FileListPane
                  entries={sortedEntries}
                  selectedName={selected}
                  sortKey={sortKey}
                  sortDir={sortDir}
                  filter={filterValue}
                  truncated={truncated}
                  isActive={isActive}
                  // Phase 125-03: file-switch routes through guardThen (EDIT-07)
                  onSelect={(name) => guardThen(() => setSelected(name))}
                  onNavigateInto={navigateInto}
                  onNavigateUp={navigateUp}
                  onSortChange={onSortChange}
                  onFilterActivate={onFilterActivate}
                  // Phase 125-04: write affordances (EDIT-09/12)
                  canWrite={canWrite}
                  onRowEdit={(entry) => {
                    guardThen(() => {
                      setSelected(entry.name)
                      // handleEdit will be triggered by the preview pane once selected
                      // For direct row edit we set selection and trigger edit
                      if (!entry.isDir && !entry.isBinary) {
                        setSelected(entry.name)
                        // If preview is loaded for this entry, open editor directly.
                        // Otherwise, selection will load the preview first.
                        if (selected === entry.name && (preview.kind === 'text' || preview.kind === 'markdown')) {
                          handleEdit()
                        }
                      }
                    })
                  }}
                  onRowRename={(entry) => {
                    // Show inline rename for this entry
                    handleRenameRequest(entry)
                  }}
                  onRowMove={(entry) => {
                    void handleMoveRequest(entry)
                  }}
                  onRowDelete={(entry) => {
                    void handleDeleteRequest(entry)
                  }}
                />
              )}
              {/* Rename inline input for a specific entry (shown inside list) */}
              {inlineMode === 'rename' && inlineTarget !== null && canWrite && (
                <InlineNameInput
                  mode="rename"
                  initialValue={inlineTarget.name}
                  onCommit={handleInlineCommit}
                  onCancel={() => { setInlineMode(null); setInlineTarget(null) }}
                />
              )}
            </div>
            <div
              className="file-browser__divider"
              role="separator"
              aria-orientation="vertical"
              aria-label="Resize file list pane"
              data-testid="file-browser-divider"
              onMouseDown={onDividerMouseDown}
            />
            {/* Phase 125-03: show Editor when in edit mode, PreviewPane otherwise */}
            {editingEntry !== null ? (
              <div className="file-browser__preview" data-testid="file-browser-preview">
                <EditorHeader
                  filename={editingEntry.name}
                  filePath={joinPath(path, editingEntry.name)}
                  dirty={editDirty}
                  saveState={saveState}
                  hasError={saveError !== null}
                  isSaving={isSaving}
                  onSave={() => { void handleSave() }}
                  onCancel={() => {
                    // Cancel routes through the guard if dirty
                    guardThen(() => {
                      setEditingEntry(null)
                      setEditDirty(false)
                    })
                  }}
                />
                {/* Inline save error bar — non-takeover; content stays visible (EDIT-06) */}
                {saveError !== null && (
                  <div
                    className="file-browser__editor-error"
                    role="alert"
                    aria-live="assertive"
                  >
                    <span>{saveError}</span>
                    <button
                      type="button"
                      className="file-browser__btn file-browser__btn--icon"
                      aria-label="Dismiss error"
                      onClick={clearSaveError}
                    >
                      ✕
                    </button>
                  </div>
                )}
                <Editor
                  filename={editingEntry.name}
                  initialContent={editContent}
                  fileSize={editingEntry.size}
                  onDirty={setEditDirty}
                  onSave={(content) => {
                    setEditContent(content)
                    void handleSave(content)
                  }}
                  onCancel={() => {
                    guardThen(() => {
                      setEditingEntry(null)
                      setEditDirty(false)
                    })
                  }}
                />
              </div>
            ) : (
              <PreviewPane
                state={preview}
                filename={selected}
                downloadUrl={downloadUrlForSelected}
                canWrite={canWrite}
                isBinary={entries.find((e) => e.name === selected)?.isBinary ?? false}
                onEdit={handleEdit}
              />
            )}
          </div>
          <StatusLine
            itemCount={visibleCount}
            totalCount={entries.length}
            filterActive={filterActive}
            filterValue={filterValue}
            onFilterChange={setFilterValue}
            onFilterDismiss={onFilterDismiss}
          />

          {/* Phase 125-03: Modals rendered at FileBrowserTab root (portal-free, QuitConfirmModal pattern) */}

          {/* Unsaved-changes navigation guard (EDIT-07) — NO beforeunload */}
          <UnsavedChangesModal
            isOpen={unsavedModalOpen}
            onSave={() => {
              setUnsavedModalOpen(false)
              // Save, then proceed with the deferred action on success.
              void handleSave().then(() => {
                if (!isConflict) {
                  const action = pendingActionRef.current
                  pendingActionRef.current = null
                  if (action) {
                    setEditingEntry(null)
                    setEditDirty(false)
                    action()
                  }
                }
              })
            }}
            onDiscard={() => {
              setUnsavedModalOpen(false)
              setEditingEntry(null)
              setEditDirty(false)
              const action = pendingActionRef.current
              pendingActionRef.current = null
              if (action) action()
            }}
            onCancel={() => {
              setUnsavedModalOpen(false)
              pendingActionRef.current = null
            }}
          />

          {/* 412 Conflict modal (EDIT-08) — isConflict from useFilesWrite */}
          <ConflictModal
            isOpen={isConflict}
            onForceOverwrite={() => {
              clearConflict()
              // Force overwrite: re-PUT with If-Match="*" (server skip-check)
              void writeFile(joinPath(path, editingEntry?.name ?? ''), editContent, '*')
            }}
            onSaveAsNew={() => {
              clearConflict()
              // Save as new file: derive {basename}-copy{ext} path
              if (editingEntry) {
                const ext = editingEntry.name.includes('.')
                  ? '.' + editingEntry.name.split('.').pop()!
                  : ''
                const base = ext
                  ? editingEntry.name.slice(0, editingEntry.name.lastIndexOf('.'))
                  : editingEntry.name
                const newName = `${base}-copy${ext}`
                const newPath = joinPath(path, newName)
                void writeFile(newPath, editContent)
              }
            }}
            onDiscard={() => {
              clearConflict()
              // Discard: re-fetch server content and replace the buffer.
              if (selected !== null) {
                const filePath = joinPath(path, selected)
                void client.readFileText(sessionId, filePath).then((body) => {
                  setEditContent(body.text)
                  setEditEtag(body.etag)
                  setEditDirty(false)
                })
              }
            }}
            onCancel={() => {
              clearConflict()
            }}
          />

          {/* Phase 125-04: Plan 04 modals (EDIT-09) */}

          {/* Delete confirm (file + recursive-dir with count) */}
          <DeleteConfirmModal
            isOpen={deleteTarget !== null}
            name={deleteTarget?.name ?? ''}
            isDir={deleteTarget?.isDir ?? false}
            fileCount={deleteFileCount}
            onConfirm={handleDeleteConfirm}
            onCancel={() => setDeleteTarget(null)}
          />

          {/* 409 Collision replace modal — Cancel DEFAULT focus */}
          <CollisionConfirmModal
            isOpen={collisionModalOpen}
            name={collisionName}
            onReplace={async () => {
              setCollisionModalOpen(false)
              const retry = collisionRetryRef.current
              collisionRetryRef.current = null
              if (retry) {
                try {
                  await retry()
                } catch {
                  setWriteOpError("Couldn't complete that. Try again.")
                }
              }
            }}
            onCancel={() => {
              setCollisionModalOpen(false)
              collisionRetryRef.current = null
            }}
          />

          {/* Move to… picker modal */}
          <MoveToPickerModal
            isOpen={moveTarget !== null}
            sessionId={sessionId}
            entry={moveTarget ?? { name: '', size: 0, mtime: '', mode: 0, isDir: false, isSymlink: false, isBinary: false }}
            currentDir={path}
            client={client}
            onMove={(destDir) => { void handleMoveConfirm(destDir) }}
            onCancel={() => setMoveTarget(null)}
          />

          {/* Generic write-operation error banner */}
          {writeOpError !== null && (
            <div
              className="file-browser__error"
              role="alert"
              aria-live="assertive"
              style={{ position: 'absolute', bottom: 40, left: 0, right: 0, zIndex: 10 }}
            >
              <span>{writeOpError}</span>
              <button
                type="button"
                className="file-browser__btn file-browser__btn--icon"
                aria-label="Dismiss error"
                onClick={() => setWriteOpError(null)}
                style={{ marginLeft: 8 }}
              >
                ✕
              </button>
            </div>
          )}
        </>
      )}
    </section>
  )
}
