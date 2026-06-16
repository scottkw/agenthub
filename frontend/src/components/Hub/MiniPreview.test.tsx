import { describe, it, expect, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import { MiniPreview } from './MiniPreview'

function renderPreview(lines: string[] | undefined) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<MiniPreview lines={lines} />)
  })
  return { container, root }
}

describe('MiniPreview', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  describe('loading state (lines === undefined)', () => {
    it('renders "Loading…" text', () => {
      const { container } = renderPreview(undefined)
      expect(container.textContent).toContain('Loading…')
    })

    it('applies hub-card__preview--loading modifier class', () => {
      const { container } = renderPreview(undefined)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).toContain('hub-card__preview--loading')
    })

    it('has aria-hidden="true" on the outer pane', () => {
      const { container } = renderPreview(undefined)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.getAttribute('aria-hidden')).toBe('true')
    })
  })

  describe('empty state (lines === [])', () => {
    it('renders "No output yet" text', () => {
      const { container } = renderPreview([])
      expect(container.textContent).toContain('No output yet')
    })

    it('applies hub-card__preview--empty modifier class', () => {
      const { container } = renderPreview([])
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).toContain('hub-card__preview--empty')
    })

    it('has aria-hidden="true" on the outer pane', () => {
      const { container } = renderPreview([])
      const pane = container.firstElementChild as HTMLElement
      expect(pane.getAttribute('aria-hidden')).toBe('true')
    })
  })

  describe('data state (lines with content)', () => {
    it('renders 4 lines with correct text', () => {
      const lines = ['line one', 'line two', 'line three', 'line four']
      const { container } = renderPreview(lines)
      const lineEls = container.querySelectorAll('.hub-card__preview-line')
      expect(lineEls).toHaveLength(4)
      lines.forEach((text, i) => {
        expect(lineEls[i].textContent).toBe(text)
      })
    })

    it('has aria-hidden="true" on the outer pane', () => {
      const { container } = renderPreview(['a', 'b'])
      const pane = container.firstElementChild as HTMLElement
      expect(pane.getAttribute('aria-hidden')).toBe('true')
    })

    it('does NOT apply loading or empty modifier classes in data state', () => {
      const { container } = renderPreview(['hello'])
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).not.toContain('--loading')
      expect(pane.className).not.toContain('--empty')
    })

    it('renders outer pane with hub-card__preview class', () => {
      const { container } = renderPreview(['hello'])
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).toContain('hub-card__preview')
    })
  })

  describe('empty-string line (non-collapsing)', () => {
    it('renders empty string as a non-breaking space to preserve row height', () => {
      const { container } = renderPreview(['', 'text'])
      const lineEls = container.querySelectorAll('.hub-card__preview-line')
      expect(lineEls).toHaveLength(2)
      // Empty-string line renders as ' ' (space) to keep height
      expect(lineEls[0].textContent).toBe(' ')
      expect(lineEls[1].textContent).toBe('text')
    })
  })
})
