// Phase 125-03 Task 2 — editor header component.
//
// Reuses .file-browser__preview-header structure from PreviewPane.tsx (PATTERNS.md
// §EditorHeader/SaveIndicator — "exact analog: PreviewPane.tsx header lines 94-117").
//
// Layout (left to right):
//   [DirtyMarker ●] filename (.file-browser__preview-name)
//   path subline (caption, cwd-relative)
//   SaveIndicator (role=status aria-live)
//   Save button (--primary)
//   Cancel button (--secondary)
//
// Dirty marker ● (U+25CF) carries meaning as a glyph, NOT by color alone
// (RELEASE-BLOCKING colorblind contract — verify at source level).
// aria-label="Modified" provides the accessible label for the bullet.

import React from 'react'
import { SaveIndicator } from './SaveIndicator'
import type { SaveState } from '../../lib/useFilesWrite'

export interface EditorHeaderProps {
  /** Filename of the open file (basename only). */
  filename: string
  /** cwd-relative path for the path subline caption. */
  filePath: string
  /** True when the editor buffer differs from the saved snapshot. */
  dirty: boolean
  /** Current three-state save indicator state. */
  saveState: SaveState
  /** True when a non-conflict save error is present. */
  hasError?: boolean
  /** Called when the user clicks the Save button (mirrors Cmd-S). */
  onSave: () => void
  /** Called when the user clicks the Cancel button — returns to read-only preview. */
  onCancel: () => void
  /** True while a save is in progress (disables Save button). */
  isSaving?: boolean
}

export function EditorHeader({
  filename,
  filePath,
  dirty,
  saveState,
  hasError = false,
  onSave,
  onCancel,
  isSaving = false,
}: EditorHeaderProps): React.ReactElement {
  return (
    <header className="file-browser__preview-header">
      {/* Filename + dirty marker */}
      <div className="file-browser__preview-name-group">
        {dirty && (
          <span
            className="file-browser__dirty-marker"
            aria-label="Modified"
            title="Unsaved changes"
          >
            {/* ● (U+25CF) bullet glyph — carries meaning; color (#7aa2f7) is decoration */}
            ●
          </span>
        )}
        <span
          className="file-browser__preview-name"
          data-testid="editor-header-filename"
        >
          {filename}
        </span>
        {/* Path subline — caption muted text (11–12px per UI-SPEC §Typography) */}
        {filePath && filePath !== filename && (
          <span className="file-browser__preview-path" title={filePath}>
            {filePath}
          </span>
        )}
      </div>

      {/* Right cluster: SaveIndicator + Save + Cancel */}
      <div className="file-browser__preview-header-actions">
        <SaveIndicator saveState={saveState} hasError={hasError} />

        <button
          type="button"
          className="file-browser__btn file-browser__btn--primary"
          onClick={onSave}
          disabled={isSaving}
          aria-label={`Save ${filename}`}
        >
          Save
        </button>

        <button
          type="button"
          className="file-browser__btn file-browser__btn--secondary"
          onClick={onCancel}
          disabled={isSaving}
          aria-label="Cancel editing"
        >
          Cancel
        </button>
      </div>
    </header>
  )
}
