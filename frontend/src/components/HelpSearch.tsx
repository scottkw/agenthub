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

/**
 * Extract a centered ~120-char snippet around the first match of `query` in
 * `text`. Returns the raw text if no match (shouldn't happen since results are
 * pre-filtered). Adds '…' ellipsis at truncation points.
 */
function extractSnippet(text: string, query: string): string {
  const lowerText = text.toLowerCase()
  const lowerQuery = query.toLowerCase()
  const idx = lowerText.indexOf(lowerQuery)
  if (idx === -1) return text.length > 120 ? text.slice(0, 120) + '…' : text

  const HALF = 60
  const start = Math.max(0, idx - HALF)
  const end = Math.min(text.length, idx + query.length + HALF)
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
          {results.map((r, i) => (
            <li
              key={`${r.sectionId}-${i}`}
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
