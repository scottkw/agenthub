// Phase 101-02 (SHELL-06 GUI half) — agent badge color resolution.
// Returns the BEM modifier suffix (without the "--" prefix) for the given
// cli string, or null when the cli isn't a known agent (caller renders the
// badge with the base class only, yielding the muted fallback color).
//
// The 5 shell variants all collapse to a single "shell" modifier — the badge
// communicates "this is a shell session", not which specific shell.
//
// Single source of truth for the per-CLI "session type" color identity.
// Consumed by:
//   - TabBar tab agent-badge dot (.tab__agent-badge--{modifier})
//   - SessionCard left spine (.hub-card[data-agent="{modifier}"])
// Keep the modifier set in sync with the .tab__agent-badge--* / .hub-card[data-agent]
// CSS palettes in style.css so the card spine can never drift from the tab dot.
export function agentBadgeModifier(cli: string): string | null {
  switch (cli) {
    case 'claude':
    case 'opencode':
    case 'codex':
    case 'gemini':
    case 'cursor':
    case 'aider':
    case 'agy':
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
