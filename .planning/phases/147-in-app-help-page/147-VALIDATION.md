---
phase: 147
slug: in-app-help-page
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-22
---

# Phase 147 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4.1.0 (jsdom 29.0.0) |
| **Config file** | `frontend/vite.config.ts` (inline `test:` block) |
| **Setup file** | `frontend/src/test-setup.ts` (add IntersectionObserver polyfill — Wave 0) |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite + type check** | `cd frontend && npx tsc --noEmit && pnpm test` |
| **Build gate** | `cd frontend && npx tsc && vite build` |
| **Estimated runtime** | ~30 seconds (vitest); +~20s for tsc |

> Project rule (memory: run-tsc-in-the-frontend-gate): vitest tolerates TS errors that `tsc && vite build` rejects. A green vitest run is NOT proof the app builds — always run `tsc` in the wave/phase gate.

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test`
- **After every plan wave:** Run `cd frontend && npx tsc --noEmit && pnpm test`
- **Before `/gsd:verify-work`:** Full suite green (`tsc` + `pnpm test`) AND `vite build` succeeds
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Task | Plan | Requirement | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|------|------|-------------|-----------------|-----------|-------------------|-------------|--------|
| Sidebar renders Help button (4th item, aria-label="Help") | nav | HELP-01 | N/A | unit | `pnpm test` → Sidebar.test.tsx | ❌ W0 (update) | ⬜ pending |
| Help button fires onOpenHelp on click | nav | HELP-01 | N/A | unit | `pnpm test` → Sidebar.test.tsx | ❌ W0 (update) | ⬜ pending |
| Help button gets `sidebar__item--active` when `activePanel === '__help__'` | nav | HELP-01 | N/A | unit | `pnpm test` → Sidebar.test.tsx | ❌ W0 (update) | ⬜ pending |
| `HELP_TAB` constant + `handleOpenHelp` exist in App.tsx (source gate) | nav | HELP-01 | N/A | unit | `pnpm test` → HelpTab.test.tsx | ❌ W0 (create) | ⬜ pending |
| HelpSearch debounce: query sets debouncedQuery after 200ms | search | HELP-01 | V5 input validation (in-memory substring) | unit | `pnpm test` → HelpSearch.test.tsx | ❌ W0 (create) | ⬜ pending |
| HelpSearch renders input with visible label "Search help…" | search | HELP-01 | N/A | unit | `pnpm test` → HelpSearch.test.tsx | ❌ W0 (create) | ⬜ pending |
| HelpSearch renders clear button with `aria-label="Clear search"` | search | HELP-01 | N/A | unit | `pnpm test` → HelpSearch.test.tsx | ❌ W0 (create) | ⬜ pending |
| HelpSearch shows empty-state when query non-empty AND 0 results | search | HELP-01 | N/A | unit | `pnpm test` → HelpSearch.test.tsx | ❌ W0 (create) | ⬜ pending |
| HelpSearch does NOT show empty-state for empty query | search | HELP-01 | N/A | unit | `pnpm test` → HelpSearch.test.tsx | ❌ W0 (create) | ⬜ pending |
| Highlighted snippet wraps matched term in `<mark class="help-search__mark">` | search | HELP-01 | XSS: plain-string React JSX (auto-escaped) | unit | `pnpm test` → HelpSearch.test.tsx | ❌ W0 (create) | ⬜ pending |
| HelpSectionNav renders a button per section with correct `aria-current` | nav | HELP-01 | N/A | unit | `pnpm test` → HelpSectionNav.test.tsx | ❌ W0 (create) | ⬜ pending |
| style.css declares `--hub-search-highlight-bg` in `:root` + `[data-ui-theme="light"]` | search | HELP-01 | N/A | unit (CSS source gate) | `pnpm test` → HelpTab.test.tsx | ❌ W0 (create) | ⬜ pending |
| HelpContent renders `<Markdown>` (react-markdown import source gate) | content | HELP-01 | XSS: react-markdown virtual DOM + rehype-sanitize | unit | `pnpm test` → HelpContent.test.tsx | ❌ W0 (create) | ⬜ pending |
| BrowserOpenURL called when external link button clicked | content | HELP-01 | Spoofing: button + BrowserOpenURL, no `<a href>` | unit | `pnpm test` → HelpContent.test.tsx | ❌ W0 (create) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Add `IntersectionObserver` polyfill to `frontend/src/test-setup.ts` (mirror existing ResizeObserver polyfill)
- [ ] Update `frontend/src/components/__tests__/Sidebar.test.tsx` — change "exactly 3 sidebar__item buttons" assertions (lines ~95, ~238) to 4; add Help-button assertions (D-01 reopens NAV-05)
- [ ] Create `frontend/src/components/__tests__/HelpTab.test.tsx` — HELP_TAB constant, handleOpenHelp source gates, `--hub-search-highlight-bg` CSS token source gate
- [ ] Create `frontend/src/components/__tests__/HelpSearch.test.tsx` — debounce, highlight `<mark>`, empty-state, clear button
- [ ] Create `frontend/src/components/__tests__/HelpSectionNav.test.tsx` — section-nav render, aria-current
- [ ] Create `frontend/src/components/__tests__/HelpContent.test.tsx` — react-markdown source gate, BrowserOpenURL call
- [ ] Add `rehype-sanitize` to `frontend/package.json` (pnpm) — defense-in-depth; react-markdown + remark-gfm already installed
- [ ] Create `frontend/src/content/help/` with the seeded `.md` content files

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Help page opens in the live Wails native webview; full Markdown renders (headings, paragraphs, code spans); section-nav active state updates on scroll; external links open the system browser | HELP-01 | Wails native webview is not accessible to Playwright/headless automation (memory: wails-dev-browser-pty-limit / devtools-disabled-in-prod) | Build with `-tags wailsassets` (or `wails dev`); click Help in sidebar; scroll content pane and confirm left-nav active item tracks; type a query and confirm debounced highlight + jump; click GitHub/Website links and confirm system browser opens. Register as **M-NN** in TESTING.md §5. |

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (polyfill, test stubs, content dir, rehype-sanitize)
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] New test files registered in TESTING.md (Suite Manifest §2, Traceability §4) + `bash tests/check-traceability-paths.sh` passes
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
