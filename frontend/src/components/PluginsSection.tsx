import React, { useState, useEffect } from 'react'
import { GetPluginSettings, SetPluginSettings } from '../wailsjs/go/main/App'
import { daemon } from '../wailsjs/go/models'

type PluginSettings = daemon.PluginSettings
const PluginSettings = daemon.PluginSettings

/**
 * PluginsSection — Phase 92 (PUI-01)
 *
 * Renders the 8-plugin enable/disable section in the Settings tab. Persists
 * via the Wails GetPluginSettings/SetPluginSettings bindings; the GUI layer
 * (app.go) emits 'settings:plugins' on a successful save which propagates the
 * new pluginConfig into every open TerminalPanel via App.tsx prop drilling.
 *
 * Phase 92 contract: TerminalPanel does NOT consume pluginConfig — the
 * pipeline is wired but inert. Phase 93 wires consumption.
 */
export function PluginsSection(): React.ReactElement {
  // Local edited state — null until GetPluginSettings resolves
  const [pluginConfig, setPluginConfig] = useState<PluginSettings | null>(null)
  const [pluginsLoaded, setPluginsLoaded] = useState(false)

  // Three-state Save (mirrors SettingsTab.tsx Save Paths cadence)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)

  useEffect(() => {
    GetPluginSettings()
      .then((s) => {
        setPluginConfig(s)
        setPluginsLoaded(true)
      })
      .catch((err) => {
        setLoadError(err instanceof Error ? err.message : String(err))
        setPluginsLoaded(true)
      })
  }, [])

  async function handleSavePlugins(): Promise<void> {
    if (!pluginConfig) return
    setSaving(true)
    setError(null)
    try {
      await SetPluginSettings(pluginConfig)
      setSaved(true)
      setTimeout(() => setSaved(false), 1500)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  // Narrow the row keys to only boolean fields so renderRow stays type-safe
  // and `checked={...}` accepts the value (PluginSettings has a nested object
  // field added in Phase 94 — see daemon models).
  type PluginBooleanKey = {
    [K in keyof PluginSettings]: PluginSettings[K] extends boolean ? K : never
  }[keyof PluginSettings]

  const toggle = (key: PluginBooleanKey) => () => {
    setPluginConfig((prev) => {
      if (!prev) return prev
      // Construct a fresh PluginSettings instance from the existing one so
      // class identity (and any nested instance fields) are preserved.
      const next = new PluginSettings({ ...prev, [key]: !prev[key] })
      return next
    })
  }

  // Helper to render one toggle row. Mirrors SettingsTab Behavior toggle markup
  // verbatim — only label + description + key change. The <input> renders
  // unconditionally (test selectors find it); only the visible <label> is
  // gated by pluginsLoaded (Pitfall #3 — flicker guard).
  function renderRow(
    key: PluginBooleanKey,
    label: string,
    description: string,
    caption?: string,
  ): React.ReactElement {
    const checked = pluginConfig?.[key] ?? false
    return (
      <div className="settings-panel__field-group" key={key as string}>
        {pluginsLoaded && pluginConfig && (
          <label
            className={`settings-panel__toggle-row${checked ? ' settings-panel__toggle-row--checked' : ''}`}
            htmlFor={`plugin-${key as string}`}
            style={saving ? { pointerEvents: 'none', opacity: 0.6 } : undefined}
          >
            <span className="settings-panel__toggle-track">
              <span className="settings-panel__toggle-thumb" />
            </span>
            <span className="settings-panel__toggle-label">{label}</span>
          </label>
        )}
        <input
          type="checkbox"
          id={`plugin-${key as string}`}
          className="settings-panel__toggle-input"
          checked={checked}
          onChange={toggle(key)}
        />
        <p className="settings-panel__description">{description}</p>
        {caption && (
          <p className="settings-panel__description settings-panel__description--italic">
            {caption}
          </p>
        )}
      </div>
    )
  }

  return (
    <>
      <h3>Plugins</h3>
      {loadError && (
        <p className="settings-panel__error">
          Could not load plugin settings — {loadError}
        </p>
      )}
      {/* Order is non-negotiable per UI-SPEC: webgl → unicode11 → search →
          webLinks → image → serialize → clipboard → progress (Pitfall #5). */}
      {renderRow('webgl', 'WebGL renderer',
        'GPU-accelerated terminal rendering with automatic DOM fallback if the GPU context is lost.')}
      {renderRow('unicode11', 'Unicode 11 widths',
        'Correct cell widths for emoji and CJK characters using the Unicode 11 width tables.',
        'Applies to new sessions you create.')}
      {renderRow('search', 'Find in scrollback',
        'Open a find bar with Cmd-F to search the terminal scrollback buffer.')}
      {renderRow('webLinks', 'Clickable web links',
        'Detect URLs in terminal output and make them clickable with Cmd-click (macOS) or Ctrl-click.')}
      {renderRow('image', 'Inline images',
        'Render images sent via sixel or the iTerm2 inline image protocol directly inside the terminal.',
        'Applies to new sessions you create.')}
      {renderRow('serialize', 'Save terminal as text',
        'Right-click a tab to export the visible scrollback as a text file.',
        'Saved files include any secrets, tokens, or sensitive data printed in the session.')}
      {renderRow('clipboard', 'Clipboard (OSC 52)',
        'Allow the running CLI to place text on the system clipboard via the OSC 52 escape sequence.')}
      {renderRow('progress', 'Progress (OSC 9;4)',
        'Show a per-tab progress underline when the running CLI emits OSC 9;4 progress updates.')}

      {error && (
        <p className="settings-panel__error">
          Could not save plugin settings — {error}
        </p>
      )}
      <div className="settings-panel__save-paths-row">
        <button
          className={`settings-panel__btn ${saved ? 'settings-panel__btn--saved' : 'settings-panel__btn--save'}`}
          onClick={() => void handleSavePlugins()}
          disabled={saving || saved || !pluginsLoaded}
        >
          {saving ? 'Saving…' : saved ? 'Saved!' : 'Save Plugins'}
        </button>
      </div>
    </>
  )
}
