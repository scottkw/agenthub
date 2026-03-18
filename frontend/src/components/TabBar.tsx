import React, { useState, useRef, useEffect } from 'react'

export interface Tab {
  id: string
  name: string
  sessionId: string
  cli: string
}

interface TabBarProps {
  tabs: Tab[]
  activeId: string | null
  onSelect: (id: string) => void
  onClose: (id: string) => void
  onRename: (id: string, name: string) => void
  onAdd: () => void
  onSettings: () => void
}

/**
 * Horizontal tab bar with add/rename/close controls.
 * Double-clicking a tab name enables inline editing.
 */
export function TabBar({
  tabs,
  activeId,
  onSelect,
  onClose,
  onRename,
  onAdd,
  onSettings,
}: TabBarProps): React.ReactElement {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  // Focus the rename input as soon as it appears.
  useEffect(() => {
    if (editingId !== null) {
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [editingId])

  function startEdit(tab: Tab, e: React.MouseEvent) {
    e.stopPropagation()
    setEditingId(tab.id)
    setEditValue(tab.name)
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
                title="Double-click to rename"
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
    </div>
  )
}
