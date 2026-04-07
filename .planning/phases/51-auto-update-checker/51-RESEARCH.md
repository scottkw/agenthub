# Phase 51: Auto-Update Checker - Research

**Researched:** 2026-04-07
**Domain:** Go auto-update detection (go-selfupdate v1.5.2), Wails event bus, React banner UI
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| UPD-01 | App checks GitHub releases for newer versions on startup and periodically (hourly) | `go-selfupdate` DetectLatest + background goroutine poller in app.go |
| UPD-02 | User sees notification banner in Welcome tab when update is available with version info | Wails EventsEmit `update:available` → React state → banner in WelcomeTab.tsx |
| UPD-03 | One-click download: opens GitHub releases page in system browser | `runtime.BrowserOpenURL` — already used in `openGitHubCallback` |
| UPD-04 | "Check for Updates" item in Help menu triggers immediate version check | Add menu item to `appMenu()` Help submenu; calls `CheckForUpdates()` bound method or emits event |
</phase_requirements>

---

## Summary

Phase 51 implements a detect-and-notify update flow: the app polls GitHub releases for a newer version on startup and every hour, then shows a dismissible banner in the Welcome tab when an update is available. Clicking "Download" opens the GitHub releases page in the system browser. No in-place binary replacement is performed — this is explicitly out of scope per the project requirements (Gatekeeper/code-signing constraint).

The primary library is `github.com/creativeprojects/go-selfupdate@v1.5.2`, which is already chosen per STATE.md. It provides `DetectLatest` and version comparison methods (`GreaterThan`) without requiring any binary download. The update check rate-limit is enforced by persisting the last-check timestamp to `~/.config/agenthub/update_check.json` — the same pattern used by `ct_disclosed` (the CT disclosure sentinel file).

The implementation follows three established project patterns: (1) injectable function types for testability (tailnet package pattern), (2) background goroutine pollers with context cancellation (startTrayPoller / startHealthPoller pattern in app.go), and (3) Wails EventsEmit for push notification to the React frontend (session:status / tailscale:health pattern).

**Primary recommendation:** Add `internal/updater` package with injectable `detectFunc` for testability, wire a `startUpdatePoller` goroutine in `app.startup()`, emit `update:available` events to the frontend, and add a banner section to `WelcomeTab.tsx`.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/creativeprojects/go-selfupdate | v1.5.2 | Detect latest GitHub release, compare semver | Chosen in STATE.md; active maintenance (Dec 2025); clean detect-only API |
| github.com/wailsapp/wails/v2 runtime | v2.10.2 (existing) | EventsEmit push to frontend, BrowserOpenURL | Already in project; exact same pattern as tailscale:health events |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json (stdlib) | — | Persist last-check timestamp to disk | Storing `{"last_check": "2026-04-07T..."}`  |
| os/filepath (stdlib) | — | Construct config path via `configDir()` | Reuse existing helper |
| time (stdlib) | — | Rate-limit (1 hour), goroutine ticker | — |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-selfupdate | Direct GitHub API `GET /repos/{owner}/{repo}/releases/latest` | Simpler HTTP call with no dependency, but loses semver comparison and asset matching |
| go-selfupdate | rhysd/go-github-selfupdate | Predecessor, less actively maintained per STATE.md decision |
| Wails event push | Polling from frontend | Push is simpler; frontend already uses EventsOn for other events |

**Installation (one new dependency):**
```bash
cd /Users/ken/dev/agenthub && go get github.com/creativeprojects/go-selfupdate@v1.5.2
```

**Version verification:** `go list -m github.com/creativeprojects/go-selfupdate@latest` returns `v1.5.2` — confirmed current as of 2026-04-07.

---

## Architecture Patterns

### Recommended Project Structure
```
internal/updater/          # new package
  updater.go               # DetectUpdate(), injectable detectFunc, rate-limit logic
  updater_test.go          # tests with fake detectFunc
app.go                     # add startUpdatePoller(), CheckForUpdates() bound method
main.go                    # add "Check for Updates" to Help menu item
frontend/src/components/
  WelcomeTab.tsx           # add UpdateBanner section (conditional render)
frontend/src/style.css     # add .update-banner CSS block
frontend/src/wailsjs/go/main/App.d.ts  # add CheckForUpdates() export
frontend/src/components/__tests__/
  WelcomeTab.test.tsx      # extend with update banner tests
```

### Pattern 1: Injectable detectFunc (mirrors tailnet injectable patterns)

The `internal/updater` package uses an injectable function type so tests never make real HTTP calls.

```go
// Source: tailnet.go injectable pattern (statusFunc, discoverFunc)
type detectFunc func(ctx context.Context, slug string) (latestVersion string, found bool, err error)

// Package-level production implementation
func defaultDetect(ctx context.Context, slug string) (string, bool, error) {
    latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug(slug))
    if err != nil || !found {
        return "", found, err
    }
    return latest.Version(), true, nil
}
```

### Pattern 2: Rate-limited check with persisted timestamp

```go
// Source: project pattern — configDir() + os.WriteFile (app.go ctDisclosurePath)
type UpdateState struct {
    LastCheck time.Time `json:"last_check"`
}

func shouldCheck(configPath string) bool {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return true // no file = never checked
    }
    var state UpdateState
    if err := json.Unmarshal(data, &state); err != nil {
        return true
    }
    return time.Since(state.LastCheck) >= time.Hour
}

func persistCheck(configPath string) error {
    data, _ := json.Marshal(UpdateState{LastCheck: time.Now()})
    return os.WriteFile(configPath, data, 0600)
}
```

### Pattern 3: Background poller goroutine (mirrors startTrayPoller / startHealthPoller)

```go
// Source: app.go startHealthPoller pattern
func (a *App) startUpdatePoller(ctx context.Context) {
    go func() {
        // Check immediately on startup
        a.runUpdateCheck(ctx)
        ticker := time.NewTicker(time.Hour)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                a.runUpdateCheck(ctx)
            case <-ctx.Done():
                return
            }
        }
    }()
}

func (a *App) runUpdateCheck(ctx context.Context) {
    info, err := updater.Check(ctx, configDir(), "scottkw/agenthub", Version)
    if err != nil || info == nil {
        return // silent: no event emitted
    }
    if a.ctx != nil && a.ctx.Value("frontend") != nil {
        runtime.EventsEmit(a.ctx, "update:available", info)
    }
}
```

### Pattern 4: Help menu item (mirrors openGitHubCallback)

```go
// Source: main.go appMenu() pattern
helpMenu.AddText("Check for Updates", nil, checkForUpdatesCallback)

func checkForUpdatesCallback(_ *menu.CallbackData) {
    if appCtx != nil {
        // Use bound method via Wails event or call app method directly
        // Option: emit an event that triggers frontend to call CheckForUpdates()
        runtime.EventsEmit(appCtx, "update:check-requested", nil)
    }
}
```

The cleaner approach: expose a `CheckForUpdates() *UpdateInfo` bound method on App, and have the menu callback emit `update:check-requested`. The frontend subscribes to this event and calls `CheckForUpdates()` immediately on receipt.

Alternatively, the menu callback can call `app.runUpdateCheck(appCtx)` directly since `app` is accessible at package level (same pattern as `appCtx`). This is simpler and avoids a round-trip through the frontend.

**Recommendation:** Store a package-level `appInstance *App` alongside `appCtx`, set in `startup()`, so `checkForUpdatesCallback` can call `appInstance.runUpdateCheck(appCtx)` directly.

### Pattern 5: Frontend update banner in WelcomeTab

```tsx
// Source: existing WelcomeTab.tsx + App.tsx EventsOn pattern
interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseURL: string
}

// In WelcomeTab:
const [update, setUpdate] = useState<UpdateInfo | null>(null)

useEffect(() => {
  const offUpdate = EventsOn('update:available', (info: UpdateInfo) => {
    setUpdate(info)
  })
  return () => offUpdate()
}, [])

// Banner JSX (conditional):
{update && (
  <div className="update-banner">
    <span>Update available: {update.currentVersion} → {update.latestVersion}</span>
    <button onClick={() => BrowserOpenURL(update.releaseURL)}>Download</button>
    <button onClick={() => setUpdate(null)}>Dismiss</button>
  </div>
)}
```

**Important:** `BrowserOpenURL` is a Wails runtime function importable from `../wailsjs/wailsjs/runtime/runtime`. Already used indirectly via the openGitHubCallback; frontend uses `runtime.BrowserOpenURL` via Wails JS runtime bindings.

### Anti-Patterns to Avoid

- **Blocking startup with update check:** The update check must run in a goroutine. `startup()` must return before the check completes.
- **Panicking on 429 / non-200:** go-selfupdate returns `err` on HTTP failures. The poller must log and return silently — no user-visible errors.
- **Hardcoding "v" prefix in version comparison:** `Version = "dev"` in local builds. The updater must skip the check when `Version == "dev"` to avoid always showing the banner.
- **Race condition on `appInstance`:** Set `appInstance` in `startup()` before starting the poller (same timing as `appCtx`).
- **Frontend calling CheckForUpdates() on every render:** Rate-limit is in the Go layer; frontend triggers it only from the menu event or on mount.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Semver comparison | String comparison, regex parsing | `go-selfupdate Release.GreaterThan()` | Handles `v` prefix stripping, pre-release tags |
| GitHub API pagination | Manual HTTP client + JSON parsing | `go-selfupdate DetectLatest()` | Handles auth, rate-limit headers, redirect |
| OS/arch asset matching | Manual filename parsing | go-selfupdate default (not needed — detect-only) | For detect-only, version string is all we need |

**Key insight:** For detect-only usage, `DetectLatest` + `Release.GreaterThan(currentVersion)` is the entire logic needed. The library handles all GitHub API details.

---

## Common Pitfalls

### Pitfall 1: Version = "dev" always triggers update banner
**What goes wrong:** In local dev builds, `Version = "dev"`. `go-selfupdate` will always report any real release as newer than "dev".
**Why it happens:** "dev" is not a valid semver string; comparison falls back to "any release > dev".
**How to avoid:** In `runUpdateCheck`, skip if `Version == "dev"` or `Version == ""`.
**Warning signs:** Update banner shows immediately on every dev launch.

### Pitfall 2: ParseSlug("scottkw/agenthub") — repo may not have releases yet
**What goes wrong:** STATE.md explicitly notes: "Validate `go-selfupdate ParseSlug("scottkw/agenthub")` matches v1.8 artifact naming before finalizing Phase 51." The GitHub releases API returns 404 if no releases exist.
**Why it happens:** No published GitHub release exists yet for scottkw/agenthub (confirmed by `curl https://api.github.com/repos/scottkw/agenthub/releases/latest` → 404).
**How to avoid:** The plan must handle the 404/`found=false` case gracefully — this is normal until the first release is published. Also, binary assets must follow the go-selfupdate naming pattern (`agenthub_{goos}_{goarch}`) for asset matching, but for detect-only, this is irrelevant — we only need the tag name.
**Warning signs:** `DetectLatest` returns `found=false` in all environments.

### Pitfall 3: Wails runtime event before DOM is ready
**What goes wrong:** `startup()` runs before `domReady()` — if the update check runs instantly and completes before React subscribes to `update:available`, the event is lost.
**Why it happens:** Wails startup lifecycle: `OnStartup` → render → `OnDomReady`. Events emitted during startup may fire before the EventsOn subscription is set up.
**How to avoid:** Two options: (a) add a small delay before the first check (e.g., 5 seconds after startup), or (b) expose a `GetUpdateInfo() *UpdateInfo` bound method the frontend polls on mount. Option (b) is more reliable and matches the `GetDaemonError()` pattern already in the codebase.
**Warning signs:** Banner never shows even when update is available.

### Pitfall 4: Rate-limiting the hourly check
**What goes wrong:** If the user opens/closes the app multiple times in an hour, each startup triggers a check.
**Why it happens:** The 1-hour rate-limit must be persisted to disk, not just in-memory.
**How to avoid:** Write `update_check.json` with timestamp after each successful check. Read it at startup to skip if within 1 hour.
**Warning signs:** GitHub API returns 429 (60 unauthenticated req/hour limit).

### Pitfall 5: appMenu() ordering — adding to existing Help submenu
**What goes wrong:** `helpMenu` is a local variable in `appMenu()`. Need to add the new item before the function returns.
**Why it happens:** The Help menu is already constructed; the new item must be inserted before `return m`.
**How to avoid:** Add `helpMenu.AddText("Check for Updates", ...)` after the existing "AgentHub on GitHub" line.
**Warning signs:** Menu item absent in the built app.

---

## Code Examples

### Detect-only usage (go-selfupdate v1.5.2)
```go
// Source: pkg.go.dev/github.com/creativeprojects/go-selfupdate@v1.5.2
import selfupdate "github.com/creativeprojects/go-selfupdate"

func detectLatestVersion(ctx context.Context, slug string) (string, bool, error) {
    latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug(slug))
    if err != nil || !found {
        return "", found, err
    }
    return latest.Version(), true, nil
}

// Version comparison — handles "v" prefix automatically
if latest.GreaterThan(currentVersion) {
    // update available
}
```

### BrowserOpenURL in frontend (Wails JS runtime)
```tsx
// Source: wailsjs/wailsjs/runtime/runtime (existing import in App.tsx)
import { EventsOn } from '../wailsjs/wailsjs/runtime/runtime'
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'

// Open releases page
BrowserOpenURL('https://github.com/scottkw/agenthub/releases')
```

### Wails bound method for on-demand check
```go
// CheckForUpdates performs an immediate update check, bypassing rate limit,
// and emits "update:available" if a newer version exists.
// Called by the Help > Check for Updates menu item.
func (a *App) CheckForUpdates() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    a.runUpdateCheck(ctx)
}
```

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| go-selfupdate | UPD-01/02 | ✓ (via go get) | v1.5.2 | — |
| Wails v2 | Runtime binding | ✓ | v2.10.2 | — |
| GitHub API (unauthenticated) | Release detection | ✓ | — | Silent fail — no banner shown |
| scottkw/agenthub GitHub releases | UPD-01 | No releases yet | — | found=false → no banner (correct behavior until first release) |

**Missing dependencies with no fallback:** None that block execution.

**Missing dependencies with fallback:**
- scottkw/agenthub has no GitHub releases yet — `DetectLatest` will return `found=false`, which is correct behavior. The banner simply never shows. This resolves itself when v1.8+ is published.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `testing` stdlib |
| Framework (Frontend) | vitest v4.1.0 |
| Config file | `frontend/vite.config.ts` |
| Quick run (Go) | `go test ./internal/updater/... -v` |
| Quick run (Frontend) | `pnpm --dir frontend test` |
| Full suite | `go test ./... && pnpm --dir frontend test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UPD-01 | Checker skips when version="dev" | unit | `go test ./internal/updater/ -run TestCheck_DevVersionSkip` | ❌ Wave 0 |
| UPD-01 | Checker returns UpdateInfo when newer version found | unit | `go test ./internal/updater/ -run TestCheck_NewerVersionFound` | ❌ Wave 0 |
| UPD-01 | Checker returns nil when already on latest | unit | `go test ./internal/updater/ -run TestCheck_AlreadyLatest` | ❌ Wave 0 |
| UPD-01 | Checker returns nil on detectFunc error (silent) | unit | `go test ./internal/updater/ -run TestCheck_DetectError` | ❌ Wave 0 |
| UPD-01 | Rate limit: skips check if within 1 hour (persisted) | unit | `go test ./internal/updater/ -run TestCheck_RateLimit` | ❌ Wave 0 |
| UPD-01 | Poller exits when context cancelled | unit | `go test ./... -run TestUpdatePollerStops` | ❌ Wave 0 |
| UPD-02 | WelcomeTab renders update banner when update prop provided | unit | `pnpm --dir frontend test` (WelcomeTab.test.tsx) | ❌ Wave 0 |
| UPD-02 | WelcomeTab shows current and latest version strings | unit | `pnpm --dir frontend test` | ❌ Wave 0 |
| UPD-03 | Banner Download button present | unit | `pnpm --dir frontend test` | ❌ Wave 0 |
| UPD-04 | CheckForUpdates bound method exists and calls detect | unit | `go test ./... -run TestCheckForUpdates` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/updater/... && pnpm --dir frontend test`
- **Per wave merge:** `go test ./... && pnpm --dir frontend test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/updater/updater.go` — package with injectable detectFunc
- [ ] `internal/updater/updater_test.go` — unit tests for all UPD-01 behaviors
- [ ] `frontend/src/components/__tests__/WelcomeTab.test.tsx` — extend with banner tests (file exists, needs new `describe` block)

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| rhysd/go-github-selfupdate | creativeprojects/go-selfupdate | STATE.md decision | Active maintenance, context-aware API |
| In-place binary replacement | Detect + open browser | Requirements decision | Avoids Gatekeeper/code-signing issues on macOS |

**Deprecated/outdated:**
- `rhysd/go-github-selfupdate`: predecessor, replaced by creativeprojects fork. STATE.md explicitly names the replacement.

---

## Open Questions

1. **ParseSlug("scottkw/agenthub") — asset naming validation**
   - What we know: No GitHub releases exist yet for scottkw/agenthub. The detect-only flow only needs the tag name, not asset filenames.
   - What's unclear: The STATE.md blocker says "Validate ParseSlug matches v1.8 artifact naming before finalizing Phase 51." For detect-only, this is irrelevant — no asset matching occurs. The concern was originally about UpdateTo(), which we don't use.
   - Recommendation: Mark this blocker as resolved for Phase 51's detect-only scope. Asset naming only matters if/when Phase UPD-F01 (silent download) is implemented.

2. **`update:check-requested` menu callback approach**
   - What we know: The `openGitHubCallback` pattern uses a package-level `appCtx`. For `CheckForUpdates`, we need access to `*App`.
   - What's unclear: Whether to add a package-level `appInstance *App` or use `runtime.EventsEmit(appCtx, "update:check-requested")` to the frontend which then calls a bound method.
   - Recommendation: Add `var appInstance *App` alongside `var appCtx`, set in `startup()`. The menu callback calls `appInstance.runUpdateCheck(appCtx)` in a goroutine. This is the simplest approach and consistent with the existing `appCtx` pattern.

3. **GetUpdateInfo() bound method for startup race avoidance**
   - What we know: Events emitted in `startup()` before the frontend's EventsOn is registered are lost. The `GetDaemonError()` method exists precisely for this pattern.
   - What's unclear: Whether to use a bound method poll-on-mount or delay the first check by a few seconds.
   - Recommendation: Store the latest `*UpdateInfo` in `App` struct (e.g., `lastUpdate *UpdateInfo` with a mutex). Expose `GetLastUpdateInfo() *UpdateInfo` as a bound method. Frontend calls this on mount (alongside other init calls) and separately subscribes to `update:available` for future events. This is the `GetDaemonError` pattern.

---

## Sources

### Primary (HIGH confidence)
- `go list -m github.com/creativeprojects/go-selfupdate@latest` — version confirmed v1.5.2
- `pkg.go.dev/github.com/creativeprojects/go-selfupdate@v1.5.2` — DetectLatest, ParseSlug, Release.GreaterThan APIs
- Project source: `app.go`, `main.go`, `internal/tailnet/tailnet.go`, `internal/daemon/tailnet_cache.go` — existing injectable, poller, and event patterns

### Secondary (MEDIUM confidence)
- `github.com/creativeprojects/go-selfupdate` README (WebFetch) — detect-only usage examples, version prefix behavior
- GitHub Docs rate limits page — 60 req/hour unauthenticated, 429 handling

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — go-selfupdate v1.5.2 confirmed via `go list`, existing project uses all other dependencies
- Architecture: HIGH — all patterns directly derived from existing project code (tailnet, app.go pollers, Wails events)
- Pitfalls: HIGH — "dev" version pitfall confirmed by code inspection; startup race confirmed by GetDaemonError pattern; repo has no releases confirmed by live API test

**Research date:** 2026-04-07
**Valid until:** 2026-07-07 (go-selfupdate is a stable library; Wails v2 API is frozen)
