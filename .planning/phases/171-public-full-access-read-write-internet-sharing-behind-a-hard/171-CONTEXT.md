# Phase 171: Public Full-Access (Read-Write) Sharing - Context

**Gathered:** 2026-07-07
**Status:** Ready for planning

<domain>
## Phase Boundary

A session owner can opt into **public read-write** Funnel sharing behind a distinct hold-to-confirm consent gate in the desktop Share modal, receiving a full-access public URL + **single-use** write code with unmistakable, colorblind-safe consent. Public terminal write becomes reachable **only** through that gate, replacing today's accidental unconditional funnel-origin write acceptance. Terminal command execution only — never public `files.write`. Short-lived (15m default / 1h hard max), single writer, immediate revocation at teardown.

This is a net **security improvement** over today's state (public RW already works, but unintentionally and unlabeled), not a new exposure.

</domain>

<spec_lock>
## Requirements (locked via SPEC.md)

**8 requirements are locked.** See `171-SPEC.md` for full requirements, boundaries, and acceptance criteria.

Downstream agents MUST read `171-SPEC.md` before planning or implementing. Requirements are not duplicated here.

**In scope (from SPEC.md):**
- Hold-to-confirm public-RW consent gate in the desktop GUI Share modal (owner-initiated)
- Single-use public write code minted by the gate; one-writer semantics; read-many + write-one coexistence with the Phase 170 reusable read code
- Terminal-only public write scope (no public `files.write`)
- Closing the accidental path: write accepted over the Funnel origin only for a gated RW session
- Short bounded RW expiry (15m default / 1h hard max; API-layer clamp)
- Immediate write-cap revocation at the teardown chokepoint (disable / funnel-off / session-exit / expiry)
- Distinct, colorblind-safe RW UI treatment (label + icon + shape)
- Threat model via `/gsd-secure-phase`

**Out of scope (from SPEC.md):**
- Public `files.write` (remote file editing/upload over the internet)
- Guest-initiated write requests (read guest self-promotion) — RW stays owner-initiated only
- CLI/headless exposure of the RW gate — desktop-GUI only
- Multiple simultaneous public writers — one-writer model
- Reusable write codes — explicitly rejected (leaked reusable write code = persistent public RCE)
- Owner re-issue of a fresh write code after consumption — single-use; re-share requires re-gating

</spec_lock>

<decisions>
## Implementation Decisions

### Enforcement & Revocation (R4, R6)
- **D-01: Gate the write grant (primary enforcement).** Stop registering the write grant at cap-issue time. Register the funnel write grant via `ws.AddGrant` **only** when the ≥3s hold-gate completes, and de-register it at the single `disableFunnelForSession` teardown chokepoint. A non-gated write cap then fails grant validation → 401; teardown de-registration gives instant revoke (R6's "next request 401") for free. Mirrors Phase 170's `funnelReadCode` lifecycle. No cap-crypto change (HMAC/JWT/40-bit entropy unchanged — SPEC constraint).
- **D-02: Defense-in-depth — `originAllowedForWrite` also becomes gate-aware.** In addition to grant-gating, `originAllowedForWrite` (`internal/webserver/capability_mw.go:191-210`) rejects funnel-origin **writes** for a session that has not passed the RW gate. Two independent barriers on an internet-RCE surface: a future bug that mis-registers a grant still can't grant public write without the gate. Maps most literally to R4's "unconditional funnel-origin write acceptance removed." secure-phase should verify both barriers.
- **D-03: Single per-session "RW-gated" state is the source of truth** for BOTH barriers (the conditional grant registration AND the `originAllowedForWrite` check). Set on hold-completion, cleared at the `disableFunnelForSession` chokepoint (covers disable / funnel-off / session-exit / expiry — all four teardown triggers).
- **D-04: Remove the accidental write-URL/code funnel-rebasing.** `issueCapabilitiesForSession` (`internal/daemon/api.go:1451-1472`) currently rebases `writeURL` to the Funnel base and issues `writeCode` on every call — this is the accidental public-write path. Remove it: the tailnet/local "Full Access Link" stays on the **tailnet** base (unchanged, out of scope to alter), and the **public funnel write cap + single-use code are minted only by the gate handler**, never at ordinary cap-issue time.
- **D-05: Terminal-only scope at gate issuance (R3).** The gate-minted public write cap grants terminal input (PTY write) only — it must strip `files.write` (and file perms) even when the session's local browse is enabled. The defense-in-depth barrier rejects `files.write` over the Funnel origin regardless of browse setting.

### Consent Gate UX (R1)
- **D-06: Separate "Danger" section.** The hold-to-confirm control lives in its own clearly-headed section (e.g. "Public write access — command execution"), physically below/separated from the Phase-166 "Internet (public)" read section, so it can never be mistaken for or mis-clicked alongside the benign read enable.
- **D-07: Linear progress bar + label hold feedback.** The ≥3s hold shows a horizontal fill bar advancing left→right with a "Holding… keep pressing" label; releasing before 3s resets the bar to 0 and issues nothing (geometry-based, colorblind-safe — not color alone).
- **D-08: "Risk-forward" consent copy baseline** (wordsmith in planning; must satisfy the SPEC prohibition — state "command execution" AND "anyone with the link", no softening):
  - Heading: "⚠ You are exposing a terminal to the internet"
  - Body: "Anyone with the link and code gets full command execution on this machine, running as your account, until you disable it or it expires (max 1 hour). A leaked link = remote code execution."
  - Control: "Hold 3s to confirm"

### RW Indicator Design (R7, colorblind-safe — hard project rule)
- **D-09: Distinct in label + icon + shape (never color alone).** Read indicator today (Phase 166) = rounded green pill, `GlobeAltIcon` + "INTERNET" text (`.hub-internet-badge`, color as reinforcement only). The RW indicator differs on all three axes:
  - **Label:** "FULL ACCESS" (consistent with existing "Full Access Link" vocabulary; heavier than "INTERNET")
  - **Icon:** `LockOpenIcon` (exposed/unlocked access) — distinct glyph from `GlobeAltIcon`
  - **Shape:** a notched / angled-corner "warning" shape — distinct geometry from the read pill's thin rounded edge; distinguishable in grayscale by shape + glyph alone
- **D-10: Same surfaces as the read badge.** The "FULL ACCESS" RW variant must render wherever the read INTERNET badge appears today (SessionCard `.hub-internet-badge`, TabBar `.tab__internet-icon`), for a gated-write session.

### Expiry & Post-Gate UX (R2, R5, R6)
- **D-11: Expiry options 15m / 30m / 1h, default 15m.** No "until I disable" / unbounded option. API-layer clamp: `ExpiresIn == 0` or `> 3600s` → effective exactly 1h (guards the API, not just the dropdown — per SPEC R5 edge coverage).
- **D-12: Write-code redemption window = share expiry.** The single-use write code stays redeemable for the whole owner-chosen share lifetime (≤ 1h), then dies — one coherent timer. Still single-use: the first `/join/exchange` redemption grants the cap and deletes the code; a second redemption fails closed ("code used"). (Diverges from the fixed 5-min `joinCodes` window; researcher to confirm the mechanism against the `joinCodes` manager / Phase 170 `IssueReusable`.)
- **D-13: Owner post-gate display.** After the hold completes: show the public write URL + single-use code + a live countdown to expiry + a disable button. After the one writer redeems the code: collapse to a "Write code used — one writer connected" state. Re-share requires **re-holding the gate** (single-use; owner re-issue is out of scope). On expiry/disable: revert to the collapsed "Enable public write…" affordance.

### Claude's Discretion
- Exact placement/DOM structure of the Danger section within the Share modal, precise motion timing of the progress bar, and the notched-shape CSS geometry — captured intent above; downstream may refine as long as D-06/D-07/D-09 constraints hold.
- Whether the per-session RW-gated state lives on the existing session struct vs a dedicated map (D-03) — researcher to pick against the live api.go / webserver structures.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Spec & security (MANDATORY)
- `.planning/phases/171-public-full-access-read-write-internet-sharing-behind-a-hard/171-SPEC.md` — the 8 locked requirements, boundaries, acceptance criteria, edge coverage, and 4 must-NOT prohibitions. MUST read before planning.
- `/gsd-secure-phase 171` output (`.planning/phases/171-.../171-SECURITY.md`, produced next) — threat model asserting no public write path exists except the gate (R8). Route is spec → discuss → **secure** → plan; do not shortcut to plan-phase.

### Backend integration points
- `internal/daemon/api.go` §`issueCapabilitiesForSession` (~1420-1500) — read/write cap minting, funnel rebasing (D-04 removes write rebasing), `joinCodes.Issue`, Phase 170 `funnelReadCode` / `IssueReusable` caching under `a.mu`, and the `disableFunnelForSession` teardown chokepoint (D-01/D-03 revocation site).
- `internal/webserver/capability_mw.go` §`originAllowedForWrite` (191-210) — the dual-origin write check; D-02 makes it gate-aware.
- Capability grant registry (`ws.AddGrant`, api.go:1448-1449) — the D-01 enforcement surface (register on gate, remove on teardown).

### Frontend integration points
- `frontend/src/components/SessionSharePanel.tsx` — Read-Only + Full Access rows + Phase-166 "Internet (public)" section (`funnelEngaged`, `publicReadCode`); the Danger section (D-06) is added here.
- `frontend/src/components/Hub/FunnelRiskPanel.tsx` — existing public-internet enable risk gate (read side); reference for tone/consent pattern.
- `frontend/src/style.css` — Phase 166 internet-badge tokens & classes (`--hub-internet-badge-*`, `.hub-internet-badge`, `.tab__internet-icon`, ~4705-4770, ~7178-7205); the RW "FULL ACCESS" badge (D-09/D-10) is added alongside.
- `frontend/src/content/help/sharing-guide.md` — sharing help copy; may need an RW section.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Grant registry (`ws.AddGrant` / grant validation)** — already the authz gate for caps; D-01 reuses it as the single write-enforcement surface (conditional registration + teardown removal).
- **Phase 170 `funnelReadCode` / `IssueReusable` + `disableFunnelForSession` chokepoint** — the exact lifecycle template for a per-session, teardown-revoked code; the write code + gated state (D-03) mirror it (read-many code coexists with the write-one code — must not regress FNL-08).
- **`joinCodes` manager (`api.go:304`, `NewJoinCodeManager(5*time.Minute)`) + `/join/exchange`** — single-use redemption path reused for the write code (D-12 extends the redemption window to share expiry).
- **Phase 166 `.hub-internet-badge` + tokens** — the colorblind-safe badge pattern (icon+text carry state, color is reinforcement); the RW badge (D-09) is a distinct sibling.
- **`SessionSharePanel` Internet section + `FunnelRiskPanel`** — established consent/exposure UX to sit the Danger section beside.

### Established Patterns
- **Colorblind-safe rule (hard):** state must be carried by icon/text/shape, never color alone — D-07 and D-09 both honor it (progress geometry; label+icon+shape).
- **Single teardown chokepoint (Phase 165/170):** all revocation funnels through `disableFunnelForSession` — D-01/D-03 must not add a second teardown path (SPEC constraint).
- **Fail-closed origin checks (Phase 165 FNL-04):** `originAllowedForWrite` already fails closed on empty BaseURL; D-02 extends the same posture to the gate check.
- **Stateless HMAC-JWT caps + in-memory grants:** caps can't be individually "unsigned"; revocation is always via grant de-registration / server state, never crypto — reinforces D-01 over a token-claim-only approach.

### Integration Points
- Daemon API: new gate handler (mints funnel write cap + single-use code, sets RW-gated state, registers write grant); `disableFunnelForSession` clears state + de-registers grant + expiry timer.
- Webserver: `originAllowedForWrite` consults RW-gated state; write path enforces terminal-only perms over Funnel origin.
- Frontend: Wails RPC to invoke the gate on hold-completion; SessionSharePanel Danger section + countdown + used-state; RW badge on SessionCard/TabBar.

</code_context>

<specifics>
## Specific Ideas

- Consent copy baseline is the **"Risk-forward"** variant (D-08) — deliberately alarming, states RCE explicitly. Do not soften in planning.
- RW indicator label is specifically **"FULL ACCESS"** with a **LockOpen** icon in a **notched/warning** shape (D-09) — the user is colorblind; verify distinctiveness at the source/DOM/grayscale level, not by eye.
- Expiry set is specifically **15m / 30m / 1h** (default 15m) — no 5m, no unbounded (D-11).

</specifics>

<deferred>
## Deferred Ideas

- **Guest-side write UI / rejection UX polish** (what the write guest sees, error states on "code used") — surfaced as a possible "explore more" area but not deep-dived; standard fail-closed 401/"code used" messaging is sufficient for this phase. Revisit if UAT surfaces confusion.
- Owner re-issue of a fresh write code without re-gating — explicitly out of scope (SPEC); single-use, re-share requires re-hold.
- Public `files.write`, guest-initiated write promotion, CLI gate, multi-writer — all SPEC out-of-scope; future phases only if requested.

None of the above are in scope for Phase 171.

</deferred>

---

*Phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard*
*Context gathered: 2026-07-07*
