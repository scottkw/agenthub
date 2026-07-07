# Phase 171: Public Full-Access (Read-Write) Sharing - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-07
**Phase:** 171-public-full-access-read-write-internet-sharing-behind-a-hard
**Areas discussed:** Enforcement & revocation, Consent gate UX, RW indicator design, Expiry & post-gate UX

Requirements were locked by 171-SPEC.md (8 reqs) — this discussion covered implementation (HOW) decisions only.

---

## Enforcement & Revocation

### Q1 — Enforcement mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Gate the write grant | Register write grant only on hold-completion; remove at `disableFunnelForSession`; non-gated cap fails grant validation → 401; instant revoke via de-registration | ✓ |
| Per-session RW flag | Keep grant as-is; add a bool consulted by `originAllowedForWrite` | |
| Distinct cap claim/scope | New claim on the cap; still needs grant/flag for revocation | |
| Let research decide | Capture the guarantee, defer mechanism | |

**User's choice:** Gate the write grant.

### Q2 — Should `originAllowedForWrite` also be gate-aware?

| Option | Description | Selected |
|--------|-------------|----------|
| Defense-in-depth: both | Grant-gating (primary) AND `originAllowedForWrite` rejects funnel-origin write for non-gated sessions | ✓ |
| Grant-gating only | Leave `originAllowedForWrite` as a pure CSRF check | |
| Let research decide | Defer the second-barrier question | |

**User's choice:** Defense-in-depth — both barriers.

**Notes:** Captured without asking (implied by R4 + boundaries): remove the accidental `writeURL`/`writeCode` funnel-rebasing at `api.go:1464-72`; tailnet Full Access Link stays on the tailnet base; the public funnel write cap is minted only by the gate handler. A single per-session "RW-gated" state backs both barriers. Revocation chokepoint and no-crypto-change were pre-locked by SPEC constraints.

---

## Consent Gate UX

### Q1 — Placement of the hold control

| Option | Description | Selected |
|--------|-------------|----------|
| In Internet section, distinct block | Separated sub-block inside the existing Internet (public) section | |
| Separate 'Danger' section | Wholly separate section with its own heading, physically distant from read controls | ✓ |
| Let UI/research decide | Capture the constraint, defer placement | |

**User's choice:** Separate 'Danger' section.

### Q2 — Hold feedback (colorblind-safe)

| Option | Description | Selected |
|--------|-------------|----------|
| Radial fill + countdown text | Radial arc + shrinking numeric countdown | |
| Linear progress bar + label | Horizontal fill bar left→right + "Holding…" label; resets on early release | ✓ |
| Let UI/research decide | Defer exact motion | |

**User's choice:** Linear progress bar + label.

### Q3 — Consent warning copy

| Option | Description | Selected |
|--------|-------------|----------|
| Blunt / plain | "Public write = remote command execution…" | |
| Risk-forward | "⚠ You are exposing a terminal to the internet… full command execution as your account… a leaked link = remote code execution… Hold 3s to confirm" | ✓ |
| Concise | "Anyone with this link can execute commands on your computer…" | |

**User's choice:** Risk-forward (all options satisfied the SPEC prohibition: state "command execution" + "anyone with the link", no softening).

---

## RW Indicator Design

Read indicator today (Phase 166): rounded green pill, GlobeAltIcon + "INTERNET" text, colorblind-safe (icon+text carry state).

### Q1 — RW label text

| Option | Description | Selected |
|--------|-------------|----------|
| FULL ACCESS | Consistent with existing "Full Access Link" vocabulary; heavier than "INTERNET" | ✓ |
| READ-WRITE | Explicit capability contrast; longer on a pill | |
| PUBLIC WRITE | Emphasizes write+public; introduces a third vocabulary term | |

**User's choice:** FULL ACCESS.

### Q2 — RW icon + shape

| Option | Description | Selected |
|--------|-------------|----------|
| CommandLine + square/double border | Terminal glyph + sharp/heavier border | |
| LockOpen + notched/warning shape | LockOpenIcon (exposed/unlocked) inside a notched/angled-corner warning shape | ✓ |
| Let UI/research decide | Defer exact glyph | |

**User's choice:** LockOpen + notched/warning shape. Differs from the read pill in label AND icon AND shape (R7), colorblind-safe.

**Notes:** RW badge renders wherever the read INTERNET badge appears (SessionCard, TabBar). User is colorblind — verify distinctiveness at source/DOM/grayscale, not by eye.

---

## Expiry & Post-Gate UX

### Q1 — Expiry options

| Option | Description | Selected |
|--------|-------------|----------|
| 5m / 15m / 30m / 1h | Four choices incl. ultra-short 5m, default 15m | |
| 15m / 30m / 1h | Three choices, default 15m | ✓ |
| Let UI/research decide | Defer exact set | |

**User's choice:** 15m / 30m / 1h (default 15m). API clamps 0 or >1h to 1h (SPEC-locked).

### Q2 — Write-code redemption window

| Option | Description | Selected |
|--------|-------------|----------|
| = share expiry | Code redeemable for the whole share lifetime (≤1h), single-use | ✓ |
| Fixed 5-min window | Reuse existing joinCodes 5-min window independent of share lifetime | |
| Let research decide | Defer to joinCodes mechanics | |

**User's choice:** = share expiry (one coherent timer, still single-use).

### Q3 — Owner post-gate display

| Option | Description | Selected |
|--------|-------------|----------|
| URL+code+countdown, then 'used' → re-gate | Show URL+code+countdown+disable; after redeem → "Write code used — one writer connected"; re-share requires re-gate; expiry/disable collapses to "Enable public write…" | ✓ |
| URL+code+countdown only (no used-state) | No special post-consumption state | |
| Let UI/research decide | Defer consumed/expired states | |

**User's choice:** URL+code+countdown, then 'used' → re-gate.

---

## Claude's Discretion

- Exact DOM structure of the Danger section, progress-bar motion timing, and notched-shape CSS geometry (intent captured; constraints D-06/D-07/D-09 hold).
- Whether the per-session RW-gated state lives on the session struct vs a dedicated map (researcher to pick against live api.go/webserver structures).

## Deferred Ideas

- Guest-side write UI / rejection UX polish — noted as a possible "explore more" area, not deep-dived; standard fail-closed 401/"code used" is sufficient. Revisit if UAT surfaces confusion.
- Owner re-issue of a fresh write code without re-gating — SPEC out-of-scope.
- Public `files.write`, guest-initiated write promotion, CLI gate, multi-writer — SPEC out-of-scope.
