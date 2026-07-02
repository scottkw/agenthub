# Phase 169: Tailscale Detection Fix - Context

**Gathered:** 2026-07-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix Tailscale connection detection on non-admin macOS accounts (Issue #120). When the
SDK read `local.Client.StatusWithoutPeers(ctx)` fails — because the `macsys`
`sameuserproof` file is root:admin 0640 and unreadable by non-admin users — `checkHealth`
in `internal/webserver/tailscale.go` currently returns early with `DaemonUp=false`, so the
UI reports "installed but not Connected." This phase adds a CLI `tailscale status` fallback
that engages **only when the SDK read fails**, reconstructing the health struct from CLI
output so connection state reports correctly.

**Not in scope:** changing the SDK-success path, new UI surfaces/error copy, or reworking
the Funnel-path `StatusWithoutPeers` call in `server.go` (only `CheckHealth` gets the
fallback in this phase).

</domain>

<decisions>
## Implementation Decisions

### Fallback data scope
- **D-01:** On fallback, parse `tailscale status --json` and populate the **full**
  `TailscaleHealth` struct — `Connected` (from `BackendState == "Running"`), `IP` (first
  `TailscaleIPs`), `HasCerts` (`len(CertDomains) > 0`), `Domain` (first `CertDomains`). Goal
  is a functional account (web-share / Funnel usable), not just a cosmetic green status.
- **D-02:** `AcceptDNS` stays at its safe-false zero value in the fallback path. No CLI
  prefs read (`tailscale debug prefs`) is added — accept-dns resilience on non-admin
  accounts is out of scope for this fix (worst case: existing DNS-03 "assume disabled"
  warning behavior, unchanged).

### Fallback trigger
- **D-03:** Fire the CLI fallback on **any** error returned by `StatusWithoutPeers` — no
  error-string classification. Robust against the sameuserproof case plus any future
  socket/permission quirk. The extra latency when tailscaled is genuinely down is bounded by
  the existing ctx deadline (D-07).

### Platform gating
- **D-04:** Run the fallback on **all platforms** — no `runtime.GOOS` gate. The CLI path
  (`tailscale status --json`) is generic and the binary is already detected cross-platform;
  this adds resilience on Windows/Linux at no extra code cost.

### CLI-also-fails behavior
- **D-05:** If the CLI fallback also fails or times out, **fall through to today's
  not-connected state** (`BinaryFound=true, DaemonUp=false, Connected=false`). No new error
  field or UI copy. The fix is purely additive — it can only improve on the status quo and
  preserves Success Criterion 2 (SDK-success path behavior unchanged).

### Spawn mechanics
- **D-06:** Reuse the binary path already resolved by `detectTailscaleBinary(customPath)`
  for the CLI invocation, so a user's custom Tailscale path is honored by the fallback too.
- **D-07:** Bound the CLI spawn by the same `ctx` deadline `CheckHealth` already carries
  (the 3–5s UI timeout) — do **not** introduce a separate timeout knob.

### Claude's Discretion
- Exact JSON struct shape used to unmarshal `tailscale status --json` (a minimal local
  struct vs. reusing a tailscale.com type) — planner/researcher's call.
- Whether the fallback is a new helper (e.g. `cliStatusFallback`) injected like `statusFunc`
  for testability, or inlined. Injection-for-tests is the established idiom in this file
  (`statusFunc`, `prefsFunc`) — lean that way.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirement
- `.planning/REQUIREMENTS.md` — FIX-05 (line 54): the locked requirement text.
- `.planning/ROADMAP.md` §"Phase 169" (lines 532–543): Goal + 2 Success Criteria.

### Implementation target
- `internal/webserver/tailscale.go` — the file being modified. `checkHealth` (line 40) is
  the 4-state cascade; the early-return at lines 51–54 is the bug site. `statusFunc` /
  `prefsFunc` injection idiom (lines 28–34) is the test seam to mirror.
- `internal/webserver/tailscale.go` `detectTailscaleBinary` (called at line 44) — reuse for
  D-06 binary-path resolution.

### Related SDK call (NOT modified this phase — awareness only)
- `internal/webserver/server.go:620` — the Funnel path's own `StatusWithoutPeers` call. Same
  SDK read, same non-admin failure mode, but out of scope for Phase 169.

No external ADRs/design docs — requirements fully captured in decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `detectTailscaleBinary(customPath)` — already resolves custom path → well-known → PATH;
  the CLI fallback shells out to exactly this binary (D-06).
- `statusFunc` / `prefsFunc` injection pattern — established testability idiom; the CLI
  fallback should follow it so tests can fake CLI output without a live daemon.

### Established Patterns
- Silent degradation on probe failure (see the accept-dns swallow at lines 71–78 and
  RESEARCH Pitfall 4 referenced in-file) — D-05's fall-through matches this house style.
- `CheckHealth` / `CheckHealthWithCustomPath` are thin wrappers over `checkHealth`; the
  fallback logic belongs inside `checkHealth` so both entrypoints get it.

### Integration Points
- `checkHealth` return feeds `GetTailscaleStatus()` → Wails frontend, and
  `internal/daemon/process.go` (lines 40, 124) gates web-server restart on
  `Connected && HasCerts && IP != ""` — another reason D-01 populates the full struct, not
  just `Connected`.

</code_context>

<specifics>
## Specific Ideas

- Root cause per prior investigation (memory `reference_tailscale_connection_detection`):
  macsys `sameuserproof` is root:admin 0640, so non-admin accounts get an SDK read error and
  fall to the "installed but not connected" branch. This phase's CLI fallback is the
  documented remedy in Issue #120.

</specifics>

<deferred>
## Deferred Ideas

- Applying the same CLI fallback to the Funnel-path `StatusWithoutPeers` call in
  `server.go:620` — would make Funnel activation itself resilient on non-admin accounts, but
  is a separate change beyond FIX-05's detection-display scope. Note for a future phase if
  #120 follow-up surfaces.
- CLI prefs fallback for `AcceptDNS` (remote file-browse DNS on non-admin accounts) —
  deferred per D-02.

</deferred>

---

*Phase: 169-Tailscale Detection Fix*
*Context gathered: 2026-07-02*
