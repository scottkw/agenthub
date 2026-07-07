# Phase 171: Public Full-Access (Read-Write) Sharing - Research

**Researched:** 2026-07-07
**Domain:** Internet-exposed capability-based authorization (Go webserver + WebSocket relay) and colorblind-safe consent UX (React)
**Confidence:** HIGH (all claims below are `[VERIFIED: codebase]` via direct file reads — this phase adds no new external dependency, so there is no library-API uncertainty; the only `[ASSUMED]` items are naming choices explicitly left to Claude's discretion by CONTEXT.md)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Enforcement & Revocation (R4, R6)**
- D-01: Gate the write grant (primary enforcement). Stop registering the write grant at cap-issue time. Register the funnel write grant via `ws.AddGrant` only when the ≥3s hold-gate completes, and de-register it at the single `disableFunnelForSession` teardown chokepoint. A non-gated write cap then fails grant validation → 401; teardown de-registration gives instant revoke for free. Mirrors Phase 170's `funnelReadCode` lifecycle. No cap-crypto change.
- D-02: Defense-in-depth — `originAllowedForWrite` (`internal/webserver/capability_mw.go:191-210`) also becomes gate-aware. Rejects funnel-origin writes for a session that has not passed the RW gate. Two independent barriers.
- D-03: Single per-session "RW-gated" state is the source of truth for BOTH barriers. Set on hold-completion, cleared at the `disableFunnelForSession` chokepoint (covers disable / funnel-off / session-exit / expiry).
- D-04: Remove the accidental write-URL/code funnel-rebasing. `issueCapabilitiesForSession` (`internal/daemon/api.go:1451-1472`) currently rebases `writeURL` to the Funnel base on every call — remove it. The tailnet/local "Full Access Link" stays on the tailnet base (unchanged, out of scope). The public funnel write cap + single-use code are minted only by the gate handler, never at ordinary cap-issue time.
- D-05: Terminal-only scope at gate issuance (R3). The gate-minted public write cap grants terminal input (PTY write) only — must strip `files.write` even when the session's local browse is enabled.

**Consent Gate UX (R1)**
- D-06: Separate "Danger" section, physically below/separated from the Phase-166 "Internet (public)" read section.
- D-07: Linear progress bar + label hold feedback (≥3s), horizontal fill left→right, "Holding… keep pressing"; releasing before 3s resets to 0, issues nothing.
- D-08: "Risk-forward" consent copy baseline:
  - Heading: "⚠ You are exposing a terminal to the internet"
  - Body: "Anyone with the link and code gets full command execution on this machine, running as your account, until you disable it or it expires (max 1 hour). A leaked link = remote code execution."
  - Control: "Hold 3s to confirm"

**RW Indicator Design (R7, colorblind-safe — hard project rule)**
- D-09: Distinct in label + icon + shape (never color alone). Read indicator = rounded green pill, `GlobeAltIcon` + "INTERNET" text. RW indicator: Label "FULL ACCESS", Icon `LockOpenIcon`, Shape = notched/angled-corner "warning" shape, distinguishable in grayscale by shape + glyph alone.
- D-10: Same surfaces as the read badge — SessionCard `.hub-internet-badge`, TabBar `.tab__internet-icon`.

**Expiry & Post-Gate UX (R2, R5, R6)**
- D-11: Expiry options 15m / 30m / 1h, default 15m. No "until I disable". API-layer clamp: `ExpiresIn == 0` or `> 3600s` → effective exactly 1h.
- D-12: Write-code redemption window = share expiry (≤1h), single-use — first `/join/exchange` redemption grants the cap and deletes the code; second redemption fails closed ("code used").
- D-13: Owner post-gate display: URL + single-use code + live countdown + disable button; after redeem → "Write code used — one writer connected"; re-share requires re-holding the gate; on expiry/disable → revert to "Enable public write…" affordance.

### Claude's Discretion
- Exact placement/DOM structure of the Danger section within the Share modal, precise motion timing of the progress bar, and the notched-shape CSS geometry — captured intent above; downstream may refine as long as D-06/D-07/D-09 constraints hold.
- Whether the per-session RW-gated state lives on the existing session struct vs a dedicated map — researcher recommendation below (see Architecture Patterns → "Where the RW-gated state should live").

### Deferred Ideas (OUT OF SCOPE)
- Guest-side write UI / rejection UX polish (what the write guest sees, error states on "code used") — standard fail-closed 401/"code used" messaging is sufficient for this phase.
- Owner re-issue of a fresh write code without re-gating — explicitly out of scope; single-use, re-share requires re-hold.
- Public `files.write`, guest-initiated write promotion, CLI gate, multi-writer — all SPEC out-of-scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FNL-09 | Owner can opt into public read-write Funnel sharing behind a distinct hard consent gate, receiving a full-access public URL + single-use write code; public write reachable ONLY through this gate; shorter default/max expiry than read shares; covered by a threat model. | This entire document. Backend enforcement mechanics: "Architecture Patterns" + "Common Pitfalls". Single-use-with-custom-TTL code primitive: "Don't Hand-Roll". UI: "Code Examples" + CSS/icon guidance. |
</phase_requirements>

## Summary

This phase closes a real, already-live accidental-RCE path and replaces it with a deliberate one. The codebase investigation surfaced one finding more important than anything in CONTEXT.md's own framing: **`originAllowedForWrite` (D-02's proposed "defense-in-depth" barrier) does not currently gate the terminal-write path at all.** It is wired into exactly one call site — `requireFilesWrite`, which guards the five `files.write` HTTP routes (`PUT /api/files/write`, etc.). The WebSocket relay route (`GET /sessions/{id}/ws`, where `MsgInput`/`MsgSessionInject` — the actual PTY command-execution frames — are handled) is wrapped only by `requireAllowedOrigin` (a *different*, read/write-agnostic Origin check shared by every viewer) and `requireCapability` (grant + signature validation). Whether a connected client may write to the PTY is decided once, at WS-upgrade time, by `sub.ReadOnly = !capability.HasPerm(claims.Perms, "write")` — a property baked into the capability token's `Perms` string at mint time and never re-checked against Origin. **This means D-01 (grant-gating) is not merely the "primary" enforcement mechanism — for the terminal-write attack surface (the one the SPEC calls out as "internet RCE"), it is the *only* mechanism capable of closing the path.** D-02's origin-gate-awareness is real and worth building, but it only reaches the files.write HTTP routes, which are already out of scope for the gate-minted cap (D-05 strips `files.write` from that cap entirely). Planning must not treat D-02 as covering MsgInput — it does not, today, and extending it there is not called for by the SPEC (D-05 already prevents the gate cap from ever carrying `files.write`).

Three further code-level gaps must be designed around, not glossed over: (1) there is no per-grant removal primitive — `ClearGrants(sessionID)` wipes *every* grant for a session (used today only on full toggle-off / browse-toggle / session-exit), so naively reusing it inside the RW teardown path would also invalidate the tailnet-mode "Full Access Link" and the ordinary read link, an unacceptable regression; a new `RemoveGrant(sessionID, grantID)` must be added. (2) `JoinCodeManager.Issue()` hardcodes the manager's single fixed TTL (5 minutes, set once in `NewJoinCodeManager(5*time.Minute)`) — it cannot mint a single-use code redeemable for up to an hour (D-12); a new `IssueSingleUseWithTTL(token, ttl)` method is needed, mirroring the existing `IssueReusable` but preserving single-use atomic-delete-on-exchange semantics (needed for the SPEC's concurrent-redeem guarantee). (3) `originAllowedForWrite` lives in package `webserver`, while the natural home for Funnel/session bookkeeping (`funnelSessions`, `funnelReadCode`, etc.) is the `daemon.API` struct in a different package — the RW-gated boolean that both D-01 and D-02 need to consult should be colocated with the existing `ws.grants` map inside `webserver.WebServer` (new `SetRWGate`/`isRWGated` methods, mirroring `AddGrant`/`isGrantActive`), not bolted onto the daemon layer with a new cross-package provider callback.

**Primary recommendation:** Build a wholly separate mint/grant/code/expiry lifecycle for the gate-issued write capability — do not extend `issueCapabilitiesForSession`'s existing `wTok`. Wire enforcement through (a) a new `ws.SetRWGate(sessionID, bool)` + `isRWGated` pair consulted by both a modified `requireCapability`-adjacent grant check (D-01, via a distinct grant ID only registered post-hold) and `originAllowedForWrite` (D-02), and (b) a new `disableFunnelWriteForSession` helper that both a dedicated "disable RW" action and the existing `disableFunnelForSession` chokepoint call, so RW teardown is total-Funnel-teardown-safe without RW-only-disable prematurely killing the read share.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Hold-to-confirm consent gate UI | Frontend Server (SSR-free desktop Wails React) | — | Desktop-GUI-only per SPEC boundary; no CLI/headless surface |
| Gate-completion → mint write cap + code | API / Backend (daemon-local mux, loopback-trust) | — | Mirrors `handleSetSessionFunnel`; same trust boundary as existing loopback-only Funnel toggle endpoints (no auth needed — owner's own GUI is the only caller) |
| Write-cap grant registration/removal | API / Backend → Webserver | — | `ws.AddGrant`/new `ws.RemoveGrant` already the authz gate for caps; daemon orchestrates, webserver enforces |
| Terminal-write authorization (MsgInput/MsgSessionInject) | Webserver (WS relay, in-process) | — | Enforced by `sub.ReadOnly` derived from grant validity + `HasPerm(claims.Perms,"write")` at WS-upgrade time — this is the ONLY enforcement point reachable by a public writer |
| Origin-level defense-in-depth for files.write | Webserver (`requireFilesWrite`/`originAllowedForWrite`) | — | Existing CSRF-style Origin check; extend for gate-awareness, but note it does not reach terminal writes |
| Single-use write-code issuance/redemption | API / Backend (`capability.JoinCodeManager`) | — | New `IssueSingleUseWithTTL` method; redemption reuses existing generic `/join/exchange` unchanged |
| RW expiry timer + teardown | API / Backend (`daemon.API`) | Webserver (`ws.SetRWGate` clear) | Mirrors `funnelExpiry`/`disableFunnelForSession` pattern exactly |
| Colorblind-safe RW badge rendering | Frontend Server (React components) | — | SessionCard / TabBar, same surfaces as the existing INTERNET badge |
| Session-list RW-active flag | API / Backend (`SessionInfo.FunnelWriteActive`) | Frontend (poll consumer) | Mirrors `SessionInfo.FunnelActive` exactly (no `omitempty` — false must serialize) |

## Standard Stack

This phase introduces **no new external dependency** — every primitive it needs (HMAC-signed capability tokens, the grant registry, the join-code manager, `@heroicons/react` icon set, the existing risk-panel/badge CSS patterns) already exists in the codebase from Phases 87/118/124/137/165/166/170. All work is additive Go + TypeScript inside existing packages.

### Core (existing, reused — no install)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `internal/capability` (in-repo) | n/a | HMAC-SHA256 capability tokens, `JoinCodeManager` | Already the sole authz primitive for web-share; SPEC explicitly forbids a crypto change |
| `@heroicons/react` | `^2.2.0` `[VERIFIED: frontend/package.json]` | `LockOpenIcon` (D-09), already used for `GlobeAltIcon`/`LinkIcon` elsewhere | `LockOpenIcon` confirmed present at `frontend/node_modules/.pnpm/@heroicons+react@2.2.0.../24/outline/LockOpenIcon.js` `[VERIFIED: filesystem]` |
| `nhooyr.io/websocket` (existing relay transport) | unchanged | WS relay carrying MsgInput/MsgSessionInject | No change needed — enforcement is at the Go application layer (`sub.ReadOnly`), not the transport |

### Supporting
None — no new supporting libraries required.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| A new capability claim bit (e.g. `PermFunnelGated`) baked into the JWT-like token at sign time | Grant-ID gating (chosen, D-01) | SPEC explicitly forbids a cap-crypto change; a claim-based approach also can't be "instantly revoked" without a token-side channel (caps are stateless once signed) — grant-registry removal is the codebase's established stateful-revocation primitive (mirrors `funnelReadCode`) |
| Reusing `issueCapabilitiesForSession`'s existing `wTok`/`writeURL` for the public RW path, just conditionally rebasing to Funnel | Wholly separate mint path (chosen) | Reusing the existing write cap would make the *tailnet* Full Access Link's grant ID the same one gating public RW — disabling public RW would then also revoke the tailnet Full Access Link (unacceptable regression); D-04 already mandates the tailnet link stays untouched |
| `JoinCodeManager.IssueReusable` + manual "delete after first redeem" from the daemon side | New `IssueSingleUseWithTTL` method (recommended) | `IssueReusable`'s `Exchange` does NOT delete atomically for reusable entries — two concurrent redeemers would both succeed before the daemon's own "already used" bookkeeping could react, violating the SPEC's concurrent-redeem edge case (R2: exactly one winner via atomic delete-on-exchange) |

**Installation:** none — no `npm install` / `go get` required for this phase.

**Version verification:** N/A — no new packages.

## Package Legitimacy Audit

**Not applicable.** This phase adds zero new external packages (Go modules or npm packages). All new code is additive to existing internal packages (`internal/capability`, `internal/webserver`, `internal/daemon`) and existing frontend components. No `go.mod`/`package.json` changes are expected.

**Packages removed due to [SLOP] verdict:** none — no packages evaluated.
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

```
 Desktop GUI (SessionShareModal)
   │
   │ 1. Owner holds "Hold 3s to confirm" (D-07 progress bar)
   ▼
 Wails RPC: App.SetSessionFunnelWrite(sessionID, expiresIn)  [NEW]
   │
   ▼
 daemon.DaemonClient.SetSessionFunnelWrite (doJSON)  [NEW, mirrors SetSessionFunnel]
   │  POST /sessions/{id}/funnel-write  (loopback socket, no auth — owner's GUI only)
   ▼
 daemon.API.handleSetSessionFunnelWrite  [NEW]
   │
   │  clamp ExpiresIn (0 or >3600 → 3600)                          (D-11, R5)
   │  mint wTok: Claims{SID, Perms:"read,write", GrantID: new}     (D-05: hardcoded, NEVER browse-aware)
   │  ws.AddGrant(sessionID, wTok.GrantID)                          (D-01)
   │  ws.SetRWGate(sessionID, true)                                 [NEW method] (D-02/D-03)
   │  code := a.joinCodes.IssueSingleUseWithTTL(wTok, ttl)          [NEW method] (D-12)
   │  start a.funnelWriteExpiry[sessionID] = time.AfterFunc(ttl, disableFunnelWriteForSession)
   ▼
 Response: { writeURL, writeCode, expiresAt }
   │
   ▼
 SessionSharePanel "Danger" section (D-06): shows URL+code+countdown+disable (D-13)

 ── Guest (public internet) ──────────────────────────────────────
 Guest opens https://<funnelhost>/join?code=<writeCode>
   │
   ▼
 handleJoinExchange  (UNCHANGED — code-class-agnostic; Exchange() atomically
   │                   deletes single-use entry, returns wTok — SPEC R2 concurrency
   │                   guarantee lives HERE, in the existing joincode.go mutex)
   ▼
 303 → /sessions/{id}?cap=<wTok>
   ▼
 GET /sessions/{id}/ws  →  requireAllowedOrigin → requireCapability → handleWSSRelay
   │        (requireAllowedOrigin: unchanged, same Origin allowlist as every viewer)
   │        (requireCapability: isGrantActive(sid, wTok.GrantID) — FALSE until gate ran (D-01)
   │                             → 403 "capability has been revoked" if not gated)
   ▼
 sub.ReadOnly = !HasPerm(wTok.Perms, "write")   →  false (writer)
   │
   ▼
 MsgInput / MsgSessionInject → hub.WriteInput / hub.HandleInject   (PTY command execution)

 ── Teardown (any of: owner disables RW / funnel-off / session-exit / expiry) ──
   disableFunnelWriteForSession(sessionID)  [NEW — called BOTH standalone (owner
     disables RW only) AND from inside the existing disableFunnelForSession
     chokepoint (full Funnel teardown cascades into RW teardown too)]
   │  ws.RemoveGrant(sessionID, wTok.GrantID)   [NEW method] — next request 401/403
   │  ws.SetRWGate(sessionID, false)
   │  a.joinCodes.Revoke(writeCode)
   │  stop + delete a.funnelWriteExpiry[sessionID]
```

### Recommended Project Structure

No new files are required for the Go side; all additions extend existing files:

```
internal/capability/
  joincode.go              # + IssueSingleUseWithTTL(token, ttl) method
internal/webserver/
  server.go                # + RemoveGrant, SetRWGate, isRWGated methods (near AddGrant/ClearGrants, ~line 300-330)
  capability_mw.go          # originAllowedForWrite: + isRWGated(claims.SID) check (D-02)
internal/daemon/
  api.go                    # + funnelWriteGrant/funnelWriteCode/funnelWriteExpiry maps on API struct (mirror funnelReadCode* block ~line 70-105)
                             # + handleSetSessionFunnelWrite handler
                             # + disableFunnelWriteForSession helper, called from BOTH the new handler's disable path AND disableFunnelForSession
                             # + mux.HandleFunc("POST /sessions/{id}/funnel-write", ...) registration (~line 216, beside /sessions/{id}/funnel)
  types.go                  # + SetSessionFunnelWriteRequest/Response structs; + SessionInfo.FunnelWriteActive bool (mirror FunnelActive, no omitempty)
  client.go                 # + DaemonClient.SetSessionFunnelWrite method (mirror SetSessionFunnel, ~line 402)
app.go                      # + App.SetSessionFunnelWrite bound method (mirror SetSessionFunnel, ~line 1057)
frontend/src/wailsjs/...    # + hand-authored TS binding in BOTH App.d.ts and models.ts (Phase 170 precedent: two files must sync)
frontend/src/components/
  SessionSharePanel.tsx     # + Danger section (D-06), hold-to-confirm control (D-07), post-gate display (D-13)
  Hub/SessionShareModal.tsx # + wiring: hold-completion → SetSessionFunnelWrite; state for writeURL/writeCode/expiresAt/used
  Hub/SessionCard.tsx       # + FULL ACCESS badge (D-09/D-10), gated on session.funnelWriteActive
  TabBar.tsx                # + FULL ACCESS tab icon (D-10)
  style.css                 # + .hub-fullaccess-badge* rules (notched shape via clip-path), distinct from .hub-internet-badge
```

### Pattern 1: Grant-scoped revocation, never `ClearGrants`

**What:** The write cap issued by the gate has its own 128-bit `GrantID`, registered via `ws.AddGrant(sessionID, grantID)` exactly like the existing read/write grants — but removed individually.
**When to use:** Any teardown of the RW gate (owner-disable, funnel-off cascade, session-exit, expiry).
**Why it matters:** `ClearGrants(sessionID)` is a blunt "delete every grant this session has" operation, currently called only at full web-share toggle-off / browse-toggle / session-exit. If RW teardown reused it, disabling public write would also silently kill the session's tailnet-mode Full Access Link and Read-Only Link (their grants share the same `ws.grants[sessionID]` map). This is a correctness-critical distinction the plan must get right from the first task.
**Example:**
```go
// internal/webserver/server.go — new method, sibling to AddGrant/ClearGrants (server.go:303-320)
// RemoveGrant removes a single grantID from sessionID's active grant set,
// leaving any other grants (e.g. the ordinary tailnet read/write grants)
// untouched. Symmetric counterpart to AddGrant; unlike ClearGrants this is
// surgical, not blanket. Idempotent: removing an absent grant is a no-op.
func (ws *WebServer) RemoveGrant(sessionID, grantID string) {
	ws.mu.Lock()
	if ws.grants[sessionID] != nil {
		delete(ws.grants[sessionID], grantID)
	}
	ws.mu.Unlock()
}
```
Source pattern: `internal/webserver/server.go:303-320` `[VERIFIED: codebase]`

### Pattern 2: RW-gate boolean lives in `webserver.WebServer`, not `daemon.API`

**What:** A new per-session boolean, `ws.rwGated map[string]bool` (guarded by the existing `ws.mu`), with `SetRWGate(sessionID string, active bool)` and `isRWGated(sessionID string) bool` methods — colocated with `ws.grants` in `server.go`.
**When to use:** Consulted by `originAllowedForWrite` (D-02) directly, in-package, no cross-package call needed.
**Why this resolves CONTEXT.md's open discretion point:** `originAllowedForWrite` lives in package `webserver`; the natural alternative (storing gate state on `daemon.API`) would require either a new provider-callback (the pattern the codebase already uses for chat/alias lookups specifically to avoid a webserver→daemon import cycle — see `chatHistoryProvider`, `aliasGet`/`aliasSet` in `server.go:149-174`) or an import-cycle risk. Since `daemon.API` already calls INTO `webserver.WebServer` for every other grant/Funnel operation (`ws.AddGrant`, `ws.ClearGrants`, `ws.EnableFunnel`, `ws.FunnelBaseURL()`), the established direction is daemon→webserver, not the reverse. Colocating the boolean in `webserver.WebServer` keeps `originAllowedForWrite` a same-package, lock-cheap check with no callback indirection, and the daemon-side gate handler simply calls `ws.SetRWGate(id, true)` alongside its existing `ws.AddGrant(...)` call — one more line in an already-established call pattern.
**Example:**
```go
// internal/webserver/server.go — sibling to grants map declaration
rwGated map[string]bool // Phase 171 / FNL-09: true once the owner's hold-to-confirm
                          // gate has completed for sessionID; false or absent = not
                          // gated. Guarded by ws.mu. Consulted by originAllowedForWrite
                          // (D-02) as the second, independent barrier alongside grant
                          // validity (D-01/D-03).

func (ws *WebServer) SetRWGate(sessionID string, active bool) {
	ws.mu.Lock()
	if ws.rwGated == nil {
		ws.rwGated = make(map[string]bool)
	}
	if active {
		ws.rwGated[sessionID] = true
	} else {
		delete(ws.rwGated, sessionID)
	}
	ws.mu.Unlock()
}

func (ws *WebServer) isRWGated(sessionID string) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.rwGated[sessionID]
}
```

### Pattern 3: `disableFunnelWriteForSession` as a shared sub-teardown, not a rename of `disableFunnelForSession`

**What:** A new, narrower helper that revokes ONLY the RW-gate state (grant, code, gate flag, expiry timer) — called from two places: (a) the owner's dedicated "disable RW" action (which must NOT tear down the read Funnel share), and (b) from inside the existing `disableFunnelForSession` (so a full Funnel-off / session-exit / read-share-expiry cascades into RW teardown too, per D-03's "all four teardown triggers").
**When to use:** Any RW-specific teardown; never call the full `disableFunnelForSession` for an RW-only disable — that function also ref-counts down `funnelSessions` and can call `ws.DisableFunnel` (tearing down the read share and the entire Funnel serve config), which directly contradicts the SPEC prohibition: *"RW teardown MUST NOT revoke or break the reusable public READ code — disabling RW / funnel-off / expiry revokes only the write cap."*
**Why this matters:** CONTEXT.md's D-03 text ("cleared at the disableFunnelForSession chokepoint... covers all four teardown triggers") reads as if a single function handles all four, but the SPEC's own R6 acceptance criteria and prohibitions require "disabling RW" (owner turns off RW while keeping read alive) and "funnel-off" (owner turns off the whole share) to be genuinely different operations with different blast radii. Splitting them via a shared inner helper reconciles both: `disableFunnelForSession` (existing, full teardown) grows one new call to `a.revokeFunnelWriteLocked(sessionID)` (mirroring exactly how it already inlines `funnelReadCode` cleanup at api.go:1806-1811), and the new "disable RW" HTTP path calls the same helper directly without touching `funnelSessions`/`funnelReadCode`/ref-counting.
**Example:**
```go
// internal/daemon/api.go — helper, called under a.mu (mirrors funnelReadCode
// cleanup block already inlined in disableFunnelForSession, api.go:1806-1811)
func (a *API) revokeFunnelWriteLocked(sessionID string, ws *webserver.WebServer) {
	if grantID, ok := a.funnelWriteGrant[sessionID]; ok {
		if ws != nil {
			ws.RemoveGrant(sessionID, grantID)
		}
		delete(a.funnelWriteGrant, sessionID)
	}
	if code, ok := a.funnelWriteCode[sessionID]; ok {
		a.joinCodes.Revoke(code)
		delete(a.funnelWriteCode, sessionID)
	}
	if ws != nil {
		ws.SetRWGate(sessionID, false)
	}
	if t, ok := a.funnelWriteExpiry[sessionID]; ok {
		t.Stop()
		delete(a.funnelWriteExpiry, sessionID)
	}
}
```

### Pattern 4: Redemption reuses `/join/exchange` unmodified

**What:** The gate-minted write code is redeemed through the *existing* generic `handleJoinExchange` (`internal/webserver/server.go:1072-1125`), which is entirely agnostic to whether the underlying token is read or write — it just atomically looks up, conditionally deletes, and 303-redirects to `/sessions/{id}?cap=<token>`.
**When to use:** Do not build a parallel `/join/exchange-write` route. The single-use vs. reusable distinction (and TTL) is decided entirely at *mint* time (`Issue`/`IssueReusable`/new `IssueSingleUseWithTTL`), not at redemption time.
**Example:** `internal/webserver/server.go:1094` — `token, err := jc.Exchange(code)` is unchanged; `jc.Exchange` already does the right thing for both code classes (`internal/capability/joincode.go:162-178`).

### Anti-Patterns to Avoid
- **Extending `originAllowedForWrite` and believing it now covers terminal writes:** It does not reach `handleWSSRelay` at all (no call site there). If D-02 is implemented but D-01's grant-gating is skipped or buggy, public terminal RCE remains fully open — the SPEC's core requirement (R4, "no public write except through the gate") is NOT satisfied by D-02 alone.
- **Reusing `ClearGrants` for RW-only teardown:** wipes the session's ordinary read/write grants too (see Pattern 1).
- **Reusing `JoinCodeManager.Issue()` (5-min fixed TTL) for the write code:** produces a code that dies at 5 minutes regardless of the owner's chosen 15m/30m/1h share duration, contradicting D-12.
- **Deriving the gate-minted cap's `Perms` from `a.engine.browseEnabledFor(sessionID)`** (the pattern `issueCapabilitiesForSession` uses for its own write cap): would let a public RW guest touch `files.write` whenever the owner has local browse enabled — this is exactly what D-05 exists to prevent. The gate handler must hardcode `Perms: "read,write"`, full stop.
- **Calling `disableFunnelForSession` from the owner's "disable RW only" button:** cascades into tearing down the read Funnel share too (see Pattern 3).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Single-use write code with a custom (non-5-min) TTL | A parallel code-store, or bespoke expiry bookkeeping in the daemon layer | `JoinCodeManager.IssueSingleUseWithTTL(token, ttl)` — a ~15-line addition mirroring the existing `IssueReusable`, changing only `reusable: true` → the zero value (`false`) while keeping the custom-`ttl` expiry | `Exchange`'s atomic lookup+conditional-delete-under-one-mutex-hold already solves the concurrent-redeem race (SPEC edge R2) for single-use entries — do not reimplement that locking |
| Redemption HTTP flow for the write code | A new `/join/exchange-write` route or a write-specific `join.html` | The existing generic `handleJoinExchange` + `join.html` (Pattern 4) | Already code-class-agnostic; a parallel route would duplicate the 303/error-body contract (`?error=invalid/expired/session-gone`) for no benefit |
| Capability signing / verification for the gate-minted cap | A second signing key, a distinct token format, or an extra claim bit | `capability.Sign(Claims{...}, key)` with the SAME signing key already loaded (`a.signingKey`) | SPEC explicitly locks the crypto model; a second key or format is unnecessary complexity and a new attack surface |
| Colorblind-safe visual distinction | A second color palette, or relying on shade/tint differences | Label + icon + shape (D-09) exactly as the existing `.hub-internet-badge` pattern already does for read (icon+text carry state, color is reinforcement only, verified in CSS comments at `style.css:4706`) | The user is colorblind; the project's own established badge pattern already solved this correctly once — replicate its structure, not its color |
| Progress/hold-timer UI | A canvas/SVG animation library | Plain CSS `transition`/`width` percentage on a `<div>` driven by `pointerdown`/`pointerup` + `setInterval` or `requestAnimationFrame`, matching the existing `.hub-funnel-risk-panel` expand-transition pattern already guarded by `prefers-reduced-motion` in `style.css` | No new dependency; the codebase already has a CSS-driven expand/collapse precedent to extend |

**Key insight:** Every primitive this phase needs to enforce or communicate the RW gate already exists in the codebase in a slightly different shape (reusable read code, grant registry, risk panel, colorblind badge). The engineering work is almost entirely "add a sibling with different lifecycle parameters," not "invent a new mechanism." The one place that genuinely needs a new mechanism — a second write-path enforcement barrier reachable by the WS relay — does not exist today even in a different shape (see Common Pitfalls, Pitfall 1) and must be built from the grant-registry primitive, not the `originAllowedForWrite` primitive.

## Common Pitfalls

### Pitfall 1: Believing `originAllowedForWrite` already covers (or, once made gate-aware, will cover) terminal writes
**What goes wrong:** A plan or implementation treats D-02 as closing the RCE path, skips or under-tests D-01's grant-gating, and public terminal command execution remains reachable the whole time — the SPEC's headline requirement (R4) silently fails while CI stays green (D-02's own unit tests, like `TestOriginAllowedForWrite_FunnelOrigin`, only exercise `originAllowedForWrite` directly via `httptest.NewRequest`, never through `handleWSSRelay`).
**Why it happens:** The existing doc-comment on `originAllowedForWrite` (`capability_mw.go:187-189`) claims it checks "MsgInput, MsgSessionInject, MsgChatSend" — this comment is **stale/aspirational**, not accurate for the current code. `grep -rn "originAllowedForWrite"` shows exactly one call site, inside `requireFilesWrite`. `MsgInput`/`MsgSessionInject` are gated purely by `sub.ReadOnly`, computed once at WS-upgrade from `HasPerm(claims.Perms, "write")` (`server.go:1296-1297`), with no Origin re-check.
**How to avoid:** Treat D-01 (grant registration gating) as THE primary and sufficient mechanism for terminal-write enforcement. Write a dedicated integration test that opens `GET /sessions/{id}/ws` with a gate-minted write cap whose grant has NOT been registered (pre-hold state) and asserts the upgrade itself fails (403/401 from `requireCapability`'s `isGrantActive` check) — this is the actual proof of "no public write except through the gate," not an `originAllowedForWrite` unit test.
**Warning signs:** A plan task that only touches `capability_mw.go` and its existing unit test file, with no new/updated test in `internal/webserver/server_test.go` or a WS-relay integration test asserting `MsgInput` is rejected pre-gate.

### Pitfall 2: `ClearGrants` blast radius
**What goes wrong:** RW-disable also silently kills the tailnet Full Access Link and Read-Only Link for the same session (their grants live in the same `ws.grants[sessionID]` set).
**Why it happens:** `ClearGrants(sessionID)` is the only existing "remove a grant" primitive; it's tempting to reuse it since it's already imported/used elsewhere in `api.go`.
**How to avoid:** Add `RemoveGrant(sessionID, grantID)` (Pattern 1) and use it exclusively for RW-gate teardown.
**Warning signs:** A test that disables public RW and then asserts the tailnet write cap (issued separately via ordinary `IssueCapabilities`) still works — if this test doesn't exist, the regression is invisible until manual UAT.

### Pitfall 3: Confusing "disable RW" with "disable the whole Funnel share"
**What goes wrong:** Clicking the RW-section's disable button (D-13) also collapses the read Funnel share / kicks read spectators, violating the SPEC prohibition "RW teardown MUST NOT revoke or break the reusable public READ code."
**Why it happens:** The existing `disableFunnelForSession` is the only teardown chokepoint in the codebase today, and CONTEXT.md's D-03 wording could be read as "route everything through it."
**How to avoid:** Pattern 3 — a `disableFunnelWriteForSession`/`revokeFunnelWriteLocked` helper called from BOTH the narrow "disable RW" action and from inside the existing full-teardown chokepoint, never the reverse.
**Warning signs:** The "disable RW" button handler calls the existing `SetSessionFunnel(id, false, 0)` RPC (which maps to `handleSetSessionFunnel`'s disable path → full `disableFunnelForSession`) instead of a new, RW-scoped RPC.

### Pitfall 4: 5-minute join-code TTL silently truncating a 15m/30m/1h RW share
**What goes wrong:** The write code appears to work at share-creation but dies at 5 minutes into a 15m+ share, well before the owner's chosen expiry — guest gets a confusing "code used"/"invalid" error that looks like a bug in redemption rather than an issuance-side TTL mismatch.
**Why it happens:** `a.joinCodes = capability.NewJoinCodeManager(5*time.Minute)` (`api.go:304` — the manager's `ttl` field is fixed at construction) and `Issue(token)` always uses that fixed field, ignoring any longer intent.
**How to avoid:** New `IssueSingleUseWithTTL(token, ttl)` method (Don't Hand-Roll table) that accepts a per-call TTL, exactly like `IssueReusable` already does but with `reusable: false`.
**Warning signs:** A plan that calls `a.joinCodes.Issue(wTok)` for the gate-minted code instead of a new method — this is the single most likely silent-bug insertion point in this phase, because `Issue` is the obvious, already-imported, already-used-elsewhere function name.

### Pitfall 5: Gate-minted cap perms leaking `files.write` via copy-paste from `issueCapabilitiesForSession`
**What goes wrong:** The new gate handler is implemented by copying `issueCapabilitiesForSession`'s claim-construction block (`api.go:1428-1432`, the `if a.engine.browseEnabledFor(sessionID) { wPerms = "read,write,files.read,files.write" }` branch) and forgetting to strip it — a public RW guest gains remote file write whenever the owner happens to have local browse enabled, directly violating D-05/R3.
**Why it happens:** That block is the only existing template for "build a write-cap Claims struct" in the codebase; it's natural to copy it.
**How to avoid:** The gate handler must hardcode `Perms: "read,write"` unconditionally — never call `browseEnabledFor` at all in this code path.
**Warning signs:** No test asserting "public write cap perms == exactly 'read,write'" independent of the session's browse-toggle state (the existing `TestFunnelPublicCode_ReadOnlyScope` tests the READ code's browse-matrix; the equivalent WRITE-code test does not yet exist and must be added).

### Pitfall 6: Expiry clamp only enforced in the frontend dropdown
**What goes wrong:** A hand-crafted API request (or a future UI bug) with `ExpiresIn: 0` or `ExpiresIn: 999999` bypasses the 1h hard max, producing an unbounded or multi-day public-write share.
**Why it happens:** `handleSetSessionFunnel`'s existing `ExpiresIn` handling (`api.go:1750-1763`) treats `0` as a *legitimate* "unbounded" signal for the read share (capped only at the 8h code backstop) — copying that pattern for RW would violate R5, which requires `0` to become exactly `3600`, never "unbounded."
**How to avoid:** New handler clamps explicitly: `if req.ExpiresIn <= 0 || req.ExpiresIn > 3600 { req.ExpiresIn = 3600 }` — server-side, unconditionally, regardless of what the dropdown sent.
**Warning signs:** A plan task description that says "reuse handleSetSessionFunnel's ExpiresIn logic" without calling out this semantic difference (0 means different things for read vs. write shares).

## Code Examples

### Adding the single-use-with-custom-TTL join-code primitive
```go
// internal/capability/joincode.go — new method, sibling to IssueReusable (joincode.go:93-105)
// IssueSingleUseWithTTL generates a new 8-character base32 join code exactly
// like Issue (same crypto/rand + joinCodeEncoding path), but accepts a custom
// per-call ttl instead of the manager's fixed ttl field. The entry remains
// single-use (reusable: false, the zero value) — Exchange still deletes it
// atomically on first successful redemption, preserving the concurrent-
// redeem guarantee (exactly one winner). This is the write-code primitive
// for a share whose owner-chosen lifetime (15m/30m/1h) differs from the
// manager's fixed 5-minute default.
func (m *JoinCodeManager) IssueSingleUseWithTTL(token string, ttl time.Duration) (string, error) {
	var raw [5]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	encoded := joinCodeEncoding.EncodeToString(raw[:])
	code := encoded[:4] + "-" + encoded[4:8]

	m.mu.Lock()
	m.codes[code] = joinEntry{token: token, expiry: m.now().Add(ttl)} // reusable: false (zero value)
	m.mu.Unlock()
	return code, nil
}
```
Source pattern: `internal/capability/joincode.go:93-105` (`IssueReusable`) `[VERIFIED: codebase]`

### Gate-minted write capability (terminal-only, hardcoded perms — D-05)
```go
// internal/daemon/api.go — inside the new handleSetSessionFunnelWrite, NOT
// inside issueCapabilitiesForSession. Deliberately does NOT call
// a.engine.browseEnabledFor(sessionID) — Pitfall 5.
var wgid [16]byte
if _, err := rand.Read(wgid[:]); err != nil { /* ... */ }
wClaims := capability.Claims{
	SID:     sessionID,
	Perms:   "read,write", // D-05: hardcoded, terminal-only, regardless of browse toggle
	IAT:     time.Now().Unix(),
	GrantID: hex.EncodeToString(wgid[:]),
	V:       1,
}
wTok, err := capability.Sign(wClaims, key)
```
Source pattern: `internal/daemon/api.go:1418-1436` (`issueCapabilitiesForSession`'s claim construction, adapted) `[VERIFIED: codebase]`

### Existing colorblind-safe badge pattern to mirror for "FULL ACCESS" (D-09)
```tsx
// frontend/src/components/Hub/SessionCard.tsx:536-542 — existing read badge (mirror shape, new class/icon/label)
{session.funnelActive && (
  <span className="hub-internet-badge">
    <GlobeAltIcon className="hub-internet-badge__icon" aria-hidden="true" />
    <span className="hub-internet-badge__label">INTERNET</span>
  </span>
)}
// NEW, same surface, gated on a NEW session.funnelWriteActive flag (mirrors FunnelActive,
// SessionInfo.FunnelWriteActive json:"funnelWriteActive" — no omitempty, api.go:694 pattern):
{session.funnelWriteActive && (
  <span className="hub-fullaccess-badge">
    <LockOpenIcon className="hub-fullaccess-badge__icon" aria-hidden="true" />
    <span className="hub-fullaccess-badge__label">FULL ACCESS</span>
  </span>
)}
```
Source: `frontend/src/components/Hub/SessionCard.tsx:536-542` `[VERIFIED: codebase]`

### Notched/warning shape via `clip-path` (D-09's non-color, non-pill geometry)
```css
/* frontend/src/style.css — sibling to .hub-internet-badge (style.css:7181-7208) */
.hub-fullaccess-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--hub-fullaccess-badge-bg);
  padding: 3px 10px 3px 8px;
  /* Notched/angled-corner "warning" shape — distinct geometry from the read
     pill's uniform var(--hub-radius-pill) border-radius. Verifiable in
     grayscale/DOM inspection without color perception (D-09). */
  clip-path: polygon(6px 0, 100% 0, 100% 100%, 0 100%, 0 6px);
}
.hub-fullaccess-badge__icon {
  width: 12px;
  height: 12px;
  color: var(--hub-fullaccess-badge-text);
}
.hub-fullaccess-badge__label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--hub-fullaccess-badge-text);
}
```
Adapted from `frontend/src/style.css:7181-7208` `[VERIFIED: codebase]`; `clip-path` notch approach is a standard CSS technique, not project-specific — `[ASSUMED]` exact polygon values (Claude's Discretion per CONTEXT.md; verify the resulting shape reads as distinct from the pill at both normal and grayscale rendering, per the mandatory colorblind source-level verification rule).

## State of the Art

No external "state of the art" applies — this is 100% in-repo architecture extension, not a public-library integration. The relevant "current approach" baseline is the codebase's own Phase 165/166/170 precedent (grant registry + reusable join codes + colorblind badges), all of which this phase extends rather than replaces.

**Deprecated/outdated:**
- The unconditional `writeURL`/`writeCode` Funnel-rebasing inside `issueCapabilitiesForSession` (`api.go:1451-1472`) is the "old approach" this phase retires (D-04). It predates any RW-specific consent model and was never a deliberate design — it fell out of the Funnel base-URL rebase logic applying identically to both `readURL` and `writeURL`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Exact `clip-path` polygon values for the "notched/warning" badge shape | Code Examples | Low — CONTEXT.md explicitly delegates exact geometry to Claude's Discretion; any distinct-from-pill geometry satisfies D-09 as long as it's grayscale-verifiable |
| A2 | Recommended endpoint/method names (`SetSessionFunnelWrite`, `handleSetSessionFunnelWrite`, `disableFunnelWriteForSession`, `RemoveGrant`, `SetRWGate`, `IssueSingleUseWithTTL`) | Architecture Patterns, Code Examples | Low — these are naming proposals only; CONTEXT.md's "Claude's Discretion" section explicitly leaves the RW-gated state's exact home to the researcher/planner, and no test or downstream artifact depends on these specific identifiers. The *behavior* they implement (grant-scoped revocation, gate-aware origin check, custom-TTL single-use code) is the load-bearing finding, verified against live code |

**If this table is empty:** N/A — two low-risk naming/geometry assumptions logged above; all functional/architectural claims are `[VERIFIED: codebase]` against the live source tree.

## Open Questions

1. **Should the RW gate be offered while the read Funnel share is only "warming up" (not yet `funnelActive`), or only once fully live?**
   - What we know: `SessionSharePanel`'s existing "Internet (public)" section renders whenever `funnelEngaged` (`warmingUp || warmupTimedOut || funnelActive`) is true, but the public URL/code row only renders once `funnelActive && !warmingUp`.
   - What's unclear: CONTEXT.md doesn't specify whether the new "Danger" section (D-06) should be visible-but-disabled during warm-up, or hidden entirely until `funnelActive`.
   - Recommendation: Gate the Danger section's hold-control on `funnelActive` (not `funnelEngaged`) — the RW cap's URL is meaningless before the Funnel base URL exists, and offering a hold-to-confirm control that can't yet produce a working link risks a confusing "held for 3s, nothing happened" UX. Render the section (heading + risk copy) during warm-up but disable the hold control until `funnelActive`, matching how `FunnelRiskPanel`'s own CTA pattern already exists.

2. **Does `SessionInfo.FunnelWriteActive` need a poll-driven countdown, or is client-side `expiresAt` timestamp math sufficient for D-13's live countdown?**
   - What we know: The existing `FunnelActive` flag is polled (session list refresh) to detect the read share's expiry client-side; `SessionShareModal` already runs a "3s poll" per its own code comments.
   - What's unclear: Whether the RW countdown (D-13) should be driven by the same poll cadence (server confirms expiry) or a pure client-side `setInterval` against a locally-stored `expiresAt`.
   - Recommendation: Use a client-side `setInterval` against the `expiresAt` timestamp returned by the gate-mint RPC for the visual countdown (cheap, no extra round-trips), but treat `session.funnelWriteActive` flipping false (via the existing poll) as authoritative for collapsing the UI back to "Enable public write…" — mirrors the existing warm-up-timeout pattern (`warmupTimedOut` local timer + `funnelActive` poll-confirmed flip).

## Environment Availability

Skipped — this phase has no new external tool/service/runtime dependency. It extends existing Go (`internal/capability`, `internal/webserver`, `internal/daemon`) and TypeScript/React code already present and building in this repo; the existing Tailscale Funnel dependency (verified present and working as of Phase 165/166/170 live UAT, per STATE.md) is unchanged by this phase.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (backend) + Vitest (frontend), both already configured |
| Config file | none dedicated — Go via `go.mod`; Vitest via `frontend/vite.config.ts` / `frontend/vitest.config.ts` (existing, unchanged) |
| Quick run command | `go test -race -short ./internal/capability/... ./internal/webserver/... ./internal/daemon/...` (backend); `cd frontend && pnpm test -- SessionSharePanel SessionCard SessionShareModal` (frontend, scoped) |
| Full suite command | `go test -race -short ./...` (backend, 375 files); `cd frontend && pnpm test` (frontend, 142 files) — per `TESTING.md` Section 2 |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| R1 | Hold-to-confirm gate: early release issues no cap/code | unit | `pnpm test -- SessionShareModal` (new `describe` block) | ❌ Wave 0 — extend existing `SessionShareModal.test.tsx` |
| R2 | Concurrent redeem of write code → exactly one winner | unit | `go test -race ./internal/capability/... -run TestJoinCodeManager_IssueSingleUseWithTTL` | ❌ Wave 0 — new test in `joincode_test.go`, mirroring existing single-use concurrency coverage pattern |
| R3 | Public write cap perms are exactly `read,write`, never `files.write`, regardless of browse toggle | unit | `go test ./internal/daemon/... -run TestFunnelWriteGate_TerminalOnlyScope` | ❌ Wave 0 — new test in `internal/daemon/funnel_test.go`, mirroring `TestFunnelPublicCode_ReadOnlyScope` (`funnel_test.go:339`) |
| R4 | Write cap over Funnel origin rejected pre-gate; accepted post-gate | integration | `go test ./internal/webserver/... -run TestHandleWSSRelay_WriteCap_RequiresGate` | ❌ Wave 0 — **critical new test**, must exercise the actual `GET /sessions/{id}/ws` upgrade path (see Pitfall 1), not just `originAllowedForWrite` in isolation |
| R5 | `ExpiresIn` 0 or >3600 clamps to exactly 3600 | unit | `go test ./internal/daemon/... -run TestHandleSetSessionFunnelWrite_ExpiryClamp` | ❌ Wave 0 — new test, mirrors `TestFunnelAutoExpiry`/`TestHandleSetSessionFunnel_Enable` (`funnel_test.go:169,272`) |
| R6 | Disable RW → writer's next request 401; read spectators unaffected | integration | `go test ./internal/daemon/... -run TestDisableFunnelWrite_RevokesGrantOnly` | ❌ Wave 0 — new test, mirrors `TestFunnelTeardown_AllTriggers` (`funnel_test.go:516`) but must ALSO assert the read code/grant survive |
| R7 | RW indicator differs in label+icon+shape (source-verifiable) | unit | `pnpm test -- SessionCard` (new assertions) | ❌ Wave 0 — extend `SessionCard.test.tsx`, mirror existing FUI-03 tests (`SessionCard.test.tsx:637-666`) |
| R8 | Threat model asserts no public write path except the gate | manual/judgment | `/gsd-secure-phase 171` output review | N/A — produced by the secure-phase agent, not an automated test |

### Sampling Rate
- **Per task commit:** scoped `go test ./internal/capability/... ./internal/webserver/... ./internal/daemon/...` and `pnpm test -- <touched components>`
- **Per wave merge:** full suite green before proceeding (`go test -race -short ./...` + `cd frontend && pnpm test` + `tsc --noEmit`)
- **Phase gate:** Full suite green before `/gsd-verify-work`; live UAT (off-tailnet, per project standing pattern for Funnel phases) required before phase closeout given the RCE severity

### Wave 0 Gaps
- [ ] `internal/capability/joincode_test.go` — add tests for `IssueSingleUseWithTTL` (mint, custom-TTL expiry, atomic single-redeem-under-race)
- [ ] `internal/webserver/server_test.go` (or new `internal/webserver/rwgate_test.go`) — `RemoveGrant`, `SetRWGate`/`isRWGated`, and (critically, per Pitfall 1) an integration test through the real `GET /sessions/{id}/ws` upgrade path proving a non-gated write cap is rejected
- [ ] `internal/daemon/funnel_test.go` — `handleSetSessionFunnelWrite` (mint, clamp, terminal-only perms regardless of browse), `disableFunnelWriteForSession`/`revokeFunnelWriteLocked` (RW-only teardown leaves read intact; full teardown cascades into RW)
- [ ] `frontend/src/components/__tests__/SessionSharePanel.test.tsx` — Danger section render/hide, hold-progress reset-on-early-release, post-gate used-state
- [ ] `frontend/src/components/Hub/SessionCard.test.tsx` — FULL ACCESS badge presence/absence, distinct class/icon/label from `.hub-internet-badge` (source-level, mirrors existing FUI-03 tests)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | yes | HMAC-SHA256 capability tokens (`internal/capability`), unchanged crypto per SPEC constraint; write-code redemption is a bearer-secret exchange (`/join/exchange`), unchanged flow |
| V3 Session Management | yes | Grant-registry revocation (`ws.grants`) IS the session-invalidation mechanism for stateless caps — `RemoveGrant`/`SetRWGate` are the session-termination primitives for this phase (V3.3-style "logout invalidates session" analog) |
| V4 Access Control | yes | Whole-token `HasPerm` permission checks (never substring), grant-ID scoping to session, terminal-only perm string hardcoded at gate-mint (D-05) — this IS the phase's core ASVS surface |
| V5 Input Validation | yes | `ExpiresIn` server-side clamp (R5/D-11) — must not trust client-submitted expiry; `code` form value length/format already validated generically by `handleJoinExchange` (unchanged) |
| V6 Cryptography | no (unchanged) | Reuses existing HMAC-SHA256 signing key and `crypto/rand` join-code generation; SPEC explicitly locks this — no new crypto surface |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Privilege escalation via read→write cap confusion (a read cap presented where write is expected) | Elevation of Privilege | Whole-token `HasPerm(claims.Perms, "write")` check (existing, `server.go:1297`) — a read cap's `Perms="read"` (or `"read,files.read"`) never satisfies this; unchanged by this phase, but the gate-minted cap must be verified to follow the same invariant |
| Grant replay after revocation (writer keeps a valid-looking cap after owner disables RW) | Tampering / Repudiation | `isGrantActive` check inside `requireCapability`, consulted on every `?cap=` request including WS upgrade — `RemoveGrant` (new) makes this instant, matching the existing `funnelReadCode` revoke-on-teardown precedent |
| TOCTOU on concurrent code redemption (two guests race the same single-use code) | Tampering | `JoinCodeManager.Exchange`'s single mutex hold across lookup+expiry-check+delete (existing, `joincode.go:162-178`) — `IssueSingleUseWithTTL` must NOT alter this locking, only the TTL source |
| Origin spoofing / CSRF on the files.write HTTP surface | Spoofing / Tampering | `originAllowedForWrite` (existing, extended for gate-awareness per D-02) — genuinely defense-in-depth for the files.write routes; **does not** apply to the WS relay surface (Pitfall 1) |
| Stale/leaked write cap accepted after Funnel origin is disabled (e.g., a guest replays a captured request after the owner reverts to tailnet-only) | Spoofing | `originAllowedForWrite`'s existing `funnelURL != "" && origin == funnelURL` fail-closed check (unchanged structurally, `capability_mw.go:203-206`) plus grant removal on any teardown |
| Public write cap accidentally carrying `files.write` (permission over-grant) | Elevation of Privilege | Gate handler hardcodes `Perms: "read,write"` unconditionally (D-05) — never derives from `browseEnabledFor` (Pitfall 5) |
| Unbounded public-write exposure window via a malformed/omitted expiry | Denial of Service (of the "share must end" property, i.e. persistent exposure) | Server-side clamp of `ExpiresIn` to exactly 3600s when 0 or out-of-range (D-11/R5, Pitfall 6) |

## Sources

### Primary (HIGH confidence — direct codebase reads this session)
- `internal/daemon/api.go` (full `issueCapabilitiesForSession`, `handleSetSessionFunnel`, `disableFunnelForSession`, `handleIssueCapabilities`, `mintFunnelReadCodeLocked`, struct field docs) — read in full for the relevant ~1800 lines
- `internal/webserver/server.go` (`AddGrant`/`ClearGrants`/`isGrantActive`, `handleWSSRelay` including the `sub.ReadOnly` derivation and the full MsgInput/MsgSessionInject/MsgChatSend switch, route registration table, `SessionInfo`/`funnelSnap` population in `handleListSessions`)
- `internal/webserver/capability_mw.go` (full file: `requireCapability`, `requireFilesRead`, `requireFilesWrite`, `originAllowedForWrite`)
- `internal/webserver/origin_mw.go` (`requireAllowedOrigin` route-wiring confirmation)
- `internal/capability/joincode.go` (full file: `Issue`, `IssueReusable`, `Revoke`, `Rebind`, `Exchange`)
- `internal/capability/capability.go` (`Claims` struct, `PermFilesRead`/`PermFilesWrite` constants, `HasPerm`)
- `internal/relay/hub.go` / `internal/relay/protocol.go` (`HandleInject`, `ErrReadOnly`, `MsgInput` constant — confirming no Origin check anywhere in the inject/input path)
- `internal/daemon/types.go` (`SessionInfo` struct, `FunnelActive` field precedent)
- `internal/daemon/client.go` (`SetSessionFunnel` wire-call pattern)
- `app.go` (`SetSessionFunnel`/`IssueCapabilities` Wails bindings)
- `frontend/src/components/SessionSharePanel.tsx` (full file — Danger-section insertion point, existing Full Access Link / Internet section structure)
- `frontend/src/components/Hub/FunnelRiskPanel.tsx` (full file — risk-copy/expiry-selector pattern to mirror)
- `frontend/src/components/Hub/SessionShareModal.tsx` (grep of state hooks — `SetSessionFunnel`/`IssueCapabilities` wiring, `funnelOn`/`warmingUp`/`publicReadCode` state shape)
- `frontend/src/components/Hub/SessionCard.tsx` (existing `.hub-internet-badge` render + icon imports)
- `frontend/src/style.css` (`.hub-internet-badge*`, `.tab__internet-icon`, `.hub-funnel-risk-panel*` CSS blocks)
- `frontend/package.json` + filesystem check confirming `@heroicons/react@2.2.0` and `LockOpenIcon` availability
- `internal/webserver/funnel_test.go` / `internal/daemon/funnel_test.go` (existing test inventory — `TestOriginAllowedForWrite_FunnelOrigin`, `TestFunnelTeardown_AllTriggers`, `TestFunnelPublicCode_ReadOnlyScope`, etc. — confirms exact current coverage and gaps)
- `TESTING.md` (Suite Manifest, run commands, per-phase note convention)
- `.planning/phases/171-.../171-CONTEXT.md`, `171-SPEC.md`, `171-DISCUSSION-LOG.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md` (upstream phase artifacts)

### Secondary (MEDIUM confidence)
None — no external documentation was consulted; this phase's entire technical surface is in-repo.

### Tertiary (LOW confidence)
None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependency; every primitive verified present in the live tree
- Architecture: HIGH — every claim (grant registry scope, `originAllowedForWrite` call sites, `sub.ReadOnly` derivation, `JoinCodeManager` TTL semantics) verified by direct code read, not inference
- Pitfalls: HIGH — Pitfall 1 (the most consequential finding) is a direct `grep` + read confirmation, not speculation; Pitfalls 2-6 are each grounded in a specific cited line range

**Research date:** 2026-07-07
**Valid until:** No expiry driver — this is in-repo architecture research, not a fast-moving external API; valid as long as the cited line ranges are unmodified by intervening work. Re-verify line numbers if Phase 172 (Hub-card layout) or any other phase touches `SessionCard.tsx`/`style.css` before Phase 171 executes.
