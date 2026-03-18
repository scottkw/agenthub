import React, { useEffect, useState, useCallback } from 'react'
import { TabBar, type Tab } from './components/TabBar'
import { TerminalPanel } from './components/TerminalPanel'
import { SettingsPanel } from './components/SettingsPanel'
import {
  CreateSession,
  ListSessions,
  KillSession,
  RenameSession,
  DetectCLIs,
  GetRelayPort,
} from './wailsjs/go/main/App'
import type { DetectedCLI } from './wailsjs/go/main/App'

/**
 * App is the root component — it owns all tab state and wires
 * the Wails-generated TypeScript bindings to the child components.
 */
function App(): React.ReactElement {
  const [tabs, setTabs] = useState<Tab[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [relayPort, setRelayPort] = useState<number | null>(null)
  const [showSettings, setShowSettings] = useState(false)
  const [detectedCLIs, setDetectedCLIs] = useState<DetectedCLI[]>([])
  const [tabCounter, setTabCounter] = useState(1)
  // Track if CLI picker dropdown is open (multiple CLIs)
  const [showCLIPicker, setShowCLIPicker] = useState(false)

  // On mount: get relay port, detect CLIs, restore any existing sessions.
  useEffect(() => {
    async function init() {
      try {
        const [port, clis, sessions] = await Promise.all([
          GetRelayPort(),
          DetectCLIs(),
          ListSessions(),
        ])
        setRelayPort(port)
        setDetectedCLIs(clis)

        // Restore existing sessions as tabs (SESS-02 reattachment after window re-show).
        if (sessions.length > 0) {
          const restoredTabs: Tab[] = sessions.map((s) => ({
            id: s.id,
            name: s.name || s.cli,
            sessionId: s.id,
            cli: s.cli,
          }))
          setTabs(restoredTabs)
          setActiveId(restoredTabs[0].id)
        }
      } catch (err) {
        console.error('[App] init failed:', err)
      }
    }
    void init()
  }, [])

  const createTab = useCallback(async (cliName: string) => {
    const defaultName = `${cliName} ${tabCounter}`
    setTabCounter((n) => n + 1)
    try {
      const sessionId = await CreateSession(cliName, defaultName)
      const tab: Tab = {
        id: sessionId,
        name: defaultName,
        sessionId,
        cli: cliName,
      }
      setTabs((prev) => [...prev, tab])
      setActiveId(sessionId)
    } catch (err) {
      console.error('[App] CreateSession failed:', err)
    }
  }, [tabCounter])

  const handleAddTab = useCallback(() => {
    if (detectedCLIs.length === 0) {
      // No CLIs found — open settings so the user can configure a path.
      setShowSettings(true)
      return
    }
    if (detectedCLIs.length === 1) {
      // Only one CLI available: create immediately.
      void createTab(detectedCLIs[0].Name)
    } else {
      // Multiple CLIs: show a picker.
      setShowCLIPicker(true)
    }
  }, [detectedCLIs, createTab])

  const handleSelectCLI = useCallback((cliName: string) => {
    setShowCLIPicker(false)
    void createTab(cliName)
  }, [createTab])

  const handleCloseTab = useCallback(async (id: string) => {
    try {
      await KillSession(id)
    } catch (err) {
      console.warn('[App] KillSession failed:', err)
    }
    setTabs((prev) => {
      const remaining = prev.filter((t) => t.id !== id)
      // Activate an adjacent tab if the closed one was active.
      if (activeId === id) {
        const idx = prev.findIndex((t) => t.id === id)
        const next = remaining[Math.max(0, idx - 1)]
        setActiveId(next?.id ?? null)
      }
      return remaining
    })
  }, [activeId])

  const handleRenameTab = useCallback(async (id: string, name: string) => {
    try {
      await RenameSession(id, name)
    } catch (err) {
      console.warn('[App] RenameSession failed:', err)
    }
    setTabs((prev) =>
      prev.map((t) => (t.id === id ? { ...t, name } : t))
    )
  }, [])

  return (
    <div className="app">
      <TabBar
        tabs={tabs}
        activeId={activeId}
        onSelect={setActiveId}
        onClose={handleCloseTab}
        onRename={handleRenameTab}
        onAdd={handleAddTab}
        onSettings={() => setShowSettings(true)}
      />

      <div className="terminal-container">
        {relayPort !== null &&
          tabs.map((tab) => (
            <TerminalPanel
              key={tab.sessionId}
              sessionId={tab.sessionId}
              isActive={tab.id === activeId}
              relayPort={relayPort}
            />
          ))}
      </div>

      {/* CLI picker dropdown — shown when multiple CLIs are detected */}
      {showCLIPicker && (
        <div className="cli-picker-overlay" onClick={() => setShowCLIPicker(false)}>
          <div className="cli-picker" onClick={(e) => e.stopPropagation()}>
            <p className="cli-picker__label">Choose a CLI:</p>
            {detectedCLIs.map((cli) => (
              <button
                key={cli.Name}
                className="cli-picker__btn"
                onClick={() => handleSelectCLI(cli.Name)}
              >
                {cli.Name}
              </button>
            ))}
          </div>
        </div>
      )}

      <SettingsPanel
        isOpen={showSettings}
        onClose={() => setShowSettings(false)}
        clis={detectedCLIs}
      />
    </div>
  )
}

export default App
