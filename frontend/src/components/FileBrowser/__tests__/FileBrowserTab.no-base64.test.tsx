/**
 * Phase 120-04 Task 1 — source-inspection guard (Pitfall 10).
 *
 * Reads the bytes of ImagePreview.tsx and FileBrowserTab.tsx and asserts the
 * absence of substrings that would route image bytes through JS memory:
 *   - `btoa(`            — base64 encoding in JS
 *   - `data:image/`      — data URIs constructed in code
 *   - `FileReader`       — FileReader-based bytes → string conversion
 *   - `.toDataURL`       — canvas-based bytes → data URI
 *   - `URL.createObjectURL` — blob: URL construction (Pitfall 10 extension)
 *
 * The image preview MUST use a direct <img src=URL> bound to the daemon's
 * /api/files/read endpoint — the bytes never enter JS.
 */
import { describe, it, expect } from 'vitest'
import { readFileSync, existsSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const IMG_PREVIEW = resolve(__dirname, '..', 'ImagePreview.tsx')
const FB_TAB = resolve(__dirname, '..', '..', 'FileBrowserTab.tsx')

const BANNED = ['btoa(', 'data:image/', 'FileReader', '.toDataURL', 'URL.createObjectURL']

function findOffenders(src: string, banned: string): string[] {
  return src
    .split('\n')
    .filter((line) => {
      if (!line.includes(banned)) return false
      const trimmed = line.trim()
      // Allow doc-comment / single-line comment mentions.
      if (trimmed.startsWith('//')) return false
      if (trimmed.startsWith('*')) return false
      return true
    })
}

function readIfExists(path: string): string {
  if (!existsSync(path)) return ''
  return readFileSync(path, 'utf8')
}

describe('no-base64 / no-blob-url source guard', () => {
  for (const banned of BANNED) {
    it(`ImagePreview.tsx contains no ${banned}`, () => {
      const src = readFileSync(IMG_PREVIEW, 'utf8')
      expect(findOffenders(src, banned)).toEqual([])
    })
  }

  it('FileBrowserTab.tsx (if exists) contains none of the banned patterns', () => {
    const src = readIfExists(FB_TAB)
    if (src === '') return
    for (const banned of BANNED) {
      expect(findOffenders(src, banned), `${banned} found`).toEqual([])
    }
  })
})
