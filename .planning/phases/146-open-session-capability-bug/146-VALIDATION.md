---
phase: 146
slug: open-session-capability-bug
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-22
---

# Phase 146 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Out-of-band redesign (broadcast approach superseded). See 146-RESEARCH.md § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (backend) + vitest (frontend) |
| **Config file** | go.mod; frontend/vitest config |
| **Quick run command** | `go test ./internal/webserver/... ./internal/daemon/... ./internal/tailnet/...` ; `cd frontend && pnpm test -- <file> --run` |
| **Full suite command** | `go test ./...` ; `cd frontend && pnpm test --run && pnpm tsc --noEmit` |
| **Estimated runtime** | ~60–90 seconds |

---

## Sampling Rate

- **After every task commit:** Run the relevant quick command (Go package or vitest file)
- **After every plan wave:** Run the full suite + `pnpm tsc --noEmit`
- **Before `/gsd:verify-work`:** Full suite green AND tsc clean
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

> Planner fills concrete task rows. Anchor requirements below; every behavior-adding task needs an `<automated>` verify or an explicit manual-UAT entry.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 146-00-* | 00 | 0 | FIX-03 | T-146-01 | discovery stays cap-free; open path exercised, not just source-inspected | unit/behavior | (planner) | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Inverted RB-03 test — `/api/sessions/meta` contains NO `ro_join_code`/`rw_join_code`
- [ ] Out-of-band open behavior test — Open button → paste modal → exchange → `?cap=` URL
- [ ] At least one assertion that crosses the actual open path (not pure source-inspection) — the blind spot that hid the prior dead-on-arrival failure

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cross-machine live open via out-of-band code | FIX-03 | Live two-Mac tailnet + native webview; not automatable | Owner shares session, sends RO (and/or RW) code out of band; viewer pastes into RemoteJoinCodeModal → session opens in browser with correct permission |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references, including ≥1 cross-boundary/behavior assertion
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
