---
phase: 158-chat-affordance-polish-fix-toggle-send-overlap-add-chat-to-t
plan: "02"
subsystem: frontend/chat-parity
tags: [chat, terminal-tab, parity, overlay, tdd]
dependency_graph:
  requires: [158-01]
  provides: [CHAT-PARITY-01]
  affects: [frontend/src/App.tsx, frontend/src/style.css, TESTING.md]
tech_stack:
  added: []
  patterns:
    - TerminalChatHost overlay host (mirrors WebShareSessionView desktop pattern)
    - TDD RED/GREEN cycle for behavioral component tests
key_files:
  created:
    - frontend/src/components/Hub/TerminalChatHost.tsx
    - frontend/src/components/Hub/TerminalChatHost.test.tsx
  modified:
    - frontend/src/App.tsx
    - frontend/src/style.css
    - TESTING.md
decisions:
  - "isActive forwarded from props (not chatOpen) — D-02 invariant: no PTY resize on toggle"
  - "TerminalChatHost wraps only TerminalPanel; StatusBar/ExitCountdownBanner stay as siblings in App.tsx terminal-wrapper"
  - "CSS .terminal-chat-host rule uses flex:1 1 auto so the host fills the column without hardcoding height"
  - "TerminalPanel import removed from App.tsx after replacement — tsc TS6133 unused-import error caught and fixed"
  - "Pre-existing style.hub.modal.test.ts failure (introduced by 158-01 lastIndexOf fragility) documented but not fixed — out of scope for 158-02"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-27"
  tasks_completed: 3
  files_changed: 5
status: complete
---

# Phase 158 Plan 02: Terminal-Tab Chat Affordance (CHAT-PARITY-01) Summary

Cross-surface chat parity gap closed: the GUI terminal tab now presents the same
chat toggle + ChatPanel overlay as the Hub interactive modal and web-share view.

## What Was Built

**TerminalChatHost** (`frontend/src/components/Hub/TerminalChatHost.tsx`) — a thin
overlay host component that mounts TerminalPanel + always-mounted ChatPanel +
hub-modal__chat-toggle button inside a single `.terminal-chat-host` div. Mirrors
WebShareSessionView's structure exactly. Props accept all TerminalPanel fields the tab
needs (sessionId, isActive, relayPort, fontSize, onFontSizeChange, theme, pluginConfig,
onWebGLContextLost, onRegisterSaver, onProgressChange, remote).

**App.tsx wiring**: replaced the bare `<TerminalPanel>` in the `tabs.map` terminal-tab
block with `<TerminalChatHost>`, forwarding all props. StatusBar and ExitCountdownBanner
remain as siblings inside `.terminal-wrapper` — outside the host, so the absolute drawer
does not cover the status bar.

**CSS**: `.terminal-chat-host` containing-block rule (position:relative; flex:1 1 auto;
min-height:0; display:flex; flex-direction:column) placed near `.terminal-wrapper`. The
unscoped `.chat-panel` overlay rule (158-01 / Phase 154) already handles the drawer
position:absolute inside any relative container. Reduced-motion suppression added to the
existing `@media (prefers-reduced-motion: reduce)` block.

**TESTING.md**: Section 2 manifest note (vitest +1, total 507), Section 4 CHAT-PARITY-01
traceability row, Section 5 manual item M-30.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | TerminalChatHost failing test | 751cb9b5 | TerminalChatHost.test.tsx (new) |
| 1 (GREEN) | TerminalChatHost implementation | ca18e341 | TerminalChatHost.tsx (new) |
| 2 | Wire into App.tsx + CSS | 3c86443c | App.tsx, style.css |
| 3 | TESTING.md registration | a1b840e1 | TESTING.md |

## TDD Gate Compliance

- RED gate: `test(158-02): add failing test for TerminalChatHost` (751cb9b5) — 11 tests, import-resolution failure confirmed.
- GREEN gate: `feat(158-02): implement TerminalChatHost` (ca18e341) — all 11 tests pass.
- No REFACTOR needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed unused TerminalPanel import from App.tsx**
- **Found during:** Task 2 — `tsc --noEmit` reported TS6133 "TerminalPanel declared but its value is never read" after replacing all usages with TerminalChatHost.
- **Fix:** Removed the `import { TerminalPanel } from './components/TerminalPanel'` line.
- **Files modified:** `frontend/src/App.tsx`
- **Commit:** 3c86443c

### Pre-existing Issue (Out of Scope — Documented Only)

**style.hub.modal.test.ts "animation: none" test** — this test uses `cssRaw.lastIndexOf('prefers-reduced-motion: reduce')` and scans 300 chars for `animation: none`. Plan 158-01 added a new `@media (prefers-reduced-motion: reduce)` block at the end of style.css (line ~6945), making it the new "last" reduce block — which contains only `transition: none`, not `animation: none`. The test was ALREADY FAILING before any 158-02 changes (verified via `git stash` regression run). This is a 158-01 scope issue; fixing the test's `lastIndexOf` fragility is deferred to the phase cleanup pass if needed.

## Verification Results

- `pnpm -C frontend exec vitest run src/components/Hub/TerminalChatHost.test.tsx`: **11/11 PASS**
- `pnpm -C frontend exec tsc --noEmit`: **0 errors**
- `pnpm -C frontend build` (vite build): **success**
- `bash tests/check-traceability-paths.sh`: **OK (exits 0)**
- HubInteractiveModal.test.tsx: **16/16 PASS** (no regression)
- Full suite: **2117/2118 pass** (1 pre-existing failure from 158-01 in style.hub.modal.test.ts)
- Manual M-30 recorded for live overlay/no-resize + cross-surface parity UAT

## Known Stubs

None — TerminalChatHost connects to the existing relay loopback path; no placeholder data.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes. The component reuses existing ChatPanel (same trust boundaries, same capability gating via HandleChatSend server-side).

## Self-Check

- [x] frontend/src/components/Hub/TerminalChatHost.tsx exists
- [x] frontend/src/components/Hub/TerminalChatHost.test.tsx exists
- [x] App.tsx imports TerminalChatHost and uses it in tabs.map
- [x] style.css contains .terminal-chat-host rule
- [x] style.css contains .terminal-chat-host .chat-panel transition:none in reduce block
- [x] TESTING.md contains CHAT-PARITY-01 traceability row
- [x] TESTING.md contains M-30 item
- [x] Commits: 751cb9b5, ca18e341, 3c86443c, a1b840e1 all present in git log
