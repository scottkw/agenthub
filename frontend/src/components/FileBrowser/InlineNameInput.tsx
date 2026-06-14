// Phase 125-04 Task 1 — Inline name-input row for create-file / mkdir / rename.
//
// Renders an input row inline within the file list. The input commits on
// Enter and cancels on Escape (UI-SPEC §5 Keyboard affordances).
//
// Modes:
//   'create-file'  — new empty file; placeholder "Filename…" (UI-SPEC locked string)
//   'new-folder'   — mkdir; placeholder "Folder name…" (UI-SPEC locked string)
//   'rename'       — in-place rename; prefilled with current name, text selected
//
// Callers:
//   - BreadcrumbBar New file / New folder buttons invoke create-file / new-folder
//   - FileRowActions Rename button invokes rename
//   - F2 keydown on selected row invokes rename
//
// All modes are gated: this component must only be rendered when canWrite is true
// (caller responsibility — InlineNameInput itself has no canWrite guard internally).

import React, { useEffect, useRef, useState } from 'react'

export type InlineNameMode = 'create-file' | 'new-folder' | 'rename'

export interface InlineNameInputProps {
  /** The mode determines the placeholder and the operation called on commit. */
  mode: InlineNameMode
  /** For rename mode: the current filename (prefilled and selected on mount). */
  initialValue?: string
  /**
   * Called when the user commits (Enter) with a non-empty trimmed name.
   * Receives the trimmed value. Returning a rejected Promise (e.g. 409) is
   * handled by the caller — InlineNameInput does not close on its own;
   * the caller must call onCancel or re-render after handling the error.
   */
  onCommit: (name: string) => Promise<void> | void
  /** Called when the user cancels (Escape) or the input loses focus without commit. */
  onCancel: () => void
}

const PLACEHOLDERS: Record<InlineNameMode, string> = {
  'create-file': 'Filename…',
  'new-folder': 'Folder name…',
  // Rename: no placeholder — current name is prefilled and selected.
  rename: '',
}

/**
 * Inline input row for create-file / mkdir / rename affordances.
 *
 * Focuses and selects the input on mount. Enter commits (trimmed, non-empty);
 * Escape cancels. Blur (focus-away) also cancels to prevent abandoned inputs.
 */
export function InlineNameInput({
  mode,
  initialValue = '',
  onCommit,
  onCancel,
}: InlineNameInputProps): React.ReactElement {
  const inputRef = useRef<HTMLInputElement>(null)
  const [value, setValue] = useState(initialValue)
  const [committing, setCommitting] = useState(false)
  // Track whether we committed so the blur handler doesn't double-cancel.
  const committedRef = useRef(false)

  // Focus and select all on mount.
  useEffect(() => {
    const el = inputRef.current
    if (!el) return
    el.focus()
    el.select()
  }, [])

  async function commit() {
    const trimmed = value.trim()
    if (!trimmed) {
      onCancel()
      return
    }
    if (committing) return
    committedRef.current = true
    setCommitting(true)
    try {
      await onCommit(trimmed)
    } finally {
      setCommitting(false)
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') {
      e.preventDefault()
      e.stopPropagation()
      void commit()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      onCancel()
    }
  }

  function handleBlur() {
    // Only cancel on blur if we haven't already committed.
    if (!committedRef.current) {
      onCancel()
    }
  }

  return (
    <li
      className="file-browser__list-row file-browser__list-row--inline-input"
      data-testid="file-browser-inline-name-input"
    >
      <input
        ref={inputRef}
        type="text"
        className="file-browser__inline-name-input"
        aria-label={
          mode === 'rename'
            ? 'Rename to'
            : mode === 'new-folder'
              ? 'New folder name'
              : 'New file name'
        }
        placeholder={PLACEHOLDERS[mode]}
        value={value}
        disabled={committing}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={handleBlur}
        autoComplete="off"
        spellCheck={false}
      />
    </li>
  )
}
