# Phase 171: Public Full-Access (Read-Write) Sharing — Specification

**Created:** 2026-07-07
**Ambiguity score:** 0.15 (gate: ≤ 0.20)
**Requirements:** 8 locked

## Goal

A session owner can opt into **public read-write** Funnel sharing behind a distinct hold-to-confirm consent gate, receiving a full-access public URL + **single-use** write code with unmistakable, colorblind-safe consent — and public terminal write becomes reachable **only** through that gate, replacing today's accidental funnel-origin write acceptance.

## Background

Grounded in current code:

- `issueCapabilitiesForSession` (`internal/daemon/api.go:1451-1470`) rebases **both** `readURL` and `writeURL` to the Funnel base (`FunnelBaseURL()`) when the session is funnel-active, by cap-issue timing.
- `originAllowedForWrite` (`internal/webserver/capability_mw.go:191-210`) accepts a write capability over the **Funnel origin unconditionally** whenever `FunnelBaseURL() != ""` — there is no read-only downgrade for the public surface, and the Funnel serve config maps `/` → the whole mux with no path restriction.
- `SessionSharePanel` renders a "Full Access Link" row whenever `writeURL`/`writeCode` are provided (`frontend/src/components/SessionSharePanel.tsx:106-116`, always-on since Phase 137), so a funnel-rebased **public write URL** is already surfaced.
- Join codes are 5-min single-use via the shared `a.joinCodes` manager (`api.go:304`, `NewJoinCodeManager(5*time.Minute)`); Phase 170 added a **reusable read** code (`IssueReusable`, 8h backstop). No write-specific gate, no write-specific expiry, no distinct RW consent UI exists.

**Net effect today:** public read-write already works, but *unintentionally and unlabeled* — a leaked Full Access link over Funnel grants internet-reachable command execution (RCE) on the host, gated only by a 40-bit code + TLS. This phase replaces that with a deliberate, gated, short-lived flow and closes the accidental transport-layer path. This is a **net security improvement**, not a new exposure.

## Requirements

1. **Hold-to-confirm consent gate**: Public read-write is enabled only behind a distinct hold-to-confirm gate in the desktop Share modal.
   - Current: No RW gate exists; the Full Access link is surfaced with no extra consent, and write is accepted over Funnel unconditionally.
   - Target: Enabling public RW requires a press-and-hold action (≥ 3s) on a dedicated control, visually and textually distinct from the read-only Funnel enable; the write URL + single-use write code are issued **only** after the hold completes.
   - Acceptance: Releasing the hold before 3s issues no write URL/code and grants no public write; completing the hold issues exactly one single-use write code and enables the public write path for that session.

2. **Single-use write code, one writer**: The public write code grants write to exactly one guest.
   - Current: Write codes use the shared 5-min single-use manager but there is no public-write-specific issuance or "one writer" guarantee.
   - Target: The gate mints a single-use write code; the first guest to redeem it at `/join/exchange` receives the write capability and the code is deleted; a second redemption of the same code fails closed with an error.
   - Acceptance: Guest A redeems the write code → write granted; guest B redeems the same code → 401/"code used"; the reusable public **read** code continues to resolve for spectators throughout (read-many, write-one).

3. **Terminal-only write scope**: Public write grants command execution only, never file write.
   - Current: The write cap perms are `read,write` (browse off) or `read,write,files.read,files.write` (browse on); over Funnel that `files.write` bit is reachable.
   - Target: The public write capability grants terminal input (write to the PTY) only; it never includes `files.write`, even when local file browse is enabled for the session.
   - Acceptance: A public write guest can send terminal input; a `files.write` operation (remote file edit/upload) over the Funnel origin is rejected regardless of the session's browse setting.

4. **No public write except through the gate**: The accidental funnel-origin write path is closed.
   - Current: `originAllowedForWrite` accepts any valid write cap over the Funnel origin whenever Funnel is active.
   - Target: A write capability is accepted over the Funnel origin **only** when the owner has passed the RW consent gate for that session; otherwise it is rejected. The unconditional funnel-origin write acceptance / cap-rebasing is removed.
   - Acceptance: With Funnel on but RW **not** gated, a write cap presented over the Funnel origin is rejected (401/403) while the read path still works; after the gate is passed, the gated write cap is accepted.

5. **Short bounded expiry**: RW shares are short-lived with a hard cap.
   - Current: Funnel shares support owner-chosen expiry up to an 8h code backstop / "until I disable" (`funnelReadCodeMaxTTL = 8h`).
   - Target: A public read-write share auto-expires with a **15-minute default** and a **1-hour hard maximum**; "until I disable" (unbounded) is not offered for RW. A requested expiry of 0 or > 1h is clamped to 1h.
   - Acceptance: The RW enable UI offers only ≤ 1h options (default 15m, no "until I disable"); an API request with `ExpiresIn == 0` or `> 3600s` for an RW share results in an effective expiry of exactly 1h.

6. **Immediate revocation on teardown**: Disabling RW cuts off the writer at once.
   - Current: Phase 170 revokes the reusable read code at the single `disableFunnelForSession` teardown chokepoint; there is no write-cap revocation.
   - Target: Disabling RW — or funnel-off, session-exit, or expiry — revokes the write capability immediately at the teardown chokepoint; the active writer's next request returns 401. Read spectators are unaffected.
   - Acceptance: After the owner disables RW, the active writer's next terminal input returns 401 (no new write authorized once disable returns); the reusable read code still resolves and read spectators keep viewing.

7. **Unmistakable, colorblind-safe RW indicator**: A public-write link can never be confused with a public-read link.
   - Current: The Internet (public) section uses a single green "INTERNET" read treatment; there is no RW-specific visual.
   - Target: A public read-write link is distinguished from a public-read link by a distinct **text label** (e.g. "FULL ACCESS / READ-WRITE"), a distinct **icon**, AND a distinct **shape/border** — not color alone — and is colorblind-safe (per the project's colorblind-safe rule).
   - Acceptance: The RW indicator differs from the read indicator in label AND icon AND shape (verifiable at the source/DOM level, not by color perception); a colorblind reviewer (or grayscale render) can still tell RW from read.

8. **Threat model**: The phase is covered by a threat model asserting the closed perimeter.
   - Current: No threat model covers public write; the accidental path is undocumented.
   - Target: `/gsd-secure-phase` produces a threat model that asserts **no public write path exists except through the gate**, enumerates the RCE risk, and confirms mitigations (single-use code, 1h cap, immediate revocation, terminal-only scope).
   - Acceptance: A SECURITY.md exists for Phase 171 whose threats map to mitigations in the implemented code, with no open write-path threat.

## Boundaries

**In scope:**
- Hold-to-confirm public-RW consent gate in the desktop GUI Share modal (owner-initiated)
- Single-use public write code minted by the gate; one-writer semantics; read-many + write-one coexistence with the Phase 170 reusable read code
- Terminal-only public write scope (no public `files.write`)
- Closing the accidental path: write accepted over the Funnel origin only for a gated RW session
- Short bounded RW expiry (15m default / 1h hard max; API-layer clamp)
- Immediate write-cap revocation at the teardown chokepoint (disable / funnel-off / session-exit / expiry)
- Distinct, colorblind-safe RW UI treatment (label + icon + shape)
- Threat model via `/gsd-secure-phase`

**Out of scope:**
- **Public `files.write`** (remote file editing/upload over the internet) — minimizes the internet-reachable surface; public write is terminal command execution only.
- **Guest-initiated write requests** (a read guest asking to be promoted from the guest UI) — RW stays owner-initiated only; avoids a guest-driven escalation surface.
- **CLI/headless exposure of the RW gate** — the hold-to-confirm gate is desktop-GUI only; a headless consent gate has no unmistakable-consent affordance.
- **Multiple simultaneous public writers** — the model is one-writer (single-use code); concurrent multi-writer arbitration is a separate concern.
- Reusable write codes — explicitly rejected (a leaked reusable write code = persistent public RCE).
- Owner re-issue of a fresh write code after consumption — not in this phase (single-use, one writer; re-share requires re-gating).

## Constraints

- **Security-sensitive (internet RCE).** Route is spec → discuss → `/gsd-secure-phase` (threat model) → plan; do not shortcut to plan-phase.
- Must reuse the existing `disableFunnelForSession` single teardown chokepoint for write-cap revocation (consistency with Phase 170; no second teardown path).
- Must not regress Phase 170's reusable read code (FNL-08) — read coexists with write.
- Colorblind-safe treatment is a hard project rule — the RW indicator must not rely on color to differ from read.
- Existing capability HMAC/entropy model is unchanged (40-bit code, HMAC-JWT cap, regen-key panic button); this phase changes *who may present a write cap over Funnel and how the code is issued*, not the crypto.

## Acceptance Criteria

- [ ] Completing the ≥3s hold issues exactly one single-use write code + write URL; releasing early issues nothing and grants no write
- [ ] With Funnel on but RW not gated, a write cap over the Funnel origin is rejected (401/403); the read path still works
- [ ] After the gate is passed, the gated write cap is accepted over the Funnel origin
- [ ] Guest A redeems the write code → write granted; guest B redeems the same code → fails closed ("code used")
- [ ] A public write guest can send terminal input but a `files.write` over the Funnel origin is rejected (browse on or off)
- [ ] The RW enable UI offers only ≤ 1h expiry options (default 15m, no "until I disable")
- [ ] An RW share request with `ExpiresIn == 0` or `> 3600s` results in an effective expiry of exactly 1h
- [ ] After the owner disables RW, the active writer's next request returns 401; read spectators keep viewing via the reusable read code
- [ ] The RW indicator differs from the read indicator in label AND icon AND shape (source-verifiable, not color-only)
- [ ] A Phase-171 SECURITY.md exists asserting no public write path except the gate, with no open write-path threat

## Edge Coverage

**Coverage:** 5/5 applicable edges resolved · 0 unresolved (1 dismissed as no-contract)

| Category | Requirement | Status | Resolution / Reason |
|----------|-------------|--------|---------------------|
| concurrency | R2 | ✅ covered | Two guests redeem the single-use write code simultaneously → exactly one wins (atomic delete-on-exchange), the other fails closed → AC "Guest A → granted; Guest B → code used" |
| authorization-boundary | R4 | ✅ covered | A write cap minted outside/before the RW gate, presented over the Funnel origin → rejected → AC "RW not gated → write over Funnel rejected" |
| boundary | R5 | ✅ covered | `ExpiresIn == 0` or `> 1h` requested for an RW share → clamped to the 1h hard max, never unbounded → AC "ExpiresIn 0 or >3600s → effective 1h" (guards the API layer, not just the dropdown) |
| gate-completeness | R1 | ✅ covered | Hold released before 3s → gate does not complete; no write URL/code issued → AC "releasing early issues nothing" |
| concurrency | R6 | ✅ covered | Owner disables RW while a keystroke is in flight → no NEW write authorized once disable returns; writer's next request 401s → AC "disable → next request 401" |
| precision | R5 | ⛔ dismissed | Rounding/tie-breaking on the expiry duration — a coarse whole-minute/second duration has no fractional or half-up/half-even contract; the boundary clamp (R5 covered) fully specifies the min/max behavior. |

## Prohibitions (must-NOT)

**Coverage:** 4/4 applicable prohibitions resolved · 0 unresolved

| Prohibition (must-NOT statement) | Requirement | Status | Verification / Reason |
|----------------------------------|-------------|--------|------------------------|
| A public READ guest MUST NOT gain write without redeeming the single-use write code through the gate (no read→write privilege escalation from a read cap over Funnel) | R4 | resolved | verification: test — a read cap over the Funnel origin cannot perform terminal input |
| RW teardown MUST NOT revoke or break the reusable public READ code — disabling RW / funnel-off / expiry revokes only the write cap | R6 | resolved | verification: test — teardown test asserts the read code still resolves and spectators keep viewing after RW disable |
| A redeemed write cap MUST NOT remain valid after its share expires or is disabled (no orphaned write cap surviving teardown) | R6 | resolved | verification: test — write cap is rejected after teardown/expiry at the chokepoint |
| The consent-gate copy MUST NOT understate the grant — it must state "command execution" and "anyone with the link" and must not use softened/euphemistic wording | R1 | resolved | verification: judgment — copy review of the gate warning text |

## Ambiguity Report

| Dimension          | Score | Min  | Status | Notes                                             |
|--------------------|-------|------|--------|---------------------------------------------------|
| Goal Clarity       | 0.88  | 0.75 | ✓      | Gated public RW, terminal-only, accidental path closed |
| Boundary Clarity   | 0.85  | 0.70 | ✓      | Four out-of-scope items explicitly excluded       |
| Constraint Clarity | 0.82  | 0.65 | ✓      | 15m/1h expiry, single-use, immediate revocation   |
| Acceptance Criteria| 0.82  | 0.70 | ✓      | 10 pass/fail criteria + 5 edges + 4 prohibitions  |
| **Ambiguity**      | 0.15  | ≤0.20| ✓      |                                                   |

## Interview Log

| Round | Perspective     | Question summary                          | Decision locked                                                        |
|-------|-----------------|-------------------------------------------|------------------------------------------------------------------------|
| 1     | Researcher      | Form of the hard consent gate?            | Hold-to-confirm (≥3s), distinct from read enable                       |
| 1     | Researcher      | What does "full access" write grant?      | Terminal command execution only — no public `files.write`              |
| 1     | Researcher      | How to close the accidental write path?   | Hard block: write over Funnel rejected unless RW-gated (fail-closed)   |
| 2     | Simplifier      | RW auto-expiry?                           | 15m default / 1h hard max; no "until I disable"                        |
| 2     | Simplifier      | Writer model / write-code semantics?      | Single-use code, one writer; 2nd redemption fails closed               |
| 2     | Simplifier      | RW vs the public read share?              | Coexist — read-many (reusable code) + write-one (single-use code)      |
| 3     | Boundary Keeper | What's explicitly OUT of scope?           | public files.write, guest-initiated write, CLI gate, multi-writer      |
| 3     | Boundary Keeper | Revocation speed on disable?              | Immediate cap revocation at the teardown chokepoint; next request 401  |
| 3     | Boundary Keeper | What makes RW unmistakable vs read?       | Label + icon + shape (colorblind-safe), never color alone              |
| 4     | Failure Analyst | Edge probe (5 covered, 1 dismissed)       | Concurrent redeem, non-gated cap rejected, expiry clamp, early release, disable-mid-write |
| 4     | Failure Analyst | Prohibition probe (4 must-NOTs)           | No read→write escalation; teardown keeps read; no orphaned write cap; consent copy not softened |

---

*Phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard*
*Spec created: 2026-07-07*
*Next step: /gsd-discuss-phase 171 — implementation decisions (how to build what's specified above), then /gsd-secure-phase 171 for the threat model*
