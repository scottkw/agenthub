---
phase: 138
slug: hub-first-navigation
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-20
---

# Phase 138 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1 (React Testing Library, JSDOM) + Playwright e2e (not required this phase) |
| **Config file** | `frontend/vite.config.ts` (vitest defaults; no standalone vitest.config) |
| **Quick run command** | `cd frontend && npx vitest run <changed-spec>` |
| **Full suite command** | `cd frontend && npx vitest run` |
| **Type gate** | `cd frontend && npx tsc --noEmit` |
| **Estimated runtime** | single-file ~2-5s; full suite ~60-120s (107+ files, ~1750 tests as of Phase 137) |

---

## Sampling Rate

- **After every task commit:** `cd frontend && npx vitest run <changed-spec>` + `npx tsc --noEmit`
- **After every plan wave:** `cd frontend && npx vitest run` (full suite)
- **Before `/gsd:verify-work`:** full suite green + tsc clean
- **Max feedback latency:** < 10s per task (single-file vitest + tsc)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 138-01-01 | 01 | 0 | NAV-02..05 | T-138-SC | N/A | unit | `cd frontend && npx vitest run src/components/__tests__/Sidebar.test.tsx src/components/__tests__/App.nav.test.tsx` | ✅ (rewrite) | ⬜ pending |
| 138-01-02 | 01 | 0 | CARD-01, CARD-03, CARD-04 | T-138-SC | N/A | unit/style | `cd frontend && npx vitest run src/components/__tests__/App.hub.test.tsx src/components/__tests__/style.hub.test.ts` | ✅ (edit) | ⬜ pending |
| 138-01-03 | 01 | 0 | CARD-02, CARD-03, CARD-04 | T-138-01 | isConnected boolean only, no token | unit | `cd frontend && npx vitest run src/components/__tests__/SessionCard.share.test.tsx` | ✅ (extend) | ⬜ pending |
| 138-02-01 | 02 | 1 | CARD-02, CARD-03 | T-138-SC | N/A | unit | `cd frontend && npx tsc --noEmit && npx vitest run src/lib/remoteAdapter.test.ts src/lib/__tests__/remoteSession.test.ts` | ✅ | ⬜ pending |
| 138-02-02 | 02 | 1 | CARD-02, CARD-03 | T-138-02, T-138-03 | provenance origin, no token in props | unit | `cd frontend && npx vitest run src/components/__tests__/SessionCard.share.test.tsx` | ✅ | ⬜ pending |
| 138-02-03 | 02 | 1 | CARD-03 | T-138-02 | colorblind-safe chip, custom-prop color | unit/style | `cd frontend && npx vitest run src/components/__tests__/SessionCard.share.test.tsx src/components/__tests__/style.hub.test.ts` | ✅ | ⬜ pending |
| 138-03-01 | 03 | 2 | CARD-04 | T-138-05, T-138-07 | two-step kill confirm, stopPropagation guard | unit/style | `cd frontend && npx vitest run src/components/__tests__/SessionCard.share.test.tsx src/components/__tests__/style.hub.test.ts` | ✅ | ⬜ pending |
| 138-03-02 | 03 | 2 | CARD-01 | T-138-06 | escaped peer hint text | unit/style | `cd frontend && npx vitest run src/components/__tests__/App.hub.test.tsx src/components/__tests__/style.hub.test.ts` | ✅ | ⬜ pending |
| 138-04-01 | 04 | 3 | NAV-02..05 | T-138-SC | narrowed UI surface | unit | `cd frontend && npx vitest run src/components/__tests__/Sidebar.test.tsx` | ✅ | ⬜ pending |
| 138-04-02 | 04 | 3 | NAV-02..05 | T-138-08, T-138-09 | Hub poll retained; no dead code | unit | `cd frontend && npx vitest run src/components/__tests__/App.nav.test.tsx src/components/__tests__/App.hub.test.tsx` | ✅ | ⬜ pending |
| 138-04-03 | 04 | 3 | NAV-03, NAV-04 | T-138-09 | no dangling imports | unit | `cd frontend && npx tsc --noEmit && npx vitest run` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements (Plan 01)

- [ ] `Sidebar.test.tsx` — assert exactly 3 items (Home/Hub/Settings), no Sessions/Remote/New Session (currently asserts `items.length === 6`)
- [ ] `App.nav.test.tsx` — NAV-02/03/04 blocks rewritten to assert absence of onOpenRemoteSessions/onOpenDaemonManager/onAdd wiring + removed Tab-type strings (Wave 0 gap surfaced during planning — not in original RESEARCH list)
- [ ] `App.hub.test.tsx` — drop DAEMON_MANAGER_TAB/REMOTE_SESSIONS_TAB/panel expectations; assert no `.hub__header`; assert new HubPanel props wired
- [ ] `style.hub.test.ts` — drop `.hub__header` assertions; add CARD-03 chip + CARD-04 destructive CSS + CARD-04 anti-regression assertions
- [ ] `SessionCard.share.test.tsx` — extend with CARD-02 origin, CARD-03 connection chip, CARD-04 Kill/remote-affordance tests (provenance-driven; stopPropagation guard)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Colorblind-safe indicator legibility | CARD-02, CARD-03 | Visual semantics | Verify at SOURCE level — icon/shape/text constants + `var(--hub-*)` custom properties in code, never color alone (user is colorblind; do NOT verify by eye) |
| Responsive grid reflow + attention pulse + mini-preview preserved with new affordances | CARD-04 | Visual/layout | dev-browser UAT: confirm pulse/float-to-top, mini-preview, grid density (240–360px), reflow intact after card additions |
| 3-item sidebar + sole New-Session entry + overflow affordances on live app | NAV-02..05, CARD-04 | Live interaction | dev-browser UAT: sidebar = Home/Hub/Settings; HubFilterBar is only creation entry; Kill two-step confirm; remote Open-in-browser + Browse-files |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (incl. App.nav.test.tsx gap found in planning)
- [x] No watch-mode flags
- [x] Feedback latency < 10s (single-file vitest + tsc)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planner-confirmed 2026-06-20
