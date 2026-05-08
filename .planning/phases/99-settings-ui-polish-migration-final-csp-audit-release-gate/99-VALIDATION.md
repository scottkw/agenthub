---
phase: 99
slug: settings-ui-polish-migration-final-csp-audit-release-gate
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-08
---

# Phase 99 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> See `99-RESEARCH.md` Validation Architecture section for the full per-criterion mapping.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (frontend)** | vitest ^4.1.0 (component tests) + Playwright ^1.59.1 (e2e) |
| **Framework (backend)** | go test (standard library + testify) |
| **Config files** | `frontend/vitest.config.ts`, `frontend/playwright.config.ts`, `internal/daemon/*_test.go` |
| **Quick run command (frontend unit)** | `cd frontend && pnpm test:unit -- --run` |
| **Quick run command (backend)** | `go test ./internal/daemon/... -run Migration -count=1` |
| **Full e2e suite** | `cd frontend && pnpm test:e2e` (after Phase 99 multi-browser config: runs Chromium + Firefox + WebKit) |
| **Estimated runtime (quick)** | ~15s frontend unit, ~5s migration test |
| **Estimated runtime (full e2e)** | ~90s for 3 browsers serial, ~45s parallel |

---

## Sampling Rate

- **After every task commit:** Run the relevant quick command (frontend unit OR migration test depending on plan).
- **After every plan wave:** Run the full e2e suite for the affected workstream.
- **Before `/gsd-verify-work`:** Full multi-browser CSP suite green + migration test green + frontend unit green.
- **Max feedback latency:** 90s (full e2e suite, 3 browsers).

---

## Per-Task Verification Map

> Skeleton — populated by planner. Each task in each PLAN.md must include an `<automated>` verify command that maps to one of the entries below, OR be flagged manual-only with documented reason.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 99-01-XX | 01 | 1 | PUI-02 | — | One-shot BannerStack confirmation does not duplicate on rapid toggle | component (vitest) | `cd frontend && pnpm test:unit -- PluginToggleBanner` | ❌ W0 | ⬜ pending |
| 99-02-XX | 02 | 2 | PUI-03 | — | `<details>` disclosure renders Search/WebLinks/InlineImage configs and dispatches sub-key RPCs | component (vitest) | `cd frontend && pnpm test:unit -- PluginsSection.disclosure` | ❌ W0 | ⬜ pending |
| 99-03-XX | 03 | 1 | — (SC-3) | — | Migration is idempotent + populates all plugin defaults + writes schemaVersion: 2 | go test | `go test ./internal/daemon/... -run TestMigration -count=1` | ✅ existing (expand) | ⬜ pending |
| 99-04-XX | 04 | 3 | — (SC-4) | — | CSP zero-violation suite green on Chromium + Firefox + WebKit | playwright | `cd frontend && pnpm test:e2e -- --project=chromium --project=firefox --project=webkit web-csp` | ❌ W0 (config edit) | ⬜ pending |
| 99-05-XX | 05 | 4 | — (SC-4) | — | iPad Safari Tailscale UAT runbook executed on real device, zero CSP violations + zero CDN requests | manual | `99-iPad-UAT.md` checklist with sign-off | manual | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/Settings/PluginToggleBanner.test.tsx` — vitest stubs for PUI-02 BannerStack one-shot dedupe
- [ ] `frontend/src/components/Settings/PluginsSection.disclosure.test.tsx` — vitest stubs for PUI-03 disclosure forms (Search, WebLinks, InlineImage)
- [ ] `frontend/playwright.config.ts` — extend `projects[]` to include `firefox` and `webkit` (config edit, not Wave 0 stub)
- [ ] No new framework install — vitest, playwright, go test all present.

*Existing infrastructure covers most of Phase 99. Wave 0 adds only the two component test files and a Playwright config edit.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| BannerStack toast affordance is unambiguous to users | PUI-02 (SC-1) | "User-facing UI review (3+ test users or structured walkthrough)" — required by ROADMAP success criterion 1 | Recruit 3 test users OR perform structured walkthrough; confirm each can articulate "I need to open a new session" within 5 seconds of toggling Unicode 11 or Inline Images. Record sign-off in `99-VERIFICATION.md`. |
| iPad Safari Tailscale full session flow | SC-4 | Real device required (no automation for iPad Safari over Tailscale) | Follow `99-iPad-UAT.md`. Capture Web Inspector console screenshots. Sign off zero CSP violations + zero CDN requests for full attach/render/scrollback/detach with all v3.2 plugins enabled. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or are flagged manual-only with documented reason
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (2 vitest test files)
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter (after planner pass)

**Approval:** pending — to be approved by plan-checker after PLAN.md generation
