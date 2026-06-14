// Phase 125-05 Task 2 — UploadQueuePanel stub (RED phase; full impl follows in GREEN).
import React from 'react'

export type UploadItemStatus = 'uploading' | 'done' | 'failed' | 'over-cap' | 'collision'

export interface UploadQueueItem {
  file: File
  status: UploadItemStatus
  progress: number
  error?: string
}

export interface UploadQueuePanelProps {
  items: UploadQueueItem[]
  onReplace: (index: number) => void
  onDismiss: () => void
}

export function UploadQueuePanel(_props: UploadQueuePanelProps): React.ReactElement {
  return <div>stub</div>
}
