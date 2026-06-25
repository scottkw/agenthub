# Phase 141 — Design-Fidelity Gap (post-hoc)

**Recorded:** 2026-06-21
**Why this exists:** Phase 141 verified `passed` (4/4) but the running app does NOT match the user's
canonical design comp. The verification checked color-token migration / ARIA / reduced-motion, but
never rendered the canonical comp (`agenthub-v4.0-redesign/AgentHub UI redesign/AgentHub Redesign (standalone).html`
+ `c-*.png`) and compared. The redesign's visual language was never adopted; every 141 plan swapped
hardcoded hex → `var(--hub-*)` with the tokens set to the SAME TokyoNight values, so dark mode is
pixel-identical to pre-141. This is the corrective scope for gap-closure (user chose: gap-closure on 141,
"match the comp closely").

## Canonical target
- Design source of truth: `agenthub-v4.0-redesign/AgentHub UI redesign/AgentHub Redesign (standalone).html`
- Per-surface render targets: `c-welcome.png`, `c-session.png`, `c-filebrowser.png`, `c-settings.png`
  (+ `c-sessions.png` / `c-remote.png` map onto Hub-first structure — structure wins, see RDS-03)
- Verification this time MUST be a rendered side-by-side comparison of app vs comp per surface.

## Delta (comp → app)

### Typography (biggest miss — user explicitly flagged "fonts are the same")
- UI font: **Plus Jakarta Sans** (comp) vs system-ui/-apple-system (app). Never adopted.
- Mono font: **JetBrains Mono** (comp) vs SF Mono/Menlo (app). Never adopted.
- Must be **vendored locally** — repo bans CDN fonts (enforced by `vendor_drift_test.go`). Add woff2 files,
  `@font-face`, and font-family tokens; apply across all surfaces (terminal xterm font included where appropriate).
- Type scale: ~14px base / 12.5px small / 19px heading; weight 600 for emphasis.

### Color palette (values must change, not just be tokenized)
- Backgrounds darker/cooler: `#14151b` / `#16181f` / `#181a21` / `#1c1e28` vs TokyoNight `#1a1b26`.
- Grays/text: `#7e8294`, `#c7cad6`, `#9398a8`, `#54586a`, `#f4f5f8`.
- Semantic: green `#4ade80`, amber `#fbbf24`, teal `#43ddb2`, orange `#e08a66`, purple `#b98bff`.
- Keep colorblind-safe semantics (icon/shape/label + color) — re-verify at hex level (user is colorblind).

### Shape / spacing
- Border-radius: comfortable 8–11px on panels/cards/inputs; 999px pills for filter chips.
- Comfortable density per Direction 01 "Refined Native".

## Locked decisions carried in
- **Accent stays BLUE** (`#7aa2f7` dark / `#3d6fe8` light) — Phase 140 D-05 rejected comp's violet `#7C8CFF`
  for WCAG/colorblind reasons. Adopt everything else from the comp. (User may override to violet — one token.)
- Hub-first structure wins over comp's pre-Hub Sessions/Remote pages (RDS-03).
- prefers-reduced-motion guards retained (RDS-04).

## Also in scope (user-requested)
- **Light/dark theme toggle** wired into Settings → Appearance (currently `[data-ui-theme=light]` exists in CSS
  but nothing in the UI sets it — there is no way for a user to reach the light theme).
