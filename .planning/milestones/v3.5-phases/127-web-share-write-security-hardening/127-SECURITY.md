---
phase: 127
slug: web-share-write-security-hardening
status: verified
threats_open: 0
asvs_level: 1
created: 2026-06-15
---

# Phase 127 — Security: Capability-Escalation Audit

> Per-phase security contract for the web-share write surface: capability-escalation
> audit, per-surface enforcement matrix, denylist threat model, and residual-risk register.
>
> **Audit scope:** all surfaces through which a caller could reach the five write
> endpoints (`PUT /api/files/write`, `POST /api/files/upload`, `DELETE /api/files/delete`,
> `POST /api/files/rename`, `POST /api/files/mkdir`) and the `HEAD /api/files/write`
> canWrite probe.  Every enforcement path is verified against live source.  No
> production code was changed as part of this audit — all enforcement was found correct.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Tailnet browser → web-share write routes | Untrusted browser must present a valid HMAC-signed `files.write` capability token AND a matching `Origin` header to reach any write endpoint. | File contents (PUT/POST), path metadata (DELETE/rename/mkdir). |
| Desktop Wails fetch → write routes | Trusted local caller (no `Origin` header). Capability token still required; loopback-trust applies to the socket, not to the capability verification on the webserver. | Same as above. |
| Caller → daemon Unix-socket / named-pipe write routes | In-process GUI/TUI only. No capability enforcement by design — loopback trust (WEB-01 precedent). This is **not** a web-share surface. | File contents; restricted to the same OS session. |
| Local daemon → remote peer (proxy) | Proxy strips any caller-supplied `?cap`, injects the deposited write token, and forwards the body. The remote peer's `requireFilesWrite` is the actual enforcer — defense in depth across the Tailscale network. | File contents forwarded as an opaque byte pipe. |
| Two concurrent writers → same target file | Optimistic-concurrency boundary. `If-Match` + atomic `temp+Sync+rename` + validator re-check inside `WriteFileAtomic` prevent torn writes. | Final file content must equal exactly one writer's complete payload. |

---

## Capability-Escalation Audit — Per-Surface Enforcement Matrix

The following table enumerates every surface that could theoretically reach a write
endpoint.  The audit verified each row against live source on 2026-06-14.

| Surface | Reachable without `files.write`? | Enforcement Path | Source Anchors | Verdict |
|---------|----------------------------------|-----------------|----------------|---------|
| Webserver `PUT/POST/DELETE/rename/mkdir` write routes | No | `requireFilesWrite` → `requireCapability` (HMAC verify + SID-scope + grant active + session enabled) → `HasPerm(claims.Perms, PermFilesWrite)` → `originAllowedForWrite` | `server.go:512-521`; `capability_mw.go` `requireFilesWrite`/`requireCapability` | **SECURE** |
| Webserver `HEAD /api/files/write` canWrite probe | No | Same `requireFilesWrite` wraps the HEAD handler; handler short-circuits after middleware on HEAD | `server.go:517`; `write.go:62-67`; `files_routes_test.go:267,286` | **SECURE** |
| Daemon Unix-socket / named-pipe write routes (`/api/files/remote/{sid}/…`) | **Yes — by design** | None. Loopback trust (WEB-01). Only in-process GUI/TUI reach the socket. | `remote_files.go:115-126`; `api.go:170` | **ACCEPTED RISK** (see Accepted Risks Log) |
| Remote proxy `/api/files/remote/{sid}/{write,…}` | No | Proxy strips caller `?cap` (case-insensitive), force-sets `session` from path, injects deposited token; **remote peer's** `requireFilesWrite` enforces cap + Origin | `remote_files.go:197-226` | **SECURE** |
| Cross-session (`files.write` token for session A used against session B path) | No | `requireCapability` rejects `claims.SID != pathID` with 403 | `capability_mw.go` `requireCapability` step 4 | **SECURE** — SID-scoped tokens |
| Token tampering / wrong perm string (e.g. `"no-files.write"`) | No | `HasPerm` uses whole-token `strings.Split`, **not** `strings.Contains` — `"no-files.write"` does not produce a `"files.write"` token | `capability.go:51` | **SECURE** — no substring bypass |
| Viewer (read-only) token default | N/A | Viewer token `Perms` is bare `"read"`. `files.write` is appended to the owner token ONLY when `filesWriteEnabledFor(sessionID)` returns true — opt-in, per-session | `api.go:1070-1079` | **SECURE** — never default-on |

### `requireFilesWrite` Enforcement Detail

`requireFilesWrite` (capability_mw.go:147-170) is a layered wrapper:

1. **`requireCapability` (outer wrap):** verifies the HMAC-signed JWT (`cap=` query param),
   rejects missing/nil signing key, collapses all token failures to a single generic 401
   (T-87-08 information-disclosure defense), enforces SID scope (`claims.SID == pathID`),
   checks grant is still active, and confirms the session is web-enabled.  Attaches verified
   `Claims` to the request context via `capability.WithClaims`.
2. **`HasPerm` check:** extracts `Claims` from context, splits `claims.Perms` on `,` using
   whole-token comparison (`capability.go:51`).  A perm string containing `"no-files.write"`
   or any other non-exact variant does NOT match.
3. **`originAllowedForWrite` (CSRF check):** see "CSRF Origin Inversion" below.

### CSRF Origin Inversion

`originAllowedForWrite` (capability_mw.go:187-198) is the **INVERSE** of
`requireAllowedOrigin` (origin_mw.go:31, used for WebSocket upgrades):

| Caller type | Origin header present? | Result |
|-------------|----------------------|--------|
| Desktop Wails `fetch()` | No (Wails sends none) | Pass vacuously — trusted local caller |
| Web-share browser (correct origin) | Yes, matches `ws.BaseURL()` | Pass |
| Web-share browser (wrong origin / CSRF attempt) | Yes, does not match `ws.BaseURL()` | 403 Forbidden |
| Any caller with empty `BaseURL` and a present Origin | Yes | 403 Forbidden (fail-closed: `allowed != "" && origin == allowed`) |

The ordering in `requireFilesWrite` is load-bearing:
`requireCapability` (401) → `HasPerm` (403 informative) → `originAllowedForWrite` (403 generic).
This ensures a missing-perm request gets the informative body before Origin is checked.

---

## Denylist Threat Model

`denylistCheck` (`sandbox.go:96-148`) is invoked from all four write methods:
`WriteFileAtomic` (`sandbox.go:219`), `Rename` (both src + dest, `sandbox.go:329,332`),
`Mkdir`/`MkdirAll` (`sandbox.go:352,372`), `Delete` (`sandbox.go:391`).  It is the
single security chokepoint for the "protected system file" policy.

**Protected set (post Phase 127-01 hardening):**
- Shell RC files: `.bashrc`, `.zshrc`, `.profile`, `.bash_profile`, `.zprofile`, `.zshenv`, `.bash_login` (case-insensitive via `strings.ToLower` after Plan 01).
- Directory prefixes: `.ssh/`, `.claude/`, `.config/agenthub/`, and the platform-correct `os.UserConfigDir()`-derived prefix (e.g. `Library/Application Support/agenthub/` on macOS).

**Phase 127-01 fixes applied to `denylistCheck`:**
1. **macOS daemon-config-dir hole (GAP 1):** the literal `.config/agenthub/` missed
   `~/Library/Application Support/agenthub/` on macOS.  Fixed by deriving the
   platform config dir from `os.UserConfigDir()` and computing its `$HOME`-relative
   form.  Test: `TestDenylist_DaemonConfigDir`.
2. **Case-insensitivity (GAP 2):** base name and relative path are now lowercased
   before comparison.  `.BASHRC` / `.SSH/authorized_keys` now match.
   Test: case-variation cases in the denylist test suite.

**Fail-closed symlink handling:** `denylistCheck` uses `filepath.EvalSymlinks` on the
*parent* directory before computing `filepath.Rel(home, canonAbs)` (sandbox.go:114-122).
A symlink that points outside `$HOME` causes `Rel` to produce a `..`-prefixed path,
and the function returns early with `nil` (denylist does not apply).  The `os.OpenRoot`
boundary is the primary escape defense; the denylist is a secondary layer.

---

## Threat Register (STRIDE)

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-127-01 | Tampering / Elevation | `denylistCheck` — macOS daemon config dir not protected | mitigate | `os.UserConfigDir()`-derived prefix added to denylist (Plan 01). `TestDenylist_DaemonConfigDir` confirms. | closed |
| T-127-02 | Tampering / Elevation | `denylistCheck` — case-sensitive comparison bypassed on case-insensitive volumes | mitigate | ASCII `strings.ToLower` applied to base name and path before comparison (Plan 01). Case-variation tests confirm. | closed |
| T-127-03 | Tampering / Elevation | Write-path symlink escape (write to target outside sandbox root via symlink) | mitigate | `os.OpenRoot` + native `root.*` methods reject escaping symlinks atomically. `TestSandbox_WritePathSymlinkEscapeBlocked` (Plan 02) confirms. | closed |
| T-127-04 | Tampering | Upload `../` filename injection (zip-slip style) | mitigate | `filepath.Base` + name guard in `Upload` (`write.go:158-168`). Pre-existing. | closed |
| T-127-05 | DoS | Over-cap upload (>50 MiB memory exhaustion) | mitigate | `MaxBytesReader` before `ParseMultipartForm` (`write.go:131`). Returns 413 with `errors.As`. Pre-existing. | closed |
| T-127-06 | Spoofing / Tampering | CSRF: browser with a cross-origin `Origin` header reaches write routes with valid cap | mitigate | `originAllowedForWrite` byte-matches `Origin` against `ws.BaseURL()`. Unit-tested all 5 routes (`capability_test.go:690-739`). e2e scenario added Plan 04. | closed |
| T-127-07 | Elevation | Capability escalation: token lacking `files.write` reaches any write surface | mitigate | Per-surface enforcement matrix above; all paths verified correct. `TestRequireFilesWrite_*` suite. | closed |
| T-127-08 | Tampering / Data Integrity | Concurrent-write lost update / torn file / leftover temp | mitigate | `If-Match` 412 + atomic `temp+Sync+rename` + validator re-check inside `WriteFileAtomic`. `TestWrite_TwoWritersIfMatchRace` + `TestWrite_InterruptedWritePreservesOriginal` (Plan 03, SEC-05) assert exactly-one-winner and original-preserved. | closed |
| T-127-09 | Information Disclosure / Elevation | Daemon loopback socket reaches write routes without a capability token | accept | Loopback trust (WEB-01). Only in-process GUI/TUI reach the socket. Not a web-share surface. (See Accepted Risks Log.) | closed |
| T-127-10 | Tampering | `stat`→`rename` TOCTOU residual in `If-Match` re-check | accept | Microscopic local-FS window; not eliminable without kernel `renameat2` (unavailable in Go stdlib). Documented residual (sandbox.go:228-238, re-check at sandbox.go:292-311). | closed |

---

## Accepted Risks Log

| Risk ID | Threat Ref | Description | Rationale | Accepted By | Date |
|---------|------------|-------------|-----------|-------------|------|
| AR-127-01 | T-127-09 | Daemon Unix-socket / named-pipe write routes have no capability enforcement | The daemon socket is a loopback-trust boundary (WEB-01 precedent, established in v3.4 read surface). Only in-process GUI and TUI components communicate over it — it is not reachable from the Tailscale web-share network. Adding capability enforcement here would require every GUI call to carry a token, adding complexity with no security benefit. The web-share surface (all webserver routes) enforces `requireFilesWrite` independently. | Plan author | 2026-06-15 |
| AR-127-02 | T-127-10 | Microscopic `stat`→`rename` TOCTOU window in `If-Match` validator re-check | `WriteFileAtomic` narrows the optimistic-concurrency window to `root.Stat(cleaned)` → `root.Rename(tmp, cleaned)`. A second concurrent writer could land between the two calls (stat fires → other writer commits → our rename executes, overwriting it). Eliminating this window requires `renameat2 RENAME_NOREPLACE` (Linux 3.15+) or `RENAME_EXCL` (macOS 10.12+), neither of which is available in Go's stdlib `os` package. The window is microscopic on a local filesystem and the `If-Match` pre-check in `Handler.Write` (`write.go:77-104`) already catches the vast majority of races before `WriteFileAtomic` is called. | Plan author | 2026-06-15 |
| AR-127-03 | (residual) | NFC/NFD Unicode denylist bypass | All protected filenames in `denylistCheck` (`.bashrc`, `.ssh`, `.claude`, `.config`, etc.) are ASCII-only. A Unicode NFC/NFD decomposition attack requires a protected name containing a composable non-ASCII character, which is not the case. `golang.org/x/text` is available in `go.mod` (v0.37.0 indirect) and could add NFC normalization at zero new-dependency cost, but the threat model does not justify it: the case-fold fix (AR-127-01 precursor) closes the realistic macOS/Windows bypass vector. Residual risk rated LOW. | Plan author | 2026-06-15 |

---

## ASVS L1 Mapping

| ASVS Category | Control | How It Applies |
|---------------|---------|----------------|
| V1 — Architecture | Sandbox boundary documents trust zones | `os.OpenRoot` is the terminal security boundary; web-share vs daemon socket documented as separate trust tiers |
| V2 — Authentication | V2.1 / Credential verification | HMAC-signed JWT capability tokens (`requireCapability`); SID + grant active + session enabled checks |
| V4 — Access Control | V4.1 / AC policy enforcement | `files.write` perm gate (`HasPerm` whole-token); cross-session SID scoping; opt-in token issuance (`filesWriteEnabledFor`) |
| V5 — Validation | V5.1 / Input validation | `validateRelativePath` (traversal, ADS, device names, UNC, null bytes); `filepath.Base` on upload filename; denylist |
| V12 — Files and Resources | V12.1 / File upload | `MaxBytesReader` 50 MiB cap before `ParseMultipartForm`; atomic `temp+Sync+rename`; no archive extraction (no zip-slip surface) |
| V13 — API and Web Service | V13.1 / CSRF | `originAllowedForWrite` origin byte-match on all state-changing verbs; method-prefix 405 routing (Go 1.22+ mux) |
| V6 — Cryptography | V6.2 / Algorithm strength | `crypto/rand` for temp-file suffix (let-it-crash on CSPRNG failure, `sandbox.go:268`); no hand-rolled crypto |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-15 | 10 | 10 | 0 | Plan author (127-03) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter
- [x] CSRF Origin inversion explicitly documented
- [x] Daemon loopback socket explicitly accepted (not a finding)
- [x] Residual TOCTOU window explicitly accepted (not eliminable in Go stdlib)
- [x] NFC/NFD residual rated LOW (all protected names ASCII)

**Approval:** verified 2026-06-15
