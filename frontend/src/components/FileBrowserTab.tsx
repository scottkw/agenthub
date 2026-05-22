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

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { FilesApiClient, FilesApiError, type FileEntry } from '../lib/filesApi'
import { useFilesCapability } from '../lib/useFilesCapability'
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

  const { state: capState, retry: retryCapability } = useFilesCapability(client, sessionId)

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
  useEffect(() => {
    setSelected(null)
    setPreview({ kind: 'idle' })
    setFilterActive(false)
    setFilterValue('')
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

  // ─── Navigation ───
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
      setPath((p) => joinPath(p, name))
    },
    [],
  )

  const navigateUp = useCallback(() => {
    setPath((p) => (p === '.' || p === '' ? p : dirname(p)))
  }, [])

  const navigateTo = useCallback(
    (segmentPath: string) => {
      // Defense-in-depth (UI-05): only accept '.' or a strict prefix of the
      // current path. The server sandbox is the actual security boundary.
      setPath((p) => (isPrefixOrEqual(segmentPath, p) ? segmentPath : p))
    },
    [],
  )

  const refresh = useCallback(() => {
    setDirNonce((n) => n + 1)
  }, [])

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
          />
          <div className="file-browser__body" ref={bodyRef}>
            <div
              className="file-browser__list-container"
              style={{ width: `${listWidthPct}%` }}
            >
              {sortedEntries.length === 0 ? (
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
                  onSelect={setSelected}
                  onNavigateInto={navigateInto}
                  onNavigateUp={navigateUp}
                  onSortChange={onSortChange}
                  onFilterActivate={onFilterActivate}
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
            <PreviewPane
              state={preview}
              filename={selected}
              downloadUrl={downloadUrlForSelected}
            />
          </div>
          <StatusLine
            itemCount={visibleCount}
            totalCount={entries.length}
            filterActive={filterActive}
            filterValue={filterValue}
            onFilterChange={setFilterValue}
            onFilterDismiss={onFilterDismiss}
          />
        </>
      )}
    </section>
  )
}
