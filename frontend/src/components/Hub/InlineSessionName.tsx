import React, { useState, useRef, useEffect } from 'react'

export interface InlineSessionNameProps {
  id: string
  name: string
  onRenamed?: (name: string) => void
}

/**
 * InlineSessionName — inline-editable session name for Hub cards.
 *
 * CR-02 fix: this component does NOT call RenameSession directly.
 * The RPC is owned by App.handleRenameTab, which is wired as the onRename
 * callback on HubPanel → SessionCardGrid → SessionCard → InlineSessionName.
 * This component fires onRenamed(trimmed) and lets the parent chain handle
 * the actual RPC + state update. This prevents the double-RPC that occurred
 * when InlineSessionName called RenameSession AND the parent's onRenamed
 * callback also propagated up to App.handleRenameTab which called it again.
 *
 * Mirrors the TabBar rename pattern (TabBar.tsx lines 140-172 and 203-225):
 * - Click the name span to enter edit mode
 * - Enter commits the rename — fires onRenamed, exits edit mode
 * - Escape cancels and restores the original name
 * - Blur with unchanged or empty-trimmed value does NOT fire onRenamed
 *
 * CSS reuse: tab__rename-input (style.css ~line 142) — no new rename-input CSS authored.
 */
export function InlineSessionName({ id: _id, name, onRenamed }: InlineSessionNameProps): React.ReactElement {
  const [editing, setEditing] = useState(false)
  const [editValue, setEditValue] = useState(name)
  const inputRef = useRef<HTMLInputElement>(null)

  // When entering edit mode, select all text in the input
  useEffect(() => {
    if (editing) {
      inputRef.current?.select()
    }
  }, [editing])

  // Keep editValue in sync if name prop changes externally while not editing
  useEffect(() => {
    if (!editing) {
      setEditValue(name)
    }
  }, [name, editing])

  function commitEdit(): void {
    const trimmed = editValue.trim()
    if (trimmed.length > 0 && trimmed !== name) {
      // CR-02: fire callback only — App.handleRenameTab owns the RenameSession RPC
      onRenamed?.(trimmed)
    } else {
      setEditValue(name)
    }
    setEditing(false)
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>): void {
    if (e.key === 'Enter') {
      commitEdit()
    }
    if (e.key === 'Escape') {
      setEditValue(name)
      setEditing(false)
    }
  }

  if (editing) {
    return (
      <input
        ref={inputRef}
        className="tab__rename-input"
        value={editValue}
        placeholder="Session name"
        onChange={(e) => setEditValue(e.target.value)}
        onBlur={commitEdit}
        onKeyDown={handleKeyDown}
        onClick={(e) => e.stopPropagation()}
      />
    )
  }

  return (
    <span
      className="hub-card__name"
      onClick={() => {
        setEditValue(name)
        setEditing(true)
      }}
    >
      {name}
    </span>
  )
}
