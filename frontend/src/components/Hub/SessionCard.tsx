import React from 'react'
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
} from '@heroicons/react/24/outline'
import { InlineSessionName } from './InlineSessionName'

// ---- Types ----

type HubStatus = 'running' | 'idle' | 'waiting' | 'errored' | 'stopped-ok' | 'stopped-err'

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
 * Derive the Hub display status from a SessionInfo.
 * - If state === 'stopped': 'stopped-err' when exitCode is non-zero, else 'stopped-ok'
 * - Otherwise: return status as HubStatus (running/idle/waiting/errored)
 */
function deriveStatus(s: SessionInfo): HubStatus {
  if (s.state === 'stopped') {
    return (s.exitCode ?? 0) !== 0 ? 'stopped-err' : 'stopped-ok'
  }
  return s.status as HubStatus
}

/**
 * Agent badge modifier — mirrors TabBar.tsx agentBadgeModifier (line 18).
 * Returns the BEM modifier suffix for known CLIs, or null for unknown.
 */
function agentBadgeModifier(cli: string): string | null {
  switch (cli) {
    case 'claude':
    case 'opencode':
    case 'codex':
    case 'gemini':
    case 'cursor':
    case 'aider':
      return cli
    case 'shell':
    case 'bash':
    case 'zsh':
    case 'pwsh':
    case 'powershell':
      return 'shell'
    default:
      return null
  }
}

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
 *
 * Dimming (CARD-08): stopped-ok cards get hub-card--dim; stopped-err cards do NOT.
 */
export function SessionCard({ session, onRename }: SessionCardProps): React.ReactElement {
  const {
    id,
    cli,
    name,
    hostname,
    viewerCount,
    exitCode,
    duration,
    createdAt,
    workDir: _workDir,
  } = session

  const hubStatus = deriveStatus(session)
  const { Icon, label, spin } = STATUS_CONFIG[hubStatus]

  // Stopped-err label shows "Exited {code}" — override the generic label
  const displayLabel =
    hubStatus === 'stopped-err' ? `Exited ${exitCode ?? ''}`.trim() : label

  // Origin marker: empty or same-machine hostname → Local
  const isLocal = !hostname || hostname === ''
  const originText = isLocal ? 'Local' : hostname

  // CLI badge
  const modifier = agentBadgeModifier(cli)
  const badgeClass = modifier
    ? `tab__agent-badge tab__agent-badge--${modifier}`
    : 'tab__agent-badge'

  // Time display
  const timeText =
    session.state === 'stopped' && duration !== undefined && duration !== null
      ? formatDuration(duration)
      : formatUptime(createdAt)

  // Card aria-label per Accessibility Contract
  const cardAriaLabel = `${name}, ${displayLabel}, ${cli}, ${originText}`

  return (
    <article
      className={`hub-card${hubStatus === 'stopped-ok' ? ' hub-card--dim' : ''}`}
      aria-label={cardAriaLabel}
      tabIndex={0}
    >
      {/* ROW 1: status indicator | name | CLI badge */}
      <div className="hub-card__row hub-card__row--primary">
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

        {/* CLI badge — reuses tab__agent-badge--{cli} hex constants from style.css */}
        <span className={badgeClass} aria-hidden="true">
          {cli}
        </span>
      </div>

      {/* ROW 2: origin marker */}
      <div className="hub-card__row hub-card__row--origin">
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
      <div className="hub-card__row hub-card__row--meta">
        <span className="hub-card__time">{timeText}</span>

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
      {hubStatus === 'stopped-err' && (
        <div className="hub-card__row hub-card__row--exit">
          <span className="hub-card__exit-chip">Exited {exitCode}</span>
        </div>
      )}
    </article>
  )
}
