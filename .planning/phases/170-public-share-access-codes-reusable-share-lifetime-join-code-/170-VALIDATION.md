---
phase: 170
slug: public-share-access-codes-reusable-share-lifetime-join-code
status: approved
nyquist_compliant: true
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

*Backfilled from the final plans after plan-check passed. Invariants that MUST be property/held-out tested: reusable code does NOT self-delete on Exchange; reusable read code resolves ONLY to the read cap (never write, browse ON and OFF); public code lifetime dies on `disableFunnelForSession` (explicit revoke, not TTL-only); public code minted once and cached (idempotent across re-issue).*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 170-01-T1 | 01 | 1 | FNL-08 | — | reusable Exchange does not delete; single-use path unchanged | impl | `go test -race -short ./internal/capability/... ./internal/webserver/...` | ❌ W0 | ⬜ pending |
| 170-01-T2 | 01 | 1 | FNL-08 | T-170-04 | reusable double-exchange + per-code TTL expiry | unit/property | `go test ./internal/capability/... -run JoinCode -v` | ❌ W0 | ⬜ pending |
| 170-01-T3 | 01 | 1 | FNL-08 | T-170-02 | public `/join/exchange` reusable double-exchange at HTTP boundary | unit | `go test ./internal/webserver/... -run TestJoinExchange -v` | ❌ W0 | ⬜ pending |
| 170-02-T1 | 02 | 2 | FNL-08 | T-170-01 / T-170-05 | mint-once-cached read-only code; TTL=min(ExpiresIn,8h); revoke in `disableFunnelForSession` | impl | `go test -race -short ./internal/daemon/...` | ❌ W0 | ⬜ pending |
| 170-02-T2 | 02 | 2 | FNL-08 | T-170-01 | read-only scope (browse ON/OFF), idempotent mint, teardown revocation | unit/property | `go test ./internal/daemon/... -run 'Funnel\|IssueCapabilities' -v` | ❌ W0 | ⬜ pending |
| 170-03-T1 | 03 | 3 | FNL-08 | — | models.ts sync + `<CodeDisplay>` public code row (supplement, not replace URL/QR) | impl | `tsc --noEmit` | ❌ W0 | ⬜ pending |
| 170-03-T2 | 03 | 3 | FNL-08 | — | publicReadCode threaded through modal warm-up + disable | impl | `tsc --noEmit` | ❌ W0 | ⬜ pending |
| 170-03-T3 | 03 | 3 | FNL-08 | — | PUBLIC section renders the reusable code row | unit | `pnpm vitest run SessionSharePanel` | ❌ W0 | ⬜ pending |
| 170-04-T1 | 04 | 4 | FNL-08 | — | Suite Manifest note + FNL-08 traceability rows | docs | `bash tests/check-traceability-paths.sh && grep -c "FNL-08" TESTING.md` | ✅ | ⬜ pending |
| 170-04-T2 | 04 | 4 | FNL-08 | — | M-46 live off-tailnet reusable-code join manual item | docs | `bash tests/check-traceability-paths.sh` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*Threat refs: T-170-01 read-only-scope (critical) · T-170-02 explicit-revoke-teardown (high) · T-170-04 idempotent-mint (high) · T-170-05 8h-TTL-cap (medium) — full definitions in the per-plan `<threat_model>` blocks.*

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-07-05
