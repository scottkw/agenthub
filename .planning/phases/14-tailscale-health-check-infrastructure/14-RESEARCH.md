# Phase 14: Tailscale Health Check Infrastructure - Research

**Researched:** 2026-03-20
**Domain:** Go — `tailscale.com/client/local` health check integration in a Wails desktop app
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| HEALTH-01 | App detects whether Tailscale is installed on the system | `lc.StatusWithoutPeers()` connection error → not installed/not running; STATE.md note confirms treating connection error as "not installed or not running" |
| HEALTH-02 | App detects whether Tailscale is connected to a tailnet | `status.BackendState == "Running"` — sourced directly from `ipnstate.Status.BackendState` field, verified in tailscale module source |
| HEALTH-03 | App detects whether HTTPS certificates are enabled in the tailnet | `len(status.CertDomains) > 0` — `CertDomains` is populated by the Tailscale control plane only when MagicDNS + HTTPS are enabled in admin console |
| HEALTH-06 | Health checks run periodically in background; modal updates automatically when user resolves issues | Background goroutine with `time.Ticker` + `runtime.EventsEmit` — mirrors the existing `status.Watch` pattern in `internal/status/detector.go` |
</phase_requirements>

---

## Summary

Phase 14 adds a new Go package function (`webserver.CheckHealth`) and a background poller goroutine in `app.go`. The function calls `local.Client{}.StatusWithoutPeers(ctx)` to query the already-running `tailscaled` daemon and derives three boolean health flags: `Installed`, `Connected`, and `HasCerts`. These are returned as a `TailscaleHealth` struct exposed to the Wails frontend via a new `GetTailscaleStatus()` method. A background goroutine polls every 10 seconds and pushes `tailscale:health` events via `runtime.EventsEmit` so the frontend receives live updates.

The `tailscale.com` module was added to `go.mod` during research (as `indirect` — the phase work will move it to `direct`). Binary size delta: **+0.5 MB** (from 9.17 MB to 9.70 MB), well within the ~25 MB budget. This phase makes zero behavioral changes to existing functionality — it only adds new code paths with no callsites until the new Wails method is wired.

**Primary recommendation:** Create `internal/webserver/tailscale.go` with `TailscaleHealth` struct and `CheckHealth()` function. Add `GetTailscaleStatus()` as a Wails-bound method on `App`. Start the background poller in `startup()`. Write table-driven unit tests using a fake `statusFn` injected via a function parameter.

---

## Standard Stack

### Core (already pulled via `go get tailscale.com/client/local@v1.96.3`)

| Package | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `tailscale.com/client/local` | v1.96.3 | Query running `tailscaled` daemon — status, cert domains, Tailscale IP | Official current API. Zero-value `local.Client{}` uses platform default socket (Unix on Linux/macOS, named pipe on Windows). No config needed. |
| `tailscale.com/ipn/ipnstate` | v1.96.3 (same module) | `ipnstate.Status` struct with `BackendState`, `CertDomains`, `TailscaleIPs`, `Self.DNSName` | Pulled transitively — not a separate `go get`. All health check fields live here. |

### Supporting (stdlib only — no new libraries)

| Package | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `context` | stdlib | Short-timeout context for `StatusWithoutPeers` calls | Always — 5-second timeout prevents UI blocking |
| `time` | stdlib | `time.Ticker` for the background poll goroutine | Background health poller |
| `github.com/wailsapp/wails/v2/pkg/runtime` | v2.10.2 (existing) | `runtime.EventsEmit` to push `tailscale:health` events to frontend | Already used for `session:status` events |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `tailscale.com/client/local` | `exec.Command("tailscale", "status", "--json")` subprocess | Subprocess requires `tailscale` CLI on PATH (not guaranteed on macOS App Store install); parsing JSON output is fragile; no automatic cert caching. Use the Go API. |
| `local.Client{}` zero value | `local.Client{Socket: customPath}` | Only needed for non-standard daemon socket paths. Default works everywhere Tailscale is installed normally. |

**Installation (already done during research):**

```bash
go get tailscale.com/client/local@v1.96.3
```

The tailscale.com module is currently marked `// indirect` in go.mod. Importing `tailscale.com/client/local` in the new `tailscale.go` file will promote it to direct automatically on next `go mod tidy`.

**Binary size delta verified:** +0.5 MB (9.17 MB → 9.70 MB). Acceptable. The ~25 MB fallback threshold (documented in STATE.md) is not at risk.

---

## Architecture Patterns

### Recommended Project Structure (new file only)

```
internal/webserver/
├── tailscale.go        ← NEW: TailscaleHealth struct + CheckHealth()
├── tailscale_test.go   ← NEW: unit tests with fake statusFn
├── server.go           (unchanged)
├── network.go          (unchanged — deleted in Phase 17)
├── tls.go              (unchanged — deleted in Phase 15)
├── auth.go             (unchanged — deleted in Phase 16)
└── tokens.go           (unchanged — deleted in Phase 16)
app.go                  ← ADD: GetTailscaleStatus() method + startHealthPoller()
app_test.go             ← ADD: TestGetTailscaleStatus test
```

### Pattern 1: `TailscaleHealth` Struct and `CheckHealth` Function

**What:** Thin wrapper around `local.Client{}.StatusWithoutPeers()` that normalizes daemon state into three booleans.

**When to use:** Any callsite that needs to know if Tailscale is usable. Called both from the Wails method (on-demand) and from the background poller.

**Example (from prior architecture research, verified against actual ipnstate source):**

```go
// Source: internal/webserver/tailscale.go (new file)
package webserver

import (
    "context"
    "tailscale.com/client/local"
)

// TailscaleHealth is the result of a health check against the local tailscaled.
// Serialised to JSON and returned to the Wails frontend via GetTailscaleStatus().
type TailscaleHealth struct {
    Installed bool   `json:"installed"` // tailscaled socket reachable
    Connected bool   `json:"connected"` // BackendState == "Running"
    HasCerts  bool   `json:"hasCerts"`  // len(CertDomains) > 0
    IP        string `json:"ip"`        // first TailscaleIP as string, empty if not connected
    Domain    string `json:"domain"`    // first CertDomain e.g. hostname.ts.net, empty if none
}

// CheckHealth queries tailscaled via local.Client and returns TailscaleHealth.
// ctx should carry a short timeout (3-5 seconds).
func CheckHealth(ctx context.Context) TailscaleHealth {
    var lc local.Client
    status, err := lc.StatusWithoutPeers(ctx)
    if err != nil {
        return TailscaleHealth{Installed: false}
    }
    h := TailscaleHealth{Installed: true}
    h.Connected = status.BackendState == "Running"
    h.HasCerts = len(status.CertDomains) > 0
    if len(status.TailscaleIPs) > 0 {
        h.IP = status.TailscaleIPs[0].String()
    }
    if len(status.CertDomains) > 0 {
        h.Domain = status.CertDomains[0]
    }
    return h
}
```

**Three health states:**

| `Installed` | `Connected` | `HasCerts` | Meaning | Frontend response |
|-------------|-------------|------------|---------|------------------|
| false | — | — | tailscaled not reachable | Show "Install/Start Tailscale" |
| true | false | — | Installed but not connected | Show "Connect to tailnet" |
| true | true | false | Connected, HTTPS disabled | Show "Enable HTTPS in Tailscale admin" |
| true | true | true | Ready | No modal — all green |

### Pattern 2: Testable `CheckHealth` via Function Injection

**What:** Accept a `statusFn` parameter for testing so unit tests never require a live `tailscaled`.

**When to use:** The exported `CheckHealth(ctx)` calls an internal `checkHealth(ctx, statusFn)` where the function is injected. Tests pass a fake that returns controlled `*ipnstate.Status` or errors.

**Example:**

```go
// Source: internal/webserver/tailscale_test.go
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

func checkHealth(ctx context.Context, fn statusFunc) TailscaleHealth {
    status, err := fn(ctx)
    if err != nil {
        return TailscaleHealth{Installed: false}
    }
    // ... same logic as CheckHealth
}

// Public wrapper uses the real client.
func CheckHealth(ctx context.Context) TailscaleHealth {
    var lc local.Client
    return checkHealth(ctx, lc.StatusWithoutPeers)
}
```

Tests:

```go
func TestCheckHealth_NotRunning(t *testing.T) {
    h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
        return nil, fmt.Errorf("connection refused")
    })
    if h.Installed {
        t.Error("expected Installed=false when daemon unreachable")
    }
}

func TestCheckHealth_Connected_NoCerts(t *testing.T) {
    h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
        return &ipnstate.Status{
            BackendState: "Running",
            TailscaleIPs: []netip.Addr{netip.MustParseAddr("100.64.0.1")},
            CertDomains:  nil,
        }, nil
    })
    if !h.Installed || !h.Connected || h.HasCerts {
        t.Errorf("unexpected health: %+v", h)
    }
}
```

### Pattern 3: Background Health Poller in `app.go`

**What:** Goroutine started in `startup()` that ticks every 10 seconds, calls `CheckHealth`, and emits `tailscale:health` events via `runtime.EventsEmit` when state changes. Mirrors the existing `status.Watch` goroutine pattern already in the codebase.

**When to use:** Enables the frontend to receive automatic updates without polling from JavaScript.

**Example:**

```go
// Source: app.go (addition to startup() or a new startHealthPoller method)
func (a *App) startHealthPoller(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()
        var last webserver.TailscaleHealth
        for {
            select {
            case <-ticker.C:
                checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
                h := webserver.CheckHealth(checkCtx)
                cancel()
                if h != last {
                    last = h
                    if ctx.Value("frontend") != nil {
                        runtime.EventsEmit(ctx, "tailscale:health", h)
                    }
                }
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

Call `a.startHealthPoller(ctx)` from `startup()` alongside `a.initTray()`.

### Anti-Patterns to Avoid

- **Calling `tailscale` CLI as subprocess:** Requires CLI on PATH (fails for macOS App Store Tailscale). Use `local.Client{}` instead.
- **Polling faster than 10 seconds:** Calls go to a local Unix socket so latency is negligible, but there is no need to hammer it. 10-second poll matches the UX requirement (user should not wait long after fixing Tailscale).
- **Distinguishing "not installed" from "not running" via error type:** There is no exported error type for this distinction. STATE.md explicitly documents: "treat connection error as not installed/not running" — both states require the same user action.
- **Caching the `local.Client` as a struct field:** `local.Client` is a value type and zero value is valid. No need to store it — create it per call.
- **Using `context.Background()` without a timeout:** Always wrap with `context.WithTimeout(ctx, 5*time.Second)` to prevent UI hangs if `tailscaled` is slow to respond.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tailscale daemon connectivity check | Custom HTTP client to `/var/run/tailscale/tailscaled.sock` | `local.Client{}.StatusWithoutPeers(ctx)` | Platform socket path varies; Windows uses named pipes; `local.Client` handles all platforms |
| "Is Tailscale connected?" logic | Parse `tailscale status --json` subprocess output | `status.BackendState == "Running"` from `ipnstate.Status` | Subprocess requires PATH; race conditions; no cert info |
| "Are certs enabled?" check | Try/catch `lc.CertPair()` and interpret errors | `len(status.CertDomains) > 0` from `StatusWithoutPeers` response | `CertDomains` is the direct signal; `CertPair` would provision a cert unnecessarily |
| Cross-platform socket path detection | Check `/var/run/tailscale/tailscaled.sock` existence | `local.Client{}` zero value | Handles Unix (macOS/Linux) and Windows named pipe automatically |

**Key insight:** The entire health check implementation fits in ~30 lines of Go using `StatusWithoutPeers`. The Tailscale local client is a thin HTTP-over-socket wrapper — there is nothing complex to implement. The value is in the normalization to `TailscaleHealth` and the testability via function injection.

---

## Common Pitfalls

### Pitfall 1: macOS App Store Tailscale Has No CLI on PATH

**What goes wrong:** `exec.LookPath("tailscale")` returns an error on macOS App Store installs, leading to false "not installed" reports even when Tailscale is running.

**Why it happens:** App Store Tailscale uses a system extension that runs the daemon, but the CLI binary is not placed on PATH.

**How to avoid:** Do NOT use `exec.LookPath` to detect installation. Use `local.Client{}.StatusWithoutPeers()` — if the daemon socket is reachable, Tailscale is running regardless of CLI. If the call fails, treat as "not installed or not running" (same user action either way, per STATE.md decision).

**Warning signs:** Tests passing on a Homebrew Tailscale dev machine, failing for App Store users.

### Pitfall 2: `ipnstate.Status.BackendState` Has Many Non-"Running" Values

**What goes wrong:** Only checking `BackendState != ""` or assuming "Started" means connected.

**Why it happens:** `BackendState` has six values: `"NoState"`, `"NeedsLogin"`, `"NeedsMachineAuth"`, `"Stopped"`, `"Starting"`, `"Running"`. Only `"Running"` means usable.

**How to avoid:** Exact string comparison: `status.BackendState == "Running"`. All other states mean not connected (different instructions in the health modal for different states is a Phase 18 concern — Phase 14 only needs the boolean).

**Warning signs:** User reports health check shows "connected" but web server fails to bind to Tailscale IP.

### Pitfall 3: `CertDomains` Empty Even When Tailscale Is Connected

**What goes wrong:** Assuming `BackendState == "Running"` implies certs are available.

**Why it happens:** HTTPS certificates require a separate admin console action (tailscale.com/admin → DNS → Enable HTTPS). Many users have Tailscale running without HTTPS enabled.

**How to avoid:** Always check `len(status.CertDomains) > 0` as a distinct third gate. Never call `GetCertificate` or `CertPair` before confirming `HasCerts == true`.

**Warning signs:** Web server fails to start with a TLS handshake error or `GetCertificate` returns an error.

### Pitfall 4: Goroutine Leak on App Shutdown

**What goes wrong:** The background poll goroutine continues running after `shutdown()` is called because it holds a reference to a cancelled context but does not observe it.

**Why it happens:** Not adding `case <-ctx.Done(): return` to the goroutine's select.

**How to avoid:** The `ctx` passed to `startHealthPoller` is the Wails app context, which Wails cancels during `OnShutdown`. The goroutine must select on `ctx.Done()` alongside the ticker.

**Warning signs:** Race detector errors in tests; process hangs after Wails window closes.

### Pitfall 5: `runtime.EventsEmit` Called Outside Wails Event Loop

**What goes wrong:** Panic from the Wails runtime when `EventsEmit` is called from a goroutine started before `startup()` completes, or from test code.

**Why it happens:** `EventsEmit` requires the Wails frontend to be initialised. Context background (used in tests) does not have the `"frontend"` key.

**How to avoid:** Guard with the existing project pattern: `if a.ctx != nil && a.ctx.Value("frontend") != nil { runtime.EventsEmit(...) }`. This pattern is already used in `app.go` for `session:status` events.

---

## Code Examples

Verified patterns from actual module source at `$(go env GOPATH)/pkg/mod/tailscale.com@v1.96.3/`.

### Check 1: Daemon reachability (HEALTH-01)

```go
// Source: tailscale.com@v1.96.3/client/local/local.go:677
var lc local.Client
status, err := lc.StatusWithoutPeers(ctx)
if err != nil {
    // tailscaled not reachable — treat as not installed/not running
    return TailscaleHealth{Installed: false}
}
```

### Check 2: Tailnet connection (HEALTH-02)

```go
// Source: tailscale.com@v1.96.3/ipn/ipnstate/ipnstate.go:40-43
// BackendState values: "NoState", "NeedsLogin", "NeedsMachineAuth", "Stopped", "Starting", "Running"
connected := status.BackendState == "Running"
```

### Check 3: HTTPS cert readiness (HEALTH-03)

```go
// Source: tailscale.com@v1.96.3/ipn/ipnstate/ipnstate.go:70-75
// CertDomains: set of DNS names for which control plane assists with TLS cert provisioning.
// Non-empty only when MagicDNS + HTTPS are enabled in admin console.
hasCerts := len(status.CertDomains) > 0
domain := ""
if hasCerts {
    domain = status.CertDomains[0]  // e.g. "hostname.tail46d69a.ts.net"
}
```

### Extracting the Tailscale IP

```go
// Source: tailscale.com@v1.96.3/ipn/ipnstate/ipnstate.go — TailscaleIPs []netip.Addr
ip := ""
if len(status.TailscaleIPs) > 0 {
    ip = status.TailscaleIPs[0].String()  // "100.x.x.x"
}
```

### Wails method binding (HEALTH-06)

```go
// Source: app.go pattern — mirrors existing GetSessionStatus pattern
func (a *App) GetTailscaleStatus() webserver.TailscaleHealth {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return webserver.CheckHealth(ctx)
}
```

### Background poller emitting events (HEALTH-06)

```go
// Mirrors existing status.Watch pattern + EventsEmit guard from app.go:157-163
func (a *App) startHealthPoller(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()
        var last webserver.TailscaleHealth
        for {
            select {
            case <-ticker.C:
                checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
                h := webserver.CheckHealth(checkCtx)
                cancel()
                if h != last {
                    last = h
                    if a.ctx != nil && a.ctx.Value("frontend") != nil {
                        runtime.EventsEmit(a.ctx, "tailscale:health", h)
                    }
                }
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `tailscale.com/client/tailscale` package-level functions | `tailscale.com/client/local.Client` methods | ~v1.20 | Old package is deprecated; all methods are forwarding aliases to `local.Client` |
| `StatusWithPeers` for health checks | `StatusWithoutPeers` | Always was available | `StatusWithoutPeers` omits peer map allocation — lighter for health check use cases |

**Deprecated/outdated:**
- `tailscale.com/client/tailscale`: Deprecated package. All exports are aliases to `local.Client`. Do not import.
- `github.com/tailscale/tscert`: A Caddy compatibility shim for cert-only use on older Go versions. Lacks status/health methods. Not applicable here.

---

## Open Questions

1. **Poll interval: 10 seconds vs configurable**
   - What we know: 10 seconds gives good UX responsiveness (user resolves issue, sees update quickly)
   - What's unclear: Whether CI environments will have Tailscale installed for integration tests
   - Recommendation: Use 10 seconds hardcoded; the background goroutine is not tested with real timing (test infrastructure skips `EventsEmit` when `frontend` key is absent)

2. **HEALTH-06 interprets "automatically without a restart" as events OR polling**
   - What we know: The success criterion says "frontend receives updated state automatically without a restart"
   - What's unclear: Whether the frontend should consume `tailscale:health` events (push) or call `GetTailscaleStatus()` on a JS-side interval (pull)
   - Recommendation: Implement both — background goroutine emits events, frontend also has `GetTailscaleStatus()` available for on-demand polling. Phase 18 decides which the frontend uses.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package |
| Config file | none — `go test ./...` from module root |
| Quick run command | `go test ./internal/webserver/ -run TestCheckHealth -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HEALTH-01 | `CheckHealth` returns `Installed: false` when `statusFn` returns error | unit | `go test ./internal/webserver/ -run TestCheckHealth_NotRunning -v` | Wave 0 |
| HEALTH-02 | `CheckHealth` returns `Connected: true` only when `BackendState == "Running"` | unit | `go test ./internal/webserver/ -run TestCheckHealth_BackendState -v` | Wave 0 |
| HEALTH-03 | `CheckHealth` returns `HasCerts: true` only when `CertDomains` is non-empty | unit | `go test ./internal/webserver/ -run TestCheckHealth_CertDomains -v` | Wave 0 |
| HEALTH-06 | `GetTailscaleStatus()` returns non-panic result on `App` | unit | `go test . -run TestGetTailscaleStatus -v` | Wave 0 |
| HEALTH-06 | Background poller does not leak goroutine on ctx cancellation | unit | `go test . -run TestHealthPollerStops -v -race` | Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/webserver/ ./`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/webserver/tailscale_test.go` — covers HEALTH-01, HEALTH-02, HEALTH-03 with fake `statusFn`
- [ ] `app_test.go` additions — covers HEALTH-06 (`TestGetTailscaleStatus`, `TestHealthPollerStops`)

*(No new test framework needed — existing `go test` infrastructure covers everything.)*

---

## Sources

### Primary (HIGH confidence)

- `$(go env GOPATH)/pkg/mod/tailscale.com@v1.96.3/client/local/local.go` — `StatusWithoutPeers` signature, `Client` struct
- `$(go env GOPATH)/pkg/mod/tailscale.com@v1.96.3/ipn/ipnstate/ipnstate.go` — `Status.BackendState` values (lines 40-43), `Status.CertDomains` (lines 70-75), `Status.TailscaleIPs` field
- `$(go env GOPATH)/pkg/mod/tailscale.com@v1.96.3/client/local/cert.go` — `GetCertificate`, `CertPair` signatures
- [pkg.go.dev — tailscale.com/client/local](https://pkg.go.dev/tailscale.com/client/local) — method list confirmed against source
- `go build` binary size measurement: before 9,166,098 bytes → after 9,695,986 bytes (+529,888 bytes = +0.5 MB)

### Secondary (MEDIUM confidence)

- `.planning/research/STACK.md` — prior v1.2 milestone stack research; all key findings verified against source in this pass
- `.planning/research/ARCHITECTURE.md` — prior architecture research for entire v1.2 milestone; Phase 14 section verified against actual codebase
- [pkg.go.dev — tailscale.com/ipn/ipnstate#Status](https://pkg.go.dev/tailscale.com/ipn/ipnstate#Status) — `BackendState` values documented

### Internal (HIGH confidence — direct codebase inspection)

- `/Users/ken/dev/agenthub/app.go` — existing `EventsEmit` guard pattern (lines 157-163, 220-224), `startup()` structure
- `/Users/ken/dev/agenthub/internal/status/detector.go` — `Watch` goroutine pattern to mirror
- `/Users/ken/dev/agenthub/internal/webserver/server.go` — existing `WebServer` struct (unchanged in this phase)
- `/Users/ken/dev/agenthub/app_test.go` — test helper patterns (`testApp`, `context.Background()` usage)

---

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — `tailscale.com/client/local@v1.96.3` verified by compiling against it; `ipnstate.Status` fields verified by reading source
- Architecture: HIGH — `CheckHealth` pattern taken directly from prior architecture research, verified against actual module source; goroutine pattern mirrors existing codebase
- Pitfalls: HIGH — macOS App Store pitfall documented in STATE.md; others verified against ipnstate source and existing codebase patterns

**Research date:** 2026-03-20
**Valid until:** 2026-04-20 (tailscale.com releases frequently; re-check version before Phase 15)
