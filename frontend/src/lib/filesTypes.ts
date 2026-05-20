// Phase 120-02 Task 3 — shared TS types for FileBrowserTab. Pure type module
// (no runtime code), imported by filesApi.ts, useFilesCapability.ts, and the
// Plan 03/04 components.

/** Column the file list is sorted by. */
export type SortKey = 'name' | 'size' | 'modified'

/** Sort direction. */
export type SortDir = 'asc' | 'desc'

/**
 * Breadcrumb segment for the path crumb trail at the top of FileBrowserTab.
 * `pathFromCwd` is the path relative to the session's cwd that, when clicked,
 * navigates the browser there (root crumb → '.').
 */
export interface BreadcrumbSegment {
  name: string
  pathFromCwd: string
}

/**
 * Discriminated union driving PreviewPane rendering. Each variant carries the
 * exact payload PreviewPane needs to render that state — Plan 04 pattern-matches
 * on `kind` to dispatch to the correct sub-component.
 *
 * Field names MUST match RESEARCH.md §"State dispatch in PreviewPane" verbatim
 * (Plan 04 will switch on these without further translation).
 */
export type PreviewState =
  | { kind: 'idle' }
  | { kind: 'loading' }
  | { kind: 'text'; text: string; size: number; mtime: string }
  | { kind: 'markdown'; text: string; size: number; mtime: string }
  | { kind: 'image'; url: string; size: number; mtime: string }
  | { kind: 'empty'; filename: string }
  | { kind: 'unsupported'; filename: string; downloadUrl: string; humanSize: string }
  | { kind: 'over-cap'; filename: string; downloadUrl: string; humanSize: string }
  | { kind: 'broken-symlink'; filename: string; targetPath: string }
  | { kind: 'read-error'; filename: string; message: string; onRetry: () => void }
  | { kind: 'forbidden-file'; filename: string }
