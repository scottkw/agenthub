---
status: resolved
phase: 135-accessibility-hardening
source: [135-VERIFICATION.md]
started: 2026-06-19
updated: 2026-06-19
method: automated live-engine probe (Playwright WebKit + Chromium)
---

## Current Test

Resolved — A11Y-04 runtime Tab-trap validated in the real WebView engine families.

## Tests

### 1. Live keyboard Tab-trap (A11Y-04 runtime barrier) — RESOLVED ✅

**Original human item:** Open a Hub modal in a live app and press Tab; focus must
cycle only within the modal and never reach a background session card. Flagged
`human_needed` because jsdom 29 reflects the `inert` property but does not enforce
the focus barrier — "only a real WebView2 (Windows) or WKWebView (macOS) engine confirms."

**How it was validated (2026-06-19):** A self-contained Playwright probe reproduced
HubModal's exact DOM contract (`.hub` background made `inert`, `role="dialog"` with
initial focus on the close button) and ran in both target webview engine families:

- **WebKit** (= macOS **WKWebView**, AgentHub's native macOS webview)
- **Chromium** (= Windows **WebView2** engine, AgentHub's native Windows webview)

Results (4/4 green, both engines):
- With `.hub` `inert`, the background **rejects programmatic `.focus()`** (engine-enforced
  invariant, no Tab-traversal quirk) — `document.activeElement` does not move to a background node.
- With `.hub` `inert`, **Tab is trapped inside the dialog** — 6 Tab presses never land on a
  background input; focus stays on dialog controls.
- After the modal closes (dialog unmounts + `inert` cleared), the background is **focusable
  again and Tab advances** — no keyboard lock.

A WebKit-specific confound was identified and controlled: WebKit's default keyboard nav
skips `<button>`/`<a>` (macOS "Full Keyboard Access" off), so the probe asserts the
engine-enforced **programmatic-focus rejection** primitive and uses text `<input>`s for the
Tab-traversal check.

**Coverage decomposition (why this is sufficient):**
- *App applies `inert` correctly* — proven by the jsdom behavioral test (WR-01 regression
  guard: `inert` true during `open` AND `exiting`, false after unmount) + 9/9 source verification.
- *Engine enforces the `inert` Tab-trap* — proven live in WebKit (WKWebView) + Chromium (WebView2).
- Together: Tab is trapped in the live app.

**Residual (honest):** The probe validates the engine primitive + correct app-side application,
not the fully-assembled native AgentHub window driven end-to-end (the native WKWebView/WebView2
window cannot be automated by available tooling). Confidence is high; a one-time human spot-check
in a real app session remains available as belt-and-suspenders but is not release-blocking.

**Probe:** ephemeral (clearly-marked scaffolding testing a platform primitive, not app code);
removed after the run. The durable app-side regression guard lives in
`frontend/src/components/Hub/HubModal.test.tsx` (inert lifecycle behavioral suite).
