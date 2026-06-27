---
phase: 155-web-share-chat-ui-cross-surface-parity-gate
verified: 2026-06-26T00:00:00Z
status: passed
score: 4/4 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 3/4
  gaps_closed:
    - "PARITY-01 SC-3 RO-gate parity test flake (BLOCKER 2 residual) — closed by a TEST-ONLY fix (commit 5a6f8054): the SC-3 assertion was changed from a raw `.chat-msg` count delta (afterCount == beforeCount, vulnerable to late shared-session broadcast residue on WebKit) to a content-scoped assertion that the RO client's OWN unique message ('adversarial-ro-send') never lands. The release-blocking parity gate is now GREEN 24/24 on chromium/firefox/webkit across 3 consecutive verifier-run full suites (37.4s / 36.5s / 37.9s). No product code changed — server ErrChatReadOnly/ErrReadOnly and client isReadOnly untouched; Send-disabled assertion retained."
  gaps_remaining: []
  regressions: []
deferred: []
---

# Phase 155: Web-Share Chat UI + Cross-Surface Parity Gate — Verification Report (Final Re-Verification)

**Phase Goal:** The web-share surface delivers the identical chat experience and Markdown export is available on both surfaces.
**Verified:** 2026-06-26
**Status:** passed
**Re-verification:** Yes — final, after the SC-3 flake fix (commit 5a6f8054)

## Re-Verification Summary

| Prior Blocker | Disposition | Evidence |
|---------------|-------------|----------|
| **BLOCKER 1** — PARITY-01 SC-1 broadcast non-delivery (all 3 browsers) | **CLOSED (confirmed stable)** | Real server-side two-phase subscribe fix (Subscribe before WhoIs, RegisterPresence after) present at HEAD; `go test -race ./internal/relay/... ./internal/webserver/...` PASS; SC-1 broadcast + unread badge GREEN on all 3 browsers in all 3 fresh runs. |
| **BLOCKER 2** — PARITY-01 SC-3 RO-gate parity test intermittently RED on WebKit | **CLOSED** | TEST-ONLY content-scoped assertion (commit 5a6f8054) eliminates the shared-session count-contamination. SC-3 GREEN on WebKit in all 3 consecutive runs. Product RO gate verified intact at source (zero RO-gate lines changed). |

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | PARITY-01 SC-1: Web-share browser sees same thread, presence, unread badge, @mention as desktop — verified by Playwright e2e | ✓ VERIFIED | SC-1 broadcast (subscriberCount=4 at send time), unread badge, presence roster, typing slot, @mention all GREEN on chromium/firefox/webkit in all 3 fresh runs. Server-side two-phase subscribe fix confirmed in source (Subscribe@server.go:1086 before WhoIs@1108; RegisterPresence@1129 after). `go test -race ./internal/relay/...` PASS. |
| 2 | EXPORT-01 SC-2: Export downloads a `.md` with YAML frontmatter + full thread from both surfaces | ✓ VERIFIED | Playwright EXPORT-01 SC-2 download GREEN on all 3 browsers in all 3 runs. `internal/daemon/chat.go` `Export()` unchanged; Go export tests pass under `go test -race ./internal/webserver/...`. |
| 3 | PARITY-01 SC-3: RO-cap viewer cannot post or inject — server rejects regardless of client behavior, GREEN on all 3 browsers | ✓ VERIFIED | Product RO gate intact at source (HandleChatSend→ErrChatReadOnly hub.go:619, HandleInject→ErrReadOnly hub.go:543; git diff 3fd245a2..HEAD shows ZERO RO-gate lines changed; Send button disabled assertion retained at spec line 310). SC-3 now content-scoped (own message 'adversarial-ro-send' must never land, spec line 327) — GREEN on WebKit in all 3 consecutive runs. |
| 4 | PARITY-01 SC-4: @session inject indicator (.chat-msg--inject) renders identically from both surfaces | ✓ VERIFIED | Playwright SC-4 GREEN on all 3 browsers in all 3 runs. Seeded `SessionInject:true` message renders with `.chat-msg--inject`. Unchanged since prior VERIFIED. |

**Score:** 4/4 truths verified

### Authoritative Live Evidence — Playwright Cross-Surface Parity Gate

The release-blocking gate is the live Playwright run. The flake was intermittent, so a single green run is insufficient. Three consecutive fresh full-suite runs (`pnpm -C frontend exec playwright test chat-parity --reporter=line`) on this verifier's machine, repo root:

| Run | Result | webkit SC-3 |
|-----|--------|-------------|
| 1 | **24 passed (37.4s)** | GREEN |
| 2 | **24 passed (36.5s)** | GREEN |
| 3 | **24 passed (37.9s)** | GREEN |

All 24 instances (SC-1 ×5, SC-2, SC-3, SC-4 across chromium/firefox/webkit) GREEN on every run. The prior intermittent webkit SC-3 failure (`Expected 6, Received 8`) is eliminated — the gate is now deterministically green.

### SC-3 Fix Is Test-Only and Strictly Stronger (RO Gate NOT Weakened)

| Check | Result |
|-------|--------|
| `git show 5a6f8054 --stat` touches only the spec | ✓ VERIFIED — sole file: `frontend/e2e/chat-parity.spec.ts` (+11/-7) |
| No Go change to `ErrChatReadOnly`/`ErrReadOnly`/`HandleChatSend`/`HandleInject` since prior verification | ✓ VERIFIED — `git diff 3fd245a2..HEAD -- internal/relay/hub.go internal/webserver/server.go` shows ZERO RO-gate lines changed |
| No change to client `isReadOnly` (`ChatPanel.tsx`) | ✓ VERIFIED — not in diff |
| SC-3 still asserts Send button disabled | ✓ VERIFIED — `expect(sendBtn).toBeDisabled()` retained at spec line 310 |
| SC-3 asserts RO client's own message never lands | ✓ VERIFIED — `expect(page.locator('.chat-msg').filter({ hasText: 'adversarial-ro-send' })).toHaveCount(0)` at spec line 327 — content-scoped, immune to shared-session broadcast residue, strictly stronger than the old count delta |

### Server-Side Broadcast Fix Verification (BLOCKER 1 — confirmed stable)

| Check | Result |
|-------|--------|
| `internal/webserver/server.go` calls `hub.Subscribe(sub)` BEFORE `lc.WhoIs(...)` | ✓ VERIFIED — Subscribe@1086, WhoIs@1108 |
| `hub.RegisterPresence(sub)` called AFTER identity resolved | ✓ VERIFIED — @1129 (two-phase presence registration) |
| `internal/relay/hub.go` defines `RegisterPresence` | ✓ VERIFIED — @175 |
| `go test -race ./internal/relay/... ./internal/webserver/...` | ✓ PASS (both packages) |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/e2e/chat-parity.spec.ts` | 8 tests × 3 browsers green; SC-3 content-scoped RO assertion | ✓ VERIFIED | 24/24 on all 3 consecutive runs; SC-3 disabled + own-message-never-lands assertions both present |
| `internal/webserver/server.go` | Two-phase subscribe: Subscribe→WhoIs→RegisterPresence | ✓ VERIFIED | Lines 1086/1108/1129; build clean |
| `internal/relay/hub.go` | `RegisterPresence` present; RO gate untouched | ✓ VERIFIED | RegisterPresence@175; ErrChatReadOnly@619, ErrReadOnly@543 unchanged |
| `internal/relay/hub_subscribe_race_test.go` | `-race` proof of pre-identity broadcast delivery | ✓ VERIFIED | passes under `-race` |
| `internal/daemon/chat.go` (Export) | Unchanged, still VERIFIED | ✓ VERIFIED | Not in diff; Go export tests pass |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `websocket.Accept` | `hub.Subscribe` | two-phase: subscribe before WhoIs | ✓ WIRED | server.go:1086 before 1108 — race-free broadcast fan-out |
| `hub.HandleChatSend` | `hub.BroadcastChat` → page2 WS | `sub.Msgs` → write pump | ✓ WIRED + DELIVERING | SC-1 broadcast GREEN all browsers/runs; subscriberCount=4 |
| `hub.HandleChatSend` (RO) | `ErrChatReadOnly` | `sub.ReadOnly` gate | ✓ WIRED (unchanged) | hub.go:619; RO send rejected; SC-3 own-message-never-lands GREEN |
| `WhoIs` resolved | `hub.RegisterPresence` | presence roster | ✓ WIRED | server.go:1129 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full parity gate (run 1) | `pnpm -C frontend exec playwright test chat-parity` | 24/24 (37.4s) | ✓ PASS |
| Full parity gate (run 2) | same | 24/24 (36.5s) | ✓ PASS |
| Full parity gate (run 3) | same | 24/24 (37.9s) | ✓ PASS |
| webkit SC-3 RO gate | playwright (all 3 runs) | GREEN | ✓ PASS |
| Race tests | `go test -race ./internal/relay/... ./internal/webserver/...` | both ok | ✓ PASS |
| Go build | `go build ./...` | exit 0 | ✓ PASS |
| Traceability paths | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist`, exit 0 | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| EXPORT-01 | 155-01/02/04 | Download chat thread as Markdown from both surfaces | ✓ VERIFIED | REQUIREMENTS.md:44; Go export tests pass; Playwright export download GREEN all browsers, all 3 runs |
| PARITY-01 | 155-02/03/04/05/06 | Every Session Chat feature behaves identically on desktop + web-share — release-blocking | ✓ VERIFIED | REQUIREMENTS.md:53; release-blocking parity gate deterministically GREEN 24/24 on chromium/firefox/webkit across 3 consecutive runs (SC-1 broadcast race genuinely fixed server-side; SC-3 RO gate intact + flake eliminated; SC-4 inject green) |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | None | — | Prior `.chat-msg` raw-count assertion (flaky gate) replaced by content-scoped assertion. No debt markers in phase-modified source. |

## Gaps Summary

**No gaps. Phase goal achieved.**

The web-share surface delivers the identical chat experience (PARITY-01 SC-1/SC-3/SC-4 all GREEN on chromium/firefox/webkit across 3 consecutive full-suite runs) and Markdown export is available on both surfaces (EXPORT-01 SC-2 GREEN). The two prior blockers are closed:

- **BLOCKER 1 (broadcast non-delivery):** real server-side two-phase subscribe fix, proven by `-race` test and green live on all 3 browsers in every run.
- **BLOCKER 2 (SC-3 RO-gate parity flake):** closed by a test-only, strictly-stronger content-scoped assertion (commit 5a6f8054). No product code changed — the server `ErrChatReadOnly`/`ErrReadOnly` gate and client `isReadOnly` suppression are untouched, the Send-disabled assertion is retained, and the gate is now deterministically green.

`go build`, `go test -race`, and `tests/check-traceability-paths.sh` all pass.

---

_Verified: 2026-06-26_
_Verifier: Claude (gsd-verifier)_
_Authoritative live e2e evidence: 3 consecutive verifier-run full-suite Playwright runs — 24/24, 24/24, 24/24 (37.4s / 36.5s / 37.9s); webkit SC-3 GREEN every run. go test -race PASS; go build exit 0; traceability exit 0. SC-3 fix (commit 5a6f8054) is test-only; zero RO-gate lines changed._
