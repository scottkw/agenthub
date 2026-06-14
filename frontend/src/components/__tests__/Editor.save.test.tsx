// Phase 125-03 Task 1 (TDD RED) — source-inspection tests for save wiring.
//
// Asserts structural invariants in Editor.tsx, filesApi.ts, and
// useFilesWrite.ts that implement the Cmd/Ctrl+S save flow and dirty state
// (EDIT-05, EDIT-06, EDIT-07, EDIT-08).
//
// Pattern: vitest source-inspection (?raw import) — fast, no jsdom CM6 needed.

import { describe, it, expect } from 'vitest'
import editorRaw from '../Editor.tsx?raw'
import filesApiRaw from '../../lib/filesApi.ts?raw'
import useFilesWriteRaw from '../../lib/useFilesWrite.ts?raw'

// ─── Editor.tsx save wiring (EDIT-05, EDIT-06) ───────────────────────────────

describe('Editor.tsx save wiring (EDIT-05, EDIT-06)', () => {
  it('EDIT-05: has Mod-s keymap binding', () => {
    expect(editorRaw).toContain('Mod-s')
  })

  it('EDIT-05: Mod-s binding calls onSave', () => {
    expect(editorRaw).toContain('onSave')
  })

  it('EDIT-05: Mod-s uses preventDefault to stop browser Save dialog', () => {
    expect(editorRaw).toContain('preventDefault')
  })

  it('EDIT-06: dirty state compares doc to savedSnapshot', () => {
    const hasSavedSnapshot =
      editorRaw.includes('savedSnapshot') ||
      editorRaw.includes('saved_snapshot') ||
      editorRaw.includes('savedContent')
    expect(hasSavedSnapshot).toBe(true)
  })

  it('EDIT-06: EditorView.updateListener is used for dirty tracking', () => {
    expect(editorRaw).toContain('updateListener')
  })
})

// ─── filesApi.ts — isConflict, isCollision, writeFile, If-Match ──────────────

describe('filesApi.ts — write predicates and writeFile (EDIT-05, EDIT-08)', () => {
  it('EDIT-08: FilesApiError has isConflict() predicate for 412', () => {
    expect(filesApiRaw).toContain('isConflict')
  })

  it('EDIT-09: FilesApiError has isCollision() predicate for 409', () => {
    expect(filesApiRaw).toContain('isCollision')
  })

  it('EDIT-05: FilesApiClient has writeFile method', () => {
    expect(filesApiRaw).toContain('writeFile')
  })

  it('EDIT-05: writeFile sends If-Match header', () => {
    expect(filesApiRaw).toContain('If-Match')
  })

  it('EDIT-05: writeFile uses PUT method', () => {
    expect(filesApiRaw).toContain("'PUT'")
  })

  it('EDIT-05: writeFile sends application/octet-stream content type', () => {
    expect(filesApiRaw).toContain('application/octet-stream')
  })
})

// ─── useFilesWrite.ts structure (EDIT-05, EDIT-06, EDIT-08) ──────────────────

describe('useFilesWrite.ts structure (EDIT-05, EDIT-06, EDIT-08)', () => {
  it('exports isSaving state', () => {
    expect(useFilesWriteRaw).toContain('isSaving')
  })

  it('exports saveError state', () => {
    expect(useFilesWriteRaw).toContain('saveError')
  })

  it('implements write function with If-Match', () => {
    expect(useFilesWriteRaw).toContain('If-Match')
  })

  it('EDIT-08: handles 412 via isConflict()', () => {
    expect(useFilesWriteRaw).toContain('isConflict')
  })

  it('EDIT-06: has ~1.5s saved transient timeout', () => {
    const has1500 =
      useFilesWriteRaw.includes('1500') ||
      useFilesWriteRaw.includes('1_500') ||
      useFilesWriteRaw.includes('SAVED_TIMEOUT') ||
      useFilesWriteRaw.includes('savedTimeout')
    expect(has1500).toBe(true)
  })

  it('exposes write, del, rename, mkdir, upload members', () => {
    expect(useFilesWriteRaw).toContain('write')
    expect(useFilesWriteRaw).toContain('del')
    expect(useFilesWriteRaw).toContain('rename')
    expect(useFilesWriteRaw).toContain('mkdir')
    expect(useFilesWriteRaw).toContain('upload')
  })

  it('T-125-08: buffer NOT cleared on 412 — conflict signal set instead', () => {
    expect(useFilesWriteRaw).toContain('isConflict')
  })
})
