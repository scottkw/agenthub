import React from 'react'
import { render } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { MiniPreview } from './MiniPreview'

describe('MiniPreview', () => {
  describe('loading state (lines === undefined)', () => {
    it('renders "Loading…" text', () => {
      const { getByText } = render(<MiniPreview lines={undefined} />)
      expect(getByText('Loading…')).toBeTruthy()
    })

    it('applies hub-card__preview--loading modifier class', () => {
      const { container } = render(<MiniPreview lines={undefined} />)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).toContain('hub-card__preview--loading')
    })

    it('has aria-hidden="true" on the outer pane', () => {
      const { container } = render(<MiniPreview lines={undefined} />)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.getAttribute('aria-hidden')).toBe('true')
    })
  })

  describe('empty state (lines === [])', () => {
    it('renders "No output yet" text', () => {
      const { getByText } = render(<MiniPreview lines={[]} />)
      expect(getByText('No output yet')).toBeTruthy()
    })

    it('applies hub-card__preview--empty modifier class', () => {
      const { container } = render(<MiniPreview lines={[]} />)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).toContain('hub-card__preview--empty')
    })

    it('has aria-hidden="true" on the outer pane', () => {
      const { container } = render(<MiniPreview lines={[]} />)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.getAttribute('aria-hidden')).toBe('true')
    })
  })

  describe('data state (lines with content)', () => {
    it('renders 4 lines with correct text', () => {
      const lines = ['line one', 'line two', 'line three', 'line four']
      const { container } = render(<MiniPreview lines={lines} />)
      const lineEls = container.querySelectorAll('.hub-card__preview-line')
      expect(lineEls).toHaveLength(4)
      lines.forEach((text, i) => {
        expect(lineEls[i].textContent).toBe(text)
      })
    })

    it('has aria-hidden="true" on the outer pane', () => {
      const { container } = render(<MiniPreview lines={['a', 'b']} />)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.getAttribute('aria-hidden')).toBe('true')
    })

    it('does NOT apply loading or empty modifier classes in data state', () => {
      const { container } = render(<MiniPreview lines={['hello']} />)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).not.toContain('--loading')
      expect(pane.className).not.toContain('--empty')
    })

    it('renders outer pane with hub-card__preview class', () => {
      const { container } = render(<MiniPreview lines={['hello']} />)
      const pane = container.firstElementChild as HTMLElement
      expect(pane.className).toContain('hub-card__preview')
    })
  })

  describe('empty-string line (non-collapsing)', () => {
    it('renders empty string as a non-breaking space to preserve row height', () => {
      const { container } = render(<MiniPreview lines={['', 'text']} />)
      const lineEls = container.querySelectorAll('.hub-card__preview-line')
      expect(lineEls).toHaveLength(2)
      // Empty-string line renders as ' ' (non-breaking space) to keep height
      expect(lineEls[0].textContent).toBe(' ')
      expect(lineEls[1].textContent).toBe('text')
    })
  })
})
