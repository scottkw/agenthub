---
phase: 81
slug: banner-notifications
status: complete
nyquist_compliant: true
wave_0_complete: true
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
| **Estimated runtime** | ~8 seconds |

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
| 81-01-01 | 01 | 1 | BAN-01 | — | N/A | structural | `cd frontend && pnpm test` | ✅ App.test.tsx | ✅ green |
| 81-01-02 | 01 | 1 | BAN-01 | — | N/A | structural | `cd frontend && pnpm test` | ✅ App.test.tsx | ✅ green |
| 81-01-03 | 01 | 1 | BAN-02 | — | N/A | behavioral | `cd frontend && pnpm test` | ✅ LocalNetworkBanner.test.tsx | ✅ green |
| 81-01-04 | 01 | 1 | BAN-02 | — | N/A | behavioral | `cd frontend && pnpm test` | ✅ UpdateBanner.test.tsx | ✅ green |
| 81-01-05 | 01 | 1 | BAN-02 | — | N/A | structural | `cd frontend && pnpm test` | ✅ App.test.tsx | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Test Coverage Detail

### BAN-01 — Vertical stacking of banners

| Test File | Test Description | Status |
|-----------|------------------|--------|
| App.test.tsx | imports UpdateBanner component | ✅ |
| App.test.tsx | imports UpdateInfo type | ✅ |
| App.test.tsx | imports GetLastUpdateInfo from wailsjs bindings | ✅ |
| App.test.tsx | declares update state at App level | ✅ |
| App.test.tsx | declares localBannerDismissed state | ✅ |
| App.test.tsx | renders banner-stack div | ✅ |
| App.test.tsx | renders LocalNetworkBanner inside banner-stack | ✅ |
| App.test.tsx | renders UpdateBanner inside banner-stack | ✅ |
| App.test.tsx | subscribes to update:available event | ✅ |

### BAN-02 — Independent dismiss with animation

| Test File | Test Description | Status |
|-----------|------------------|--------|
| LocalNetworkBanner.test.tsx | renders dismiss button when onDismiss is provided | ✅ |
| LocalNetworkBanner.test.tsx | does not render dismiss button when onDismiss is not provided | ✅ |
| LocalNetworkBanner.test.tsx | calls onDismiss when dismiss button clicked | ✅ |
| LocalNetworkBanner.test.tsx | applies className when provided | ✅ |
| LocalNetworkBanner.test.tsx | dismiss button has correct aria-label | ✅ |
| UpdateBanner.test.tsx | renders update version information | ✅ |
| UpdateBanner.test.tsx | renders Download Update button | ✅ |
| UpdateBanner.test.tsx | renders Dismiss button with correct aria-label | ✅ |
| UpdateBanner.test.tsx | calls onDismiss when Dismiss button clicked | ✅ |
| UpdateBanner.test.tsx | has role="alert" for accessibility | ✅ |
| UpdateBanner.test.tsx | has aria-live="polite" | ✅ |
| UpdateBanner.test.tsx | applies className when provided (for banner-exit animation) | ✅ |
| UpdateBanner.test.tsx | calls BrowserOpenURL with releaseURL on download click | ✅ |
| UpdateBanner.test.tsx | renders "Update available:" message text | ✅ |
| App.test.tsx | defines handleDismissLocalBanner callback | ✅ |
| App.test.tsx | defines handleDismissUpdate callback | ✅ |
| App.test.tsx | passes onDismiss to LocalNetworkBanner | ✅ |
| App.test.tsx | passes onDismiss to UpdateBanner | ✅ |
| App.test.tsx | passes className with banner-exit for exit animation | ✅ |
| App.test.tsx | resets localBannerDismissed when webServerMode changes | ✅ |

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual stacking appearance | BAN-01 | Layout is verified structurally but visual confirmation requires rendering | Run app, trigger both banners, verify vertical stacking |
| Dismiss animation smoothness | BAN-02 | CSS transition timing is perceptual | Dismiss a banner, verify fade + collapse animation is smooth (~200ms) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete

---

## Validation Audit 2026-04-17

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All requirements verified against existing test files. 434 tests passing across 21 test files.
