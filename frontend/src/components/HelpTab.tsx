// Phase 147: In-App Help Page — container component.
//
// Owns: search query state (with 200ms debounce), active section state,
// the per-paragraph search index (built once via useMemo([])), and the
// filtered results. Assembles HelpSearch + HelpSectionNav + HelpContent
// into the full Help page layout.

import React, { useState, useMemo, useCallback, useRef, useEffect } from 'react'
import { HelpSearch } from './HelpSearch'
import { HelpSectionNav } from './HelpSectionNav'
import { HelpContent } from './HelpContent'
import gettingStartedMd from '../content/help/getting-started.md?raw'
import faqMd from '../content/help/faq.md?raw'

// ---------------------------------------------------------------------------
// Types (also exported so tests + other components can import them)
// ---------------------------------------------------------------------------

export interface SearchEntry {
  sectionId: string
  sectionLabel: string
  text: string
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Strip common Markdown syntax characters to produce plain text suitable
 * for full-text search. This is a lightweight implementation — good enough
 * for the maintainer-authored bundled content.
 */
function stripMd(text: string): string {
  return (
    text
      // Remove headings
      .replace(/^#{1,6}\s+/gm, '')
      // Remove bold/italic markers
      .replace(/\*{1,3}([^*]+)\*{1,3}/g, '$1')
      .replace(/_{1,3}([^_]+)_{1,3}/g, '$1')
      // Remove inline code
      .replace(/`([^`]+)`/g, '$1')
      // Remove links — keep link text
      .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
      // Remove images
      .replace(/!\[([^\]]*)\]\([^)]+\)/g, '$1')
      // Remove HTML tags
      .replace(/<[^>]+>/g, '')
      // Collapse whitespace
      .replace(/\s+/g, ' ')
      .trim()
  )
}

// ---------------------------------------------------------------------------
// Section metadata (must match HelpSectionNav's SECTIONS)
// ---------------------------------------------------------------------------

const SECTION_META: ReadonlyArray<{ id: string; label: string; markdown: string }> = [
  { id: 'help-getting-started', label: 'Getting Started', markdown: gettingStartedMd },
  { id: 'help-faq', label: 'Frequently Asked Questions', markdown: faqMd },
]

// ---------------------------------------------------------------------------
// HelpTab
// ---------------------------------------------------------------------------

export function HelpTab(): React.ReactElement {
  // All Markdown concatenated for the content pane (single render with both sections)
  const allMarkdown = useMemo(() => `${gettingStartedMd}\n\n${faqMd}`, [])

  // Search state
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Active section (starts at the first section)
  const [activeSection, setActiveSection] = useState<string>('help-getting-started')

  // Content pane ref passed to HelpSectionNav for IntersectionObserver root
  const contentPaneRef = useRef<HTMLDivElement>(null)

  // Debounced query handler: update display query immediately, defer search
  const handleQueryChange = useCallback((raw: string) => {
    setQuery(raw)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => setDebouncedQuery(raw), 200)
  }, [])

  // Cleanup debounce timer on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [])

  // Search index — built once at mount (module-level constants so deps=[])
  const searchIndex = useMemo<ReadonlyArray<SearchEntry>>(() => {
    const entries: SearchEntry[] = []
    for (const { id, label, markdown } of SECTION_META) {
      // Split on blank lines (paragraph boundaries)
      const paragraphs = markdown.split(/\n\n+/)
      for (const para of paragraphs) {
        const plain = stripMd(para)
        // Skip short fragments (headings, stubs)
        if (plain.length > 20) {
          entries.push({ sectionId: id, sectionLabel: label, text: plain })
        }
      }
    }
    return entries
  }, []) // [] — gettingStartedMd and faqMd are module-level constants

  // Filtered results (empty for blank query)
  const results = useMemo<ReadonlyArray<SearchEntry>>(() => {
    const q = debouncedQuery.trim().toLowerCase()
    if (!q) return []
    return searchIndex.filter((e) => e.text.toLowerCase().includes(q))
  }, [debouncedQuery, searchIndex])

  // Jump to section: scroll + clear search + update activeSection
  const handleJumpToSection = useCallback((sectionId: string) => {
    const el = document.getElementById(sectionId)
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    setActiveSection(sectionId)
    handleQueryChange('')
  }, [handleQueryChange])

  return (
    <div className="help-tab">
      {/* Sticky search bar */}
      <div className="help-tab__search">
        <HelpSearch
          query={query}
          results={results}
          onQueryChange={handleQueryChange}
          onJumpToSection={handleJumpToSection}
        />
      </div>

      {/* Two-column layout: left nav + content pane */}
      <div className="help-tab__layout">
        <HelpSectionNav
          activeSection={activeSection}
          onSectionChange={setActiveSection}
          contentPaneRef={contentPaneRef}
        />
        <div className="help-tab__content" ref={contentPaneRef}>
          <h1 className="help-tab__title">Help</h1>
          <HelpContent markdown={allMarkdown} />
        </div>
      </div>
    </div>
  )
}
