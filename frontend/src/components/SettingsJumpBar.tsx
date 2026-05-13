import React from 'react'

/**
 * Phase 104 SETUI-01/02: Sticky jump-link bar rendered at the top of the
 * Settings tab. Each link is a plain anchor (`<a href="#settings-{slug}">`),
 * so the browser handles the smooth-scroll behaviour via CSS
 * (`scroll-behavior: smooth` on `.settings-panel__body`). The matching
 * sections in `SettingsTab.tsx` and `PluginsSection.tsx` carry the
 * `id="settings-{slug}"` attributes referenced here.
 *
 * Section order mirrors the on-screen render order so the bar reads the
 * same direction the user scrolls (Plugins is rendered last in the body
 * today but is listed first here per the SETUI-01 spec — clicking it
 * jumps down to the Plugins section near the bottom).
 */
export interface JumpBarLink {
  label: string
  id: string
}

export const SETTINGS_JUMP_LINKS: ReadonlyArray<JumpBarLink> = [
  { label: 'Plugins', id: 'settings-plugins' },
  { label: 'Behavior', id: 'settings-behavior' },
  { label: 'Session Behavior', id: 'settings-session-behavior' },
  { label: 'Appearance', id: 'settings-appearance' },
  { label: 'Web Server', id: 'settings-web-server' },
  { label: 'Security', id: 'settings-security' },
  { label: 'Paths', id: 'settings-paths' },
]

export function SettingsJumpBar(): React.ReactElement {
  return (
    <nav className="settings-jump-bar" aria-label="Settings sections">
      {SETTINGS_JUMP_LINKS.map((link) => (
        <a
          key={link.id}
          href={`#${link.id}`}
          className="settings-jump-bar__link"
          data-target={link.id}
        >
          {link.label}
        </a>
      ))}
    </nav>
  )
}
