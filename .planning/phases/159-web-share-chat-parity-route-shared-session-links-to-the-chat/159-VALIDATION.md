---
phase: 159
slug: web-share-chat-parity-route-shared-session-links-to-the-chat
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-27
---

# Phase 159 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib `net/http/httptest`) |
| **Config file** | none — existing Go test infra (`internal/webserver/server_test.go`) |
| **Quick run command** | `go test ./internal/webserver/ -run 'TerminalPage\|WebServerToggle' -count=1` |
| **Full suite command** | `go test ./internal/webserver/... -count=1` |
| **Estimated runtime** | ~5–15 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 159-01-01 | 01 | 1 | WEBCHAT-01 | — | `handleTerminalPage` issues 302 to `/app/?session=&cap=` only after cap validation | unit | `go test ./internal/webserver/ -run TerminalPageRedirect -count=1` | ❌ W0 | ⬜ pending |
| 159-01-02 | 01 | 1 | WEBCHAT-01 | — | session-id + cap token survive redirect URL-encoded (incl. JWT `+/=`) | unit | `go test ./internal/webserver/ -run TerminalPageRedirect -count=1` | ❌ W0 | ⬜ pending |
| 159-01-03 | 01 | 1 | WEBCHAT-02 / PARITY-01 | — | existing `TestWebServerToggle` updated to assert 302 + Location (no redirect-follow) | unit | `go test ./internal/webserver/ -run WebServerToggle -count=1` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/server_test.go` — add `TestTerminalPageRedirect` (asserts 302, Location query params, URL-encoding of cap token); reuse existing test cap/session fixtures

*Existing infrastructure (httptest + cap fixtures in `server_test.go`) covers the rest.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Remote guest opening the ACTUALLY-SHARED `/sessions/{id}?cap=` link lands on a chat-capable surface, can send/receive chat, AND keeps Phase 157-04 host-authority scaling through the redirect | WEBCHAT-02, PARITY-01 (+ Phase 157-04 regression guard) | web-share WS blocks automated input (per `reference_live_uat_daemon_gotchas`); must drive a real shared link in a live daemon; the scale check needs a real rendered xterm with live host-authority resize frames | Produce a real share link via the daemon (issue cap for a live session), open the `/sessions/{id}?cap=` URL in a browser, confirm redirect to `/app/`, confirm ChatPanel + toggle + unread/mention badge + presence render and a chat round-trips. Then resize the host PTY and confirm the redirected SPA guest re-scales to honor the new host-authority grid (downscale-to-fit, cap at 1.0, no clipping — guest does not drive its own grid). Register as M-31 in TESTING.md. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
