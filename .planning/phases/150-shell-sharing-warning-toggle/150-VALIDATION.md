---
phase: 150
slug: shell-sharing-warning-toggle
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-23
---

# Phase 150 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (daemon) + vitest (frontend) |
| **Config file** | `frontend/vitest.config.ts`; Go uses standard `go test ./...` |
| **Quick run command** | `cd frontend && pnpm vitest run <changed test>` / `go test ./internal/daemon/...` |
| **Full suite command** | `cd frontend && pnpm vitest run && pnpm tsc --noEmit` ; `go test ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run the relevant quick command (go test for daemon, vitest for frontend)
- **After every plan wave:** Run the full suite for the touched surface
- **Before `/gsd:verify-work`:** Full suite must be green (vitest + `tsc --noEmit` + `go test ./...`)
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

> Filled by the planner from the D-04 behavior matrix and the RESEARCH.md "Validation Architecture" section.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 150-01-01 | 01 | 1 | SET-01 | — | New setting persists across restart (default ON via `*bool`) | unit | `go test ./internal/daemon/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

> Concrete from RESEARCH.md. Planner to confirm exact paths.

- [ ] Go daemon test covering `Get/SetShellWebShareWarningEnabled` persistence + OFF→ON re-arm (resets `shellWebShareWarned`)
- [ ] Frontend vitest covering the SettingsTab toggle state machine + confirm-on-disable
- [ ] Frontend vitest covering shell-warning interception on both surfaces (App.tsx StatusBar path + SessionShareModal ON-path)

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live web-share of a real shell session fires the warning banner on the actual share surface | SET-01 | Live PTY web-share cannot be driven through the automated bridge (no PTY) — see TESTING.md Section 5 | Web-share a shell session via Hub Share modal and via StatusBar; confirm banner appears when enabled+unacked, absent when disabled |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
