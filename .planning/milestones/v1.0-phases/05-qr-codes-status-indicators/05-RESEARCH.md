# Phase 5: QR Codes + Status Indicators - Research

**Researched:** 2026-03-18
**Domain:** QR code generation (Go + React), PTY output heuristic parsing, Wails event bus
**Confidence:** HIGH (architecture) / MEDIUM (per-CLI status patterns)

---

## Summary

Phase 5 adds two independent features: QR code display for web-served sessions and live status badges on tabs. Both features share a common data path — the Hub that already receives all PTY output frames is the natural tap point for both the session URL (to generate a QR) and the status heuristics (to classify output).

**QR codes** are simple: the session URL already exists (`GetWebServerURL` + session ID), and the `skip2/go-qrcode` library generates an in-memory PNG from any string in one call. The PNG is base64-encoded, returned to the Wails frontend via a bound method, and rendered with `<QRCodeSVG>` from `qrcode.react`, or — simpler still — as a plain `<img src="data:image/png;base64,…">` element without adding an npm dependency.

**Status detection** is more nuanced. Claude Code emits a `❯` prompt when idle and the text "ctrl+c to interrupt" when actively processing. `[y/n]` or `[Y/n]` in the output indicates it is waiting for user decision. Process exit (Hub.Done closed) means errored if the exit code was non-zero or stopped if zero. These patterns come from empirical community research (claude-tmux, agent-of-empires) and are the best available heuristics since Claude Code does not expose structured state via OSC sequences or API. Other CLIs (Gemini CLI, Codex, OpenCode) require similar empirical observation — this is noted as an open question.

**Delivery architecture:** A new `internal/status` package runs a per-Hub goroutine that consumes a tee of PTY output bytes, maintains a rolling window, and emits status transitions. The Wails App struct subscribes to these transitions and calls `runtime.EventsEmit` so the React frontend receives live updates without polling.

**Primary recommendation:** Use `skip2/go-qrcode` for Go-side PNG generation (zero dependencies beyond stdlib), return base64 PNG via a bound Wails method, render as `<img>` in React. For status, tap the Hub scrollback tee with a regex-based state machine in a new `internal/status` package; push transitions via `runtime.EventsEmit`.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| QR-01 | App generates QR codes for all web-served session URLs | skip2/go-qrcode.Encode() returns []byte PNG in one call; base64-encode and expose via Wails bound method |
| QR-02 | QR codes displayed in desktop app and on web dashboard | Desktop: img element with data URI in React; Dashboard: inline SVG or img via /api/sessions/{id}/qr endpoint |
| STAT-01 | Each tab shows session status: running, waiting for input, idle, or errored | Wails EventsEmit pushes status transitions; React stores per-session status in state |
| STAT-02 | Status detection uses heuristic parsing of CLI output patterns | PTY output tee in Hub-connected goroutine; regex state machine in internal/status package |
</phase_requirements>

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/skip2/go-qrcode` | v0.0.0-20200617195104 | Generate QR PNG in-memory | Zero deps beyond stdlib; `Encode()` returns `[]byte` directly; widely used |
| `github.com/wailsapp/wails/v2/pkg/runtime` | (already in go.mod) | `EventsEmit` to push status to frontend | Already available; correct pattern for backend→frontend push |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/base64` | stdlib | Encode PNG bytes for JSON/data URI | Always — no external dep needed |
| `regexp` | stdlib | Compile status-detection patterns | Compile once at init, reuse per-output-chunk |
| `qrcode.react` | v3.x (npm) | Render QR in React if preferred over `<img>` | OPTIONAL — plain `<img data:image/png;base64,…>` works without it |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| skip2/go-qrcode | yeqown/go-qrcode v2 | yeqown adds richer styling (colors, logo) but pulls extra dependencies; not needed here |
| img data URI | qrcode.react npm | qrcode.react generates the QR in the browser from the URL string, avoiding a backend round-trip; viable alternative but creates npm dependency and requires passing the URL to React (which it already has) |
| EventsEmit push | polling GetSessionStatus() | Polling adds latency and wastes cycles; EventsEmit is the idiomatic Wails pattern |

**Installation (Go):**
```bash
go get github.com/skip2/go-qrcode
```

**Installation (npm — only if qrcode.react chosen):**
```bash
cd frontend && pnpm add qrcode.react
```

---

## Architecture Patterns

### Recommended Project Structure

New files this phase:

```
internal/
└── status/
    ├── detector.go        # StatusDetector type, per-session goroutine, regex patterns
    └── detector_test.go   # Table-driven tests for each pattern

app.go                     # Add: GetSessionQRCode(), GetSessionStatus(), startStatusDetector()
                           # status map: sessionID -> SessionStatus

frontend/src/
├── App.tsx                # Subscribe EventsOn("session:status", ...), pass status to TabBar
└── components/
    ├── TabBar.tsx          # Accept status prop, render badge per tab
    └── QRModal.tsx         # New: modal overlay showing QR code img + URL

web/
└── dashboard.html         # Add: QR img rendered inline per session row
```

### Pattern 1: QR Code Generation (Go)

**What:** Bound Wails method returns base64-encoded PNG for the session URL.
**When to use:** Called once when web serving is toggled on for a session, or on demand.

```go
// Source: skip2/go-qrcode pkg.go.dev — Encode() signature
func (a *App) GetSessionQRCode(sessionID string) (string, error) {
    a.mu.RLock()
    ws := a.webServer
    a.mu.RUnlock()
    if ws == nil {
        return "", fmt.Errorf("web server not running")
    }
    url := fmt.Sprintf("%s/sessions/%s", ws.BaseURL(), sessionID)
    png, err := qrcode.Encode(url, qrcode.Medium, 256)
    if err != nil {
        return "", fmt.Errorf("GetSessionQRCode: %w", err)
    }
    return base64.StdEncoding.EncodeToString(png), nil
}
```

### Pattern 2: QR Code Display (React)

**What:** Render QR as `<img>` using the base64 string from Go.
**When to use:** When session has web serving enabled and user clicks a "Show QR" button.

```tsx
// No external npm dependency needed
interface QRModalProps {
  sessionId: string
  onClose: () => void
}

function QRModal({ sessionId, onClose }: QRModalProps) {
  const [qrDataURL, setQrDataURL] = useState<string | null>(null)
  useEffect(() => {
    GetSessionQRCode(sessionId).then(b64 => {
      setQrDataURL(`data:image/png;base64,${b64}`)
    })
  }, [sessionId])
  return (
    <div className="qr-modal-overlay" onClick={onClose}>
      <div className="qr-modal" onClick={e => e.stopPropagation()}>
        {qrDataURL && <img src={qrDataURL} alt="Session QR code" width={256} height={256} />}
        <button onClick={onClose}>Close</button>
      </div>
    </div>
  )
}
```

### Pattern 3: QR Code on Web Dashboard

**What:** Add a `/api/sessions/{id}/qr` GET endpoint that serves the PNG directly (Content-Type: image/png). The dashboard renders it as `<img src="/api/sessions/ID/qr">`.
**When to use:** When a session is web-enabled and the dashboard is displaying its row.

```go
// In webserver/server.go setupRoutes():
mux.HandleFunc("GET /api/sessions/{id}/qr", ws.dashboardAuth(ws.handleSessionQR))

// Handler:
func (ws *WebServer) handleSessionQR(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    if !ws.isSessionEnabled(sessionID) {
        http.NotFound(w, r); return
    }
    url := fmt.Sprintf("%s/sessions/%s", ws.BaseURL(), sessionID)
    png, err := qrcode.Encode(url, qrcode.Medium, 256)
    if err != nil {
        http.Error(w, "qr error", 500); return
    }
    w.Header().Set("Content-Type", "image/png")
    w.Write(png)
}
```

### Pattern 4: Status Detector (Go)

**What:** A goroutine that taps the Hub's output stream, runs regex over a rolling tail, and emits Wails events on state transitions.
**When to use:** Started when a session hub is created; stopped when the hub shuts down.

```go
// internal/status/detector.go

type SessionStatus string

const (
    StatusRunning  SessionStatus = "running"   // CLI actively producing output
    StatusWaiting  SessionStatus = "waiting"   // Prompt shown, waiting for y/n or text
    StatusIdle     SessionStatus = "idle"       // Prompt shown, no pending question
    StatusErrored  SessionStatus = "errored"   // Process exited non-zero or hub closed with error
)

type Detector struct {
    sessionID string
    onStatus  func(sessionID string, status SessionStatus)
}

// Claude Code prompt patterns (empirical, from claude-tmux and community research):
var (
    // reClaudeWorking: output contains "ctrl+c to interrupt" — Claude is processing
    reClaudeWorking = regexp.MustCompile(`(?i)ctrl\+c to interrupt`)
    // reClaudeIdle: the ❯ prompt appears without an interrupt message
    reClaudeIdle    = regexp.MustCompile(`❯\s*$`)
    // reWaiting: y/n confirmation prompt — any CLI
    reWaiting       = regexp.MustCompile(`\[y/n\]|\[Y/n\]|\[y/N\]`)
)
```

### Pattern 5: Status Push via Wails Events

**What:** The App wires status transitions to `runtime.EventsEmit`.

```go
// In app.go, when creating a hub:
go status.Watch(hub, sessionID, func(id string, s status.SessionStatus) {
    runtime.EventsEmit(a.ctx, "session:status", map[string]string{
        "sessionId": id,
        "status":    string(s),
    })
})
```

```tsx
// In App.tsx useEffect:
const unsub = EventsOn("session:status", (data: {sessionId: string, status: string}) => {
    setSessionStatuses(prev => ({ ...prev, [data.sessionId]: data.status }))
})
return () => unsub()
```

### Anti-Patterns to Avoid

- **Polling `GetSessionStatus()` from the frontend:** Adds roundtrip latency on every poll interval; EventsEmit push is the idiomatic Wails pattern.
- **Writing QR PNG to disk:** Unnecessary I/O; skip2/go-qrcode returns bytes in memory — keep it there.
- **Parsing raw ANSI bytes without stripping sequences:** ANSI CSI sequences (`\x1b[...m`) will confuse string pattern matches. Strip escape codes from the rolling tail before applying regex.
- **Sharing a single `[]byte` buffer across goroutines:** The Hub already copies frames with `MakeOutputFrame` so each frame is safe — status detector reads its own copy.
- **Blocking the Hub's drain loop for status detection:** The status detector must run in its own goroutine subscribing to the Hub (same as a WebSocket client), not inline in the drain path.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| QR code matrix generation | Custom Reed-Solomon encoder | `skip2/go-qrcode` | QR encoding has 40 version levels, 4 ECC modes, masking algorithms — O(100) edge cases |
| ANSI escape stripping | Custom regex | `leaanthony/go-ansi-parser` (already indirect dep) or simple `\x1b\[[0-9;]*[mABCDEFGHJKLMST]` regex | The existing indirect dep can strip sequences; a simple regex covers 99% of cases |
| Base64 PNG data URI | Custom encoder | `encoding/base64` stdlib | stdlib is correct and zero-cost |

**Key insight:** QR code generation looks simple (a 2D barcode) but has multiple interleaved error-correction stages and 8 mask patterns that must all be evaluated — always use a library.

---

## Common Pitfalls

### Pitfall 1: ANSI Escape Codes in Status Patterns
**What goes wrong:** Regex for `❯` or `ctrl+c to interrupt` fails to match because the text is surrounded by ANSI color codes (e.g., `\x1b[1;32m❯\x1b[0m`).
**Why it happens:** Claude Code uses color styling for its prompt and status lines.
**How to avoid:** Strip ANSI sequences from the rolling tail before applying status regexes. A single regex `\x1b\[[0-9;]*[a-zA-Z]` handles most CSI sequences.
**Warning signs:** Status always shows "running" even when Claude is visually idle.

### Pitfall 2: Hub Subscriber Not Unsubscribed on Shutdown
**What goes wrong:** Status detector goroutine leaks after session close; the channel blocks forever.
**Why it happens:** Hub.Done() must be selected in the status detector's loop.
**How to avoid:** Use the same select-on-done pattern used in `handleWSSRelay`:
```go
select {
case frame := <-sub.Msgs: ...
case <-hub.Done(): return
}
```

### Pitfall 3: QR Code URL Before Server Starts
**What goes wrong:** `GetSessionQRCode` is called before `StartWebServer` — `ws.BaseURL()` returns `""` and the QR code encodes an empty URL.
**Why it happens:** Frontend may call the method optimistically when toggling web serving.
**How to avoid:** Return an error if `ws == nil` or `ws.Addr() == ""`. Guard at the call site: show the QR button only when `webServerRunning` is true.

### Pitfall 4: Status Detector Race on Hub Not Yet Started
**What goes wrong:** Subscribing to the Hub before `hub.Run()` is called is fine (Subscribe is safe pre-Run), but if the detector goroutine starts reading from `sub.Msgs` before Run starts, it will block. This is benign but wastes a goroutine slot.
**Why it happens:** App.CreateSession wires everything up before the Hub drain goroutine starts.
**How to avoid:** Start the status detector goroutine inside `go hub.Run()` — or just accept the benign block since it unblocks as soon as Run starts.

### Pitfall 5: Per-CLI Pattern Coverage is Incomplete
**What goes wrong:** Status shows "running" for Gemini CLI / Codex / OpenCode at all times because their prompt patterns differ from Claude Code.
**Why it happens:** STATE.md documents this as a known concern: "Per-CLI status indicator output patterns for Codex, Gemini CLI, OpenCode undocumented — empirical testing needed."
**How to avoid:** Design the detector with a pluggable pattern set keyed by CLI name. Claude Code patterns are confirmed; other CLI patterns are initially best-effort and can be refined empirically. Use a conservative fallback: if no pattern matches, default to "running" (not "idle") to avoid false "done" signals.

### Pitfall 6: Rolling Buffer Size vs. Pattern Visibility
**What goes wrong:** The rolling tail used for status detection is too short — the `ctrl+c to interrupt` text scrolled off before the detector checked.
**Why it happens:** Claude Code renders multi-line status bars that can be dozens of bytes of ANSI data.
**How to avoid:** Keep a rolling tail of ~4 KB of stripped (ANSI-removed) text. This is enough to catch all prompt/status patterns without significant memory cost per session.

---

## Code Examples

### skip2/go-qrcode in-memory PNG

```go
// Source: pkg.go.dev/github.com/skip2/go-qrcode
import qrcode "github.com/skip2/go-qrcode"

// Returns raw PNG bytes — no file I/O
png, err := qrcode.Encode("https://100.64.1.1:7443/sessions/abc123", qrcode.Medium, 256)
if err != nil { return err }
b64 := base64.StdEncoding.EncodeToString(png)
// b64 is safe to embed in JSON or a data: URI
```

### Wails EventsEmit (Go → Frontend)

```go
// Source: wails.io/docs/reference/runtime/events (v2)
import "github.com/wailsapp/wails/v2/pkg/runtime"

runtime.EventsEmit(ctx, "session:status", map[string]string{
    "sessionId": "abc123",
    "status":    "waiting",
})
```

### Wails EventsOn (TypeScript)

```tsx
// Source: frontend/src/wailsjs/wailsjs/runtime/runtime.d.ts (already in repo)
import { EventsOn } from '../wailsjs/wailsjs/runtime/runtime'

useEffect(() => {
  const off = EventsOn('session:status', (data: {sessionId: string; status: string}) => {
    setSessionStatuses(prev => ({ ...prev, [data.sessionId]: data.status }))
  })
  return off  // EventsOn returns an unsubscribe function
}, [])
```

### ANSI Escape Strip Regex

```go
// Covers the vast majority of CSI sequences emitted by Claude Code
var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b[()][AB012]|\x1b[=>]`)

func stripANSI(b []byte) []byte {
    return reANSI.ReplaceAll(b, nil)
}
```

### Status Detection State Machine (skeleton)

```go
// internal/status/detector.go
type Detector struct {
    sessionID string
    cli       string          // e.g. "claude", "gemini", "codex", "opencode"
    tail      []byte          // rolling window, max 4 KB stripped text
    current   SessionStatus
    onTransit func(string, SessionStatus)
}

func (d *Detector) Feed(raw []byte) {
    stripped := stripANSI(raw)
    d.tail = appendTail(d.tail, stripped, 4096)
    next := d.classify()
    if next != d.current {
        d.current = next
        d.onTransit(d.sessionID, next)
    }
}

func (d *Detector) classify() SessionStatus {
    tail := string(d.tail)
    if reWaiting.MatchString(tail)      { return StatusWaiting }
    if reClaudeWorking.MatchString(tail) { return StatusRunning }
    if reClaudeIdle.MatchString(tail)   { return StatusIdle }
    return StatusRunning // conservative default
}
```

---

## CLI Status Pattern Reference

This section documents known and inferred patterns per CLI.

### Claude Code (`claude`)

| State | Pattern | Source | Confidence |
|-------|---------|--------|------------|
| Running/Processing | `ctrl+c to interrupt` (case-insensitive) in recent output | claude-tmux project (empirical) | MEDIUM |
| Idle (done, waiting for input) | `❯` prompt at end of line, no interrupt message | claude-tmux project (empirical) | MEDIUM |
| Waiting (y/n) | `[y/n]`, `[Y/n]`, `[y/N]` anywhere in output | claude-tmux project + community | MEDIUM |
| Errored | Hub.Done() closed + process exit non-zero | PTY backend StatesStopped | HIGH |

### Gemini CLI (`gemini`)

| State | Pattern | Source | Confidence |
|-------|---------|--------|------------|
| Running | Continuous output; no recognizable prompt | Inferred | LOW |
| Idle | `>` or `?` or other prompt at line end | Needs empirical observation | LOW |

### Codex CLI (`codex`)

| State | Pattern | Source | Confidence |
|-------|---------|--------|------------|
| Running | Active output, tool use markers | Inferred | LOW |
| Idle | Prompt at end of output | Needs empirical observation | LOW |

### OpenCode (`opencode`)

| State | Pattern | Source | Confidence |
|-------|---------|--------|------------|
| All states | Patterns undocumented publicly | Needs empirical observation | LOW |

**Design decision:** Detector must accept a `patterns PatternSet` argument keyed by CLI name so each CLI's patterns can be tuned independently without changing the core state machine. Unknown CLIs use a conservative "running until hub closes" default.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| QR codes written to disk as PNG files | In-memory PNG bytes via `Encode()` | skip2/go-qrcode API | No temp files, no cleanup required |
| Polling for status from frontend | Push via EventsEmit | Wails v2 | Zero latency status updates, no polling overhead |
| Hardcoded Claude-only status patterns | Pluggable PatternSet per CLI | Phase 5 design | Extensible to all 4 CLIs without refactor |

**Deprecated/outdated:**
- Writing QR to a temp file and serving it: unnecessary with in-memory generation
- Using `qrcode.WriteFile()` from skip2: only needed for disk-backed workflows

---

## Open Questions

1. **Gemini CLI, Codex, OpenCode prompt patterns**
   - What we know: These CLIs exist in the codebase (REQUIREMENTS.md CLI-01) but their output patterns are undocumented
   - What's unclear: What prompt symbol/string appears when each CLI is idle vs processing
   - Recommendation: Implement Claude Code patterns (MEDIUM confidence) in Phase 5; stub LOW-confidence patterns as "running until hub closes" with a comment noting they need empirical tuning. Create a `PatternSet` type so patterns can be updated in a follow-up PR.

2. **Should the web dashboard QR be embedded inline or via an endpoint?**
   - What we know: The dashboard is a static HTML file embedded in the binary (`web/dashboard.html`); it cannot import Go libraries
   - What's unclear: Whether to add a `/api/sessions/{id}/qr` endpoint or generate QR client-side with a JS library
   - Recommendation: Add the endpoint (consistent with the existing `/ca.crt` pattern); no JS QR library needed. The endpoint is protected by `dashboardAuth` middleware already used on other `/api/sessions/` routes.

3. **Wails context availability when emitting events**
   - What we know: `runtime.EventsEmit` requires a `context.Context` that must come from `app.startup(ctx)`; the `a.ctx` is set in `startup()` and used by all other runtime calls in the codebase
   - What's unclear: Whether status detector goroutines can safely capture `a.ctx` or must receive it via channel
   - Recommendation: Pass `a.ctx` to the detector at creation time (same pattern used by `CreateSession`); this is safe because the detector shuts down when the hub closes, which always happens before `a.shutdown()` clears state.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package (existing) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./internal/status/... -v` |
| Full suite command | `go test ./... -race` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| QR-01 | `GetSessionQRCode` returns valid base64 PNG for a running web server | unit | `go test . -run TestGetSessionQRCode -v` | Wave 0 |
| QR-01 | `qrcode.Encode` produces scannable QR for a real URL | unit | `go test ./internal/... -run TestQREncode` | Wave 0 |
| QR-02 | `/api/sessions/{id}/qr` returns PNG with correct Content-Type | integration | `go test ./internal/webserver/... -run TestQREndpoint` | Wave 0 |
| QR-02 | QR endpoint returns 404 for non-web-enabled session | integration | `go test ./internal/webserver/... -run TestQREndpointNotEnabled` | Wave 0 |
| STAT-01 | `session:status` Wails event fires on state transition | integration | `go test . -run TestStatusEventEmit` (mock ctx) | Wave 0 |
| STAT-02 | Detector correctly classifies Claude Code patterns | unit | `go test ./internal/status/... -run TestDetector` | Wave 0 |
| STAT-02 | ANSI stripping does not corrupt prompt detection | unit | `go test ./internal/status/... -run TestANSIStrip` | Wave 0 |
| STAT-02 | Hub subscriber cleanup on Detector shutdown (no goroutine leak) | unit | `go test ./internal/status/... -run TestDetectorShutdown` | Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/status/... ./internal/webserver/... -v`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/status/detector_test.go` — covers STAT-02 patterns + ANSI stripping + shutdown
- [ ] `internal/webserver/server_test.go` additions — QR endpoint tests (TestQREndpoint, TestQREndpointNotEnabled)
- [ ] `app_test.go` additions — TestGetSessionQRCode against testApp helper

---

## Sources

### Primary (HIGH confidence)

- `pkg.go.dev/github.com/skip2/go-qrcode` — Encode() signature, RecoveryLevel constants, in-memory PNG return type
- `/Users/ken/dev/agenthub/frontend/src/wailsjs/wailsjs/runtime/runtime.d.ts` — EventsOn/EventsEmit TypeScript signatures (in-repo, authoritative)
- `/Users/ken/dev/agenthub/internal/relay/hub.go` — Hub subscriber/broadcast/Done pattern (in-repo)
- `/Users/ken/dev/agenthub/internal/webserver/server.go` — existing route patterns, dashboardAuth middleware (in-repo)
- `/Users/ken/dev/agenthub/app.go` — GetWebServerURL, ToggleWebServing, EventsEmit usage context (in-repo)

### Secondary (MEDIUM confidence)

- [github.com/nielsgroen/claude-tmux](https://github.com/nielsgroen/claude-tmux) — Claude Code status patterns: `❯` = idle, `ctrl+c to interrupt` = working, `[y/n]` = waiting
- [github.com/anthropics/claude-code/issues/25307](https://github.com/anthropics/claude-code/issues/25307) — confirms Claude Code uses tab title emoji `·` (working) / `✳` (idle); no OSC sequence exposed externally

### Tertiary (LOW confidence)

- WebSearch: Gemini CLI, Codex, OpenCode status patterns — no authoritative source found; patterns need empirical observation
- [wails.io events docs](https://wails.io/docs/reference/runtime/events/) — EventsEmit Go signature (403 at research time; confirmed by in-repo .d.ts file)

---

## Metadata

**Confidence breakdown:**
- Standard stack (QR generation): HIGH — skip2/go-qrcode API verified via pkg.go.dev; in-memory Encode() confirmed
- Standard stack (Wails events): HIGH — EventsEmit/EventsOn confirmed from in-repo runtime.d.ts
- Architecture: HIGH — follows existing Hub subscriber pattern exactly; no new primitives
- Claude Code status patterns: MEDIUM — empirically documented by community tools; not official API
- Other CLI patterns: LOW — undocumented; need empirical testing

**Research date:** 2026-03-18
**Valid until:** 2026-06-18 (stable libraries; CLI pattern strings could change with Claude Code updates)
