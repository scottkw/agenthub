---
phase: 138
slug: hub-first-navigation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-20
---

# Phase 138 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1 (React Testing Library) + Playwright e2e |
| **Config file** | `frontend/vitest.config.*` (existing) |
| **Quick run command** | `cd frontend && npx vitest run <changed-spec>` |
| **Full suite command** | `cd frontend && npx vitest run` |
| **Estimated runtime** | ~{N} seconds (planner to confirm) |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npx vitest run <changed-spec>`
- **After every plan wave:** Run `cd frontend && npx vitest run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** {N} seconds (planner to confirm)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| {N}-01-01 | 01 | 0 | NAV-05 | — | N/A | unit | `cd frontend && npx vitest run Sidebar.test` | ✅ | ⬜ pending |

*Planner fills the full map. Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/Sidebar.test.tsx` — update to assert exactly 3 items (Home/Hub/Settings), no Sessions/New Session (currently asserts `items.length === 6`)
- [ ] `frontend/src/components/__tests__/App.hub.test.tsx` / `style.hub.test.ts` — drop `.hub__header` expectations
- [ ] New specs for: card local/remote indicator (CARD-02), connected-vs-available indicator (CARD-03), overflow-menu Kill (decision 1), remote open-in-browser + browse buttons (decisions 2/3), peer status hint (decision 4)

*Existing vitest + Playwright infrastructure covers the framework; Wave 0 is test updates + new specs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Colorblind-safe indicator legibility | CARD-02, CARD-03 | Visual semantics | Verify at SOURCE level — icon/shape/text constants in code, never color alone (user is colorblind; do not verify by eye) |
| Responsive grid reflow + attention pulse preserved with new card affordances | CARD-04 | Visual/layout | dev-browser UAT: confirm pulse/float-to-top, mini-preview, grid density, reflow intact after redesign |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < {N}s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
