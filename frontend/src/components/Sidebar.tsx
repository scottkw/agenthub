import React, { useState, useCallback } from 'react'
import {
  Bars3Icon,
  HomeIcon,
  Cog6ToothIcon,
  Squares2X2Icon,   // Phase 131 / HUB-01
  QuestionMarkCircleIcon,  // Phase 147 / HELP-01
} from '@heroicons/react/24/outline'
import type { HubGroupDef } from '../lib/hubGroups'

const STORAGE_KEY = 'sidebar-collapsed'

// Phase 138 / NAV-02..05: sidebar collapsed to Home, Hub, Settings only.
// onOpenRemoteSessions, onOpenDaemonManager, onAdd removed — those panels are deleted.
// POL-05: group sub-list props added to main sidebar.
interface SidebarProps {
  onHome: () => void
  onSettings: () => void
  onOpenHub: () => void           // Phase 131 / HUB-01
  activePanel?: string            // Phase 131 / Pitfall-8: active state indicator
  // POL-05 additions — group sub-list nested under Hub item
  groupDefs: HubGroupDef[]
  activeGroupId: string | null
  onGroupSelect: (id: string | null) => void
  onCreateGroup: (name: string) => void
  onDropOnGroup: (groupId: string, mKey: string) => void
  groupCounts: Record<string, { running: number; total: number; attention: number; waiting: number }>
  globalGroupCounts: { running: number; total: number; attention: number; waiting: number }
  onOpenHelp: () => void  // Phase 147 / HELP-01
}

// ---- GroupItem — single drag-drop item in the sub-list ----
// CARRY-01: <li> owns visual classes + drag handlers; inner <button> owns interactive ARIA.

interface GroupItemProps {
  id: string | null  // null = "All"
  label: string
  counts: { running: number; total: number; attention: number; waiting: number }
  isActive: boolean
  onGroupSelect: (id: string | null) => void
  onOpenHub: () => void
  onDropOnGroup: (groupId: string, mKey: string) => void
}

function GroupItem({
  id,
  label,
  counts,
  isActive,
  onGroupSelect,
  onOpenHub,
  onDropOnGroup,
}: GroupItemProps): React.ReactElement {
  const [isDragOver, setIsDragOver] = useState(false)

  // CARRY-01: drag handlers on <li> so the full item area is a drop target
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
    'sidebar__group-item',
    isActive ? 'sidebar__group-item--active' : '',
    isDragOver ? 'sidebar__group-item--drag-over' : '',
  ].filter(Boolean).join(' ')

  return (
    // CARRY-01: <li> retains visual classes + drag handlers; no role/aria-selected/tabIndex
    <li
      className={itemClass}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      {/* CARRY-01: native button handles Enter/Space; aria-pressed conveys toggle state */}
      {/* Count in aria-label only (accessible); textContent = label for test findability */}
      <button
        type="button"
        className="sidebar__group-item__btn"
        aria-pressed={isActive}
        aria-label={`${label} group, ${counts.running}/${counts.total} sessions`}
        onClick={() => { onGroupSelect(id); onOpenHub() }}
        onKeyDown={(e) => {
          if (e.key === ' ') {
            e.preventDefault()
            onGroupSelect(id)
            onOpenHub()
          }
        }}
      >
        {label}
      </button>
    </li>
  )
}

// ---- Sidebar ----

export function Sidebar({
  onHome,
  onSettings,
  onOpenHub,
  activePanel,
  groupDefs,
  activeGroupId,
  onGroupSelect,
  onCreateGroup,
  onDropOnGroup,
  groupCounts,
  globalGroupCounts,
  onOpenHelp,
}: SidebarProps): React.ReactElement {
  const [collapsed, setCollapsed] = useState<boolean>(
    () => localStorage.getItem(STORAGE_KEY) === 'true'
  )

  // POL-05: inline group creation state
  const [creating, setCreating] = useState(false)
  const [inputValue, setInputValue] = useState('')

  const toggle = () => {
    setCollapsed((prev) => {
      const next = !prev
      localStorage.setItem(STORAGE_KEY, String(next))
      return next
    })
  }

  // POL-05: drag auto-expand — when a drag enters sidebar while collapsed, temporarily
  // show the group list for the drag duration. We track drag-over count to handle
  // nested drag-enter/leave events, restoring collapse state on drag-end.
  const [dragExpandActive, setDragExpandActive] = useState(false)
  const dragCountRef = React.useRef(0)

  const handleNavDragEnter = useCallback((e: React.DragEvent<HTMLElement>) => {
    if (!collapsed) return
    e.preventDefault()
    dragCountRef.current++
    if (dragCountRef.current === 1) setDragExpandActive(true)
  }, [collapsed])

  const handleNavDragLeave = useCallback(() => {
    if (!collapsed) return
    dragCountRef.current--
    if (dragCountRef.current <= 0) {
      dragCountRef.current = 0
      setDragExpandActive(false)
    }
  }, [collapsed])

  const handleNavDrop = useCallback(() => {
    dragCountRef.current = 0
    setDragExpandActive(false)
  }, [])

  // Effective expanded state: collapsed but drag-expanding shows the group list
  const effectiveExpanded = !collapsed || dragExpandActive

  // POL-05: inline group creation handlers (copied from GroupSidebar pattern)
  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      const trimmed = inputValue.trim()
      if (trimmed) {
        onCreateGroup(trimmed)
        setCreating(false)
        setInputValue('')
      }
      // Empty Enter: no-op (input stays open)
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

  const showGroupList = effectiveExpanded && groupDefs.length > 0

  return (
    <nav
      className={`sidebar${collapsed ? ' sidebar--collapsed' : ''}`}
      aria-label="Main navigation"
      onDragEnter={handleNavDragEnter}
      onDragLeave={handleNavDragLeave}
      onDrop={handleNavDrop}
    >
      <button
        className="sidebar__toggle"
        onClick={toggle}
        aria-label="Toggle sidebar"
      >
        <Bars3Icon className="sidebar__icon" />
      </button>

      <button
        className="sidebar__item"
        onClick={onHome}
        aria-label="Home"
      >
        <HomeIcon className="sidebar__icon" />
        {!collapsed && <span className="sidebar__label">Home</span>}
      </button>

      <button
        className={`sidebar__item${activePanel === '__hub__' ? ' sidebar__item--active' : ''}`}
        onClick={onOpenHub}
        aria-label="Hub"
      >
        <Squares2X2Icon className="sidebar__icon" />
        {!collapsed && <span className="sidebar__label">Hub</span>}
      </button>

      {/* POL-05: group sub-list — nested under Hub, shown when expanded + groups exist */}
      {showGroupList && (
        <ul className="sidebar__group-list" aria-label="Hub groups">
          {/* "All" item — always first */}
          <GroupItem
            id={null}
            label="All"
            counts={globalGroupCounts}
            isActive={activeGroupId === null}
            onGroupSelect={onGroupSelect}
            onOpenHub={onOpenHub}
            onDropOnGroup={onDropOnGroup}
          />
          {/* Named group items */}
          {groupDefs.map((g) => (
            <GroupItem
              key={g.id}
              id={g.id}
              label={g.name}
              counts={groupCounts[g.id] ?? { running: 0, total: 0, attention: 0, waiting: 0 }}
              isActive={activeGroupId === g.id}
              onGroupSelect={onGroupSelect}
              onOpenHub={onOpenHub}
              onDropOnGroup={onDropOnGroup}
            />
          ))}
        </ul>
      )}

      {/* POL-05: inline group creation — shown when expanded + groups exist */}
      {effectiveExpanded && groupDefs.length > 0 && (
        creating ? (
          <input
            className="sidebar__group-new-input"
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
            className="sidebar__group-new"
            onClick={() => { setCreating(true); setInputValue('') }}
          >
            New group
          </button>
        )
      )}

      <div className="sidebar__bottom">
        <button
          className={`sidebar__item${activePanel === '__help__' ? ' sidebar__item--active' : ''}`}
          onClick={onOpenHelp}
          aria-label="Help"
        >
          <QuestionMarkCircleIcon className="sidebar__icon" />
          {!collapsed && <span className="sidebar__label">Help</span>}
        </button>
        <button
          className="sidebar__item"
          onClick={onSettings}
          aria-label="Settings"
        >
          <Cog6ToothIcon className="sidebar__icon" />
          {!collapsed && <span className="sidebar__label">Settings</span>}
        </button>
      </div>
    </nav>
  )
}
