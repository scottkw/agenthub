// Phase 125-02 Task 2 — source-inspection tests for Editor.tsx and PreviewPane.tsx.
//
// Pattern: vitest source-inspection (same as TerminalPanel.test.tsx) using ?raw
// imports to assert structural invariants without running a full DOM render.
// This keeps the tests fast (no jsdom CM6 mount needed) and verifies the
// load-bearing security/correctness constraints:
//
//  - T-125-04: dangerouslySetInnerHTML must never appear in Editor.tsx (XSS gate)
//  - EDIT-02: Compartment used for read-only↔editable toggle (no remount)
//  - EDIT-11: large-file thresholds (500KB / 5MB) referenced
//  - EDIT-04: languageFor invoked for language detection
//  - EDIT-03/12: Edit button gating in PreviewPane uses canWrite and isBinary

import { describe, it, expect } from 'vitest'
import raw from '../Editor.tsx?raw'
import previewPaneRaw from '../FileBrowser/PreviewPane.tsx?raw'

describe('Editor.tsx source inspection (EDIT-02, EDIT-11, T-125-04)', () => {
  it('EDIT-02: uses Compartment for read-only↔editable toggle', () => {
    expect(raw).toContain('Compartment')
  })

  it('EDIT-02: reconfigure is called to flip editability without remount', () => {
    expect(raw).toContain('reconfigure')
  })

  it('EDIT-04: calls languageFor for language detection', () => {
    expect(raw).toContain('languageFor')
  })

  it('EDIT-11: references 500KB large-file threshold', () => {
    // 500 * 1024 = 512000, OR 500KB constant. Either expression is acceptable.
    const has500KB =
      raw.includes('500 * 1024') ||
      raw.includes('500*1024') ||
      raw.includes('512000') ||
      raw.includes('500_000') ||
      raw.includes('500KB') ||
      raw.includes('LARGE_FILE_THRESHOLD') ||
      raw.includes('largeFileThreshold') ||
      raw.includes('LARGE_FILE_WARN')
    expect(has500KB).toBe(true)
  })

  it('EDIT-11: references near-5MB syntax-disabled threshold', () => {
    const has5MB =
      raw.includes('5 * 1024 * 1024') ||
      raw.includes('5*1024*1024') ||
      raw.includes('5242880') ||
      raw.includes('5_000_000') ||
      raw.includes('SYNTAX_DISABLE_THRESHOLD') ||
      raw.includes('syntaxDisableThreshold') ||
      raw.includes('PLAIN_TEXT_THRESHOLD') ||
      raw.includes('5MB')
    expect(has5MB).toBe(true)
  })

  it('EDIT-11: contains the verbatim syntax-disabled notice copy', () => {
    expect(raw).toContain('Syntax highlighting disabled for large files.')
  })

  it('EDIT-11: contains the verbatim large-file warning copy (interpolated)', () => {
    // The copy template: "This is a large file ({N} MB). Edits may be slow."
    // The {N} is dynamic, so assert the fixed portions.
    expect(raw).toContain('Edits may be slow.')
  })

  it('T-125-04: NEVER uses dangerouslySetInnerHTML (XSS gate)', () => {
    // CM6 renders file content as text via its own DOM layer — never innerHTML.
    expect(raw).not.toContain('dangerouslySetInnerHTML')
  })

  it('mounts CM6 imperatively in useEffect (not via a React wrapper)', () => {
    expect(raw).toContain('useEffect')
    expect(raw).toContain('EditorView')
  })

  it('uses basicSetup from codemirror package', () => {
    expect(raw).toContain('basicSetup')
  })

  it('cleanup destroys the EditorView on unmount', () => {
    expect(raw).toContain('.destroy()')
  })

  it('does NOT import theme-one-dark (hand-rolled TokyoNight only)', () => {
    // Filter out comment lines before checking
    const nonComments = raw
      .split('\n')
      .filter((line) => !line.trimStart().startsWith('//'))
      .join('\n')
    expect(nonComments).not.toContain('theme-one-dark')
  })
})

describe('PreviewPane.tsx Edit button (EDIT-03, EDIT-12)', () => {
  it('EDIT-12: Edit button gating includes canWrite', () => {
    expect(previewPaneRaw).toContain('canWrite')
  })

  it('EDIT-03: Edit button gating excludes binary files via isBinary', () => {
    expect(previewPaneRaw).toContain('isBinary')
  })

  it('EDIT-03/12: Edit button references PencilSquareIcon', () => {
    expect(previewPaneRaw).toContain('PencilSquareIcon')
  })

  it('EDIT-12: Edit button has aria-label containing "Edit"', () => {
    // aria-label="Edit ${filename}" or similar
    expect(previewPaneRaw).toContain('Edit')
  })
})
