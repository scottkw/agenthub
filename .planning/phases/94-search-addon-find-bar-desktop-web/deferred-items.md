# Deferred Items — Phase 94

Out-of-scope discoveries surfaced during plan execution. Per scope-boundary
rule, these are NOT fixed in 94-* plans because they are not caused by Phase
94 changes. Tracked here for the next planning cycle.

## From 94-06 (gap-closure: find-bar animation wiring)

### Sidebar.test.tsx — 20 pre-existing failures

**File:** `frontend/src/components/__tests__/Sidebar.test.tsx`
**Status:** failures exist on `main` BEFORE 94-06 changes (verified via
`git stash` + run). Unrelated to FindBar / SearchAddon / TerminalPanel.
**Impact:** does NOT regress because of 94-06; the failure surface is
entirely in the Sidebar nav component.
**Recommended next step:** open a tracking task for a future hygiene plan
(no phase yet) — not appropriate to fix mid-Phase-94 because Sidebar is
out of Phase 94's scope (Phase 94 = SearchAddon find bar).

The Phase 94 sweep (`pnpm test src/components/FindBar src/lib/isXtermFocused
src/__tests__/App.plugin-event src/components/__tests__/TerminalPanel.search
src/components/__tests__/TerminalPanel.search.exit src/components/__tests__/
PluginsSection`) passes 95/95.
