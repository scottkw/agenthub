/**
 * FindBar — Phase 94 SRC-01 / SRC-02 / SRC-04.
 *
 * Fully controlled React component. Owns ZERO search state — every value
 * (query, matchCount, currentMatchIndex, searchOptions) is provided by the
 * parent (TerminalPanel), and every interaction notifies the parent through
 * a callback prop. The parent owns the SearchAddon instance and the
 * SetPluginSettings persistence call (Pitfall #2 — never re-sync mid-open).
 *
 * Visual contract: 94-UI-SPEC §"Find Bar: Detailed Visual Specification"
 *   — verbatim copy/aria/CSS class names; zero design discretion.
 *
 * Decoration / theme contract (SRC-04): the parent's findNext call MUST omit
 * the `decorations` ISearchOptions field so xterm uses theme.selectionBackground
 * automatically across all 138 themes. FindBar itself never invokes
 * SearchAddon directly, so there is no decorations site here — verified by
 * FindBar.visual.test.tsx source-inspection.
 */
import React, { useEffect, useRef } from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'

export interface FindBarSearchOptions {
  regex: boolean
  caseSensitive: boolean
  wholeWord: boolean
}

export interface FindBarProps {
  query: string
  onQueryChange: (q: string) => void
  /** Total match count from SearchAddon onDidChangeResults. 0 when query empty or no matches. */
  matchCount: number
  /** 0-based current match index from SearchAddon. -1 when no query / no matches. */
  currentMatchIndex: number
  searchOptions: FindBarSearchOptions
  onSearchOptionsChange: (opts: FindBarSearchOptions) => void
  onNext: () => void
  onPrev: () => void
  onClose: () => void
  /**
   * Optional sequence counter — when bumped, FindBar re-focuses its input.
   * TerminalPanel uses this to handle Cmd-F-while-already-open by re-focusing
   * the search field even though `findBarOpen` did not flip false→true.
   */
  focusSeq?: number
}

export function FindBar({
  query,
  onQueryChange,
  matchCount,
  currentMatchIndex,
  searchOptions,
  onSearchOptionsChange,
  onNext,
  onPrev,
  onClose,
  focusSeq,
}: FindBarProps): React.ReactElement {
  const inputRef = useRef<HTMLInputElement>(null)

  // Auto-focus on mount + on focusSeq bump (Cmd-F when bar already open).
  useEffect(() => {
    inputRef.current?.focus()
  }, [focusSeq])

  function handleContainerKeyDown(e: React.KeyboardEvent<HTMLDivElement>): void {
    if (e.key === 'Escape') {
      e.stopPropagation()
      onClose()
    }
  }

  function handleInputKeyDown(e: React.KeyboardEvent<HTMLInputElement>): void {
    // Cmd-G / Ctrl-G next; Cmd-Shift-G / Ctrl-Shift-G prev (UI-SPEC
    // §"Next / Previous Match"). The Cmd-F that opened the bar is intercepted
    // at TerminalPanel's window handler — once the input is focused, Cmd-F is
    // handled there if at all (re-focuses input via focusSeq bump).
    const modifier = e.metaKey || e.ctrlKey
    if (modifier && e.key.toLowerCase() === 'g') {
      e.preventDefault()
      if (e.shiftKey) onPrev()
      else onNext()
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      if (e.shiftKey) onPrev()
      else onNext()
    }
  }

  // Match count rendering rules (UI-SPEC §"Match Count" + Copywriting Contract).
  // - empty query  → hidden via .find-bar__count--hidden (visibility:hidden)
  // - has query, count=0 → "0 of 0" + .find-bar__count--no-results modifier
  // - has query, count>0 → "{i+1} of {count}", default color
  const hasQuery = query.length > 0
  const noResults = hasQuery && matchCount === 0
  const countClass = [
    'find-bar__count',
    !hasQuery && 'find-bar__count--hidden',
    noResults && 'find-bar__count--no-results',
  ]
    .filter(Boolean)
    .join(' ')
  const countText = !hasQuery
    ? '' // visibility:hidden preserves layout; text content irrelevant
    : matchCount === 0
      ? '0 of 0'
      : `${currentMatchIndex + 1} of ${matchCount}`

  function toggleClass(active: boolean, suffix: 'case' | 'regex' | 'word'): string {
    return [`find-bar__toggle--${suffix}`, active && 'find-bar__toggle--active']
      .filter(Boolean)
      .join(' ')
  }

  function flipOption(key: keyof FindBarSearchOptions): void {
    onSearchOptionsChange({ ...searchOptions, [key]: !searchOptions[key] })
  }

  // Disabled state for nav arrows (UI-SPEC §"Icon Buttons / Nav button DISABLED state").
  // Keep the buttons enabled when matches exist; even a single match supports
  // wrap-around via SearchAddon. Mark as disabled only when there are no matches.
  const navDisabled = matchCount === 0

  return (
    <div
      className="find-bar"
      role="search"
      aria-label="Find in terminal"
      onKeyDown={handleContainerKeyDown}
    >
      <input
        ref={inputRef}
        className="find-bar__input"
        type="text"
        placeholder="Find…"
        aria-label="Search"
        autoComplete="off"
        spellCheck={false}
        value={query}
        onChange={(e) => onQueryChange(e.target.value)}
        onKeyDown={handleInputKeyDown}
      />
      <span className={countClass} aria-live="polite" aria-atomic="true">
        {countText}
      </span>
      <div className="find-bar__nav">
        <button
          type="button"
          className="find-bar__btn--prev"
          aria-label="Previous match"
          title="Previous match (Shift-Enter)"
          onClick={onPrev}
          disabled={navDisabled}
        >
          {/* Inline SVG ▲ (chevron-up) at 14×14, currentColor — matches UI-SPEC §"Internal Layout". */}
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
            <path
              d="M3 9l4-4 4 4"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </button>
        <button
          type="button"
          className="find-bar__btn--next"
          aria-label="Next match"
          title="Next match (Enter)"
          onClick={onNext}
          disabled={navDisabled}
        >
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
            <path
              d="M3 5l4 4 4-4"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </button>
      </div>
      <div className="find-bar__divider" aria-hidden="true" />
      <div className="find-bar__options">
        <button
          type="button"
          className={toggleClass(searchOptions.caseSensitive, 'case')}
          aria-label="Case sensitive"
          aria-pressed={searchOptions.caseSensitive}
          title="Case sensitive"
          onClick={() => flipOption('caseSensitive')}
        >
          <span style={{ fontSize: 11, fontWeight: 600, lineHeight: 1 }} aria-hidden="true">
            Aa
          </span>
        </button>
        <button
          type="button"
          className={toggleClass(searchOptions.regex, 'regex')}
          aria-label="Regular expression"
          aria-pressed={searchOptions.regex}
          title="Regular expression"
          onClick={() => flipOption('regex')}
        >
          <span style={{ fontSize: 11, fontWeight: 600, lineHeight: 1 }} aria-hidden="true">
            .*
          </span>
        </button>
        <button
          type="button"
          className={toggleClass(searchOptions.wholeWord, 'word')}
          aria-label="Whole word"
          aria-pressed={searchOptions.wholeWord}
          title="Whole word"
          onClick={() => flipOption('wholeWord')}
        >
          <span style={{ fontSize: 11, fontWeight: 600, lineHeight: 1 }} aria-hidden="true">
            [ab]
          </span>
        </button>
      </div>
      <button
        type="button"
        className="find-bar__close"
        aria-label="Close find bar"
        title="Close (Esc)"
        onClick={onClose}
      >
        <XMarkIcon style={{ width: 16, height: 16 }} aria-hidden="true" />
      </button>
    </div>
  )
}
