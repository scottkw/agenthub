// Phase 147: In-App Help Page — Sticky left section nav with IntersectionObserver scroll-spy.
//
// Renders <nav> with a <button> per section; the active section button carries
// aria-current="true" and the --active modifier class. Clicking a button scrolls
// the section into view (smooth) and calls onSectionChange. An IntersectionObserver
// on the content pane fires onSectionChange as the user scrolls.

import React, { useEffect } from 'react'

export const SECTIONS = [
  { id: 'help-getting-started', label: 'Getting Started' },
  { id: 'help-chat', label: 'Chat' },
  // Phase 166-04 / HLP-01: must stay in sync with SECTION_META in HelpTab.tsx (Pitfall 5)
  { id: 'help-sharing', label: 'Sharing Outside Your Tailnet' },
  { id: 'help-faq', label: 'Frequently Asked Questions' },
] as const

interface HelpSectionNavProps {
  activeSection: string
  onSectionChange: (id: string) => void
  // React 19 changed useRef<T>(null) to return RefObject<T | null>
  contentPaneRef: React.RefObject<HTMLDivElement | null>
}

export function HelpSectionNav({
  activeSection,
  onSectionChange,
  contentPaneRef,
}: HelpSectionNavProps): React.ReactElement {
  // IntersectionObserver scroll-spy: tracks which section is visible in the
  // content pane and updates activeSection accordingly.
  useEffect(() => {
    // Guard: contentPaneRef may not be attached yet (SSR / test env)
    const root = contentPaneRef.current
    if (!root) return

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            onSectionChange(entry.target.id)
          }
        }
      },
      {
        root,
        // rootMargin: clear the 80px sticky search bar at the top,
        // and ignore everything in the bottom 60% of the viewport so
        // only the top-most visible section becomes active.
        rootMargin: '-80px 0px -60% 0px',
        threshold: 0,
      },
    )

    // Observe each section anchor element
    for (const section of SECTIONS) {
      const el = document.getElementById(section.id)
      if (el) observer.observe(el)
    }

    return () => {
      observer.disconnect()
    }
  }, [contentPaneRef, onSectionChange])

  return (
    <nav className="help-tab__nav" aria-label="Help sections">
      <ul className="help-nav__list">
        {SECTIONS.map(({ id, label }) => {
          const isActive = activeSection === id
          return (
            <li key={id} className="help-nav__item">
              <button
                type="button"
                className={`help-nav__link${isActive ? ' help-nav__link--active' : ''}`}
                aria-current={isActive ? 'true' : undefined}
                onClick={() => {
                  const el = document.getElementById(id)
                  if (el) {
                    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
                  }
                  onSectionChange(id)
                }}
              >
                {label}
              </button>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
