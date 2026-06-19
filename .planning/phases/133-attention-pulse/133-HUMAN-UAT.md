---
status: partial
phase: 133-attention-pulse
source: [133-VERIFICATION.md]
started: 2026-06-16
updated: 2026-06-19
---

## Current Test

[awaiting live testing — 4 runtime items; all require a session in a waiting/errored/non-zero-exit (attention) state, which needs the native webview to drive a terminal — the external `wails dev` browser bridge cannot reach the PTY]

## Tests

### 1. Pulse animation visual fidelity (ATTN-01)
expected: a waiting/errored/exited-non-zero card shows an amber-gold (#e0af68 dark / #b45309 light) pulsing border + BellAlertIcon; under prefers-reduced-motion: reduce the border is static (no animation) while icon + position still convey attention. Source-verified: tokens, keyframe gated under no-preference, static fallback, COLORBLIND-SAFE comments.
result: [pending]

### 2. Debounced float-to-top + FLIP timing (ATTN-02/04)
expected: when a card enters attention state it floats to the top OF ITS GROUP after a ~1s debounce, animating smoothly (FLIP slide), not jumping. Code reviewer specifically flagged this timing for live confirmation. Source-verified: two-memo sorted order gated on debouncedSortKey; single useLayoutEffect FLIP with capture-in-cleanup; reduced-motion gate.
result: [pending]

### 3. Collapsed-group attention badge (ATTN-06)
expected: a COLLAPSED group sidebar item containing an attention card shows the attention badge (BellAlertIcon + count); hidden when expanded; replaces the needs-input badge when attnCount>0. Source-verified: condition fixed to `collapsed && attention>0`; the Phase 132 inversion bug fixed.
result: [pending]

### 4. ATTN-05 end-to-end clear via modal
expected: resolving a waiting session from inside its modal clears the card's attention with no reload. Mechanism (status-driven clear) is proven via unit test (waiting→running clears live); the full modal path is BLOCKED ON PHASE 134 (modal doesn't exist yet). Re-confirm once Phase 134 lands.
result: [pending — modal path deferred to Phase 134]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 1

## Gaps

- Items 1-3 need a real attention session (waiting / errored / non-zero exit). The external `wails dev` browser bridge cannot drive the terminal PTY to produce one, so these are operator checks on a built/native app (e.g., run a command that exits non-zero in a session → stopped-err → attention). Item 4's modal trigger is deferred to Phase 134; the underlying status-driven clear is unit-proven now.

## Live Re-test 2026-06-19 (post-milestone tech-debt review)

Re-attempted via `wails dev` + dev-browser. No new live confirmation possible: all 4 items require a session in a waiting/errored/non-zero-exit (attention) state, and the external bridge cannot drive the PTY to produce one. Confirmed empirically this run that killing or exiting a session in this env *removes* it from the daemon rather than leaving an attention/stopped card (daemon was in a version-mismatch state — treated as an env artifact, not a finding). Item 4's modal now EXISTS (Phase 134 shipped), so the end-to-end ATTN-05 path is testable on a clean native build, but still needs a real waiting session. Underlying status-driven behavior remains unit-proven (HubPanel.test.tsx ATTN-03/05, SessionCardGrid FLIP/sort, GroupSidebar collapsed-badge). Operator confirmation on a clean native build still required for items 1-4.
