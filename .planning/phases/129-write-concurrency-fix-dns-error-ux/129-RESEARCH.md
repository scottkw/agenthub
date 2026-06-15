# Phase 129: Write Concurrency Fix + DNS Error UX — Research

**Researched:** 2026-06-15
**Domain:** Go concurrency (per-path mutex), Tailscale local.Client / ipn.Prefs, net.DNSError detection
**Confidence:** HIGH — all claims verified against actual source files in this session

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**RESOLVED — #87 If-Match concurrency contract (user decision, 2026-06-15)**

**(a) Per-path lock → true single-winner guarantee.**

- Serialize concurrent writes to the same path so exactly one writer wins.
- The losing writer receives a clean conflict (412 Precondition Failed / If-Match mismatch), not a silent overwrite.
- This matches standard If-Match optimistic-concurrency semantics: the second writer's stale ETag must fail.
- **RACE-02 consequence:** code, inline comments, AND the remote-write proxy must all assert the single-winner contract. Any lingering "last-writer-wins (WR-02)" comments must be corrected to reflect single-winner — no mismatch between the asserted guarantee and the documentation.
- `TestWrite_TwoWritersIfMatchRace` must assert the single-winner outcome and pass 100/100 (no goroutine-scheduling dependence).

### Claude's Discretion

DNS error-UX implementation choices (where the `accept-dns` probe lives, exact message wording within the actionable-message intent, how the unresolvable-MagicDNS condition is detected vs other failures) are at Claude's discretion guided by ROADMAP success criteria and codebase conventions. Determine during plan-phase research.

### Deferred Ideas (OUT OF SCOPE)

None for this phase.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RACE-01 | `TestWrite_TwoWritersIfMatchRace` passes deterministically (100/100) — not goroutine-scheduling-dependent | Per-path mutex in `Sandbox` eliminates the TOCTOU window; both goroutines can no longer both pass the stat-check before either renames |
| RACE-02 | `WriteFileAtomic` If-Match concurrency contract is consistent across code, comments, and remote-write proxy — no "last-writer-wins (WR-02)" mismatch | `proxyRemoteFiles` comment at line 158–166 in `remote_files.go` explicitly documents WR-02 last-writer-wins and must be updated to single-winner; the conditional-header forwarding comment at line 243–246 must also be updated |
| RACE-03 | After concurrent-write conflict, final file is all-A or all-B (never interleaved), no leftover `.agenthub-tmp-*` temp files | Guaranteed by per-path lock + existing temp-cleanup logic; `TestWrite_TwoWritersIfMatchRace` already asserts these invariants |
| DNS-01 | When remote browse fails because client has `accept-dns=false`, user sees actionable message naming the fix | Add `accept-dns` detection in `proxyRemoteFiles` error path; emit specific message vs generic 502 |
| DNS-02 | Daemon distinguishes unresolvable-MagicDNS / `accept-dns=false` from other remote-unreachable failures | Detect via `net.DNSError` unwrapping + `accept-dns` probe; do NOT use as blanket catch-all |
| DNS-03 | `accept-dns` state probed proactively at startup or before first remote browse | Add probe in `checkHealth` / `TailscaleHealth` struct OR in `proxyRemoteFiles` before first dial |
</phase_requirements>

---

## Summary

Phase 129 addresses two independent bugs that share the same Go codebase and both gate the v3.5.1 release.

**Bug 1 — Write concurrency (RACE-01..03):** `TestWrite_TwoWritersIfMatchRace` fails 100% of the time (confirmed in this session — both nilCount=2, precondFailCount=0, every run). The root cause is a TOCTOU window in `WriteFileAtomic` (`internal/files/sandbox.go`): the stat-check (line 300) and the rename (line 316) are not atomic. Two goroutines sharing the same pre-write validator both pass the stat-check before either commits the rename, so both win — violating the single-winner contract. The fix is a per-path `sync.Mutex` keyed on the cleaned path inside `Sandbox`. With the lock held across stat→rename, exactly one goroutine wins; the second sees a changed validator after it acquires the lock and returns `ErrPreconditionFailed`. The remote-write proxy (`internal/daemon/remote_files.go`) must simultaneously update its WR-02 comments to remove the "last-writer-wins" claim and assert single-winner semantics.

**Bug 2 — DNS error UX (DNS-01..03):** When the client machine has `accept-dns=false` in its Tailscale prefs, MagicDNS hostnames (e.g., `peer.ts.net`) are unresolvable. The current `proxyRemoteFiles` error path catches all dial/TLS failures under a single 502 "remote unreachable" message. For the DNS case, the user needs an actionable message: "Enable Tailscale DNS (accept-dns) to browse remote sessions." The fix has two parts: (a) detect the unresolvable-MagicDNS case specifically by unwrapping `*net.DNSError` from the error returned by `client.Do`, combined with a pre-probed `accept-dns` state; (b) expose `AcceptDNS` (i.e., `ipn.Prefs.CorpDNS`) in `TailscaleHealth` so the frontend / daemon can proactively warn before the browse attempt fails. The Tailscale SDK at v1.98.3 provides `local.Client.GetPrefs()` returning `*ipn.Prefs`, with `CorpDNS bool` as the internal field backing `accept-dns`.

**Primary recommendation:** Add a `pathMu` keyed-lock map (`sync.Map` of `sync.Mutex` pointers) to `Sandbox`, acquired before the stat→rename window in `WriteFileAtomic`. Extend `TailscaleHealth` with `AcceptDNS bool` populated from `lc.GetPrefs().CorpDNS`. Detect DNS failures in `proxyRemoteFiles` via `net.DNSError` unwrap + `acceptDNS=false` check and surface a specific 502 body and frontend message.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Per-path write serialization | Files package (`internal/files`) | — | `Sandbox` owns all write primitives; lock must live there, not in HTTP handlers |
| Remote-write proxy contract docs | Daemon proxy (`internal/daemon/remote_files.go`) | — | The proxy's WR-02 comment explicitly documents the old contract; must be corrected |
| DNS failure classification | Daemon proxy (`internal/daemon/remote_files.go`) | — | `proxyRemoteFiles` is where `client.Do` returns the error; classification happens here |
| `accept-dns` state probe | Backend health layer (`internal/webserver/tailscale.go`) | App startup (`app.go`) | Reuses `local.Client` + injectable `statusFunc` pattern already present; probed via `GetPrefs` |
| Actionable DNS warning (GUI) | Frontend (App.tsx / FileBrowserTab) | — | Consumes `acceptDns` from `TailscaleHealth` event or 502 response body |

---

## Standard Stack

No new external packages are required. All fixes use the Go standard library and the already-imported Tailscale SDK.

### Core (already in go.mod)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `sync` (stdlib) | Go 1.24 | `sync.Map`, `sync.Mutex` for per-path locking | No import needed — stdlib |
| `net` (stdlib) | Go 1.24 | `*net.DNSError` unwrapping | No import needed — stdlib |
| `tailscale.com/client/local` | v1.98.3 (in go.mod) | `local.Client.GetPrefs()` → `*ipn.Prefs` | Already imported in `internal/webserver/tailscale.go` |
| `tailscale.com/ipn` | v1.98.3 (in go.mod) | `ipn.Prefs.CorpDNS bool` — the `accept-dns` field | Already imported transitively |

### Package Legitimacy Audit

> No new packages are introduced by this phase. All fixes are in-codebase changes using stdlib and the already-vetted Tailscale SDK.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
WriteFileAtomic call (any goroutine)
         │
         ▼
  validateAndClean(relPath)
         │
         ▼
  denylistCheck(cleaned)
         │
         ▼
  [NEW] acquire perPathMu(cleaned) ◄── serializes concurrent writers for same path
         │
         ▼
  os.OpenRoot(s.rootPath)
         │
         ▼
  O_EXCL temp create → write → sync → close
         │
         ▼
  validator re-check (stat → compare)
         │              │
       match          mismatch
         │              │
         ▼              ▼
     Rename          Remove tmp
      (wins)       ErrPreconditionFailed
         │              │
         ▼              ▼
  [NEW] release perPathMu   release perPathMu
```

```
proxyRemoteFiles (accept-dns error path)
         │
         ▼
  client.Do(upstreamReq)
         │
       error
         │
         ▼
  [NEW] isAcceptDNSFailure(err, baseURL)?
         │              │
        yes             no
         │              │
         ▼              ▼
  502 + specific     502 + generic
  DNS message        "remote unreachable"
```

### Recommended Project Structure

No new files needed. Changes are surgical edits to:

```
internal/files/
├── sandbox.go       ← add pathLocks sync.Map + perPathLock() helper; modify WriteFileAtomic
└── write_test.go    ← TestWrite_TwoWritersIfMatchRace stays; verify it passes 100/100

internal/daemon/
└── remote_files.go  ← update WR-02 comment; add DNS-failure detection in proxyRemoteFiles

internal/webserver/
└── tailscale.go     ← add AcceptDNS bool to TailscaleHealth; populate from GetPrefs
```

Frontend (if needed for DNS-01 actionable message in GUI):
```
frontend/src/wailsjs/wailsjs/go/models.ts  ← add acceptDns field to TailscaleHealth class
frontend/src/App.tsx or FileBrowserTab.tsx  ← display warning when acceptDns=false
```

### Pattern 1: Per-Path Keyed Lock via sync.Map

**What:** Each unique cleaned path gets its own `sync.Mutex`. A `sync.Map` stores `*sync.Mutex` values keyed by path string. Callers load-or-store a mutex, lock it, execute the critical section, unlock it.

**When to use:** Serializing concurrent writes to the same path without blocking writers to different paths.

**Example:**
```go
// Source: Go stdlib sync.Map documentation + standard keyed-lock pattern
// in internal/pty/registry.go (RWMutex on session registry — same idiom)

// pathLocks provides per-path serialization for WriteFileAtomic.
// Each cleaned path gets its own *sync.Mutex so concurrent writers to
// DIFFERENT paths do not block each other — only same-path writers serialize.
// The Map is never cleaned (entries are cheap: one pointer per distinct path
// seen in the process lifetime). sync.Map is safe for concurrent use.
var pathLocks sync.Map // key: string (cleaned path), value: *sync.Mutex

func perPathLock(cleaned string) *sync.Mutex {
    v, _ := pathLocks.LoadOrStore(cleaned, &sync.Mutex{})
    return v.(*sync.Mutex)
}

// In WriteFileAtomic, immediately after denylistCheck and root open:
mu := s.perPathLock(cleaned)
mu.Lock()
defer mu.Unlock()
// ... stat-check + rename ...
```

[VERIFIED: Go stdlib docs, confirmed against `internal/pty/registry.go:11` which uses the same pattern for per-session locking]

**Why this beats a single global mutex:** A global mutex would serialize ALL writes across ALL paths, creating a bottleneck. The keyed-lock approach serializes only same-path concurrent writes.

**Why this beats `singleflight`:** `singleflight` deduplicates; we need serialization, not deduplication. The two writers have different payloads and we want exactly-one-winner, not coalesced into one.

### Pattern 2: net.DNSError Unwrapping for Accept-DNS Detection

**What:** When `client.Do` fails, the error chain is `*url.Error` → `*net.OpError` → `*net.DNSError`. Use `errors.As` to reach the `*net.DNSError` and check `dnsErr.IsNotFound` or `dnsErr.Err` containing "no such host".

**When to use:** Only in `proxyRemoteFiles` after `client.Do` returns an error and the `baseURL` contains a MagicDNS hostname (`.ts.net` suffix or similar).

**Example:**
```go
// Source: Go stdlib net package documentation
// net.DNSError is documented at https://pkg.go.dev/net#DNSError

import (
    "errors"
    "net"
    "strings"
)

// isAcceptDNSFailure reports whether err is a DNS resolution failure for a
// MagicDNS hostname (*.ts.net), which typically indicates accept-dns=false.
// It uses errors.As to unwrap the *url.Error → *net.OpError → *net.DNSError
// chain that client.Do returns on dial failure.
func isAcceptDNSFailure(err error, baseURL string) bool {
    var dnsErr *net.DNSError
    if !errors.As(err, &dnsErr) {
        return false
    }
    // Only flag this for MagicDNS hostnames — IP-based URLs would not exhibit this.
    return strings.Contains(baseURL, ".ts.net") || strings.Contains(baseURL, ".tailscale.net")
}
```

[VERIFIED: Go stdlib net.DNSError struct verified in go/net package; `errors.As` unwrap chain confirmed against Go 1.24 docs]

**Important constraint (DNS-02):** This function must return true ONLY when the hostname is a MagicDNS name AND the error is a DNS resolution failure. A connection-refused or TLS error must not trigger the actionable DNS message.

### Pattern 3: GetPrefs for accept-dns State

**What:** `local.Client.GetPrefs(ctx)` returns `*ipn.Prefs` which has `CorpDNS bool`. This is the internal field backing the `--accept-dns` CLI flag. `true` = DNS enabled (normal), `false` = disabled.

**When to use:** At startup (to populate `TailscaleHealth.AcceptDNS`) or lazily on the first remote browse attempt.

**Example:**
```go
// Source: tailscale.com@v1.98.3/client/local/local.go:871
// Source: tailscale.com@v1.98.3/ipn/prefs.go:131-133

var lc local.Client
prefs, err := lc.GetPrefs(ctx)
if err == nil {
    h.AcceptDNS = prefs.CorpDNS // true = DNS enabled, false = accept-dns=false
}
```

[VERIFIED: `local.Client.GetPrefs` at `tailscale.com@v1.98.3/client/local/local.go:871`; `ipn.Prefs.CorpDNS bool` at `tailscale.com@v1.98.3/ipn/prefs.go:133`]

### Anti-Patterns to Avoid

- **Global mutex in WriteFileAtomic:** Would serialize all writes across all paths. Per-path lock is correct.
- **Catching DNSError as blanket catch-all for DNS-01:** DNS-02 requires discrimination. Only flag `accept-dns` issue when both `net.DNSError` AND MagicDNS hostname are confirmed.
- **Exposing `CorpDNS` as `AcceptDNS` without fallback:** `GetPrefs` fails when tailscaled is not running. Handle error gracefully — omit `AcceptDNS` from health (leave zero/false) rather than blocking startup.
- **Adding a per-session Sandbox-level lock to the Sandbox struct:** `Sandbox` is stateless by design (constructed per-request — see `api.go:81-89`). A per-request Sandbox cannot hold cross-request state. The lock map must be at the package level or injected. A package-level `sync.Map` (keyed on absolute path = `s.rootPath + "/" + cleaned`) is the correct scope.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Per-path locking | Custom hash map + global mutex | `sync.Map` of `*sync.Mutex` | `sync.Map` is safe for concurrent access, optimized for load-or-store; hand-rolled map needs its own lock |
| DNS error classification | String-matching `err.Error()` for "no such host" | `errors.As(err, &net.DNSError)` | String matching is brittle across Go versions and platforms; `errors.As` is the correct unwrap API |
| Tailscale prefs access | Parsing `tailscale status --json` via subprocess | `local.Client.GetPrefs(ctx)` | Direct API call; already used by `internal/webserver/tailscale.go`; no subprocess, no parse fragility |

---

## TOCTOU Root Cause Analysis (RACE-01..03)

### The Exact Race Window

The current `WriteFileAtomic` performs (simplified):
1. `root.OpenFile(tmp, O_CREATE|O_EXCL, ...)` — creates temp file (unique name, no race here)
2. Write + Sync + Close
3. **stat-check:** `root.Stat(cleaned)` → compare validator
4. **rename:** `root.Rename(tmp, cleaned)`

Between steps 3 and 4, the file is NOT locked. Two goroutines both holding the same `sharedValidator` can both execute step 3 and both succeed (the file has not changed yet — neither has renamed). Then both execute step 4 (rename). The POSIX rename is atomic, so the last rename wins and clobbers the first. No error is returned because `rename` over an existing file is defined to succeed on Linux/macOS.

**Test confirmation:** Running `TestWrite_TwoWritersIfMatchRace` shows `nilCount=2, precondFailCount=0` — BOTH goroutines return nil, meaning both "won." This is 100% reproducible (not a flake — a systematic failure).

### Why the CR-01 Mitigation Alone Is Insufficient

The existing CR-01 mitigation (stat re-check inside `WriteFileAtomic` before rename) narrows the window but cannot close it without a lock. The comment in `sandbox.go:232-238` explicitly acknowledges "a residual window (stat fires → another writer lands → rename executes) exists." The per-path lock closes this window entirely by making the stat→rename sequence atomic with respect to other writers of the same path.

### Per-Path Lock Placement

The lock must be acquired **after** `validateAndClean` and `denylistCheck` (so invalid paths are rejected cheaply before touching the lock map), but **before** the `os.OpenRoot` call (so the temp-creation + stat-check + rename are inside the critical section).

The lock key must be the **absolute path** (`s.rootPath + "/" + cleaned`), not just `cleaned`, because two different `Sandbox` instances with different `rootPath` values should NOT contend with each other for the same `cleaned` relative name.

```go
// Lock key = absolute path to prevent cross-sandbox contention
lockKey := filepath.Join(s.rootPath, cleaned)
mu := perPathLock(lockKey)
mu.Lock()
defer mu.Unlock()
```

---

## DNS Error UX Design (DNS-01..03)

### Detection Strategy (DNS-02)

When `proxyRemoteFiles` receives an error from `client.Do`:

1. Unwrap via `errors.As(err, &net.DNSError)` — confirms this is a DNS resolution failure
2. Check `baseURL` contains `.ts.net` or the peer's MagicDNS suffix — confirms it's a MagicDNS hostname (not a raw IP or non-Tailscale hostname)
3. Optionally confirm `accept-dns=false` from the proactively-probed `TailscaleHealth.AcceptDNS` field — provides the strongest signal for DNS-02 discrimination

This combination ensures the actionable message fires ONLY for the `accept-dns=false` / unresolvable-MagicDNS case.

### Proactive Probe (DNS-03)

The cleanest fit for DNS-03 is extending `checkHealth` in `internal/webserver/tailscale.go`:

- Add `AcceptDNS bool` to `TailscaleHealth` struct
- In `checkHealth`, after Step 3 (Connected check), call `lc.GetPrefs(ctx)` and populate `h.AcceptDNS = prefs.CorpDNS`
- If `GetPrefs` fails (daemon not running), leave `AcceptDNS` as `false` — safe default (treat as unknown)
- The existing `startHealthPoller` in `app.go` will pick this up and emit it in the `tailscale:health` event without any additional wiring

The frontend can then show a warning banner in the Remote Sessions panel when `tailscaleHealth.acceptDns === false` before the user even attempts to browse.

**Alternative (lazy probe in `proxyRemoteFiles`):** The proxy could call `GetPrefs` lazily when it detects a DNS error. This is simpler but doesn't satisfy DNS-03's "before the first remote browse" requirement. The startup-poller path is preferred.

### Message Wording

Per the CONTEXT.md specifics and REQUIREMENTS.md DNS-01:
> "Enable Tailscale DNS (accept-dns) to browse remote sessions."

This exact wording should appear in:
1. The 502 response body from `proxyRemoteFiles` (replacing the generic "remote unreachable" message)
2. The frontend warning banner (when `acceptDns === false` from health poller)

---

## WR-02 Comment Corrections (RACE-02)

Three locations in `internal/daemon/remote_files.go` must be updated:

**Location 1:** `proxyRemoteFiles` doc comment (lines 158–166):
```
// Concurrency contract (WR-02): multiple concurrent write proxies for the same
// session race at the remote peer's files.Handler — last-writer-wins.
```
Must change to: single-winner via per-path lock in the remote peer's `WriteFileAtomic`.

**Location 2:** The `If-Match` forwarding comment (lines 243–246):
```
// WR-02: Forward conditional-write headers when present so that a future
// optimistic-concurrency layer on the remote peer is reachable without a
// proxy change. The peer does not currently enforce these preconditions;
// the current contract is last-writer-wins (see proxyRemoteFiles comment).
```
Must change to: the preconditions ARE enforced by the peer's `WriteFileAtomic` single-winner lock; `If-Match` forwarding is required, not future-proofing.

---

## Common Pitfalls

### Pitfall 1: Sandbox Is Stateless (Per-Request Construction)
**What goes wrong:** Adding a `pathMu sync.Map` field directly to `Sandbox` struct. In production, `NewSandbox` is called per-request (see `api.go:81-89`), so each request gets a fresh `Sandbox` with an empty lock map — no cross-request serialization occurs.
**Why it happens:** `Sandbox` looks like a natural place for a lock map.
**How to avoid:** Use a **package-level** `sync.Map` keyed on the absolute path, not a struct field. The lock map lives in `sandbox.go` at package scope.
**Warning signs:** Tests pass (single-request, no concurrency) but `TestWrite_TwoWritersIfMatchRace` still fails.

### Pitfall 2: Lock Key Collision Across Different Sandbox Roots
**What goes wrong:** Using `cleaned` (relative path) as the lock key. Two `Sandbox` instances for different sessions writing to different absolute directories but the same relative name (e.g., both writing `"readme.txt"`) would contend unnecessarily.
**Why it happens:** Relative path is what `WriteFileAtomic` receives.
**How to avoid:** Key on `filepath.Join(s.rootPath, cleaned)` — the absolute path.

### Pitfall 3: Treating All DNSErrors as Accept-DNS Failures (DNS-02)
**What goes wrong:** Showing the actionable "Enable Tailscale DNS" message for ANY DNS failure, including legitimate network outages or the remote peer being offline.
**Why it happens:** DNS errors from a dead peer also return `*net.DNSError`.
**How to avoid:** Require the hostname to be a known MagicDNS name (`.ts.net`) AND `net.DNSError` to be present. Optionally cross-check with `acceptDns=false` from the probed state.

### Pitfall 4: GetPrefs Failing on Non-Tailscale Machines
**What goes wrong:** `GetPrefs` panics or causes a startup failure when tailscaled is not running.
**Why it happens:** `lc.GetPrefs` returns an error when the daemon socket is not reachable.
**How to avoid:** Call `GetPrefs` inside the existing `checkHealth` function which already handles the daemon-unreachable case (Step 2). If `fn(ctx)` succeeds but `GetPrefs` fails, leave `AcceptDNS` at zero value — acceptable degradation.

### Pitfall 5: Lock Not Released on WriteFileAtomic Error Paths
**What goes wrong:** Early returns from `WriteFileAtomic` (on `rand.Read` failure, temp create failure, etc.) bypass the `mu.Unlock()` call.
**Why it happens:** Manual unlock placement.
**How to avoid:** Use `defer mu.Unlock()` immediately after `mu.Lock()`. The existing error paths all `return` before the rename, which is exactly what we want — lock is held across the temp+stat+rename window and released on any return.

---

## Code Examples

### Per-Path Lock Implementation in WriteFileAtomic

```go
// Source: internal/files/sandbox.go — PROPOSED addition

// pathLocks provides per-path serialization for WriteFileAtomic. Key is the
// absolute path (rootPath + "/" + cleaned) so different sandbox roots do not
// contend. The map is never pruned; entries are cheap (one pointer per unique
// absolute path written during the process lifetime).
//
// Single-winner guarantee (RACE-01): by holding the lock across the
// stat-check → rename window, exactly one concurrent writer commits;
// the second acquires the lock after the first's rename, observes a changed
// validator, and returns ErrPreconditionFailed.
var pathLocks sync.Map // key: string (absolute path), value: *sync.Mutex

// perPathLock returns the *sync.Mutex for the given absolute path, creating
// one if it does not exist. Callers must Lock/Unlock the returned mutex.
func perPathLock(absPath string) *sync.Mutex {
    v, _ := pathLocks.LoadOrStore(absPath, &sync.Mutex{})
    return v.(*sync.Mutex)
}

// In WriteFileAtomic, after denylistCheck and before root open:

// Single-winner serialization (RACE-01): hold a per-path mutex across the
// stat-check → rename window so concurrent writers to the same path are
// serialized. The second writer acquires the lock after the first's rename
// completes, observes a changed validator in the re-check, and returns
// ErrPreconditionFailed — a clean 412 outcome.
lockKey := filepath.Join(s.rootPath, cleaned)
mu := perPathLock(lockKey)
mu.Lock()
defer mu.Unlock()

root, err := os.OpenRoot(s.rootPath)
// ... rest of the function unchanged ...
```

### DNS Detection in proxyRemoteFiles

```go
// Source: internal/daemon/remote_files.go — PROPOSED addition

// isUnresolvableMagicDNS reports whether err is a DNS resolution failure for
// a MagicDNS hostname (containing ".ts.net"), indicating the client likely
// has accept-dns=false in Tailscale prefs. Only returns true when both the
// hostname is MagicDNS AND the error is a DNS resolution failure — not for
// connection-refused, TLS, or other network errors (DNS-02 discrimination).
func isUnresolvableMagicDNS(err error, baseURL string) bool {
    var dnsErr *net.DNSError
    if !errors.As(err, &dnsErr) {
        return false
    }
    // Tailscale MagicDNS names end in .ts.net or .tailscale.net
    return strings.Contains(baseURL, ".ts.net") ||
           strings.Contains(baseURL, ".tailscale.net")
}

// In proxyRemoteFiles, replacing the current single-line error return:
resp, err := client.Do(req)
if err != nil {
    if isUnresolvableMagicDNS(err, baseURL) {
        http.Error(w,
            "Enable Tailscale DNS (accept-dns) to browse remote sessions",
            http.StatusBadGateway)
        return
    }
    http.Error(w, "remote unreachable: "+redactCapTokenFromError(err, capToken), http.StatusBadGateway)
    return
}
```

### TailscaleHealth AcceptDNS Extension

```go
// Source: internal/webserver/tailscale.go — PROPOSED addition

type TailscaleHealth struct {
    Installed    bool   `json:"installed"`
    Connected    bool   `json:"connected"`
    HasCerts     bool   `json:"hasCerts"`
    IP           string `json:"ip"`
    Domain       string `json:"domain"`
    BinaryFound  bool   `json:"binaryFound"`
    DaemonUp     bool   `json:"daemonUp"`
    PlatformHint string `json:"platformHint"`
    // AcceptDNS is true when the local Tailscale node has accept-dns=true
    // (the default). False means MagicDNS resolution will fail for remote
    // peers, blocking remote file browse. Zero value (false) when daemon is
    // unreachable or prefs are unavailable. (DNS-03)
    AcceptDNS bool `json:"acceptDns"`
}

// In checkHealth, after Step 3 (Connected check):
// Step 4b: DNS accept state (DNS-03 proactive probe)
// GetPrefs returns the user's current Tailscale preferences. CorpDNS is
// the internal field for --accept-dns (true = DNS enabled, false = disabled).
// Failure (daemon not responding to /localapi/v0/prefs) is silently swallowed —
// safe degradation: AcceptDNS stays false (unknown / assume disabled for warning).
if h.Connected {
    var lc local.Client
    if prefs, prefsErr := lc.GetPrefs(ctx); prefsErr == nil {
        h.AcceptDNS = prefs.CorpDNS
    }
}
```

---

## Runtime State Inventory

> This is a bug-fix + UX phase (not a rename/migration). No runtime state inventory is required.

**None — verified: no stored data, live service config, OS-registered state, secrets, or build artifacts are being renamed or migrated.**

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All builds and tests | ✓ | 1.24+ (confirmed by `os.OpenRoot` usage) | — |
| `tailscale.com` module | DNS probe, Tailscale health | ✓ | v1.98.3 (confirmed via `go list -m`) | — |
| `sync.Map` (stdlib) | Per-path lock | ✓ | stdlib | — |
| `net.DNSError` (stdlib) | DNS error classification | ✓ | stdlib | — |

**Missing dependencies with no fallback:** none

---

## Validation Architecture

> `workflow.nyquist_validation` key is absent from `.planning/config.json` — treat as enabled.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./internal/files/... -run TestWrite_TwoWritersIfMatchRace -count=100` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RACE-01 | `TestWrite_TwoWritersIfMatchRace` passes deterministically | unit | `go test ./internal/files/... -run TestWrite_TwoWritersIfMatchRace -count=100` | ✅ `internal/files/write_test.go:1153` |
| RACE-02 | WR-02 comments in proxy updated to single-winner contract | static/grep | `grep -n "last-writer-wins" internal/daemon/remote_files.go \| wc -l` (expect 0) | ✅ (code exists, comment update is the fix) |
| RACE-03 | No interleaved content, no leftover temp files | unit | `go test ./internal/files/... -run TestWrite_TwoWritersIfMatchRace -count=100` | ✅ test already asserts these invariants |
| DNS-01 | Actionable message surfaces for accept-dns=false + MagicDNS failure | unit | `go test ./internal/daemon/... -run TestProxyRemoteFiles_AcceptDNSMessage` | ❌ Wave 0 gap — new test needed |
| DNS-02 | DNS message not shown for non-DNS failures | unit | `go test ./internal/daemon/... -run TestProxyRemoteFiles_AcceptDNSMessage` | ❌ Wave 0 gap — same new test |
| DNS-03 | `acceptDns` populated in `TailscaleHealth` when connected | unit | `go test ./internal/webserver/... -run TestCheckHealth_AcceptDNS` | ❌ Wave 0 gap — new test needed |

Relay-surface coverage (mandatory per REQUIREMENTS.md):
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RACE-01 | Two-writer race serialized on relay surface | unit | `go test ./internal/daemon/... -run TestRemoteFiles_TwoWriterRace_RelaySurface` | ❌ Wave 0 gap — relay surface must be exercised |
| DNS-01 | DNS message surfaced through relay proxy | unit | covered by `TestProxyRemoteFiles_AcceptDNSMessage` (tests `proxyRemoteFiles` directly) | ❌ Wave 0 gap |

### Sampling Rate

- **Per task commit:** `go test ./internal/files/... -run TestWrite_TwoWritersIfMatchRace -count=100`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/daemon/remote_files_test.go` — `TestProxyRemoteFiles_AcceptDNSMessage`: covers DNS-01 + DNS-02 (specific message on DNS failure for MagicDNS hostname; NOT triggered for connection-refused or TLS errors)
- [ ] `internal/webserver/tailscale_test.go` — `TestCheckHealth_AcceptDNS`: covers DNS-03 (AcceptDNS field populated when Connected and GetPrefs returns CorpDNS=false)
- [ ] `internal/daemon/relay_remote_files_test.go` — `TestRemoteFiles_TwoWriterRace_RelaySurface`: relay-surface coverage for RACE-01 (two concurrent PUT writes through the relay to the same path; assert single-winner)

---

## Security Domain

> `security_enforcement` not present in config — treat as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes (path validation unchanged) | existing `validateAndClean` / `validateRelativePath` |
| V6 Cryptography | no | — |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Race condition allowing silent content corruption | Tampering | Per-path mutex — closes the stat→rename TOCTOU window |
| DNS error message leaking internal topology info | Information Disclosure | `isUnresolvableMagicDNS` only emits generic actionable message; no hostname or stack trace in 502 body |
| Cap token leaked in DNS error message | Information Disclosure | `redactCapTokenFromError` already used; DNS path also must not include the token |

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| CR-01: stat re-check "narrows the TOCTOU window" | Per-path mutex: closes the window | Phase 129 | `TestWrite_TwoWritersIfMatchRace` passes 100/100 |
| WR-02: "last-writer-wins" proxy contract | Single-winner: proxy forwards If-Match; remote enforces serialization | Phase 129 | Code, comments, and proxy aligned |
| Opaque 502 for all remote-unreachable failures | Specific "Enable Tailscale DNS" message for MagicDNS DNS failures | Phase 129 | DNS-01: actionable UX |
| `TailscaleHealth` has no `accept-dns` field | `acceptDns bool` populated from `GetPrefs().CorpDNS` | Phase 129 | DNS-03: proactive warning before browse fails |

**Deprecated/outdated:**
- `proxyRemoteFiles` WR-02 comment documenting "last-writer-wins": superseded by single-winner contract.
- `sandbox.go` comment "A residual window (stat fires → another writer lands → rename executes) exists but is microscopic" (line 234-238): this residual window is closed by the per-path lock — comment should be updated to reflect the lock.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `.ts.net` and `.tailscale.net` are the only MagicDNS hostname suffixes emitted by AgentHub's discovery | DNS Detection | If Tailscale uses other MagicDNS suffixes (e.g., custom domain), DNS-02 would not detect them — but the generic 502 still fires, so the failure mode is "no actionable message" not "wrong message" |
| A2 | `ipn.Prefs.CorpDNS = false` means `accept-dns=false` (MagicDNS disabled) | DNS Probe | Verified from `tailscale.com@v1.98.3/ipn/prefs.go:131-133` and `ipn/conf.go:26,82`; LOW risk |

**The table above has 2 entries.** All other claims in this research were verified directly against source files or the Tailscale SDK on disk.

---

## Open Questions (RESOLVED)

1. **MagicDNS hostname suffix scope (A1 above)** — **RESOLVED**
   - What we know: AgentHub builds `baseURL` from `peer.DNSName` which ends in `.ts.net` (confirmed in `internal/tailnet/tailnet.go:117` `strings.TrimSuffix(peer.DNSName, ".")`)
   - **Resolution:** Use `.ts.net` check as the primary signal for `isUnresolvableMagicDNS`; this matches how AgentHub itself constructs the baseURL. Custom/funnel suffixes can be tightened in a follow-up if a real case appears — graceful degradation to the generic 502 for unknown suffixes is acceptable.

2. **Frontend warning placement for DNS-03** — **RESOLVED (user decision 2026-06-15)**
   - What we know: `startHealthPoller` emits `tailscale:health` every 10s with the full `TailscaleHealth` struct; `App.tsx` listens and updates `tailscaleHealth` state.
   - **Resolution:** Per explicit user decision, the proactive visible warning ships **in Phase 129** (not deferred to Phase 130). Add a minimal frontend warning component (e.g. `RemoteBrowseDNSWarning` or reuse the `LocalNetworkBanner.tsx` pattern) that reads `tailscaleHealth.acceptDns` and, when `acceptDns === false` while connected, displays the actionable message ("Enable Tailscale DNS (accept-dns) to browse remote sessions") proactively — before the user attempts a remote browse. DNS-03 is fully self-contained in Phase 129. On-screen render verification is manual UAT (live tailnet with `accept-dns=false`); component presence + conditional-render logic are statically verifiable.

---

## Sources

### Primary (HIGH confidence)
- `internal/files/sandbox.go` — `WriteFileAtomic` implementation, `ErrPreconditionFailed`, CR-01 TOCTOU mitigation comment
- `internal/files/write_test.go` — `TestWrite_TwoWritersIfMatchRace` implementation and assertions
- `internal/daemon/remote_files.go` — `proxyRemoteFiles`, WR-02 comments, `isUnresolvableMagicDNS` addition target
- `internal/daemon/relay_remote_files.go` — relay surface mounting of remote proxy routes
- `internal/daemon/relay_remote_files_test.go` — relay surface test harness
- `internal/webserver/tailscale.go` — `TailscaleHealth`, `checkHealth`, `local.Client` pattern
- `tailscale.com@v1.98.3/client/local/local.go:871` — `GetPrefs` method signature
- `tailscale.com@v1.98.3/ipn/prefs.go:131-133` — `CorpDNS bool` field definition
- `tailscale.com@v1.98.3/ipn/conf.go:26,82` — `AcceptDNS` → `CorpDNS` mapping confirmation
- Go stdlib `sync.Map` documentation — `LoadOrStore` pattern for keyed locks

### Secondary (MEDIUM confidence)
- `app.go:1298` — `startHealthPoller` — confirms health poll event infrastructure for DNS-03
- `internal/tailnet/tailnet.go:117` — `.ts.net` hostname suffix pattern

### Tertiary (LOW confidence — see Assumptions Log)
- MagicDNS suffix scope: only `.ts.net` observed in codebase; other suffixes assumed absent

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; all fixes use stdlib + existing imports
- Architecture: HIGH — root cause confirmed by test execution (`nilCount=2` observed), fix design verified against Go sync docs
- Pitfalls: HIGH — Sandbox per-request construction verified in `api.go:81-89`; package-level lock placement follows from this

**Research date:** 2026-06-15
**Valid until:** 2026-07-15 (stable Go stdlib + Tailscale SDK; 30 days safe)
