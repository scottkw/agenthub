import React from 'react'

/**
 * ShareSegmentedControl — the three-way access-tier tablist for the Hub Share
 * modal (Phase 173 / SM-03, SM-07). Net-new UI infrastructure: this app has no
 * prior role="tablist" precedent, so the roving-tabindex + arrow-key contract
 * below is implemented per the WAI-ARIA Authoring Practices Guide tablist
 * pattern (173-RESEARCH.md "Pattern 1"), not a UI library.
 *
 * Presentational only. It emits `onSelect(id)` for enabled segments — the
 * shell (SessionShareModal, plan 06) owns the active-tab state and decides
 * which tabs are disabled (tab-availability gating for Internet-* tabs is
 * enforced there, not here; see 173-05-PLAN.md threat model).
 *
 * Colorblind-safe danger cue (SM-07, owner is colorblind — verify at source,
 * never by eye): the Full-access tab is distinguished by the `is-danger`
 * class (drives the inset ring in style.css) AND a "⚠" glyph the component
 * itself prefixes onto the sub label whenever `danger` is true — belt and
 * suspenders with the CSS ring, independent of what the caller passes in
 * `sub`.
 */

/** Stable tab id union — exported so the shell (plan 06) can reuse it. */
export type ShareTab = 'tailnet' | 'internet-ro' | 'internet-fa'

export interface ShareSegmentedControlTab {
  id: ShareTab
  /** Primary label, e.g. "Tailnet" / "Internet". */
  main: string
  /** Secondary label, e.g. "Private" / "Read-only" / "Full access". */
  sub: string
  /** Gated (not yet available) — renders aria-disabled/disabled + 'N/A' sub text. */
  disabled?: boolean
  /** Marks the destructive/danger tier (Full-access). */
  danger?: boolean
}

export interface ShareSegmentedControlProps {
  tabs: ShareSegmentedControlTab[]
  /** Currently-active tab id. */
  active: ShareTab
  /** Fired only for enabled segments (click or arrow-key navigation). */
  onSelect: (id: ShareTab) => void
}

function classNames(...parts: Array<string | false | undefined>): string {
  return parts.filter(Boolean).join(' ')
}

export function ShareSegmentedControl({
  tabs,
  active,
  onSelect,
}: ShareSegmentedControlProps): React.ReactElement {
  const enabledTabs = tabs.filter((t) => !t.disabled)

  function moveSelection(delta: 1 | -1): void {
    if (enabledTabs.length === 0) return
    const currentIdx = enabledTabs.findIndex((t) => t.id === active)
    const fromIdx = currentIdx === -1 ? 0 : currentIdx
    const nextIdx = (fromIdx + delta + enabledTabs.length) % enabledTabs.length
    const next = enabledTabs[nextIdx]
    if (next) onSelect(next.id)
  }

  return (
    <div className="share-segbar" role="tablist" aria-label="Share access tier">
      {tabs.map((t) => {
        const isActive = active === t.id
        const subText = t.disabled ? 'N/A' : t.danger ? `⚠ ${t.sub}` : t.sub
        return (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={isActive}
            aria-disabled={t.disabled}
            disabled={t.disabled}
            tabIndex={isActive ? 0 : -1}
            className={classNames(
              'share-seg',
              isActive && 'is-active',
              t.danger && 'is-danger',
            )}
            onClick={() => {
              if (!t.disabled) onSelect(t.id)
            }}
            onKeyDown={(e) => {
              if (e.key === 'ArrowRight') {
                e.preventDefault()
                moveSelection(1)
              } else if (e.key === 'ArrowLeft') {
                e.preventDefault()
                moveSelection(-1)
              }
            }}
          >
            <span className="share-seg__main">{t.main}</span>
            <span className="share-seg__sub">{subText}</span>
          </button>
        )
      })}
    </div>
  )
}
