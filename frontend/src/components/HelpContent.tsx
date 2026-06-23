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

// Allow-listed URL schemes for external links. rehype-sanitize already strips
// dangerous hrefs (javascript:, data:, etc.) before this renderer sees them, but
// we validate again as defense-in-depth before handing anything to BrowserOpenURL.
const SAFE_LINK_SCHEME = /^(https?:|mailto:)/i

function isSafeExternalHref(href: string | undefined): href is string {
  return typeof href === 'string' && SAFE_LINK_SCHEME.test(href)
}

/**
 * Derive a plain-text accessible label from a React node. react-markdown passes
 * `children` as a React node (string, array, or element). Template-stringifying a
 * non-string node yields "[object Object]", so we recursively extract text; if no
 * text is recoverable we fall back to the href.
 */
function nodeToText(node: React.ReactNode): string {
  if (typeof node === 'string') return node
  if (typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(nodeToText).join('')
  if (React.isValidElement(node)) {
    return nodeToText((node.props as { children?: React.ReactNode }).children)
  }
  return ''
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
        a: ({ href, children }) => {
          const labelText = nodeToText(children).trim() || href || 'link'
          return (
            <button
              type="button"
              className="help-content__external-link"
              onClick={() => {
                if (isSafeExternalHref(href)) BrowserOpenURL(href)
              }}
              aria-label={`${labelText} (opens in browser)`}
            >
              {children}
              <ArrowTopRightOnSquareIcon
                style={{ width: 14, height: 14 }}
                aria-hidden="true"
              />
            </button>
          )
        },
      }}
    >
      {markdown}
    </Markdown>
  )
}
