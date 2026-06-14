// Phase 120-04 Task 1 — preview pane dispatcher.
//
// Pure switch over PreviewState.kind that delegates to a leaf component or
// renders a small inline state (idle, empty, loading, broken-symlink,
// forbidden-file). The pane is wrapped in a <section role="region"
// aria-label="File preview" aria-live="polite"> so screen readers announce
// state transitions (UI-SPEC §Accessibility "Preview pane").
//
// The preview header (filename + size + Download button) is rendered ONLY
// for kinds with a backing file (text/markdown/image/unsupported/over-cap/
// empty). Non-file kinds (idle/loading/read-error/forbidden-file/
// broken-symlink) skip the header.
//
// The DOM identity of the inner wrapper (`data-testid="file-browser-preview"`)
// is constant across every kind — Playwright (UI-14) targets it without
// branching on state.

import React from 'react'
import { ArrowDownTrayIcon, PencilSquareIcon } from '@heroicons/react/24/outline'
import type { PreviewState } from '../../lib/filesTypes'
import { humanSize as fmtHumanSize } from '../../lib/humanSize'
import { TextPreview } from './TextPreview'
import { MarkdownPreview } from './MarkdownPreview'
import { ImagePreview } from './ImagePreview'
import { UnsupportedFile } from './UnsupportedFile'
import { NetworkErrorState } from './NetworkErrorState'
import { DownloadButton } from './DownloadButton'

export interface PreviewPaneProps {
  state: PreviewState
  /**
   * Filename of the currently-selected entry, threaded in for the header.
   * `null` when no selection (kind='idle'). Text/markdown/image states do
   * not carry filename in their payloads (PreviewState design) so the
   * orchestrator threads it separately.
   */
  filename: string | null
  /** Download URL for the currently-selected entry, or null when no selection. */
  downloadUrl: string | null
  /**
   * Phase 125-02: true when the current session has write permission on this
   * surface (desktop: SessionInfo.filesWrite; web-share: files.write cap bit).
   * When true and the selected file is text/markdown and not binary, the pencil
   * Edit button is rendered. UI authority only — server is the real gate (EDIT-12).
   */
  canWrite?: boolean
  /**
   * Phase 125-02: true when the selected entry is a binary file (isBinary from
   * FileEntry). Binary files never show the Edit affordance (EDIT-03).
   */
  isBinary?: boolean
  /**
   * Phase 125-02: callback invoked when the user clicks the Edit button.
   * Opens the Editor component for the currently-selected file.
   */
  onEdit?: () => void
}

/**
 * True iff the given state has a "backing file" — i.e. a real entry the user
 * picked, whose size and filename are meaningful. Drives whether the preview
 * header renders.
 */
function hasHeaderForKind(kind: PreviewState['kind']): boolean {
  switch (kind) {
    case 'text':
    case 'markdown':
    case 'image':
    case 'empty':
    case 'unsupported':
    case 'over-cap':
      return true
    default:
      return false
  }
}

/**
 * Extract a byte-size for the header display from the discriminated state
 * when present. Returns null when the kind has no size payload.
 */
function sizeOf(state: PreviewState): number | null {
  switch (state.kind) {
    case 'text':
    case 'markdown':
    case 'image':
      return state.size
    case 'empty':
      return 0
    default:
      return null
  }
}

export function PreviewPane({
  state,
  filename,
  downloadUrl,
  canWrite = false,
  isBinary = false,
  onEdit,
}: PreviewPaneProps): React.ReactElement {
  const showHeader = hasHeaderForKind(state.kind) && filename !== null
  const size = sizeOf(state)

  // Edit button is shown when canWrite AND text/markdown AND NOT binary (EDIT-03/12).
  // Absence is the colorblind-safe signal for binary/no-permission (no greyed ghost).
  const showEditButton =
    canWrite &&
    !isBinary &&
    (state.kind === 'text' || state.kind === 'markdown') &&
    onEdit !== undefined

  return (
    <section
      role="region"
      aria-label="File preview"
      aria-live="polite"
      className="file-browser__preview"
      data-testid="file-browser-preview"
    >
      {showHeader && (
        <header className="file-browser__preview-header">
          <span
            className="file-browser__preview-name"
            data-testid="file-browser-preview-name"
          >
            {filename}
          </span>
          {size !== null && (
            <span className="file-browser__preview-size">{fmtHumanSize(size)}</span>
          )}
          {downloadUrl !== null && (
            <DownloadButton
              url={downloadUrl}
              filename={filename ?? ''}
              className="file-browser__btn file-browser__btn--icon"
              ariaLabel={`Download ${filename ?? ''}`}
              title="Download"
            >
              <ArrowDownTrayIcon width={14} height={14} aria-hidden="true" />
            </DownloadButton>
          )}
          {/* Phase 125-02 Edit button — gated on canWrite && !isBinary && text/markdown.
              Absence is the signal for binary/no-permission (colorblind contract). */}
          {showEditButton && (
            <button
              className="file-browser__btn file-browser__btn--icon"
              aria-label={`Edit ${filename ?? ''}`}
              title="Edit"
              onClick={onEdit}
            >
              <PencilSquareIcon width={14} height={14} aria-hidden="true" />
            </button>
          )}
        </header>
      )}
      <div className="file-browser__preview-body">{renderBody(state, filename)}</div>
    </section>
  )
}

function renderBody(state: PreviewState, filename: string | null): React.ReactElement {
  switch (state.kind) {
    case 'idle':
      return (
        <div className="file-browser__preview--idle">Select a file to preview</div>
      )
    case 'loading':
      return (
        <div
          className="file-browser__preview--loading"
          data-testid="file-browser-preview-loading"
        >
          <div className="file-browser__spinner" aria-hidden="true" />
          <span>Loading preview…</span>
        </div>
      )
    case 'text':
      return <TextPreview source={state.text} />
    case 'markdown':
      return <MarkdownPreview source={state.text} />
    case 'image':
      // Phase 120 WR-07 — pass the real filename through so ImagePreview can
      // emit alt={filename} instead of alt="". An empty alt tells screen
      // readers to skip the image entirely, which is wrong for previewed
      // user content (a blind user has no way to identify the currently
      // previewed file when focused on the image element). The header
      // already shows the name visually, but the alt is the only audible
      // signal for assistive tech. Fall back to 'image' if filename is null.
      return <ImagePreview src={state.url} filename={filename ?? 'image'} />
    case 'empty':
      return (
        <div className="file-browser__preview--empty">
          <h3 className="file-browser__preview-heading">Empty file</h3>
          <p className="file-browser__preview-body-muted">0 bytes</p>
        </div>
      )
    case 'unsupported':
      return (
        <UnsupportedFile
          kind="unsupported"
          filename={state.filename}
          humanSize={state.humanSize}
          downloadUrl={state.downloadUrl}
        />
      )
    case 'over-cap':
      return (
        <UnsupportedFile
          kind="over-cap"
          filename={state.filename}
          humanSize={state.humanSize}
          downloadUrl={state.downloadUrl}
        />
      )
    case 'broken-symlink':
      return (
        <div
          className="file-browser__preview--broken-symlink"
          data-testid="file-browser-broken-symlink"
          role="alert"
        >
          <h3 className="file-browser__preview-heading">Broken symlink</h3>
          <p className="file-browser__preview-body">
            Target {state.targetPath} does not exist.
          </p>
        </div>
      )
    case 'read-error':
      return <NetworkErrorState scope="preview" onRetry={state.onRetry} />
    case 'forbidden-file':
      return (
        <div
          className="file-browser__preview--forbidden"
          data-testid="file-browser-forbidden"
          role="alert"
        >
          <h3 className="file-browser__preview-heading">Cannot read this file</h3>
          <p className="file-browser__preview-body">
            The session does not have permission to read it.
          </p>
        </div>
      )
  }
}
