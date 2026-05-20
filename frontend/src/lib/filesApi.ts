// Phase 120 — typed API client interface for the daemon /api/files/* surface.
//
// NOTE (Wave 2 parallel execution): the FULL implementation (FilesApiClient
// class + FilesApiError class + fetch wiring) ships in Plan 02 (Wave 1). This
// file currently exports only the shared TS shapes that Plan 03 components
// import (FileEntry — used by FileListPane and FileRow). When Wave 1 merges,
// Plan 02's file supersedes this — the FileEntry export must remain
// byte-identical.

export interface FileEntry {
  name: string
  size: number
  mtime: string
  mode: number
  isDir: boolean
  isSymlink: boolean
  isBinary: boolean
  mime?: string
}
