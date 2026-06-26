---
phase: 153
slug: session-pty-bridge
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-26
---

# Phase 153 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (Go 1.26.3) |
| **Config file** | none — `go test` discovers by convention |
| **Quick run command** | `go test -race -short -run 'TestSanitize|TestInject' ./internal/relay/... ./internal/webserver/...` |
| **Full suite command** | `go test -race -short ./...` |
| **Estimated runtime** | ~30–60 seconds (quick); full suite per repo norm |

---

## Sampling Rate

- **After every task commit:** Run `go test -race -short -run 'TestSanitize|TestInject' ./internal/relay/... ./internal/webserver/...`
- **After every plan wave:** Run `go test -race -short ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 153-01-* | 01 | 1 | SEC-02 | V5 / T-bidi,T-csi,T-osc,T-c1,T-newline | Only printable text + exactly one trailing `\n` reaches WriteInput | unit | `go test -race -short -run TestSanitizePTYText ./internal/relay/...` | ❌ W0 | ⬜ pending |
| 153-02-* | 02 | 1 | MENTION-02 | — | RW-cap MsgSessionInject writes sanitized text to PTY stdin + broadcasts SessionInject msg | integration | `go test -race -short -run TestInject_RWCap ./internal/relay/...` | ❌ W0 | ⬜ pending |
| 153-02-* | 02 | 1 | MENTION-03 | — | MsgChatSend / stray frame does NOT write to PTY; only MsgSessionInject does | unit | `go test -race -short -run TestInject_OnlyDedicatedFrame ./internal/relay/...` | ❌ W0 | ⬜ pending |
| 153-03-* | 03 | 2 | SEC-01 (relay) | V4 / T-eop | RO client MsgSessionInject → NAK frame; WriteInput never called | integration | `go test -race -short -run TestInject_ROCap_RelayPath ./internal/relay/...` | ❌ W0 | ⬜ pending |
| 153-03-* | 03 | 2 | SEC-01 (web) | V4 / T-eop | RO JWT client hand-crafted MsgSessionInject → NAK frame; WriteInput never called | integration | `go test -race -short -run TestInjectRO_WebPath ./internal/webserver/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · ❌ W0 = file created in Wave 0*

---

## Wave 0 Requirements

- [ ] `internal/relay/sanitize_test.go` — SEC-02 sanitizer corpus (LF/CR/CRLF, null bytes, CSI, OSC, C1 controls, bidi overrides)
- [ ] `internal/relay/server_inject_test.go` — MENTION-02, MENTION-03, SEC-01 (relay path, incl. adversarial RO frame)
- [ ] `internal/webserver/inject_test.go` — SEC-01 web path (adversarial RO JWT frame)
- [ ] TESTING.md registration — Suite Manifest §2 (Go count +3) + Traceability §4 rows for MENTION-02/03, SEC-01, SEC-02

*Existing `go test` infrastructure covers execution; only the new test files above are missing.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Rendered "→ injected into terminal" chat indicator | MENTION-02 (visual) | No composer/thread UI until Phase 154 | Deferred to Phase 154 UAT; Phase 153 proves the SessionInject:true message is persisted + broadcast (asserted in integration test) |

*All Phase 153 security behaviors have automated verification; only the deferred visual indicator is manual (and out of this phase's UI scope).*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
