---
phase: 171
slug: public-full-access-read-write-internet-sharing-behind-a-hard
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-07
---

# Phase 171 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Frameworks** | Go `go test` (backend: `internal/capability`, `internal/webserver`, `internal/daemon`) + `vitest` via `pnpm test` (frontend) + `tsc --noEmit` type-gate (frontend) + repo shell checks (`bash tests/*.sh`) |
| **Config files** | Go: per-package `_test.go` (no build tags — repo standing rule); frontend: `frontend/vitest.config.ts` + `frontend/tsconfig.json` (existing — no new config) |
| **Quick run command** | Backend (per commit, scoped): `go test -race ./internal/capability/... ./internal/webserver/... ./internal/daemon/...` · Frontend (per commit, scoped): `cd frontend && pnpm exec tsc --noEmit && pnpm test -- SessionSharePanel SessionCard.share` |
| **Full suite command** | `go test -race ./internal/...` **&&** `cd frontend && pnpm exec tsc --noEmit && pnpm test && pnpm build` **&&** `bash tests/check-traceability-paths.sh` |
| **Estimated runtime** | Scoped: ~20–40s · Full: ~3–5 min (Go `-race` + vitest + `pnpm build`) |

---

## Sampling Rate

- **After every task commit:** Run the task's own `<automated>` verify command (scoped `go test -run '<Tests>'` or scoped `pnpm test -- <spec>`).
- **After every plan wave:** Run the wave's package suite un-scoped (e.g. Wave 1/2 → `go test -race ./internal/capability/... ./internal/webserver/... ./internal/daemon/...`; Wave 3 → `cd frontend && pnpm exec tsc --noEmit && pnpm test`).
- **Before `/gsd-verify-work`:** Full suite must be green, including `pnpm build` (wails-dev parity — vitest alone does not prove the app builds, per the standing lesson) and `bash tests/check-traceability-paths.sh`.
- **Max feedback latency:** ~40 seconds (scoped commands).

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 171-01-01 | 01 | 1 | FNL-09 | T-171-03 / T-171-04 | Single-use write code, custom-TTL honored (not truncated to 5m), atomic single-redeem under race | unit | `go test -race ./internal/capability/... -run 'TestJoinCodeManager_IssueSingleUseWithTTL'` | ✅ joincode_test.go (extended) | ⬜ pending |
| 171-01-02 | 01 | 1 | FNL-09 | T-171-01 / T-171-02 | Non-gated write cap rejected at the REAL WS upgrade (isGrantActive); RemoveGrant surgical (sibling grants survive) | integration | `go test -race ./internal/webserver/... -run 'TestRemoveGrant\|TestSetRWGate\|TestHandleWSSRelay_WriteCap_RequiresGate'` | ✅ rwgate_test.go (new) | ⬜ pending |
| 171-01-03 | 01 | 1 | FNL-09 | T-171-05 | originAllowedForWrite (D-02 defense-in-depth for files.write only) rejects funnel-origin write unless isRWGated | unit | `go test -race ./internal/webserver/... -run 'TestOriginAllowedForWrite'` | ✅ rwgate_test.go (new) | ⬜ pending |
| 171-02-01 | 02 | 2 | FNL-09 | T-171-07 | D-04 write-rebase removed; structs + client + Wails TS bindings compile | build | `go build ./... && cd frontend && pnpm exec tsc --noEmit` | ✅ types.go/api.go/client.go/app.go | ⬜ pending |
| 171-02-02 | 02 | 2 | FNL-09 | T-171-06 / T-171-09 | Gate-minted cap Perms exactly "read,write" (never files.write, browse-independent); expiry clamped ≤3600 server-side | unit | `go test -race ./internal/daemon/... -run 'TestFunnelWriteGate_TerminalOnlyScope\|TestHandleSetSessionFunnelWrite_ExpiryClamp'` | ✅ funnel_test.go (extended) | ⬜ pending |
| 171-02-03 | 02 | 2 | FNL-09 | T-171-08 / T-171-10 | Surgical RW teardown at all four triggers; read code + read grant survive an RW-only disable; no orphaned write cap | integration | `go test -race ./internal/daemon/... -run 'TestDisableFunnelWrite_RevokesGrantOnly\|TestFunnelWriteTeardown'` | ✅ funnel_test.go (extended) | ⬜ pending |
| 171-03-01 | 03 | 3 | FNL-09 | T-171-13 | FULL ACCESS badge uses clip-path (shape distinction, not color); distinct token hex | type + grep | `cd frontend && pnpm exec tsc --noEmit && grep -q 'clip-path' src/style.css && grep -q 'hub-fullaccess-badge-text' src/style.css` | ✅ style.css | ⬜ pending |
| 171-03-02 | 03 | 3 | FNL-09 | T-171-11 / T-171-12 | ≥3s hold gate; early release issues nothing; consent copy contains "command execution" + "anyone with the link" | unit | `cd frontend && pnpm test -- SessionSharePanel` | ✅ SessionSharePanel.test.tsx (extended) | ⬜ pending |
| 171-03-03 | 03 | 3 | FNL-09 | T-171-13 | FULL ACCESS badge/icon distinct from read INTERNET badge in class+icon+label; coexistence + read-survives | unit | `cd frontend && pnpm test -- SessionCard.share && pnpm exec tsc --noEmit` | ✅ SessionCard.share.test.tsx (extended) | ⬜ pending |
| 171-04-01 | 04 | 4 | FNL-09 | T-171-14 | TESTING.md reconciled; traceability path-check green; live off-tailnet M-item recorded | script | `bash tests/check-traceability-paths.sh` | ✅ TESTING.md | ⬜ pending |
| 171-04-02 | 04 | 4 | FNL-09 | — | Sharing Guide public-write section states RCE risk plainly; frontend tsc clean | type + grep | `cd frontend && pnpm exec tsc --noEmit && grep -qi 'FULL ACCESS' src/content/help/sharing-guide.md` | ✅ sharing-guide.md | ⬜ pending |
| 171-04-03 | 04 | 4 | FNL-09 | T-171-15 | 171-SECURITY.md asserts no public write path except the gate; threats→mitigations→verifying tests mapped | script | `test -f .../171-SECURITY.md && grep -q 'no public write path' .../171-SECURITY.md && grep -q 'TestHandleWSSRelay_WriteCap_RequiresGate' .../171-SECURITY.md && grep -q 'gsd-secure-phase 171' .../171-SECURITY.md` | ✅ 171-SECURITY.md (written by task) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*Path abbreviation `.../` = `.planning/phases/171-public-full-access-read-write-internet-sharing-behind-a-hard/`.*

---

## Wave 0 Requirements

**Existing test infrastructure covers all phase requirements — no new harness needed.**

- Go: `go test` with `-race` is the established backend runner; `internal/capability/joincode_test.go` and `internal/daemon/funnel_test.go` already exist and are extended in place. `internal/webserver/rwgate_test.go` is a NEW test file but adds no new framework/config — it uses the package's existing relay/upgrade test setup.
- Frontend: `vitest` (`pnpm test`) + `tsc --noEmit` are the established runners; `SessionSharePanel.test.tsx` and `SessionCard.share.test.tsx` already exist and are extended in place.
- Shell: `tests/check-traceability-paths.sh` already exists (repo standing rule).
- No `pip`/framework install, no watch-mode flags, no new config files.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Off-tailnet public-write end-to-end (the RCE-severity closed-perimeter acceptance gate) | FNL-09 (SPEC R4/R6) | No Tailscale Funnel tailnet in CI; requires a signed production build + a real off-tailnet device | Section-5 M-47: pass the hold-gate → redeem the single-use write code from an off-tailnet device → confirm terminal input executes → owner disables RW → writer's next input returns 401 while a separate read spectator (reusable read code) keeps viewing → confirm single-use (2nd redemption fails closed). |
| Live write delivery over Funnel origin distinguishable from read | FNL-09 (SPEC R7) | Colorblind-safe indicator is source-verified in tests; live render on a prod build is the human confirmation | Verify the FULL ACCESS badge/tab-icon (clip-path shape + LockOpenIcon + "FULL ACCESS" label) renders on a running signed build alongside the read INTERNET badge. |

*All automatable phase behaviors have automated verification; the two rows above are the non-CI-reachable live gates recorded as TESTING.md Section-5 items.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none — existing infrastructure covers all)
- [x] No watch-mode flags
- [x] Feedback latency < 40s (scoped commands)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-07-07
