/**
 * sanitizeFilename — Phase 97 (SER-01) — pure helper that converts an
 * arbitrary tab-name string into a filesystem-safe basename for use with
 * the native Save File dialog.
 *
 * Defense-in-depth: the OS Save dialog also rejects path-traversal and
 * reserved names. This helper provides a clean default before the dialog
 * opens so the user sees a sensible suggested filename rather than a
 * corrupted one (Pitfall #4 in 97-RESEARCH.md). Returns the literal
 * 'session' as a neutral fallback when the input cannot be converted to
 * a usable basename (empty, leading-dot, Windows reserved name).
 *
 * NO network calls. NO logging. NO DOM. NO console output.
 */

// Windows reserved device names (case-insensitive). Files with these names
// cannot be created on FAT/NTFS even with extensions. See
// https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file.
const WINDOWS_RESERVED_RE = /^(con|prn|aux|nul|com[1-9]|lpt[1-9])$/i

export function sanitizeFilename(name: string): string {
  // Step 1: trim outer whitespace, collapse internal whitespace runs.
  const collapsed = name.trim().replace(/\s+/g, '_')
  // Step 2: replace anything outside [a-zA-Z0-9_.-] with underscore.
  // \w is [A-Za-z0-9_]; we also keep '-' and '.' (the latter for ext).
  const sanitized = collapsed.replace(/[^\w\-.]/g, '_')
  // Step 3: empty or leading-dot → fallback (defends against hidden files
  // and against returning empty string to the OS Save dialog).
  if (sanitized === '' || sanitized.startsWith('.')) return 'session'
  // Step 4: Windows reserved name → fallback.
  if (WINDOWS_RESERVED_RE.test(sanitized)) return 'session'
  return sanitized
}
