// Phase 120-04 Task 1 — empty-directory state leaf.
//
// Rendered INSIDE the list pane (replacing the rows) when /api/files/list
// returns zero entries. This is distinct from PermissionDeniedTakeover which
// replaces the entire tab body. UI-SPEC §"State machine — directory listing"
// row "empty" specifies this exact shape.
//
// Copy is locked verbatim per UI-SPEC §"Empty / status copy".

import React from 'react'

export interface EmptyDirectoryStateProps {
  /** Relative path from session cwd — shown muted under the heading for context. */
  relativePathFromCwd: string
}

export function EmptyDirectoryState({
  relativePathFromCwd,
}: EmptyDirectoryStateProps): React.ReactElement {
  return (
    <div
      className="file-browser__empty"
      data-testid="file-browser-empty"
      role="status"
    >
      <h3 className="file-browser__empty-heading">This directory is empty.</h3>
      <p className="file-browser__empty-path">{relativePathFromCwd}</p>
    </div>
  )
}
