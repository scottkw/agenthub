---
phase: 49
slug: app-menus-version-injection
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-06
---

# Phase 49 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) / go test (backend) |
| **Config file** | `frontend/vitest.config.ts` / Go standard |
| **Quick run command** | `cd frontend && npx vitest run --reporter=verbose` |
| **Full suite command** | `cd frontend && npx vitest run && cd ../&& go test ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npx vitest run --reporter=verbose`
- **After every plan wave:** Run `cd frontend && npx vitest run && cd ../ && go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 49-01-01 | 01 | 1 | MENU-01 | integration | `go build -tags wailsassets ./...` | ✅ | ⬜ pending |
| 49-01-02 | 01 | 1 | MENU-02 | manual | macOS menu bar inspection | N/A | ⬜ pending |
| 49-02-01 | 02 | 1 | VER-01 | unit | `cd frontend && npx vitest run` | ✅ | ⬜ pending |
| 49-02-02 | 02 | 1 | VER-02 | integration | `go build -ldflags "-X main.Version=test" ./...` | ✅ | ⬜ pending |
| 49-03-01 | 03 | 1 | UI-01 | visual | grep for border-radius in CSS | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| macOS menu bar shows File/Edit/Window/Help | MENU-02 | Native OS menu requires visual inspection | Build app, launch, verify menu bar items |
| Cmd+C/V/X/Z work in xterm.js terminals | MENU-01 | Keyboard shortcut behavior in terminal context | Open terminal tab, type text, use Cmd shortcuts |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
