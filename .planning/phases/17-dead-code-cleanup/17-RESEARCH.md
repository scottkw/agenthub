# Phase 17: Dead Code Cleanup - Research

**Researched:** 2026-03-20
**Domain:** Go dead code elimination, React/TypeScript dead binding removal
**Confidence:** HIGH

## Summary

Phase 17 deletes code that existed solely to support three removed features: (1) generic VPN interface selection, (2) auth middleware and token infrastructure, and (3) Settings UI for password, tokens, and VPN interface picker. All three features were stripped in Phases 15–16 at the functional layer; the remaining artifacts are structural scaffolding that compiles but serves no purpose.

The codebase currently builds cleanly (`go build ./...` passes) and all tests pass (Go: 5 packages OK, frontend: 84 tests OK). This phase is pure deletion — no new logic, no replacement code. The risk profile is low: Go's compiler will immediately flag any deletion that breaks a dependency chain, and the existing test suite provides a regression net.

**Primary recommendation:** Delete files and symbols in dependency order (deepest first), run `go build ./...` and both test suites after each wave to confirm nothing breaks.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CLEAN-01 | Generic VPN interface binding code is removed | `internal/webserver/network.go` + `network_test.go` identified; `app.go:GetNetworkInterfaces` + `app_test.go:TestGetNetworkInterfaces` identified; Wails binding stubs in both binding sets identified |
| CLEAN-02 | Auth middleware, token generation, and related backend routes are removed | Confirmed absent from server.go routes; test-only references in server_test.go (TestLoginRouteNotRegistered, TestTokenRouteNotRegistered) confirm routes are gone but tests document the absence — these tests may be kept or removed per planner judgment |
| CLEAN-03 | Settings UI for password, tokens, and VPN interface selection is removed | SettingsPanel.tsx has no password/token UI (already done in Phase 16); `GetNetworkInterfaces` binding in App.js/App.d.ts not called from any component — dead export; both Wails binding files need cleanup |
</phase_requirements>

## What Actually Needs Deleting

This section is the core research finding. Each item is a verified dead artifact.

### Backend (Go)

#### CLEAN-01: Generic VPN interface binding code

**File: `internal/webserver/network.go`** — DELETE ENTIRELY
- `NetworkInterface` struct (lines 7-13)
- `tailscaleCIDR` var and `init()` (lines 16-20)
- `IsTailscaleIP()` function (lines 22-25)
- `ListInterfaces()` function (lines 27-76)
- The entire file exists only for the VPN interface picker. Nothing in the production code path uses these after Phase 15 moved to `local.Client` direct IP query.

**File: `internal/webserver/network_test.go`** — DELETE ENTIRELY
- Tests for `IsTailscaleIP`, `ListInterfaces`, `NetworkInterface` struct, `TestTailscaleIPDetectedInList`
- These tests exclusively cover code in network.go

**File: `app.go`** — DELETE `GetNetworkInterfaces` method (lines 299-307)
```go
// DELETE THIS BLOCK:
func (a *App) GetNetworkInterfaces() []webserver.NetworkInterface {
    ifaces, err := webserver.ListInterfaces()
    if err != nil {
        return []webserver.NetworkInterface{}
    }
    return ifaces
}
```

**File: `app_test.go`** — DELETE `TestGetNetworkInterfaces` test (lines 164-172)
```go
// DELETE THIS BLOCK:
func TestGetNetworkInterfaces(t *testing.T) {
    ...
}
```
Also remove `GetNetworkInterfaces` from the mock/test imports if referenced.

**Import cleanup in `app.go`**: After removing `GetNetworkInterfaces`, the `webserver.NetworkInterface` type reference is gone. Check if `webserver` package import is still needed — it is still needed for `webserver.TailscaleHealth`, `webserver.CheckHealth`, `webserver.NewWebServer`, `webserver.Config`. The import stays; only the method is deleted.

#### CLEAN-01: Dependency verification — `IsTailscaleIP` usage
`IsTailscaleIP` is currently used inside `network.go`'s `ListInterfaces` (line 70) and in `network_test.go`. After deleting both files, no remaining code references it. Confirmed by grep: no other Go file imports or calls `IsTailscaleIP` or `ListInterfaces`.

#### CLEAN-02: Auth middleware / token routes

The backend routes (`/login`, `/api/sessions/{id}/token`) were already removed in Phase 16. The remaining auth-related Go code is:

**File: `internal/webserver/server_test.go`** — PARTIAL: three test functions document absence of auth
- `TestLoginRouteNotRegistered` (line 380): verifies `/login` returns non-200
- `TestTokenRouteNotRegistered` (line 395): verifies `/api/sessions/{id}/token` returns non-200
- `TestSessionAccessWithoutAuth` (line 408): verifies session accessible without auth

These tests are regression guards — they verify auth routes are NOT present. The planner must decide: keep them as guards (preferred, adds no code), or delete them as unnecessary if the phase goal is "absent from source". Research recommendation: **keep** — they are zero-code assertions that protect against accidental re-introduction.

**File: `app_test.go`** — `TestStartWebServerNoPasswordRequired` (lines 183-200): verifies no password error. Same rationale — it's a guard, not dead code. **Keep**.

No auth middleware Go source code was found in the codebase. Phase 16 already removed all backend auth logic. CLEAN-02 backend work is limited to verifying nothing was missed.

### Frontend (TypeScript/React)

#### CLEAN-01: VPN interface binding stubs

**File: `frontend/src/wailsjs/go/main/App.js`** — REMOVE line 16:
```js
export const GetNetworkInterfaces = ()  => Call('main.App.GetNetworkInterfaces', [])
```
No component imports or calls `GetNetworkInterfaces`. Confirmed by grep: only the mock in `SettingsPanel.test.tsx` references it.

**File: `frontend/src/wailsjs/go/main/App.d.ts`** — REMOVE:
- `NetworkInterface` interface (lines 17-21)
- `GetNetworkInterfaces` function declaration (line 32)

**File: `frontend/src/wailsjs/wailsjs/go/main/App.js`** — REMOVE `GetNetworkInterfaces` function (lines 17-18)

**File: `frontend/src/wailsjs/wailsjs/go/main/App.d.ts`** — REMOVE `GetNetworkInterfaces` declaration (line 13)

**File: `frontend/src/wailsjs/wailsjs/go/models.ts`** — REMOVE `webserver.NetworkInterface` class (lines 49-64)
After removal, the `webserver` namespace has only `TailscaleHealth` — keep the namespace, just remove the `NetworkInterface` class.

**File: `frontend/src/components/__tests__/SettingsPanel.test.tsx`** — REMOVE mock entry (line 9):
```ts
GetNetworkInterfaces: vi.fn().mockResolvedValue([]),
```
This mock is in the `vi.mock()` factory for the App module — removing the unused mock entry is the safe approach. Confirm `GetNetworkInterfaces` is not referenced elsewhere in the test file (it is not — the mock was defensive).

#### CLEAN-03: Settings UI for password/tokens/VPN interface

The SettingsPanel.tsx was already cleaned in Phase 16 — no password field, no token UI, no VPN interface picker currently exists in the component. The SettingsPanel.test.tsx already asserts this (`Security tab does not exist`, `no password input rendered`).

**Remaining CLEAN-03 work** is the dead binding exports for `GetNetworkInterfaces` (covered above under CLEAN-01) — that was the interface picker's backing API call.

### Summary Table of Deletions

| File | Action | Lines / Symbols |
|------|--------|-----------------|
| `internal/webserver/network.go` | DELETE FILE | Entire file (77 lines) |
| `internal/webserver/network_test.go` | DELETE FILE | Entire file (121 lines) |
| `app.go` | REMOVE METHOD | `GetNetworkInterfaces` (~9 lines) |
| `app_test.go` | REMOVE TEST | `TestGetNetworkInterfaces` (~9 lines) |
| `frontend/src/wailsjs/go/main/App.js` | REMOVE LINE | `GetNetworkInterfaces` export |
| `frontend/src/wailsjs/go/main/App.d.ts` | REMOVE BLOCK | `NetworkInterface` interface + `GetNetworkInterfaces` declaration |
| `frontend/src/wailsjs/wailsjs/go/main/App.js` | REMOVE FUNCTION | `GetNetworkInterfaces` |
| `frontend/src/wailsjs/wailsjs/go/main/App.d.ts` | REMOVE LINE | `GetNetworkInterfaces` declaration |
| `frontend/src/wailsjs/wailsjs/go/models.ts` | REMOVE CLASS | `webserver.NetworkInterface` class |
| `frontend/src/components/__tests__/SettingsPanel.test.tsx` | REMOVE LINE | `GetNetworkInterfaces` mock entry |

## Architecture Patterns

### Deletion Order (Dependency-Safe)

```
Wave 1: Backend leaf (no callers after deletion)
  internal/webserver/network.go        — DELETE FILE
  internal/webserver/network_test.go   — DELETE FILE

Wave 2: Backend caller cleanup
  app.go                               — REMOVE GetNetworkInterfaces method
  app_test.go                          — REMOVE TestGetNetworkInterfaces test

Wave 3: Frontend binding stubs (two sets — manual and wails-generated)
  frontend/src/wailsjs/go/main/App.js  — REMOVE line
  frontend/src/wailsjs/go/main/App.d.ts — REMOVE NetworkInterface + GetNetworkInterfaces
  frontend/src/wailsjs/wailsjs/go/main/App.js — REMOVE function
  frontend/src/wailsjs/wailsjs/go/main/App.d.ts — REMOVE declaration
  frontend/src/wailsjs/wailsjs/go/models.ts — REMOVE NetworkInterface class
  frontend/src/components/__tests__/SettingsPanel.test.tsx — REMOVE mock entry

Wave 4: Verify
  go build ./...
  go test ./...
  pnpm --prefix frontend test -- --run
```

### Go Deletion Safety Pattern

When deleting a Go file that exports types used elsewhere:
1. Delete the file
2. Run `go build ./...` immediately
3. Compiler will list every broken reference — fix each one
4. Never guess at transitive usage — let the compiler find it

For this phase, `network.go` exports `NetworkInterface`, `IsTailscaleIP`, and `ListInterfaces`. After verifying no remaining Go code references these symbols (confirmed by grep), file deletion is safe. The only references are `app.go:GetNetworkInterfaces` (being deleted) and test files (being deleted).

### Import Hygiene After Deletion

After removing `GetNetworkInterfaces` from `app.go`, verify the `webserver` package import is still needed. It is — `webserver.TailscaleHealth`, `webserver.CheckHealth`, `webserver.Config`, and `webserver.NewWebServer` all remain in `app.go`. The import line stays unchanged.

### Wails Binding Files Are Manual Stubs (Not Auto-Generated Here)

The project has TWO sets of Wails binding files:
- `frontend/src/wailsjs/go/main/App.js` + `App.d.ts` — hand-maintained stubs (see comment: "AUTO-GENERATED by Wails — DO NOT edit manually" is a comment convention, but in this project they are manually curated)
- `frontend/src/wailsjs/wailsjs/go/main/App.js` + `App.d.ts` + `../models.ts` — the actual Wails-generated bindings (comment: "Cynhyrchwyd y ffeil hon yn awtomatig")

Both sets must be updated. Wails regenerates the second set on `wails dev`/`wails build`, so those changes may be overwritten on next build. However, the planner should include them — correct source state matters for tests and IDE type-checking even if they'll be regenerated later.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|-------------|
| Finding all dead references | Manual grep | `go build ./...` compiler errors after file deletion |
| Verifying no auth routes remain | Custom scanner | Existing tests `TestLoginRouteNotRegistered`, `TestTokenRouteNotRegistered` |
| Frontend dead export detection | ESLint unused-exports | Remove and run tests — vitest will catch missing mocks |

## Common Pitfalls

### Pitfall 1: Forgetting the Second Wails Binding Set
**What goes wrong:** Developer removes `GetNetworkInterfaces` from `frontend/src/wailsjs/go/main/App.js` but misses the parallel set at `frontend/src/wailsjs/wailsjs/go/main/App.js`. TypeScript compilation passes but Wails runtime may behave inconsistently.
**How to avoid:** Both paths are documented in the deletion table above. Delete from both.

### Pitfall 2: Removing `webserver` Import After `GetNetworkInterfaces` Deletion
**What goes wrong:** Developer assumes the `webserver` import in `app.go` is dead after removing `GetNetworkInterfaces`, and deletes the import — breaking all other `webserver.*` usage.
**How to avoid:** `go build ./...` catches it immediately. The import must remain.

### Pitfall 3: Breaking SettingsPanel Tests by Removing Mock
**What goes wrong:** Removing `GetNetworkInterfaces` from the mock factory in SettingsPanel.test.tsx without checking if the mock factory has side effects on other exported functions.
**How to avoid:** The mock uses `vi.mock('../../wailsjs/go/main/App', () => ({...}))` — only listed exports are mocked. Removing `GetNetworkInterfaces` from the factory is safe if the component no longer imports it. Confirm by checking SettingsPanel.tsx imports (confirmed: it does not import `GetNetworkInterfaces`).

### Pitfall 4: Treating Auth-Absence Tests as Dead Code
**What goes wrong:** Developer deletes `TestLoginRouteNotRegistered` and `TestTokenRouteNotRegistered` assuming they cover removed code — but they are regression guards for routes that should STAY removed.
**How to avoid:** Do not delete these tests. They have zero implementation code; they only assert HTTP responses on non-routes.

### Pitfall 5: Deleting `IsTailscaleIP` Before Checking `network_test.go`
**What goes wrong:** If `network.go` is deleted but `network_test.go` is kept, compilation fails because `network_test.go` references `webserver.IsTailscaleIP`, `webserver.ListInterfaces`, etc.
**How to avoid:** Delete both files together in Wave 1.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test` (stdlib) |
| Framework (Frontend) | vitest v4.1.0 |
| Config file | `frontend/vitest.config.ts` (inferred from pnpm test script) |
| Quick run (Go) | `go test ./...` |
| Quick run (Frontend) | `pnpm --prefix /Users/ken/dev/agenthub/frontend test -- --run` |
| Full suite | Both above sequentially |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | Coverage |
|--------|----------|-----------|-------------------|---------|
| CLEAN-01 | `network.go` file deleted, `go build ./...` passes | build verification | `go build ./...` | Run after Wave 1+2 |
| CLEAN-01 | `GetNetworkInterfaces` absent from compiled binary (source check) | source inspection | `grep -r GetNetworkInterfaces /Users/ken/dev/agenthub --include="*.go"` after deletion | Post-deletion |
| CLEAN-02 | Auth routes absent from server.go | existing test | `go test ./internal/webserver/... -run TestLoginRouteNotRegistered\|TestTokenRouteNotRegistered` | Already green |
| CLEAN-03 | No password/token/VPN UI in SettingsPanel | existing test | `pnpm --prefix /Users/ken/dev/agenthub/frontend test -- --run --reporter=verbose` | Already green |
| CLEAN-01/03 | Frontend bindings clean, tests pass | existing tests | `pnpm --prefix /Users/ken/dev/agenthub/frontend test -- --run` | Run after Wave 3 |

### Sampling Rate
- **Per wave:** `go build ./... && go test ./...`
- **Phase gate:** Full suite green (both Go and frontend) before `/gsd:verify-work`

### Wave 0 Gaps
None — existing test infrastructure covers all phase requirements. No new test files needed.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Generic VPN interface enumeration for binding | Tailscale IP from `local.Client` directly | Phase 15 | `network.go` became dead code |
| App-layer auth (password, tokens, middleware) | No auth — Tailscale network-level control | Phase 16 | Auth routes/routes/UI removed |
| `GetNetworkInterfaces` Wails binding exported | No binding needed | Phase 16 | Binding is dead export |

## Open Questions

1. **Should `TestLoginRouteNotRegistered` and `TestTokenRouteNotRegistered` be kept or deleted?**
   - What we know: They test absence of routes, not presence. Zero implementation code. Currently passing.
   - What's unclear: Project policy on "absence tests" — are they considered documentation or noise?
   - Recommendation: Keep them. They cost nothing and protect against accidental route re-introduction.

2. **Are the Wails-generated bindings under `wailsjs/wailsjs/` overwritten on next `wails build`?**
   - What we know: The comment says "DO NOT EDIT" and they are auto-generated by `wails dev/build`. The `NetworkInterface` class in `models.ts` will be regenerated from the Go struct if it still exists.
   - Implication: Deleting `network.go` removes the Go struct → next `wails build` will regenerate `models.ts` without `NetworkInterface`. The manual deletion in Wave 3 ensures correctness for tests/type-checking without requiring a Wails build.
   - Recommendation: Delete from both binding sets; the Wails regeneration will confirm correctness on next build.

## Sources

### Primary (HIGH confidence)
- Source inspection of all relevant files — direct read of current codebase state
- `go build ./...` — confirmed clean build baseline
- `go test ./...` — confirmed all 5 packages passing
- `pnpm --prefix frontend test -- --run` — confirmed 84 frontend tests passing

### Secondary (N/A)
No external library research needed — this is pure deletion of internal project code.

## Metadata

**Confidence breakdown:**
- Dead code identification: HIGH — direct source inspection of every file involved
- Deletion safety: HIGH — `go build` provides immediate verification; compiler errors are deterministic
- Test impact: HIGH — confirmed by reading every test file that touches deleted symbols
- Frontend binding impact: HIGH — confirmed by grepping all component imports

**Research date:** 2026-03-20
**Valid until:** Indefinite — source code does not change until the phase executes
