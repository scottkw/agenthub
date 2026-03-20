---
phase: 13
slug: build-script
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) + go test (backend) + bash integration |
| **Config file** | `frontend/vite.config.ts` (test section) |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `cd frontend && pnpm test && cd .. && go test ./...` |
| **Estimated runtime** | ~15 seconds (unit); BUILD integration tests ~5-10 min each |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test` (regression guard)
- **After every plan wave:** Run `cd frontend && pnpm test && cd .. && go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green + BUILD integration checks
- **Max feedback latency:** 15 seconds (unit tests)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 13-01-01 | 01 | 1 | BUILD-01 | integration | `bash build.sh --platform macos && test -d build/bin/agenthub.app` | ❌ W0 | ⬜ pending |
| 13-01-02 | 01 | 1 | BUILD-03 | integration | `bash build.sh --platform windows && test -f build/bin/agenthub.exe` | ❌ W0 | ⬜ pending |
| 13-01-03 | 01 | 1 | BUILD-02 | integration | `bash build.sh --platform linux && file build/bin/agenthub \| grep ELF` | ❌ W0 | ⬜ pending |
| 13-01-04 | 01 | 1 | BUILD-04 | integration | `bash build.sh --all && test -d build/bin/agenthub.app && test -f build/bin/agenthub.exe` | ❌ W0 | ⬜ pending |
| 13-02-01 | 02 | 2 | BUILD-05 | manual | N/A — requires Apple Developer credentials | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- None — this phase creates a new `build.sh` file from scratch. No existing test infrastructure gaps. Integration test commands are manual verification steps run after the script is built.

*Existing infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `build.sh --platform macos --sign` produces signed+notarized build passing `spctl --assess` | BUILD-05 | Requires Apple Developer credentials (APPLE_DEV_ID, team ID, app-specific password) | 1. Set APPLE_DEV_ID, APPLE_TEAM_ID, APPLE_APP_PASSWORD env vars 2. Run `bash build.sh --platform macos --sign` 3. Verify `spctl --assess -v build/bin/agenthub.app` exits 0 with "accepted" |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
