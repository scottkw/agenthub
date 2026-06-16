import React, { useState, useRef, useEffect } from 'react'

export interface Tab {
  id: string
  name: string
  sessionId: string
  cli: string
  type?: 'terminal' | 'welcome' | 'daemon-manager' | 'remote-sessions' | 'settings' | 'file-browser' | 'hub'
}

// Phase 101-02 (SHELL-06 GUI half) — agent badge color resolution.
// Returns the BEM modifier suffix (without the "--" prefix) for the given
// cli string, or null when the cli isn't a known agent (caller renders the
// badge with the base class only, yielding the muted fallback color).
//
// The 5 shell variants all collapse to a single "shell" modifier — the badge
// communicates "this is a shell session", not which specific shell.
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

// Phase 101-02 — human-readable agent label for the tab tooltip suffix.
// Locked copy: shells use "Shell — DISPLAYNAME" (em-dash U+2014); AI CLIs
// use the same product names the new-session modal shows. Unknown CLIs
// fall through to the raw cli string so the tab still renders a useful
// hover tip.
function agentDisplayName(cli: string): string {
  const shellMap: Record<string, string> = {
    shell: 'system default',
    bash: 'bash',
    zsh: 'zsh',
    pwsh: 'PowerShell',
    powershell: 'Windows PowerShell',
  }
  if (shellMap[cli] !== undefined) return 'Shell — ' + shellMap[cli]
  const cliMap: Record<string, string> = {
    claude: 'Claude Code',
    opencode: 'OpenCode',
    codex: 'OpenAI Codex',
    gemini: 'Gemini CLI',
    cursor: 'Cursor',
    aider: 'Aider',
  }
  return cliMap[cli] ?? cli
}

interface TabBarProps {
  tabs: Tab[]
  activeId: string | null
  onSelect: (id: string) => void
  onClose: (id: string) => void
  onRename: (id: string, name: string) => void
  /**
   * Phase 97 SER-01: invoked when user picks the save menu item from
   * the right-click context menu. The handler in App.tsx looks up the
   * tab's session, runs the saver closure, strips ANSI, sanitizes the
   * filename, and calls the SaveTerminalSession Wails RPC.
   */
  onRequestSave?: (tabId: string) => void
  sessionStatuses?: Record<string, string>
  exitCountdowns?: Record<string, number>  // sessionId -> seconds remaining
  /**
   * Phase 98 PRG-02 — per-tab progress value (0-100) keyed by sessionId.
   * Populated by App.tsx handleProgressChange; consumed in Wave 3 (Plan 04)
   * to render the progress underline on each tab. Declared optional here so
   * the prop can be wired at Wave 2 and consumed at Wave 3 without a
   * TypeScript error between waves.
   */
  tabProgress?: Record<string, number>
  /**
   * Phase 120-04 UI-01 — open (or focus) the FileBrowserTab for a session via
   * the tab's right-click context menu. Shown only for terminal tabs (those
   * with a truthy sessionId).
   */
  onBrowseFiles?: (sessionId: string, sessionName: string) => void
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
  onRequestSave,
  sessionStatuses,
  exitCountdowns,
  tabProgress,
  onBrowseFiles,
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
        {tabs.map((tab) => {
          // Phase 101-02 (SHELL-06 GUI half) — agent badge between status dot
          // and tab name. Decorative (aria-hidden); the same info is exposed
          // textually in the tab-name tooltip suffix.
          const badgeModifier = agentBadgeModifier(tab.cli)
          const badgeClass = badgeModifier
            ? `tab__agent-badge tab__agent-badge--${badgeModifier}`
            : 'tab__agent-badge'
          // Tab tooltip carries both the rename hint AND the agent-type
          // suffix — e.g., "Double-click or right-click to rename · Shell — bash".
          // For unknown CLIs (cli=""), omit the suffix so welcome / daemon /
          // settings tabs don't render a meaningless "()" string.
          const agentLabel = tab.cli ? agentDisplayName(tab.cli) : ''
          const titleText = agentLabel
            ? `Double-click or right-click to rename · ${agentLabel}`
            : 'Double-click or right-click to rename'
          return (
          <div
            key={tab.id}
            className={`tab${tab.id === activeId ? ' tab--active' : ''}${exitCountdowns?.[tab.sessionId] ? ' tab--exiting' : ''}`}
            onClick={() => onSelect(tab.id)}
          >
            <span
              className={`tab__status tab__status--${sessionStatuses?.[tab.sessionId] || 'running'}`}
              title={sessionStatuses?.[tab.sessionId] || 'running'}
            />
            <span className={badgeClass} aria-hidden="true" />
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
                title={titleText}
              >
                {tab.name}
              </span>
            )}
            {exitCountdowns?.[tab.sessionId] && (
              <span className="tab__countdown">{exitCountdowns[tab.sessionId]}s</span>
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
            {/* Phase 98 PRG-02 — per-tab progress underline. transform (not width) for GPU
                compositor smoothness (Pitfall #4); transform-origin: left grows L→R; transition
                in style.css handles the 200ms ease-out animation on value changes. */}
            <div
              className="tab__progress"
              style={{
                transform: `scaleX(${(tabProgress?.[tab.sessionId] ?? 0) / 100})`,
              }}
              data-testid={`tab-progress-${tab.id}`}
            />
          </div>
          )
        })}
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
          <button
            role="menuitem"
            className="tab__context-menu__item"
            onClick={() => {
              onRequestSave?.(contextMenu.tabId)
              setContextMenu(null)
            }}
          >
            Save Terminal As…
          </button>
          {/* Phase 120-04 UI-01 — Browse files menu item, gated on terminal
              tabs (those with a truthy sessionId). Welcome / Settings /
              file-browser tabs do not get this item. */}
          {(() => {
            const tab = tabs.find((t) => t.id === contextMenu.tabId)
            if (!tab?.sessionId) return null
            if (!onBrowseFiles) return null
            return (
              <button
                role="menuitem"
                className="tab__context-menu__item"
                data-testid="tab-context-browse-files"
                onClick={() => {
                  onBrowseFiles(tab.sessionId, tab.name)
                  setContextMenu(null)
                }}
              >
                Browse files
              </button>
            )
          })()}
        </div>
      )}
    </div>
  )
}
