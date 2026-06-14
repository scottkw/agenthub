// Phase 125-05 Task 2 — UploadDropOverlay stub (RED phase; full impl follows in GREEN).
import React from 'react'

export interface UploadDropOverlayProps {
  onDrop: (files: File[]) => void
  onDragLeave: () => void
}

export function UploadDropOverlay(_props: UploadDropOverlayProps): React.ReactElement {
  return <div data-testid="upload-drop-overlay">stub</div>
}
