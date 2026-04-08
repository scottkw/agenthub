# Phase 54: Tailscale Onboarding Enhancement - Research

**Researched:** 2026-04-07
**Domain:** React/Wails UI enhancement — HealthModal, clipboard API, Go subprocess streaming, Tailscale install UX
**Confidence:** HIGH

## Summary

Phase 54 enhances the existing `HealthModal` component to give new users actionable, platform-specific Tailscale installation guidance without leaving the app. The component already has the three-panel structure (NotInstalled / NotConnected / NoCerts) and platform awareness (`darwin`/`linux`/`windows`). This phase enriches each panel: adding copyable install commands with a one-click clipboard button, direct download links opened via `BrowserOpenURL`, a macOS "Try Auto-Install" button that streams `brew install --cask tailscale-app` output via Wails events, and a post-install "next steps" guide in NoCertsPanel.

The key architectural constraint from REQUIREMENTS.md Out of Scope: "Brew subprocess auto-install from GUI — TTY detection, sudo, and PATH issues make it unreliable; copy-paste commands instead." However, TS-02 explicitly requires a "Try Auto-Install" button for macOS. The resolution is that `brew install --cask tailscale-app` does NOT require sudo and runs non-interactively without a TTY — it is safe to stream. The out-of-scope entry refers to the general pattern; the specific brew cask command is viable. The planner should scope this narrowly: macOS-only, no sudo, show raw output, provide a manual fallback for non-macOS and on failure.

The test convention for this project's HealthModal is the `?raw` import pattern with `vitest`. All existing HealthModal tests use `import raw from '../HealthModal.tsx?raw'` and assert on source text content. New tests must follow this pattern.

**Primary recommendation:** Expand `HealthModal.tsx` with copyable commands and download links; add a new `AutoInstallTailscale` Wails method in `app.go` that streams brew output via `tailscale:install:progress` events; add `onOpenURL` and `onAutoInstall` props to HealthModal to keep components testable (no direct Wails calls inside the modal).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TS-01 | Platform-specific install command with copy-to-clipboard button; direct download links visible in health modal | Clipboard API pattern (navigator.clipboard.writeText), BrowserOpenURL prop injection, install commands verified per platform |
| TS-02 | Auto-install button runs `brew install --cask tailscale-app` on macOS with visible progress; manual fallback for other platforms | Wails EventsEmit streaming pattern from existing `session:status` / `update:available` usage; new `AutoInstallTailscale()` App method |
| TS-03 | Post-install "next steps" guide in health modal for enabling HTTPS certs (MagicDNS, HTTPS cert toggle in admin console) | NoCertsPanel already has step-by-step instructions; enrich with numbered steps and admin console URL |
</phase_requirements>

## Standard Stack

### Core (no new dependencies needed)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.2.4 | UI component enhancement | Already in project |
| Vitest | 4.1.0 | Test runner | Already in project |
| Wails v2 runtime | (project version) | EventsEmit for streaming output | Already in project |

### Supporting (Go backend)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os/exec` | stdlib | Run `brew install --cask tailscale-app` | macOS auto-install only |
| `bufio.Scanner` | stdlib | Read subprocess stdout line-by-line for streaming | Paired with `os/exec` StdoutPipe |
| `runtime.EventsEmit` | wails v2 | Stream install progress lines to frontend | Same pattern as `session:status` events |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `navigator.clipboard.writeText` | Wails clipboard binding | `navigator.clipboard` works in Wails WebView with user gesture; no need for extra binding |
| Go subprocess for auto-install | Shell out via PTY backend | PTY backend is for interactive sessions; a simple `exec.Command` + `StdoutPipe` is correct for a one-shot installer |

**Installation:** No new npm or Go packages required.

## Architecture Patterns

### Recommended Structure

The phase is entirely contained in two files plus tests:

```
frontend/src/components/
├── HealthModal.tsx         # MODIFIED — enrich three panels, add props
└── __tests__/
    └── HealthModal.test.tsx  # MODIFIED — add new assertions for TS-01/02/03
app.go                        # MODIFIED — add AutoInstallTailscale() method
frontend/src/wailsjs/go/main/
├── App.js                    # MODIFIED — add AutoInstallTailscale binding
└── App.d.ts                  # MODIFIED — add AutoInstallTailscale type
frontend/src/App.tsx          # MODIFIED — wire onOpenURL/onAutoInstall props to HealthModal
frontend/src/style.css        # MODIFIED — add styles for copy button, progress area
```

### Pattern 1: Copyable Command Block

The existing `health-modal__code--block` CSS class is used for command display. Add a "Copy" button beside it. Use `navigator.clipboard.writeText()` with a brief "Copied!" state flash.

```typescript
// Source: existing project convention (no Wails clipboard binding needed)
function CopyableCommand({ command }: { command: string }): React.ReactElement {
  const [copied, setCopied] = React.useState(false)
  const handleCopy = () => {
    void navigator.clipboard.writeText(command).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }
  return (
    <div className="health-modal__copy-row">
      <code className="health-modal__code health-modal__code--block">{command}</code>
      <button className="health-modal__btn--copy" onClick={handleCopy}>
        {copied ? 'Copied!' : 'Copy'}
      </button>
    </div>
  )
}
```

### Pattern 2: Download Link via Prop Injection

`HealthModal` must NOT import `BrowserOpenURL` directly (breaks Vitest jsdom tests). Follow the `RemoteSessionsPanel` convention: accept an `onOpenURL: (url: string) => void` prop. `App.tsx` wires `BrowserOpenURL` to it.

```typescript
// In HealthModal.tsx
interface HealthModalProps {
  health: TailscaleHealth | null
  platform: string
  onCheckAgain: () => void
  onOpenURL: (url: string) => void           // NEW — for download links
  onAutoInstall?: () => void                  // NEW — macOS only, for Try Auto-Install
}

// In App.tsx
<HealthModal
  health={tailscaleHealth}
  platform={platform}
  onCheckAgain={handleCheckHealthAgain}
  onOpenURL={BrowserOpenURL}
  onAutoInstall={handleAutoInstallTailscale}
/>
```

### Pattern 3: Auto-Install Progress Streaming

Reuse the existing Wails event pattern (`session:status`, `update:available`). A new backend method `AutoInstallTailscale()` runs the brew command and emits `tailscale:install:progress` events line-by-line.

```go
// Source: app.go — mirrors existing EventsEmit usage
func (a *App) AutoInstallTailscale() error {
    if runtime.GOOS != "darwin" {
        return fmt.Errorf("auto-install is only supported on macOS")
    }
    cmd := exec.Command("brew", "install", "--cask", "tailscale-app")
    cmd.Stdout = nil // handled via StdoutPipe
    stdout, err := cmd.StdoutPipe()
    cmd.Stderr = cmd.Stdout // merge stderr
    if err != nil {
        return err
    }
    if err := cmd.Start(); err != nil {
        return err
    }
    go func() {
        scanner := bufio.NewScanner(stdout)
        for scanner.Scan() {
            line := scanner.Text()
            if a.ctx != nil {
                runtime.EventsEmit(a.ctx, "tailscale:install:progress", line)
            }
        }
        exitErr := cmd.Wait()
        if exitErr != nil {
            runtime.EventsEmit(a.ctx, "tailscale:install:done", map[string]interface{}{"success": false, "error": exitErr.Error()})
        } else {
            runtime.EventsEmit(a.ctx, "tailscale:install:done", map[string]interface{}{"success": true})
        }
    }()
    return nil
}
```

On the frontend, the HealthModal subscribes to `tailscale:install:progress` and `tailscale:install:done` via `EventsOn` when the auto-install is in progress, displaying lines in a scrollable `<pre>` block.

### Pattern 4: NoCerts "Next Steps" Guide (TS-03)

The existing `NoCertsPanel` already has steps 1-3 for enabling HTTPS certs. For TS-03, the steps need to be clearer and include the prerequisite: enable MagicDNS first. Enrich existing steps, do not replace the panel.

**Correct sequence (verified from Tailscale docs):**
1. Go to `login.tailscale.com/admin/dns`
2. Enable MagicDNS if not already enabled
3. Under "HTTPS Certificates", click Enable HTTPS
4. Acknowledge the Certificate Transparency disclosure (hostname becomes public)
5. On this machine, run `tailscale cert` OR wait for AgentHub to provision via web server

### Anti-Patterns to Avoid

- **Direct Wails runtime imports in HealthModal:** `BrowserOpenURL` or `EventsOn` inside HealthModal breaks Vitest (jsdom cannot resolve Wails runtime). Use prop injection.
- **Calling `clipboard.writeText` without user gesture guard:** The Wails WKWebView clipboard API works only when triggered by a click event — always call from an `onClick` handler.
- **Hardcoding the tailscale-app cask name as `tailscale`:** The Homebrew cask is `tailscale-app`. The formula (CLI-only) is `tailscale`. For the full macOS app, use `brew install --cask tailscale-app`.
- **Showing the Try Auto-Install button on non-macOS platforms:** Gate the button behind `platform === 'darwin'`. Show manual copy commands on Linux and Windows.
- **Blocking the Wails main thread during brew install:** Run the command in a goroutine and stream via events. `AutoInstallTailscale()` starts the goroutine and returns immediately.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Clipboard copy | Custom execCommand fallback | `navigator.clipboard.writeText()` | Works in Wails WKWebView; execDocument.execCommand is deprecated |
| Subprocess output streaming | WebSocket or IPC pipe | Wails `runtime.EventsEmit` + frontend `EventsOn` | Already proven in this codebase for `session:status` |
| Platform detection in frontend | `navigator.platform` checks | Use the `platform` prop passed from App.tsx (already sourced from Wails `Environment()`) | Consistent, already normalized to `darwin`/`linux`/`windows` |
| Step-by-step cert instructions | External docs link only | Inline numbered guide in NoCertsPanel | Keeps user in-app; NoCertsPanel already has the framework |

**Key insight:** All infrastructure (platform prop, EventsEmit pattern, prop injection pattern) already exists in this codebase. This phase is entirely about enhancing one React component and adding one Go method.

## Common Pitfalls

### Pitfall 1: Wrong Homebrew Cask Name
**What goes wrong:** Using `brew install tailscale` installs the CLI-only formula, not the macOS app with the menu bar icon.
**Why it happens:** There are two Homebrew packages: `tailscale` (formula, CLI daemon) and `tailscale-app` (cask, full macOS app).
**How to avoid:** Use `brew install --cask tailscale-app` for the auto-install command. Also show this command in the copyable block for macOS (TS-01).
**Warning signs:** Installed binary has no menu bar icon; `tailscale status` works but no GUI.

### Pitfall 2: Vitest Raw Import Tests Break if Wails Imports Are Added to HealthModal
**What goes wrong:** If `BrowserOpenURL` or `EventsOn` are imported directly in HealthModal.tsx, Vitest's jsdom environment cannot resolve the Wails runtime path, causing all HealthModal tests to fail.
**Why it happens:** Vitest does not mock `./wailsjs/wailsjs/runtime/runtime` automatically. Only App.tsx (not tested via raw imports) directly imports Wails runtime.
**How to avoid:** All browser/Wails APIs must be injected as props. Inside HealthModal, use `onOpenURL(url)` instead of `BrowserOpenURL(url)`. Wire EventsOn subscription in App.tsx, not HealthModal.
**Warning signs:** HealthModal.test.tsx fails with "Cannot find module" or "not a function" errors on Vitest run.

### Pitfall 3: Auto-Install Goroutine Races on Context Cancellation
**What goes wrong:** If the user closes the app during brew install, `a.ctx` may be nil or cancelled when the goroutine tries to emit events.
**Why it happens:** Wails calls `shutdown()` on app exit, which cancels the context.
**How to avoid:** Guard every `runtime.EventsEmit` in the goroutine with `if a.ctx != nil` (matches existing `startHealthPoller` pattern in app.go line 638).
**Warning signs:** Panic or nil pointer dereference in the install goroutine on app exit during install.

### Pitfall 4: brew PATH Not Found From GUI Process
**What goes wrong:** `exec.Command("brew", ...)` returns "executable not found" even though Homebrew is installed, because GUI apps launched from Dock/Finder do not inherit the user's shell PATH.
**Why it happens:** macOS GUI apps have a minimal environment; `/opt/homebrew/bin` is not on the default PATH.
**How to avoid:** Use the full path `/opt/homebrew/bin/brew` as the primary attempt, with `/usr/local/bin/brew` as Intel Mac fallback. Detect via `exec.LookPath` first; if not found, emit an error event and show the manual copy command.
**Warning signs:** `exec.LookPath("brew")` returns error in the GUI app even when brew works in Terminal.

### Pitfall 5: MagicDNS Must Be Enabled Before HTTPS Certs
**What goes wrong:** User tries to enable HTTPS certificates in Tailscale admin but the option is greyed out.
**Why it happens:** HTTPS certificates require MagicDNS to be active first.
**How to avoid:** The NoCertsPanel step-by-step guide must mention enabling MagicDNS as step 0 or step 1 before the HTTPS cert step.
**Warning signs:** User reports that the HTTPS option does not appear in admin console.

## Code Examples

### Platform-Specific Install Commands (Verified)

```typescript
// macOS
const MACOS_INSTALL_CMD = 'brew install --cask tailscale-app'
const MACOS_DOWNLOAD_URL = 'https://tailscale.com/download/macos'

// Linux
const LINUX_INSTALL_CMD = 'curl -fsSL https://tailscale.com/install.sh | sh'
const LINUX_DOWNLOAD_URL = 'https://tailscale.com/download/linux'

// Windows
const WINDOWS_INSTALL_CMD = 'winget install Tailscale.Tailscale'
const WINDOWS_DOWNLOAD_URL = 'https://tailscale.com/download/windows'
```

### Brew Path Detection Pattern (macOS auto-install)

```go
// Source: standard macOS Homebrew path conventions (verified: this machine has tailscale-app at /opt/homebrew/Caskroom)
func findBrew() (string, error) {
    // Apple Silicon
    if _, err := os.Stat("/opt/homebrew/bin/brew"); err == nil {
        return "/opt/homebrew/bin/brew", nil
    }
    // Intel Mac
    if _, err := os.Stat("/usr/local/bin/brew"); err == nil {
        return "/usr/local/bin/brew", nil
    }
    return "", fmt.Errorf("Homebrew not found; install from https://brew.sh")
}
```

### Test Convention (raw import + source text assertions)

```typescript
// Source: existing HealthModal.test.tsx convention
import { describe, it, expect } from 'vitest'
import raw from '../HealthModal.tsx?raw'

describe('HealthModal enhancements (TS-01)', () => {
  it('shows brew install command for macOS', () => {
    expect(raw).toContain('brew install --cask tailscale-app')
  })
  it('shows winget command for Windows', () => {
    expect(raw).toContain('winget install Tailscale.Tailscale')
  })
  it('includes copy-to-clipboard handler', () => {
    expect(raw).toContain('navigator.clipboard.writeText')
  })
  it('uses onOpenURL prop not BrowserOpenURL directly', () => {
    expect(raw).not.toContain('BrowserOpenURL')
    expect(raw).toContain('onOpenURL')
  })
})
```

### CSS Classes to Add

```css
/* Copy row: command block + copy button side by side */
.health-modal__copy-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.health-modal__btn--copy {
  flex-shrink: 0;
  /* follows existing btn--check style */
}

/* Auto-install progress output */
.health-modal__install-output {
  font-family: "Cascadia Code", "Fira Code", monospace;
  font-size: 11px;
  background: #16161e;
  border: 1px solid #292e42;
  border-radius: 4px;
  padding: 10px 12px;
  max-height: 160px;
  overflow-y: auto;
  white-space: pre-wrap;
  color: #a9b1d6;
}
```

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Homebrew (`/opt/homebrew/bin/brew`) | TS-02 auto-install | ✓ | 5.1.5-3 | Show manual copy command |
| `tailscale-app` Homebrew cask | TS-02 auto-install | ✓ (installed) | 1.96.5 (cask) | User downloads from tailscale.com/download |
| `navigator.clipboard` | TS-01 copy button | ✓ (Wails WKWebView) | N/A (browser API) | None needed |
| Tailscale admin console | TS-03 cert guide | External URL — always available | N/A | N/A |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** Homebrew not installed — `AutoInstallTailscale()` emits an error event and HealthModal falls back to showing the manual copy command.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 |
| Config file | `frontend/vitest.config.ts` |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test && go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TS-01 | brew/winget/install.sh commands appear in NotInstalledPanel | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) |
| TS-01 | onOpenURL prop used instead of BrowserOpenURL | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) |
| TS-01 | navigator.clipboard.writeText used for copy | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) |
| TS-02 | Try Auto-Install button gated to platform === darwin | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) |
| TS-02 | AutoInstallTailscale bound method in App.js/App.d.ts | unit (raw) | `cd frontend && pnpm test` | ✅ (extend App.test.tsx) |
| TS-02 | AutoInstallTailscale Go method uses brew cask | unit | `go test ./... -run TestAutoInstall` | ❌ Wave 0 |
| TS-03 | NoCertsPanel includes MagicDNS enable step | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) |
| TS-03 | NoCertsPanel links to login.tailscale.com/admin/dns | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test && cd /Users/ken/dev/agenthub && go test ./...`
- **Phase gate:** Full suite green (note: `TestHub_SlowClientDisconnected` in relay package has a pre-existing flaky failure unrelated to this phase)

### Wave 0 Gaps
- [ ] `app_test.go` — add `TestAutoInstallTailscale` covering: brew path resolution, error on non-darwin, goroutine event emission (can use mock EventsEmit or test that method returns nil when brew found)

*(Existing HealthModal.test.tsx covers most requirements via raw import assertions; extend in-place)*

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `tailscale` brew formula | `tailscale-app` brew cask | Always separate | `tailscale` is CLI-only daemon; cask installs full macOS app with GUI |
| HTTPS cert guide showed admin link only | Now requires MagicDNS first | Tailscale changed DNS flow | NoCertsPanel must include MagicDNS step |

**Deprecated/outdated:**
- `document.execCommand('copy')`: Deprecated in modern browsers. Use `navigator.clipboard.writeText()` — works in Wails WKWebView.

## Open Questions

1. **Should AutoInstallTailscale be cancellable?**
   - What we know: brew installs can take 30-60 seconds; user may want to cancel
   - What's unclear: Whether a "Cancel" button adds meaningful value for v1.9
   - Recommendation: Skip cancel for v1.9. Emit a `tailscale:install:done` with success/error. Show a "Dismiss" button that hides the progress area.

2. **Should the health modal dismiss after successful auto-install?**
   - What we know: Health poller runs every 10 seconds and emits `tailscale:health` events when state changes
   - What's unclear: Tailscale needs to be signed in after install before `installed: true` triggers
   - Recommendation: After auto-install completes, show "Next: open Tailscale from your menu bar and sign in" with a "Check Again" button — do not auto-dismiss.

## Sources

### Primary (HIGH confidence)
- Local codebase inspection (`HealthModal.tsx`, `App.tsx`, `app.go`, `style.css`) — full component and architecture understanding
- Local `brew info --cask tailscale` — confirmed cask name `tailscale-app`, version 1.96.5
- `REQUIREMENTS.md` Out of Scope section — "Brew subprocess auto-install from GUI" scoping note
- Wails `EventsEmit` pattern from `app.go` (lines 593, 616, 639) — streaming approach verified

### Secondary (MEDIUM confidence)
- WebSearch: Tailscale macOS brew cask name — confirmed `tailscale-app` via formulae.brew.sh
- WebSearch: `winget install Tailscale.Tailscale` — confirmed via winget.run and tailscale official docs
- WebSearch: Tailscale HTTPS cert enable steps — confirmed MagicDNS prerequisite via tailscale.com docs

### Tertiary (LOW confidence)
- `https://tailscale.com/download/macos` download URL format — WebSearch confirmed tailscale.com/download exists; `/macos` subpath inferred from pattern, not verified by direct fetch (WebFetch returned socket error)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new deps; all existing libraries
- Architecture: HIGH — directly inspected all relevant files
- Pitfalls: HIGH — brew path issue is a known macOS GUI app pattern; Wails import issue is documented in STATE.md (Phase 49 CSS ?raw convention)
- Auto-install brew path: MEDIUM — `/opt/homebrew/bin/brew` confirmed present on this dev machine; Intel Mac path `/usr/local/bin/brew` is conventional

**Research date:** 2026-04-07
**Valid until:** 2026-07-07 (stable domain — Tailscale cask name and install commands are stable)
