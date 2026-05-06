/**
 * openLink — Phase 95 (LNK-04) — single platform-aware URL-opening helper.
 *
 * Desktop (Wails): routes through BrowserOpenURL → OS default browser via
 * Wails IPC. Web: window.open with '_blank' + 'noopener,noreferrer' so the
 * new tab cannot pivot via window.opener. Both paths re-validate scheme
 * (defense-in-depth — caller is expected to have checked already).
 *
 * NEVER assign to location.href, window.location, or any current-tab
 * navigation sink. The 95-06 grep regression test enforces this.
 */

import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime';

export type ModifierMode = 'platform' | 'cmd' | 'ctrl' | 'none';

/**
 * Resolve modifier-pressed semantics by platform + user-configured mode.
 *  'platform' — Cmd on macOS, Ctrl elsewhere (default; matches VS Code, Terminal.app)
 *  'cmd'      — always require Cmd (macOS-only convenience; on linux+win, returns false unless metaKey is set)
 *  'ctrl'     — always require Ctrl
 *  'none'     — bypass modifier requirement; LNK-03 risk gates still apply
 *
 * Pitfall #3 (95-RESEARCH): event.metaKey is the Super/Cmd key on Linux/Windows
 * (not the Ctrl key) — so 'platform' mode explicitly checks ctrlKey when
 * navigator.platform is non-Mac, NOT metaKey.
 */
export function isModifierPressed(event: MouseEvent, mode: ModifierMode): boolean {
  if (mode === 'none') return true;
  const isMac =
    typeof navigator !== 'undefined' &&
    navigator.platform.toUpperCase().includes('MAC');
  if (mode === 'platform') return isMac ? event.metaKey : event.ctrlKey;
  if (mode === 'cmd') return event.metaKey;
  if (mode === 'ctrl') return event.ctrlKey;
  return false;
}

/**
 * Open a URL via the platform-correct path. Scheme is re-validated at the
 * deepest layer (defense-in-depth — never trust the caller).
 *
 * - Wails desktop: BrowserOpenURL (opens in OS default browser, NEVER inside the WebView)
 * - Web (Tailscale-served): window.open(url, '_blank', 'noopener,noreferrer')
 *
 * NEVER navigates the current tab. NEVER omits 'noopener,noreferrer' on web.
 */
export function openLink(url: string): void {
  if (!/^(https?:|mailto:)/i.test(url)) return; // Defense-in-depth scheme gate.
  const hasWails =
    typeof window !== 'undefined' &&
    typeof (window as { runtime?: { BrowserOpenURL?: unknown } }).runtime
      ?.BrowserOpenURL === 'function';
  if (hasWails) {
    BrowserOpenURL(url);
  } else {
    window.open(url, '_blank', 'noopener,noreferrer');
  }
}
