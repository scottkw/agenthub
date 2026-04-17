# Phase 80: Tailscale Detection - Pattern Map

**Mapped:** 2026-04-16
**Files analyzed:** 6 new/modified files
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/webserver/tailscale.go` | service | request-response | `internal/webserver/tailscale.go` itself | exact (extend) |
| `internal/webserver/tailscale_paths.go` | utility | transform | `internal/daemon/path_windows.go` + `path_other.go` | role-match |
| `internal/webserver/tailscale_test.go` | test | request-response | `internal/webserver/tailscale_test.go` itself | exact (extend) |
| `frontend/src/components/SettingsTab.tsx` | component | request-response | `frontend/src/components/SettingsTab.tsx` itself | exact (extend) |
| `frontend/src/components/LocalNetworkBanner.tsx` | component | request-response | `frontend/src/components/LocalNetworkBanner.tsx` itself | exact (extend) |
| `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` | test | request-response | `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` itself | exact (extend) |

## Pattern Assignments

### `internal/webserver/tailscale.go` (service, request-response) — MODIFY

**Analog:** `internal/webserver/tailscale.go` (current file, extend in place)

**Current struct pattern** (lines 12-18):
```go
type TailscaleHealth struct {
	Installed bool   `json:"installed"` // tailscaled socket reachable
	Connected bool   `json:"connected"` // BackendState == "Running"
	HasCerts  bool   `json:"hasCerts"`  // len(CertDomains) > 0
	IP        string `json:"ip"`        // first TailscaleIP as string, empty if not connected
	Domain    string `json:"domain"`    // first CertDomain e.g. hostname.ts.net, empty if none
}
```

**Add these fields to TailscaleHealth:**
```go
BinaryFound  bool   `json:"binaryFound"`  // binary exists on disk (new step 1)
DaemonUp     bool   `json:"daemonUp"`     // daemon socket reachable (was "Installed")
PlatformHint string `json:"platformHint"` // "macos" | "linux" | "windows" for frontend instructions
```

**Injectable statusFunc pattern** (lines 21-39) — preserve exactly, extend cascade:
```go
// statusFunc is the injectable status function type for testability.
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

// checkHealth is the internal testable health check function.
// It accepts an injected statusFn so tests can pass a fake without a live daemon.
func checkHealth(ctx context.Context, fn statusFunc) TailscaleHealth {
	status, err := fn(ctx)
	if err != nil {
		return TailscaleHealth{Installed: false}
	}
	h := TailscaleHealth{Installed: true}
	h.Connected = status.BackendState == "Running"
	// ...
}
```

**New 4-state cascade shape to implement:**
```go
func checkHealth(ctx context.Context, customPath string, fn statusFunc) TailscaleHealth {
	h := TailscaleHealth{PlatformHint: runtime.GOOS}

	// Step 1: Binary detection — custom path → well-known → PATH
	binary := detectTailscaleBinary(customPath)
	if binary == "" {
		return h // BinaryFound=false, DaemonUp=false, Installed=false, Connected=false
	}
	h.BinaryFound = true

	// Step 2: Daemon socket probe
	status, err := fn(ctx)
	if err != nil {
		return h // BinaryFound=true, DaemonUp=false, Installed=false, Connected=false
	}
	h.DaemonUp = true
	h.Installed = true // legacy compat: "Installed" now means "daemon up"

	// Step 3: Connection state
	h.Connected = status.BackendState == "Running"

	// Step 4: Certs (only if connected)
	if h.Connected {
		h.HasCerts = len(status.CertDomains) > 0
		if len(status.TailscaleIPs) > 0 {
			h.IP = status.TailscaleIPs[0].String()
		}
		if len(status.CertDomains) > 0 {
			h.Domain = status.CertDomains[0]
		}
	}
	return h
}
```

**Public API** (lines 44-47) — extend to read custom path from settings:
```go
// CheckHealth queries tailscaled via local.Client and returns TailscaleHealth.
func CheckHealth(ctx context.Context) TailscaleHealth {
	var lc local.Client
	return checkHealth(ctx, lc.StatusWithoutPeers)
}
```

---

### `internal/webserver/tailscale_paths.go` (utility, transform) — NEW FILE

**Analog:** `internal/daemon/path_windows.go` (build tag pattern) + `internal/daemon/path_other.go`

**Build tag pattern from `path_windows.go`** (lines 1-23):
```go
//go:build windows

package daemon

import (
	"os"
	"path/filepath"
)

func platformExtraBins() []string {
	var paths []string
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		paths = append(paths, filepath.Join(appdata, "npm"))
	}
	// ...
	paths = append(paths, `C:\Program Files\Tailscale`)
	return paths
}
```

**Companion no-op from `path_other.go`** (lines 1-9):
```go
//go:build !windows

package daemon

func platformExtraBins() []string {
	return nil
}
```

**Pattern to apply — single file using runtime.GOOS (per Claude's Discretion — no build tags needed since paths are read at runtime):**
```go
package webserver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// tailscaleWellKnownPaths returns the ordered list of well-known binary paths
// for the current platform. Detection order: custom path → these → PATH.
func tailscaleWellKnownPaths() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Tailscale.app/Contents/MacOS/Tailscale",
			"/opt/homebrew/bin/tailscale",
			"/usr/local/bin/tailscale",
		}
	case "linux":
		return []string{
			"/usr/bin/tailscale",
			"/usr/sbin/tailscale",
			"/snap/bin/tailscale",
			"/var/lib/flatpak/exports/bin/tailscale",
			filepath.Join(home, ".local", "share", "flatpak", "exports", "bin", "tailscale"),
		}
	case "windows":
		return []string{
			`C:\Program Files\Tailscale\tailscale.exe`,
			`C:\Program Files (x86)\Tailscale\tailscale.exe`,
		}
	}
	return nil
}

// detectTailscaleBinary finds the tailscale binary. Detection order:
//  1. customPath (non-empty, must exist and be executable)
//  2. Well-known platform paths (tailscaleWellKnownPaths)
//  3. PATH (exec.LookPath)
//
// Returns the resolved path or "" if not found.
func detectTailscaleBinary(customPath string) string {
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			return customPath
		}
	}
	for _, p := range tailscaleWellKnownPaths() {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("tailscale"); err == nil {
		return p
	}
	return ""
}
```

---

### `internal/webserver/tailscale_test.go` (test, request-response) — MODIFY

**Analog:** `internal/webserver/tailscale_test.go` (current file, extend in place)

**Inject pattern — all tests use this shape** (lines 12-32):
```go
func TestCheckHealth_NotRunning(t *testing.T) {
	h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
		return nil, fmt.Errorf("dial unix: connection refused")
	})

	if h.Installed {
		t.Error("expected Installed=false when daemon unreachable")
	}
}
```

**Table-driven pattern for multiple states** (lines 34-60):
```go
func TestCheckHealth_BackendState(t *testing.T) {
	tests := []struct {
		state         string
		wantConnected bool
	}{
		{state: "Stopped", wantConnected: false},
		{state: "Running", wantConnected: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.state, func(t *testing.T) {
			h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
				return &ipnstate.Status{BackendState: tc.state}, nil
			}, "")
			// assertions...
		})
	}
}
```

**New tests to add — same inject pattern, 4 new state scenarios:**
```go
// Binary not found → all false
func TestCheckHealth_BinaryNotFound(t *testing.T) {
	h := checkHealth(context.Background(), /* fn never called */ nil, "/nonexistent/tailscale")
	if h.BinaryFound { t.Error("expected BinaryFound=false") }
	if h.DaemonUp   { t.Error("expected DaemonUp=false") }
}

// Binary found, daemon unreachable → BinaryFound=true, DaemonUp=false
func TestCheckHealth_DaemonStopped(t *testing.T) {
	// uses a temp dir stub binary so os.Stat finds it
	// then fn returns an error
}

// Binary found, daemon up, not connected → DaemonUp=true, Connected=false
// Binary found, daemon up, connected → all true
```

**Filesystem mock pattern (from detect_test.go lines 12-39):**
```go
// Write a stub executable so os.Stat finds it
dir := t.TempDir()
stubPath := filepath.Join(dir, "tailscale")
if err := os.WriteFile(stubPath, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
    t.Fatalf("writing stub: %v", err)
}
// Pass stubPath as customPath to checkHealth(ctx, fn, stubPath)
```

---

### `frontend/src/components/SettingsTab.tsx` (component, request-response) — MODIFY

**Analog:** `frontend/src/components/SettingsTab.tsx` (current file, extend in place)

**Imports pattern** (lines 1-22):
```typescript
import React, { useState, useEffect } from 'react'
import {
  UpdateCLIPath,
  GetCLIPaths,
  OpenFileDialog,
  // ...
} from '../wailsjs/go/main/App'
```

**Props interface extension** (lines 26-39) — add `daemonUp` to tailscaleHealth shape:
```typescript
interface SettingsTabProps {
  tailscaleHealth: {
    installed: boolean
    connected: boolean
    hasCerts: boolean
    ip: string
    domain: string
    // NEW:
    binaryFound: boolean
    daemonUp: boolean
    platformHint: string
  } | null
  // ...
}
```

**Status class function** (lines 159-164) — extend to 4 states:
```typescript
function tailscaleStatusClass(h: SettingsTabProps['tailscaleHealth']): string {
  if (!h) return ''
  if (h.installed && h.connected) return 'ok'
  if (h.installed) return 'warn'
  return 'error'
}
// → extend to handle h.binaryFound and h.daemonUp for 4 states
```

**Status text function** (lines 166-171) — extend to 4 states:
```typescript
function tailscaleStatusText(h: SettingsTabProps['tailscaleHealth']): string {
  if (!h) return 'Checking\u2026'
  if (h.installed && h.connected) return 'Connected'
  if (h.installed) return 'Not Connected'
  return 'Not Installed'
}
// → extend: Not Installed / Installed (daemon stopped) / Running (disconnected) / Connected
```

**Tailscale status block in JSX** (lines 265-282) — add checklist details:
```tsx
<div className="settings-panel__field-group">
  <label className="settings-panel__label">Tailscale Status</label>
  <div className="ts-status">
    {tailscaleHealth && (
      <span className={`ts-status__dot ts-status__dot--${tailscaleStatusClass(tailscaleHealth)}`} />
    )}
    <span className="ts-status__text">{tailscaleStatusText(tailscaleHealth)}</span>
  </div>
  <p className="settings-panel__description" ...>
    {/* platform-specific hint text */}
  </p>
  {/* NEW: stepped checklist — only when not connected */}
</div>
```

**Tailscale path row** (lines 454-492) — existing pattern, already present, no change needed for the path input itself. The `customPaths['tailscale']` + `handleBrowse('tailscale')` pattern is already implemented.

---

### `frontend/src/components/LocalNetworkBanner.tsx` (component, request-response) — MODIFY

**Analog:** `frontend/src/components/LocalNetworkBanner.tsx` (current file, extend in place)

**Props interface** (lines 3-8) — add `daemonUp` prop:
```typescript
interface LocalNetworkBannerProps {
  visible: boolean
  tailscaleConnected: boolean
  tailscaleInstalled: boolean   // currently means "binary found OR daemon up"
  onOpenURL: (url: string) => void
}
// → rename/add: tailscaleDaemonUp: boolean to distinguish "binary exists but daemon stopped"
```

**3-state branch pattern** (lines 25-71) — add 4th state for daemon stopped:
```typescript
// Current pattern — 3 branches:
if (tailscaleConnected) { return <upgrading banner> }
if (tailscaleInstalled) { return <start tailscale banner> }
return <install tailscale banner>

// D-06 new pattern — add daemon-stopped branch:
if (tailscaleConnected) { return <upgrading banner> }
if (tailscaleDaemonUp) { return <start tailscale banner (existing)> }
if (tailscaleBinaryFound) { return <daemon stopped banner — platform-specific text, no button> }
return <install tailscale banner>
```

**No action buttons for daemon-stopped state (D-06):**
```tsx
// "daemon stopped" state — text only, no CTA button:
<div className="local-network-banner" role="status">
  <span className="local-network-banner__icon">{'\u26a0'}</span>
  <span className="local-network-banner__message">
    Tailscale installed — daemon not running.
  </span>
  <span className="local-network-banner__sub">
    {platformHint === 'darwin' && 'Open Tailscale from Applications or the menu bar.'}
    {platformHint === 'linux' && 'Run: sudo systemctl start tailscaled'}
    {platformHint === 'windows' && 'Open Tailscale from the Start menu or system tray.'}
  </span>
</div>
```

---

### `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` (test) — MODIFY

**Analog:** `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` (current file, extend in place)

**Test render helper pattern** (lines 7-15):
```typescript
function renderBanner(props: { visible: boolean; tailscaleConnected: boolean; tailscaleInstalled: boolean; onOpenURL: (url: string) => void }) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(LocalNetworkBanner, props))
  })
  return { container, root }
}
```

**Assertion style** (lines 26-50):
```typescript
it('renders Install Tailscale when not installed and not connected', () => {
  const onOpenURL = vi.fn()
  ;({ container, root } = renderBanner({ visible: true, tailscaleConnected: false, tailscaleInstalled: false, onOpenURL }))
  expect(container.textContent).toContain('Install Tailscale')
  const ctaBtn = container.querySelector('button')
  expect(ctaBtn?.textContent).toContain('Install Tailscale')
})
```

**New tests to add — same shape, new states:**
```typescript
// daemonUp=false, binaryFound=true → daemon stopped state, no button
it('shows daemon-stopped message when binary found but daemon down', () => {
  ;({ container, root } = renderBanner({ ..., tailscaleBinaryFound: true, tailscaleDaemonUp: false }))
  expect(container.textContent).toContain('daemon not running')
  const buttons = container.querySelectorAll('button')
  expect(buttons.length).toBe(0)  // D-06: no action buttons
})

// Platform hint text tests per GOOS value
```

---

## Shared Patterns

### Injectable statusFunc (testability)
**Source:** `internal/webserver/tailscale.go` lines 21-22
**Apply to:** All new/extended health check logic in `tailscale.go`
```go
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)
```
Inject through all internal functions. The public `CheckHealth()` remains the only entry point that wires a real `local.Client`.

### Build tag for platform files (if split into separate files)
**Source:** `internal/daemon/path_windows.go` line 1, `internal/daemon/path_other.go` line 1
**Apply to:** `internal/webserver/tailscale_paths.go` if split into platform files
```go
//go:build windows
// or
//go:build !windows
```
Claude's Discretion: a single file with `runtime.GOOS` switch is simpler and avoids two files for this case.

### os.Stat existence check (file-on-disk detection)
**Source:** `internal/daemon/path.go` lines 41-44
**Apply to:** `tailscale_paths.go` `detectTailscaleBinary()`
```go
if _, err := os.Stat(dir); err == nil {
    extra = append(extra, dir)
}
```

### UserHomeDir for ~ expansion
**Source:** `internal/daemon/path.go` lines 17-20
**Apply to:** `tailscale_paths.go` for Linux flatpak user path
```go
home, err := os.UserHomeDir()
if err != nil {
    return
}
```

### React source-inspection test style
**Source:** `frontend/src/components/__tests__/SettingsTab.test.tsx` lines 1-5
**Apply to:** Any new SettingsTab or LocalNetworkBanner test files
```typescript
import raw from '../../components/SettingsTab.tsx?raw'
// then: expect(raw).toContain('...')
```
Use DOM rendering (createRoot + flushSync) for behavioral tests, `?raw` imports for structural/contract tests.

### Wails binding nil-guard pattern
**Source:** `app.go` lines 307-311
**Apply to:** Any `app.go` extension that reads custom Tailscale path
```go
func (a *App) UpdateCLIPath(name, path string) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.UpdateCLIPath(name, path)
}
```

### Health poller struct comparison
**Source:** `app.go` lines 764-776
**Apply to:** No change needed — `startHealthPoller` uses `h != last` struct equality, which automatically covers new fields added to `TailscaleHealth`
```go
if h != last {
    last = h
    runtime.EventsEmit(a.ctx, "tailscale:health", h)
}
```

---

## No Analog Found

All files have close analogs. No files require falling back to RESEARCH.md patterns exclusively.

---

## Metadata

**Analog search scope:** `internal/webserver/`, `internal/daemon/`, `internal/pty/`, `frontend/src/components/`, `app.go`
**Files scanned:** 12
**Pattern extraction date:** 2026-04-16
