// Phase 94 SRC-01 — focus-conditioning helper.
//
// Returns true iff document.activeElement is contained within termContainer.
// Used by TerminalPanel's Cmd-F handler to gate find-bar opening so that
// browser-native Cmd-F passes through when focus is on a sibling element
// (sidebar, modal, settings panel) — see RESEARCH §"Pattern 2 —
// Focus-conditioned Cmd-F" + Pitfall #1 ("modal-sibling activeElement").
//
// Pure: no React, no DOM mutation, no globals besides `document`. Safe to
// import from any module; jsdom provides `document.activeElement` in tests.
export function isXtermFocused(termContainer: HTMLElement | null): boolean {
  if (!termContainer || !document.activeElement) return false
  return termContainer.contains(document.activeElement)
}
