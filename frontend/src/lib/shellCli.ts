/**
 * Shell-session detection for the web-share security warning (Phase 150 SET-01).
 *
 * A session's `cli` is a bare command for agent CLIs ('claude', 'codex') but a
 * full binary PATH for shell sessions ('/bin/zsh', 'C:\\...\\powershell.exe'),
 * because shells are launched from their resolved $SHELL / discovered path.
 * The warning gate therefore must normalize to the basename (minus a .exe
 * suffix, case-insensitive) before testing membership — matching the bare set
 * against a path silently fails for every real shell session.
 *
 * Single source of truth shared by both share surfaces (App.tsx StatusBar path
 * and SessionShareModal Hub path) so they cannot drift apart (cross-surface
 * parity is release-blocking).
 */
export const SHELL_CLIS = new Set(['shell', 'bash', 'zsh', 'pwsh', 'powershell'])

export function isShellCli(cli: string | undefined | null): boolean {
  if (!cli) return false
  const base = cli.split(/[/\\]/).pop() ?? cli
  const normalized = base.toLowerCase().replace(/\.exe$/, '')
  return SHELL_CLIS.has(normalized)
}
