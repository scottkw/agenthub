# Phase 171: Public Full-Access (Read-Write) Sharing - Pattern Map

**Mapped:** 2026-07-07
**Files analyzed:** 15 (11 backend/Go, 4 frontend/TS-TSX + CSS)
**Analogs found:** 15 / 15 (all covered — this phase is additive-to-existing-files; RESEARCH.md's Phase-170 precedent is the closest analog for every file)

**Note on analog choice:** Per orchestrator guidance, Phase 170's reusable-read-code lifecycle (`funnelReadCode`, `handleSetSessionFunnel`, `SessionShareModal`/`CodeDisplay`, Funnel serve-config path) is the primary analog set for every new piece of Phase 171 work, since 171 is "the same lifecycle shape, single-use + write-scoped, with a consent gate in front." RESEARCH.md already performed this exact extraction with verified line numbers; this document packages it as planner-ready per-file assignments.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/capability/joincode.go` (+`IssueSingleUseWithTTL`) | service (crypto/token mint) | CRUD (mint/expire) | `IssueReusable` in same file (joincode.go:93-105) | exact |
| `internal/webserver/server.go` (+`RemoveGrant`, `SetRWGate`, `isRWGated`) | service (in-memory registry) | CRUD (add/remove/check) | `AddGrant`/`ClearGrants`/`isGrantActive` in same file (server.go:303-330) | exact |
| `internal/webserver/capability_mw.go` (`originAllowedForWrite` +gate check) | middleware | request-response | same function, existing gate-unaware version (capability_mw.go:191-210) | exact |
| `internal/daemon/api.go` (+`handleSetSessionFunnelWrite`) | controller/handler | request-response | `handleSetSessionFunnel` (api.go: ExpiresIn handling ~1750-1783) | exact |
| `internal/daemon/api.go` (+`revokeFunnelWriteLocked`/`disableFunnelWriteForSession`) | service (teardown helper) | event-driven (multi-trigger revocation) | inlined `funnelReadCode` cleanup block inside `disableFunnelForSession` (api.go:1800-1819) | exact |
| `internal/daemon/api.go` (+`issueCapabilitiesForSession` write-rebase removal) | service | CRUD | same function, existing write-rebase block (api.go:1451-1472) | exact |
| `internal/daemon/api.go` (+funnel-write route registration) | route | request-response | `/sessions/{id}/funnel` registration (api.go ~216) | exact |
| `internal/daemon/types.go` (+`SetSessionFunnelWriteRequest/Response`, `SessionInfo.FunnelWriteActive`) | model | CRUD | `SetSessionFunnelRequest/Response`, `SessionInfo.FunnelActive` (types.go) | exact |
| `internal/daemon/client.go` (+`SetSessionFunnelWrite`) | service (RPC client) | request-response | `DaemonClient.SetSessionFunnel` (client.go ~402) | exact |
| `app.go` (+`App.SetSessionFunnelWrite`) | controller (Wails binding) | request-response | `App.SetSessionFunnel` (app.go ~1057) | exact |
| `internal/daemon/funnel_test.go` (+ new RW tests) | test | request-response / integration | `TestFunnelPublicCode_ReadOnlyScope`, `TestFunnelTeardown_AllTriggers`, `TestHandleSetSessionFunnel_Enable`, `TestFunnelAutoExpiry` (funnel_test.go:169,272,339,516) | exact |
| `frontend/src/components/SessionSharePanel.tsx` (+Danger section) | component | request-response (RPC-driven UI) | Phase-166 "Internet (public)" section in same file | exact |
| `frontend/src/components/Hub/SessionShareModal.tsx` (+hold-gate wiring) | component/hook (state+RPC orchestration) | request-response | existing `funnelOn`/`warmingUp`/`publicReadCode` state wiring in same file (Phase 170) | exact |
| `frontend/src/components/Hub/SessionCard.tsx` (+`FULL ACCESS` badge) | component | CRUD (render from poll state) | `.hub-internet-badge` render block (SessionCard.tsx:538-543) | exact |
| `frontend/src/style.css` (+`.hub-fullaccess-badge*`) | config (styles) | n/a | `.hub-internet-badge*` block + tokens (style.css:4707-4708, 4766-4767, 7181-7208) | exact |

## Pattern Assignments

### `internal/capability/joincode.go` — `IssueSingleUseWithTTL` (service, CRUD)

**Analog:** `IssueReusable`, same file, lines 93-105.

**Core pattern to copy** (mint a code, same crypto/rand + encoding, but per-call ttl instead of the manager's fixed field, and `reusable: false` — the zero value — so `Exchange` keeps its atomic delete-on-redeem single-use semantics):
```go
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
Do NOT touch `Exchange` (joincode.go:162-178) — it is already code-class-agnostic and provides the atomic single-mutex-hold lookup+delete that satisfies the concurrent-redeem guarantee (R2). No new redemption path needed.

---

### `internal/webserver/server.go` — `RemoveGrant`, `SetRWGate`/`isRWGated` (service, CRUD)

**Analog:** `AddGrant`/`ClearGrants`/`isGrantActive`, same file, lines 303-330 (verified live):
```go
func (ws *WebServer) AddGrant(sessionID, grantID string) {
	ws.mu.Lock()
	if ws.grants[sessionID] == nil {
		ws.grants[sessionID] = make(map[string]struct{})
	}
	ws.grants[sessionID][grantID] = struct{}{}
	ws.mu.Unlock()
}

func (ws *WebServer) ClearGrants(sessionID string) {
	ws.mu.Lock()
	delete(ws.grants, sessionID)
	ws.mu.Unlock()
}

func (ws *WebServer) isGrantActive(sessionID, grantID string) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	if ws.grants[sessionID] == nil {
		return false
	}
	_, ok := ws.grants[sessionID][grantID]
	...
}
```

**New surgical `RemoveGrant`** (do NOT reuse `ClearGrants` — it wipes every grant for the session, including the ordinary tailnet read/write grants; see RESEARCH Pitfall 2):
```go
func (ws *WebServer) RemoveGrant(sessionID, grantID string) {
	ws.mu.Lock()
	if ws.grants[sessionID] != nil {
		delete(ws.grants[sessionID], grantID)
	}
	ws.mu.Unlock()
}
```

**New `rwGated` map + accessors** — mirror the same lock discipline (`ws.mu`), colocated with `grants` in the `WebServer` struct:
```go
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

**Critical note (not a pattern-copy issue, an enforcement-scope issue):** `isGrantActive` — via `requireCapability`, wired ahead of `handleWSSRelay` — is the ONLY mechanism that reaches the actual PTY-write path (`MsgInput`/`MsgSessionInject`). `originAllowedForWrite` does not (RESEARCH Pitfall 1, verified via grep — single call site inside `requireFilesWrite`). Treat `RemoveGrant`/grant-gating as the primary enforcement; `SetRWGate`/`isRWGated` is real defense-in-depth for `originAllowedForWrite` but only reaches the `files.write` HTTP routes, which the gate cap never carries perms for anyway (D-05).

---

### `internal/webserver/capability_mw.go` — `originAllowedForWrite` gate-awareness (middleware, request-response)

**Analog:** existing function, capability_mw.go:191-210 (unchanged structure; add one `isRWGated` check inside).

**Read the existing function first** (Read tool, lines 185-215) before editing — the doc-comment above it is stale/aspirational (claims it covers MsgInput/MsgSessionInject; it does not). Do not copy that comment's claim into new code; correct it while editing.

---

### `internal/daemon/api.go` — `handleSetSessionFunnelWrite` (controller, request-response)

**Analog:** `handleSetSessionFunnel`'s `ExpiresIn` handling, api.go ~1750-1783 — BUT do not copy its `0 == unbounded` semantics (RESEARCH Pitfall 6). New handler clamps explicitly:
```go
if req.ExpiresIn <= 0 || req.ExpiresIn > 3600 {
	req.ExpiresIn = 3600
}
```

**Claims-construction analog** — `issueCapabilitiesForSession`'s write-cap claim block, api.go:1418-1436, ADAPTED (strip the `browseEnabledFor` branch entirely per D-05 / RESEARCH Pitfall 5):
```go
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
Response shape mirrors `SetSessionFunnelResponse` (api.go:1782) — add `{ writeURL, writeCode, expiresAt }`.

**Do NOT** put this new mint logic inside `issueCapabilitiesForSession` — build a wholly separate lifecycle (RESEARCH primary recommendation) so the tailnet Full Access Link's grant ID is never conflated with the public RW grant ID.

---

### `internal/daemon/api.go` — `disableFunnelForSession` + new `revokeFunnelWriteLocked`/`disableFunnelWriteForSession` (service, event-driven teardown)

**Analog:** the existing inlined `funnelReadCode` cleanup block inside `disableFunnelForSession`, api.go:1800-1819 (verified live):
```go
func (a *API) disableFunnelForSession(ctx context.Context, sessionID string) {
	a.mu.Lock()
	// Stop and remove the expiry timer if present (T-165-13 double-fire prevention).
	if t, ok := a.funnelExpiry[sessionID]; ok {
		t.Stop()
		delete(a.funnelExpiry, sessionID)
	}
	// FNL-08 / T-170-02: revoke the cached reusable public-share join code, if
	// one was ever minted, BEFORE the funnelSessions delete below...
	if code, ok := a.funnelReadCode[sessionID]; ok {
		a.joinCodes.Revoke(code)
		delete(a.funnelReadCode, sessionID)
	}
	delete(a.funnelReadCodeExpiry, sessionID)
	delete(a.funnelReadCodeTTL, sessionID)
	delete(a.funnelSessions, sessionID)
	remaining := len(a.funnelSessions)
	ws := a.webServer
	...
}
```

**New helper, mirroring this block's shape exactly, but scoped to RW only** (called from BOTH the narrow "disable RW" action AND appended as one new line inside the block above — never the reverse; RESEARCH Pattern 3 / Pitfall 3):
```go
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
**Do NOT** call `ClearGrants` or `disableFunnelForSession` for an RW-only disable — both have wider blast radius than the SPEC allows (see Common Pitfalls 2/3 in RESEARCH.md).

---

### `internal/daemon/api.go` — `issueCapabilitiesForSession` write-rebase removal (D-04)

**Analog:** the existing write-rebase block in the same function, api.go:1451-1472. Read this block directly (offset 1440, limit 40) before editing — remove the funnel-rebasing of `writeURL`/`writeCode` only; the tailnet-base `writeURL` computation stays untouched.

---

### `internal/daemon/types.go` — new request/response structs + `SessionInfo.FunnelWriteActive`

**Analog:** `SetSessionFunnelRequest`/`SetSessionFunnelResponse` and `SessionInfo.FunnelActive` field (same file). Mirror the JSON tag convention exactly — `FunnelActive` has **no** `omitempty` (false must serialize so pollers see it flip); `FunnelWriteActive` must follow the same rule.

---

### `internal/daemon/client.go` — `DaemonClient.SetSessionFunnelWrite`

**Analog:** `SetSessionFunnel` (client.go ~402) — same `doJSON` POST-to-loopback-socket pattern, new endpoint path `/sessions/{id}/funnel-write`.

---

### `app.go` — `App.SetSessionFunnelWrite` Wails binding

**Analog:** `App.SetSessionFunnel` (app.go ~1057) — thin passthrough to `DaemonClient`. Remember RESEARCH's noted Phase-170 precedent: hand-authored TS bindings must be added to BOTH `frontend/src/wailsjs/go/main/App.d.ts` and the corresponding `models.ts`/generated-mirror file — two files must sync, this is not auto-generated in this repo's workflow.

---

### `internal/daemon/funnel_test.go` — new RW tests (test, integration/unit)

**Analogs (all in same file, verified names from RESEARCH):**
- `TestFunnelPublicCode_ReadOnlyScope` (line 339) → mirror for `TestFunnelWriteGate_TerminalOnlyScope` (R3: perms exactly `read,write`, independent of browse toggle)
- `TestFunnelTeardown_AllTriggers` (line 516) → mirror for `TestDisableFunnelWrite_RevokesGrantOnly`, but ALSO assert the read code/grant survive an RW-only disable
- `TestHandleSetSessionFunnel_Enable` (line 272) / `TestFunnelAutoExpiry` (line 169) → mirror for `TestHandleSetSessionFunnelWrite_ExpiryClamp` (R5)
- **New, most critical, no existing analog in this file** — `TestHandleWSSRelay_WriteCap_RequiresGate` in `internal/webserver` package: must exercise the REAL `GET /sessions/{id}/ws` upgrade path with a gate-minted write cap whose grant has NOT been registered, asserting the upgrade itself fails. Do not write only an `originAllowedForWrite` unit test and consider R4 covered (RESEARCH Pitfall 1 — this is the single most consequential testing gap identified).

---

### `frontend/src/components/SessionSharePanel.tsx` — Danger section (component, request-response)

**Analog:** the existing Phase-166 "Internet (public)" section in the same file (`funnelEngaged`, `publicReadCode` state) — read this section directly before writing the Danger section; place it physically below/separated per D-06.

**Secondary analog for tone/consent pattern:** `frontend/src/components/Hub/FunnelRiskPanel.tsx` (full file) — existing public-internet risk gate; CSS expand/collapse transition guarded by `prefers-reduced-motion`.

---

### `frontend/src/components/Hub/SessionShareModal.tsx` — hold-gate wiring (component/hook)

**Analog:** existing `funnelOn`/`warmingUp`/`publicReadCode` state wiring + `SetSessionFunnel`/`IssueCapabilities` RPC calls in the same file (Phase 170 precedent) — add sibling state for `writeURL`/`writeCode`/`expiresAt`/`used`, and wire hold-completion (`pointerdown`/`pointerup` timer) to the new `SetSessionFunnelWrite` RPC.

---

### `frontend/src/components/Hub/SessionCard.tsx` — `FULL ACCESS` badge (component, CRUD/render)

**Analog:** the existing `.hub-internet-badge` render block, verified live at lines 535-543:
```tsx
{/* Phase 166 / FUI-03 — internet exposure badge.
    COLORBLIND-SAFE: GlobeAltIcon shape + "INTERNET" text carry state; color is reinforcement only.
    Dark hex #43ddb2 / light hex #0d7a5c — verify at source, NOT by eye (user is colorblind). */}
{session.funnelActive && (
  <span className="hub-internet-badge">
    <GlobeAltIcon className="hub-internet-badge__icon" aria-hidden="true" />
    <span className="hub-internet-badge__label">INTERNET</span>
  </span>
)}
```

**New sibling block** (gated on new `session.funnelWriteActive` flag, distinct class/icon/label per D-09):
```tsx
{session.funnelWriteActive && (
  <span className="hub-fullaccess-badge">
    <LockOpenIcon className="hub-fullaccess-badge__icon" aria-hidden="true" />
    <span className="hub-fullaccess-badge__label">FULL ACCESS</span>
  </span>
)}
```
Same surface repeats in `TabBar.tsx` (`.tab__internet-icon` equivalent) per D-10 — mirror this exact same-file pattern there too.

---

### `frontend/src/style.css` — `.hub-fullaccess-badge*` (config)

**Analog:** `.hub-internet-badge*` block, verified live at lines 4707-4708 (dark tokens), 4766-4767 (light tokens), 7181-7208 (rules):
```css
--hub-internet-badge-bg: rgba(67, 221, 178, 0.15);   /* dark, line 4707 */
--hub-internet-badge-text: #43ddb2;                   /* dark, line 4708 */
--hub-internet-badge-bg: rgba(13, 122, 92, 0.13);     /* light, line 4766 */
--hub-internet-badge-text: #0d7a5c;                   /* light, line 4767 */

.hub-internet-badge { /* line 7181 */
  ...
  background: var(--hub-internet-badge-bg);
}
.hub-internet-badge__icon { color: var(--hub-internet-badge-text); }
.hub-internet-badge__label { color: var(--hub-internet-badge-text); }
```

**New sibling rules** — same structural pattern, new token names (`--hub-fullaccess-badge-bg`/`-text`, both dark+light variants), notched geometry via `clip-path` instead of the pill's `border-radius` (D-09 — shape, not just color, must differ):
```css
.hub-fullaccess-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--hub-fullaccess-badge-bg);
  padding: 3px 10px 3px 8px;
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
Exact `clip-path` polygon and color values are `[ASSUMED]`/Claude's Discretion (CONTEXT.md) — verify grayscale-distinctness from the pill at the source/DOM level (user is colorblind; do not verify by eye).

## Shared Patterns

### Single teardown chokepoint, but split by blast radius
**Source:** `internal/daemon/api.go:1800-1819` (`disableFunnelForSession`)
**Apply to:** `handleSetSessionFunnelWrite`'s disable path, `disableFunnelForSession` (append one new call), and any future RW-teardown trigger.
Never reuse the full `disableFunnelForSession` for an RW-only disable — it ref-counts down `funnelSessions` and can call `ws.DisableFunnel`, killing the read share too. Use `revokeFunnelWriteLocked` as the shared narrow sub-teardown, called from both the narrow path and appended inside the full chokepoint.

### Grant-scoped revocation, never `ClearGrants`
**Source:** `internal/webserver/server.go:303-330`
**Apply to:** All RW-gate revocation call sites.
`ClearGrants(sessionID)` wipes every grant (tailnet read+write included). Use the new `RemoveGrant(sessionID, grantID)` exclusively for RW-gate teardown.

### Colorblind-safe state encoding (icon + text + shape, never color alone)
**Source:** `frontend/src/style.css:7181-7208` + `SessionCard.tsx:535-543` comments
**Apply to:** `.hub-fullaccess-badge`, hold-progress bar (D-07), any new RW-state UI.
Existing badge pattern already documents this as a hard project rule with hex values called out in comments for exact verification — replicate the comment-with-hex-values documentation convention, not just the CSS.

### `SessionInfo` boolean flags never use `omitempty`
**Source:** `internal/daemon/types.go` (`FunnelActive` field)
**Apply to:** New `FunnelWriteActive` field — false must serialize so the frontend poller can detect a flip from true→false (RW share ended).

### Server-side clamps are independent of client-side dropdowns
**Source:** RESEARCH Pitfall 6 (no direct code precedent — this is a new invariant for this phase)
**Apply to:** `handleSetSessionFunnelWrite`'s `ExpiresIn` handling — clamp `<=0 || >3600` to exactly `3600`, unconditionally, regardless of what the frontend dropdown sends. Do NOT copy `handleSetSessionFunnel`'s existing `0 == unbounded` semantics for this handler.

## No Analog Found

None — every file in scope has a direct, verified same-repo analog (Phase 165/166/170 precedent). This phase is explicitly "add a sibling with a different lifecycle parameter" per RESEARCH.md's key insight; no net-new architectural mechanism is required except the WS-relay-integration test noted above (which has a well-defined test-writing analog in `TestFunnelTeardown_AllTriggers` even though its target code path — `handleWSSRelay` — is new to have a dedicated pre-gate-rejection test).

## Metadata

**Analog search scope:** `internal/capability/`, `internal/webserver/`, `internal/daemon/`, `app.go`, `frontend/src/components/`, `frontend/src/components/Hub/`, `frontend/src/style.css` — all scoped by RESEARCH.md's own file-by-file verified reads (Phase 171 researcher already performed exhaustive live-codebase verification with exact line numbers for every claim in this document).
**Files scanned:** 15 target files against ~10 analog source files, all directly read/grep-verified this session or in RESEARCH.md's cited primary sources.
**Pattern extraction date:** 2026-07-07
