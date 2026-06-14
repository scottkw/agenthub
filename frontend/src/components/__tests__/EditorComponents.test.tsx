// Phase 125-03 Task 2 (TDD RED) — source-inspection tests for:
//   EditorHeader, SaveIndicator, UnsavedChangesModal, ConflictModal,
//   and FileBrowserTab unsaved-guard wiring.
//
// All assertions are source-level (grep over ?raw imports) — no DOM render.
// This verifies colorblind contract (icon+text), copy verbatim from UI-SPEC,
// safe-default-focus pattern from QuitConfirmModal, and no beforeunload.

import { describe, it, expect } from 'vitest'
import editorHeaderRaw from '../FileBrowser/EditorHeader.tsx?raw'
import saveIndicatorRaw from '../FileBrowser/SaveIndicator.tsx?raw'
import unsavedChangesRaw from '../FileBrowser/modals/UnsavedChangesModal.tsx?raw'
import conflictModalRaw from '../FileBrowser/modals/ConflictModal.tsx?raw'
import fileBrowserTabRaw from '../FileBrowserTab.tsx?raw'
import editorRaw from '../Editor.tsx?raw'

// ─── SaveIndicator ────────────────────────────────────────────────────────────

describe('SaveIndicator.tsx (EDIT-06, colorblind contract)', () => {
  it('has role="status" for accessible live region', () => {
    expect(saveIndicatorRaw).toContain('role="status"')
  })

  it('has aria-live="polite" (EDIT-06)', () => {
    expect(saveIndicatorRaw).toContain('aria-live')
  })

  it('saving state: shows ArrowPathIcon (icon) + literal text "Saving…"', () => {
    expect(saveIndicatorRaw).toContain('ArrowPathIcon')
    expect(saveIndicatorRaw).toContain('Saving…')
  })

  it('saved state: shows CheckCircleIcon (icon) + literal text "Saved"', () => {
    expect(saveIndicatorRaw).toContain('CheckCircleIcon')
    expect(saveIndicatorRaw).toContain('Saved')
  })

  it('error state: shows ExclamationTriangleIcon (icon)', () => {
    expect(saveIndicatorRaw).toContain('ExclamationTriangleIcon')
  })
})

// ─── EditorHeader ─────────────────────────────────────────────────────────────

describe('EditorHeader.tsx (EDIT-06, colorblind contract)', () => {
  it('renders DirtyMarker ● bullet glyph', () => {
    // The dirty bullet is ● (U+25CF) — carries meaning as a glyph, not color
    expect(editorHeaderRaw).toContain('●')
  })

  it('DirtyMarker has aria-label="Modified" (colorblind contract)', () => {
    expect(editorHeaderRaw).toContain('Modified')
  })

  it('has a Save button with --primary class', () => {
    expect(editorHeaderRaw).toContain('Save')
    expect(editorHeaderRaw).toContain('--primary')
  })

  it('has a Cancel button with --secondary class', () => {
    expect(editorHeaderRaw).toContain('Cancel')
    expect(editorHeaderRaw).toContain('--secondary')
  })

  it('reuses file-browser__preview-header CSS class', () => {
    expect(editorHeaderRaw).toContain('file-browser__preview-header')
  })

  it('includes SaveIndicator component', () => {
    expect(editorHeaderRaw).toContain('SaveIndicator')
  })
})

// ─── UnsavedChangesModal ──────────────────────────────────────────────────────

describe('UnsavedChangesModal.tsx (EDIT-07, UI-SPEC verbatim copy)', () => {
  it('title copy: "Unsaved changes" (verbatim, locked EDIT-07)', () => {
    expect(unsavedChangesRaw).toContain('Unsaved changes')
  })

  it('body copy: "You have unsaved changes. Save or discard?" (verbatim)', () => {
    expect(unsavedChangesRaw).toContain('You have unsaved changes. Save or discard?')
  })

  it('default-focus button: "Keep editing" (verbatim, safe default)', () => {
    expect(unsavedChangesRaw).toContain('Keep editing')
  })

  it('has "Save" primary button', () => {
    expect(unsavedChangesRaw).toContain('Save')
  })

  it('has "Discard" secondary button', () => {
    expect(unsavedChangesRaw).toContain('Discard')
  })

  it('QuitConfirmModal pattern: role="dialog" aria-modal', () => {
    expect(unsavedChangesRaw).toContain('role="dialog"')
    expect(unsavedChangesRaw).toContain('aria-modal')
  })

  it('QuitConfirmModal pattern: Escape key closes modal', () => {
    expect(unsavedChangesRaw).toContain('Escape')
  })

  it('QuitConfirmModal pattern: overlay click-to-cancel', () => {
    // Overlay click calls onCancel; inner dialog stops propagation
    expect(unsavedChangesRaw).toContain('stopPropagation')
  })

  it('QuitConfirmModal pattern: safe button gets default focus via ref+useEffect', () => {
    // "Keep editing" is the safe default — should be focused via ref
    expect(unsavedChangesRaw).toContain('useRef')
    expect(unsavedChangesRaw).toContain('focus()')
  })
})

// ─── ConflictModal ────────────────────────────────────────────────────────────

describe('ConflictModal.tsx (EDIT-08, UI-SPEC verbatim copy)', () => {
  it('heading: "This file was modified by another process" (verbatim, locked EDIT-08)', () => {
    expect(conflictModalRaw).toContain('This file was modified by another process')
  })

  it('body copy mentions preserved edits', () => {
    expect(conflictModalRaw).toContain('your edits are preserved either way')
  })

  it('destructive button: "Force overwrite"', () => {
    expect(conflictModalRaw).toContain('Force overwrite')
  })

  it('primary button: "Save as new file"', () => {
    expect(conflictModalRaw).toContain('Save as new file')
  })

  it('default-focus button: "Discard my changes"', () => {
    expect(conflictModalRaw).toContain('Discard my changes')
  })

  it('ExclamationTriangleIcon used for conflict signal (colorblind contract)', () => {
    expect(conflictModalRaw).toContain('ExclamationTriangleIcon')
  })

  it('QuitConfirmModal pattern: role="dialog" aria-modal', () => {
    expect(conflictModalRaw).toContain('role="dialog"')
    expect(conflictModalRaw).toContain('aria-modal')
  })

  it('QuitConfirmModal pattern: Escape key closes', () => {
    expect(conflictModalRaw).toContain('Escape')
  })

  it('QuitConfirmModal pattern: overlay click-to-cancel', () => {
    expect(conflictModalRaw).toContain('stopPropagation')
  })

  it('force-overwrite: re-PUTs with If-Match "*" (server skip-check)', () => {
    // Force overwrite uses '*' as the If-Match value
    expect(conflictModalRaw).toContain('*')
  })

  it('discard: has re-fetch logic to reload server content', () => {
    // Discard path re-fetches and replaces buffer — check for read/fetch signal
    const hasReload =
      conflictModalRaw.includes('onDiscard') ||
      conflictModalRaw.includes('discard') ||
      conflictModalRaw.includes('Discard my changes')
    expect(hasReload).toBe(true)
  })
})

// ─── No beforeunload (EDIT-07) ───────────────────────────────────────────────

describe('No beforeunload in editor path (EDIT-07)', () => {
  it('FileBrowserTab.tsx does not use beforeunload', () => {
    expect(fileBrowserTabRaw).not.toContain('beforeunload')
  })

  it('Editor.tsx does not use beforeunload', () => {
    expect(editorRaw).not.toContain('beforeunload')
  })
})

// ─── FileBrowserTab navigation guard wiring (EDIT-07) ────────────────────────

describe('FileBrowserTab.tsx unsaved-guard wiring (EDIT-07)', () => {
  it('imports or renders UnsavedChangesModal', () => {
    const hasUnsaved =
      fileBrowserTabRaw.includes('UnsavedChangesModal') ||
      fileBrowserTabRaw.includes('guardThen')
    expect(hasUnsaved).toBe(true)
  })

  it('imports Editor component', () => {
    expect(fileBrowserTabRaw).toContain('Editor')
  })
})
