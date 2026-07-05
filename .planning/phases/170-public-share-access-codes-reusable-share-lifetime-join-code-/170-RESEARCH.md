# Phase 170: Public Share Access Codes (read) - Research

**Researched:** 2026-07-05
**Domain:** Go capability/join-code lifecycle (internal/capability, internal/daemon, internal/webserver) + React Share-modal UI (frontend/src/components)
**Confidence:** HIGH (all claims below are grounded in real file:line reads of the current codebase; no external libraries or new dependencies are involved)

<user_constraints>
## User Constraints (from ROADMAP.md Phase 170 section — no separate CONTEXT.md exists)

### Locked Decisions (2026-07-05 discussion, carried in ROADMAP.md)

- Add **per-code TTL + reusable (non-single-use)** semantics to `JoinCodeManager` (today all codes are 5-min single-use — wrong for a public share meant to last the auto-expiry window and serve multiple viewers).
- The public code's lifetime is **tied to the funnel auto-expiry** (dies exactly when the share does).
- **Read-only scope only** — the reusable code resolves to the funnel *read* cap; it must never map to the write cap.
- Keep **40-bit crypto/rand** entropy (2⁴⁰ over an ≤8h window is not brute-forceable even without rate-limiting).
- **Supplement, not replace** the existing self-contained cap-URL + QR flow.

### Claude's Discretion

None explicitly marked — the ROADMAP section is fully prescriptive. Implementation details not covered by the bullets above (exact Go method names, exact field names, exact revocation mechanism) are open for the planner, informed by this research.

### Deferred Ideas (OUT OF SCOPE)

- Public **write** access (single-use write code, hard consent gate) — explicitly deferred to **Phase 171** (spec-first, FNL-09). Do not let Phase 170 touch the write cap or the write join code.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FNL-08 | A Funnel/public share surfaces a **reusable, share-lifetime join code** in the Share UI so a recipient who cannot scan the QR or paste the full capability URL can join **read-only** with a short code; the code is valid only for the funnel share's lifetime, resolves to the read capability only (never write), and supplements (does not replace) the self-contained cap-URL/QR. (Amends FNL-03's "single-use join code" for the public read case.) | See "JoinCodeManager today", "Funnel auto-expiry plumbing", "Read-cap resolution", "Share modal frontend" sections below — this research maps every clause of FNL-08 to a concrete code anchor and a concrete gap to close. |
</phase_requirements>

## Summary

Phase 170 closes a real UAT dead-end: the Share modal's "Internet (public)" section (`frontend/src/components/SessionSharePanel.tsx:313-382`) shows a public Funnel URL, Copy/Open/QR buttons, and a disable button — but **no join code**, unlike the Read-Only and Full-Access sections directly above it, each of which renders a `<CodeDisplay>` row (`SessionSharePanel.tsx:249`, `:300`). A recipient who cannot scan the QR or paste the long capability URL has nothing to type at `/join`.

The root cause is architectural, not cosmetic: `JoinCodeManager` (`internal/capability/joincode.go`) has exactly one TTL (5 minutes, set once at construction — `internal/daemon/api.go:259`) and `Exchange()` **always deletes** the entry on a successful lookup (`joincode.go:79-93`, the TOCTOU-safe lookup+delete). This is correct for the existing Read-Only/Full-Access codes (spoken/typed once, by one recipient, shortly after issuance) but wrong for a public share meant to live for up to 8 hours and be used by an unbounded number of anonymous viewers.

Closing the gap requires three coordinated changes, all narrowly scoped:

1. **`internal/capability/joincode.go`**: add a `reusable bool` field to `joinEntry`, a new `IssueReusable(token string, ttl time.Duration) (string, error)` method (same 40-bit crypto/rand code-gen as `Issue`, just a per-call TTL instead of the manager-wide one), and change `Exchange` to skip the `delete` when `entry.reusable` is true (still enforcing the TTL check). The existing `Issue`/`Exchange` contract for non-reusable codes is untouched — all 6 existing tests in `joincode_test.go` keep passing unmodified.
2. **`internal/daemon/api.go`**: `handleSetSessionFunnel` (line 1581) is the only place that knows the user's chosen `ExpiresIn`. On enable, mint a reusable code bound to a **read-only** capability token and cache it per-session (new `a.funnelReadCode map[string]string`, mirroring the existing `funnelSessions`/`funnelExpiry` map pattern at lines 71-80). `disableFunnelForSession` (line 1645) — the single function all 4 in-process teardown triggers already route through — must explicitly revoke that cached code (new `JoinCodeManager.Revoke(code string)` method) so the code dies immediately on manual disable, web-share-off, or session-exit, not just when its TTL backstop elapses.
3. **Frontend**: surface the new code in `SessionSharePanel.tsx`'s existing "Internet (public)" block (`:313-382`) with the same `<CodeDisplay>` component already used for the other two sections (`SessionSharePanel.tsx:8-57`) — no new UI primitive needed.

**Primary recommendation:** Extend `JoinCodeManager` with a `reusable`-flag + per-call-TTL `IssueReusable`/`Revoke` pair (leaving `Issue`/`Exchange` behavior for existing single-use codes untouched), mint the public read code once at Funnel-enable time in `handleSetSessionFunnel`, cache it so repeat `IssueCapabilities` calls don't rotate it, and revoke it explicitly inside the existing `disableFunnelForSession` chokepoint so it truly dies exactly when the share does — not merely when a timer coincidentally fires.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Reusable join-code generation + per-code TTL semantics | API/Backend | — | `JoinCodeManager` lives in `internal/capability`, called only from `internal/daemon/api.go` (`issueCapabilitiesForSession`, `handleSetSessionFunnel`) |
| Funnel-lifetime binding (code dies with the share) | API/Backend | — | `funnelExpiry` timer + `disableFunnelForSession` (`internal/daemon/api.go:1636-1662`) already own every teardown trigger; the public code's revocation must hook into this exact chokepoint |
| Read-cap-only scope enforcement (never the write cap) | API/Backend | — | `issueCapabilitiesForSession` (`api.go:1336-1430`) already mints separate `rTok`/`wTok`; the reusable code must be bound to `rTok` only, at the point of minting |
| Public code display in Share modal | Browser/Client | API/Backend (data source) | `SessionSharePanel.tsx` renders; the daemon supplies the code value via the Wails-bound `IssueCapabilitiesResponse` struct, which is a straight pass-through (`app.go:1138-1143` → `internal/daemon/client.go` → HTTP) |
| Code-entry page (recipient side, `/join`) | API/Backend | Browser/Client | `join.html` is served by `internal/webserver/server.go:handleJoin` (embedded static page); `handleJoinExchange` (`server.go:1072-1125`) is the server logic a browser's form POST hits — it already calls the shared `JoinCodeManager.Exchange`, so fixing `Exchange`'s reusable-skip-delete logic fixes this page for free (no code-entry-page changes needed) |

## Standard Stack

No new external packages are required. This phase extends existing Go stdlib usage only:

| Package | Purpose | Why Standard |
|---------|---------|--------------|
| `crypto/rand` | 40-bit code entropy (already used, `joincode.go:4,56`) | Same generator reused for the new `IssueReusable` — do not introduce a second RNG source |
| `encoding/base32` | Code alphabet (already used, `joincode.go:5,13`) | RFC 4648 standard alphabet, no 0/O/1/I/l ambiguity (D-10) — reuse unchanged |
| `sync` | Mutex-guarded map (already used, `joincode.go:6,34`) | Same TOCTOU-safe pattern extends to the new reusable path |
| `time` | TTL/expiry (already used) | `time.Time`/`time.Duration` — no new abstraction needed |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| In-process map + explicit revocation | A separate reusable-code TTL sweep goroutine | Rejected: the codebase's existing comment (`joincode.go:24-26`) explicitly chose lazy expiry-on-Exchange over a background sweeper to keep the design simple; a public reusable code doesn't change that tradeoff — explicit revocation at the 4 known teardown call sites is simpler and more correct than a timer race |
| Binding the reusable code's TTL to a fixed max (e.g. 8h) always | Binding it to the user's actual `ExpiresIn` selection, with a fallback for the "Until I disable" (0) case | See Open Questions — the "0 = no auto-expiry" sentinel has no natural TTL; recommend explicit revocation as primary mechanism + a generous backstop TTL, not a hard product decision this research can make unilaterally |

**Installation:** None — no `go get` / `npm install` required for this phase.

**Version verification:** Not applicable (stdlib only, no third-party packages).

## Package Legitimacy Audit

**Not applicable.** This phase introduces zero new external packages (Go stdlib: `crypto/rand`, `encoding/base32`, `sync`, `time` — all already imported by the file being extended). No `npm install` / `go get` / `pip install` occurs. Skip the Package Legitimacy Gate for this phase's planning; the planner should not add a package-legitimacy checkpoint since none is needed.

## Architecture Patterns

### System Architecture Diagram

```
 Recipient's browser                    AgentHub daemon (Go)                    Owner's Wails app (React)
 ───────────────────                    ─────────────────────                   ──────────────────────────

 1. Owner enables Funnel
    in Share modal ──────────────────────────────────────────────────────────▶  SessionShareModal.tsx
                                                                                 handleFunnelEnable()
                                                                                    │ SetSessionFunnel(id, true, expiresIn)
                                                                                    ▼
                                        POST /sessions/{id}/funnel ◀───────────────┘
                                        handleSetSessionFunnel (api.go:1581)
                                           │ ws.EnableFunnel(...)
                                           │ a.funnelSessions[id]=true
                                           │ a.funnelExpiry[id] = time.AfterFunc(dur, disableFunnelForSession)
                                           │ [NEW] mint read-only reusable code:
                                           │   rTok = read-only Claims → Sign
                                           │   code, _ = a.joinCodes.IssueReusable(rTok, dur)
                                           │   a.funnelReadCode[id] = code
                                           ▼
                                        SetSessionFunnelResponse{FunnelURL, [NEW] PublicReadCode}


 2. Owner's modal warm-up
    poll flips funnelActive ────────────────────────────────────────────────▶  useEffect (SessionShareModal.tsx:371-392)
                                                                                    │ IssueCapabilities(id) [re-issue]
                                        POST /sessions/{id}/capabilities ◀─────────┘
                                        issueCapabilitiesForSession (api.go:1336)
                                           │ isFunnelSession → base = FunnelBaseURL()
                                           │ [NEW] returns cached a.funnelReadCode[id]
                                           │        (does NOT mint a new one — idempotent)
                                           ▼
                                        IssueCapabilitiesResponse{ReadURL, WriteURL,
                                                                   ReadCode, WriteCode,
                                                                   [NEW] PublicReadCode}
                                                                                    │
                                                                                    ▼
                                                                                 SessionSharePanel.tsx
                                                                                 "Internet (public)" section
                                                                                 renders <CodeDisplay code={publicReadCode}/>


 3. Recipient types the short code
    at https://host.ts.net/join
    (join.html, GET /join) ─────────▶  handleJoin (server.go:1051) serves static join.html
                                                                                 (no daemon/Wails involvement — public path)

 4. Recipient submits the code
    (POST /join/exchange, form) ────▶  handleJoinExchange (server.go:1072)
                                           │ ws.joinCodes.Exchange(code)   ◀── SAME JoinCodeManager instance
                                           │   [NEW] entry.reusable==true → TTL-checked, NOT deleted
                                           │ 303 redirect → /sessions/{id}?cap=<readOnlyToken>
                                           ▼
                                        Recipient's browser follows redirect,
                                        joins read-only, session stays reachable
                                        for the NEXT recipient with the same code


 5. Funnel torn down (any of 4 triggers:
    toggle-off / web-share-off /
    session-exit / auto-expiry timer) ─▶  disableFunnelForSession (api.go:1645)
                                           │ [NEW] if code, ok := a.funnelReadCode[id]; ok {
                                           │          a.joinCodes.Revoke(code)
                                           │          delete(a.funnelReadCode, id)
                                           │       }
                                           │ existing: stop timer, delete funnelSessions[id],
                                           │           ws.DisableFunnel() if ref-count==0
                                           ▼
                                        Code no longer resolves — /join/exchange now
                                        404s exactly when the share tears down, not just
                                        when the TTL backstop eventually elapses.
```

### Recommended Code Structure (extensions to existing files — no new files needed for the Go side)

```
internal/capability/joincode.go     # + reusable field on joinEntry, + IssueReusable(), + Revoke()
internal/daemon/api.go              # + funnelReadCode map, + mint-on-enable in handleSetSessionFunnel,
                                     #   + cache-and-reuse in issueCapabilitiesForSession,
                                     #   + revoke-on-teardown in disableFunnelForSession
internal/daemon/types.go            # + PublicReadCode field on SetSessionFunnelResponse and/or
                                     #   IssueCapabilitiesResponse (see Open Questions for which)
frontend/src/components/SessionSharePanel.tsx   # + <CodeDisplay> row in the "Internet (public)" block
frontend/src/components/Hub/SessionShareModal.tsx  # + thread publicReadCode through cachedShare/funnelUrl state
frontend/src/wailsjs/wailsjs/go/models.ts        # + mirror the new Go struct field (hand-synced, see Pitfall 3)
```

### Pattern 1: Idempotent code minting keyed by session (NEW pattern for this phase)

**What:** Unlike the existing Read-Only/Full-Access codes — which are deliberately re-minted (and thus silently rotated) on every `IssueCapabilities` call (browse toggle, warm-up re-issue, server-restart reseed; see `SessionShareModal.tsx:189-198`, `:215-220`, `:274-279`) — the public reusable code must be minted **once per Funnel-enable** and then always returned as-is on subsequent lookups. Re-minting on every call would silently invalidate a code already read aloud or copy-pasted to a remote viewer.
**When to use:** Any time a capability artifact is handed to an anonymous, potentially-offline recipient who cannot be pushed a live update.
**Example:**
```go
// Source: pattern derived from internal/daemon/api.go:1336-1430 (issueCapabilitiesForSession),
// extended for idempotent reuse — no direct upstream precedent, this is new for Phase 170.
a.mu.Lock()
code, alreadyMinted := a.funnelReadCode[sessionID]
a.mu.Unlock()
if !alreadyMinted {
    code, err = a.joinCodes.IssueReusable(rTok, remainingFunnelTTL(sessionID))
    if err != nil { return /* ... */ }
    a.mu.Lock()
    a.funnelReadCode[sessionID] = code
    a.mu.Unlock()
}
```

### Pattern 2: Single-chokepoint teardown revocation

**What:** `disableFunnelForSession` (`api.go:1645-1662`) is documented as the single function every in-process Funnel teardown trigger routes through (toggle-off site 1, web-share-off site 2, session-exit site 3, expiry-timer site 5 — daemon-stop site 4 is a process-exit case where in-memory state is discarded wholesale, so no explicit revocation is needed there). Hooking the reusable-code revocation into this one function, rather than duplicating it at each of the 4 call sites, guarantees FNL-08's "dies exactly when the share does" for every teardown path with one code change.
**When to use:** Whenever a resource's lifetime must track a session/session-feature's lifetime and there is an existing single-teardown-function pattern to hook into.
**Example:**
```go
// Source: internal/daemon/api.go:1645-1662 (existing disableFunnelForSession),
// extended with the revocation hook (new for Phase 170).
func (a *API) disableFunnelForSession(ctx context.Context, sessionID string) {
	a.mu.Lock()
	if t, ok := a.funnelExpiry[sessionID]; ok {
		t.Stop()
		delete(a.funnelExpiry, sessionID)
	}
	if code, ok := a.funnelReadCode[sessionID]; ok { // NEW
		a.joinCodes.Revoke(code)                      // NEW
		delete(a.funnelReadCode, sessionID)            // NEW
	}
	delete(a.funnelSessions, sessionID)
	remaining := len(a.funnelSessions)
	ws := a.webServer
	a.mu.Unlock()

	if ws != nil && remaining == 0 {
		_ = ws.DisableFunnel(ctx)
	}
}
```

### Anti-Patterns to Avoid

- **Relying on TTL alone for "dies exactly when the share does":** the existing `funnelExpiry` mechanism is a `time.AfterFunc` timer, not a stored absolute deadline (there is no `funnelExpiryAt map[string]time.Time` today — only `funnelExpiry map[string]*time.Timer`, `api.go:76-80`). A TTL-only public code would still resolve for its remaining TTL window after a MANUAL disable (toggle-off), which is a real security gap — a "torn down" share would still be joinable by anyone holding the code until the TTL backstop elapses. Explicit revocation in `disableFunnelForSession` (Pattern 2) is required, not optional.
- **Re-minting the reusable code on every `IssueCapabilities` call:** the RO/Full-Access codes intentionally rotate on every re-issue (browse toggle, warm-up re-issue) because they're single-use anyway. Applying the same rotation to the reusable public code breaks the "reusable" contract — see Pattern 1.
- **Sharing one `JoinCodeManager.ttl` for both single-use and reusable codes:** the manager's `ttl` field (`joincode.go:36`) is a single duration used by `Issue`. Do not repurpose it as a global reusable-TTL; add a per-call TTL parameter instead (`IssueReusable(token, ttl)`), or the existing 5-minute single-use codes and the up-to-8-hour public code cannot coexist correctly.
- **Letting the reusable code resolve to the write-capability token:** `issueCapabilitiesForSession` mints both `rTok` (perms `"read"` or `"read,files.read"`) and `wTok` (perms `"read,write"` or more) at lines 1382-1389. The public reusable code must only ever be issued against `rTok`. Phase 171 (FNL-09) is the ONLY phase that should introduce a public write code — Phase 170 must not create a code path that could resolve to `wTok`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Short-code generation / entropy | A new custom code generator | Reuse `joinCodeEncoding` (base32 RFC4648, no 0/O/1/I/l ambiguity) and the existing 5-byte `crypto/rand.Read` pattern from `Issue` (`joincode.go:54-61`) inside the new `IssueReusable` | The locked constraint explicitly requires keeping the 40-bit entropy; duplicating or reinventing the encoding risks introducing ambiguous characters or a weaker RNG |
| TOCTOU-safe lookup+consume | A read-then-write two-step check | The existing single-mutex-hold pattern in `Exchange` (`joincode.go:79-93`), extended with a `reusable` branch that skips only the `delete` call, keeping the same lock scope | The existing `TestJoinCodeManager_ConcurrentExchangeIsAtomic` test (`joincode_test.go:98-123`) proves the current atomicity; any rewrite that doesn't preserve one-mutex-hold-per-Exchange-call risks reintroducing the exact race the comment at `joincode.go:28-32` calls out |
| Absolute funnel-expiry deadline tracking | A parallel expiry-timestamp cache computed independently in the frontend or in a new subsystem | Compute the TTL for the public code directly from `req.ExpiresIn` at the moment `handleSetSessionFunnel` (api.go:1581) processes the enable request — this is the one place that already has the authoritative duration | `req.ExpiresIn` (seconds) is already validated/typed in `SetSessionFunnelRequest` (`types.go:154-159`); recomputing it elsewhere risks drift from the real timer |

**Key insight:** Every piece of machinery this phase needs — entropy generation, TOCTOU-safe consumption, and the single-teardown-chokepoint — already exists in the codebase for a slightly different purpose (single-use codes). The correct design is surgical extension of `JoinCodeManager` and `disableFunnelForSession`, not a parallel implementation.

## Common Pitfalls

### Pitfall 1: TTL-only revocation leaves a "ghost" public entry point
**What goes wrong:** If the reusable code's only defense is its TTL, then disabling Funnel manually (or via web-share-off, or session-exit) leaves the code valid and resolvable via `/join/exchange` until the TTL naturally elapses — even though the session is no longer publicly exposed by design.
**Why it happens:** The existing single-use codes never needed this distinction because `Exchange` deletes them on first use regardless of TTL — a manual "disable" was never a coherent concept for a single-use code (whoever redeemed it, redeemed it).
**How to avoid:** Explicit `Revoke(code)` call inside `disableFunnelForSession` (Pattern 2), not reliance on TTL expiry.
**Warning signs:** A test that disables Funnel and then asserts the OLD public code still returns 200/303 from `/join/exchange` instead of 404/410.

### Pitfall 2: Re-minting the code on every `IssueCapabilities` call silently breaks distributed codes
**What goes wrong:** `issueCapabilitiesForSession` is called repeatedly during a session's Funnel lifetime — at warm-up completion (`SessionShareModal.tsx:377`), on every browse toggle (`SessionShareModal.tsx:273`), and on server-restart reseed (`SessionShareModal.tsx:187`). If the public code is minted the same way as `readCode`/`writeCode` (unconditional `a.joinCodes.Issue(rTok)` on every call), it rotates every time any of those triggers fire, silently breaking a code already given to a remote recipient.
**Why it happens:** Copy-pasting the existing `readCode, err = a.joinCodes.Issue(rTok)` line (api.go:1421) without adding the idempotency check.
**How to avoid:** Cache-and-reuse per session (Pattern 1) — check `a.funnelReadCode[sessionID]` before minting.
**Warning signs:** A test/manual-UAT where the public code visibly changes value after toggling "Enable remote file browsing" while Funnel is active.

### Pitfall 3: Wails TS binding drift
**What goes wrong:** `frontend/src/wailsjs/wailsjs/go/models.ts` mirrors Go structs (`IssueCapabilitiesResponse` class, `models.ts:33-52`) but is **hand-synced**, not build-time-generated in this repo's current workflow (see the most recent commit in git history, `fb7963f1 chore(169): sync wails TS binding — add TailscaleHealth.permissionLimited`, which is a dedicated sync commit separate from the feature commit). Adding a new Go field without a matching `models.ts` edit means `resp.publicReadCode` (or whatever name is chosen) is `undefined` at runtime in the compiled frontend even though TypeScript may not catch it if the field is optional.
**Why it happens:** No CI step regenerates `models.ts` automatically from `internal/daemon/types.go` in this repo's current setup (confirmed: no `wails generate module` step found wired into the build/test scripts checked during this research — flag as `[ASSUMED]`, not exhaustively verified against every CI config file).
**How to avoid:** Any plan that adds a field to `SetSessionFunnelResponse` or `IssueCapabilitiesResponse` must include a task to hand-edit `models.ts` in the same PR, mirroring the `fb7963f1` precedent.
**Warning signs:** `pnpm tsc` passes (per project's own gotcha in CLAUDE.md: "vitest tolerates TS errors that `tsc && vite build` rejects") but the field silently reads as `undefined` in a `wails dev` / production build.

### Pitfall 4: "Until I disable" (ExpiresIn=0) has no natural TTL for the public code
**What goes wrong:** `FunnelRiskPanel.tsx:23-29` offers a `0` sentinel ("Until I disable") that skips registering a `funnelExpiry` timer entirely (`api.go:1617`: `if req.ExpiresIn > 0 { ... }`). The reusable public code's TTL is supposed to be "tied to the funnel auto-expiry" — but there IS no auto-expiry in this case, so a literal TTL binding has nothing to bind to.
**Why it happens:** The design constraint text ("tied to the funnel auto-expiry ... dies exactly when the share does") conflates two mechanisms: (a) TTL matching the auto-expiry duration, and (b) explicit revocation at manual-disable time. Mechanism (b) covers the ExpiresIn=0 case correctly (Pitfall 1's fix already handles it); mechanism (a) simply has no value to use when ExpiresIn=0.
**How to avoid:** Treat explicit revocation (Pattern 2) as the PRIMARY mechanism for "dies when the share does" in all cases; treat the TTL as a defense-in-depth backstop that should use a generous fallback (recommend capping at the 8-hour max preset, i.e. `28800` seconds) when `ExpiresIn==0`, so a code cannot outlive a crashed-daemon or edge-case-missed-revocation scenario indefinitely. **This fallback value is an assumption for the planner/user to confirm — see Open Questions.**
**Warning signs:** A public code minted under "Until I disable" that never expires even across a daemon restart (daemon restart wipes `JoinCodeManager` in-memory anyway — `SessionShareModal.tsx:165-167` comment confirms "the daemon's JoinCodeManager is wiped" on restart — so this is bounded by process lifetime at minimum, but not by the 8h entropy-safety assumption during a single long-lived daemon run).

## Code Examples

### Existing single-use Issue/Exchange (the pattern being extended)
```go
// Source: internal/capability/joincode.go:54-93 (current code, verbatim)
func (m *JoinCodeManager) Issue(token string) (string, error) {
	var raw [5]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	encoded := joinCodeEncoding.EncodeToString(raw[:])
	code := encoded[:4] + "-" + encoded[4:8]

	m.mu.Lock()
	m.codes[code] = joinEntry{token: token, expiry: m.now().Add(m.ttl)}
	m.mu.Unlock()
	return code, nil
}

func (m *JoinCodeManager) Exchange(code string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.codes[code]
	if !ok {
		return "", ErrCodeNotFound
	}
	if m.now().After(entry.expiry) {
		delete(m.codes, code)
		return "", ErrCodeExpired
	}
	delete(m.codes, code) // <-- must become conditional on !entry.reusable
	return entry.token, nil
}
```

### Existing Funnel-aware URL swap (the pattern the public code's TTL/revocation must plug into)
```go
// Source: internal/daemon/api.go:1400-1429 (issueCapabilitiesForSession, current code, verbatim)
ws.AddGrant(sessionID, rClaims.GrantID)
ws.AddGrant(sessionID, wClaims.GrantID)

base := ws.BaseURL()
a.mu.RLock()
isFunnelSession := a.funnelSessions[sessionID]
a.mu.RUnlock()
if isFunnelSession {
	if fb := ws.FunnelBaseURL(); fb != "" {
		base = fb
	}
}
readURL = base + "/sessions/" + sessionID + "?cap=" + rTok
writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok

readCode, err = a.joinCodes.Issue(rTok)
// ... writeCode = a.joinCodes.Issue(wTok) similarly
```

### Existing frontend CodeDisplay component to reuse verbatim for the public section
```tsx
// Source: frontend/src/components/SessionSharePanel.tsx:8-57 (current code, reuse as-is)
function CodeDisplay({ label, code }: { label: string; code: string }): React.ReactElement {
  const [copied, setCopied] = useState(false)
  async function handleCopyCode(): Promise<void> {
    try {
      await ClipboardSetText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch { /* clipboard failure — code remains visible for manual copy */ }
  }
  return ( /* ... existing markup, data-testid="join-code-text" ... */ )
}
```
The "Internet (public)" block (`SessionSharePanel.tsx:315-382`) should gain a `<CodeDisplay label="Join code:" code={publicReadCode} />` row placed after the existing public-URL row (~line 360), mirroring the placement pattern already used for the Read-Only (`:249`) and Full-Access (`:300`) sections.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| All join codes: 5-minute TTL, always single-use | Two code classes: single-use 5-min (RO/Full-Access links, unchanged) + reusable, per-share-lifetime TTL (new public code) | Phase 170 (this phase) | `JoinCodeManager` becomes a dual-mode manager; `Exchange` gains a conditional-delete branch keyed on the new `reusable` field |
| Public Funnel section shows URL + QR only | Public Funnel section shows URL + QR + join code | Phase 170 | Closes the FNL-08 UAT dead-end (typing the URL with no code available) |

**Deprecated/outdated:** None — this is additive, not a replacement of the FNL-03 single-use code flow for RO/Full-Access links (explicitly required by the "Supplement, not replace" locked decision).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | New Go method names `IssueReusable(token string, ttl time.Duration) (string, error)` and `Revoke(code string)` are the right shape/names — not found in the codebase, proposed by this research | Architecture Patterns, Don't Hand-Roll | Low — purely a naming/API-shape choice the planner is free to adjust; the underlying mechanism (per-entry reusable flag + explicit delete) is what matters |
| A2 | The new field should be named `PublicReadCode` (Go) / `publicReadCode` (JSON/TS) and delivered via both `SetSessionFunnelResponse` (immediate, at enable time) and cached for retrieval via `IssueCapabilitiesResponse` (for modal-reopen / warm-up re-issue) | Architecture Patterns diagram, Pitfall 3 | Low-medium — if the planner instead adds a dedicated new endpoint (e.g. `GET /sessions/{id}/funnel/code`) or a new `SessionInfo` field, the frontend wiring changes shape but the backend mechanics (idempotent mint + explicit revoke) are unaffected |
| A3 | When `ExpiresIn==0` ("Until I disable"), the recommended TTL backstop for the public code should default to the max preset (8 hours / 28800s) rather than being unbounded or erroring | Common Pitfalls (Pitfall 4) | Medium — this is a product/security decision, not a code-archaeology fact; the planner or `/gsd-discuss-phase` follow-up should confirm this default explicitly since it affects the "not brute-forceable" entropy argument (2⁴⁰ over an *unbounded* window is a materially different security claim than over ≤8h) |
| A4 | No CI step auto-regenerates `frontend/src/wailsjs/wailsjs/go/models.ts` from Go structs — based on observing the `fb7963f1` "sync wails TS binding" commit pattern in git log, not an exhaustive audit of every CI workflow file | Common Pitfalls (Pitfall 3) | Low — worst case the planner adds a redundant manual-sync task that a generator would have made unnecessary; does not block correctness |
| A5 | `disableFunnelForSession` site 4 (daemon stop via `ws.Stop()`) does not need an explicit `Revoke` call because process exit discards the in-memory `JoinCodeManager` wholesale | Architecture Patterns (Pattern 2) | Low — consistent with the existing code comment at `api.go:1640-1641` ("Site 4 ... is covered by ws.Stop()→DisableFunnel ... do NOT add a second call there") which establishes the same reasoning for the existing Funnel-session cleanup |

**If this table is empty:** N/A — see rows above; none of these are compliance/retention/performance claims, all are implementation-shape choices flagged for planner/user confirmation.

## Open Questions

1. **Where does `PublicReadCode` live in the response wire types — `SetSessionFunnelResponse`, `IssueCapabilitiesResponse`, or both?**
   - What we know: `SetSessionFunnelResponse` (types.go:161-166) is returned once, synchronously, from the enable call — but the frontend's actual URL-display flow re-issues via `IssueCapabilities` after the warm-up poll confirms `funnelActive` (`SessionShareModal.tsx:371-392`), and needs the code available on modal-reopen for an already-active share (when `SetSessionFunnelResponse` was never re-called).
   - What's unclear: Whether to duplicate the field on both response types (simplest, some redundancy) or introduce a new dedicated accessor.
   - Recommendation: Add it to `IssueCapabilitiesResponse` (idempotently cached server-side, per Pattern 1) since that is the call already re-invoked on warm-up-completion and on modal-reopen; optionally omit it from `SetSessionFunnelResponse` entirely to minimize the wire surface, since the frontend's warm-up effect will fetch it moments later.

2. **What TTL should back the public code when the user picks "Until I disable" (`ExpiresIn == 0`)?**
   - What we know: The FunnelRiskPanel's max preset is 28800s (8h) — the exact number the locked design constraint's "≤8h window" entropy argument is built on.
   - What's unclear: Whether a code with no natural TTL should (a) get an 8h backstop TTL despite the user's "until I disable" choice, (b) get an unbounded/very-long TTL and rely 100% on explicit revocation, or (c) simply not offer the public reusable code at all when ExpiresIn==0.
   - Recommendation: Confirm with the user/planner before locking; this research recommends option (a) as the safest default that preserves the stated entropy-safety argument, but it is a product decision (Assumption A3), not a code fact.

3. **Should the reusable code display distinguish itself visually/copy-wise from the single-use RO/Full-Access codes (e.g. "Public join code (reusable):" vs "Join code:")?**
   - What we know: The existing `CodeDisplay` component takes a `label` prop already (`SessionSharePanel.tsx:9-13`), so differentiated labeling is a one-line change.
   - What's unclear: Whether product/UX wants explicit "reusable" language to set recipient expectations, given the RO/Full-Access codes are single-use and look identical otherwise.
   - Recommendation: Planner should pick a label distinct from the existing bare "Join code:" (e.g. "Public join code:") to avoid recipient confusion about single-use vs. reusable semantics — low-risk UX polish, not a blocking decision.

## Environment Availability

Skipped — this phase has no new external tool/service/runtime dependencies. It extends existing Go stdlib code and existing React components already present and building in this repo; Tailscale Funnel infrastructure itself was already verified available and working in Phase 165/166 (see STATE.md M-37..M-40 live UAT results).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) + vitest 3.x |
| Config file | none dedicated — `go test ./...` at repo root; `frontend/vite.config.ts` / `package.json` `"test": "vitest run"` |
| Quick run command | `go test ./internal/capability/... ./internal/daemon/... -run JoinCode -v` (backend) / `cd frontend && pnpm test -- SessionSharePanel SessionShareModal` (frontend) |
| Full suite command | `go test -race -short ./...` (Go) and `cd frontend && pnpm test` (vitest) — both already wired into CI per `TESTING.md` Section 2 |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FNL-08 (reusable, multi-use) | A reusable code can be exchanged more than once before expiry/revocation | unit | `go test ./internal/capability/... -run TestJoinCodeManager_IssueReusable -v` | ❌ Wave 0 — new test in `internal/capability/joincode_test.go` |
| FNL-08 (read-only scope) | The reusable code's `Exchange` result decodes to a token whose `Perms` never contains `write` | unit | `go test ./internal/daemon/... -run TestFunnelPublicCode_ReadOnlyScope -v` | ❌ Wave 0 — new test in `internal/daemon/funnel_test.go` (extends the existing `TestIssueCapabilities_FunnelURL`/`TestExchangeJoinCode_FunnelURL_GateIntact` file) |
| FNL-08 (TTL tied to funnel expiry, timer path) | After the funnel auto-expiry timer fires, the public code no longer resolves | integration | `go test ./internal/daemon/... -run TestFunnelAutoExpiry -v` (extend existing test) | ✅ exists, needs extension — `internal/daemon/funnel_test.go:270-323` |
| FNL-08 (dies on ALL teardown triggers, not just the timer) | Each of the 4 in-process teardown triggers (toggle-off, web-share-off, session-exit, timer) immediately invalidates the public code | integration | `go test ./internal/daemon/... -run TestFunnelTeardown_AllTriggers -v` (extend existing table test) | ✅ exists, needs extension — `internal/daemon/funnel_test.go:333-` |
| FNL-08 (idempotent — not rotated on repeat IssueCapabilities calls) | Calling `IssueCapabilities` twice while Funnel is active returns the SAME public code both times | unit/integration | `go test ./internal/daemon/... -run TestIssueCapabilities_FunnelPublicCode_Idempotent -v` | ❌ Wave 0 — new test, same file as above |
| FNL-08 (frontend display) | The "Internet (public)" section renders a `<CodeDisplay>` row with the public code | component | `cd frontend && pnpm test -- SessionSharePanel` (extend existing file) | ✅ exists, needs extension — `frontend/src/components/__tests__/SessionSharePanel.test.tsx` |
| FNL-08 (code-entry page reuse works) | `/join/exchange` (the public browser-facing endpoint) accepts the reusable code and does NOT 404 on a second use before expiry | integration | `go test ./internal/webserver/... -run TestJoinExchange -v` | ❌ Wave 0 — no dedicated webserver join test file exists today (current coverage is indirect via `csp_integration_test.go` asserting only HTTP 200 on `/join`); recommend a small new `internal/webserver/join_test.go` covering the reusable-code double-exchange case directly against `handleJoinExchange` |

### Sampling Rate
- **Per task commit:** `go test ./internal/capability/... ./internal/daemon/... -run JoinCode -v` and/or `cd frontend && pnpm test -- SessionSharePanel SessionShareModal` depending on which layer the task touches
- **Per wave merge:** `go test -race -short ./...` and `cd frontend && pnpm test`
- **Phase gate:** Full suite green (`go test -race -short ./...` + `cd frontend && pnpm test` + `cd frontend && pnpm exec tsc --noEmit` per the project's own "run tsc in the frontend gate" standing lesson) before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/capability/joincode_test.go` — extend with reusable-code tests (multi-exchange success, TTL-expiry-then-404, revoke-then-404) covering the new `IssueReusable`/`Revoke` methods
- [ ] `internal/daemon/funnel_test.go` — extend `TestFunnelAutoExpiry` and `TestFunnelTeardown_AllTriggers`; add a new idempotent-mint test and a read-only-scope test
- [ ] `internal/webserver/join_test.go` — new file; no dedicated join-exchange test file exists today (current coverage is indirect via `csp_integration_test.go`'s bare-200 assertion) — needed to directly prove the reusable code survives a second `/join/exchange` POST at the public HTTP layer, not just at the `JoinCodeManager` unit-test layer
- [ ] `frontend/src/components/__tests__/SessionSharePanel.test.tsx` — extend with a public-join-code-visible assertion in the "Internet (public)" section tests
- [ ] `TESTING.md` — per the project's own standing rule (`.claude/CLAUDE.md` / repo `CLAUDE.md`), add a new Suite Manifest note (Section 2) and traceability rows for FNL-08 (Section 4) once the new/extended test files land; run `bash tests/check-traceability-paths.sh` before committing

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | no | This is a capability/possession-based join flow (D-09), not password authentication — out of scope |
| V3 Session Management | no | No session/cookie state introduced; the capability token itself is the bearer credential |
| V4 Access Control | **yes** | The reusable code MUST resolve only to the read-scoped `rTok` (`Perms` never containing `write`/`files.write`) — enforced entirely server-side in `issueCapabilitiesForSession`'s existing perm-matrix logic (`api.go:1382-1389`); the new code must be minted against `rClaims`/`rTok` only, never `wClaims`/`wTok` |
| V5 Input Validation | yes | The `/join/exchange` form body (`code` field) is already validated for emptiness (`server.go:1077-1081`) — no new validation surface introduced, the reusable-code path reuses the identical `jc.Exchange(code)` call |
| V6 Cryptography | yes | 40-bit `crypto/rand` entropy — locked constraint explicitly retains this; never hand-roll a weaker generator for the reusable path (see Don't Hand-Roll) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Reusable code outliving its share (ghost access after manual disable) | Elevation of Privilege / Information Disclosure | Explicit `Revoke()` call inside `disableFunnelForSession` at all 4 in-process teardown sites (Pattern 2) — do not rely on TTL alone (Pitfall 1) |
| Reusable code silently rotated, breaking the "reusable" contract but ALSO potentially leaving the OLD code still resolvable if not explicitly revoked on rotation | Tampering / availability regression (not a security hole per se, but a correctness bug with security-adjacent trust implications — a user might believe an old code is dead when it is still live) | Idempotent mint-once-per-enable (Pattern 1); if a future change legitimately needs to rotate the code, it must also `Revoke` the old one in the same operation |
| Brute-force guessing of the 8-char code | Spoofing | Already mitigated by the existing 40-bit entropy (2^40 ≈ 1.1×10^12 combinations) — locked constraint explicitly re-affirms this is sufficient without rate-limiting for an ≤8h window; do not weaken the alphabet or byte count when adding `IssueReusable` |
| Public read code accidentally granted `files.write` when browse is toggled ON for a Funnel session | Elevation of Privilege | The existing perm-matrix (D-03/D-04, `api.go:1369-1387`) already scopes `files.write` to the write token only, even with browse ON (`rPerms` gets only `files.read`, never `files.write`) — the new reusable code inherits this correctly AS LONG AS it is minted from `rTok`/`rClaims`, never independently constructed |

## Sources

### Primary (HIGH confidence — direct file reads this session)
- `internal/capability/joincode.go` (full file) — JoinCodeManager struct, Issue, Exchange, entropy/encoding
- `internal/capability/joincode_test.go` (full file) — existing test coverage/contract for single-use codes
- `internal/capability/errors.go`, `internal/capability/export_test.go` — sentinel errors, test clock seam
- `internal/daemon/api.go` (lines 45-80, 172, 239-275, 1330-1662) — funnelSessions/funnelExpiry maps, issueCapabilitiesForSession, handleIssueCapabilities, handleExchangeJoinCode, handleSetSessionFunnel, disableFunnelForSession
- `internal/daemon/types.go` (full file) — SessionInfo, SetSessionFunnelRequest/Response, IssueCapabilitiesResponse, ExchangeJoinCodeRequest/Response
- `internal/daemon/funnel_test.go` (lines 1-60, 263-323, 482-538) — existing Funnel test patterns and fixtures
- `internal/webserver/server.go` (lines 1040-1125, plus grep hits at 115-119, 345-351) — handleJoin, handleJoinExchange, joinCodes field/SetJoinCodes
- `frontend/src/components/Hub/SessionShareModal.tsx` (lines 1-50, 140-450) — cachedShare state, seeding effects, Funnel warm-up state machine
- `frontend/src/components/SessionSharePanel.tsx` (full file) — CodeDisplay component, Internet (public) section (the exact gap)
- `frontend/src/components/Hub/FunnelRiskPanel.tsx` (full file) — EXPIRY_OPTIONS presets (confirms 8h max / 0-sentinel)
- `frontend/src/wailsjs/wailsjs/go/models.ts` (lines 1-52) — hand-synced TS mirror of Go wire types
- `app.go` (lines 1040-1178) — Wails-bound App methods (SetSessionFunnel, IssueCapabilities, ExchangeJoinCode, GetCapabilityQRCode)
- `.planning/ROADMAP.md` (Phase 170 section, lines 392, 571-590) — locked design constraints
- `.planning/REQUIREMENTS.md` (FNL-08 row, lines 16-22, 92) — requirement text and phase mapping
- `.planning/STATE.md` (v4.2 section) — UAT finding that motivated this phase, root-cause note on api.go:259

### Secondary (MEDIUM confidence)
- `TESTING.md` (Suite Manifest lines 22-44, traceability rows 326-336) — current test counts and FNL-01/03/05/07 traceability patterns to mirror for FNL-08

### Tertiary (LOW confidence / assumed)
- Whether any CI workflow auto-regenerates `models.ts` (Assumption A4) — inferred from a single git-log commit pattern, not an exhaustive CI config audit

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, pure extension of existing stdlib-only code already in the repo
- Architecture: HIGH — every claim traced to a specific file:line in the current codebase; the extension points (Exchange's delete call, disableFunnelForSession's single-chokepoint teardown, issueCapabilitiesForSession's perm-matrix) are all read directly, not inferred
- Pitfalls: HIGH — Pitfalls 1-3 are derived directly from reading the exact code paths involved (timer-vs-deadline gap, re-mint-on-every-call pattern, hand-synced models.ts precedent); Pitfall 4 / Open Question 2 is a genuine product-decision gap, correctly flagged as such rather than guessed

**Research date:** 2026-07-05
**Valid until:** 30 days (stable internal codebase, no external API/version drift risk — the only invalidation vector is if Phase 171 or another concurrent phase touches `JoinCodeManager`/`disableFunnelForSession` first, which would require re-reading those files before planning)
