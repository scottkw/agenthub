# Phase 50: Tailscale Peer Discovery - Research

**Researched:** 2026-04-07
**Domain:** Go — Tailscale LocalAPI, concurrent HTTP probing, daemon route extension
**Confidence:** HIGH

## Summary

Phase 50 creates `internal/tailnet` — a pure Go package that discovers online tailnet peers and probes which ones are running AgentHub. It has no UI dependency; the only surface that touches the existing codebase is a new `GET /tailnet/peers` route added to the daemon's Unix-socket API.

The project already uses `tailscale.com/client/local` and `tailscale.com/ipn/ipnstate` extensively in `internal/webserver/tailscale.go`. The injectable-`statusFunc` pattern there is the exact template for the new package: expose a pure inner function that accepts an injected func, wrap it in a public function that calls `local.Client{}.Status()`. `golang.org/x/sync/errgroup` is already an indirect dependency and provides `SetLimit(5)` for the goroutine pool.

The single non-trivial constraint is TLS. Tailscale HTTPS certificates are issued by Let's Encrypt for the peer's FQDN (e.g. `host.tail46d69a.ts.net`) and contain no IP SANs. Probing via IP address fails with a TLS handshake error. The probe URL must use `DNSName` (with trailing dot stripped) as the host. This is confirmed by the STATE.md blocker: "Confirm Tailscale Let's Encrypt certs are FQDN-only (no IP SANs)". Research confirms this: FQDN-only.

**Primary recommendation:** Model `internal/tailnet` exactly on `internal/webserver/tailscale.go` — injectable status func, no live daemon in tests. Use `httptest.NewTLSServer` + `srv.Client()` to test `ProbePeer` without a real Tailscale peer.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REM-01 | User can discover AgentHub instances running on other tailnet peers automatically | `DiscoverPeers()` via `local.Client{}.Status()`, `ProbePeer()` via HTTPS GET `/api/sessions`, daemon `GET /tailnet/peers` with 30s cache |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `tailscale.com/client/local` | v1.96.3 (already in go.mod) | Query local tailscaled for peer list | Already used in webserver package |
| `tailscale.com/ipn/ipnstate` | v1.96.3 (already in go.mod) | `ipnstate.Status`, `ipnstate.PeerStatus` types | Already used in webserver package |
| `golang.org/x/sync/errgroup` | v0.19.0 (already in go.mod) | Goroutine pool with `SetLimit(5)` | Already an indirect dep; `SetLimit` avoids manual semaphore |
| `net/http/httptest` | stdlib | TLS test server for `ProbePeer` tests | Standard Go testing tool |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `sync` stdlib | — | `sync.RWMutex` for 30-second result cache in daemon | For cache struct guarding cached result and timestamp |
| `time` stdlib | — | Cache TTL check (`time.Since(cachedAt) < 30*time.Second`) | TTL expiry in cache |
| `crypto/tls` stdlib | — | `tls.Config` for probe HTTP client | Needed to trust system CAs for Tailscale Let's Encrypt certs |

**Installation:** No new dependencies required. All are in go.mod already.

## Architecture Patterns

### Recommended Package Structure
```
internal/tailnet/
├── tailnet.go         # DiscoverPeers(), ProbePeer(), injectable funcs, types
└── tailnet_test.go    # 100% function coverage, go test -race
```

The daemon route and cache live in `internal/daemon/api.go` (new handler) and a new `internal/daemon/tailnet_cache.go` (cache struct), following the existing file-per-concern pattern.

### Pattern 1: Injectable Status Function (mirrors webserver/tailscale.go)

**What:** Expose a `statusFunc` type alias and a pure inner function for testability. The public function binds the live `local.Client{}.Status`.

**When to use:** Any function that calls tailscaled; enables unit tests without a live daemon.

**Example:**
```go
// Source: internal/webserver/tailscale.go (existing project pattern)
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

func discoverPeers(ctx context.Context, fn statusFunc) ([]Peer, error) {
    status, err := fn(ctx)
    if err != nil {
        return nil, err
    }
    peers := make([]Peer, 0)
    for _, p := range status.Peer {
        if p.Online {
            peers = append(peers, peerFromStatus(p))
        }
    }
    return peers, nil
}

// Public entry point — no injectable needed here; callers inject via probeFunc
func DiscoverPeers(ctx context.Context) ([]Peer, error) {
    var lc local.Client
    return discoverPeers(ctx, lc.Status)
}
```

### Pattern 2: Goroutine Pool with errgroup.SetLimit

**What:** Use `errgroup.WithContext` + `g.SetLimit(5)` to run peer probes concurrently capped at 5 active goroutines.

**When to use:** Any fan-out operation with a bounded concurrency requirement.

**Example:**
```go
// Source: golang.org/x/sync/errgroup (SetLimit available since v0.1.0)
type probeFunc func(ctx context.Context, peer Peer) bool

func probeAll(ctx context.Context, peers []Peer, fn probeFunc) []Peer {
    var mu sync.Mutex
    var found []Peer

    g, gctx := errgroup.WithContext(ctx)
    g.SetLimit(5)

    for _, p := range peers {
        p := p // capture loop variable (pre-Go 1.22 requirement; harmless in 1.26)
        g.Go(func() error {
            if fn(gctx, p) {
                mu.Lock()
                found = append(found, p)
                mu.Unlock()
            }
            return nil
        })
    }
    _ = g.Wait()
    return found
}
```

Note: In Go 1.22+ loop variable capture is automatic, but explicit capture (`p := p`) is still idiomatic and harmless.

### Pattern 3: HTTPS Probe with 2-Second Timeout

**What:** Create a `*http.Client` with 2-second timeout and system-CA TLS config, probe the peer's `/api/sessions` endpoint on its Tailscale FQDN.

**Critical constraint:** Tailscale Let's Encrypt certs are FQDN-only (no IP SANs). Must use `DNSName` field (strip trailing dot), not `TailscaleIPs[0]`.

**Default port:** 7443 (matches `SettingsPanel.tsx` default; probe should try this port).

**Example:**
```go
// Source: internal/webserver/server.go (project pattern), stdlib docs
type probeFunc func(ctx context.Context, peer Peer) bool

func probePeer(ctx context.Context, peer Peer, client *http.Client) bool {
    // DNSName ends with a dot — strip it.
    host := strings.TrimSuffix(peer.DNSName, ".")
    url := fmt.Sprintf("https://%s:%d/api/sessions", host, DefaultProbePort)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return false
    }
    resp, err := client.Do(req)
    if err != nil {
        return false
    }
    resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}

func ProbePeer(ctx context.Context, peer Peer) bool {
    probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    client := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
        },
    }
    return probePeer(probeCtx, peer, client)
}
```

The injectable `probeFunc` for tests should accept a pre-built `*http.Client` (from `httptest.NewTLSServer`).

### Pattern 4: 30-Second Result Cache in Daemon

**What:** A simple struct with `sync.RWMutex`, the cached result slice, and a `time.Time` of last population. On each request, check `time.Since(cachedAt) < 30*time.Second`; if stale, run discovery+probe and repopulate.

**When to use:** This is the correct approach given that STATE.md says "HTTP polling is simpler and sufficient for v1.9" and "Real-time push notifications... deferred."

**Example:**
```go
// Source: internal/daemon/api.go (sync.RWMutex cache pattern already established)
type tailnetCache struct {
    mu       sync.RWMutex
    result   []tailnet.Peer
    cachedAt time.Time
}

const cacheTTL = 30 * time.Second

func (c *tailnetCache) get() ([]tailnet.Peer, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    if time.Since(c.cachedAt) < cacheTTL {
        return c.result, true
    }
    return nil, false
}

func (c *tailnetCache) set(peers []tailnet.Peer) {
    c.mu.Lock()
    c.result = peers
    c.cachedAt = time.Now()
    c.mu.Unlock()
}
```

### Peer Type

```go
// internal/tailnet/tailnet.go
type Peer struct {
    Hostname     string   // PeerStatus.HostName
    DNSName      string   // PeerStatus.DNSName (with trailing dot — strip for URLs)
    TailscaleIPs []string // PeerStatus.TailscaleIPs as strings
    OS           string   // PeerStatus.OS
    Online       bool     // PeerStatus.Online
}
```

### PeerStatus.Online Semantics

`PeerStatus.Online` is `true` when the node is connected to the Tailscale control plane. This is the correct field to filter by — it excludes stale/offline entries from the peer map. (`LastSeen` is only populated for offline nodes; `Active` means recent packet activity, which is stricter than desired.)

### Daemon Route

New route in `api.go`:
```go
a.mux.HandleFunc("GET /tailnet/peers", a.handleTailnetPeers)
```

Handler: check cache, if stale run `tailnet.DiscoverAndProbe(ctx)` (which calls `DiscoverPeers` then `probeAll`), populate cache, return JSON array. Non-blocking because the handler itself runs the discovery synchronously but returns the cache result. The "does not block the calling goroutine" success criterion means the goroutine pool doesn't block the caller's goroutine — `g.Wait()` is called inside the handler (acceptable), but peer probes run concurrently within that wait.

### Anti-Patterns to Avoid

- **Probing via IP address:** Tailscale certs have no IP SANs. Use `DNSName` (stripped of trailing dot).
- **Using `StatusWithoutPeers`:** This omits the `Peer` map. Must call `lc.Status()` (full status).
- **`InsecureSkipVerify: true` in probe client:** Tailscale Let's Encrypt certs are valid — system CAs trust them. Using `InsecureSkipVerify` hides real connectivity problems and is a security smell. Do not use it.
- **Blocking on probe errors:** Each `ProbePeer` call must respect context cancellation and the 2-second timeout, not hang indefinitely.
- **Shared mutable state in goroutines without mutex:** Accumulating results from concurrent probes requires a `sync.Mutex` around the append.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Concurrency limit | Custom semaphore channel | `errgroup.SetLimit(5)` | Already in go.mod; cleaner and race-safe |
| TLS cert management | Custom cert pool | System CA (default `http.Transport`) | Tailscale LE certs are publicly trusted |
| Peer list | Tailscale Services API | `local.Client{}.Status()` | STATE.md explicitly rules out the alpha Services API |

**Key insight:** The injectable-function testing pattern already used in `webserver/tailscale.go` eliminates the need for test doubles or mocking frameworks.

## Common Pitfalls

### Pitfall 1: DNSName Trailing Dot
**What goes wrong:** Using `peer.DNSName` directly in a URL produces `https://host.ts.net.:7443/api/sessions` — the trailing dot makes URL parsing or TLS SNI fail silently.
**Why it happens:** The `PeerStatus.DNSName` field is documented as "ends with a dot."
**How to avoid:** Always `strings.TrimSuffix(peer.DNSName, ".")` before building URLs.
**Warning signs:** HTTP client error "no such host" or TLS dial failure when using DNSName directly.

### Pitfall 2: Using StatusWithoutPeers
**What goes wrong:** `local.Client{}.StatusWithoutPeers()` returns a Status with an empty `Peer` map — `DiscoverPeers` returns nothing.
**Why it happens:** The existing `CheckHealth` in `webserver/tailscale.go` uses `StatusWithoutPeers` (it only needs local status). Phase 50 needs full peer list.
**How to avoid:** Inject `lc.Status` (not `lc.StatusWithoutPeers`) into `discoverPeers`.
**Warning signs:** `DiscoverPeers` always returns an empty slice in integration testing.

### Pitfall 3: Race on Result Accumulation
**What goes wrong:** Multiple goroutines append to the same `[]Peer` slice without synchronization → data race, detected by `go test -race`.
**Why it happens:** errgroup goroutines run concurrently; slice append is not atomic.
**How to avoid:** Use a `sync.Mutex` around the append inside the goroutine, as shown in Pattern 2.
**Warning signs:** `go test -race` reports data race on `found` slice.

### Pitfall 4: 100% Function Coverage Requirement
**What goes wrong:** The public wrappers `DiscoverPeers()` and `ProbePeer()` are not tested because tests only call the inner functions.
**Why it happens:** Testing inner functions is easy; testing public wrappers requires live tailscaled or careful injection.
**How to avoid:** Test public wrappers too — either with build tag guards or by also having one test that exercises the public function path (even if it expects an error due to no live daemon).
**Warning signs:** `go test -covermode=atomic -coverprofile=c.out` shows public functions at 0%.

### Pitfall 5: Cache Thundering Herd
**What goes wrong:** Multiple concurrent HTTP requests arrive when cache is stale → all trigger simultaneous discovery+probe calls.
**Why it happens:** Check-then-act without a write lock.
**How to avoid:** Use `sync.Mutex` (write lock, not RWMutex) around the full stale-check-and-populate cycle, or use `sync.Once`-style refresh. Simple approach: take write lock for the stale check + populate; accept that one request blocks others briefly.
**Warning signs:** Multiple simultaneous requests to `/tailnet/peers` when cache expires trigger O(N) simultaneous probe goroutine pools.

## Code Examples

### tailnet.go Skeleton
```go
// Source: mirrors internal/webserver/tailscale.go pattern
package tailnet

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "sync"
    "time"
    "crypto/tls"

    "golang.org/x/sync/errgroup"
    "tailscale.com/client/local"
    "tailscale.com/ipn/ipnstate"
)

const DefaultProbePort = 7443

type Peer struct {
    Hostname     string   `json:"hostname"`
    DNSName      string   `json:"dnsName"`      // FQDN with trailing dot
    TailscaleIPs []string `json:"tailscaleIPs"`
    OS           string   `json:"os"`
    Online       bool     `json:"online"`
}

type statusFunc func(ctx context.Context) (*ipnstate.Status, error)
type probeFunc  func(ctx context.Context, peer Peer) bool

func discoverPeers(ctx context.Context, fn statusFunc) ([]Peer, error) {
    status, err := fn(ctx)
    if err != nil {
        return nil, err
    }
    peers := make([]Peer, 0, len(status.Peer))
    for _, p := range status.Peer {
        if !p.Online {
            continue
        }
        pr := Peer{
            Hostname: p.HostName,
            DNSName:  p.DNSName,
            OS:       p.OS,
            Online:   true,
        }
        for _, ip := range p.TailscaleIPs {
            pr.TailscaleIPs = append(pr.TailscaleIPs, ip.String())
        }
        peers = append(peers, pr)
    }
    return peers, nil
}

func DiscoverPeers(ctx context.Context) ([]Peer, error) {
    var lc local.Client
    return discoverPeers(ctx, lc.Status)
}

func probeAll(ctx context.Context, peers []Peer, fn probeFunc) []Peer {
    var mu sync.Mutex
    var found []Peer

    g, gctx := errgroup.WithContext(ctx)
    g.SetLimit(5)
    for _, p := range peers {
        p := p
        g.Go(func() error {
            if fn(gctx, p) {
                mu.Lock()
                found = append(found, p)
                mu.Unlock()
            }
            return nil
        })
    }
    _ = g.Wait()
    return found
}

func probePeer(ctx context.Context, peer Peer, client *http.Client) bool {
    host := strings.TrimSuffix(peer.DNSName, ".")
    url := fmt.Sprintf("https://%s:%d/api/sessions", host, DefaultProbePort)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return false
    }
    resp, err := client.Do(req)
    if err != nil {
        return false
    }
    resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}

func ProbePeer(ctx context.Context, peer Peer) bool {
    probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    client := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
        },
    }
    return probePeer(probeCtx, peer, client)
}

func DiscoverAndProbe(ctx context.Context) ([]Peer, error) {
    peers, err := DiscoverPeers(ctx)
    if err != nil {
        return nil, err
    }
    return probeAll(ctx, peers, ProbePeer), nil
}
```

### tailnet_test.go Key Patterns
```go
// Source: internal/webserver/tailscale_test.go (injectable pattern)
//         net/http/httptest (TLS server for probe tests)

// Test discoverPeers via injectable statusFunc
func TestDiscoverPeers_OnlineOnly(t *testing.T) {
    peers, err := discoverPeers(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
        return &ipnstate.Status{
            Peer: map[key.NodePublic]*ipnstate.PeerStatus{
                {}: {HostName: "online-host", DNSName: "online-host.ts.net.", Online: true},
                {}: {HostName: "offline-host", DNSName: "offline-host.ts.net.", Online: false},
            },
        }, nil
    })
    // assert len(peers) == 1, peers[0].Hostname == "online-host"
}

// Test probePeer via httptest.NewTLSServer + srv.Client()
func TestProbePeer_Found(t *testing.T) {
    srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/sessions" {
            w.WriteHeader(http.StatusOK)
            return
        }
        http.NotFound(w, r)
    }))
    defer srv.Close()

    // Extract host from srv.URL; construct a Peer with DNSName pointing to loopback
    // Override DefaultProbePort by using the injectable probePeer(ctx, peer, client)
    peer := Peer{DNSName: "127.0.0.1."} // probe uses loopback
    result := probePeer(context.Background(), peer, srv.Client())
    // assert result == true
}
```

Note: `httptest.NewTLSServer` uses a self-signed cert; `srv.Client()` is pre-configured to trust it. This is the correct approach for testing probe logic without a real Tailscale peer.

### Daemon Route Addition
```go
// In internal/daemon/api.go registerRoutes()
a.mux.HandleFunc("GET /tailnet/peers", a.handleTailnetPeers)

// In internal/daemon/api.go (new handler)
func (a *API) handleTailnetPeers(w http.ResponseWriter, r *http.Request) {
    if cached, ok := a.tailnetCache.get(); ok {
        writeJSON(w, http.StatusOK, cached)
        return
    }
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()
    peers, err := tailnet.DiscoverAndProbe(ctx)
    if err != nil {
        // Return empty slice, not 500 — Tailscale may not be connected
        peers = []tailnet.Peer{}
    }
    a.tailnetCache.set(peers)
    writeJSON(w, http.StatusOK, peers)
}
```

### DaemonClient Method Addition
```go
// In internal/daemon/client.go
func (c *DaemonClient) ListTailnetPeers() ([]tailnet.Peer, error) {
    var peers []tailnet.Peer
    if err := c.doJSON(http.MethodGet, "/tailnet/peers", nil, &peers); err != nil {
        return nil, err
    }
    if peers == nil {
        peers = []tailnet.Peer{}
    }
    return peers, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Tailscale Services API (alpha) | `local.Client{}.Status()` + HTTP probe | Always — Services API was never stable | STATE.md explicitly rules out the alpha API |
| `StatusWithoutPeers` | `Status()` with full peer map | N/A | Must use `Status()` to get `Peer` map |

**Deprecated/outdated:**
- Tailscale Services API for peer discovery: Alpha/unstable, explicitly excluded per REQUIREMENTS.md "Out of Scope" table.

## Open Questions

1. **Port discovery — what if a peer runs AgentHub on a non-default port?**
   - What we know: Default port is 7443 (from `SettingsPanel.tsx`). Probe targets this port.
   - What's unclear: If a peer chose a different port (EADDRINUSE fallback in webserver), it's not discoverable via this approach.
   - Recommendation: Phase 50 probes port 7443 only. Non-default ports are a v2+ problem. Document this assumption.

2. **Thundering herd on cache expiry**
   - What we know: Using `sync.Mutex` write-lock for the full stale check+populate is simplest.
   - What's unclear: If probing takes several seconds, concurrent requests block at the mutex.
   - Recommendation: Use a simple mutex around full stale-check-and-populate. Phase 50 doesn't need background refresh; acceptable to block for up to 10 seconds (discovery timeout).

3. **TLS: system CA vs. InsecureSkipVerify**
   - What we know: Tailscale certs are issued by Let's Encrypt → trusted by system CAs. FQDN-only (no IP SANs) confirmed by research.
   - What's unclear: Whether the probe HTTP client on macOS dev machine needs any special configuration (should just work with default transport).
   - Recommendation: Use default TLS (system CAs). Do NOT use `InsecureSkipVerify`. If probe tests fail in CI, the `httptest.NewTLSServer` + `srv.Client()` pattern isolates the concern.

## Environment Availability

Step 2.6: SKIPPED — Phase 50 is purely code changes with no new external dependencies. All required tools (Go 1.26.1, existing `tailscale.com` module) are confirmed present.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none (go test invoked directly) |
| Quick run command | `go test ./internal/tailnet/... -race` |
| Full suite command | `go test ./... -race` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REM-01 | `DiscoverPeers` returns only online peers | unit | `go test ./internal/tailnet/... -race -run TestDiscoverPeers` | ❌ Wave 0 |
| REM-01 | `ProbePeer` detects running AgentHub | unit | `go test ./internal/tailnet/... -race -run TestProbePeer` | ❌ Wave 0 |
| REM-01 | Concurrent probes capped at 5 goroutines | unit | `go test ./internal/tailnet/... -race -run TestProbeAll` | ❌ Wave 0 |
| REM-01 | `GET /tailnet/peers` daemon route with cache | unit | `go test ./internal/daemon/... -race -run TestTailnetPeers` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/tailnet/... -race`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** `go test ./... -race` green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/tailnet/tailnet.go` — package does not exist yet; must be created in Wave 0
- [ ] `internal/tailnet/tailnet_test.go` — test file for 100% function coverage
- [ ] No framework install needed — Go stdlib test tooling confirmed available

## Sources

### Primary (HIGH confidence)
- `go doc tailscale.com/ipn/ipnstate PeerStatus` — PeerStatus struct with `Online` bool, `DNSName` ("ends with a dot"), `TailscaleIPs`
- `go doc tailscale.com/ipn/ipnstate Status` — Status.Peer map, BackendState
- `go doc tailscale.com/client/local Client` — local.Client zero value, `Status()` vs `StatusWithoutPeers()`
- `go doc golang.org/x/sync/errgroup Group.SetLimit` — SetLimit(n) for goroutine cap
- `go doc net/http/httptest NewTLSServer` + `Server.Client()` — TLS test server pattern
- `internal/webserver/tailscale.go` — project's injectable statusFunc pattern (direct code inspection)
- `internal/daemon/api.go` — route registration pattern, writeJSON, cache pattern with sync.RWMutex
- `go.mod` — confirmed all required libraries already present

### Secondary (MEDIUM confidence)
- WebSearch: "Tailscale TLS certificate SAN IP addresses vs FQDN ts.net" — confirmed FQDN-only SANs, no IP SANs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in go.mod, verified with `go doc`
- Architecture: HIGH — directly mirrors existing `webserver/tailscale.go` pattern in same project
- Pitfalls: HIGH — DNSName trailing dot and StatusWithoutPeers confirmed from `go doc`; race condition is a known Go pattern
- TLS probe approach: MEDIUM-HIGH — FQDN-only confirmed by research, but exact runtime behavior on dev machine not tested (httptest pattern isolates this)

**Research date:** 2026-04-07
**Valid until:** 2026-05-07 (tailscale.com module stable; go.mod version locked)
