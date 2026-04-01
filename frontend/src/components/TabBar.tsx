import React, { useState, useRef, useEffect } from 'react'

export interface Tab {
  id: string
  name: string
  sessionId: string
  cli: string
  type?: 'terminal' | 'welcome'
}

interface TabBarProps {
  tabs: Tab[]
  activeId: string | null
  onSelect: (id: string) => void
  onClose: (id: string) => void
  onRename: (id: string, name: string) => void
  onAdd: () => void
  onSettings: () => void
  sessionStatuses?: Record<string, string>
}

/**
 * Horizontal tab bar with add/rename/close controls.
 * Double-clicking or right-clicking a tab name enables inline editing.
 */
export function TabBar({
  tabs,
  activeId,
  onSelect,
  onClose,
  onRename,
  onAdd,
  onSettings,
  sessionStatuses,
}: TabBarProps): React.ReactElement {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const [contextMenu, setContextMenu] = useState<{ tabId: string; x: number; y: number } | null>(null)

  // Focus the rename input as soon as it appears.
  useEffect(() => {
    if (editingId !== null) {
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [editingId])

  // Dismiss context menu on outside click or Escape.
  useEffect(() => {
    if (contextMenu === null) return
    function handleOutsideClick(_e: MouseEvent) {
      setContextMenu(null)
    }
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') setContextMenu(null)
    }
    document.addEventListener('mousedown', handleOutsideClick)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handleOutsideClick)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [contextMenu])

  function startEdit(tab: Tab, e: React.MouseEvent) {
    e.stopPropagation()
    setEditingId(tab.id)
    setEditValue(tab.name)
  }

  function startEditById(tabId: string) {
    const tab = tabs.find(t => t.id === tabId)
    if (tab) {
      setEditingId(tab.id)
      setEditValue(tab.name)
    }
  }

  function commitEdit() {
    if (editingId !== null) {
      const trimmed = editValue.trim()
      if (trimmed.length > 0) {
        onRename(editingId, trimmed)
      }
      setEditingId(null)
    }
  }

  function cancelEdit() {
    setEditingId(null)
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') commitEdit()
    if (e.key === 'Escape') cancelEdit()
  }

  return (
    <div className="tab-bar">
      <div className="tab-list">
        {tabs.map((tab) => (
          <div
            key={tab.id}
            className={`tab${tab.id === activeId ? ' tab--active' : ''}`}
            onClick={() => onSelect(tab.id)}
          >
            <span
              className={`tab__status tab__status--${sessionStatuses?.[tab.sessionId] || 'running'}`}
              title={sessionStatuses?.[tab.sessionId] || 'running'}
            />
            {editingId === tab.id ? (
              <input
                ref={inputRef}
                className="tab__rename-input"
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                onBlur={commitEdit}
                onKeyDown={handleKeyDown}
                onClick={(e) => e.stopPropagation()}
              />
            ) : (
              <span
                className="tab__name"
                onDoubleClick={(e) => startEdit(tab, e)}
                onContextMenu={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  setContextMenu({ tabId: tab.id, x: e.clientX, y: e.clientY })
                }}
                title="Double-click or right-click to rename"
              >
                {tab.name}
              </span>
            )}
            <button
              className="tab__close"
              onClick={(e) => {
                e.stopPropagation()
                onClose(tab.id)
              }}
              title="Close tab"
              aria-label={`Close ${tab.name}`}
            >
              ×
            </button>
          </div>
        ))}
      </div>

      <div className="tab-bar__controls">
        <button
          className="tab-bar__btn tab-bar__btn--add"
          onClick={onAdd}
          title="New terminal tab"
          aria-label="New terminal tab"
        >
          +
        </button>
        <button
          className="tab-bar__btn tab-bar__btn--settings"
          onClick={onSettings}
          title="Settings"
          aria-label="Settings"
        >
          &#9881;{/* gear icon */}
        </button>
      </div>

      {contextMenu && tabs.some(t => t.id === contextMenu.tabId) && (
        <div
          className="tab__context-menu"
          role="menu"
          style={{ position: 'fixed', top: contextMenu.y, left: contextMenu.x }}
          onMouseDown={(e) => e.stopPropagation()}
        >
          <button
            role="menuitem"
            className="tab__context-menu__item"
            onClick={() => {
              startEditById(contextMenu.tabId)
              setContextMenu(null)
            }}
          >
            Rename
          </button>
        </div>
      )}
    </div>
  )
}
