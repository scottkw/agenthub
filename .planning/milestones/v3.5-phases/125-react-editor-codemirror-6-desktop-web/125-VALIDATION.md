---
phase: 125
slug: react-editor-codemirror-6-desktop-web
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-14
---

# Phase 125 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (React component/unit) + Playwright (cross-browser e2e: Chromium/Firefox/WebKit) + go test (If-Match/412 server change, vendor_drift_test.go) |
| **Config file** | frontend/ vitest config; playwright.config (Wave 0 may add); Go native |
| **Quick run command** | `cd frontend && pnpm test` (component) |
| **Full suite command** | `cd frontend && pnpm test && pnpm exec playwright test` + `go test ./internal/files/... ./internal/daemon/...` |
| **Estimated runtime** | component ~seconds; Playwright cross-browser ~minutes |

---

## Sampling Rate

- **After every task commit:** relevant component test (`pnpm test -- --run <Component>`) or `go test ./internal/files/...`
- **After every plan wave:** `cd frontend && pnpm test` + `go test -race ./internal/...`
- **Before `/gsd:verify-work`:** full Playwright cross-browser suite green; vendor_drift_test passes; zero CSP violations
- **Max feedback latency:** component <30s; e2e is the pre-verify gate

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (planner fills) | — | 0 | EDIT-05/08 | concurrency | If-Match mismatch → 412 | go unit | `go test ./internal/files/...` | ❌ W0 | ⬜ pending |
| (planner fills) | — | 1-4 | EDIT-01..12 | — | editor + write affordances gated on canWrite | vitest | `cd frontend && pnpm test` | ❌ W0 | ⬜ pending |
| (planner fills) | — | 5 | EDIT-13 | CSP/authz | cross-browser e2e, zero CSP violations | playwright | `pnpm exec playwright test` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Server: `Handler.Write` reads `If-Match`, returns 412 on mismatch; read route emits `ETag` (net-new — EDIT-05/08)
- [ ] `internal/files/` (or webserver) If-Match/412 unit tests
- [ ] `vendor_drift_test.go` — package.json ↔ pnpm-lock CodeMirror version parity gate (EDIT-01)
- [ ] Playwright fixture: add a `WRITE_CAP` variant carrying `files.write` (existing fixtures only mint read/files.read)
- [ ] Playwright config for Chromium + Firefox + WebKit (if not present)

*Framework present (vitest + go test); Playwright may need install in Wave 0.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CodeMirror Tab / Cmd-V inside the Wails WebView vs Phase 49 clipboard handler | EDIT (editor) | Wails WebView keyboard/clipboard interaction not reliably Playwright-drivable on desktop | Open editor in desktop app, test Tab indent + paste; confirm no conflict with the app clipboard handler |
| Desktop GUI visual render of editor + affordances | EDIT-01..12 | Wails desktop app not headless-automatable | Open a file, edit, save, exercise affordances in the running app |

*Web-share surface IS Playwright-automatable; desktop Wails interactions are the manual residue.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency acceptable (component <30s)
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
