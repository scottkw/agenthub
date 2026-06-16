/* GRID-03: group sidebar — collapsible group list with running/total counts and needs-input badge */
/* COLORBLIND-SAFE: needs-input badge dark hex #f59e0b — reinforcement only; PauseCircleIcon carries the state */
/* COLORBLIND-SAFE: needs-input badge light hex #b45309 — WCAG AA 7.6:1 on white; icon carries state */
import React, { useState, useCallback } from 'react'
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  PauseCircleIcon,
} from '@heroicons/react/24/outline'
import type { HubGroupDef } from '../../lib/hubGroups'
import { memberKey } from '../../lib/hubGroups'
import { deriveHubStatus } from '../../lib/hubStatus'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// ---- Counts ----

interface GroupCounts {
  running: number
  total: number
  waiting: number
}

function computeCounts(sessions: SessionInfo[], memberKeys: Set<string>): GroupCounts {
  let running = 0
  let total = 0
  let waiting = 0
  for (const s of sessions) {
    const key = memberKey(s.name, s.workDir)
    if (!memberKeys.has(key)) continue
    total++
    const st = deriveHubStatus(s)
    if (st === 'running' || st === 'idle' || st === 'waiting') running++
    if (st === 'waiting') waiting++
  }
  return { running, total, waiting }
}

function computeGlobalCounts(sessions: SessionInfo[]): GroupCounts {
  let running = 0
  let waiting = 0
  for (const s of sessions) {
    const st = deriveHubStatus(s)
    if (st === 'running' || st === 'idle' || st === 'waiting') running++
    if (st === 'waiting') waiting++
  }
  return { running, total: sessions.length, waiting }
}

// ---- NeedsInputBadge ----

interface NeedsInputBadgeProps {
  count: number
}

function NeedsInputBadge({ count }: NeedsInputBadgeProps): React.ReactElement | null {
  if (count === 0) return null
  const label = count === 1 ? '1 session needs input' : `${count} sessions need input`
  return (
    <span
      className="hub__group-sidebar-item__needs-input-badge"
      aria-label={label}
    >
      {/* COLORBLIND-SAFE: needs-input badge dark hex #f59e0b — reinforcement only; PauseCircleIcon carries the state */}
      <PauseCircleIcon className="w-3 h-3" aria-hidden="true" />
      <span>{count}</span>
    </span>
  )
}

// ---- GroupSidebarItem ----

export interface GroupSidebarItemProps {
  id: string | null  // null = "All"
  label: string
  counts: GroupCounts
  isActive: boolean
  collapsed: boolean
  onGroupSelect: (id: string | null) => void
  onDropOnGroup: (groupId: string, mKey: string) => void
}

export function GroupSidebarItem({
  id,
  label,
  counts,
  isActive,
  collapsed,
  onGroupSelect,
  onDropOnGroup,
}: GroupSidebarItemProps): React.ReactElement {
  const [isDragOver, setIsDragOver] = useState(false)

  const handleDragOver = useCallback((e: React.DragEvent<HTMLLIElement>) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(true)
  }, [])

  const handleDragLeave = useCallback(() => {
    setIsDragOver(false)
  }, [])

  const handleDrop = useCallback((e: React.DragEvent<HTMLLIElement>) => {
    e.preventDefault()
    setIsDragOver(false)
    if (id === null) return  // Cannot drop on "All"
    const key = e.dataTransfer.getData('text/plain')
    if (key) {
      onDropOnGroup(id, key)
    }
  }, [id, onDropOnGroup])

  const itemClass = [
    'hub__group-sidebar-item',
    isActive ? 'hub__group-sidebar-item--active' : '',
    isDragOver ? 'hub__group-sidebar-item--drag-over' : '',
  ].filter(Boolean).join(' ')

  return (
    <li
      className={itemClass}
      role="option"
      aria-selected={isActive ? 'true' : 'false'}
      onClick={() => onGroupSelect(id)}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      {!collapsed && (
        <span className="hub__group-sidebar-item__name">{label}</span>
      )}
      {!collapsed && (
        <span className="hub__group-sidebar-item__count">
          {counts.running}/{counts.total}
        </span>
      )}
      {!collapsed && counts.waiting > 0 && (
        <NeedsInputBadge count={counts.waiting} />
      )}
    </li>
  )
}

// ---- GroupSidebar ----

export interface GroupSidebarProps {
  groupDefs: HubGroupDef[]
  sessions: SessionInfo[]
  activeGroupId: string | null
  collapsed: boolean
  onToggle: () => void
  onGroupSelect: (id: string | null) => void
  onCreateGroup: (name: string) => void
  onDropOnGroup: (groupId: string, mKey: string) => void
}

const SIDEBAR_LIST_ID = 'hub-group-sidebar-list'

export function GroupSidebar({
  groupDefs,
  sessions,
  activeGroupId,
  collapsed,
  onToggle,
  onGroupSelect,
  onCreateGroup,
  onDropOnGroup,
}: GroupSidebarProps): React.ReactElement {
  const [creating, setCreating] = useState(false)
  const [inputValue, setInputValue] = useState('')

  const globalCounts = computeGlobalCounts(sessions)

  const handleNewGroupClick = () => {
    setCreating(true)
    setInputValue('')
  }

  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      const trimmed = inputValue.trim()
      if (trimmed) {
        onCreateGroup(trimmed)
        setCreating(false)
        setInputValue('')
      }
      // Empty/whitespace Enter: no-op (input stays open)
    } else if (e.key === 'Escape') {
      setCreating(false)
      setInputValue('')
    }
  }

  const handleInputBlur = () => {
    const trimmed = inputValue.trim()
    if (trimmed) {
      onCreateGroup(trimmed)
    }
    setCreating(false)
    setInputValue('')
  }

  const isAllActive = activeGroupId === null

  return (
    <aside
      className={`hub__group-sidebar${collapsed ? ' hub__group-sidebar--collapsed' : ''}`}
    >
      {/* Toggle button */}
      <button
        type="button"
        className="hub__group-sidebar-toggle"
        onClick={onToggle}
        aria-label={collapsed ? 'Expand group sidebar' : 'Collapse group sidebar'}
        aria-expanded={!collapsed}
        aria-controls={SIDEBAR_LIST_ID}
      >
        {collapsed ? (
          <ChevronRightIcon className="w-4 h-4" aria-hidden="true" />
        ) : (
          <ChevronLeftIcon className="w-4 h-4" aria-hidden="true" />
        )}
      </button>

      {/* Heading — only when expanded */}
      {!collapsed && (
        <span className="hub__group-sidebar-heading">Groups</span>
      )}

      {/* Group list */}
      <ul
        id={SIDEBAR_LIST_ID}
        className="hub__group-sidebar-list"
        role="listbox"
      >
        {/* "All" item */}
        <GroupSidebarItem
          id={null}
          label="All"
          counts={globalCounts}
          isActive={isAllActive}
          collapsed={collapsed}
          onGroupSelect={onGroupSelect}
          onDropOnGroup={onDropOnGroup}
        />

        {/* Named group items */}
        {groupDefs.map((g) => {
          const memberKeySet = new Set(g.memberKeys)
          const counts = computeCounts(sessions, memberKeySet)
          return (
            <GroupSidebarItem
              key={g.id}
              id={g.id}
              label={g.name}
              counts={counts}
              isActive={activeGroupId === g.id}
              collapsed={collapsed}
              onGroupSelect={onGroupSelect}
              onDropOnGroup={onDropOnGroup}
            />
          )
        })}
      </ul>

      {/* New group button or inline input */}
      {!collapsed && (
        creating ? (
          <input
            className="hub__group-sidebar-new-input"
            type="text"
            placeholder="Group name…"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleInputKeyDown}
            onBlur={handleInputBlur}
            autoFocus
          />
        ) : (
          <button
            type="button"
            className="hub__group-sidebar-new"
            onClick={handleNewGroupClick}
          >
            New group
          </button>
        )
      )}
    </aside>
  )
}
