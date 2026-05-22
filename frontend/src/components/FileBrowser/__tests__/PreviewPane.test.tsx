/**
 * @vitest-environment jsdom
 *
 * Phase 120-04 Task 1 — PreviewPane dispatcher tests.
 *
 * Covers all 11 PreviewState kinds. Tests assert DOM-visible behavior, not
 * internal component identity: the dispatcher must produce the right testid
 * + the right copy + the right control wiring for each branch.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { act } from 'react'
import { PreviewPane } from '../PreviewPane'

let container: HTMLDivElement
let root: Root

beforeEach(() => {
  container = document.createElement('div')
  document.body.appendChild(container)
  root = createRoot(container)
})

afterEach(() => {
  act(() => {
    root.unmount()
  })
  container.remove()
})

function render(node: React.ReactElement): void {
  act(() => {
    root.render(node)
  })
}

describe('PreviewPane', () => {
  it('wraps everything in role="region" aria-label="File preview"', () => {
    render(
      <PreviewPane state={{ kind: 'idle' }} filename={null} downloadUrl={null} />,
    )
    const region = container.querySelector('[data-testid="file-browser-preview"]')
    expect(region).not.toBeNull()
    expect(region!.getAttribute('role')).toBe('region')
    expect(region!.getAttribute('aria-label')).toBe('File preview')
    expect(region!.getAttribute('aria-live')).toBe('polite')
  })

  it('idle state renders "Select a file to preview"', () => {
    render(
      <PreviewPane state={{ kind: 'idle' }} filename={null} downloadUrl={null} />,
    )
    expect(container.textContent).toContain('Select a file to preview')
  })

  it('loading state renders a loading affordance', () => {
    render(
      <PreviewPane state={{ kind: 'loading' }} filename={'x.txt'} downloadUrl={null} />,
    )
    const loading = container.querySelector(
      '[data-testid="file-browser-preview-loading"]',
    )
    expect(loading).not.toBeNull()
    expect(container.textContent).toContain('Loading preview')
  })

  it('text state renders TextPreview with the source text', () => {
    render(
      <PreviewPane
        state={{ kind: 'text', text: 'hello world', size: 11, mtime: '2026-05-20' }}
        filename={'note.txt'}
        downloadUrl={'http://x/api/files/read?path=note.txt'}
      />,
    )
    const txt = container.querySelector('[data-testid="file-browser-preview-text"]')
    expect(txt).not.toBeNull()
    expect(txt!.textContent).toBe('hello world')
  })

  it('markdown state renders MarkdownPreview', () => {
    render(
      <PreviewPane
        state={{ kind: 'markdown', text: '# Hello', size: 7, mtime: '2026-05-20' }}
        filename={'README.md'}
        downloadUrl={'http://x/api/files/read?path=README.md'}
      />,
    )
    const md = container.querySelector('[data-testid="file-browser-preview-markdown"]')
    expect(md).not.toBeNull()
    // react-markdown renders # into an <h1> with the text
    const h1 = container.querySelector('h1')
    expect(h1).not.toBeNull()
    expect(h1!.textContent).toBe('Hello')
  })

  it('image state renders <img> with src equal to the provided url', () => {
    const url = 'http://127.0.0.1:6789/api/files/read?session=s1&path=cat.png'
    render(
      <PreviewPane
        state={{ kind: 'image', url, size: 12345, mtime: '2026-05-20' }}
        filename={'cat.png'}
        downloadUrl={url}
      />,
    )
    const img = container.querySelector(
      '[data-testid="file-browser-preview-image"] img',
    ) as HTMLImageElement | null
    expect(img).not.toBeNull()
    expect(img!.getAttribute('src')).toBe(url)
  })

  it('unsupported state renders Download button with href = downloadUrl', () => {
    const url = 'http://x/api/files/read?session=s&path=blob.zip'
    render(
      <PreviewPane
        state={{
          kind: 'unsupported',
          filename: 'blob.zip',
          downloadUrl: url,
          humanSize: '2.0 MB',
        }}
        filename={'blob.zip'}
        downloadUrl={url}
      />,
    )
    const binary = container.querySelector('[data-testid="file-browser-binary"]')
    expect(binary).not.toBeNull()
    // Find the body-area download button (inside UnsupportedFile, not the header)
    const downloads = container.querySelectorAll('[data-testid="file-browser-download"]')
    expect(downloads.length).toBeGreaterThanOrEqual(1)
    const bodyDownload = Array.from(downloads).find((el) =>
      el.closest('[data-testid="file-browser-binary"]'),
    ) as HTMLElement | undefined
    expect(bodyDownload).toBeDefined()
    // Phase 120 UAT-1: in Wails desktop mode the button has data-download-url
    // (no href); in web mode the <a> has href. Either way the URL plumbs.
    const got =
      bodyDownload!.getAttribute('href') ??
      bodyDownload!.getAttribute('data-download-url')
    expect(got).toBe(url)
  })

  it('over-cap state renders the size in the body copy', () => {
    const url = 'http://x/api/files/read?session=s&path=huge.bin'
    render(
      <PreviewPane
        state={{
          kind: 'over-cap',
          filename: 'huge.bin',
          downloadUrl: url,
          humanSize: '12.4 MB',
        }}
        filename={'huge.bin'}
        downloadUrl={url}
      />,
    )
    const overCap = container.querySelector('[data-testid="file-browser-over-cap"]')
    expect(overCap).not.toBeNull()
    expect(overCap!.textContent).toContain('12.4 MB')
    expect(overCap!.textContent).toContain('File too large to preview')
  })

  it('empty state renders "Empty file" + "0 bytes"', () => {
    render(
      <PreviewPane
        state={{ kind: 'empty', filename: 'empty.txt' }}
        filename={'empty.txt'}
        downloadUrl={'http://x/api/files/read?path=empty.txt'}
      />,
    )
    expect(container.textContent).toContain('Empty file')
    expect(container.textContent).toContain('0 bytes')
  })

  it('broken-symlink renders target path', () => {
    render(
      <PreviewPane
        state={{
          kind: 'broken-symlink',
          filename: 'dangling',
          targetPath: '../missing/foo',
        }}
        filename={'dangling'}
        downloadUrl={null}
      />,
    )
    const bs = container.querySelector('[data-testid="file-browser-broken-symlink"]')
    expect(bs).not.toBeNull()
    expect(bs!.textContent).toContain('Broken symlink')
    expect(bs!.textContent).toContain('../missing/foo')
  })

  it('read-error renders Retry button; clicking calls onRetry', () => {
    const onRetry = vi.fn()
    render(
      <PreviewPane
        state={{
          kind: 'read-error',
          filename: 'failing.txt',
          message: 'network error',
          onRetry,
        }}
        filename={'failing.txt'}
        downloadUrl={null}
      />,
    )
    const err = container.querySelector('[data-testid="file-browser-network-error"]')
    expect(err).not.toBeNull()
    const retry = container.querySelector(
      '[data-testid="file-browser-network-error-retry"]',
    ) as HTMLButtonElement | null
    expect(retry).not.toBeNull()
    act(() => {
      retry!.click()
    })
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('forbidden-file renders explicit copy (not raw 403)', () => {
    render(
      <PreviewPane
        state={{ kind: 'forbidden-file', filename: 'secret.txt' }}
        filename={'secret.txt'}
        downloadUrl={null}
      />,
    )
    const forb = container.querySelector('[data-testid="file-browser-forbidden"]')
    expect(forb).not.toBeNull()
    expect(forb!.textContent).toContain('Cannot read this file')
    expect(forb!.textContent).toContain('The session does not have permission')
    expect(forb!.textContent).not.toContain('403')
  })

  it('renders preview header (name + size) for kinds with a backing file', () => {
    render(
      <PreviewPane
        state={{ kind: 'text', text: 'abc', size: 1024, mtime: '2026-05-20' }}
        filename={'note.txt'}
        downloadUrl={'http://x/api/files/read?path=note.txt'}
      />,
    )
    const nameEl = container.querySelector(
      '[data-testid="file-browser-preview-name"]',
    )
    expect(nameEl).not.toBeNull()
    expect(nameEl!.textContent).toBe('note.txt')
    expect(container.textContent).toContain('1.0 KB')
  })

  it('does NOT render preview header for idle state', () => {
    render(
      <PreviewPane state={{ kind: 'idle' }} filename={null} downloadUrl={null} />,
    )
    const nameEl = container.querySelector(
      '[data-testid="file-browser-preview-name"]',
    )
    expect(nameEl).toBeNull()
  })
})
