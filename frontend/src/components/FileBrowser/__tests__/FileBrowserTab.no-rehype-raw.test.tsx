/**
 * Phase 120-04 Task 1 — source-inspection guard (Pitfall 9).
 *
 * Reads the bytes of MarkdownPreview.tsx and FileBrowserTab.tsx and asserts
 * the absence of substrings that would re-enable raw HTML rendering in the
 * markdown preview path. A regression that adds `rehype-raw`, a `rehypePlugins`
 * prop, or `dangerouslySetInnerHTML` will fail this test before it can ship.
 *
 * The test runs in node (vitest's default jsdom env still exposes node:fs)
 * so we can read the raw file contents without a build step.
 */
import { describe, it, expect } from 'vitest'
import { readFileSync, existsSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const MD_PREVIEW = resolve(__dirname, '..', 'MarkdownPreview.tsx')
const FB_TAB = resolve(__dirname, '..', '..', 'FileBrowserTab.tsx')

function readIfExists(path: string): string {
  if (!existsSync(path)) return ''
  return readFileSync(path, 'utf8')
}

describe('no-rehype-raw source guard', () => {
  it('MarkdownPreview.tsx contains no rehype-raw reference', () => {
    const src = readFileSync(MD_PREVIEW, 'utf8')
    // The doc-comment may mention "rehype-raw" textually as a thing we do NOT
    // import. Tolerate that by allowing the substring only on lines that
    // start with `//` (single-line comments) or that contain "NO `rehype-raw`".
    const offending = src
      .split('\n')
      .filter((line) => {
        if (!line.includes('rehype-raw')) return false
        const trimmed = line.trim()
        // Allow doc-comment mentions; flag anything else.
        if (trimmed.startsWith('//')) return false
        if (trimmed.startsWith('*')) return false
        // Flag actual imports / option references.
        return true
      })
    expect(offending).toEqual([])
  })

  it('MarkdownPreview.tsx contains no rehypePlugins prop', () => {
    const src = readFileSync(MD_PREVIEW, 'utf8')
    const offending = src
      .split('\n')
      .filter((line) => {
        if (!line.includes('rehypePlugins')) return false
        const trimmed = line.trim()
        if (trimmed.startsWith('//')) return false
        if (trimmed.startsWith('*')) return false
        return true
      })
    expect(offending).toEqual([])
  })

  it('MarkdownPreview.tsx contains no dangerouslySetInnerHTML', () => {
    const src = readFileSync(MD_PREVIEW, 'utf8')
    const offending = src
      .split('\n')
      .filter((line) => {
        if (!line.includes('dangerouslySetInnerHTML')) return false
        const trimmed = line.trim()
        if (trimmed.startsWith('//')) return false
        if (trimmed.startsWith('*')) return false
        return true
      })
    expect(offending).toEqual([])
  })

  it('FileBrowserTab.tsx (if exists) contains no rehype-raw / rehypePlugins / dangerouslySetInnerHTML', () => {
    const src = readIfExists(FB_TAB)
    if (src === '') {
      // Allowed — Task 2 has not yet authored the orchestrator. Test is a no-op
      // until then. Once FileBrowserTab.tsx exists, this branch never fires.
      return
    }
    for (const banned of ['rehype-raw', 'rehypePlugins', 'dangerouslySetInnerHTML']) {
      const offending = src
        .split('\n')
        .filter((line) => {
          if (!line.includes(banned)) return false
          const trimmed = line.trim()
          if (trimmed.startsWith('//')) return false
          if (trimmed.startsWith('*')) return false
          return true
        })
      expect(offending, `${banned} found in FileBrowserTab.tsx`).toEqual([])
    }
  })

  it('MarkdownPreview.tsx imports react-markdown and remark-gfm', () => {
    const src = readFileSync(MD_PREVIEW, 'utf8')
    expect(src).toMatch(/from ['"]react-markdown['"]/)
    expect(src).toMatch(/from ['"]remark-gfm['"]/)
  })
})
