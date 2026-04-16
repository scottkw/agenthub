---
phase: 81
slug: banner-notifications
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-16
---

# Phase 81 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4.1.0 |
| **Config file** | `frontend/vite.config.ts` |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `cd frontend && pnpm test` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test`
- **After every plan wave:** Run `cd frontend && pnpm test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 81-01-01 | 01 | 1 | BAN-01 | — | N/A | structural | `cd frontend && pnpm test` | ❌ W0 | ⬜ pending |
| 81-01-02 | 01 | 1 | BAN-01 | — | N/A | structural | `cd frontend && pnpm test` | ❌ W0 | ⬜ pending |
| 81-01-03 | 01 | 1 | BAN-02 | — | N/A | structural | `cd frontend && pnpm test` | ❌ W0 | ⬜ pending |
| 81-01-04 | 01 | 1 | BAN-02 | — | N/A | structural | `cd frontend && pnpm test` | ❌ W0 | ⬜ pending |
| 81-01-05 | 01 | 1 | BAN-02 | — | N/A | structural | `cd frontend && pnpm test` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/UpdateBanner.test.tsx` — new file, covers BAN-02 (update banner dismiss)
- [ ] `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` — extend with dismiss button tests (covers BAN-02)
- [ ] `frontend/src/components/__tests__/App.test.tsx` — extend with BannerStack integration tests (covers BAN-01)
- [ ] `frontend/src/components/__tests__/WelcomeTab.test.tsx` — remove update-banner describe block (tests will break after extraction)

*Existing infrastructure covers framework requirements. No new framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual stacking appearance | BAN-01 | Layout is verified structurally but visual confirmation requires rendering | Run app, trigger both banners, verify vertical stacking |
| Dismiss animation smoothness | BAN-02 | CSS transition timing is perceptual | Dismiss a banner, verify fade + collapse animation is smooth (~200ms) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
