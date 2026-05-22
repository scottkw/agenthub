/**
 * @vitest-environment jsdom
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { act } from 'react'
import { BreadcrumbBar } from '../BreadcrumbBar'
import type { BreadcrumbSegment } from '../../../lib/filesTypes'

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

const baseSegments: BreadcrumbSegment[] = [
  { name: 'session', pathFromCwd: '' },
  { name: 'src', pathFromCwd: 'src' },
  { name: 'components', pathFromCwd: 'src/components' },
]

describe('BreadcrumbBar', () => {
  it('renders one <li> per BreadcrumbSegment input', () => {
    render(
      <BreadcrumbBar
        segments={baseSegments}
        refreshedAt={null}
        onNavigateTo={vi.fn()}
        onRefresh={vi.fn()}
      />,
    )
    const items = container.querySelectorAll('nav[data-testid="file-browser-breadcrumb"] ol li')
    expect(items.length).toBe(3)
  })

  it('root is non-clickable when user is AT root (1 segment)', () => {
    render(
      <BreadcrumbBar
        segments={[{ name: 'session', pathFromCwd: '' }]}
        refreshedAt={null}
        onNavigateTo={vi.fn()}
        onRefresh={vi.fn()}
      />,
    )
    const firstSeg = container.querySelector(
      '[data-testid="file-browser-breadcrumb-segment-0"]',
    )
    expect(firstSeg).not.toBeNull()
    expect(firstSeg!.tagName.toLowerCase()).toBe('span')
    expect(firstSeg!.getAttribute('aria-current')).toBe('page')
  })

  it('root IS clickable when user is in a subdirectory (regression for UAT-1 bug)', () => {
    // User clicks "agenthub" subdir → breadcrumb becomes [session, agenthub]
    // Root must be a clickable button so user can navigate back to cwd root.
    const onNavigateTo = vi.fn()
    render(
      <BreadcrumbBar
        segments={[
          { name: 'session', pathFromCwd: '' },
          { name: 'agenthub', pathFromCwd: 'agenthub' },
        ]}
        refreshedAt={null}
        onNavigateTo={onNavigateTo}
        onRefresh={vi.fn()}
      />,
    )
    const firstSeg = container.querySelector(
      '[data-testid="file-browser-breadcrumb-segment-0"]',
    ) as HTMLElement
    expect(firstSeg).not.toBeNull()
    expect(firstSeg.tagName.toLowerCase()).toBe('button')
    act(() => {
      firstSeg.click()
    })
    expect(onNavigateTo).toHaveBeenCalledWith('')
  })

  it('root is clickable in deep paths too (3+ segments)', () => {
    // baseSegments = [session, src, components] — root is not current, must be a button.
    const onNavigateTo = vi.fn()
    render(
      <BreadcrumbBar
        segments={baseSegments}
        refreshedAt={null}
        onNavigateTo={onNavigateTo}
        onRefresh={vi.fn()}
      />,
    )
    const firstSeg = container.querySelector(
      '[data-testid="file-browser-breadcrumb-segment-0"]',
    ) as HTMLElement
    expect(firstSeg.tagName.toLowerCase()).toBe('button')
    act(() => {
      firstSeg.click()
    })
    expect(onNavigateTo).toHaveBeenCalledWith('')
  })

  it('last segment has aria-current="page" and is text-only (not a button)', () => {
    render(
      <BreadcrumbBar
        segments={baseSegments}
        refreshedAt={null}
        onNavigateTo={vi.fn()}
        onRefresh={vi.fn()}
      />,
    )
    const lastSeg = container.querySelector(
      '[data-testid="file-browser-breadcrumb-segment-2"]',
    )
    expect(lastSeg).not.toBeNull()
    expect(lastSeg!.getAttribute('aria-current')).toBe('page')
    expect(lastSeg!.tagName.toLowerCase()).not.toBe('button')
  })

  it('clicking a middle segment fires onNavigateTo with its pathFromCwd', () => {
    const onNavigateTo = vi.fn()
    render(
      <BreadcrumbBar
        segments={baseSegments}
        refreshedAt={null}
        onNavigateTo={onNavigateTo}
        onRefresh={vi.fn()}
      />,
    )
    const middle = container.querySelector(
      '[data-testid="file-browser-breadcrumb-segment-1"]',
    ) as HTMLElement
    expect(middle.tagName.toLowerCase()).toBe('button')
    act(() => {
      middle.click()
    })
    expect(onNavigateTo).toHaveBeenCalledWith('src')
  })

  it('"Last refreshed Ns ago" text reflects time delta', () => {
    const twelveSecondsAgo = new Date(Date.now() - 12_000).toISOString()
    render(
      <BreadcrumbBar
        segments={baseSegments}
        refreshedAt={twelveSecondsAgo}
        onNavigateTo={vi.fn()}
        onRefresh={vi.fn()}
      />,
    )
    const refreshedText = container.querySelector(
      '.file-browser__breadcrumb-refreshed',
    )
    expect(refreshedText).not.toBeNull()
    expect(refreshedText!.textContent).toMatch(/Last refreshed \d+s ago/)
  })

  it('refresh button click fires onRefresh', () => {
    const onRefresh = vi.fn()
    render(
      <BreadcrumbBar
        segments={baseSegments}
        refreshedAt={null}
        onNavigateTo={vi.fn()}
        onRefresh={onRefresh}
      />,
    )
    const refreshBtn = container.querySelector(
      '[data-testid="file-browser-refresh"]',
    ) as HTMLElement
    expect(refreshBtn).not.toBeNull()
    act(() => {
      refreshBtn.click()
    })
    expect(onRefresh).toHaveBeenCalledTimes(1)
  })

  it('exposes UI-SPEC data-testid attributes', () => {
    render(
      <BreadcrumbBar
        segments={baseSegments}
        refreshedAt={null}
        onNavigateTo={vi.fn()}
        onRefresh={vi.fn()}
      />,
    )
    expect(container.querySelector('[data-testid="file-browser-breadcrumb"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="file-browser-breadcrumb-segment-0"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="file-browser-breadcrumb-segment-1"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="file-browser-breadcrumb-segment-2"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="file-browser-refresh"]')).not.toBeNull()
  })
})
