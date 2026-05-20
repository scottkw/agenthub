// Phase 120-04 Task 1 — text preview leaf.
//
// Pure presentational component: receives plain text (already fetched by
// FileBrowserTab via FilesApiClient.readFileText) and renders it in a
// monospace, pre-formatted, scrollable container. CSS (file-browser__preview--text)
// owns the visual contract — font-family inherits the body monospace stack,
// font-size: 12px per UI-SPEC §Typography "Mono preview".
//
// No state. No effects. No DOM mutation outside JSX. Source text is rendered
// inside <code>{source}</code> — React escapes any embedded HTML automatically
// (no dangerouslySetInnerHTML), so a malicious source file cannot inject markup.

import React from 'react'

export interface TextPreviewProps {
  /** UTF-8 decoded file contents (≤5 MB by the time we reach this branch). */
  source: string
}

export function TextPreview({ source }: TextPreviewProps): React.ReactElement {
  return (
    <pre
      className="file-browser__preview--text"
      data-testid="file-browser-preview-text"
    >
      <code>{source}</code>
    </pre>
  )
}
