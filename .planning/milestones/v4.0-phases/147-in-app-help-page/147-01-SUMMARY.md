---
phase: 147
plan: "01"
subsystem: frontend-test-infra
tags: [help-page, tdd, test-stubs, markdown-content, polyfill]
dependency_graph:
  requires: []
  provides: [help-content-md, intersection-observer-polyfill, rehype-sanitize, help-test-stubs]
  affects: [frontend/src/test-setup.ts, frontend/package.json, frontend/src/content/help/]
tech_stack:
  added: [rehype-sanitize@6.0.0]
  patterns: [readFileSync-source-gate, vitest-red-stub, polyfill-as-unknown-cast]
key_files:
  created:
    - frontend/src/content/help/getting-started.md
    - frontend/src/content/help/faq.md
    - frontend/src/components/__tests__/HelpTab.test.tsx
    - frontend/src/components/__tests__/HelpSearch.test.tsx
    - frontend/src/components/__tests__/HelpSectionNav.test.tsx
    - frontend/src/components/__tests__/HelpContent.test.tsx
  modified:
    - frontend/src/test-setup.ts
    - frontend/package.json
    - frontend/pnpm-lock.yaml
    - frontend/src/components/__tests__/Sidebar.test.tsx
    - TESTING.md
decisions:
  - "SidebarTestProps extended type used to allow onOpenHelp before Plan 02 adds it to SidebarProps — cast removed when Plan 02 makes it a real prop (GREEN signal)"
  - "HelpSearch RED stubs: null-component catch pattern causes 2 negative empty-state tests to pass early (toBeNull assertions on non-rendered output); this is expected and acceptable — the positive assertions all fail"
  - "Sidebar count assertion approach: THREE count assertions at lines ~92/~238/~379 changed 3→4; group-li count at line ~468 stays 3 (All + 2 groups)"
metrics:
  duration: ~35 minutes
  completed: "2026-06-22"
  tasks_completed: 4
  files_count: 11
---

# Phase 147 Plan 01: Help Page Wave 0 Infrastructure Summary

**One-liner:** IntersectionObserver polyfill + rehype-sanitize install + Getting Started/FAQ Markdown seed + four failing RED test stubs for all Help components.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Verify rehype-sanitize legitimacy (human pre-approved) | (pre-approved) | — |
| 2 | Install rehype-sanitize + add IntersectionObserver polyfill | f1ed10be | frontend/src/test-setup.ts, package.json, pnpm-lock.yaml |
| 3 | Create seeded Help Markdown content | 84ab439c | frontend/src/content/help/getting-started.md, faq.md |
| 4 | Update Sidebar tests + create four failing Help stubs | 00fe2c8e | Sidebar.test.tsx, HelpTab/Search/SectionNav/Content.test.tsx, TESTING.md |

## What Was Built

### Task 2: rehype-sanitize + IntersectionObserver polyfill
- `pnpm add rehype-sanitize@6.0.0` installed (supply-chain gate T-147-SC pre-approved by user)
- `frontend/src/test-setup.ts` extended with an IntersectionObserver no-op polyfill after the existing ResizeObserver block, using `as unknown as typeof IntersectionObserver` cast (stub omits callback+options constructor signature)
- Existing 1825 vitest tests: all pass, no regressions

### Task 3: Seeded Markdown content
- `frontend/src/content/help/getting-started.md`: 7 blank-line-separated paragraphs (> 20 chars each) covering session creation, shells, session switching, file browser, web sharing, and settings
- `frontend/src/content/help/faq.md`: All 6 seed questions from UI-SPEC Copywriting Contract (DevTools-in-prod, network sharing, remote file browse, logs location, updating, reporting bugs); DevTools answer references `wails dev`, web-share to Chrome, and `~/Library/Application Support/agenthub/`
- GitHub Issues link in the "report a bug" answer (the only permitted Markdown link per plan)
- No `keyboard-shortcuts.md` (omitted per A2 — no documented shortcuts in v1)

### Task 4: Sidebar updates + 4 failing test stubs

**Sidebar.test.tsx:**
- 3 count assertions (lines ~92, ~238, ~379): `3 → 4` (Home, Hub, Settings, Help)
- Group-li count assertion (line ~468) stays 3 — the All + 2 named groups `<li>` count, not sidebar items
- `SidebarTestProps` extended type bridges the pre-Plan-02 `onOpenHelp` prop gap
- `onOpenHelp: vi.fn()` added to `renderSidebar` defaultProps
- New `describe('Sidebar Help item (Phase 147)')` block: 4 assertions mirroring the Hub block

**HelpTab.test.tsx (RED):**
- Source gates: `HELP_TAB`, `id: '__help__'`, `type: 'help'`, `handleOpenHelp` in App.tsx
- CSS gates: `--hub-search-highlight-bg` in `:root` and `[data-ui-theme="light"]`

**HelpSearch.test.tsx (RED):**
- Visible label "Search help…"; clear button `aria-label="Clear search"`
- Empty-state `.help-search__empty` with query interpolation
- `<mark class="help-search__mark">` around matched substring

**HelpSectionNav.test.tsx (RED):**
- Renders buttons per section (Getting Started + FAQ minimum)
- `aria-current="true"` and `.help-nav__link--active` on active section
- `onSectionChange` fires with section id on click

**HelpContent.test.tsx (RED):**
- Source gate: `HelpContent.tsx` imports `react-markdown` and `BrowserOpenURL`
- Source gate: no `<a href` literal in `HelpContent.tsx`
- DOM: clicking `.help-content__external-link` calls `BrowserOpenURL` with URL
- DOM: rendered output has no `<a[href]>` elements

**TESTING.md:**
- vitest count: 112 → 116
- 4 new HELP-01 traceability rows (HelpTab/Search/SectionNav/Content test files)
- M-14 manual checklist item (Help page visual verification in live native WebView)

## Test Results

| Suite | Before | After |
|-------|--------|-------|
| Existing vitest (1825 tests, 112 files) | 1825 pass | 1825 pass |
| New Help stubs (4 files, 25 tests) | — | 22 fail (RED, intended) |
| Sidebar updates (53 tests) | 46 pass, 0 fail | 46 pass, 7 fail (RED) |
| **Total** | 1825 pass | 1825 pass + 29 fail (RED) |

The 29 RED failures are the intended Wave 0 scaffold. Plans 02 (nav wiring) and 03 (components) turn them GREEN.

## Deviations from Plan

### Auto-fixed: SidebarTestProps type bridge

**Found during:** Task 4

**Issue:** Adding `onOpenHelp: vi.fn()` to `renderSidebar` defaultProps caused `tsc --noEmit` to fail (`TS2353: Object literal may only specify known properties`) because `SidebarProps` doesn't have `onOpenHelp` until Plan 02.

**Fix:** Added `type SidebarTestProps = Parameters<typeof Sidebar>[0] & { onOpenHelp?: () => void }` and cast the render call at the `act(...)` boundary. This is explicitly documented as a GREEN signal — removing the cast when Plan 02 adds the real prop confirms implementation is correct.

**Files modified:** `frontend/src/components/__tests__/Sidebar.test.tsx`

**Commit:** 00fe2c8e

### Auto-noted: 2 HelpSearch negative tests pass in RED state

2 of 7 HelpSearch tests (`does NOT show .help-search__empty when query is empty` and `does NOT show .help-search__empty when results are non-empty`) pass in RED state. This is because the null-component stub renders nothing, and `container.querySelector('.help-search__empty')` correctly returns `null`. Both assertions use `toBeNull()` which trivially passes. The meaningful positive assertions (label, clear button, empty-state content, mark highlight) all fail as intended. No action needed.

## Known Stubs

None — this plan creates test stubs and content, not component stubs. The HelpTab/Search/SectionNav/Content test files are intentionally failing scaffolds. No component implementations exist yet; Plans 02 and 03 create them.

## Threat Flags

No new network endpoints, auth paths, file access patterns, or schema changes introduced. Plan operates entirely within frontend test infrastructure and bundled content.

## Self-Check: PASSED

- `frontend/src/test-setup.ts` — FOUND
- `frontend/src/content/help/getting-started.md` — FOUND
- `frontend/src/content/help/faq.md` — FOUND
- `frontend/src/components/__tests__/HelpTab.test.tsx` — FOUND
- `frontend/src/components/__tests__/HelpSearch.test.tsx` — FOUND
- `frontend/src/components/__tests__/HelpSectionNav.test.tsx` — FOUND
- `frontend/src/components/__tests__/HelpContent.test.tsx` — FOUND
- Commit f1ed10be — FOUND (`chore(147-01): install rehype-sanitize + add IntersectionObserver polyfill`)
- Commit 84ab439c — FOUND (`feat(147-01): add seeded Help Markdown content`)
- Commit 00fe2c8e — FOUND (`test(147-01): update Sidebar tests + create four failing Help component stubs`)
