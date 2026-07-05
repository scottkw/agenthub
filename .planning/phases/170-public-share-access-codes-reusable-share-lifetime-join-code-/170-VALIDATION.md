---
phase: 170
slug: public-share-access-codes-reusable-share-lifetime-join-code
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-05
---

# Phase 170 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `go test` (backend) + vitest (frontend) |
| **Config file** | none for Go; `frontend/vitest.config.ts` for vitest |
| **Quick run command** | `go test ./internal/capability/... ./internal/webserver/...` |
| **Full suite command** | `go test ./... && (cd frontend && pnpm vitest run)` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run the quick run command for the touched package
- **After every plan wave:** Run the full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

*Populated by the planner — one row per task, mapping each to its automated verification and (where relevant) threat reference. Invariants that MUST be property/held-out tested: reusable code does NOT self-delete on Exchange; reusable read code resolves ONLY to the read cap (never write); public code lifetime dies on `disableFunnelForSession` (explicit revoke, not TTL-only); public code minted once and cached (idempotent across re-issue).*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | FNL-08 | TBD | reusable read code never resolves to write cap | unit/property | `go test ./internal/capability/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/capability/joincode_test.go` — extend for reusable + per-code TTL semantics (non-consuming Exchange, Revoke, expiry)
- [ ] `internal/webserver/join_test.go` (new) — public `/join/exchange` resolves a reusable read code to the read cap
- [ ] `internal/daemon/funnel_test.go` — public code revoked on `disableFunnelForSession` (all teardown paths)
- [ ] `frontend/**/SessionSharePanel.test.tsx` — PUBLIC section renders the reusable code row

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Off-tailnet recipient types the public URL, lands on the code-entry page, enters the reusable code, and joins read-only | FNL-08 | Requires a real Funnel-granted tailnet + off-tailnet device (live end-to-end) | Enable Funnel on a session, copy the reusable public code from the Share modal PUBLIC section, on an off-tailnet device open the public URL, enter the code, confirm read-only join; verify a second device can reuse the same code |

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
