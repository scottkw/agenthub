// Phase 162 UAT harness — isolated mount of the real Settings jump bar + search
// against a faithful body (7 sections, Terminal Plugins last). Temporary; safe to delete.
import React from 'react'
import { createRoot } from 'react-dom/client'
import './src/style.css'
import { SettingsJumpBar } from './src/components/SettingsJumpBar'
import { SettingsSearch } from './src/components/SettingsSearch'

// Filler to make each section tall enough that hash navigation produces real scroll.
function Filler(): React.ReactElement {
  return (
    <>
      {Array.from({ length: 8 }).map((_, i) => (
        <p key={i} className="settings-panel__description">
          Placeholder content line {i + 1} for scroll height.
        </p>
      ))}
    </>
  )
}

function Harness(): React.ReactElement {
  return (
    // Mirror the real shell: .settings-panel > .settings-tab > .settings-panel__body
    <div className="settings-panel" style={{ height: '70vh' }}>
      <div className="settings-tab">
        <div className="settings-panel__body">
          <SettingsJumpBar />
          <SettingsSearch />
          <h3 id="settings-behavior">Behavior</h3>
          <Filler />
          <h3 id="settings-session-behavior">Session Behavior</h3>
          <Filler />
          <h3 id="settings-appearance">Appearance</h3>
          <Filler />
          <h3 id="settings-web-server">Web Server</h3>
          <Filler />
          <h3 id="settings-security">Security</h3>
          <Filler />
          <h3 id="settings-paths">Paths</h3>
          <Filler />
          {/* Replicates PluginsSection.tsx:289 exactly — last section in the body. */}
          <h3 id="settings-plugins">Terminal Plugins</h3>
          <Filler />
        </div>
      </div>
    </div>
  )
}

createRoot(document.getElementById('root')!).render(<Harness />)
