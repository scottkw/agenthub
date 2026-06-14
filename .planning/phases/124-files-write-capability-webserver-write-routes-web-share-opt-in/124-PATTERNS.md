# Phase 124: files.write Capability + Webserver Write Routes + Web-Share Opt-In - Pattern Map

**Mapped:** 2026-06-14
**Files analyzed:** 13 (11 modified, 2 created)
**Analogs found:** 13 / 13 (every file has a Phase 118/119 read-side or Phase 88 Origin precedent in-tree)

> This phase is **pure gating + UI** — Phase 123 froze the write engine (`files.Handler.{Write,Upload,Delete,Rename,Mkdir}`, `Sandbox`, denylist). No filesystem logic is added. Every new file copies an existing read-side analog almost verbatim, with **three deliberate inversions** flagged in `## Critical Inversions` below — the planner MUST NOT copy those analogs verbatim.

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/capability/capability.go` | model (claim vocab) | transform | `PermFilesRead` const (`capability.go:30`) | exact |
| `internal/webserver/capability_mw.go` | middleware | request-response | `requireFilesRead` (`capability_mw.go:102`) | exact |
| `internal/webserver/capability_mw.go` (Origin helper) | middleware | request-response | `requireAllowedOrigin` (`origin_mw.go:31`) | **INVERSE — see Critical Inversion 1** |
| `internal/webserver/server.go` | route (mount) | request-response | read mounts (`server.go:502-505`) + `filesDispatch` (`:492`) | exact |
| `internal/daemon/api.go` (cap mint) | service | transform | `issueCapabilitiesForSession` (`api.go:1048-1060`) | role-match (**per-session, not global — Inversion 2**) |
| `internal/daemon/api.go` (remote routes) | route (mount) | request-response | remote read routes (`api.go:161-164`) | exact |
| `internal/daemon/engine.go` | service/store | transform | `daemonSettings` (`engine.go:93-102`), `filesReadEnabled` (`:514`), `loadSettingsFromDisk` (`:143`) | role-match (**per-session map — Inversion 2**) |
| `internal/daemon/plugin_settings.go` | config | — | `CurrentSchemaVersion = 3` (`plugin_settings.go:8`) | exact |
| `internal/daemon/remote_files.go` | service (proxy) | streaming/file-I/O | `proxyRemoteFiles` (`remote_files.go:167-203`) | **INVERSE on body — see Critical Inversion 3** |
| `internal/daemon/engine_migration_test.go` | test | — | `TestSettingsMigration_FilesReadDefaultsTrue` (`:206`), `...SchemaVersionRewrite` (`:267`) | exact (**invert assertion to `false`**) |
| `internal/webserver/capability_test.go` | test | — | `TestRequireFilesRead` (`:445`), `TestRequireCapability_UnchangedByPhase118` (`:575`) | exact |
| `internal/tui/files.go` | component (TUI) | event-driven | `renderFilesStatusLine` (`files.go:283`) | role-match |
| `frontend/src/components/SessionSharePanel.tsx` | component (React) | event-driven | existing link-row + `handleCopy` (`SessionSharePanel.tsx`) | exact |
| `frontend/src/components/HomeDirWriteWarning.tsx` (NEW) | component (React) | request-response | `webgl-recovery-banner` markup + `--shell-warning` modifier (`style.css:1794, 1851`) | role-match |

---

## Critical Inversions (DO NOT COPY VERBATIM)

These three analogs are the closest structural match BUT have opposite semantics. Copying verbatim ships a bug. The planner must write purpose-built code per the spec below.

### Inversion 1: write Origin check is the INVERSE of `requireAllowedOrigin` on absent Origin

`requireAllowedOrigin` (`origin_mw.go:31-51`) **REJECTS** an absent `Origin` header (it is WS-upgrade-only; browsers always send Origin on WS). The new `originAllowedForWrite` must **PASS vacuously** on absent Origin (desktop Wails `fetch()` sends none) and reject only a *present-and-mismatched* Origin. Fail closed only when `BaseURL()` is empty AND an Origin is present.

**Analog to study (then invert the empty-origin branch):** `origin_mw.go:32-49`
```go
origin := r.Header.Get("Origin")
if origin == "" {
    http.Error(w, "forbidden", http.StatusForbidden) // <-- INVERT: write check must `return true` here
    return
}
allowed := ws.BaseURL()
if allowed == "" || origin != allowed {
    http.Error(w, "forbidden", http.StatusForbidden)
    return
}
```
**Target shape (RESEARCH Pitfall 1):** `if origin == "" { return true }` then strict `allowed != "" && origin == allowed`.

### Inversion 2: `FilesWrite` is PER-SESSION, `FilesRead` is GLOBAL

`filesReadEnabled()` (`engine.go:514-518`) is a single daemon-wide `*bool` (`e.filesRead`). CAP-04 requires a **per-session** toggle. Do NOT add a sibling global `e.filesWrite *bool` consumed identically. Add a per-session in-memory map (`map[string]bool` under `e.mu`) plus a persisted `FilesWrite bool` settings *default* (false) that seeds new sessions. Cap mint reads `filesWriteEnabledFor(sessionID)`, not a global. (Open Question 1 — planner confirms persistence model against SC#5.)

### Inversion 3: `proxyRemoteFiles` passes `nil` body — must forward `r.Body` for write verbs

`remote_files.go:169` builds the upstream request with a `nil` body (correct for GET/HEAD read proxy). For PUT/POST/PATCH write verbs this silently drops the payload. Forward `r.Body` for write verbs AND forward the inbound `Content-Type` request header (currently only RESPONSE headers `:188-199` are copied — multipart boundary + `application/json` must be preserved on the request).

---

## Pattern Assignments

### `internal/capability/capability.go` (model, add `PermFilesWrite`)

**Analog:** `PermFilesRead` (`capability.go:27-30`)
```go
// PermFilesRead is the capability bit token granting access to /api/files/*
const PermFilesRead = "files.read"
```
**Add directly beneath:** `const PermFilesWrite = "files.write"` with a mirrored doc comment (gated via `HasPerm` whole-token, never substring). Do NOT touch `Claims` struct (`:19-25`) — field order is HMAC-load-bearing; appending to the `Perms` *string* at mint time is the only safe change. `HasPerm` (`:44-54`) is reused as-is.

---

### `internal/webserver/capability_mw.go` (middleware, add `requireFilesWrite` + `originAllowedForWrite`)

**Analog:** `requireFilesRead` (`capability_mw.go:102-119`) — copy structure exactly, compose `requireCapability` → `ClaimsFromContext` → `HasPerm` → (NEW) Origin check → `next`.
```go
func (ws *WebServer) requireFilesRead(next http.HandlerFunc) http.HandlerFunc {
	return ws.requireCapability(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := capability.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "files.read capability required", http.StatusForbidden)
			return
		}
		if !capability.HasPerm(claims.Perms, capability.PermFilesRead) {
			http.Error(w, "files.read capability required", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}
```
**For the write wrapper:** swap the literal body string to `"files.write capability required"` (load-bearing contract assertion, SC#1) and `PermFilesRead` → `PermFilesWrite`, then insert the `originAllowedForWrite(r)` gate (returns 403 "forbidden" when false) BEFORE `next`. Ordering invariant (Pitfall 6): `requireCapability` (401 on bad/absent cap) → `HasPerm` (403) → Origin (403). A valid cap missing `files.write` MUST reach the `HasPerm` 403 branch, not 404/401.

**Origin helper** — see Critical Inversion 1; study `origin_mw.go:32-49` but invert the empty-origin branch.

**SEPARATION INVARIANT (CAP-02):** `requireFilesWrite` is a THIRD wrapper. Do NOT add a `files.write` case to `requireCapability` or `requireFilesRead`. The static-grep gate (`capability_test.go:575`) pins this.

---

### `internal/webserver/server.go` (route mounts, 5 write routes)

**Analog:** `filesDispatch` closure (`server.go:492-501`) + read mounts (`:502-505`). `filesDispatch` reads `ws.filesHandler` at request time (nil-safe → 503) and is method-prefixed for Go 1.22+ auto-405.
```go
filesDispatch := func(op func(*files.Handler) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := ws.filesHandler
		if h == nil {
			http.Error(w, "files handler not configured", http.StatusServiceUnavailable)
			return
		}
		op(h)(w, r)
	}
}
mux.HandleFunc("GET /api/files/list", ws.requireFilesRead(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.List })))
```
**Add 5 write mounts** with `requireFilesWrite` and the verbs frozen by the daemon side (`api.go:149-153`): `PUT /api/files/write`→`h.Write`, `POST /api/files/upload`→`h.Upload`, `DELETE /api/files/delete`→`h.Delete`, `POST /api/files/rename`→`h.Rename`, `POST /api/files/mkdir`→`h.Mkdir`. Reuse the SAME `filesDispatch` closure; do not duplicate it.

---

### `internal/daemon/api.go` — cap minting (`issueCapabilitiesForSession`)

**Analog:** `api.go:1048-1060`
```go
ownerPerms := "read,write"
if a.engine.filesReadEnabled() {
	ownerPerms = "read,write," + capability.PermFilesRead
}
rClaims := capability.Claims{SID: sessionID, Perms: "read", IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
wClaims := capability.Claims{SID: sessionID, Perms: ownerPerms, IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}
```
**Append (per Inversion 2):** after the `filesReadEnabled()` block, `if a.engine.filesWriteEnabledFor(sessionID) { ownerPerms += "," + capability.PermFilesWrite }`. The `wClaims`/`writeURL` "Full Access Link" is ALREADY `read,write`; CAP-05's viewer toggle controls whether THAT cap additionally carries `files.write` (Assumption A2 — minimal change, no 3rd cap). The read-only `rClaims` (`Perms: "read"`) is NEVER affected. HMAC field order unchanged → old tokens still verify but lack the bit → fail-safe 403.

### `internal/daemon/api.go` — remote write proxy routes

**Analog:** remote read route registration (`api.go:161-164`)
```go
a.mux.HandleFunc("GET /api/files/remote/{sessionID}/list", a.handleRemoteFilesList)
a.mux.HandleFunc("GET /api/files/remote/{sessionID}/stat", a.handleRemoteFilesStat)
a.mux.HandleFunc("GET /api/files/remote/{sessionID}/read", a.handleRemoteFilesRead)
a.mux.HandleFunc("HEAD /api/files/remote/{sessionID}/read", a.handleRemoteFilesRead)
```
**Add 5 (CAP-10):** `PUT .../write`, `POST .../upload`, `DELETE .../delete`, `POST .../rename`, `POST .../mkdir` → handlers that call the (fixed) `proxyRemoteFiles`. Daemon socket routes (`api.go:149-153`) are auth-less loopback-trust — NO middleware, NO Origin check here.

---

### `internal/daemon/engine.go` (settings field + per-session map + load)

**`daemonSettings` analog** (`engine.go:93-102`):
```go
type daemonSettings struct {
	...
	FilesRead     *bool          `json:"filesRead,omitempty"`
	Plugins       PluginSettings `json:"plugins"`
	SchemaVersion int            `json:"schemaVersion"`
}
```
Add `FilesWrite bool json:"filesWrite,omitempty"` (plain bool, zero-value false — NOT `*bool` default-true like `FilesRead`; CAP-08 opt-in-for-all).

**`filesReadEnabled` analog** (`engine.go:514-518`) — write a `filesWriteEnabledFor(sessionID string) bool` that reads a per-session map under `e.mu.RLock()` (Inversion 2), NOT a global flag.

**`GetSessionWorkDir`** (`engine.go:502-506`) returns the EvalSymlinks-resolved cwd — reuse directly for the CAP-06 home-dir signal.

**`loadSettingsFromDisk` defaults-merge** (`engine.go:143-197`): the pre-`Unmarshal` literal at `:156-160` sets `FilesRead: &tr`. For `FilesWrite` (plain bool default false) the zero value already gives the opt-in default — do NOT pre-populate a true. The `needsUpgradeWrite := s.SchemaVersion < CurrentSchemaVersion` (`:189`) + `saveSettingsToDisk()` (`:195`) rewrite path fires automatically on the 3→4 bump.

---

### `internal/daemon/plugin_settings.go` (schemaVersion bump)

**Analog:** `plugin_settings.go:3-8`
```go
// v3.4 (Phase 118) adds the daemon-wide FilesRead flag and bumps to 3.
const CurrentSchemaVersion = 3
```
Bump to `4`; extend the doc comment with the Phase 124 / `FilesWrite` rationale.

---

### `internal/daemon/remote_files.go` (`proxyRemoteFiles` body fix — Inversion 3)

**Analog (the bug):** `remote_files.go:167-199`
```go
upstreamURL := strings.TrimRight(baseURL, "/") + "/api/files/" + op + "?" + q.Encode()
req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, nil) // <-- nil drops write body
```
**Fix:** forward `r.Body` for `PUT`/`POST`/`PATCH`, and copy the inbound request `Content-Type` header (the cap-token stripping at `:154-165` and response-header forwarding at `:188-199` are correct — KEEP them). Preserve the `?cap=` force-set + caller-cap strip (anti-smuggling, `:156-165`).

---

### `internal/daemon/engine_migration_test.go` (CAP-08 migration tests)

**Analog:** `TestSettingsMigration_FilesReadDefaultsTrue` (`:206-222`) and `TestSettingsMigration_FilesReadSchemaVersionRewrite` (`:267-288`).
```go
func TestSettingsMigration_FilesReadDefaultsTrue(t *testing.T) {
	dir := t.TempDir()
	copyV32FixtureToTempDir(t, dir)
	e := &SessionEngine{configDir: dir, cliPaths: make(map[string]string)}
	e.loadSettingsFromDisk(dir)
	if e.filesRead == nil { t.Fatalf(...) }
	if !*e.filesRead { t.Errorf(...) }
}
```
**Write `TestSettingsMigration_FilesWriteDefaultsFalse`:** same harness, **invert** the assertion to expect `FilesWrite == false` after loading a v3-fixture with no `filesWrite` key. Mirror the rewrite test (`:267`) to assert on-disk `schemaVersion == 4` after load.

---

### `internal/webserver/capability_test.go` (CAP-09 integration + static gate)

**Integration analog:** `TestRequireFilesRead` (`:445-568`) — `newHarness` mounts a sentinel under the wrapper on a one-off mux; `issueCapFor(t, ws, sid, perms)` mints tokens; assert 403 body contains the perm string and sentinel did not run on miss.
```go
mux.HandleFunc("/test/files", ws.requireFilesRead(func(w http.ResponseWriter, r *http.Request) {
	ran = true
	claims, ok := capability.ClaimsFromContext(r.Context())
	...
}))
```
**Write `TestRequireFilesWrite`:** for each of the 5 routes with its correct verb — cap `"read,write"` (no `files.write`) → 403; cap `"read,write,files.write"` → 2xx; plus mismatched-`Origin` header → 403 and absent-`Origin` → passes (SC#2).

**Static-grep gate analog:** `TestRequireCapability_UnchangedByPhase118` (`:575-602`)
```go
data, _ := os.ReadFile("capability_mw.go")
src := string(data)
idx := strings.Index(src, "func (ws *WebServer) requireCapability(")
... body := rest[:end+2]
if strings.Contains(body, "files.read") { t.Errorf(...) }
```
**Write `TestHasPerm_NoStringsContains_Write`:** scope to the write-path source files (`capability_mw.go`, `../../internal/daemon/api.go`, etc.) and fail any file containing `strings.Contains(...` together with `"files.write"` — forces `capability.HasPerm`.

---

### `internal/tui/files.go` (CAP-06 home-dir warning line — TUI parity, RELEASE-BLOCKING)

**Analog:** `renderFilesStatusLine` (`files.go:283-327`) — lipgloss styled, `truncate`/`truncateLeft` width-clamped, `Foreground(m.styles.X).Width(w).Render(...)`.
```go
return lipgloss.NewStyle().Foreground(m.styles.FgMuted).Width(w).Render(body)
```
**Add a warning line above the status line in `renderFilesTab`** (`files.go:331`+) using `m.styles.StatusWaiting` (amber, `styles.go:62` `#8c6c3e`/`#e0af68`). Content VERBATIM (UI-SPEC):
`⚠ Warning: cwd is $HOME — writes can affect dotfiles, SSH keys, and shell config. Protected files are blocked.`
The `⚠` glyph + literal `Warning:` carry the signal (colorblind-safe); amber is reinforcement only. Do NOT use bold-alone. Trigger condition mirrors the GUI banner (same session/cwd ⇒ both surfaces show — parity is release-blocking).

---

### `frontend/src/components/SessionSharePanel.tsx` (CAP-05 viewer opt-in)

**Analog:** existing link-row + `handleCopy` (`SessionSharePanel.tsx`):
```tsx
<div className="session-share-panel__link-row">
  <span className="session-share-panel__label">Full Access Link</span>
  <span className="session-share-panel__url" title={writeURL}>{writeURL}</span>
  ...
```
**Add** a `session-share-panel__write-optin` row ABOVE the Full Access Link row using the `settings-panel__toggle-row` markup (see shared pattern below). Label VERBATIM `Allow file editing`, default OFF, `role="switch"` + `aria-checked`. Toggle-ON opens an inline confirmation (reuse existing inline-confirm styling, no new modal) with the verbatim CAP-05 body; confirm → issued viewer cap carries `files.write`; cancel → revert OFF. Disabled (`opacity:0.6`, `aria-disabled`) unless the owner toggle (Surface 1) is ON. Reuse the `handleCopy` 1.5s transient feedback pattern for the `Saved` confirmation.

---

### `frontend/src/components/HomeDirWriteWarning.tsx` (NEW, CAP-06 GUI banner)

**Analog:** `webgl-recovery-banner` markup + `--shell-warning` modifier (`style.css:1794-1842`, `:1851-1856`).
```css
.webgl-recovery-banner {
  background: #16161e; border: 1px solid #292e42;
  border-left: 3px solid #7aa2f7; border-radius: 4px;
  padding: 12px 16px; margin-bottom: 24px; display: flex; ...
}
.webgl-recovery-banner--shell-warning { border-left: 3px solid #f7768e; ... } /* destructive red */
```
**Create** the banner with a NEW BEM modifier `--home-write-warning` setting `border-left: 3px solid #f59e0b` (amber, matches `local-network-banner__icon`; NOT destructive red — this is cautionary). Prefix message with a `⚠` icon span (color `#f59e0b`). Two lines, VERBATIM (UI-SPEC):
- Heading (13px/600): `Warning: writes can affect your home directory`
- Body (13px/400): `This session's working directory is your home folder. File writes here can modify dotfiles, SSH keys, and shell config (~/.zshrc, ~/.ssh, ~/.claude). Protected system files are always blocked.`
Reuse `webgl-recovery-banner__dismiss` (`XMarkIcon`, `:1817`). Standing caution — NOT timer-auto-dismissed; re-shows on re-enable. The `⚠` glyph + word `Warning:` carry the meaning (colorblind-safe); amber border is decoration. NO new hex (`#f59e0b` already in `local-network-banner`).

---

## Shared Patterns

### Permission check — whole-token `HasPerm`
**Source:** `internal/capability/capability.go:44-54`
**Apply to:** every `files.write` permission check (`requireFilesWrite`, cap mint).
```go
func HasPerm(perms, perm string) bool {
	if perms == "" || perm == "" { return false }
	for _, t := range strings.Split(perms, ",") {
		if t == perm { return true }
	}
	return false
}
```
NEVER `strings.Contains(perms, "files.write")` — false-positives on `"no-files.write"`. The static-grep gate enforces this.

### Capability minting — append to `Perms` string only
**Source:** `internal/daemon/api.go:1055-1060` + `capability.Sign` (`capability.go:59`)
**Apply to:** all cap issuance. The `Claims` struct field order is HMAC-load-bearing — never reorder. Only mutate the `Perms` string (`ownerPerms += ","+...`) before `Sign`.

### Toggle switch (GUI) — reuse verbatim
**Source:** `frontend/src/style.css:680-728` (`settings-panel__toggle-*`)
**Apply to:** Surface 1 (owner "Enable file writes") AND Surface 2 (viewer "Allow file editing").
```css
.settings-panel__toggle-row { display:flex; align-items:center; gap:10px; cursor:pointer; min-height:44px; }
.settings-panel__toggle-track { width:36px; height:20px; border-radius:10px; background:#16161e; border:1px solid #292e42; }
.settings-panel__toggle-row--checked .settings-panel__toggle-track { background:#7aa2f7; border-color:#7aa2f7; }
.settings-panel__toggle-row--checked .settings-panel__toggle-thumb { transform:translateX(16px); background:#1a1b26; }
```
Add `role="switch"` + `aria-checked` to the input (a11y). Thumb position (left=OFF / right=ON) is the primary colorblind-safe signal — position, not color. Do NOT re-spec the grandfathered `gap:10px` / `min-height:44px`.

### Settings defaults-merge migration
**Source:** `internal/daemon/engine.go:143-197`
**Apply to:** the schemaVersion 3→4 bump. Pre-populate defaults in the `daemonSettings` literal BEFORE `json.Unmarshal`; `needsUpgradeWrite` + `saveSettingsToDisk()` rewrite fires automatically. `FilesWrite` plain-bool zero value = false default (no pre-population needed, unlike `FilesRead`'s `&tr`).

### Home-dir signal — EvalSymlinks normalization (CAP-06)
**Source:** `internal/files/sandbox.go:96-106` (denylist)
**Apply to:** the server-side `homeDir bool` per-session computation (GUI + TUI both consume).
```go
home, _ := os.UserHomeDir()
if resolved, err := filepath.EvalSymlinks(home); err == nil { home = resolved }
// compare against GetSessionWorkDir(sid) which is ALREADY EvalSymlinks-resolved (engine.go:502)
```
Pitfall 4: comparing against a raw `os.UserHomeDir()` returns false on macOS (`/var` vs `/private/var`) — the warning never shows. Mirror the denylist's `EvalSymlinks(home)` exactly.

### Origin CSRF (write-only) — see Critical Inversion 1
**Source (to invert, NOT copy):** `internal/webserver/origin_mw.go:31-51`
**Apply to:** all 5 webserver write routes (state-changing verbs only). Daemon socket routes get NO Origin check (loopback-trust).

---

## No Analog Found

None. Every file has a direct in-tree precedent (Phase 118/119 read-side, Phase 88 Origin, Phase 81 banner-stack, Phase 94 settings-toggle CSS). The three "inversions" above are semantic divergences from a real analog, not missing analogs.

---

## Metadata

**Analog search scope:** `internal/capability/`, `internal/webserver/`, `internal/daemon/`, `internal/files/`, `internal/tui/`, `frontend/src/components/`, `frontend/src/style.css`
**Files scanned (read):** capability.go, capability_mw.go, origin_mw.go, server.go, api.go, engine.go, plugin_settings.go, remote_files.go, sandbox.go, engine_migration_test.go, capability_test.go, files.go, styles.go, style.css, SessionSharePanel.tsx
**Pattern extraction date:** 2026-06-14
