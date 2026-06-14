// Phase 125-02 Task 1 — extension → CM6 LanguageSupport mapper.
//
// Returns a CM6 Extension (LanguageSupport or StreamLanguage) for the given
// filename, suitable for use inside a Compartment.
//
// Strategy:
//   - Bash/shell/zsh: @codemirror/legacy-modes StreamLanguage (no native Lezer
//     shell grammar exists — @codemirror/lang-shell does NOT exist on npm,
//     verified RESEARCH Open Q2).
//   - Everything else: @codemirror/language-data registry (.find() + .load()),
//     which covers 120+ languages via lazy dynamic imports (Vite code-splits them).
//   - Unknown extension: return [] (plain text, no language pack).

import { languages } from '@codemirror/language-data'
import { StreamLanguage } from '@codemirror/language'
import type { Extension } from '@codemirror/state'
import { shell } from '@codemirror/legacy-modes/mode/shell'

/**
 * Returns a CM6 language extension for the given filename, or `[]` (plain text)
 * when the extension is unknown.
 *
 * The returned value is safe to pass to `language.of(...)` inside a Compartment.
 */
export async function languageFor(filename: string): Promise<Extension> {
  const ext = filename.split('.').pop()?.toLowerCase() ?? ''

  // Bash / shell — use legacy-modes StreamLanguage (no native Lezer grammar).
  if (['sh', 'bash', 'zsh'].includes(ext)) {
    return StreamLanguage.define(shell)
  }

  // Delegate to language-data registry: lazy dynamic import per language.
  // .find() matches on extensions[] or filename regex (e.g. Makefile, Dockerfile).
  const desc = languages.find(
    (l) => l.extensions?.includes(ext) || l.filename?.test(filename),
  )
  if (!desc) return [] // unknown extension → plain text

  const support = await desc.load() // dynamic import; Vite code-splits
  return support
}
