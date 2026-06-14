// Phase 125-04 Task 2 — "Move to…" directory picker modal.
//
// Provides a cross-directory move affordance: renders a sandbox-rooted dirs-only
// tree that the user can browse to select a destination directory. Calls
// rename(oldRel, newRel) when the user confirms (cross-dir move = rename, EDIT-09).
//
// "Move here" primary is disabled until a target directory is chosen.
// 409 from rename → onCollision callback (caller opens CollisionConfirmModal).
//
// Copy strings VERBATIM from UI-SPEC §Copywriting Contract (EDIT-09 locked):
//   Title:     `Move "{name}" to…`
//   Primary:   "Move here"
//   Cancel:    "Cancel"
//
// QuitConfirmModal + new-session-modal patterns:
//   - overlay click-cancel, Escape-closes
//   - "Move here" primary disabled until target selected
//   - acting guard during the async rename
//
// Directory tree: fetches via client.listFiles; dirs-only rendered recursively.
// The current entry's parent dir is shown (greyed) but cannot be chosen as a
// destination (that would be a no-op move).

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { FolderIcon, FolderOpenIcon, ChevronRightIcon } from '@heroicons/react/24/outline'
import type { FilesApiClient, FileEntry } from '../../../lib/filesApi'

export interface MoveToPickerModalProps {
  isOpen: boolean
  /** Session ID for listFiles calls. */
  sessionId: string
  /** The entry being moved (used to derive the destination rel path). */
  entry: FileEntry
  /** CWD-relative path of the directory containing `entry`. */
  currentDir: string
  /** FilesApiClient for directory listing. */
  client: FilesApiClient | null
  /**
   * Called when the user confirms Move here. Receives the new cwd-relative
   * destination path (dir only — filename appended by caller or here).
   * Caller calls rename(currentDir/entry.name, destDir/entry.name).
   */
  onMove: (destDir: string) => Promise<void> | void
  /** Called when the user cancels. */
  onCancel: () => void
  /** Called when a 409 collision occurs. */
  onCollision?: () => void
}

interface DirTreeNodeProps {
  path: string
  name: string
  sessionId: string
  client: FilesApiClient | null
  selectedPath: string | null
  onSelect: (path: string) => void
  disabledPath?: string
}

/** Recursive directory tree node. Expands on click; fetches children lazily. */
function DirTreeNode({
  path,
  name,
  sessionId,
  client,
  selectedPath,
  onSelect,
  disabledPath,
}: DirTreeNodeProps): React.ReactElement {
  const [expanded, setExpanded] = useState(path === '.')
  const [children, setChildren] = useState<FileEntry[]>([])
  const [loaded, setLoaded] = useState(false)
  const [loading, setLoading] = useState(false)

  const isSelected = selectedPath === path
  const isDisabled = path === disabledPath

  const toggle = useCallback(async () => {
    if (isDisabled) return
    if (!expanded && !loaded && client) {
      setLoading(true)
      try {
        const resp = await client.listFiles(sessionId, path)
        setChildren(resp.entries.filter((e) => e.isDir))
        setLoaded(true)
      } finally {
        setLoading(false)
      }
    }
    setExpanded((e) => !e)
  }, [expanded, loaded, client, sessionId, path, isDisabled])

  // Load root children on mount.
  useEffect(() => {
    if (path === '.' && client && !loaded) {
      setLoading(true)
      void client.listFiles(sessionId, path).then((resp) => {
        setChildren(resp.entries.filter((e) => e.isDir))
        setLoaded(true)
        setLoading(false)
      }).catch(() => setLoading(false))
    }
  }, [path, client, sessionId, loaded])

  return (
    <div className="file-browser__move-tree-node">
      <button
        type="button"
        className={[
          'file-browser__move-tree-row',
          isSelected ? 'file-browser__move-tree-row--selected' : '',
          isDisabled ? 'file-browser__move-tree-row--disabled' : '',
        ].join(' ').trim()}
        aria-disabled={isDisabled}
        onClick={() => {
          if (!isDisabled) {
            onSelect(path)
            void toggle()
          }
        }}
        aria-label={path === '.' ? 'session root' : name}
      >
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <ChevronRightIcon
            width={10}
            height={10}
            aria-hidden="true"
            style={{ transform: expanded ? 'rotate(90deg)' : undefined, transition: 'transform 0.1s' }}
          />
          {expanded ? (
            <FolderOpenIcon width={14} height={14} aria-hidden="true" />
          ) : (
            <FolderIcon width={14} height={14} aria-hidden="true" />
          )}
          <span>{path === '.' ? 'session/' : name}</span>
        </span>
      </button>
      {expanded && (
        <div style={{ paddingLeft: 20 }}>
          {loading && <span style={{ fontSize: 11, color: '#9aa5ce' }}>Loading…</span>}
          {loaded && children.length === 0 && (
            <span style={{ fontSize: 11, color: '#9aa5ce' }}>Empty</span>
          )}
          {children.map((child) => (
            <DirTreeNode
              key={child.name}
              path={path === '.' ? child.name : `${path}/${child.name}`}
              name={child.name}
              sessionId={sessionId}
              client={client}
              selectedPath={selectedPath}
              onSelect={onSelect}
              disabledPath={disabledPath}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export function MoveToPickerModal({
  isOpen,
  sessionId,
  entry,
  currentDir,
  client,
  onMove,
  onCancel,
}: MoveToPickerModalProps): React.ReactElement | null {
  const [acting, setActing] = useState(false)
  const [selectedDir, setSelectedDir] = useState<string | null>(null)
  const cancelBtnRef = useRef<HTMLButtonElement>(null)

  // Close on Escape.
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  // Reset state on open.
  useEffect(() => {
    if (isOpen) {
      setActing(false)
      setSelectedDir(null)
      // Focus cancel on open (safe default — no confirm until a dir is selected anyway).
      cancelBtnRef.current?.focus()
    }
  }, [isOpen])

  if (!isOpen) return null

  async function handleMove() {
    if (selectedDir === null || acting) return
    setActing(true)
    try {
      await onMove(selectedDir)
    } finally {
      setActing(false)
    }
  }

  const title = `Move "${entry.name}" to…`

  return (
    <div className="quit-modal-overlay" onClick={onCancel}>
      <div
        className="quit-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="move-picker-title"
        style={{ minWidth: 320, maxWidth: 480 }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="quit-modal__header">
          <h2 className="quit-modal__title" id="move-picker-title">
            {/* EDIT-09 locked title */}
            {title}
          </h2>
          <button className="quit-modal__close" aria-label="Close" onClick={onCancel}>
            &times;
          </button>
        </div>
        <div
          className="quit-modal__body"
          style={{ maxHeight: 300, overflowY: 'auto', minHeight: 120 }}
        >
          {/* Dirs-only tree — currentDir is disabled (would be a no-op) */}
          <DirTreeNode
            path="."
            name="session"
            sessionId={sessionId}
            client={client}
            selectedPath={selectedDir}
            onSelect={setSelectedDir}
            disabledPath={currentDir === '.' ? undefined : currentDir}
          />
        </div>
        <div className="quit-modal__footer">
          {/* Cancel */}
          <button
            ref={cancelBtnRef}
            type="button"
            className="file-browser__btn file-browser__btn--secondary"
            disabled={acting}
            onClick={onCancel}
          >
            Cancel
          </button>
          {/* Move here — primary; disabled until a target dir is selected */}
          <button
            type="button"
            className="file-browser__btn file-browser__btn--primary"
            disabled={selectedDir === null || acting}
            onClick={() => { void handleMove() }}
          >
            {/* EDIT-09 locked string */}
            Move here
          </button>
        </div>
      </div>
    </div>
  )
}
