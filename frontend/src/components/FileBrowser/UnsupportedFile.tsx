// Phase 120-04 Task 1 — unsupported-file preview leaf.
//
// Renders two distinct refusal states sharing the same DOM shape:
//   - kind='unsupported' — binary file (e.g. .zip, .so, .exe)
//   - kind='over-cap'    — file exceeds the 5 MB preview cap (FS-08 → 413)
//
// Both states surface a Download <a> linking to the same /api/files/read
// endpoint with the cap token — the daemon serves the bytes directly via
// HTTP Content-Disposition: attachment. Because the link is a plain <a download>
// the browser handles streaming on its own; the JS heap never holds the bytes.
//
// Copy is locked verbatim to UI-SPEC §"Error copy"; do not paraphrase.

import React from 'react'
import { DownloadButton } from './DownloadButton'

export interface UnsupportedFileProps {
  kind: 'unsupported' | 'over-cap'
  filename: string
  /** Human-formatted size string (e.g. "12.4 MB"). */
  humanSize: string
  /** Download URL (built via FilesApiClient.buildDownloadUrl). */
  downloadUrl: string
}

export function UnsupportedFile({
  kind,
  filename,
  humanSize,
  downloadUrl,
}: UnsupportedFileProps): React.ReactElement {
  const isOverCap = kind === 'over-cap'
  const heading = isOverCap
    ? 'File too large to preview'
    : "Sorry, we can't display this file."
  const body = isOverCap
    ? `Previews are limited to 5 MB. This file is ${humanSize}.`
    : 'This file is not text. Download it to view in another app.'
  const testId = isOverCap ? 'file-browser-over-cap' : 'file-browser-binary'

  return (
    <div
      className="file-browser__preview--unsupported"
      data-testid={testId}
      role="region"
      aria-label={heading}
    >
      <h3 className="file-browser__preview-heading">{heading}</h3>
      <p className="file-browser__preview-body">{body}</p>
      <DownloadButton
        url={downloadUrl}
        filename={filename}
        className="file-browser__btn file-browser__btn--primary"
        ariaLabel={`Download ${filename}`}
        title="Download"
      >
        Download
      </DownloadButton>
    </div>
  )
}
