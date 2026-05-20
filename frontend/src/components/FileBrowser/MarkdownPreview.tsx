// Phase 120-04 Task 1 — markdown preview leaf.
//
// Renders Markdown using react-markdown@10.1.0 + remark-gfm@^4 (GFM tables,
// task lists, strikethrough, autolinks). The CSP / XSS contract is enforced
// here by what we DO NOT import:
//
//   - NO `rehype-raw`        — raw HTML in source renders as escaped text
//   - NO `rehypePlugins`     — no plugin pipeline that could re-enable raw HTML
//   - NO `dangerouslySetInnerHTML` — React JSX-escapes everything we pass
//
// The source-inspection test in __tests__/FileBrowserTab.no-rehype-raw.test.tsx
// reads this file's bytes and asserts the absence of those substrings, so any
// future regression that adds them fails CI before it can ship.

import React from 'react'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

export interface MarkdownPreviewProps {
  /** Markdown source text (already fetched, ≤5 MB). */
  source: string
}

export function MarkdownPreview({ source }: MarkdownPreviewProps): React.ReactElement {
  return (
    <div
      className="file-browser__preview--markdown"
      data-testid="file-browser-preview-markdown"
    >
      <Markdown remarkPlugins={[remarkGfm]}>{source}</Markdown>
    </div>
  )
}
