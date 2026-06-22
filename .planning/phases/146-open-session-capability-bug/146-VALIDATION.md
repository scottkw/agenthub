---
phase: 146
slug: open-session-capability-bug
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-22
validated: 2026-06-22
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
| 146-01 | 01 | 0 | RB-03, FIX-03 | T-146-01 | RED contracts: discovery cap-free; open path crosses the actual entry point (fills prior blind spot) | unit/behavior | `go test ./internal/webserver/ -run TestSessionsMeta_NoJoinCodesInResponse` ; `pnpm exec vitest run src/components/__tests__/App.open-remote.test.tsx` | ✅ `internal/webserver/sessions_meta_embed_test.go`, `frontend/src/components/__tests__/App.open-remote.test.tsx` | ✅ green |
| 146-02 | 02 | 0 | RB-03 | T-146-01 | `/api/sessions/meta` is cap-free (no `ro_join_code`/`rw_join_code`); broadcast wiring deleted | unit (inverted absence) | `go test ./internal/webserver/ -run TestSessionsMeta_NoJoinCodesInResponse -count=1` | ✅ `internal/webserver/sessions_meta_embed_test.go` | ✅ green |
| 146-03 | 03 | 0 | FIX-03 | T-146-01 | Out-of-band open: Open button → RemoteJoinCodeModal → exchange → `BrowserOpenURL` with `?cap=` URL | behavior (render + path) | `cd frontend && pnpm exec vitest run src/components/__tests__/App.open-remote.test.tsx` | ✅ `frontend/src/components/__tests__/App.open-remote.test.tsx` | ✅ green |
| 146-04 | 04 | 0 | FIX-03 | — | Regression-convention compliance; traceability paths resolve | doc gate | `bash tests/check-traceability-paths.sh` | ✅ `TESTING.md`, `tests/check-traceability-paths.sh` | ✅ green |
| 146-05 | 05 | 0 | FIX-03, GAP-146-A | T-146-05-01/02 | Held-cap reuse: second open reuses `RemoteCapStore` cap (no new mint, no second code); SID-correct fallback URL; used/expired copy | behavior (daemon + frontend) | `go test ./internal/daemon/ -run RemoteSessionOpenURL -count=1` ; `pnpm exec vitest run src/components/__tests__/App.open-remote.test.tsx` | ✅ `internal/daemon/open_remote_session_url_test.go`, `frontend/src/components/__tests__/App.open-remote.test.tsx` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Inverted RB-03 test — `/api/sessions/meta` contains NO `ro_join_code`/`rw_join_code` — `TestSessionsMeta_NoJoinCodesInResponse` (green)
- [x] Out-of-band open behavior test — Open button → paste modal → exchange → `?cap=` URL — `App.open-remote.test.tsx` held-cap + no-cap paths (green)
- [x] At least one assertion that crosses the actual open path (not pure source-inspection) — `TestRemoteSessionOpenURL_HeldCap`/`NoCap` exercise daemon `RemoteCapStore.Get` → cap-bearing URL composition, not a source grep (green)

*All Wave 0 requirements covered by green automated tests.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cross-machine live open via out-of-band code | FIX-03 | Live two-Mac tailnet + native webview; not automatable | Owner shares session, sends RO (and/or RW) code out of band; viewer pastes into RemoteJoinCodeModal → session opens in browser with correct permission |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references, including ≥1 cross-boundary/behavior assertion (`TestRemoteSessionOpenURL_*`)
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-22 — all requirements have green automated coverage; sole manual item (live two-Mac open, M-13) is inherently unautomatable.

---

## Validation Audit 2026-06-22

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Phase 146 audited in State A (existing VALIDATION.md). All five plans' requirements (RB-03, FIX-03, GAP-146-A) map to green automated tests confirmed re-run during this audit:
- `internal/webserver/sessions_meta_embed_test.go::TestSessionsMeta_NoJoinCodesInResponse` — PASS
- `internal/daemon/open_remote_session_url_test.go::TestRemoteSessionOpenURL_HeldCap`/`NoCap` — PASS
- `frontend/src/components/__tests__/App.open-remote.test.tsx` — 15/15 PASS

No tests generated — coverage already complete (the planning-stage VALIDATION.md was stale at `draft`/placeholder; reconciled to executed reality). The lone manual item (live two-Mac tailnet RO/RW open, TESTING.md M-13) cannot be automated and remains in Manual-Only above.
