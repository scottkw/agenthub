# Phase 61: SERVE-02 Frontend Integration Fix - Research

**Researched:** 2026-04-09
**Domain:** Wails Go-TypeScript bindings, React state seeding, App.tsx webEnabled flow
**Confidence:** HIGH

## Summary

Phase 59 successfully implemented all backend plumbing for SERVE-02: `daemon/types.go` has `WebEnabled bool`, `handleListSessions` enriches it from the web server's `IsSessionEnabled`, and `handleCreateSession` auto-enables new sessions. The Wails binding `app.go:ListSessions()` returns the Go-local `SessionInfo` (not `daemon.SessionInfo`), which does NOT have the `WebEnabled` field — so the field never reaches the frontend even though the daemon sends it.

The quick task `260409-vop` rewrote `App.tsx` to remove the `HealthModal` and wire Settings as a tab. During that rewrite, two blocks of SERVE-02 frontend code were deleted from `App.tsx`:

1. **`init()` restore path** — the block that iterated `sessions`, read `s.webEnabled`, and called `setWebEnabled` / `setSessionURLs`.
2. **`createTab` new-session path** — the block that, after `CreateSession` returned, called `setWebEnabled` / `setSessionURLs` when `webServerRunning` was true.

Additionally the `webServerRunning` dependency was dropped from `createTab`'s `useCallback` dependency array.

Three files need surgical edits:
- `app.go` — add `WebEnabled` field to the local `SessionInfo` struct and copy it in `ListSessions()`
- `frontend/src/wailsjs/go/main/App.d.ts` — add `webEnabled: boolean` to the `SessionInfo` interface
- `frontend/src/App.tsx` — restore both deleted seeding blocks and restore the `webServerRunning` dependency

There are also 11 failing frontend tests: 8 in `App.test.tsx` and 3 in `App.nav.test.tsx`. These are stale assertions that check for code removed by `260409-vop` (HealthModal, SettingsPanel modal, `env.platform`, `setShowSettings`). These are tech debt for Phase 62, but the research flags them here so the planner is aware.

**Primary recommendation:** Three-file surgical fix. No new libraries. No new API endpoints. All backend plumbing is already correct — this is purely a frontend wiring gap.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib | 1.26.2 | `app.go` Go struct extension | Already the project language |
| TypeScript | 5.x (project) | `App.d.ts` type stub update | Hand-maintained Wails type stub pattern already in use |
| React 18 / useState | 18.x | `App.tsx` state seeding | Already in use throughout |

No new libraries required.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Adding `WebEnabled` to `app.go` local `SessionInfo` | Having `app.go:ListSessions()` return `[]daemon.SessionInfo` directly | Returning `daemon.SessionInfo` directly would expose all daemon package types to Wails bindings; the existing pattern uses a separate local type to control the Wails API surface |

## Architecture Patterns

### The Three-Layer Type Chain

The Wails binding chain for session data is:

```
daemon/types.go:SessionInfo      (source of truth, has WebEnabled)
         ↓ client.ListSessions() returns this
app.go:DaemonClient.ListSessions()  returns []daemon.SessionInfo
         ↓ app.go:ListSessions() maps to local type
app.go:SessionInfo               (MISSING WebEnabled — root cause)
         ↓ Wails codegen picks up app.go types
frontend/src/wailsjs/go/main/App.d.ts:SessionInfo  (MISSING webEnabled)
         ↓ TypeScript import
frontend/src/App.tsx             (cannot read s.webEnabled — field absent)
```

The fix must be applied at **all three layers** in sequence.

### Pattern 1: app.go local SessionInfo extension
**What:** Add `WebEnabled bool` to the local `SessionInfo` struct in `app.go` and copy it in `ListSessions()`.
**Source:** [VERIFIED: reading app.go lines 31-38 and 223-243]

```go
// app.go — current (broken)
type SessionInfo struct {
    ID        string `json:"id"`
    CLI       string `json:"cli"`
    Name      string `json:"name"`
    State     string `json:"state"`
    CreatedAt string `json:"createdAt"`
    Hostname  string `json:"hostname"`
    // WebEnabled field missing here
}

// app.go:ListSessions() current (broken)
result[i] = SessionInfo{
    ID:        s.ID,
    CLI:       s.CLI,
    Name:      s.Name,
    State:     s.State,
    CreatedAt: s.CreatedAt,
    Hostname:  s.Hostname,
    // WebEnabled not mapped here
}
```

**After fix:**
```go
type SessionInfo struct {
    ID         string `json:"id"`
    CLI        string `json:"cli"`
    Name       string `json:"name"`
    State      string `json:"state"`
    CreatedAt  string `json:"createdAt"`
    Hostname   string `json:"hostname"`
    WebEnabled bool   `json:"webEnabled"`
}

result[i] = SessionInfo{
    ID:         s.ID,
    CLI:        s.CLI,
    Name:       s.Name,
    State:      s.State,
    CreatedAt:  s.CreatedAt,
    Hostname:   s.Hostname,
    WebEnabled: s.WebEnabled,
}
```

### Pattern 2: App.d.ts hand-maintained stub update
**What:** The `App.d.ts` file is noted as "AUTO-GENERATED by Wails" but the project treats it as hand-maintained (the Phase 59 summary explicitly says "Add webEnabled to SessionInfo TypeScript type in hand-maintained App.d.ts stub"). Add `webEnabled: boolean` to the `SessionInfo` interface.
**Source:** [VERIFIED: reading App.d.ts lines 4-11]

```typescript
export interface SessionInfo {
  id: string
  cli: string
  name: string
  state: string
  createdAt: string
  hostname: string
  webEnabled: boolean  // ADD THIS
}
```

### Pattern 3: App.tsx init() restore path
**What:** After restoring tabs from `ListSessions()`, seed `webEnabled` state from `s.webEnabled` field for sessions that are web-enabled.
**Source:** [VERIFIED: reading Phase 59 SUMMARY.md — this code was written in commit 817ebbd, then deleted by quick task f5f8143]

The deleted block goes after the `sessions.forEach((s) => { GetSessionStatus... })` loop, inside `if (sessions.length > 0)`:

```typescript
// Seed webEnabled state from daemon's SessionInfo.webEnabled field (SERVE-02 restore).
if (running) {
  const enabledMap: Record<string, boolean> = {}
  const urlMap: Record<string, string> = {}
  let serverURL: string | undefined
  try {
    serverURL = await GetWebServerURL()
  } catch (_) { /* ignore */ }

  sessions.forEach((s) => {
    if (s.webEnabled) {
      enabledMap[s.id] = true
      if (serverURL) {
        urlMap[s.id] = `${serverURL}/sessions/${s.id}`
      }
    }
  })
  if (Object.keys(enabledMap).length > 0) {
    setWebEnabled(enabledMap)
    setSessionURLs(urlMap)
  }
}
```

Note: The `running` variable is in scope here — it comes from `const [port, clis, sessions, running, health] = await Promise.all(...)`.

### Pattern 4: App.tsx createTab restore path
**What:** After `CreateSession` returns successfully and the tab is added, if the web server is running, seed `webEnabled` for the new session.
**Source:** [VERIFIED: reading Phase 59 SUMMARY.md — this code was written in commit 817ebbd, then deleted by quick task f5f8143]

The deleted block goes after `setActiveId(sessionId)` inside the `try` block:

```typescript
// Auto-seed webEnabled state for new sessions when web server is running (SERVE-02).
if (webServerRunning) {
  setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))
  try {
    const url = await GetWebServerURL()
    if (url) {
      setSessionURLs((prev) => ({ ...prev, [sessionId]: `${url}/sessions/${sessionId}` }))
    }
  } catch (_) { /* URL fetch failure is non-fatal */ }
}
```

And the `useCallback` dependency array must include `webServerRunning`:
```typescript
}, [tabCounter, webServerRunning])
```

### Anti-Patterns to Avoid
- **Calling `ToggleWebServing` in `createTab`:** The daemon already auto-enables the session in `handleCreateSession`. The frontend only needs to mirror that state — not redundantly call the toggle. Calling `ToggleWebServing` again would be a no-op (session already enabled) but is semantically wrong.
- **Using `(s as any).webEnabled`:** The Phase 59 plan suggested this as a fallback. Since the fix adds `webEnabled` to `App.d.ts` properly, the cast is unnecessary. Use `s.webEnabled` directly after the type is updated.
- **Changing `daemon.SessionInfo` again:** The daemon type is correct as-is. Only `app.go`'s local `SessionInfo` needs the new field.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Querying web-enabled state | New `/webserver/sessions` endpoint | `SessionInfo.webEnabled` field already in daemon response | Field is already implemented in `daemon/types.go` and populated in `handleListSessions` |
| URL construction | Separate URL fetch per session | Existing `GetWebServerURL()` + `/sessions/${id}` pattern | Same pattern used in `handleToggleWeb` throughout App.tsx |

## Common Pitfalls

### Pitfall 1: app.go's local SessionInfo is distinct from daemon.SessionInfo
**What goes wrong:** Developer adds `WebEnabled` to `daemon/types.go` (already done in Phase 59) but forgets that `app.go` defines its own `SessionInfo` struct (lines 31-38) that Wails uses for TypeScript generation.
**Why it happens:** There are two `SessionInfo` types in the codebase: `daemon.SessionInfo` (the HTTP DTO) and `main.SessionInfo` (the Wails-bound type). The conversion happens in `app.go:ListSessions()` lines 231-242.
**How to avoid:** Edit `app.go` local `SessionInfo` AND the mapping in `ListSessions()`. Both are required.
**Warning signs:** TypeScript type check passes but `s.webEnabled` is always `undefined` at runtime.

### Pitfall 2: `running` variable scope in init()
**What goes wrong:** The `running` variable from `Promise.all` is in the correct scope for the restore block, but the existing polling block (`if (!running)`) already uses it. The webEnabled seeding block must go inside `if (sessions.length > 0)`, after the status seeding `forEach`, to be in scope and sequential.
**Why it happens:** The init() function has a complex control flow with async/await. The `running` value must be read before any async calls that might change `webServerRunning` state.
**How to avoid:** Place the webEnabled restore block immediately after the `sessions.forEach` status seeding loop, still inside `if (sessions.length > 0)`, still inside the outer `try` block.

### Pitfall 3: createTab dependency array
**What goes wrong:** The `useCallback` closure for `createTab` captures `webServerRunning` but the dependency array does not include it. When `webServerRunning` changes to `true` (after web server auto-starts), the stale closure still reads `false`.
**Why it happens:** React `useCallback` memoization — the function is not recreated unless its deps change.
**How to avoid:** Change `}, [tabCounter])` to `}, [tabCounter, webServerRunning])`.

### Pitfall 4: Failing tests in App.test.tsx (8 failures) and App.nav.test.tsx (3 failures)
**What goes wrong:** The current test suite has 11 failing tests due to stale assertions from before `260409-vop`. These check for `HealthModal`, `SettingsPanel`, `env.platform`, `setShowSettings`, `Environment()` — all removed by the quick task.
**Why it happens:** The quick task rewrote App.tsx but did not update the tests. Phase 62 is the planned cleanup.
**How to avoid:** Phase 61 should NOT fix these tests — they are scoped to Phase 62. Phase 61 should only fix SERVE-02 wiring and can add a new test assertion that `s.webEnabled` is used in init. Do not accidentally fix or break Phase 62 tests.

## Code Examples

### Current app.go SessionInfo (broken — missing WebEnabled)
```go
// Source: app.go lines 31-38 [VERIFIED]
type SessionInfo struct {
    ID        string `json:"id"`
    CLI       string `json:"cli"`
    Name      string `json:"name"`
    State     string `json:"state"`
    CreatedAt string `json:"createdAt"`
    Hostname  string `json:"hostname"`
}
```

### Current app.go ListSessions mapping (broken — does not copy WebEnabled)
```go
// Source: app.go lines 231-242 [VERIFIED]
result[i] = SessionInfo{
    ID:        s.ID,
    CLI:       s.CLI,
    Name:      s.Name,
    State:     s.State,
    CreatedAt: s.CreatedAt,
    Hostname:  s.Hostname,
}
```

### Current App.d.ts SessionInfo (broken — missing webEnabled)
```typescript
// Source: frontend/src/wailsjs/go/main/App.d.ts lines 4-11 [VERIFIED]
export interface SessionInfo {
  id: string
  cli: string
  name: string
  state: string
  createdAt: string
  hostname: string
}
```

### Current App.tsx createTab (broken — webEnabled seeding deleted)
```typescript
// Source: frontend/src/App.tsx lines 216-228 [VERIFIED]
// After CreateSession succeeds: setTabs, setActiveId — then nothing
// Missing: webEnabled seeding block and webServerRunning dependency
}, [tabCounter])
```

### Daemon side is CORRECT (no changes needed)
```go
// Source: internal/daemon/types.go [VERIFIED — Phase 59 complete]
type SessionInfo struct {
    ID         string `json:"id"`
    CLI        string `json:"cli"`
    Name       string `json:"name"`
    State      string `json:"state"`
    CreatedAt  string `json:"createdAt"`
    Hostname   string `json:"hostname"`
    WebEnabled bool   `json:"webEnabled"`
}

// Source: internal/daemon/api.go handleListSessions [VERIFIED — Phase 59 complete]
// Enriches sessions[i].WebEnabled from ws.IsSessionEnabled(sessions[i].ID)

// Source: internal/daemon/api.go handleCreateSession [VERIFIED — Phase 59 complete]
// Calls ws.EnableSession(id) when web server is running
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| App.tsx init used `(s as any).webEnabled` cast | Should use `s.webEnabled` typed field | Phase 61 adds type | TypeScript type safety for webEnabled field |
| Frontend seeded webEnabled in createTab and init | Deleted by quick task 260409-vop | 2026-04-09 | SERVE-02 broken: StatusBar always shows "WEB OFF" |
| app.go SessionInfo mirrors daemon.SessionInfo without WebEnabled | Will match after Phase 61 | Phase 61 | Enables correct Wails binding |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `pnpm vitest run` is the correct test runner command | Validation Architecture | Low — Phase 59 SUMMARY uses `pnpm test --run`; actual invocation is `pnpm vitest run` based on testing above |

**All other claims in this research were verified by direct source code reading.**

## Open Questions

1. **Should Phase 61 also fix the 11 failing frontend tests?**
   - What we know: Phase 62 is explicitly scoped for "fix stale tests" (App.test.tsx and App.nav.test.tsx). The 8 failures in App.test.tsx and 3 in App.nav.test.tsx are entirely due to deleted HealthModal/SettingsPanel code.
   - What's unclear: Whether the planner wants Phase 61 to produce a clean test suite or leave test fixes to Phase 62.
   - Recommendation: Phase 61 MUST NOT touch Phase 62 test fixes. Phase 61's plan should note the pre-existing 11 test failures as known state, not regression.

2. **Does the `retryInit` path in App.tsx also need the webEnabled seeding?**
   - What we know: `retryInit` (lines 401-446) also calls `ListSessions()` and restores tabs. The same webEnabled seeding block was not added there in Phase 59 (the SUMMARY only mentions `init()`).
   - What's unclear: Whether the retry path is exercised enough to matter.
   - Recommendation: Add the webEnabled seeding to `retryInit` as well — it's a 5-line addition and prevents a subtle inconsistency when the daemon retries after initial failure.

## Environment Availability

Step 2.6: SKIPPED — Phase 61 is a pure code edit with no external dependencies beyond Go toolchain and Node/pnpm, both verified present.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `testing` package + `go test` |
| Framework (frontend) | Vitest 4.1.0 |
| Config file (frontend) | `frontend/vite.config.ts` |
| Quick run command (Go) | `go build ./...` |
| Full suite command (Go) | `go test ./...` |
| Quick run command (frontend) | `cd frontend && pnpm vitest run 2>&1 \| grep -E "(passed\|failed)"` |
| Full suite command (frontend) | `cd frontend && pnpm vitest run` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SERVE-02 | app.go ListSessions maps WebEnabled field | source check / compile | `go build ./...` | Verify via `grep WebEnabled app.go` |
| SERVE-02 | App.d.ts SessionInfo includes webEnabled | TypeScript compile | `cd frontend && pnpm tsc --noEmit` | ✅ existing |
| SERVE-02 | createTab seeds webEnabled when webServerRunning | source check | `grep -c 'webEnabled.*webServerRunning\|webServerRunning.*webEnabled' frontend/src/App.tsx` | ❌ Wave 0 |
| SERVE-02 | init seeds webEnabled from s.webEnabled | source check | `grep 's.webEnabled' frontend/src/App.tsx` | ❌ Wave 0 |

**Pre-existing test failures (NOT regressions from Phase 61):**
- `App.test.tsx`: 8 failures (HealthModal, Environment, platform, setShowSettings references — Phase 62 scope)
- `App.nav.test.tsx`: 3 failures (SettingsPanel modal pattern — Phase 62 scope)
- Phase 61 plan should assert the failure count does not increase.

### Wave 0 Gaps
- No new test files needed — the existing test infrastructure covers this phase. The source-inspection pattern (raw string checks) used in `App.test.tsx` and `App.nav.test.tsx` is the appropriate approach for new SERVE-02 assertions.

## Security Domain

This phase makes no changes to authentication, authorization, or cryptographic code. The `webEnabled` field is a boolean flag read from daemon state — no new user inputs, no new data paths.

ASVS: Not applicable for this phase. Security enforcement applies to Phase 60 (local network TLS + auth), which is already complete.

## Sources

### Primary (HIGH confidence)
- `app.go` lines 31-38 (local SessionInfo struct — verified missing WebEnabled)
- `app.go` lines 223-243 (ListSessions mapping — verified WebEnabled not copied)
- `frontend/src/wailsjs/go/main/App.d.ts` lines 4-11 (TypeScript interface — verified missing webEnabled)
- `frontend/src/App.tsx` lines 201-228 (createTab — verified seeding deleted)
- `frontend/src/App.tsx` lines 84-158 (init — verified restore block deleted)
- `internal/daemon/types.go` (daemon SessionInfo — verified WebEnabled present)
- `internal/daemon/api.go` (handleListSessions enrichment — verified present)
- `internal/daemon/api.go` (handleCreateSession auto-enable — verified present)
- `.planning/phases/59-auto-serve-sessions/59-01-SUMMARY.md` (implementation record of what Phase 59 built and what commits contain the deleted code)
- `git diff 817ebbd f5f8143 -- frontend/src/App.tsx` (direct diff proving which lines were deleted)

### Secondary (MEDIUM confidence)
- `frontend/src/components/__tests__/App.test.tsx` (11 pre-existing failures identified — Phase 62 scope)

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Root cause: HIGH — verified by git diff between Phase 59 completion commit and quick task rewrite
- Fix prescription: HIGH — three files, three surgical edits, all verified against actual source
- Test impact: HIGH — failing tests are pre-existing and scoped to Phase 62, not regressions

**Research date:** 2026-04-09
**Valid until:** 2026-05-09 (internal codebase, no external API dependencies)

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SERVE-02 | New sessions have web serving enabled automatically when the web server is running | Root cause identified: three-layer type chain broken at (1) app.go local SessionInfo missing WebEnabled field, (2) App.d.ts interface missing webEnabled, (3) App.tsx createTab and init seeding blocks deleted by quick task 260409-vop. Fix: add field to app.go struct + mapping, add to App.d.ts, restore both deleted blocks in App.tsx. StatusBar will then show correct web toggle state. |
</phase_requirements>
