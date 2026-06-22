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

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib `testing`) + vitest (frontend) |
| **Config file** | `frontend/vitest.config.ts`; Go: none (stdlib) |
| **Quick run command** | `go test -race -short ./internal/webserver/... ./internal/daemon/... && cd frontend && pnpm tsc --noEmit && pnpm test` |
| **Full suite command** | `go test -race -short ./... && cd frontend && pnpm test` |
| **Estimated runtime** | ~90 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -race -short ./internal/webserver/... ./internal/daemon/... && cd frontend && pnpm tsc --noEmit && pnpm test`
- **After every plan wave:** Run `go test -race -short ./... && cd frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 146-W0 | 00 | 0 | FIX-03 | — | test scaffolding (red) | unit | `go test ./internal/webserver/... ./internal/daemon/...` | ❌ W0 | ⬜ pending |
| FIX-03 | — | 1 | FIX-03 | — | `handleSessionsMeta` embeds `ro_join_code`/`rw_join_code` for web-enabled sessions | unit (Go) | `go test -race -short ./internal/webserver/... -run TestSessionsMeta_EmbedJoinCodes` | ❌ W0 | ⬜ pending |
| FIX-03 | — | 1 | FIX-03 | T (RB-03 access ctrl) | RB-03 preserved — no raw `cap=` token in meta; only single-use join codes | unit (Go) | `go test -race -short ./internal/webserver/... -run TestSessionsMeta_NoCapInResponse` | ✅ needs update | ⬜ pending |
| FIX-03 | — | 1 | FIX-03 | — | `handleSessionsMeta` returns empty/absent codes when `joinCodeIssuer` is nil | unit (Go) | `go test -race -short ./internal/webserver/... -run TestSessionsMeta_NilIssuer` | ❌ W0 | ⬜ pending |
| FIX-03 | — | 1 | FIX-03 | T (replay) | `mintSessionJoinCodes` mints fresh RO+RW codes + registers grants (single-use, TTL) | unit (Go) | `go test -race -short ./internal/daemon/... -run TestMintSessionJoinCodes` | ❌ W0 | ⬜ pending |
| FIX-03 | — | 2 | FIX-03 | — | `adaptRemoteSession` passes through `roJoinCode`/`rwJoinCode` | unit (vitest) | `cd frontend && pnpm test -- remoteAdapter` | ✅ needs extension | ⬜ pending |
| FIX-03 | — | 2 | FIX-03 | — | `handleOpenRemoteSession` with code present → `ExchangeJoinCodeAtURL` → opens cap-bearing URL | unit (vitest) | `cd frontend && pnpm test -- App.open-remote` | ❌ W0 | ⬜ pending |
| FIX-03 | — | 2 | FIX-03 | T (dead-end 401, D-03) | no join code (not shared) → informative UI, no raw 401 dead-end | unit (vitest) | `cd frontend && pnpm test -- App.open-remote` | ❌ W0 | ⬜ pending |
| FIX-03 | — | 2 | FIX-03 | T (expiry) | exchange-failed (expired/session-gone) → informative banner | unit (vitest) | `cd frontend && pnpm test -- App.open-remote` | ❌ W0 | ⬜ pending |
| FIX-03 | — | 2 | FIX-03 | T (escalation, D-05/D-06) | RW code chosen only when peer = self; else RO (no silent escalation) | unit (vitest) | `cd frontend && pnpm test -- App.open-remote` | ❌ W0 | ⬜ pending |
| FIX-03 | — | 2 | FIX-03 | — | D-08: local "Open" calls `onOpenSession`, not `onOpenInBrowser` | unit (vitest) | `cd frontend && pnpm test -- SessionCard` | ✅ add assertion | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/sessions_meta_embed_test.go` — `TestSessionsMeta_EmbedJoinCodes`, `TestSessionsMeta_NilIssuer`; update `TestSessionsMeta_NoCapInResponse` allowed-keys list to include `ro_join_code`/`rw_join_code` (assert no raw `cap` field — RB-03 guard)
- [ ] `internal/daemon/mint_join_codes_test.go` — `TestMintSessionJoinCodes` (grant registration + single-use code issuance)
- [ ] `frontend/src/components/__tests__/App.open-remote.test.tsx` — `handleOpenRemoteSession` with/without join code, exchange error, D-05/D-06 RO-vs-RW selection
- [ ] `frontend/src/lib/__tests__/remoteAdapter.test.ts` (extend existing) — `roJoinCode`/`rwJoinCode` pass-through

*Existing files needing assertion updates: `SessionCard` test (D-08), `remoteAdapter` test.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| "Open in browser" opens the live session (not a 401 page) for a shared remote session | FIX-03 | Requires two real Macs on the same tailnet; the `:34115` wails-dev bridge has no real tailnet peer and web-share WS blocks automated input (see live-UAT memory) | On Mac A: start a session, enable Share (RO and/or RW). On Mac B: open AgentHub Hub, locate Mac A's remote session card, click "Open in browser". Expect: browser opens the live session UI (terminal visible), NOT "capability required". Repeat for RO-only share (expect RO open) and owner re-attach (expect RW per D-06). Add as new `M-NN` item in TESTING.md §5. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
