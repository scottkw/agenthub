import React, { useMemo, useState } from 'react'
import { SETTINGS_JUMP_LINKS } from './SettingsJumpBar'

/**
 * Phase 104 SETUI-03: Autocomplete-style search box. Users type a few
 * characters and pick a setting from the dropdown of matches; clicking
 * a result scrolls to the section anchor (`<a href="#settings-{slug}">`
 * uses the browser's built-in hash-jump, combined with
 * `scroll-margin-top` on the section h3 so the sticky jump-bar does not
 * occlude the heading).
 *
 * Index scope (per spec): section labels + top-level toggle/button labels.
 * Deep plugin sub-options are intentionally excluded — they live inside
 * the Plugins section's <details> blocks and are not directly indexable
 * without coupling tightly to PluginsSection internals.
 */
interface SearchEntry {
  /** User-visible label, e.g. "Start minimized to system tray". */
  label: string
  /** Anchor id (without "#"), e.g. "settings-behavior". */
  target: string
}

const SEARCH_INDEX: ReadonlyArray<SearchEntry> = [
  // Section headers — always indexable.
  ...SETTINGS_JUMP_LINKS.map((l) => ({ label: l.label, target: l.id })),
  // Top-level toggle and button labels (kept in sync with SettingsTab.tsx).
  { label: 'Start minimized to system tray', target: 'settings-behavior' },
  { label: 'Auto-close tab on exit', target: 'settings-session-behavior' },
  { label: 'Terminal Theme', target: 'settings-appearance' },
  { label: 'Tailscale Status', target: 'settings-web-server' },
  { label: 'Certificate Transparency', target: 'settings-web-server' },
  { label: 'Port', target: 'settings-web-server' },
  { label: 'Start Web Server', target: 'settings-web-server' },
  { label: 'Stop Web Server', target: 'settings-web-server' },
  { label: 'Regenerate Signing Key', target: 'settings-security' },
  { label: 'CLI Paths', target: 'settings-paths' },
]

export function SettingsSearch(): React.ReactElement {
  const [query, setQuery] = useState('')

  const results = useMemo<ReadonlyArray<SearchEntry>>(() => {
    const q = query.trim().toLowerCase()
    if (q === '') return []
    return SEARCH_INDEX.filter((e) => e.label.toLowerCase().includes(q))
  }, [query])

  function jumpTo(target: string): void {
    // Update the hash so the browser performs its native scroll-to-anchor
    // (respects `scroll-margin-top` and `scroll-behavior: smooth`).
    // Falling back to scrollIntoView keeps the jump working even when the
    // element is inside a scroll container that ignores the URL hash.
    const el = document.getElementById(target)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
    if (typeof window !== 'undefined' && window.location) {
      // Reset then set so clicking the same target twice still jumps.
      try {
        window.history.replaceState(null, '', `#${target}`)
      } catch {
        // ignore — jsdom/older browsers
      }
    }
    setQuery('')
  }

  return (
    <div className="settings-search">
      <input
        type="search"
        className="settings-search__input"
        placeholder="Search settings…"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        aria-label="Search settings"
      />
      {results.length > 0 && (
        <ul className="settings-search__results" role="listbox">
          {results.map((r) => (
            <li
              key={`${r.target}:${r.label}`}
              className="settings-search__result"
              role="option"
              aria-selected="false"
              data-target={r.target}
              tabIndex={0}
              onClick={() => jumpTo(r.target)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  jumpTo(r.target)
                }
              }}
            >
              {r.label}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
