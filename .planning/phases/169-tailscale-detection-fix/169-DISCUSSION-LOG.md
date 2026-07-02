# Phase 169: Tailscale Detection Fix - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-02
**Phase:** 169-Tailscale Detection Fix
**Areas discussed:** Fallback data scope, Fallback trigger, Platform gating, CLI-also-fails behavior

---

## Fallback data scope

| Option | Description | Selected |
|--------|-------------|----------|
| Full struct | Parse `tailscale status --json` → Connected, IP, HasCerts, Domain; AcceptDNS safe-false | ✓ |
| Connected-only (cosmetic) | Flip Connected from BackendState only; IP/certs/domain stay empty | |
| Full struct + prefs call | Full struct plus a CLI prefs read for AcceptDNS | |

**User's choice:** Full struct (Recommended).
**Notes:** Goal is a functional non-admin account (web-share/Funnel usable), not just a green
status. AcceptDNS left at safe-false — no CLI prefs read added (deferred).

---

## Fallback trigger

| Option | Description | Selected |
|--------|-------------|----------|
| Any SDK read error | Fall back on any `StatusWithoutPeers` error; simplest and most robust | ✓ |
| Permission-style errors only | Only fall back on access/permission-denied signatures | |

**User's choice:** Any SDK read error (Recommended).
**Notes:** Avoids brittle error-string matching. Extra latency on a genuinely-down daemon is
bounded by the existing ctx timeout.

---

## Platform gating

| Option | Description | Selected |
|--------|-------------|----------|
| All platforms | Run fallback anywhere the SDK read fails; generic CLI path | ✓ |
| macOS-only (runtime.GOOS) | Gate behind `runtime.GOOS == "darwin"` | |

**User's choice:** All platforms (Recommended).
**Notes:** Mechanism isn't macOS-specific even though #120's root cause is; free resilience
on Win/Linux.

---

## CLI-also-fails behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Current not-connected | Fall through to today's behavior (BinaryFound=true, DaemonUp=false, Connected=false) | ✓ |
| Distinct hint/error | Report a new state/message distinguishing SDK+CLI failure from daemon-down | |

**User's choice:** Current not-connected (Recommended).
**Notes:** Purely additive fix; no new error surface. Preserves Success Criterion 2.

---

## Claude's Discretion

- Exact JSON unmarshal struct shape for `tailscale status --json`.
- Whether the CLI fallback is a new injected helper (mirroring `statusFunc`/`prefsFunc`) or
  inlined — injection-for-tests idiom preferred.
- Spawn mechanics captured without a question: reuse `detectTailscaleBinary` path (honor
  custom path); bound by the existing `ctx` deadline (no new timeout knob).

## Deferred Ideas

- Apply the same CLI fallback to the Funnel-path `StatusWithoutPeers` call in `server.go:620`.
- CLI prefs fallback for `AcceptDNS` on non-admin accounts.
