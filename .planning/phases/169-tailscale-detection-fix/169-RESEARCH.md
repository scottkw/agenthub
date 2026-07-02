# Phase 169: Tailscale Detection Fix (RE-SCOPED) - Research

**Researched:** 2026-07-02
**Domain:** Go / macOS local-IPC permission semantics (tailscaled `sameuserproof` gate), Wails frontend status UI
**Confidence:** HIGH (make-or-break question resolved against vendored `tailscale.com@v1.98.3` source + live macsys filesystem)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Revert 169-01's CLI-fallback code in full — remove `cliStatusFunc` type, `runTailscaleStatusCLI`, `realCLIStatusFunc`, the `cli` parameter on `checkHealth`, the fallback branch inside the `err != nil` arm, and the three fallback unit tests (`TestCheckHealth_CLIFallback_Success/_NotInvokedOnSDKSuccess/_AlsoFails`). Restore `checkHealth`'s pre-169-01 signature and the `CheckHealth` / `CheckHealthWithCustomPath` wrappers. Reverting is a prerequisite, not a separate cleanup.
- **D-02:** When the SDK read fails, distinguish **"permission-limited"** (daemon running, status unreadable) from **"daemon genuinely down."** Detection mechanism is research-gated (resolved below — file-probe, NOT error-classification).
- **D-03:** Represent the new condition with a dedicated, additive field on `TailscaleHealth` (e.g. `PermissionLimited bool`, exact name/shape planner's call). **Do NOT set `Connected=true`** in this state (hard constraint). `DaemonUp` semantics in this state is a planner decision; note `internal/daemon/process.go` gates restart on `Connected && HasCerts && IP != ""`, so a permission-limited node never triggers a restart regardless.
- **D-04:** Permission-limited detection is **macOS-macsys-specific**. Guard behind `runtime.GOOS == "darwin"` plus the file check. Windows/Linux and the macOS App-Store distribution keep today's behavior.
- **D-05:** Surface the permission-limited state in the `SettingsTab` Tailscale status block as a **distinct** message/row (not "Daemon Stopped" / "Not Connected") with actionable guidance: grant this account admin, or install the Homebrew `tailscale` build. Exact copy/visual treatment is Claude's discretion but must be accurate (never imply Connected). Extend the existing 4-step cascade at `SettingsTab.tsx:737+`.
- **D-06:** The SDK-success path (Steps 3–4 of `checkHealth`) is **byte-for-byte unchanged** (SC2). The fix only touches the `err != nil` arm and the struct/UI.

### Claude's Discretion
- Exact detection mechanism (file-stat + EACCES vs. SDK-error classification), pending research — **resolved below: file-probe.**
- Exact new field name/shape on `TailscaleHealth` and the `DaemonUp` semantics in the permission-limited state.
- Exact UI copy and placement of the guidance.
- Whether detection lives in a small injectable helper (mirroring the `statusFunc`/`prefsFunc` test-seam idiom) so it can be unit-tested without a live macsys install.

### Deferred Ideas (OUT OF SCOPE)
- Privileged helper / LaunchDaemon to read `sameuserproof` (the mechanism the real macsys GUI uses).
- Applying honest detection / guidance to the Funnel-path `StatusWithoutPeers` call in `server.go:620`.
- CLI prefs fallback for `AcceptDNS` on non-admin accounts.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FIX-05 | Honest, permission-aware Tailscale detection: on non-admin macOS macsys accounts where `sameuserproof` is `0640 root:admin`, distinguish "daemon running but status unreadable" from "daemon down", and surface it accurately in Settings with actionable guidance — never a false Connected. | §Decision Verdict (file-probe, not error-classification), §Standard Stack (stdlib probe helper), §Architecture Patterns (injectable `permProbeFunc` seam), §Code Examples (detection helper + UI cascade), §Common Pitfalls (stale file, os.Stat-vs-os.Open, admin-machine can't repro). |
</phase_requirements>

## Summary

The re-scope hinges on one make-or-break question (Research Need #1): **is the SDK read error classifiable** so we can tell "permission-blocked macsys" apart from "daemon down"? Answer, verified against the pinned `tailscale.com@v1.98.3` source: **No.** The `EACCES` from reading the `0640 root:admin` `sameuserproof` file is **swallowed twice** before it reaches `StatusWithoutPeers`' caller — once in `portAndTokenFromSameUserProof` (which replaces any error with the generic `ErrTokenNotFound`), and again when the dialer falls back to a non-existent unix socket whose own dial error is what actually surfaces. The final error `checkHealth` sees on a permission-blocked macsys node is **byte-indistinguishable** from a genuinely-down daemon. Error-classification (candidate *a*) is therefore not viable.

That decides it in favor of the **`sameuserproof` file-probe** (candidate *b*). Verified live on this machine's real macsys install: `/Library/Tailscale` is `drwxr-xr-x root:wheel` (0755 — world-listable, so `filepath.Glob` works for any user), it contains `sameuserproof-56771` at `-rw-r----- root:admin` (0640) and an `ipnport` symlink (`lrwxr-xr-x`, world-readable) pointing at the port. The critical syscall distinction the CONTEXT flagged is real: **`os.Stat` succeeds** on the 0640 file (stat needs only directory-traverse permission, not file-read permission) and therefore does NOT distinguish the states; **`os.Open` (O_RDONLY) is the syscall that returns `EACCES`** for a non-admin caller. Detection must attempt an *open*, not a stat.

The clean mechanism: on darwin, `filepath.Glob("/Library/Tailscale/sameuserproof-*")`; for each match `os.Open` it — if it opens successfully we're not permission-limited (the SDK would have worked); if it returns `fs.ErrPermission` (`EACCES`) the macsys daemon wrote the file but this account can't read it → permission-limited. Because the file can be stale after a daemon stop, an optional but recommended honesty enhancement confirms liveness by `os.Readlink`-ing `ipnport` and TCP-dialing `127.0.0.1:<port>` (the localapi listener accepts the connection without a token; only HTTP auth needs it). This yields a truly honest "daemon running but unreadable" and eliminates the stale-file false positive.

**Primary recommendation:** Revert the 169-01 CLI fallback (D-01), add an injectable `permProbeFunc` seam to `checkHealth` mirroring `statusFunc`/`prefsFunc`, whose real darwin implementation does glob → `os.Open` → `EACCES` detection (with the optional `ipnport` + TCP liveness confirmation), set a new additive `PermissionLimited bool` on `TailscaleHealth` (Connected stays false), and add a distinct guidance row/message to the SettingsTab cascade. Scope strictly to `runtime.GOOS == "darwin"` + macsys file signature. No new dependencies; stdlib only.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Classify SDK-read failure (permission-limited vs. daemon-down) | API / Backend (Go, `internal/webserver/tailscale.go`) | — | The permission signature is an OS filesystem fact readable only server-side; must not leak into the frontend as a probe. |
| Filesystem probe of `/Library/Tailscale` | API / Backend (Go stdlib `os`/`filepath`) | — | Privileged-path read attempt belongs next to the existing daemon probe, behind an injectable seam for testability. |
| Represent the new state on the wire | API / Backend (`TailscaleHealth` struct + JSON tag) | Frontend (mirrored TS interface) | Additive serialized field; the frontend is a pure consumer. |
| Surface guidance to the user | Frontend (SettingsTab cascade) | — | Presentation-only; extends the existing 4-step status cascade. |
| Web-server upgrade gating | API / Backend (`internal/daemon/process.go`) | — | Unchanged: gate is `Connected && HasCerts && IP != ""`; a permission-limited node (Connected=false) never triggers a restart. |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `os` / `path/filepath` / `io/fs` | go 1.26.4 (repo toolchain) | Glob + `os.Open` + `errors.Is(err, fs.ErrPermission)` for the EACCES probe | Already the idiom in this file; no dependency risk `[VERIFIED: go version on host]` |
| Go stdlib `net` | go 1.26.4 | Optional `net.DialTimeout("tcp", "127.0.0.1:<port>", …)` liveness check | Mirrors tailscale's own `checkConn` staleness guard in `readMacsysSameUserProof` `[CITED: safesocket_darwin.go:295-303]` |
| `tailscale.com` | v1.98.3 (pinned in `go.mod`) | `local.Client.StatusWithoutPeers` (unchanged), `ipnstate.Status` (unchanged) | Already vendored; this phase adds NO new tailscale API calls `[VERIFIED: go.mod]` |

### Supporting
None — this phase is a revert + additive stdlib probe + additive struct field + frontend copy. No new imports beyond possibly `errors`, `io/fs`, and (if liveness check adopted) `net`, `strconv`, `time`, all stdlib.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| File-probe (glob + `os.Open` EACCES) | SDK-error string/type classification | **Rejected** — the EACCES is swallowed and overwritten by a unix-socket dial error; not classifiable. See §Decision Verdict. |
| `os.Open` for EACCES | `os.Stat` + mode-bit inspection | `os.Stat` succeeds on the 0640 file (no distinguishing error) — you'd have to hand-compute "am I in the owning group?" from `Sys().(*syscall.Stat_t)` uid/gid vs. `os.Getgroups()`. Far more fragile than letting the kernel answer via an actual open. |
| Optional `ipnport` + TCP liveness | File-presence alone | File-presence-only can false-positive on a stale `sameuserproof` left after a daemon stop. TCP liveness is the honest confirmation; planner's call whether SC "distinct from daemon-down" demands it. |

**Installation:** None. No `go get`. No package additions.

**Version verification:** `go version` → `go1.26.4 darwin/arm64` `[VERIFIED: host]`. `tailscale.com v1.98.3` present in module cache at `/Users/ken/go/pkg/mod/tailscale.com@v1.98.3/` and matches `go.mod` `[VERIFIED: CR-01 + module cache]`.

## Package Legitimacy Audit

> Not applicable — this phase installs **no external packages**. It reverts prior code and adds Go-standard-library calls plus one additive struct field. No `go get`, no `npm install`.

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Decision Verdict: File-Probe, NOT Error-Classification

This is the phase's pivotal decision. The evidence (traced through the pinned source) is unambiguous.

### Why error-classification (candidate a) is DEAD

The call chain for `local.Client.StatusWithoutPeers` on a non-admin macsys account:

1. `StatusWithoutPeers` → `DoLocalRequest` → `defaultDialer` `[CITED: client/local/local.go:112-127]`.
2. `defaultDialer` calls `safesocket.LocalTCPPortAndToken()` → `localTCPPortAndTokenDarwin()` `[CITED: safesocket_darwin.go:69-85]`.
3. Not a GUI app → `portAndTokenFromSameUserProof()` `[CITED: safesocket_darwin.go:77-81]`.
4. `portAndTokenFromSameUserProof` tries `readMacosSameUserProof()` (App-Store `lsof` path — fails, returns `ErrTokenNotFound`), then `readMacsysSameUserProof()` `[CITED: safesocket_darwin.go:346-361]`.
5. `readMacsysSameUserProof` does `os.Readlink(ipnport)` (**succeeds** — symlink is world-readable), then `os.ReadFile("sameuserproof-<port>")` → **`EACCES`**, returns `(0,"",EACCES)` `[CITED: safesocket_darwin.go:277-306]`.
6. **First swallow:** `portAndTokenFromSameUserProof` only checks `err == nil`; on any error it discards `EACCES` and returns the generic `ErrTokenNotFound` `[CITED: safesocket_darwin.go:352-360]`.
7. **Second swallow:** back in `defaultDialer`, `LocalTCPPortAndToken` returned an error, so it falls through to `safesocket.ConnectContext(ctx, lc.socket())` — dialing `paths.DefaultTailscaledSocket()`, which **does not exist** on a macsys install (there is no plain unix socket — confirmed live: `/Library/Tailscale` has only the sameuserproof file + `ipnport` symlink). That dial's own error (no such file / connection refused) is what finally surfaces `[CITED: client/local/local.go:126]`.

The `EACCES` never reaches `checkHealth`. The error it sees is a unix-socket dial failure — **identical** to what a genuinely-stopped daemon produces. Classification by errno, type, or message is impossible. `[VERIFIED: tailscale.com@v1.98.3 source trace]`

### Why file-probe (candidate b) works — live evidence

On this machine's real macsys install `[VERIFIED: live filesystem, 2026-07-02]`:

```
drwxr-xr-x root:wheel  /Library/Tailscale                  # 0755 — world-listable → Glob works for any user
lrwxr-xr-x root:wheel  /Library/Tailscale/ipnport -> 56771 # world-readable symlink → Readlink works
-rw-r----- root:admin  /Library/Tailscale/sameuserproof-56771  # 0640 → os.Open EACCES for non-admin
```

- **Glob is safe:** the directory is 0755, so `filepath.Glob("/Library/Tailscale/sameuserproof-*")` lists entries for a non-admin user (no directory-read EACCES landmine).
- **`os.Open` distinguishes, `os.Stat` does not:** `os.Stat` on the 0640 file **succeeds** for any user (stat needs only parent-dir traverse). `os.Open`/`os.OpenFile(O_RDONLY)` **fails with `EACCES`** for a non-admin caller. Confirmed by construction: this account (`ken`) IS in the `admin` group and CAN read the file — which is exactly why the bug does not reproduce here. `[VERIFIED: id -Gn shows admin; cat succeeds]`
- **macsys filename shape:** for the macsys system extension, `initSameUserProofToken` writes `sameuserproof-<port>` (port only, no token in the name) at perm `0640`, then `Fchown(fd, 0, 80 /*admin*/)` `[CITED: safesocket_darwin.go:237-265]`. The App-Store variant instead uses `lsof` against an `IPNExtension`-owned file under the user's container (`readMacosSameUserProof`) and never writes to `/Library/Tailscale` — so the `/Library/Tailscale/sameuserproof-*` signature is **macsys-specific**, satisfying D-04 and IN-02.

### Recommended mechanism (hybrid)

Primary (meets SC, minimal): glob `sameuserproof-*` → `os.Open` each → if any returns `fs.ErrPermission` ⇒ **permission-limited**.

Recommended enhancement (honest liveness, resolves stale-file false positive): after detecting the EACCES signature, `os.Readlink("/Library/Tailscale/ipnport")` → `strconv.Atoi` → `net.DialTimeout("tcp", "127.0.0.1:"+port, 1s)`. Dial success ⇒ daemon genuinely running but unreadable (report permission-limited). Dial failure ⇒ stale file, daemon actually down (report daemon-down, today's behavior). This mirrors tailscale's own `checkConn` staleness guard `[CITED: safesocket_darwin.go:295-303]`.

## Architecture Patterns

### System Architecture Diagram

```
                 checkHealth(ctx, statusFn, customPath, prefsFn, permProbeFn)
                              │
              ┌───────────────┴───────────────┐
              │ Step 1: detectTailscaleBinary  │  binary=="" ─► BinaryFound=false, return
              └───────────────┬───────────────┘
                              │ BinaryFound=true
              ┌───────────────▼───────────────┐
              │ Step 2: status, err := fn(ctx) │  (SDK StatusWithoutPeers — UNCHANGED)
              └───────┬───────────────┬────────┘
                err==nil│              │err!=nil  ◄── the ONLY arm this phase edits
       ┌──────────────▼──────┐   ┌────▼─────────────────────────────────────┐
       │ Step 3: Connected =  │   │ permProbeFn()  (NEW, nullable, darwin)    │
       │  BackendState==Running│   │   glob /Library/Tailscale/sameuserproof-*│
       │ Step 4: certs + DNS   │   │   os.Open → EACCES ? ──► permission-      │
       │ (UNCHANGED, D-06)     │   │   [opt] ipnport→TCP dial confirms live    │
       └──────────┬────────────┘   └────┬──────────────────────┬─────────────┘
                  │                       │permission-limited     │not limited
                  │              ┌────────▼─────────┐   ┌─────────▼──────────┐
                  │              │ PermissionLimited │   │ today's not-       │
                  │              │ = true            │   │ connected fallthru │
                  │              │ Connected = false │   │ (DaemonUp=false)   │
                  │              └────────┬─────────┘   └─────────┬──────────┘
                  └───────────────────────┴───────────────────────┘
                                          │
                            TailscaleHealth (JSON) ─► Wails RPC ─► SettingsTab cascade
```

The file-to-implementation mapping (which function edits which file) is in the Component Responsibilities table below, not the diagram.

### Component Responsibilities
| File | Change |
|------|--------|
| `internal/webserver/tailscale.go` | Revert CLI-fallback code (D-01); add `permProbeFunc` type + `realPermProbe()`; add `PermissionLimited` field (D-03); wire the new arm inside `err != nil` (D-06 keeps Steps 3-4 untouched); update `CheckHealth`/`CheckHealthWithCustomPath` wrappers. |
| `internal/webserver/tailscale_test.go` | Remove the 3 CLI-fallback tests (D-01); update every existing `checkHealth(...)` call site to the new signature; add permission-limited unit tests via injected `permProbeFunc` fakes. |
| `frontend/src/components/SettingsTab.tsx` | Add `permissionLimited: boolean` to the `tailscaleHealth` interface (lines 51-60); add a distinct status label/text/description branch (lines 295-309, 735-751) + optional diagnostics row (752-786) with guidance copy (D-05). |
| `TESTING.md` | Update traceability rows for removed/added test files; tighten M-45 wording to "macsys (Standalone signed installer)" per IN-02. |

### Pattern 1: Injectable probe seam (mirror `statusFunc`/`prefsFunc`)
**What:** A nullable function-typed parameter that isolates real filesystem access from branch logic, so unit tests inject fakes.
**When to use:** Any detection helper that touches privileged OS state you cannot recreate in CI (a `0640 root:admin` file cannot be constructed by an unprivileged test).
**Example:**
```go
// Source: pattern from internal/webserver/tailscale.go statusFunc/prefsFunc (this repo)
// permProbeFunc reports whether the local tailscaled is a macsys install whose
// status is unreadable due to OS permissions (daemon running, EACCES on sameuserproof).
// Returns false on non-darwin, non-macsys, or genuinely-down daemon.
type permProbeFunc func(ctx context.Context) bool
```

### Anti-Patterns to Avoid
- **Classifying the SDK error string:** dead on arrival — the EACCES is swallowed (see §Decision Verdict). Never `strings.Contains(err.Error(), "permission")` here; the error is a unix-socket dial failure.
- **Using `os.Stat` to detect EACCES:** stat succeeds on the 0640 file; it cannot distinguish permission-limited from readable. Must `os.Open`.
- **Setting `Connected=true` or `DaemonUp=true` speculatively:** hard SC violation. Permission-limited must never imply Connected. `DaemonUp` semantics is a planner decision, but the daemon-restart gate (`Connected && HasCerts && IP`) is safe either way.
- **Running the probe on all platforms / App-Store:** must be `runtime.GOOS == "darwin"` + the `/Library/Tailscale/sameuserproof-*` macsys signature only (D-04, IN-02).
- **File-presence-only without liveness:** a stale `sameuserproof` after a daemon stop would false-positive as permission-limited. Prefer the `ipnport` + TCP dial confirmation.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detect "can't read this file due to perms" | uid/gid math against `os.Getgroups()` from `syscall.Stat_t` | `os.Open` + `errors.Is(err, fs.ErrPermission)` | The kernel already answers the exact question; group-math is fragile across supplementary-group and ACL edge cases. |
| Confirm daemon liveness | Re-implement the localapi handshake | `net.DialTimeout` to the `ipnport` port | Tailscale itself only TCP-dials to test staleness (`checkConn`); a plain connect is sufficient and needs no token. |
| Reconstruct macsys vs App-Store | New heuristics | The `/Library/Tailscale/sameuserproof-*` path IS the macsys discriminator | App-Store writes to a per-user container via `IPNExtension`, never `/Library/Tailscale` (`readMacosSameUserProof`). |

**Key insight:** every "clever" detection alternative here re-derives something the vendored tailscale source or the kernel already tells you directly.

## Runtime State Inventory

> This is a code + UI change, not a rename/migration. No stored data, service config, OS-registered state, secrets, or build artifacts are keyed on any renamed string.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no datastore keys involved. | none |
| Live service config | Reads (does not write) `/Library/Tailscale/sameuserproof-*` and `ipnport`, owned by tailscaled. AgentHub only probes. | none (read-only) |
| OS-registered state | None. | none |
| Secrets/env vars | None. The sameuserproof token is never read by AgentHub (that's the whole point — we can't and don't). | none |
| Build artifacts | The 169-01 CLI-fallback code is stale-on-revert; `go build` after D-01 revert produces the corrected binary. | rebuild via normal `wails build -tags wailsassets` |

## Common Pitfalls

### Pitfall 1: The fix cannot be verified on this (or any admin) machine
**What goes wrong:** A developer runs the built app on their Mac, sees Tailscale detection working, and declares the bug fixed.
**Why it happens:** The bug only manifests for accounts NOT in the `admin` group. This machine's `ken` account IS in `admin` (verified: `id -Gn` → `admin`), so `os.Open` on the 0640 file succeeds and the SDK path works normally.
**How to avoid:** M-45 must be run on a genuine **non-admin** macOS account with a **macsys** (Standalone signed-installer) Tailscale install. This is a manual checklist item (`human_judgment: true`), unavoidable — flag it prominently in the plan as the acceptance gate, not a formality.
**Warning signs:** A "verified" claim from a machine where `id -Gn | grep admin` matches.

### Pitfall 2: Stale `sameuserproof` file after daemon stop
**What goes wrong:** Reporting permission-limited when tailscaled has actually stopped but left its file behind.
**Why it happens:** `initSameUserProofToken` removes old files on *startup*; there is no shutdown cleanup in the vendored source, so a file can outlive the daemon.
**How to avoid:** Add the `ipnport` → `net.DialTimeout` liveness confirmation. Dial failure ⇒ report daemon-down, not permission-limited.
**Warning signs:** UI shows "permission-limited" immediately after the user quit Tailscale.

### Pitfall 3: `os.Stat` chosen over `os.Open`
**What goes wrong:** Probe never detects EACCES; every macsys node (admin or not) looks the same.
**Why it happens:** `stat()` needs only directory-traverse permission and returns metadata for a 0640 file without error. Only an actual `open()` for read triggers the permission check.
**How to avoid:** Use `os.Open`/`os.OpenFile(path, os.O_RDONLY, 0)` and `errors.Is(err, fs.ErrPermission)`.
**Warning signs:** Unit test with a locally-created unreadable file "passes" but real macsys never trips the branch.

### Pitfall 4: TCC / sandbox interference reading `/Library/Tailscale`
**What goes wrong:** On some hardened configs, reads under `/Library` could theoretically be mediated.
**Why it happens:** macOS TCC generally does not gate `/Library/Tailscale` (it is not a TCC-protected location like `~/Desktop`, `~/Documents`, or removable volumes), and the directory is 0755. Verified: listing + readlink + stat all succeed for a normal user here.
**How to avoid:** Treat any *unexpected* error from Glob/Open (not `fs.ErrPermission`, not `fs.ErrNotExist`) as "unknown → fall through to today's daemon-down behavior" — never as permission-limited. Log it at `slog.Debug` for future diagnosis (addresses review IN-01's visibility gap).
**Warning signs:** Neither EACCES nor ENOENT but some other errno.

### Pitfall 5: Forgetting to update existing `checkHealth` call sites after signature change
**What goes wrong:** Test file won't compile.
**Why it happens:** Every existing test passes positional args including the (removed) `cli` param and (new) `permProbeFunc`. D-01 removes one param; D-02 adds one.
**How to avoid:** Grep all `checkHealth(` call sites; the 8+ existing tests in `tailscale_test.go` each need the arg list updated. Keep the injection-idiom order consistent.

## Code Examples

### Detection helper with injectable seam (real impl + test fake)
```go
// Source: composed from tailscale.com@v1.98.3 safesocket_darwin.go semantics + this repo's statusFunc idiom
import (
	"context"
	"errors"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// permProbeFunc reports whether tailscaled is a macsys install that is running
// but whose status is unreadable to this OS account (0640 root:admin sameuserproof).
type permProbeFunc func(ctx context.Context) bool

const macsysSharedDir = "/Library/Tailscale"

func realPermProbe() permProbeFunc {
	return func(ctx context.Context) bool {
		if runtime.GOOS != "darwin" {
			return false
		}
		matches, err := filepath.Glob(filepath.Join(macsysSharedDir, "sameuserproof-*"))
		if err != nil || len(matches) == 0 {
			return false // not a macsys install, or dir unlistable → today's behavior
		}
		blocked := false
		for _, m := range matches {
			f, oerr := os.Open(m)
			if oerr == nil {
				f.Close()
				return false // readable → SDK would have worked; not permission-limited
			}
			if errors.Is(oerr, fs.ErrPermission) {
				blocked = true
			}
		}
		if !blocked {
			return false
		}
		// Recommended liveness confirmation (resolves stale-file false positive).
		return macsysDaemonAlive(ctx)
	}
}

func macsysDaemonAlive(ctx context.Context) bool {
	portStr, err := os.Readlink(filepath.Join(macsysSharedDir, "ipnport"))
	if err != nil {
		return false
	}
	if _, err := strconv.Atoi(portStr); err != nil {
		return false
	}
	d := net.Dialer{Timeout: time.Second}
	conn, err := d.DialContext(ctx, "tcp", "127.0.0.1:"+portStr)
	if err != nil {
		return false // stale file, daemon actually down
	}
	conn.Close()
	return true
}
```

### Wiring into the `err != nil` arm (D-06 keeps Steps 3-4 untouched)
```go
// after reverting the CLI fallback, checkHealth signature:
// func checkHealth(ctx context.Context, fn statusFunc, customPath string, pf prefsFunc, perm permProbeFunc) TailscaleHealth
	status, err := fn(ctx)
	if err != nil {
		if perm != nil && perm(ctx) {
			h.PermissionLimited = true // Connected/DaemonUp stay false — no false Connected (D-03)
			// DaemonUp semantics here is the planner's call; the restart gate is safe either way.
		}
		return h
	}
	h.DaemonUp = true
	h.Installed = true
	// ... Steps 3-4 byte-for-byte unchanged ...
```

### Unit test via injected fake (no live macsys needed)
```go
func TestCheckHealth_PermissionLimited(t *testing.T) {
	stub := stubBinary(t)
	statusFn := func(ctx context.Context) (*ipnstate.Status, error) {
		return nil, fmt.Errorf("dial unix /var/run/tailscaled.socket: connect: no such file or directory")
	}
	perm := func(ctx context.Context) bool { return true } // fake: macsys blocked
	h := checkHealth(context.Background(), statusFn, stub, nil, perm)
	if !h.PermissionLimited {
		t.Error("expected PermissionLimited=true")
	}
	if h.Connected {
		t.Error("SC violation: Connected must stay false when permission-limited")
	}
}

func TestCheckHealth_DaemonDown_NotPermissionLimited(t *testing.T) {
	stub := stubBinary(t)
	statusFn := func(ctx context.Context) (*ipnstate.Status, error) {
		return nil, fmt.Errorf("connect: connection refused")
	}
	perm := func(ctx context.Context) bool { return false } // fake: no macsys signature
	h := checkHealth(context.Background(), statusFn, stub, nil, perm)
	if h.PermissionLimited {
		t.Error("expected PermissionLimited=false for genuine daemon-down")
	}
}
```

### Frontend cascade extension (D-05)
```tsx
// SettingsTab.tsx — add to the tailscaleHealth interface (lines 51-60):
//   permissionLimited: boolean
function tailscaleStatusText(h: SettingsTabProps['tailscaleHealth']): string {
  if (!h) return 'Checking…'
  if (h.connected) return 'Connected'
  if (h.permissionLimited) return 'Permission Limited'   // NEW, distinct from Not Connected
  if (h.daemonUp) return 'Not Connected'
  if (h.binaryFound) return 'Daemon Stopped'
  return 'Not Installed'
}
// description branch (near lines 735-751): when h.permissionLimited, render e.g.
//   "Tailscale is running, but AgentHub can't read its status on this macOS account.
//    Grant this account admin access, or install the Homebrew `tailscale` build
//    (which uses a different socket path)."
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 169-01 same-user CLI `tailscale status --json` fallback | Honest permission-aware file-probe (this phase) | 2026-07-02 (CR-01) | The CLI runs as the same OS user, hits the identical 0640 gate, and the binary isn't setuid → zero extra privilege. Reverted. |
| Report EACCES-blocked node as "Daemon Stopped"/"Not Connected" | Distinct "Permission Limited" state with guidance | this phase | Users get an accurate, actionable message instead of a false negative. |

**Deprecated/outdated:**
- The `cliStatusFunc` / `runTailscaleStatusCLI` / `realCLIStatusFunc` code and its 3 tests: dead, being reverted (D-01).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The localapi TCP listener accepts a plain `net.Dial` (no token needed to establish the connection; token only gates HTTP Basic-Auth). | Decision Verdict / liveness check | If the listener refused unauthenticated TCP connects, the liveness check would false-negative. Low risk — `defaultDialer` establishes the TCP connection before any auth header (`local.go:120-123`), and auth is added per-request in `DoLocalRequest`. Mitigation: the file-probe alone (without liveness) still satisfies the SC; liveness is an enhancement. |
| A2 | `/Library/Tailscale` is not TCC-gated on the target non-admin machines. | Pitfall 4 | If some hardened config gates it, Glob/Open could return an unexpected error; handled by the "unknown error → fall through to daemon-down" rule, so no false Connected. |
| A3 | macsys always writes `sameuserproof-<port>` directly under `/Library/Tailscale` (not a versioned subdir) in builds users have installed. | Decision Verdict | Verified for v1.98.3-era install live; older/newer macsys builds could differ. The glob tolerates the exact suffix; if the layout changed, detection returns false (safe — today's behavior), never a false positive. |

**Note:** A1–A3 are all *fail-safe*: every uncertainty degrades to "report today's daemon-down state," never to a false Connected. This aligns with the hard SC.

## Open Questions (RESOLVED)

1. **Should `DaemonUp` be true in the permission-limited state?** — **RESOLVED (adopted in 169-01 Task 2): DaemonUp=true (and Installed=true) when liveness is confirmed.**
   - What we know: The restart gate (`Connected && HasCerts && IP`) is unaffected either way; a permission-limited node has Connected=false.
   - What's unclear: Whether the diagnostics cascade reads cleaner with DaemonUp=true (honest: the daemon IS up) or DaemonUp=false (keeps the existing "daemon running" row tied to a *readable* daemon).
   - Recommendation: Planner's call (D-03 explicitly delegates). Leaning DaemonUp=true when liveness confirmed, since it's honest and the new `permissionLimited` flag drives the distinct UI copy regardless. **→ Adopted: 169-01 Task 2 sets DaemonUp=true/Installed=true in the permission-limited guard.**

2. **Adopt the TCP liveness check, or file-probe only?** — **RESOLVED (adopted in 169-01 Task 2): liveness check adopted (`macsysDaemonAlive` via `ipnport`→`net.DialTimeout`).**
   - What we know: File-probe alone meets the letter of the SC; liveness resolves the stale-file edge case.
   - What's unclear: How often a stale `sameuserproof` persists in practice.
   - Recommendation: Adopt liveness — it's ~10 lines of stdlib, mirrors tailscale's own `checkConn`, and makes "distinct from daemon-down" (D-02) genuinely true rather than approximate. **→ Adopted: `macsysDaemonAlive()` gates PermissionLimited on a successful loopback dial to the `ipnport` port.**

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build + `go test` | ✓ | go1.26.4 darwin/arm64 | — |
| `tailscale.com` module | unchanged SDK calls | ✓ (module cache) | v1.98.3 | — |
| macsys Tailscale install (non-admin acct) | **M-45 live acceptance only** | ✗ on this dev box (account is admin) | — | **No fallback** — live verification requires a genuine non-admin macsys account; cannot be simulated in CI or on an admin machine. |

**Missing dependencies with no fallback:**
- A non-admin macOS account with macsys Tailscale for the M-45 acceptance gate. Unit tests (injected fakes) prove the branch logic; they cannot prove real-world EACCES routing. The plan must carry M-45 as a `human_judgment: true` gate.

## Validation Architecture

> `workflow.nyquist_validation` is absent from `.planning/config.json` → treated as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (backend); vitest (frontend, unaffected here) |
| Config file | none (Go); `frontend/vitest.config.*` (not exercised by this phase's core change) |
| Quick run command | `go test ./internal/webserver/ -run TestCheckHealth -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FIX-05 | Permission-limited state detected & flagged, Connected stays false | unit (injected `permProbeFunc` fake) | `go test ./internal/webserver/ -run TestCheckHealth_PermissionLimited -x` | ❌ Wave 0 (new) |
| FIX-05 | Genuine daemon-down NOT flagged permission-limited | unit | `go test ./internal/webserver/ -run TestCheckHealth_DaemonDown_NotPermissionLimited -x` | ❌ Wave 0 (new) |
| FIX-05 | SDK-success path byte-identical (SC2) | unit (regression) | `go test ./internal/webserver/ -run TestCheckHealth_FullyHealthy -x` | ✅ exists |
| FIX-05 | CLI fallback fully removed (D-01) | build/compile | `go build ./... && go vet ./...` | ✅ (revert) |
| FIX-05 | UI shows distinct guidance, never "Connected" | manual (real non-admin macsys) | M-45 checklist | ❌ manual gate |

### Sampling Rate
- **Per task commit:** `go test ./internal/webserver/ -run TestCheckHealth -count=1 && gofmt -l internal/webserver/ && go vet ./internal/webserver/`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** full backend suite green + `golangci-lint run` clean + frontend `tsc && vite build` (per memory `feedback_post_merge_gate_run_tsc`) + M-45 live acceptance before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] `internal/webserver/tailscale_test.go` — add `TestCheckHealth_PermissionLimited` and `TestCheckHealth_DaemonDown_NotPermissionLimited` (cover FIX-05 branch); remove the 3 CLI-fallback tests; update all existing `checkHealth(...)` call sites to the new signature.
- [ ] `TESTING.md` — update Section 4 traceability (removed/added test files under `internal/webserver/tailscale_test.go`), and tighten M-45 (Section 5, Category W) wording to "macsys Standalone signed-installer" per IN-02. Run `bash tests/check-traceability-paths.sh` before commit (repo standing rule).
- [ ] No framework install needed.

## Security Domain

> `security_enforcement` not disabled in config → included. This phase's security surface is small (a read-only filesystem probe) but non-trivial.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — (we deliberately do NOT read the auth token; permission-limited means we can't and don't) |
| V4 Access Control | yes | Probe is strictly read-only; honors the OS permission boundary rather than circumventing it. No privileged helper (deferred, out of scope). |
| V5 Input Validation | yes | `os.Readlink` result parsed via `strconv.Atoi` before use in a fixed `127.0.0.1:<port>` dial — no shell, no user-controlled path. Glob pattern is a hardcoded constant. |
| V6 Cryptography | no | — |
| V12 File Resources | yes | Fixed, hardcoded path (`/Library/Tailscale`); no path traversal, no user input in the filename glob. |

### Known Threat Patterns for this change
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Symlink/path confusion via `ipnport` | Tampering | The symlink target is parsed as an integer port only; it is never used as a filesystem path. A malicious target yields a failed `Atoi` → false (safe). |
| False "Connected" leading to insecure exposure decision | Spoofing / Elevation | Hard SC: permission-limited never sets Connected; the web-server restart gate additionally requires `HasCerts && IP`, unreachable in this state. |
| Reading another user's secret token | Info disclosure | We explicitly do NOT read the token (that's the bug's root cause and by design). The probe only observes *that* open fails with EACCES; it never obtains contents. |

## Sources

### Primary (HIGH confidence)
- `tailscale.com@v1.98.3/safesocket/safesocket_darwin.go` — `localTCPPortAndTokenDarwin` (69-85), `initSameUserProofToken` perms/Fchown (215-268), `readMacsysSameUserProof` + `checkConn` (277-306), `portAndTokenFromSameUserProof` error-swallow (346-361).
- `tailscale.com@v1.98.3/client/local/local.go` — `defaultDialer` TCP-then-unix fallback (112-127).
- `tailscale.com@v1.98.3/version/prop.go` — macsys vs App-Store bundle IDs and `IsMacSysGUI`/`IsMacAppStore` discrimination (26-32, 88-173).
- Live filesystem on this host (2026-07-02): `/Library/Tailscale` = 0755 root:wheel; `sameuserproof-56771` = 0640 root:admin; `ipnport` → 56771; account `ken` ∈ `admin` (bug does not repro here).
- `internal/webserver/tailscale.go`, `tailscale_test.go`, `internal/daemon/process.go`, `frontend/src/components/SettingsTab.tsx` (this repo).
- `169-REVIEW.md` CR-01/IN-02, `169-CONTEXT.md` decisions.

### Secondary (MEDIUM confidence)
- POSIX open/stat permission semantics (stat needs dir-traverse; open needs file-read) — standard, corroborated by the live `os.Open` success-as-admin observation.

### Tertiary (LOW confidence)
- None material — the make-or-break question was resolved against source, not inference.

## Metadata

**Confidence breakdown:**
- Decision (file-probe over error-classification): HIGH — traced through vendored source; the double error-swallow is explicit in the code.
- File signature & perms: HIGH — verified live on the actual macsys install.
- macsys/App-Store scoping: HIGH — distinct write paths in source (`/Library/Tailscale` vs `IPNExtension` container).
- Liveness-check necessity: MEDIUM — depends on stale-file frequency in the wild.
- M-45 live outcome: cannot be pre-verified (admin machine); the branch logic is unit-provable, the real-world routing is not.

**Research date:** 2026-07-02
**Valid until:** 2026-08-01 (stable — pinned dependency; re-check only if `tailscale.com` is bumped or macsys changes its `/Library/Tailscale` layout).
