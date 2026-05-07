/**
 * stripAnsi — Phase 97 (SER-01) — pure helper that strips ANSI escape
 * sequences from arbitrary text. Used between SerializeAddon.serialize()
 * (which emits ANSI-laden text by design — it is built for terminal
 * replay, not human reading) and the file write performed via the
 * SaveTerminalSession Wails RPC.
 *
 * The regex covers SGR, ECH, CUF/CUB/CUU/CUD, and DEC private mode
 * sequences — i.e. the complete vocabulary SerializeAddon emits when
 * called with { excludeModes: true }. See 97-RESEARCH.md §"ANSI Output
 * Audit" (lines 139-167) for the audit-verified emit table.
 *
 * NO network calls. NO logging. NO DOM. NO console output.
 */

// Pattern: ESC ('\x1b') followed by '[', optional '?' for DEC private
// modes, zero-or-more digits/semicolons, then a single-letter terminator.
// Audit-verified to cover SGR (m), ECH (X), cursor moves (A/B/C/D), and
// DEC private modes (l/h with ? prefix). The 'g' flag enables single-pass
// global replacement. See 97-RESEARCH.md §"Strip regex" line 158.
const ANSI_ESCAPE_RE = /\x1b\[\??[0-9;]*[a-zA-Z]/g

export function stripAnsi(input: string): string {
  return input.replace(ANSI_ESCAPE_RE, '')
}
