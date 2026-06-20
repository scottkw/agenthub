import React, { useState } from 'react'
import {
  Bars3Icon,
  HomeIcon,
  Cog6ToothIcon,
  Squares2X2Icon,   // Phase 131 / HUB-01
} from '@heroicons/react/24/outline'

const STORAGE_KEY = 'sidebar-collapsed'

// Phase 138 / NAV-02..05: sidebar collapsed to Home, Hub, Settings only.
// onOpenRemoteSessions, onOpenDaemonManager, onAdd removed — those panels are deleted.
interface SidebarProps {
  onHome: () => void
  onSettings: () => void
  onOpenHub: () => void           // Phase 131 / HUB-01
  activePanel?: string            // Phase 131 / Pitfall-8: active state indicator
}

export function Sidebar({
  onHome,
  onSettings,
  onOpenHub,
  activePanel,
}: SidebarProps): React.ReactElement {
  const [collapsed, setCollapsed] = useState<boolean>(
    () => localStorage.getItem(STORAGE_KEY) === 'true'
  )

  const toggle = () => {
    setCollapsed((prev) => {
      const next = !prev
      localStorage.setItem(STORAGE_KEY, String(next))
      return next
    })
  }

  return (
    <nav
      className={`sidebar${collapsed ? ' sidebar--collapsed' : ''}`}
      aria-label="Main navigation"
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

      <div className="sidebar__bottom">
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
