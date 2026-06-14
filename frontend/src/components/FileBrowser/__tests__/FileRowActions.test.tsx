// Phase 125-04 Task 1 (TDD RED) — source-inspection tests for write affordances.
//
// Asserts structural invariants in:
//   - filesApi.ts: del/rename/mkdir methods + oldRel
//   - useFilesWrite.ts: real del/rename/mkdir implementations (not stubs)
//   - InlineNameInput.tsx: Enter/Escape, "Filename…" placeholder
//   - FileRow.tsx: canWrite gating on FileRowActions
//   - BreadcrumbBar.tsx: New file / New folder canWrite buttons
//
// Also asserts DELETE / COLLISION / MOVETO modals (Task 2 targets):
//   - DeleteConfirmModal.tsx: "files inside" text for recursive-dir count
//   - CollisionConfirmModal.tsx: "already exists" + Cancel default focus
//   - MoveToPickerModal.tsx: "Move here" primary
//
// Pattern: vitest source-inspection (?raw import) — fast, no jsdom needed.

import { describe, it, expect } from 'vitest'
import filesApiRaw from '../../../lib/filesApi.ts?raw'
import useFilesWriteRaw from '../../../lib/useFilesWrite.ts?raw'

// ─── filesApi.ts — del / rename / mkdir ──────────────────────────────────────

describe('filesApi.ts — del/rename/mkdir (EDIT-09)', () => {
  it('has del method using DELETE', () => {
    expect(filesApiRaw).toContain('del(')
    expect(filesApiRaw).toContain("'DELETE'")
  })

  it('has rename method posting {oldRel, newRel} JSON', () => {
    expect(filesApiRaw).toContain('rename(')
    expect(filesApiRaw).toContain('oldRel')
    expect(filesApiRaw).toContain('newRel')
  })

  it('has mkdir method using POST', () => {
    expect(filesApiRaw).toContain('mkdir(')
  })

  it('rename uses POST method', () => {
    expect(filesApiRaw).toContain("'POST'")
  })
})

// ─── useFilesWrite.ts — real del/rename/mkdir (no stub throws) ──────────────

describe('useFilesWrite.ts — real del/rename/mkdir implementations (EDIT-09)', () => {
  it('del calls client.del (not stub throw)', () => {
    expect(useFilesWriteRaw).toContain('client.del')
  })

  it('rename calls client.rename (not stub throw)', () => {
    expect(useFilesWriteRaw).toContain('client.rename')
  })

  it('mkdir calls client.mkdir (not stub throw)', () => {
    expect(useFilesWriteRaw).toContain('client.mkdir')
  })

  it('del is no longer the stub (no "Plan 04" throw)', () => {
    // The stub threw 'del not implemented (Plan 04)' — verify it's gone
    expect(useFilesWriteRaw).not.toContain('del not implemented')
  })

  it('rename is no longer the stub', () => {
    expect(useFilesWriteRaw).not.toContain('rename not implemented')
  })

  it('mkdir is no longer the stub', () => {
    expect(useFilesWriteRaw).not.toContain('mkdir not implemented')
  })
})

// ─── InlineNameInput.tsx ─────────────────────────────────────────────────────

describe('InlineNameInput.tsx — Enter/Escape + placeholders (EDIT-09)', () => {
  it('contains InlineNameInput export', async () => {
    const { default: raw } = await import('../InlineNameInput.tsx?raw')
    expect(raw).toContain('InlineNameInput')
  })

  it('commits on Enter key', async () => {
    const { default: raw } = await import('../InlineNameInput.tsx?raw')
    expect(raw).toContain('Enter')
  })

  it('cancels on Escape key', async () => {
    const { default: raw } = await import('../InlineNameInput.tsx?raw')
    expect(raw).toContain('Escape')
  })

  it('has "Filename…" placeholder (verbatim UI-SPEC)', async () => {
    const { default: raw } = await import('../InlineNameInput.tsx?raw')
    expect(raw).toContain('Filename…')
  })

  it('has "Folder name…" placeholder (verbatim UI-SPEC)', async () => {
    const { default: raw } = await import('../InlineNameInput.tsx?raw')
    expect(raw).toContain('Folder name…')
  })
})

// ─── FileRow.tsx — canWrite gating on FileRowActions ─────────────────────────

describe('FileRow.tsx — canWrite-gated FileRowActions (EDIT-09/12)', () => {
  it('references canWrite prop', async () => {
    const { default: raw } = await import('../FileRow.tsx?raw')
    expect(raw).toContain('canWrite')
  })

  it('has Edit action gated on !isDir && !isBinary', async () => {
    const { default: raw } = await import('../FileRow.tsx?raw')
    // Edit requires canWrite AND !isDir AND !isBinary (UI-SPEC §1)
    const hasGate =
      raw.includes('isDir') && raw.includes('isBinary') && raw.includes('canWrite')
    expect(hasGate).toBe(true)
  })

  it('has Delete action button', async () => {
    const { default: raw } = await import('../FileRow.tsx?raw')
    expect(raw).toContain('Delete')
  })

  it('has Rename action button', async () => {
    const { default: raw } = await import('../FileRow.tsx?raw')
    expect(raw).toContain('Rename')
  })

  it('has Move action button', async () => {
    const { default: raw } = await import('../FileRow.tsx?raw')
    expect(raw).toContain('Move')
  })
})

// ─── BreadcrumbBar.tsx — New file / New folder toolbar buttons ───────────────

describe('BreadcrumbBar.tsx — New file/folder toolbar buttons (EDIT-09/12)', () => {
  it('has canWrite prop', async () => {
    const { default: raw } = await import('../BreadcrumbBar.tsx?raw')
    expect(raw).toContain('canWrite')
  })

  it('has "New file" label/tooltip (verbatim UI-SPEC)', async () => {
    const { default: raw } = await import('../BreadcrumbBar.tsx?raw')
    expect(raw).toContain('New file')
  })

  it('has "New folder" label/tooltip (verbatim UI-SPEC)', async () => {
    const { default: raw } = await import('../BreadcrumbBar.tsx?raw')
    expect(raw).toContain('New folder')
  })

  it('has DocumentPlusIcon for New file', async () => {
    const { default: raw } = await import('../BreadcrumbBar.tsx?raw')
    expect(raw).toContain('DocumentPlusIcon')
  })

  it('has FolderPlusIcon for New folder', async () => {
    const { default: raw } = await import('../BreadcrumbBar.tsx?raw')
    expect(raw).toContain('FolderPlusIcon')
  })
})

// ─── DeleteConfirmModal.tsx — "files inside" + recursive count ───────────────

describe('DeleteConfirmModal.tsx — recursive-dir count (EDIT-09)', () => {
  it('contains "files inside" for recursive-dir count (EDIT-09 locked string)', async () => {
    const { default: raw } = await import('../modals/DeleteConfirmModal.tsx?raw')
    expect(raw).toContain('files inside')
  })

  it('has Cancel button with default focus (safe action)', async () => {
    const { default: raw } = await import('../modals/DeleteConfirmModal.tsx?raw')
    // Must have Cancel as default-focused button (QuitConfirmModal pattern)
    expect(raw).toContain('Cancel')
    expect(raw).toContain('cancelBtnRef')
  })

  it('has destructive Delete button (not default focused)', async () => {
    const { default: raw } = await import('../modals/DeleteConfirmModal.tsx?raw')
    expect(raw).toContain('Delete')
  })

  it('has TrashIcon glyph (colorblind-safe)', async () => {
    const { default: raw } = await import('../modals/DeleteConfirmModal.tsx?raw')
    expect(raw).toContain('TrashIcon')
  })
})

// ─── CollisionConfirmModal.tsx — "already exists" + Cancel default focus ─────

describe('CollisionConfirmModal.tsx — 409 replace modal (EDIT-09/10)', () => {
  it('contains "already exists" body text (locked string)', async () => {
    const { default: raw } = await import('../modals/CollisionConfirmModal.tsx?raw')
    expect(raw).toContain('already exists')
  })

  it('has Cancel as default focus (locked — safe-default invariant)', async () => {
    const { default: raw } = await import('../modals/CollisionConfirmModal.tsx?raw')
    expect(raw).toContain('cancelBtnRef')
  })

  it('has Replace destructive button', async () => {
    const { default: raw } = await import('../modals/CollisionConfirmModal.tsx?raw')
    expect(raw).toContain('Replace')
  })

  it('Cancel button is rendered (verbatim UI-SPEC)', async () => {
    const { default: raw } = await import('../modals/CollisionConfirmModal.tsx?raw')
    expect(raw).toContain('Cancel')
  })
})

// ─── MoveToPickerModal.tsx — "Move here" primary ─────────────────────────────

describe('MoveToPickerModal.tsx — cross-directory move picker (EDIT-09)', () => {
  it('contains "Move here" primary button (verbatim UI-SPEC)', async () => {
    const { default: raw } = await import('../modals/MoveToPickerModal.tsx?raw')
    expect(raw).toContain('Move here')
  })

  it('has "Move to…" in title (verbatim UI-SPEC)', async () => {
    const { default: raw } = await import('../modals/MoveToPickerModal.tsx?raw')
    expect(raw).toContain('Move')
  })

  it('calls rename with both paths (cross-dir move = rename)', async () => {
    const { default: raw } = await import('../modals/MoveToPickerModal.tsx?raw')
    expect(raw).toContain('rename')
  })
})
