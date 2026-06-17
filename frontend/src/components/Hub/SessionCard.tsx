import React, { useState, useEffect, useRef, useCallback } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import {
  ArrowPathIcon,
  CheckCircleIcon,
  PauseCircleIcon,
  ExclamationCircleIcon,
  StopCircleIcon,
  ComputerDesktopIcon,
  GlobeAltIcon,
  EyeIcon,
  Bars3Icon,
  EllipsisHorizontalIcon,
  BellAlertIcon,          // ATTN-01: attention icon — colorblind-safe shape carrier
} from '@heroicons/react/24/outline'
import { InlineSessionName } from './InlineSessionName'
// WR-01: deriveHubStatus extracted to shared util (was triplicated across SessionCard/HubFilterBar/HubPanel)
import { deriveHubStatus } from '../../lib/hubStatus'
import type { HubStatus } from '../../lib/hubStatus'
import { memberKey, type HubGroupDef } from '../../lib/hubGroups'
import { MiniPreview } from './MiniPreview'

// ---- STATUS_CONFIG ----
// COLORBLIND-SAFE: every status has unique icon shape + text label; color is reinforcement only.
// Hex values are authoritative source of truth — verify at source, not by eye (user is colorblind).
const STATUS_CONFIG: Record<
  HubStatus,
  { Icon: React.ComponentType<React.SVGProps<SVGSVGElement>>; label: string; spin: boolean }
> = {
  /* COLORBLIND-SAFE: status dot dark hex #3b82f6 (running) — reinforcement only; ArrowPathIcon carries the state */
  running: { Icon: ArrowPathIcon, label: 'Running', spin: true },
  /* COLORBLIND-SAFE: status dot dark hex #22c55e (idle) — reinforcement only; CheckCircleIcon carries the state */
  idle: { Icon: CheckCircleIcon, label: 'Idle', spin: false },
  /* COLORBLIND-SAFE: status dot dark hex #f59e0b (waiting) — reinforcement only; PauseCircleIcon carries the state */
  waiting: { Icon: PauseCircleIcon, label: 'Needs input', spin: false },
  /* COLORBLIND-SAFE: status dot dark hex #ef4444 (errored) — reinforcement only; ExclamationCircleIcon carries the state */
  errored: { Icon: ExclamationCircleIcon, label: 'Error', spin: false },
  /* COLORBLIND-SAFE: status dot dark hex #565f89 (stopped/done) — reinforcement only; StopCircleIcon carries the state */
  'stopped-ok': { Icon: StopCircleIcon, label: 'Done', spin: false },
  /* COLORBLIND-SAFE: exit-code dark hex #f7768e (non-zero exit) — reinforcement only; "Exited {code}" text carries the state */
  /* COLORBLIND-SAFE: status dot dark hex #f7768e (non-zero exit) — reinforcement only; ExclamationCircleIcon carries the state */
  /* COLORBLIND-SAFE: status dot light hex #1d4ed8 (running) — WCAG AA 7.1:1 on white; icon carries state */
  /* COLORBLIND-SAFE: status dot light hex #1a7f37 (idle) — WCAG AA 5.0:1 on white; icon carries state */
  /* COLORBLIND-SAFE: status dot light hex #92400e (waiting) — WCAG AA 7.6:1 on white; icon carries state */
  /* COLORBLIND-SAFE: status dot light hex #b91c1c (errored) — WCAG AA 6.0:1 on white; icon carries state */
  /* COLORBLIND-SAFE: status dot light hex #4b5563 (stopped/done) — WCAG AA 7.0:1 on white; icon carries state */
  /* COLORBLIND-SAFE: status dot light hex #c0394f (non-zero exit) — WCAG AA 4.7:1 on white; icon carries state */
  /* HUB-04 LIGHT THEME: verified WCAG AA for --hub-accent #3d6fe8 on #ffffff (4.5:1) */
  /* HUB-04 LIGHT THEME: verified WCAG AA for --hub-destructive #c0394f on #ffffff (4.7:1) */
  'stopped-err': { Icon: ExclamationCircleIcon, label: 'Exited', spin: false },
}

// ---- Helpers ----

/**
 * Format seconds into "Xh Ym" uptime string (for running sessions).
 * UI-SPEC Copywriting Contract: "2h 14m"
 */
function formatUptime(createdAt: string): string {
  const startMs = new Date(createdAt).getTime()
  const elapsedSec = Math.max(0, Math.floor((Date.now() - startMs) / 1000))
  return formatHM(elapsedSec)
}

/**
 * Format seconds into "Ran Xh Ym" string (for stopped sessions).
 * UI-SPEC Copywriting Contract: "Ran 2h 14m"
 */
function formatDuration(seconds: number): string {
  return `Ran ${formatHM(seconds)}`
}

function formatHM(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m`
}

// ---- Props ----

export interface SessionCardProps {
  session: SessionInfo
  onRename?: (id: string, name: string) => void
  /**
   * Opens (or focuses) this session's terminal tab. Phase 131 UAT follow-up —
   * re-attach to a running session whose terminal tab is not open in this window.
   * Surfaced as an explicit "Open" button (not whole-card click — card-click is
   * reserved for the Phase 134 modal gesture). Only shown for live sessions.
   */
  onOpenSession?: (sessionId: string, name: string, cli: string) => void
  /** CARD-07: tail lines from usePreviewPoller; undefined = loading */
  previewLines?: string[]
  /** GROUP-02: group definitions for the "Move to group" overflow menu */
  groupDefs?: HubGroupDef[]
  /** GROUP-02: fires when user assigns this card to a group via menu */
  onAssignGroup?: (memberKey: string, groupId: string) => void
  /** ATTN-01: true when isAttentionStatus(deriveHubStatus(session)) is true */
  isAttention?: boolean
  /**
   * Phase 134 — fires when card body is clicked (not Open/menu/drag-handle/rename input).
   * Receives the session and the card's bounding rect (used for grow animation origin).
   */
  onCardClick?: (session: SessionInfo, rect: DOMRect) => void
}

// ---- Component ----

/**
 * SessionCard — presentational card for the Hub session grid.
 *
 * Colorblind-safe: every status renders icon + text label.
 * Color is reinforcement only (see STATUS_CONFIG hex comments above).
 *
 * Layout:
 *   ROW 1: status-indicator | InlineSessionName | CLI badge
 *   ROW 2: origin marker (Local / peer hostname)
 *   ROW 3: uptime/duration + viewer count (only when >0)
 *   ROW 4: exit-code chip (only when stopped with non-zero exit)
 *   ROW 5: Open button (when live, Phase 131)
 *   ROW 6: MiniPreview (CARD-07 plain-text snapshot; NO xterm)
 *
 * Dimming (CARD-08): stopped-ok cards get hub-card--dim; stopped-err cards do NOT.
 */
export function SessionCard({
  session,
  onRename,
  onOpenSession,
  previewLines,
  groupDefs,
  onAssignGroup,
  isAttention,
  onCardClick,
}: SessionCardProps): React.ReactElement {
  const {
    id,
    cli,
    name,
    hostname,
    viewerCount,
    exitCode,
    duration,
    createdAt,
  } = session

  const hubStatus = deriveHubStatus(session)
  const { Icon, label, spin } = STATUS_CONFIG[hubStatus]

  // Stopped-err label shows "Exited {code}" — override the generic label
  const displayLabel =
    hubStatus === 'stopped-err' ? `Exited ${exitCode ?? ''}`.trim() : label

  // Origin marker: empty or same-machine hostname → Local
  const isLocal = !hostname || hostname === ''
  const originText = isLocal ? 'Local' : hostname

  // Time display
  // IN-03: remote sessions use createdAt = new Date().toISOString() (set at poll time),
  // so formatUptime would show "0m" for ~29s then reset on the next 30s remote poll.
  // Omit uptime for remote sessions entirely — there is no reliable createdAt from the wire.
  const timeText =
    hostname && hostname !== ''
      ? '' // remote session — no reliable createdAt; omit rather than show misleading "0m"
      : session.state === 'stopped' && duration !== undefined && duration !== null
      ? formatDuration(duration)
      : formatUptime(createdAt)

  // Card aria-label per Accessibility Contract
  // ATTN-01: append ", needs attention" suffix when attention is active (Accessibility Contract item 3)
  const cardAriaLabel = `${name}, ${displayLabel}, ${cli}, ${originText}${isAttention ? ', needs attention' : ''}`

  /* GROUP-04: membership key = "${session.name}:::${session.workDir}" — survives session-id churn */
  const memberKeyForSession = memberKey(name, session.workDir)

  // Drag state
  const [isDragging, setIsDragging] = useState(false)

  // Menu state
  const [menuOpen, setMenuOpen] = useState(false)
  const menuBtnRef = useRef<HTMLButtonElement>(null)

  // Determine if session is currently in any named group
  const isInNamedGroup = (groupDefs ?? []).some((g) =>
    g.memberKeys.includes(memberKeyForSession)
  )

  // Close menu on Escape key (global listener)
  useEffect(() => {
    if (!menuOpen) return
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        setMenuOpen(false)
        menuBtnRef.current?.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [menuOpen])

  // Close menu on click-outside
  const menuRef = useRef<HTMLDivElement>(null)
  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (
      menuRef.current &&
      !menuRef.current.contains(e.target as Node) &&
      !menuBtnRef.current?.contains(e.target as Node)
    ) {
      setMenuOpen(false)
    }
  }, [])

  useEffect(() => {
    if (!menuOpen) return
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [menuOpen, handleClickOutside])

  function handleAssign(groupId: string) {
    onAssignGroup?.(memberKeyForSession, groupId)
    setMenuOpen(false)
  }

  return (
    <article
      className={[
        'hub-card',
        hubStatus === 'stopped-ok' ? 'hub-card--dim' : '',
        isDragging ? 'hub-card--dragging' : '',
        isAttention ? 'hub-card--attention' : '',
      ].filter(Boolean).join(' ')}
      draggable="true"
      onDragStart={(e) => {
        e.dataTransfer.setData('text/plain', memberKeyForSession)
        e.dataTransfer.effectAllowed = 'move'
        setIsDragging(true)
      }}
      onDragEnd={() => setIsDragging(false)}
      onClick={(e) => {
        // Defense-in-depth: verify click did not originate from controlled children
        const target = e.target as HTMLElement
        if (target.closest('.hub-card__open')) return
        if (target.closest('.hub-card__menu-btn')) return
        if (target.closest('.hub-card__menu')) return
        if (target.closest('.InlineSessionName input')) return
        if (isDragging) return
        onCardClick?.(session, (e.currentTarget as HTMLElement).getBoundingClientRect())
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onCardClick?.(session, (e.currentTarget as HTMLElement).getBoundingClientRect())
        }
      }}
      aria-label={cardAriaLabel}
      tabIndex={0}
    >
      {/* Drag handle — visible on hover (CSS opacity transition); aria-hidden for screen readers */}
      <span
        className="hub-card__drag-handle"
        aria-label="Drag to reorder"
        aria-hidden="true"
      >
        <Bars3Icon className="w-4 h-4" />
      </span>

      {/* Overflow menu button — visible on hover; keyboard-reachable via focus-within */}
      <button
        ref={menuBtnRef}
        type="button"
        className="hub-card__menu-btn"
        aria-label={`Card options for ${name}`}
        aria-expanded={menuOpen}
        aria-haspopup="menu"
        onClick={(e) => { e.stopPropagation(); setMenuOpen((p) => !p) }}
      >
        <EllipsisHorizontalIcon className="w-4 h-4" />
      </button>

      {/* Group overflow menu */}
      {menuOpen && (
        <div ref={menuRef} className="hub-card__menu" role="menu">
          <div className="hub-card__menu-item hub-card__menu-item--header" role="presentation">
            Move to group
          </div>
          {(groupDefs ?? []).map((g) => (
            <button
              key={g.id}
              type="button"
              className="hub-card__menu-item hub-card__menu-item--group"
              role="menuitem"
              onClick={() => handleAssign(g.id)}
            >
              {g.name}
            </button>
          ))}
          {/* WR-01: hide "Other (default)" when the session is already ungrouped.
              Showing it to an ungrouped session is a no-op and a wasted saveGroups write.
              When the session IS in a named group this item moves it back to "Other",
              making the separate "Remove from group" section redundant — removed. */}
          {isInNamedGroup && (
            <button
              type="button"
              className="hub-card__menu-item hub-card__menu-item--group"
              role="menuitem"
              onClick={() => handleAssign('__other__')}
            >
              Other (default)
            </button>
          )}
        </div>
      )}

      {/* ROW 1: status indicator | name | CLI badge */}
      {/* CR-01: hub-card__row1 matches CSS definition (was hub-card__row hub-card__row--primary) */}
      <div className="hub-card__row1">
        {/* ATTN-01: attention icon — inline left of status icon; COLORBLIND-SAFE: BellAlertIcon carries state */}
        {isAttention && (
          <span className="hub-card__attn-icon" aria-label="Needs attention">
            {/* CRITICAL: NO Tailwind w-4 h-4 — size via .hub-card__attn-icon svg in style.css (Plan 02) */}
            <BellAlertIcon aria-hidden="true" />
          </span>
        )}
        <span className="hub-card__status-indicator">
          <Icon
            className={`hub-card__status-icon${spin ? ' hub-card__status-icon--spin' : ''}`}
            aria-label={displayLabel}
          />
          <span className="hub-card__status-label">{displayLabel}</span>
        </span>

        <InlineSessionName
          id={id}
          name={name}
          onRenamed={(newName) => onRename?.(id, newName)}
        />

        {/* CLI badge — hub-card__badge text-chip pattern (WR-03: replaces tab__agent-badge dot) */}
        <span className="hub-card__badge">
          {cli}
        </span>
      </div>

      {/* ROW 2: origin marker */}
      {/* CR-01: hub-card__row2 matches CSS definition (was hub-card__row hub-card__row--origin) */}
      <div className="hub-card__row2">
        <span className="hub-card__origin">
          {isLocal ? (
            <>
              <ComputerDesktopIcon className="hub-card__origin-icon" aria-hidden="true" />
              <span>Local</span>
            </>
          ) : (
            <>
              <GlobeAltIcon className="hub-card__origin-icon" aria-hidden="true" />
              <span>{hostname}</span>
            </>
          )}
        </span>
      </div>

      {/* ROW 3: uptime/duration + viewer count */}
      {/* CR-01: hub-card__row3 matches CSS definition (was hub-card__row hub-card__row--meta) */}
      <div className="hub-card__row3">
        {/* CR-01: hub-card__uptime matches CSS definition (was hub-card__time) */}
        <span className="hub-card__uptime">{timeText}</span>

        {viewerCount > 0 && (
          <span className="hub-card__viewers">
            <EyeIcon className="hub-card__viewers-icon" aria-hidden="true" />
            <span>
              {viewerCount} {viewerCount === 1 ? 'viewer' : 'viewers'}
            </span>
          </span>
        )}
      </div>

      {/* ROW 4: exit-code chip (only for non-zero exit) */}
      {/* CR-01: hub-card__row4 matches CSS definition (was hub-card__row hub-card__row--exit) */}
      {hubStatus === 'stopped-err' && (
        <div className="hub-card__row4">
          {/* IN-02: aria-hidden since exit code is already in card aria-label */}
          <span className="hub-card__exit-chip" aria-hidden="true">Exited {exitCode}</span>
        </div>
      )}

      {/* ROW 5: actions — Open re-attaches the session's terminal tab.
          Phase 131 UAT follow-up. Only for live sessions (a stopped session has
          no PTY to attach to). Text label (not color) keeps it colorblind-safe. */}
      {onOpenSession && session.state !== 'stopped' && (
        <div className="hub-card__row5">
          <button
            type="button"
            className="hub-card__open"
            onClick={(e) => { e.stopPropagation(); onOpenSession(id, name, cli) }}
            aria-label={`Open ${name}`}
          >
            Open
          </button>
        </div>
      )}

      {/* ROW 6: MiniPreview — CARD-07: plain text snapshot; NO xterm instance; polling interval 3s shared across all cards */}
      <MiniPreview lines={previewLines} />
    </article>
  )
}
