// Phase 120-04 Task 1 — image preview leaf.
//
// Direct <img src={url}> rendering. The URL points at
// /api/files/read?session=…&path=…&cap=… (built by FilesApiClient.buildImageUrl).
// The browser fetches the bytes natively — they never enter JS memory, never
// pass through `FileReader`, never round-trip through `btoa()`, never become
// a `data:image/…` URI, never become a `blob:` URL.
//
// This is the CSP-safe contract from UI-SPEC §CSP Contract: img-src 'self'
// covers the same-origin /api/files/read endpoint without needing to allow
// `data:` for images larger than the existing 1x1 transparent pixel.
//
// The cap token leakage via network panel is documented (T-120-14) and
// accepted because every other AgentHub session URL has the same shape.
//
// The source-inspection test in __tests__/FileBrowserTab.no-base64.test.tsx
// reads this file's bytes and asserts the absence of `btoa(`, `data:image/`,
// `FileReader`, `.toDataURL`, and `URL.createObjectURL` so any future
// regression that re-introduces an in-JS encoding pipeline fails CI.

import React from 'react'

export interface ImagePreviewProps {
  /** Direct GET URL for the image bytes (built via FilesApiClient.buildImageUrl). */
  src: string
  /** Filename — used as alt text for accessibility. */
  filename: string
}

export function ImagePreview({ src, filename }: ImagePreviewProps): React.ReactElement {
  return (
    <div
      className="file-browser__preview--image"
      data-testid="file-browser-preview-image"
    >
      <img
        src={src}
        alt={filename}
        style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain' }}
      />
    </div>
  )
}
