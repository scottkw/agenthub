---
phase: 151
slug: message-schema-chatstore
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-25
---

# Phase 151 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Reconstructed from artifacts (State B) on 2026-06-25 — all requirements covered by
> existing automated Go tests; no test generation required.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib `testing`) |
| **Config file** | none — `go.mod` (module `github.com/scottkw/agenthub`) |
| **Quick run command** | `go test ./internal/relay/ ./internal/daemon/ ./internal/webserver/ -run Chat -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~1s (chat subset, no race) · ~25s (daemon with `-race`) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/ -run Chat -count=1`
- **After every plan wave:** Run `go test ./internal/relay/ ./internal/daemon/ ./internal/webserver/ -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~25 seconds (race-enabled daemon package)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 151-01-01 | 01 | 1 | PERSIST-01 | — | ChatMessage round-trips every field through one JSONL line | unit | `go test ./internal/relay/ -run Chat -count=1` | ✅ | ✅ green |
| 151-01-02 | 01 | 1 | PERSIST-01 | T-151 path-traversal | JSONL append persists; fresh store reloads full history (restart survival); sessionID allowlist rejects traversal | unit | `go test ./internal/daemon/ -run 'TestChatStoreRestartSurvival|TestChatStoreRejectPathTraversal' -count=1` | ✅ | ✅ green |
| 151-01-03 | 01 | 1 | PERSIST-02, PERSIST-03 | — | Messages() returns full in-order thread; 10k cap rejects via ErrChatCapReached; concurrent appends pass -race | unit | `go test ./internal/daemon/ -run 'TestChatCapEnforcement|TestChatConcurrentAppend' -race -count=1` | ✅ | ✅ green |
| 151-02-01 | 02 | 2 | PERSIST-01, PERSIST-03 | — | Export() renders all fields to Markdown; Delete() removes JSONL + clears mirror (idempotent) | unit | `go test ./internal/daemon/ -run 'TestChatExportFields|TestChatDelete' -count=1` | ✅ | ✅ green |
| 151-02-02 | 02 | 2 | PERSIST-01, PERSIST-03 | — | SessionEngine chatStores: store created on CreateSession, Delete()+map removal on KillSession (no orphan) | unit | `go test ./internal/daemon/ -run 'TestEngineChatStoreFor' -count=1` | ✅ | ✅ green |
| 151-03-01 | 03 | 3 | PERSIST-01, PERSIST-02 | — | GET /api/chat/{id}/history+export on relay loopback returns full thread / Markdown; restart returns all prior; invalid id → 404 | unit | `go test ./internal/daemon/ -run 'TestChatRoutes' -count=1` | ✅ | ✅ green |
| 151-03-02 | 03 | 3 | PERSIST-02 | T-151-04 cap isolation | Webserver chat routes capability-gated: 401 no/invalid cap, 403 wrong-session cap, 200 valid same-session; internal error → 500 not 404 | unit | `go test ./internal/webserver/ -run Chat -count=1` | ✅ | ✅ green |
| 151-03-03 | 03 | 3 | PERSIST-01/02/03 | — | TESTING.md Suite Manifest delta + traceability rows; path check passes | doc/script | `bash tests/check-traceability-paths.sh` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No Wave 0 test stubs were
needed — every PERSIST requirement was satisfied with behavior-targeting Go tests
written during execution (44 chat test functions across 5 files).

---

## Requirement → Test Coverage (Nyquist)

| Requirement | Definition | Status | Targeting tests (green) |
|-------------|-----------|--------|--------------------------|
| **PERSIST-01** | Thread persisted for the session's life; survives daemon/app restart | COVERED | `TestChatStoreRestartSurvival`, `TestChatRoutes_RestartSurvival`, `TestChatAppendRoundTrip`, `TestChatMessageRoundTrip`, `TestChatConcurrentAppend` (-race), `TestChatExportFields` |
| **PERSIST-02** | Late-joining participant loads the full thread scrollback | COVERED | `TestChatRoutes_History`, `TestChatRoutes_RestartSurvival`, `TestChatStoreMessagesReturnsCopy`, `TestChatWeb_ValidCap_History` |
| **PERSIST-03** | Thread deleted when session deleted; hard per-session message cap | COVERED | `TestEngineChatStoreFor_AfterKill`, `TestChatDeleteRemovesFile`, `TestChatDeleteIdempotent`, `TestChatCapEnforcement` |

---

## Manual-Only Verifications

All phase-151 behaviors have automated verification.

> Forward-looking note (not a phase-151 gap): true end-to-end chat — real inbound
> messages flowing through the relay/WS ingestion path and rendering in the GUI/web
> chat panel — cannot be exercised until Phase 152 (Relay Protocol + Identity +
> Presence) and the later UI phases exist. The REST contract those surfaces will
> consume is fully covered here.

---

## Validation Sign-Off

- [x] All tasks have automated verify (no Wave 0 dependencies)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none — fully covered)
- [x] No watch-mode flags
- [x] Feedback latency < 25s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-06-25
