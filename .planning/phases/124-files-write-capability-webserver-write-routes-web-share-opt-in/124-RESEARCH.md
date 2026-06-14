# Phase 124: files.write Capability + Webserver Write Routes + Web-Share Opt-In - Research

**Researched:** 2026-06-14
**Domain:** Go HTTP capability middleware + CSRF, settings schema migration, React/Wails opt-in UI, lipgloss TUI parity
**Confidence:** HIGH (all findings read from live source in this repo; Go stdlib behavior verified against go1.26.4)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- `files.write` is OPT-IN for ALL tokens (owner and viewer) — no owner default-on. (Confirmed by commit 808767f "make files.write opt-in for all tokens".)
- CSRF protection reuses the Phase 88 `Origin`-check pattern: mismatched Origin → 403; absent Origin (desktop Wails fetch) passes vacuously.
- Permission checks MUST use the `HasPerm` whole-token comma-split helper — never `strings.Contains(perms, "files.write")`.
- Settings migrate `schemaVersion: 3 → 4` with `FilesWrite: false` default.

### Claude's Discretion
Remaining implementation details (middleware wiring, toggle component structure, migration mechanics) at Claude's discretion — use ROADMAP success criteria, Phase 123 patterns, and codebase conventions.

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CAP-01 | `PermFilesWrite = "files.write"` constant in `internal/capability/capability.go`; gated via `HasPerm` (not `strings.Contains`) | `HasPerm` exists at `capability.go:42`; mirror `PermFilesRead` const at `capability.go:31`. |
| CAP-02 | New `requireFilesWrite` middleware (separate from `requireCapability` AND `requireFilesRead`) gates all five webserver write routes | Mirror `requireFilesRead` (`capability_mw.go:102`); MUST be a third separate wrapper — see Architecture Pattern 2. |
| CAP-03 | `requireFilesWrite` adds CSRF `Origin` check for POST/PUT/PATCH/DELETE: reject when Origin present and ≠ FQDN; absent Origin passes vacuously | The existing `requireAllowedOrigin` (`origin_mw.go:31`) rejects absent Origin — CAP-03 needs the INVERSE on absent. New logic required — see Pitfall 1. |
| CAP-04 | `files.write` opt-in for every token; per-session "Enable file writes" toggle gates owner cap; viewers need further opt-in | Owner cap minted at `api.go:1049-1060` `issueCapabilitiesForSession`. `FilesRead` is GLOBAL today; `FilesWrite` must be PER-SESSION — see Open Question 1. |
| CAP-05 | Web-share grant UI exposes `files.write` opt-in toggle (default OFF), separate from `files.read`, with confirmation copy | `SessionSharePanel.tsx` "Full Access Link" row already renders the `writeURL`; opt-in gates whether that cap carries `files.write`. |
| CAP-06 | Home-dir write warning in GUI AND TUI when files.write enabled for a `$HOME`-cwd session | Compute server-side via `GetSessionWorkDir` vs `os.UserHomeDir` (mirror `sandbox.go:96` denylist). TUI line above `renderFilesStatusLine` (`files.go:283`). |
| CAP-07 | Webserver write routes mounted via `SetFilesHandlerProvider` pattern; cap middleware on webserver side only | `filesDispatch` closure at `server.go:494`; daemon routes are auth-less (`api.go:147`+). |
| CAP-08 | Settings `schemaVersion: 4` migration via `defaultSettings()` constructor-merge; per-field test; web-share opt-in persists default `false` | `CurrentSchemaVersion = 3` at `plugin_settings.go:8`; bump to 4. Migration test template: `engine_migration_test.go:206`. |
| CAP-09 | Cap-denied integration tests: viewer w/o `files.write` → 403 on all 5 endpoints; granted → 2xx. Static-grep `HasPerm` gate | Mirror `TestRequireCapability_UnchangedByPhase118` (`capability_test.go:575`) for the write gate. |
| CAP-10 | Remote daemon-proxy write routes `/api/files/remote/{sid}/{write,upload,delete,rename,mkdir}`; `proxyRemoteFiles` forwards `r.Body` for PUT/POST/PATCH | `remote_files.go:168` currently passes `nil` body — the bug. Routes registered at `api.go:161-164` (read only today). |

</phase_requirements>

## Summary

Phase 124 is a **brownfield Go + React/Wails + lipgloss-TUI** phase. It adds **zero external packages** — every primitive it needs already exists in the tree as a Phase 118/119 (`files.read`) precedent or a Phase 88 (Origin) precedent. The work is: (1) add one capability constant, (2) add one new middleware wrapper that composes `requireCapability` + a `HasPerm("files.write")` check + a CSRF Origin check, (3) mount it on five new webserver write routes that dispatch to the already-built `files.Handler.{Write,Upload,Delete,Rename,Mkdir}` from Phase 123, (4) thread a per-session `files.write` opt-in through cap minting, (5) bump the settings schema 3→4, (6) build two GUI toggles + a colorblind-safe home-dir warning banner, (7) replicate that warning in the TUI for release-blocking parity, and (8) fix the `proxyRemoteFiles` nil-body bug and add the five remote write proxy routes.

The single biggest architectural decision the planner must resolve is **CAP-04's "per-session" requirement vs. the existing GLOBAL `FilesRead *bool` model**: `files.read` is a single daemon-wide setting (`engine.filesRead`), but CAP-04 explicitly says "a per-session toggle." The owner-cap path at `api.go:1049` injects `files.read` for ALL sessions based on one global flag. Phase 124 must introduce per-session write state. See Open Question 1 — this is the only genuine design fork.

**Primary recommendation:** Mirror the Phase 118/119 `files.read` pipeline exactly (constant → separate middleware wrapper → mounted dispatch closure → cap-mint injection → settings field + migration test), but make the write opt-in **per-session in-memory state** (not the global-bool pattern) and use a **purpose-built Origin check** that passes vacuously on absent Origin (the inverse of the existing WS-only `requireAllowedOrigin`).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| `files.write` cap bit + `HasPerm` | API / capability pkg | — | Token claims are minted and verified server-side; `internal/capability` owns the claim vocabulary. [VERIFIED: internal/capability/capability.go] |
| `requireFilesWrite` middleware + CSRF | Frontend Server (webserver) | — | The webserver is the capability-bearing tier; the daemon Unix socket is auth-less loopback-trust. CSRF lives only where browsers reach. [VERIFIED: internal/webserver/capability_mw.go, internal/daemon/api.go:379 comment] |
| Five write routes mount | Frontend Server (webserver) | API (daemon handler) | Webserver mounts `files.Handler` write methods behind middleware; daemon socket mounts the same handler auth-less. [VERIFIED: server.go:494, api.go:147] |
| Per-session write opt-in state | API (daemon engine) | — | Cap minting (`issueCapabilitiesForSession`) reads engine state to decide perms. [VERIFIED: api.go:1049, engine.go:512] |
| Settings schemaVersion 4 migration | API (daemon engine) | Database/Storage (settings.json) | `loadSettingsFromDisk` defaults-merge owns schema evolution. [VERIFIED: engine.go:139] |
| Owner + viewer opt-in toggles | Browser / Client (React) | Frontend Server (Wails binding) | `DaemonManagerPanel` + `SessionSharePanel` render toggles; Wails `IssueCapabilities` binding carries intent to Go. [VERIFIED: DaemonManagerPanel.tsx:3, SessionSharePanel.tsx] |
| Home-dir warning (GUI) | Browser / Client (React) | API (homeDir signal) | React renders the banner; the daemon supplies a `homeDir` boolean per session. [CITED: 124-UI-SPEC.md Surface 3] |
| Home-dir warning (TUI) | Frontend Server (TUI binary) | API (engine cwd state) | lipgloss line in `renderFilesTab`; engine already knows the session cwd. [VERIFIED: internal/tui/files.go:283] |
| Remote write proxy (CAP-10) | API (daemon proxy) | — | `proxyRemoteFiles` forwards to the tailnet peer's webserver. [VERIFIED: internal/daemon/remote_files.go] |

## Project Constraints (from CLAUDE.md)

- **Go conventions:** `go fmt`, `golangci-lint`, context-aware functions (`ctx context.Context` first param) — Phase 123 client methods already follow this; write surfaces must too.
- **JS/TS:** `camelCase`, `PascalCase` components, ESLint + Prettier, TypeScript types. React frontend is bespoke BEM CSS — no shadcn/tailwind.
- **Let-it-crash / no silent fallbacks:** `or {}`-style swallowing is forbidden. The existing `requireAllowedOrigin` fails closed on empty `BaseURL()`; the new write Origin check must also fail safe.
- **Security = ASVS:** input validation, access control, CSRF are first-class. This phase IS an access-control + CSRF phase.
- **Cross-surface parity is RELEASE-BLOCKING:** the home-dir warning MUST land in both GUI and TUI (SC#4). A parity gap blocks release. [CITED: MEMORY.md feedback_cross_surface_parity]
- **User is COLORBLIND:** warnings need icon + text, not color alone. Verify at source level (glyph + literal text tokens in code), not by eye. [CITED: MEMORY.md user_colorblind]
- **Python venv rule:** not applicable (this phase is Go + TypeScript only).
- **Wails build:** production builds need `-tags wailsassets` (relevant only if a release build is exercised during UAT). [CITED: MEMORY.md project_wails_build_requires_tags]

## Standard Stack

No new libraries. Everything is Go stdlib or already-vendored.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http` | go1.26.4 | Middleware, routing (`http.ServeMux` method-prefix verbs) | Already the project's HTTP layer; Go 1.22+ mux auto-405s wrong verbs. [VERIFIED: go version go1.26.4] |
| `internal/capability` | in-tree | `Claims`, `HasPerm`, `Sign`, `Verify`, `PermFilesRead` | The capability vocabulary; add `PermFilesWrite` here. [VERIFIED: capability.go] |
| `internal/files` | in-tree (Phase 123) | `Handler.{Write,Upload,Delete,Rename,Mkdir}`, `Sandbox`, denylist | Write handlers + sandbox primitives frozen in Phase 123. [VERIFIED: 123-03-SUMMARY] |
| `charm.land/lipgloss/v2` | vendored | TUI warning line styling | Existing TUI styling lib; `StatusWaiting` token already defined. [VERIFIED: internal/tui/styles.go:62] |
| `@heroicons/react/20/solid` | vendored | GUI icons (`XMarkIcon` for dismiss) | Already used by existing banners. [CITED: 124-UI-SPEC.md] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `encoding/json` | go1.26.4 | settings.json marshal/migration | The defaults-merge migration pattern. [VERIFIED: engine.go:153] |
| Go stdlib `path/filepath` `EvalSymlinks` | go1.26.4 | home-dir cwd comparison for CAP-06 signal | Mirror the denylist's `EvalSymlinks($HOME)` pattern. [VERIFIED: sandbox.go:96-100] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| New `requireFilesWrite` wrapper | Add `files.write` case to `requireCapability` switch | REJECTED by CAP-02 — would break every terminal/relay/plugin route. The `TestRequireCapability_UnchangedByPhase118` static-grep gate pins this invariant. [VERIFIED: capability_test.go:575] |
| Per-session in-memory write state | Reuse the global `FilesRead *bool` shape for `FilesWrite` | The global pattern contradicts CAP-04's explicit "per-session toggle." See Open Question 1. |

**Installation:** None — no `go get`, no `pnpm add`.

## Package Legitimacy Audit

Not applicable. This phase installs **zero external packages**. All code uses Go stdlib, the in-tree `internal/*` packages, and already-vendored frontend deps (`@heroicons/react`, `charm.land/lipgloss/v2`). No registry verification required.

## Architecture Patterns

### System Architecture Diagram

```
                          ┌─────────────────────────────────────────┐
   Desktop GUI (Wails)    │  React frontend                          │
   fetch() — NO Origin    │  DaemonManagerPanel ─ owner write toggle │
        │                 │  SessionSharePanel  ─ viewer opt-in      │
        │                 │  HomeDirWriteWarning banner (CAP-06)     │
        │                 └──────────────┬───────────────────────────┘
        │                                │ Wails binding: IssueCapabilities(sid, opts)
        ▼                                ▼
   ┌────────────────────────────────────────────────────────────────┐
   │  Webserver tier (internal/webserver)                            │
   │                                                                 │
   │  GET  /api/files/{list,stat,read}   → requireFilesRead ─┐       │
   │  PUT  /api/files/write              ┐                    │       │
   │  POST /api/files/upload            ├→ requireFilesWrite ─┤       │
   │  DELETE /api/files/delete          │  (NEW):             │       │
   │  POST /api/files/rename            │   1. requireCapability      │
   │  POST /api/files/mkdir             ┘   2. HasPerm(files.write)   │
   │                                        3. CSRF Origin check      │
   │                                            ▼                     │
   │                              filesDispatch(ws.filesHandler.X)    │
   └────────────────────────────────────┬────────────────────────────┘
                                         │ same *files.Handler instance
   Web-share viewer (browser)            ▼
   fetch() — Origin: https://host  ┌──────────────────────────┐
        ▲                          │ files.Handler write methods│ (Phase 123)
        │                          │  → Sandbox.WriteFileAtomic │
        │                          │  → denylist ($HOME RC files)│
   ┌────┴───────────────────┐      └──────────────────────────┘
   │ Remote daemon proxy     │
   │ (CAP-10)                │  POST /api/files/remote/{sid}/write ──► tailnet peer
   │ proxyRemoteFiles        │  forwards r.Body for PUT/POST/PATCH    webserver
   │ (FIX: nil → r.Body)     │  (currently nil — the bug)
   └─────────────────────────┘

   Daemon Unix socket (internal/daemon/api.go): same 5 write routes, AUTH-LESS
   (loopback-trust, WEB-01) — NO middleware, NO Origin check. Local TUI/GUI use these.
```

### Recommended Project Structure (files touched)
```
internal/capability/capability.go         # + PermFilesWrite const
internal/webserver/capability_mw.go        # + requireFilesWrite wrapper (+ Origin check)
internal/webserver/server.go               # + 5 write route mounts via filesDispatch
internal/daemon/api.go                      # + per-session write state read in cap mint;
                                            #   + 5 remote write proxy routes
internal/daemon/engine.go                   # + FilesWrite field, per-session write map,
                                            #   schemaVersion bump consumers
internal/daemon/plugin_settings.go          # CurrentSchemaVersion 3 → 4
internal/daemon/remote_files.go             # proxyRemoteFiles: forward r.Body for write verbs
internal/daemon/engine_migration_test.go    # + TestSettingsMigration_FilesWriteDefaultsFalse
internal/webserver/capability_test.go       # + TestHasPerm_NoStringsContains_Write gate
internal/tui/files.go                       # + home-dir warning line in renderFilesTab
frontend/src/components/DaemonManagerPanel.tsx   # owner "Enable file writes" toggle
frontend/src/components/SessionSharePanel.tsx    # viewer "Allow file editing" opt-in
frontend/src/components/HomeDirWriteWarning.tsx  # NEW banner (or inline in DaemonManagerPanel)
frontend/src/style.css                      # + .session-share-panel__write-optin,
                                            #   + --home-write-warning banner modifier (NO new hex)
```

### Pattern 1: Capability constant + HasPerm (mirror PermFilesRead)
**What:** Add `PermFilesWrite = "files.write"` next to `PermFilesRead`. Always gate via `HasPerm` (whole-token comma-split), NEVER `strings.Contains`.
**When to use:** Every write-path permission check.
```go
// Source: internal/capability/capability.go (mirror PermFilesRead at line 31)
const PermFilesWrite = "files.write"

// HasPerm("read,write,files.write", "files.write") -> true
// HasPerm("read,no-files.write",    "files.write") -> false  // critical: substring would false-positive
// (HasPerm splits on "," and compares whole tokens — capability.go:42)
```

### Pattern 2: requireFilesWrite as a THIRD separate wrapper
**What:** Compose `requireCapability` (HMAC/grant/session checks) → `HasPerm(files.write)` → CSRF Origin check → handler. Do NOT touch `requireCapability` or `requireFilesRead`.
**When to use:** All five webserver write routes.
```go
// Source: mirror internal/webserver/capability_mw.go:102 (requireFilesRead)
func (ws *WebServer) requireFilesWrite(next http.HandlerFunc) http.HandlerFunc {
	return ws.requireCapability(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := capability.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "files.write capability required", http.StatusForbidden)
			return
		}
		if !capability.HasPerm(claims.Perms, capability.PermFilesWrite) {
			http.Error(w, "files.write capability required", http.StatusForbidden)
			return
		}
		// CSRF Origin check — see Pitfall 1 for the vacuous-on-absent logic.
		if !ws.originAllowedForWrite(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}
```
The literal body string `"files.write capability required"` is a load-bearing contract assertion (mirrors `requireFilesRead`'s `"files.read capability required"` and SC#1's "403 not 404/401" requirement).

### Pattern 3: Mount via filesDispatch (the Phase 119 precedent)
**What:** Reuse the `filesDispatch` closure so the handler is read at request time (nil-safe), method-prefixed for auto-405.
```go
// Source: internal/webserver/server.go:494-501 (filesDispatch) + 503-506 (read mounts)
mux.HandleFunc("PUT /api/files/write",     ws.requireFilesWrite(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.Write })))
mux.HandleFunc("POST /api/files/upload",   ws.requireFilesWrite(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.Upload })))
mux.HandleFunc("DELETE /api/files/delete", ws.requireFilesWrite(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.Delete })))
mux.HandleFunc("POST /api/files/rename",   ws.requireFilesWrite(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.Rename })))
mux.HandleFunc("POST /api/files/mkdir",    ws.requireFilesWrite(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.Mkdir })))
```
**The five verbs (frozen by Phase 123 daemon routes, `api.go`):** `PUT write`, `POST upload`, `DELETE delete`, `POST rename`, `POST mkdir`. SC#1's per-endpoint "correct method" coverage maps to these exactly.

### Pattern 4: Settings migration (defaults-merge constructor)
**What:** Bump `CurrentSchemaVersion 3 → 4`, add `FilesWrite` to `daemonSettings`, pre-populate the default in `loadSettingsFromDisk` BEFORE `Unmarshal`. For CAP-04's per-session requirement, the persisted default is `false`.
```go
// Source: internal/daemon/plugin_settings.go:8
const CurrentSchemaVersion = 4 // was 3 (Phase 124 / CAP-08)

// daemonSettings (engine.go:96): owner-level write default field.
// NOTE: unlike FilesRead (*bool default *true), FilesWrite defaults FALSE (CAP-04 opt-in-for-all).
// Use *bool with omitempty if a tri-state (unset vs explicit-false) is needed; or a plain
// bool that the constructor leaves at its zero value (false) — see Open Question 1.
```
**Migration test template:** `engine_migration_test.go:206` (`TestSettingsMigration_FilesReadDefaultsTrue`) — invert the assertion to `FilesWrite == false` for `TestSettingsMigration_FilesWriteDefaultsFalse` (SC#5). Also add a schemaVersion-rewrite test mirroring `TestSettingsMigration_FilesReadSchemaVersionRewrite` (`:267`).

### Pattern 5: Per-session write opt-in in cap minting
**What:** The owner cap (`wClaims`) gets `files.write` appended ONLY when the session's write toggle is ON. The viewer/join-code cap gets it only on a further explicit opt-in.
```go
// Source: internal/daemon/api.go:1049-1060 (issueCapabilitiesForSession)
ownerPerms := "read,write"
if a.engine.filesReadEnabled() {
	ownerPerms = "read,write," + capability.PermFilesRead
}
// Phase 124: append files.write when the per-session write toggle is ON.
if a.engine.filesWriteEnabledFor(sessionID) {   // NEW per-session check
	ownerPerms += "," + capability.PermFilesWrite
}
// Viewer (write-share) cap: add files.write ONLY on the explicit viewer opt-in (CAP-05).
```
**Important nuance:** The viewer "Full Access Link" (`writeURL`/`wClaims`) is ALREADY a `read,write` cap today — CAP-05's toggle controls whether THAT cap additionally carries `files.write`. The read-only link (`rClaims`, `Perms: "read"`) is never affected.

### Pattern 6: Home-dir signal (server-computed, GUI+TUI consume)
**What:** Compute `homeDir bool` per session server-side by comparing the EvalSymlinks-resolved session cwd against the EvalSymlinks-resolved `$HOME`. Mirror the exact normalization the denylist already does (`sandbox.go:96-100`) to avoid the macOS `/var` vs `/private/var` mismatch that bit Phase 123 (123-01-SUMMARY deviation 1).
```go
// Source pattern: internal/files/sandbox.go:96-100
home, _ := os.UserHomeDir()
resolvedHome, _ := filepath.EvalSymlinks(home)
wd := e.GetSessionWorkDir(sessionID) // already EvalSymlinks-resolved (engine.go:506)
isHomeCwd := wd == resolvedHome      // exact-match; sub-dirs of $HOME are NOT $HOME-cwd
```
The GUI reads `homeDir` (via the Wails session-info / IssueCapabilities response); the TUI reads engine cwd state directly.

### Anti-Patterns to Avoid
- **`strings.Contains(claims.Perms, "files.write")`** — false-positives on `"no-files.write"`. The `TestHasPerm_NoStringsContains_Write` static-grep gate (SC#3) MUST fail any write-path file containing this. [VERIFIED: PITFALLS.md:254]
- **Adding `files.write` to `requireCapability`** — breaks every non-file route (terminal/relay/plugin). CAP-02 explicitly rejects this. [VERIFIED: CAP-02, capability_test.go:575]
- **Copying `requireAllowedOrigin` verbatim for the write check** — it REJECTS absent Origin (WS-only). The write check must PASS on absent Origin. See Pitfall 1.
- **Modeling `FilesWrite` as a single global bool like `FilesRead`** — contradicts CAP-04's "per-session." See Open Question 1.
- **`color`-only warnings** — user is colorblind; every warning needs `⚠` glyph + literal "Warning:" text. [VERIFIED: 124-UI-SPEC.md, MEMORY.md]
- **Touching `internal/tui/files_client.go`** — the `FilesClient` interface is intentionally NOT extended for writes until Phase 126 (T-123-19 scope guard). [VERIFIED: 123-04-SUMMARY]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic write / rename / mkdir / delete | New filesystem code | `files.Handler.{Write,Upload,Delete,Rename,Mkdir}` (Phase 123) | Frozen, fuzz-gated, denylist-protected. [VERIFIED: 123-03-SUMMARY] |
| Permission whole-token check | substring/regex | `capability.HasPerm` | Comma-split semantics already correct + tested. [VERIFIED: capability.go:42] |
| HMAC cap minting | New signer | `capability.Sign` + extend `Perms` string | Field order is load-bearing; appending to `Perms` is the only safe change. [VERIFIED: capability.go:17] |
| Settings migration | New migration framework | `loadSettingsFromDisk` defaults-merge | The established Pitfall-14/16 pattern. [VERIFIED: engine.go:139] |
| GUI toggle | New component | `settings-panel__toggle-*` BEM classes | Reuse verbatim; add `role="switch"`+`aria-checked`. [VERIFIED: style.css:680-728] |
| GUI warning banner | New banner | `webgl-recovery-banner` + new `--home-write-warning` modifier | Reuse structure; only add the amber `#f59e0b` left-border modifier (existing hex). [CITED: 124-UI-SPEC.md] |
| Multipart upload body (remote proxy) | Re-parse form | Forward `r.Body` opaquely | The proxy is a byte pipe; just stop passing `nil`. See CAP-10. [VERIFIED: remote_files.go:168] |

**Key insight:** Phase 123 deliberately built the entire write engine (sandbox + handlers + daemon routes + client methods) and froze it. Phase 124 is **pure gating + UI** — it adds no filesystem logic. The temptation to "improve" the write primitives must be resisted (Chesterton's Fence — they are fuzz-gated and security-reviewed).

## Runtime State Inventory

This phase is partly a settings-schema migration, so the inventory applies.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `settings.json` at `daemonConfigDir()` with `schemaVersion: 3` and no `filesWrite` key. | **Data migration (automatic):** `loadSettingsFromDisk` defaults-merge rewrites to `schemaVersion: 4` with `FilesWrite: false` on next daemon start (idempotent, mirrors v3.2→v3.3). No manual migration. [VERIFIED: engine.go:184-196] |
| Live service config | Outstanding cap tokens already issued to web-share viewers encode `Perms` at mint time. A token minted before Phase 124 has no `files.write` bit — it will simply 403 on write routes (correct fail-safe). | **None** — old tokens degrade safely; re-issue (toggle off/on) to grant write. The HMAC field order is unchanged (still `SID,Perms,IAT,GrantID,V`), so old tokens still VERIFY. [VERIFIED: capability.go:17] |
| OS-registered state | None — no OS scheduler/launchd/registry entries reference write capability. | None — verified by absence in grep of `internal/`. |
| Secrets/env vars | None — no new secret or env var. The HMAC signing key (`KeyStore`) is unchanged. | None. |
| Build artifacts | Wails frontend bundle (`/app/`) is rebuilt from React source; the new toggle/banner components ship in the next `pnpm build`. TUI binary recompiles with the new warning line. | **Rebuild required:** frontend `pnpm build` (Wails `-tags wailsassets` for prod) + Go recompile. No stale-artifact migration beyond normal build. |

**Canonical question — what runtime state survives a source update?** Only previously-issued cap tokens (handled: they fail-safe to 403 without `files.write`) and the on-disk `settings.json` (handled: automatic schema 3→4 defaults-merge). Both are covered by existing patterns.

## Common Pitfalls

### Pitfall 1: The write Origin check is the INVERSE of `requireAllowedOrigin` on absent Origin
**What goes wrong:** Copying `requireAllowedOrigin` (`origin_mw.go:31`) verbatim would 403 the desktop Wails `fetch()` (which sends NO `Origin` header), breaking all local writes.
**Why it happens:** `requireAllowedOrigin` is WS-upgrade-only and browser-only, so it correctly rejects absent Origin (D-05). But CAP-03 / Phase 88-pattern for HTTP write verbs requires the OPPOSITE: absent Origin passes vacuously (desktop), present Origin must match FQDN.
**How to avoid:** Write a purpose-built check:
```go
func (ws *WebServer) originAllowedForWrite(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // desktop Wails fetch sends no Origin — pass vacuously (CAP-03)
	}
	allowed := ws.BaseURL()
	return allowed != "" && origin == allowed // present → strict byte-for-byte match
}
```
Only enforce on state-changing verbs (POST/PUT/PATCH/DELETE) — all five write routes qualify. Fail closed if `BaseURL()` is empty AND Origin is present (never silently pass).
**Warning signs:** Local desktop writes 403 in UAT; or a cross-origin web-share write succeeds when it should 403. SC#2 tests both branches.

### Pitfall 2: `FilesRead` is global; `FilesWrite` must be per-session
**What goes wrong:** Reusing the `engine.filesRead *bool` single-flag pattern makes write a daemon-wide on/off, but CAP-04 + UI-SPEC Surface 1 require a PER-SESSION "Enable file writes" toggle.
**Why it happens:** `files.read` shipped as a global operator preference (Phase 118 FS-14); the obvious move is to copy it. The requirement differs.
**How to avoid:** Add a per-session write map (e.g. `sessionWrites map[string]bool` under `e.mu`) plus an `FilesWrite bool` settings *default* that seeds new sessions. The cap mint reads `filesWriteEnabledFor(sessionID)`, not a global. See Open Question 1 for the persistence model decision.
**Warning signs:** Enabling write on one session enables it on all sessions; or the GUI toggle has no per-session effect.

### Pitfall 3: `proxyRemoteFiles` drops the request body for write verbs
**What goes wrong:** `remote_files.go:168` builds the upstream request with `nil` body — correct for GET/HEAD (read proxy) but silently discards the PUT/POST payload for write proxy, so remote writes appear to "succeed" with empty content.
**Why it happens:** The proxy was written for read-only routes where body is always nil. CAP-10 explicitly flags this.
**How to avoid:** Forward `r.Body` for PUT/POST/PATCH:
```go
var body io.Reader
if r.Method == http.MethodPut || r.Method == http.MethodPost || r.Method == http.MethodPatch {
	body = r.Body
}
req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, body)
// Forward Content-Type (multipart boundary, octet-stream, application/json) — currently not copied.
```
**Critical:** also forward the inbound `Content-Type` header (the proxy currently only copies RESPONSE headers, not request headers) — multipart upload needs its boundary preserved, and rename needs `application/json`.
**Warning signs:** Remote write 2xx but file is empty; remote upload fails to parse multipart; remote rename gets an empty JSON body → 400.

### Pitfall 4: macOS $HOME symlink mismatch in the home-dir check (CAP-06)
**What goes wrong:** Comparing `GetSessionWorkDir` (EvalSymlinks-resolved, e.g. `/private/var/...`) against a raw `os.UserHomeDir()` (`/var/...` on macOS) returns false for an actual home session — the warning never shows. This EXACT bug bit Phase 123's denylist (123-01-SUMMARY deviation 1).
**Why it happens:** `os.UserHomeDir()` returns `$HOME` as-is; the sandbox stores the EvalSymlinks-canonical form.
**How to avoid:** `filepath.EvalSymlinks(home)` before comparing — copy the denylist's exact approach (`sandbox.go:96-100`).
**Warning signs:** The TUI/GUI warning never appears on a `cwd=$HOME` session during UAT.

### Pitfall 5: Colorblind-unsafe warning rendering
**What goes wrong:** Relying on amber color (GUI) or `StatusWaiting` foreground (TUI) alone to signal "warning" — invisible to the colorblind user.
**Why it happens:** Color is the easy signal.
**How to avoid:** Every warning carries the `⚠` glyph AND the literal word `Warning:`. Verify at SOURCE level (grep the glyph + text token in code), not by eye. TUI must NOT use bold-alone either. [VERIFIED: 124-UI-SPEC.md Color §, MEMORY.md user_colorblind]
**Warning signs:** UAT "passes" by visual inspection without confirming the glyph+text tokens exist in source.

### Pitfall 6: 403 must be distinguishable from 404/401 (SC#1)
**What goes wrong:** A cap WITHOUT `files.write` hitting a write route returns 404 (route-not-found) or 401 (no cap) instead of 403 (cap present, perm missing).
**Why it happens:** Wrong middleware ordering — if the perm check runs before `requireCapability`, or if the route isn't mounted, you get the wrong status.
**How to avoid:** The wrapper order is `requireCapability` (401 on bad/absent cap) → `HasPerm` (403 on present-cap-missing-perm) → Origin (403). A valid cap missing `files.write` must reach the `HasPerm` branch and 403. SC#1 asserts exactly this. The `requireFilesRead` test (`capability_test.go:434`) shows the 401-before-403 ordering.

## Code Examples

### Static-grep gate (SC#3, mirror the read-side gate)
```go
// Source: mirror internal/webserver/capability_test.go:575 (TestRequireCapability_UnchangedByPhase118)
// TestHasPerm_NoStringsContains_Write asserts no write-path source uses
// strings.Contains for the files.write check — must use HasPerm (Pitfall 4 / T-118-13 analog).
func TestHasPerm_NoStringsContains_Write(t *testing.T) {
	for _, f := range []string{"capability_mw.go", "../../internal/daemon/api.go" /* etc. */} {
		data, _ := os.ReadFile(f)
		if strings.Contains(string(data), `strings.Contains(`) &&
			strings.Contains(string(data), `"files.write"`) {
			t.Errorf("%s: use capability.HasPerm, not strings.Contains, for files.write", f)
		}
	}
}
```
The planner should scope the file list to the actual write-path files and assert the precise forbidden pattern (`strings.Contains(<perms>, "files.write")`), mirroring how the read-side gate bounds the `requireCapability` function body.

### Capability-denied integration test (SC#1 / CAP-09)
```go
// Mirror internal/webserver/capability_test.go TestRequireFilesRead (line 434).
// For each of the 5 write routes, with its correct verb:
//   cap with Perms="read,write" (NO files.write) → expect 403
//   cap with Perms="read,write,files.write"      → expect 2xx
// Plus: write verb with mismatched Origin header → 403 (CSRF, SC#2)
//       write verb with absent Origin            → passes the Origin gate (SC#2)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `files.read` global operator flag (`FilesRead *bool`, default true) | `files.write` per-session opt-in (default OFF for all) | Phase 124 (this) | Write is strictly opt-in; read defaulted on. Do not blindly copy the read pattern. |
| Read routes are GET-only, CSRF-safe by convention | Write routes are POST/PUT/DELETE — need explicit Origin CSRF check | Phase 124 | New `originAllowedForWrite` helper required. [VERIFIED: PITFALLS.md:128] |
| `proxyRemoteFiles` passes `nil` body (read-only) | Forward `r.Body` + Content-Type for write verbs | Phase 124 (CAP-10) | Remote writes were impossible before. |

**Deprecated/outdated:** Nothing deprecated. The `os.Root` write API gaps noted in old research are already handled by Phase 123's `Sandbox` (which uses native `root.Rename`/`root.Mkdir`/`root.RemoveAll`).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The GUI receives the `homeDir` boolean via the `IssueCapabilities` response (or session-info), not a separate endpoint. | Pattern 6 / CAP-06 | LOW — the exact wire path is planner discretion; any server→GUI channel works. UI-SPEC says "GUI receives a `homeDir: true` signal from the daemon." |
| A2 | The viewer write-share cap is the existing `writeURL`/`wClaims` (`read,write`) cap, and CAP-05's toggle adds `files.write` to it (rather than minting a 3rd cap). | Pattern 5 / CAP-05 | MEDIUM — if the planner mints a separate viewer cap, the `issueCapabilitiesForSession` signature changes more. Either is valid; A2 is the minimal change. |

**Note:** Both assumptions are implementation-shape choices within Claude's Discretion (per CONTEXT.md), not factual claims about external systems. No user confirmation is blocking; the planner picks the shape.

## Open Questions

1. **Per-session write opt-in: persistence model.**
   - What we know: CAP-04 requires a per-session toggle; CAP-08 requires `FilesWrite: false` to persist in `settings.json` schemaVersion 4. The existing `FilesRead` is a single global `*bool`.
   - What's unclear: Is `FilesWrite` (a) a global *default* in settings that seeds each new session's in-memory write state (sessions are ephemeral, so per-session persistence across restart may be moot since sessions don't survive daemon restart), or (b) a per-session persisted map? Sessions in this codebase do NOT survive daemon restart (the engine rebuilds session state on start), so "web-share opt-in persists across daemon restarts" (SC#5) most plausibly means the **settings-level default** persists, while the per-session toggle is in-memory for live sessions.
   - Recommendation: Model `FilesWrite bool` as a persisted settings DEFAULT (schemaVersion 4, default false) AND a per-session in-memory override map (`map[string]bool` under `e.mu`). New sessions inherit the settings default (false); the GUI per-session toggle flips the in-memory map; cap minting reads the map. The migration test (`TestSettingsMigration_FilesWriteDefaultsFalse`) asserts the settings default. The "persists across daemon restarts" criterion is satisfied by the settings default (since sessions themselves don't persist). **Planner should confirm this interpretation against the ROADMAP SC#5 wording during planning.**

2. **Does the home-dir warning need a new daemon endpoint or can it ride existing session-info?**
   - What we know: `handleSessionInfo` (`server.go:746`) exists and populates the terminal status bar with perms.
   - What's unclear: whether to extend that response with `homeDir`/`canWrite` or add it to `IssueCapabilitiesResponse`.
   - Recommendation: Extend the existing response the GUI already polls per session (least new surface). Planner's discretion.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All Go changes | ✓ | go1.26.4 | — |
| Node/pnpm | React frontend build | ✓ (assumed; project uses pnpm) | — | — |
| Wails CLI | Desktop GUI build/UAT | ✓ (project tooling) | — | `wails dev` for DevTools UAT (prod has none) |

No external services. No missing blocking dependencies.

## Validation Architecture

> `workflow.nyquist_validation` is absent from config.json — treated as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (+ `-race`); frontend `vitest` (DaemonManagerPanel.test.tsx exists) |
| Config file | none for Go (stdlib); frontend uses vitest config in `frontend/` |
| Quick run command | `go test -race ./internal/webserver/ ./internal/daemon/ -run 'FilesWrite\|requireFilesWrite\|HasPerm' -count=1` |
| Full suite command | `go test -race ./... -count=1` + `pnpm --dir frontend test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CAP-01 | `PermFilesWrite` const + HasPerm | unit | `go test ./internal/capability/ -run HasPerm` | ✅ (HasPerm tests exist) |
| CAP-02/03/09 | 403 w/o write, 2xx w/ write, Origin 403, all 5 routes | integration | `go test -race ./internal/webserver/ -run requireFilesWrite` | ❌ Wave 0 |
| CAP-03 (SC#2) | mismatched Origin → 403; absent Origin → pass | unit | `go test ./internal/webserver/ -run OriginWrite` | ❌ Wave 0 |
| CAP-09 (SC#3) | static-grep no `strings.Contains(...,"files.write")` | static | `go test ./internal/webserver/ -run TestHasPerm_NoStringsContains_Write` | ❌ Wave 0 |
| CAP-08 (SC#5) | schemaVersion 3→4, FilesWrite default false | unit | `go test ./internal/daemon/ -run TestSettingsMigration_FilesWrite` | ❌ Wave 0 |
| CAP-10 | remote proxy forwards body for write verbs | integration | `go test -race ./internal/daemon/ -run RemoteFilesWrite` | ❌ Wave 0 |
| CAP-04/05 | owner+viewer toggle includes files.write in cap | unit (Go cap mint) + component (React) | `go test ./internal/daemon/ -run IssueCapabilities` + `pnpm test SessionSharePanel` | ⚠ partial |
| CAP-06 | home-dir warning GUI + TUI (parity) | component + manual/UAT | `pnpm test HomeDirWriteWarning` + TUI source-grep for `⚠ Warning: cwd is $HOME` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** the targeted `-run` command for that task's req.
- **Per wave merge:** `go test -race ./internal/webserver/ ./internal/daemon/ ./internal/capability/ -count=1` + frontend tests.
- **Phase gate:** `go test -race ./... -count=1` green + `pnpm --dir frontend test` green before `/gsd:verify-work`.

### Wave 0 Gaps
- [ ] `internal/webserver/capability_test.go` — add `TestRequireFilesWrite` (403/2xx, all 5 routes) + `TestHasPerm_NoStringsContains_Write` static gate (covers CAP-02/03/09)
- [ ] `internal/webserver/` — Origin-check unit test for vacuous-on-absent + strict-on-present (CAP-03 / SC#2)
- [ ] `internal/daemon/engine_migration_test.go` — `TestSettingsMigration_FilesWriteDefaultsFalse` + schemaVersion-rewrite test (CAP-08 / SC#5)
- [ ] `internal/daemon/remote_files_test.go` — remote write-proxy body-forwarding test (CAP-10)
- [ ] `frontend/src/components/__tests__/` — SessionSharePanel write opt-in + HomeDirWriteWarning component tests (CAP-05/06)
- [ ] TUI: source-grep assertion for the verbatim home-dir warning string (CAP-06 parity) — no framework install needed.

## Security Domain

> `security_enforcement` not set false in config → enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | HMAC-SHA256 cap tokens (`capability.Verify`); unchanged. |
| V3 Session Management | yes | Grant-list revocation (`isGrantActive`); session-bound caps (SEC-03). |
| V4 Access Control | yes | `requireFilesWrite` + `HasPerm` whole-token check; 403 on missing perm (CAP-02/09). |
| V5 Input Validation | yes | Sandbox `validateAndClean` + denylist (Phase 123); multipart `filepath.Base` + 50 MiB cap. |
| V6 Cryptography | yes (existing) | HMAC signing key via `KeyStore`; constant-time `hmac.Equal`. Never hand-roll — unchanged this phase. |
| (CSRF — cross-cutting) | yes | `originAllowedForWrite` Origin check on state-changing verbs (CAP-03 / Pitfall 1). |

### Known Threat Patterns for Go webserver + capability-token + web-share

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| CSRF on write verbs from a malicious same-tailnet site | Tampering | Origin check: present-Origin must match FQDN; absent passes vacuously (CAP-03). [VERIFIED: PITFALLS.md:139] |
| Substring false-positive (`no-files.write` matches) | Elevation of Privilege | `HasPerm` comma-split; static-grep gate forbids `strings.Contains`. [VERIFIED: capability.go:42, PITFALLS.md:254] |
| Cap scope creep (write beyond session cwd) | Elevation of Privilege | Session-bound cap (SEC-03) + Sandbox `os.Root` confinement (Phase 123). |
| Shell-RC / SSH-key overwrite in $HOME session | Tampering | Sandbox denylist (Phase 123 FSW-06) + home-dir warning (CAP-06) defense-in-depth. |
| Remote proxy smuggling a different cap | Spoofing | `proxyRemoteFiles` strips caller-supplied `?cap=` and force-sets from store (already done, remote_files.go). Keep when adding write verbs. |
| Stale pre-Phase-124 cap silently gaining write | Elevation of Privilege | Old caps lack the `files.write` token → 403 (fail-safe). HMAC field order unchanged → old caps still verify but can't write. |

## Sources

### Primary (HIGH confidence)
- `internal/capability/capability.go` — `Claims`, `HasPerm` (:42), `PermFilesRead` (:31), `Sign`/`Verify`.
- `internal/webserver/capability_mw.go` — `requireCapability` (:37), `requireFilesRead` (:102).
- `internal/webserver/origin_mw.go` — `requireAllowedOrigin` (:31) Phase 88 Origin check.
- `internal/webserver/server.go` — `filesDispatch` (:494), read route mounts (:503), `handleSessionInfo` (:746).
- `internal/daemon/api.go` — `issueCapabilitiesForSession` (:1049), daemon write routes, remote route registration (:161-164).
- `internal/daemon/engine.go` — `daemonSettings` (:96), `loadSettingsFromDisk` (:139), `filesReadEnabled` (:512), `GetSessionWorkDir` (:502).
- `internal/daemon/plugin_settings.go` — `CurrentSchemaVersion = 3` (:8).
- `internal/daemon/remote_files.go` — `proxyRemoteFiles` (:168, nil-body bug).
- `internal/daemon/engine_migration_test.go` — FilesRead migration test templates (:206, :267).
- `internal/files/sandbox.go` — denylist `EvalSymlinks($HOME)` (:96-100).
- `internal/tui/files.go` — `renderFilesStatusLine` (:283), `renderFilesTab` (:331); `internal/tui/styles.go` tokens (:62).
- `frontend/src/components/SessionSharePanel.tsx`, `DaemonManagerPanel.tsx`; `frontend/src/style.css` toggle CSS (:680-728).
- Phase 123 SUMMARYs (01-04) — frozen write API contract.
- `.planning/REQUIREMENTS.md` CAP-01..CAP-10; `124-CONTEXT.md`; `124-UI-SPEC.md`.

### Secondary (MEDIUM confidence)
- `.planning/research/PITFALLS.md` — CSRF (Pitfall 4), HasPerm substring (Pitfall 7), atomic write, upload abuse.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new deps; all primitives read from live source.
- Architecture: HIGH — direct mirror of the Phase 118/119 read-side pipeline + Phase 88 Origin precedent, all source-confirmed.
- Pitfalls: HIGH — each pitfall traced to a specific line in this repo (the macOS symlink bug actually occurred in Phase 123).
- Open Question 1 (per-session persistence model): MEDIUM — the SC#5 wording is interpretable; recommendation given, planner should confirm.

**Research date:** 2026-06-14
**Valid until:** 2026-07-14 (stable — internal codebase, no fast-moving external deps)
