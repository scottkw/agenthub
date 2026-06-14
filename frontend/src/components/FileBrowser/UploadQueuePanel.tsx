/**
 * UploadQueuePanel — Phase 125-05 Task 2 (EDIT-10).
 *
 * Shows a per-file upload queue with determinate N% progress, done/failed states,
 * over-cap skip messages, and inline Replace? prompts for 409 collisions.
 *
 * Colorblind contract (RELEASE-BLOCKING):
 *   - In progress: N% text (no color-only signal)
 *   - Done: CheckCircleIcon + "Done" text
 *   - Failed: ExclamationTriangleIcon + "Failed — try again" text
 *   - Over-cap: ExclamationTriangleIcon + verbatim skip copy
 *   - Collision: ArrowPathIcon + inline Replace? prompt
 * Color is reinforcement only — never the sole signal.
 *
 * Chrome reuses .new-session-modal* (UI-SPEC §Design System).
 * Progress live region: role="status" aria-live="polite" (UI-SPEC colorblind).
 */

import React from 'react'
import {
  CheckCircleIcon,
  ExclamationTriangleIcon,
  ArrowUpTrayIcon,
} from '@heroicons/react/24/outline'

export type UploadItemStatus = 'uploading' | 'done' | 'failed' | 'over-cap' | 'collision'

export interface UploadQueueItem {
  /** Stable per-item id used to update progress by identity (WR-03). */
  id: string
  file: File
  status: UploadItemStatus
  /** Integer 0–100. */
  progress: number
  /** Optional error detail (used for 'failed' status). */
  error?: string
}

export interface UploadQueuePanelProps {
  items: UploadQueueItem[]
  /** Called with the item index when the user chooses Replace on a 409 row. */
  onReplace: (index: number) => void
  /** Called when the user closes/dismisses the queue panel. */
  onDismiss: () => void
}

/** Queue title copy per UI-SPEC §Copywriting: single vs. multi. */
function queueTitle(items: UploadQueueItem[]): string {
  if (items.length === 1) return `Uploading ${items[0].file.name}`
  return `Uploading ${items.length} files`
}

export function UploadQueuePanel({
  items,
  onReplace,
  onDismiss,
}: UploadQueuePanelProps): React.ReactElement {
  return (
    <div
      className="new-session-modal__overlay"
      style={{ position: 'fixed', bottom: 48, right: 16, top: 'auto', left: 'auto', display: 'flex', alignItems: 'flex-end', justifyContent: 'flex-end', background: 'none', zIndex: 200 }}
    >
      <div
        className="new-session-modal"
        role="region"
        aria-label="Upload queue"
        style={{ width: 340, maxHeight: 400, overflowY: 'auto' }}
      >
        {/* Panel header */}
        <div className="new-session-modal__header" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <h3 className="new-session-modal__title" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <ArrowUpTrayIcon width={14} height={14} aria-hidden="true" />
            {queueTitle(items)}
          </h3>
          <button
            type="button"
            className="file-browser__btn file-browser__btn--icon"
            aria-label="Dismiss upload queue"
            onClick={onDismiss}
          >
            ✕
          </button>
        </div>

        {/* Per-file rows */}
        <div
          className="new-session-modal__body"
          role="status"
          aria-live="polite"
          aria-label="Upload progress"
        >
          {items.map((item, idx) => (
            <UploadRow key={idx} item={item} index={idx} onReplace={onReplace} />
          ))}
        </div>
      </div>
    </div>
  )
}

interface UploadRowProps {
  item: UploadQueueItem
  index: number
  onReplace: (index: number) => void
}

function UploadRow({ item, index, onReplace }: UploadRowProps): React.ReactElement {
  const { file, status, progress } = item

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 4,
        padding: '8px 0',
        borderBottom: '1px solid #292e42',
      }}
    >
      {/* File name */}
      <span style={{ fontSize: 13, color: '#c0caf5', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {file.name}
      </span>

      {/* Status row */}
      {status === 'uploading' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {/* Determinate progress bar */}
          <div
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={progress}
            aria-label={`Uploading ${file.name}`}
            style={{
              height: 4,
              background: '#292e42',
              borderRadius: 2,
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                height: '100%',
                width: `${progress}%`,
                background: '#7aa2f7',
                transition: 'width 0.1s linear',
              }}
            />
          </div>
          {/* N% text — non-color progress signal (colorblind contract) */}
          <span style={{ fontSize: 11, color: '#9aa5ce' }}>{progress}%</span>
        </div>
      )}

      {status === 'done' && (
        <span
          style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, color: '#9ece6a' }}
        >
          {/* CheckCircleIcon + "Done" — icon+text carry the meaning, color is decoration */}
          <CheckCircleIcon width={13} height={13} aria-hidden="true" />
          Done
        </span>
      )}

      {status === 'failed' && (
        <span
          style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, color: '#f7768e' }}
        >
          {/* ExclamationTriangleIcon + "Failed — try again" — icon+text, color is decoration */}
          <ExclamationTriangleIcon width={13} height={13} aria-hidden="true" />
          Failed — try again
        </span>
      )}

      {status === 'over-cap' && (
        <span
          style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, color: '#e0af68' }}
        >
          {/* ExclamationTriangleIcon + skip copy — icon+text, color is decoration */}
          <ExclamationTriangleIcon width={13} height={13} aria-hidden="true" />
          &quot;{file.name}&quot; is too large (max 50 MB) and was skipped.
        </span>
      )}

      {status === 'collision' && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, color: '#9aa5ce' }}>
          <span>A file named &quot;{file.name}&quot; already exists.</span>
          {/* Inline Replace? prompt — does not block other rows (EDIT-10) */}
          <button
            type="button"
            className="file-browser__btn file-browser__btn--primary"
            style={{ fontSize: 12, padding: '2px 8px' }}
            aria-label={`Replace ${file.name}`}
            onClick={() => onReplace(index)}
          >
            Replace
          </button>
        </div>
      )}
    </div>
  )
}
