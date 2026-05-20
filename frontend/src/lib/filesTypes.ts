// Phase 120 — shared types for FileBrowser components.
// This module declares the type vocabulary shared by the FileListPane,
// BreadcrumbBar, StatusLine, and PreviewPane (Plan 04).
//
// NOTE (Wave 2 parallel execution): a richer version of this file ships in
// Plan 02 (PreviewState discriminated union). This file provides ONLY the
// types Plan 03 components depend on. When Wave 1 merges, Plan 02's content
// supersedes this file — the SortKey / SortDir / BreadcrumbSegment exports
// must remain byte-identical.

export type SortKey = 'name' | 'size' | 'modified'
export type SortDir = 'asc' | 'desc'

export interface BreadcrumbSegment {
  /** Display name of the segment (e.g. "src", "components"). */
  name: string
  /**
   * Relative path from session cwd to this segment.
   * "" for the root segment.
   * "src/components" for nested segments.
   */
  pathFromCwd: string
}
