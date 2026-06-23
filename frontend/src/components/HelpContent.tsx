// Phase 147-02: HelpContent — Markdown renderer with external-link buttons.
//
// Renders bundled Markdown content safely using react-markdown + remark-gfm +
// rehype-sanitize. All anchor links in the Markdown source are intercepted
// and rendered as BrowserOpenURL buttons so they open in the system browser
// instead of navigating inside the Wails webview. No dangerouslySetInnerHTML.

import React from 'react'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize'
import type { Options as Schema } from 'rehype-sanitize'
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'
import { ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline'

// Extended sanitize schema: allow <mark> + className for future highlight
// injection (defensive — HelpContent itself does not inject <mark>, but
// rehype-sanitize strips it by default which would break any future plugin).
const sanitizeSchema: Schema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), 'mark'],
  attributes: {
    ...defaultSchema.attributes,
    mark: ['className'],
  },
}

export function HelpContent({ markdown }: { markdown: string }): React.ReactElement {
  return (
    <Markdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[[rehypeSanitize, sanitizeSchema]]}
      components={{
        // Inline code / keyboard shortcuts rendered with monospace style.
        code: ({ children, ...props }) => (
          <code className="help__kbd" {...props}>
            {children}
          </code>
        ),
        // All Markdown link elements become BrowserOpenURL buttons so they open
        // in the system browser and never navigate the Wails webview.
        a: ({ href, children }) => (
          <button
            type="button"
            className="help-content__external-link"
            onClick={() => href && BrowserOpenURL(href)}
            aria-label={`${children} (opens in browser)`}
          >
            {children}
            <ArrowTopRightOnSquareIcon
              style={{ width: 14, height: 14 }}
              aria-hidden="true"
            />
          </button>
        ),
      }}
    >
      {markdown}
    </Markdown>
  )
}
