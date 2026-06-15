// Phase 125-02 Task 2 — CodeMirror 6 editor component.
//
// Mounts CM6 imperatively in a useEffect (NOT via @uiw/react-codemirror wrapper
// — that hides the Compartment API needed for EDIT-02 read-only↔editable toggle).
//
// Key design decisions (RESEARCH Anti-Patterns + PATTERNS.md):
//  - EditorView mounted once; stays alive while the component is mounted.
//  - `editable` Compartment toggles read-only↔editable via reconfigure().
//  - `language` Compartment set lazily from languageFor(filename).
//  - Initial doc = the `initialContent` prop (reuse preview bytes — NO re-fetch).
//  - Theme: hand-rolled EditorView.theme({...}) using TokyoNight tokens from
//    style.css. ZERO new hex values. @codemirror/theme-one-dark is NOT used.
//  - CM6 renders file content as TEXT via its own DOM layer (T-125-04 XSS gate).
//  - Large-file guards: >500KB → LargeFileNotice; near-5MB → plain-text mode.
//
// Save wiring (Cmd/Ctrl+S keymap + onSave/onDirty callbacks) is accepted as
// props with safe no-op defaults — implementation lands in Plan 03.

import React, { useEffect, useRef, useState } from 'react'
import { EditorView, keymap } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { indentWithTab } from '@codemirror/commands'
import { basicSetup } from 'codemirror'
import { languageFor } from '../lib/languageFor'

// ─── Size thresholds ─────────────────────────────────────────────────────────

/** Files > 500 KB show the LargeFileNotice warn-then-proceed banner (EDIT-11). */
const LARGE_FILE_WARN_THRESHOLD = 500 * 1024

/**
 * Files at or above this size mount in plain-text mode (no language pack) and
 * show the syntax-disabled caption (EDIT-11). The server read cap is 5 MiB
 * (maxPreviewBytes); this is the practical ceiling.
 */
const PLAIN_TEXT_THRESHOLD = 5 * 1024 * 1024

// ─── TokyoNight CM6 theme (hand-rolled — ZERO new hex values) ────────────────
//
// Every hex value below is already declared in style.css / UI-SPEC §Color.
// @codemirror/theme-one-dark is intentionally NOT used (introduces non-TokyoNight
// hexes; releases the colorblind contract guard — RESEARCH Open Q4 resolution).

const tokyoNightTheme = EditorView.theme(
  {
    // Editor chrome
    '&': {
      backgroundColor: '#1a1b26',
      color: '#c0caf5',
      fontFamily: 'monospace',
      fontSize: '13px',
      height: '100%',
    },
    '.cm-content': {
      caretColor: '#7aa2f7',
      padding: '0',
    },
    '.cm-cursor, .cm-dropCursor': {
      borderLeftColor: '#7aa2f7',
    },
    '.cm-focused .cm-cursor': {
      borderLeftColor: '#7aa2f7',
    },
    // Selection
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
      backgroundColor: '#283457',
    },
    // Gutter
    '.cm-gutters': {
      backgroundColor: '#16161e',
      color: '#9aa5ce',
      border: 'none',
      borderRight: '1px solid #292e42',
    },
    '.cm-activeLineGutter': {
      backgroundColor: '#1e2030',
      color: '#7aa2f7',
    },
    '.cm-activeLine': {
      backgroundColor: '#1e2030',
    },
    // Focus ring (accessibility)
    '&.cm-focused': {
      outline: '2px solid #7aa2f7',
      outlineOffset: '-2px',
    },
    // Scrollbar
    '.cm-scroller': {
      overflow: 'auto',
    },
  },
  { dark: true },
)

// ─── Props ────────────────────────────────────────────────────────────────────

export interface EditorProps {
  /** Filename (for language detection and aria-label). */
  filename: string
  /** Initial document text (reused from the already-loaded preview — no re-fetch). */
  initialContent: string
  /** File size in bytes (for large-file guards). */
  fileSize: number
  /**
   * Called when the document changes. Receives current doc string and dirty state.
   * No-op default — save wiring lands in Plan 03.
   */
  onDirty?: (dirty: boolean) => void
  /**
   * Called when Cmd/Ctrl+S is pressed with the current document content.
   * No-op default — save wiring lands in Plan 03.
   */
  onSave?: (content: string) => void
  /**
   * Called when the user clicks the Cancel edit button, or Escape in a modal.
   * Returns to read-only PreviewPane.
   */
  onCancel?: () => void
}

// ─── LargeFileNotice ─────────────────────────────────────────────────────────

interface LargeFileNoticeProps {
  sizeBytes: number
  onDismiss: () => void
}

function LargeFileNotice({ sizeBytes, onDismiss }: LargeFileNoticeProps): React.ReactElement {
  const sizeMB = (sizeBytes / (1024 * 1024)).toFixed(1)
  return (
    <div className="file-browser__editor-notice" role="alert" aria-live="assertive">
      {/* UI-SPEC verbatim copy (locked, EDIT-11): */}
      <span>This is a large file ({sizeMB} MB). Edits may be slow.</span>
      <button
        className="file-browser__btn file-browser__btn--icon"
        aria-label="Dismiss large file warning"
        onClick={onDismiss}
      >
        ✕
      </button>
    </div>
  )
}

// ─── Editor ──────────────────────────────────────────────────────────────────

export function Editor({
  filename,
  initialContent,
  fileSize,
  onDirty,
  onSave,
  onCancel: _onCancel,
}: EditorProps): React.ReactElement {
  const mountEl = useRef<HTMLDivElement | null>(null)
  const viewRef = useRef<EditorView | null>(null)
  const [largeDismissed, setLargeDismissed] = useState<boolean>(false)
  const [syntaxDisabled, setSyntaxDisabled] = useState<boolean>(false)

  const isLargeFile = fileSize >= LARGE_FILE_WARN_THRESHOLD
  const isSyntaxDisabledSize = fileSize >= PLAIN_TEXT_THRESHOLD

  // ─── CM6 mount ─────────────────────────────────────────────────────────────
  useEffect(() => {
    if (!mountEl.current) return

    const editableCompartment = new Compartment()
    const languageCompartment = new Compartment()

    const savedSnapshot = initialContent

    const view = new EditorView({
      parent: mountEl.current,
      state: EditorState.create({
        doc: initialContent,
        extensions: [
          basicSetup,
          tokyoNightTheme,
          // Start read-only; flip to editable when user clicks Edit button.
          editableCompartment.of([
            EditorView.editable.of(false),
            EditorState.readOnly.of(true),
          ]),
          languageCompartment.of([]),
          keymap.of([
            {
              key: 'Mod-s',
              preventDefault: true,
              run: () => {
                if (onSave) onSave(view.state.doc.toString())
                return true
              },
            },
            // 125-UI-SPEC:304 — Tab inserts indentation inside CM6 (not a focus
            // change). basicSetup intentionally omits this (CM6 a11y default),
            // so it must be added explicitly. Release-blocking parity item;
            // confirmed missing during the Phase 125 Wails WebView UAT.
            indentWithTab,
          ]),
          EditorView.updateListener.of((update) => {
            if (update.docChanged && onDirty) {
              onDirty(update.state.doc.toString() !== savedSnapshot)
            }
          }),
        ],
      }),
    })

    viewRef.current = view

    // Flip to editable (edit-enter: called via parent-controlled prop in Plan 03).
    view.dispatch({
      effects: editableCompartment.reconfigure([
        EditorView.editable.of(true),
        EditorState.readOnly.of(false),
      ]),
    })

    // Apply language pack lazily (unless near-5MB cap → plain text).
    if (!isSyntaxDisabledSize) {
      setSyntaxDisabled(false)
      void languageFor(filename).then((lang) => {
        if (viewRef.current === view && lang) {
          view.dispatch({
            effects: languageCompartment.reconfigure(lang),
          })
        }
      })
    } else {
      setSyntaxDisabled(true)
    }

    return () => {
      view.destroy()
      viewRef.current = null
    }
    // Intentionally only run on mount — filename/fileSize do not remount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <section
      role="region"
      aria-label={`Edit ${filename}`}
      className="file-browser__preview"
      data-testid="file-browser-preview"
    >
      {/* Large-file warning (>500KB, warn-then-proceed, non-blocking) */}
      {isLargeFile && !isSyntaxDisabledSize && !largeDismissed && (
        <LargeFileNotice sizeBytes={fileSize} onDismiss={() => setLargeDismissed(true)} />
      )}

      {/* Syntax-disabled caption (near 5MB) — UI-SPEC verbatim copy (locked, EDIT-11) */}
      {syntaxDisabled && (
        <div className="file-browser__editor-notice file-browser__editor-notice--info" role="status">
          Syntax highlighting disabled for large files.
        </div>
      )}

      {/* CM6 mount point — CM6 renders content as TEXT via its own DOM layer,
          never via innerHTML (T-125-04 XSS gate). */}
      <div
        ref={mountEl}
        className="file-browser__editor-mount"
        aria-label={`Code editor for ${filename}`}
        style={{ flex: 1, minHeight: 0 }}
      />
    </section>
  )
}
