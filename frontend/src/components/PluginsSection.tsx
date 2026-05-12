import React, { useState, useEffect, useRef } from 'react'
import {
  GetPluginSettings,
  SetPluginSettings,
  SetSearchConfig,
  SetWebLinksConfig,
  SetImageConfig,
} from '../wailsjs/go/main/App'
import { daemon } from '../wailsjs/go/models'

type PluginSettings = daemon.PluginSettings
const PluginSettings = daemon.PluginSettings

/**
 * PluginsSection — Phase 92 (PUI-01), Phase 99 (PUI-02)
 *
 * Renders the 8-plugin enable/disable section in the Settings tab. Persists
 * via the Wails Get/Set plugin-settings bindings; the GUI layer
 * (app.go) emits 'settings:plugins' on a successful save which propagates the
 * new pluginConfig into every open TerminalPanel via App.tsx prop drilling.
 *
 * Phase 92 contract: TerminalPanel does NOT consume pluginConfig — the
 * pipeline is wired but inert. Phase 93 wires consumption.
 *
 * Phase 99 PUI-02: onPluginToggleSideEffect callback fires after a successful
 * save when unicode11 or image booleans changed vs the last-saved snapshot.
 * See PluginToggleKind + PluginsSectionProps exports below the function.
 */
export function PluginsSection({
  onPluginToggleSideEffect,
}: PluginsSectionProps = {}): React.ReactElement {
  // Local edited state — null until GetPluginSettings resolves
  const [pluginConfig, setPluginConfig] = useState<PluginSettings | null>(null)
  const [pluginsLoaded, setPluginsLoaded] = useState(false)

  // Phase 99 PUI-02: snapshot of the last-saved state for diff detection.
  // Initialized when GetPluginSettings resolves; updated after each successful save.
  const lastSavedRef = useRef<PluginSettings | null>(null)

  // Phase 99 PUI-03: 500ms debounce for ImageConfig.storageLimit number input
  // — avoids spamming SetImageConfig RPC while user types digits (T-99-05).
  const imageStorageDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => {
    return () => {
      if (imageStorageDebounceRef.current) {
        clearTimeout(imageStorageDebounceRef.current)
      }
    }
  }, [])

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
        lastSavedRef.current = s
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
      // Phase 99 PUI-02: compute diff between last-saved snapshot and current saved value.
      // Only unicode11 and image trigger the side-effect (cannot hot-swap — new sessions only).
      const prior = lastSavedRef.current
      const kinds: PluginToggleKind[] = []
      if (prior) {
        if (prior.unicode11 !== pluginConfig.unicode11) kinds.push('unicode11')
        if (prior.image !== pluginConfig.image) kinds.push('image')
      }
      lastSavedRef.current = pluginConfig
      if (kinds.length > 0 && onPluginToggleSideEffect) {
        onPluginToggleSideEffect(kinds)
      }
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
    disclosure?: React.ReactNode,
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
        {disclosure}
      </div>
    )
  }

  // Phase 99 PUI-03: inline <details> disclosures for plugins with meaningful
  // runtime config. Each disclosure dispatches a SUB-KEY RPC (SetSearchConfig /
  // SetWebLinksConfig / SetImageConfig) — never the full-snapshot save RPC —
  // to honor the PUI-04 anti-race contract from Phase 94-07 WR-03.
  const renderSearchDisclosure = (): React.ReactNode => {
    if (!pluginsLoaded || !pluginConfig) return null
    const sc = pluginConfig.searchConfig
    const dispatch = (next: daemon.SearchConfig) => {
      SetSearchConfig(next).catch(() => { /* silent — sub-key RPC */ })
    }
    return (
      <details className="settings-panel__details">
        <summary>Search defaults</summary>
        <label className="settings-panel__toggle-row" htmlFor="search-default-regex">
          <input
            type="checkbox"
            id="search-default-regex"
            checked={sc.regex}
            onChange={(e) => dispatch(new daemon.SearchConfig({ ...sc, regex: e.target.checked }))}
          />
          <span className="settings-panel__toggle-label">Regex</span>
        </label>
        <label className="settings-panel__toggle-row" htmlFor="search-default-case">
          <input
            type="checkbox"
            id="search-default-case"
            checked={sc.caseSensitive}
            onChange={(e) => dispatch(new daemon.SearchConfig({ ...sc, caseSensitive: e.target.checked }))}
          />
          <span className="settings-panel__toggle-label">Case sensitive</span>
        </label>
        <label className="settings-panel__toggle-row" htmlFor="search-default-word">
          <input
            type="checkbox"
            id="search-default-word"
            checked={sc.wholeWord}
            onChange={(e) => dispatch(new daemon.SearchConfig({ ...sc, wholeWord: e.target.checked }))}
          />
          <span className="settings-panel__toggle-label">Whole word</span>
        </label>
      </details>
    )
  }

  const renderWebLinksDisclosure = (): React.ReactNode => {
    if (!pluginsLoaded || !pluginConfig) return null
    const wc = pluginConfig.webLinksConfig
    const dispatch = (next: daemon.WebLinksConfig) => {
      SetWebLinksConfig(next).catch(() => { /* silent — sub-key RPC */ })
    }
    return (
      <details className="settings-panel__details">
        <summary>Link click behavior</summary>
        <label className="settings-panel__field-group">
          <span className="settings-panel__description">Modifier</span>
          <select
            value={wc.modifier}
            onChange={(e) => dispatch(new daemon.WebLinksConfig({ ...wc, modifier: e.target.value }))}
          >
            <option value="platform">Platform default (Cmd on macOS, Ctrl elsewhere)</option>
            <option value="cmd">Cmd</option>
            <option value="ctrl">Ctrl</option>
            <option value="none">No modifier (plain click)</option>
          </select>
        </label>
        <label className="settings-panel__toggle-row">
          <input
            type="checkbox"
            checked={wc.confirmOSC8}
            onChange={(e) => dispatch(new daemon.WebLinksConfig({ ...wc, confirmOSC8: e.target.checked }))}
          />
          <span className="settings-panel__toggle-label">Confirm OSC 8 hyperlinks (mismatched display vs href)</span>
        </label>
        <label className="settings-panel__toggle-row">
          <input
            type="checkbox"
            checked={wc.confirmIDN}
            onChange={(e) => dispatch(new daemon.WebLinksConfig({ ...wc, confirmIDN: e.target.checked }))}
          />
          <span className="settings-panel__toggle-label">Confirm IDN / Punycode URLs</span>
        </label>
        <label className="settings-panel__toggle-row">
          <input
            type="checkbox"
            checked={wc.confirmTyposquat}
            onChange={(e) => dispatch(new daemon.WebLinksConfig({ ...wc, confirmTyposquat: e.target.checked }))}
          />
          <span className="settings-panel__toggle-label">Confirm typosquat patterns</span>
        </label>
      </details>
    )
  }

  const renderImageDisclosure = (): React.ReactNode => {
    if (!pluginsLoaded || !pluginConfig) return null
    const ic = pluginConfig.imageConfig
    return (
      <details className="settings-panel__details">
        <summary>Storage limit</summary>
        <label className="settings-panel__field-group">
          <span className="settings-panel__description">Per-terminal decoded image storage cap (default 16 MB; max 1000 MB).</span>
          <input
            type="number"
            min={1}
            max={1000}
            step={1}
            value={ic.storageLimit}
            onChange={(e) => {
              const raw = Number(e.target.value)
              const v = Math.max(1, Math.min(1000, Number.isFinite(raw) ? raw : 16))
              setPluginConfig((prev) => {
                if (!prev) return prev
                return new PluginSettings({
                  ...prev,
                  imageConfig: new daemon.ImageConfig({ storageLimit: v }),
                })
              })
              if (imageStorageDebounceRef.current) clearTimeout(imageStorageDebounceRef.current)
              imageStorageDebounceRef.current = setTimeout(() => {
                SetImageConfig(new daemon.ImageConfig({ storageLimit: v })).catch(() => { /* silent */ })
              }, 500)
            }}
          />
          {' MB'}
        </label>
      </details>
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
        'Open a find bar with Cmd-F to search the terminal scrollback buffer.',
        undefined,
        renderSearchDisclosure())}
      {renderRow('webLinks', 'Clickable web links',
        'Detect URLs in terminal output and make them clickable with Cmd-click (macOS) or Ctrl-click.',
        undefined,
        renderWebLinksDisclosure())}
      {renderRow('image', 'Inline images',
        'Render images sent via sixel or the iTerm2 inline image protocol directly inside the terminal.',
        'Applies to new sessions you create.',
        renderImageDisclosure())}
      {renderRow('serialize', 'Save terminal as text',
        'Right-click a tab to export the visible scrollback as a text file.',
        'Saved files include any secrets, tokens, or sensitive data printed in the session.')}
      {renderRow('clipboard', 'Clipboard (OSC 52)',
        'Allow the running CLI to place text on the system clipboard via the OSC 52 escape sequence.')}
      {renderRow('progress', 'Progress (OSC 9;4)',
        'Show a per-tab progress underline when the running CLI emits OSC 9;4 progress updates.',
        'Default OFF in v3.2 — flips ON in v3.3 after field validation.')}

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

/**
 * Phase 99 PUI-02 type exports.
 * Declared after PluginsSection body so the plugin key literals ('webgl', 'unicode11', etc.)
 * in the renderRow call sites retain their UI-SPEC positional order for source-inspection tests.
 * TypeScript module-level type declarations are accessible throughout the module regardless
 * of declaration position (they are not runtime values).
 */
export type PluginToggleKind = 'unicode11' | 'image'

export interface PluginsSectionProps {
  onPluginToggleSideEffect?: (kinds: PluginToggleKind[]) => void
}
