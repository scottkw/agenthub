# Phase 166: Funnel Frontend + Help Guide - Context

**Gathered:** 2026-06-30
**Status:** Ready for planning

<domain>
## Phase Boundary

The frontend for internet-sharing a session via Tailscale Funnel, plus a Help article. Specifically: a risk-acknowledgment flow off the Share modal's Funnel toggle (with auto-expiry selector + Help cross-link), a persistent colorblind-safe "internet exposed" indicator on the Hub card and session tab, the public Funnel URL display (copy + QR + TLS "starting up…" state), one-click disable, the local-fallback disabled state, and a new in-app Help guide covering both the Funnel path and the device-share + ACL alternative.

Covers FUI-01..06 and HLP-01..02. The Funnel **backend** (daemon `SetSessionFunnel`, teardown, URL builders, `funnelActive`) shipped in Phase 165 — this phase only wires the UI to that on-ramp. No backend Funnel changes, no notifications (Phase 167), no bug fixes (Phase 168).

</domain>

<decisions>
## Implementation Decisions

### Risk-acknowledgment dialog (FUI-01, FUI-02, FUI-06)
- **D-01:** Presentation is an **inline expanding panel within the existing Share modal** — NOT a stacked/nested modal and NOT a full content-swap. Flipping the Funnel toggle reveals the risk panel inline; it collapses on Cancel or after enabling. Avoids modal-over-modal focus-trap complexity.
- **D-02:** Acknowledgment gesture is an **explicit confirm button** labeled "Enable internet share" (visually distinct from Cancel), sitting directly below the risk statement. No checkbox, no type-to-confirm. The deliberate click on the consequential button IS the acknowledgment.
- **D-03:** The panel is shown on **every enable** — no "don't show again" affordance (FUI-01 hard requirement).
- **D-04:** Panel content order: ⚠ risk statement ("reachable from the public internet; the join code is the only gate for its TTL; short-lived read-only shares recommended") → auto-expiry selector → Help cross-link ("Tighter containment? →" linking to the new Help guide, FUI-06) → [Cancel] [Enable internet share].

### Auto-expiry selector (FUI-02)
- **D-05:** Duration presets: **30m / 1h / 4h / 8h**, with **1h as the default**.
- **D-06:** Additionally offer an **"until I disable" (no auto-expiry)** option. ⚠ This deliberately relaxes the auto-expiry safety rail per explicit user choice — the persistent exposure indicator (D-08/D-09) and one-click disable (D-13) are the compensating controls for a non-expiring share. No custom-minutes input.
- **D-07:** The selector feeds the backend binding `SetSessionFunnel(sessionID, true, expirySeconds)`. **RESEARCH ITEM:** confirm how the Phase-165 daemon treats `expirySeconds <= 0` — the "until I disable" option must map to whatever sentinel (likely `0`) the backend already interprets as "no auto-expiry". Do NOT invent a new sentinel; match the existing `funnelExpiry` map semantics.

### Internet-exposure indicator (FUI-03) — colorblind-safe, hard constraint
- **D-08:** **Hub card** shows a full badge: **globe icon + "INTERNET"** (uppercase text). Heroicons `GlobeAltIcon` is available in the codebase.
- **D-09:** **Session tab** shows the **globe icon ONLY** (no visible text) to conserve tab width. To satisfy FUI-03's "text label" requirement and screen-reader accessibility, the tab globe MUST carry an `aria-label`/`title` of **"Internet exposed"** — the text label exists, it is just not rendered as visible chrome on the tab.
- **D-10:** Colorblind-safety: state is encoded by **icon shape (globe) + text**, never by color alone (the user is colorblind — verify at hex/source level, not by eye, per standing convention). The globe glyph itself is the shape cue on the tab.
- **D-11:** Placement is an inline pill/badge on the card header and inline on the tab — no full-width banner. Always visible while `funnelActive` is true; removed immediately when it goes false.

### Funnel URL display + warm-up (FUI-04, FUI-05)
- **D-12:** Show **one single public Funnel URL** in its **own clearly-labeled "Internet (public)" section**, visually separate from the existing tailnet read-only/full-access dual links. The Funnel URL uses the **read-only cap by default** (matches FUI-01's read-only recommendation; do not expose a public write link). Section includes copy-to-clipboard + QR (reuse `GetCapabilityQRCode`).
- **D-13:** **One-click disable** (FUI-04): a single control in the "Internet (public)" section calls `SetSessionFunnel(id, false, 0)`, which fully tears down the Funnel config (FNL-05) and clears the indicator immediately.
- **D-14:** **Warm-up UX (FUI-05, success criterion 3):** immediately after enable, show a **"Starting up… (TLS warming up)"** state with the URL/QR muted or disabled, and **poll** `funnelActive` (and/or probe URL reachability) until it actually answers, then reveal the live URL + QR + copy. The user must only ever see a working link. Needs a poll interval + a reasonable timeout/fallback message if warm-up never completes.
- **D-15:** **Local-fallback state (M-36 carry-forward):** when the web server is in local-network fallback mode (Tailscale not running, `webServerMode === 'local'`), the Funnel toggle is **disabled** with an inline explanatory note (e.g. "Internet sharing requires Tailscale running"). No risk panel, no `SetSessionFunnel` call, no serve config — matches the backend's fail-closed behavior. The toggle re-enables when running under tailscale mode.

### Help guide (HLP-01, HLP-02)
- **D-16:** Add a **new top-level Help section** titled **"Sharing Outside Your Tailnet"** (markdown file under `frontend/src/content/help/`, registered in `HelpTab.tsx` SECTIONS), placed **after "Chat" and before "Frequently Asked Questions"** in nav order. Follow the existing `?raw` markdown-import + `{ id, label, markdown }` pattern (id e.g. `help-sharing`).
- **D-17:** The article documents **both** paths: (1) the Funnel path (one-click internet share from the Share modal) and (2) the **contained device-share + ACL alternative** (share a single device, tag it `tag:agenthub`, restrict shared users to AgentHub's port).
- **D-18:** Include the **copy-pasteable `autogroup:shared` → `tcp:7443` grant** block, explicitly call out the **wildcard-default (`*→*`) gotcha**, and link to the relevant **Tailscale docs**. (HLP-02 — verify the port matches the app's actual web-share port; STATE references `:7443` for local web-share.)

### Claude's Discretion
- Exact pill/badge styling, spacing, animation of the inline risk panel, and the poll interval/timeout for warm-up are left to planning/implementation, provided they honor the decisions above and the colorblind-safety constraint.
- Precise Help-article prose, heading structure, and which specific Tailscale doc URLs to cite (researcher/planner to confirm current canonical URLs).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap
- `.planning/REQUIREMENTS.md` §FUI (FUI-01..06), §HLP (HLP-01..02) — the locked requirements this phase satisfies.
- `.planning/ROADMAP.md` "Phase 166" — goal + 5 success criteria (success criterion wording for the indicator and warm-up).

### Phase 165 backend on-ramp (the seam this phase wires to)
- `.planning/phases/165-funnel-backend/165-VERIFICATION.md` — what the backend actually delivers (`SetSessionFunnel`, teardown triggers, `funnelActive`).
- `.planning/phases/165-funnel-backend/165-04-PLAN.md`, `165-05-PLAN.md` / `165-05-SUMMARY.md` — loopback-HTTP Funnel architecture, co-location assumption, and the fail-closed local-fallback behavior (basis for D-15).
- `.planning/phases/165-funnel-backend/165-UAT.md` — M-34/M-35/M-36 findings; M-36 explicitly defers the Share-modal fallback wording/disable to this phase.

### Existing frontend surfaces to extend (full paths)
- `frontend/src/components/Hub/SessionShareModal.tsx` — two-toggle Share modal; the Funnel toggle + inline risk panel land here.
- `frontend/src/components/SessionSharePanel.tsx` — existing read/write link rows + QR (`GetCapabilityQRCode`); the new "Internet (public)" section follows this pattern.
- `frontend/src/components/Hub/SessionCard.tsx` — Hub card; D-08 badge.
- `frontend/src/components/HelpTab.tsx` — Help SECTIONS registry (D-16); `frontend/src/content/help/*.md` — existing articles to mirror.

### Wails bindings (already generated by Phase 165)
- `frontend/src/wailsjs/go/main/App.d.ts` — `SetSessionFunnel(sessionID, enabled, expirySeconds)`, `GetCapabilityQRCode(joinURL)`.
- `frontend/src/wailsjs/go/models.ts` / `wailsjs/wailsjs/go/models.ts` — `SessionInfo.funnelActive: boolean`.

### Standing conventions
- `frontend/.../` colorblind rule — verify color-based UI at hex/source level, not by eye (user is colorblind).
- `TESTING.md` — add new test files to the suite manifest + traceability map (repo convention).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `SessionSharePanel.tsx` `CodeDisplay` + QR action pattern (`GetCapabilityQRCode`) — directly reusable for the single Funnel URL section (copy + QR).
- Heroicons `GlobeAltIcon` (`@heroicons/react/24/outline`) — the indicator's non-color icon.
- `SessionShareModal.tsx` already receives `webServerMode` (`'tailscale' | 'local' | null`) and `webServerRunning` props — D-15's local-fallback disable can gate on `webServerMode !== 'tailscale'` without new plumbing.
- `HelpTab.tsx` SECTIONS array + `?raw` markdown import — the established pattern for adding the new Help article (mirrors the Phase-159/quick-task `chat.md` addition).

### Established Patterns
- Share modal is server-truth-seeded (`session.webEnabled`, `funnelActive`) and re-issues caps on restart — the Funnel UI state should likewise seed from `SessionInfo.funnelActive`, not local-only state (cross-surface parity is release-blocking).
- Modal animation phase machine (entering→open→exiting) + focus-return already exist in `SessionShareModal` — the inline risk panel rides inside this, no new modal lifecycle.

### Integration Points
- Funnel toggle → inline risk panel → `SetSessionFunnel(id, true, expirySeconds)` on confirm.
- `SessionInfo.funnelActive` drives both the indicator (D-08/D-09) and the URL section visibility; comes through the existing session-info poll.
- One-click disable → `SetSessionFunnel(id, false, 0)` → indicator clears on next poll/echo.

</code_context>

<specifics>
## Specific Ideas

- Risk panel ASCII reference the user approved:
  ```
  Share modal
  ┌─ [✕] Internet share ───────┐
  │ ⚠ This makes the session     │
  │   reachable from the public  │
  │   internet. Join code is the │
  │   only gate.                 │
  │ Auto-expire: [1h ▾]          │
  │ Tighter containment? → Help  │
  │ [ Cancel ]  [ Enable share ] │
  └──────────────────────┘
  ```
- Indicator: Hub card `⊕ INTERNET` (globe + word); session tab globe-only with "Internet exposed" tooltip/aria-label.

</specifics>

<deferred>
## Deferred Ideas

- Public **write** access over Funnel (read+write dual Funnel URLs) — deliberately out; public default is read-only. Revisit only if a real use case appears.
- Custom-minutes auto-expiry input — not needed; presets + "until I disable" cover the range.
- Automating device-share / ACL edits via the Tailscale admin API — explicitly out of v4.2 scope (FUT-01 / Issue #107); the Help guide documents the manual path only.

None of the above belong in Phase 166.

</deferred>

---

*Phase: 166-funnel-frontend-help-guide*
*Context gathered: 2026-06-30*
