// Phase 147: In-App Help Page — Search input, snippet highlight, empty state.
//
// HelpSearch receives pre-filtered results from HelpTab (debounce lives there)
// and renders: a visible label, search input, clear button, snippet results with
// <mark> highlight, and a GitHub Issues empty state. External links use
// BrowserOpenURL — never <a href>.

import React from 'react'
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'

export interface SearchEntry {
  sectionId: string
  sectionLabel: string
  text: string
}

interface HelpSearchProps {
  query: string
  results: ReadonlyArray<SearchEntry>
  onQueryChange: (raw: string) => void
  onJumpToSection: (id: string) => void
}

// ---------------------------------------------------------------------------
// Snippet helpers
// ---------------------------------------------------------------------------

// Characters of context to show on each side of a match within a snippet.
const SNIPPET_RADIUS = 60
// Maximum length of the no-match fallback snippet (≈ a match plus both radii).
const SNIPPET_MAX = SNIPPET_RADIUS * 2

// UTF-16 surrogate ranges. slice() operates on code units, so a cut that lands
// between a high and low surrogate splits a single code point (emoji, some CJK)
// into a lone surrogate that renders as '�'.
const HIGH_SURROGATE_START = 0xd800
const HIGH_SURROGATE_END = 0xdbff
const LOW_SURROGATE_START = 0xdc00
const LOW_SURROGATE_END = 0xdfff

function isHighSurrogate(code: number): boolean {
  return code >= HIGH_SURROGATE_START && code <= HIGH_SURROGATE_END
}

function isLowSurrogate(code: number): boolean {
  return code >= LOW_SURROGATE_START && code <= LOW_SURROGATE_END
}

/**
 * Adjust a slice boundary so it never falls between the two halves of a
 * surrogate pair. A `start` boundary is nudged forward past a leading low
 * surrogate; an `end` boundary is nudged back before a trailing high surrogate.
 */
function clampToCodePoint(text: string, index: number, boundary: 'start' | 'end'): number {
  if (index <= 0 || index >= text.length) return index
  if (boundary === 'start') {
    // If index points at a low surrogate, its high surrogate is at index-1, so
    // the pair was already split before index — step forward past the low half.
    return isLowSurrogate(text.charCodeAt(index)) ? index + 1 : index
  }
  // boundary === 'end': if the char just before the cut is a high surrogate, its
  // low half would be dropped — pull the cut back before the high surrogate.
  return isHighSurrogate(text.charCodeAt(index - 1)) ? index - 1 : index
}

/**
 * Extract a centered snippet (~SNIPPET_MAX chars) around the first match of
 * `query` in `text`. Returns the raw text if no match (shouldn't happen since
 * results are pre-filtered). Adds '…' ellipsis at truncation points. Slice
 * boundaries are clamped so they never split a Unicode surrogate pair.
 */
function extractSnippet(text: string, query: string): string {
  const lowerText = text.toLowerCase()
  const lowerQuery = query.toLowerCase()
  const idx = lowerText.indexOf(lowerQuery)
  if (idx === -1) {
    if (text.length <= SNIPPET_MAX) return text
    const cut = clampToCodePoint(text, SNIPPET_MAX, 'end')
    return text.slice(0, cut) + '…'
  }

  const start = clampToCodePoint(text, Math.max(0, idx - SNIPPET_RADIUS), 'start')
  const end = clampToCodePoint(
    text,
    Math.min(text.length, idx + query.length + SNIPPET_RADIUS),
    'end',
  )
  const snippet =
    (start > 0 ? '…' : '') +
    text.slice(start, end) +
    (end < text.length ? '…' : '')
  return snippet
}

/**
 * Render a text snippet with the matched query substring wrapped in
 * <mark class="help-search__mark">. Uses plain-string split — no
 * dangerouslySetInnerHTML.
 */
function HighlightedSnippet({
  text,
  query,
}: {
  text: string
  query: string
}): React.ReactElement {
  const snippet = extractSnippet(text, query)
  if (!query) return <span>{snippet}</span>

  const lowerSnippet = snippet.toLowerCase()
  const lowerQuery = query.toLowerCase()
  const parts: React.ReactNode[] = []
  let lastIndex = 0

  let idx = lowerSnippet.indexOf(lowerQuery, lastIndex)
  while (idx !== -1) {
    if (idx > lastIndex) {
      parts.push(snippet.slice(lastIndex, idx))
    }
    parts.push(
      <mark key={idx} className="help-search__mark">
        {snippet.slice(idx, idx + query.length)}
      </mark>,
    )
    lastIndex = idx + query.length
    idx = lowerSnippet.indexOf(lowerQuery, lastIndex)
  }
  if (lastIndex < snippet.length) {
    parts.push(snippet.slice(lastIndex))
  }

  return <span>{parts}</span>
}

// ---------------------------------------------------------------------------
// HelpSearch component
// ---------------------------------------------------------------------------

export function HelpSearch({
  query,
  results,
  onQueryChange,
  onJumpToSection,
}: HelpSearchProps): React.ReactElement {
  const showEmpty = query.trim() !== '' && results.length === 0
  const showResults = results.length > 0

  return (
    <div className="help-search">
      <label htmlFor="help-search-input">Search help…</label>
      <div className="help-search__row">
        <input
          id="help-search-input"
          type="search"
          className="help-search__input"
          placeholder="Search help…"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Escape') onQueryChange('')
          }}
        />
        <button
          type="button"
          className="help-search__clear"
          aria-label="Clear search"
          onClick={() => onQueryChange('')}
        >
          ×
        </button>
      </div>

      {showResults && (
        <ul className="help-search__results" role="listbox">
          {results.map((r) => (
            <li
              key={`${r.sectionId}-${r.text.slice(0, 48)}`}
              className="help-search__result"
              role="option"
              aria-selected="false"
              tabIndex={0}
              onClick={() => onJumpToSection(r.sectionId)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  onJumpToSection(r.sectionId)
                }
              }}
            >
              <div className="help-search__snippet">
                <HighlightedSnippet text={r.text} query={query} />
              </div>
              <button
                type="button"
                className="help-search__jump"
                onClick={(e) => {
                  e.stopPropagation()
                  onJumpToSection(r.sectionId)
                }}
              >
                Go to {r.sectionLabel} →
              </button>
            </li>
          ))}
        </ul>
      )}

      {showEmpty && (
        <div className="help-search__empty">
          <p className="help-search__empty-heading">
            No results for &ldquo;{query}&rdquo;
          </p>
          <p className="help-search__empty-body">
            Try a different search term, or{' '}
            <button
              type="button"
              className="help-content__external-link"
              onClick={() =>
                BrowserOpenURL('https://github.com/scottkw/agenthub/issues')
              }
              aria-label="GitHub Issues (opens in browser)"
            >
              report an issue on GitHub
            </button>
            .
          </p>
        </div>
      )}
    </div>
  )
}
