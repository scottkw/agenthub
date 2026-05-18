# Phase 111: Web bridge OSC/DA response consumption - Research

**Researched:** 2026-05-18
**Domain:** Server-side WebSocket relay, terminal escape-sequence parsing (Go)
**Confidence:** HIGH

## Summary

Phase 111 fixes a leak in the AgentHub web bridge where OSC 10/11 (color queries) and CSI c (DA1) responses emitted by xterm.js on the browser are forwarded verbatim to PTY stdin, causing programs like `chafa --format=sixel` to render their probe responses as printable garbage on the next prompt. The fix is a small streaming filter in `internal/webserver/server.go` that sits between `relay.ParseFrame(msg)` returning `MsgInput` and `hub.WriteInput(payload)` — consuming bytes that fall inside an OSC 10 / OSC 11 / DA1 envelope, forwarding everything else verbatim, and buffering partial envelopes across WebSocket frames.

The browser-side code (`web/assets/terminal.js` for the served web surface, `frontend/src/components/TerminalPanel.tsx` for the Wails desktop) is identical in shape: `term.onData((data) => sendInput(data))`. xterm.js's `onData` event fires for both real keystrokes AND for automatic terminal-query replies that xterm.js generates internally (verified at `frontend/node_modules/.pnpm/@xterm+xterm@6.0.0/node_modules/@xterm/xterm/src/common/InputHandler.ts:1672` for DA1 and `src/browser/CoreBrowserTerminal.ts:217` for OSC 10/11). The bug is that the server-side relay has zero awareness of the distinction.

**Primary recommendation:** Implement a small `InputAbsorber` type in a new file `internal/webserver/oscabsorb.go` with a single `Filter([]byte) []byte` method and a per-subscriber instance held in the read pump. The state machine has three states (outside, in-OSC-body, in-CSI-body) plus a small carry-over buffer for partial frames. Wire it into both `handleWSSRelay` (in `internal/webserver/server.go:753-756`) and — out of scope for this phase per CONTEXT, but worth noting — the parallel call site at `internal/relay/server.go:124-128`.

## User Constraints (from CONTEXT.md)

### Locked Decisions

**Root cause:**
- Web-only bug — desktop (Wails webview) goes through xterm.js directly and consumes those responses cleanly; the dedicated web relay path in `internal/webserver` does NOT.
- The web bridge currently does not strip / consume OSC 10/11 / DA1 responses that xterm.js (on the browser) emits in reply to terminal queries from the running program — so chafa's color queries get echoed back to shell stdin instead of being absorbed.
- Pre-existing (not a v3.3 regression).

**Fix shape:**
- Server-side consumption in the web bridge (in `internal/webserver/relay.go` or equivalent). The bridge proxies WebSocket frames between browser and PTY. It must parse the browser → PTY direction for terminal-query responses (OSC 10, OSC 11, CSI c / DA1) and consume them — NOT forward them to the PTY's stdin.
- Why server-side, not client-side: parity with how the Wails webview path already handles this — xterm.js on the desktop side absorbs the responses before they hit any forwarding path. The web bridge needs to mirror that absorption. A client-side fix in the browser would split the contract between web and desktop and add complexity.
- Three response shapes to consume (no false positives):
  - `\x1b]10;rgb:RRRR/GGGG/BBBB\x1b\\` — OSC 10 FG color reply (BEL `\x07` terminator also valid).
  - `\x1b]11;rgb:RRRR/GGGG/BBBB\x1b\\` — OSC 11 BG color reply (BEL terminator also valid).
  - `\x1b[?<params>c` — CSI c / DA1 reply (numeric params like `62;4;9;22`).
- Streaming state machine across WebSocket frames — these responses can be split across frames. A naive byte-by-byte regex won't work. Use a small state machine that tracks "in OSC body" / "in CSI body" / "outside" and buffers the response across frames if needed.
- Forward everything else verbatim — user keystrokes, control sequences not in the absorption set, etc. Must NOT regress normal terminal interaction.

**Test surface (WEB-03):**
- Go-level regression test in `internal/webserver/` that feeds the relay synthetic OSC 10/11 / DA1 responses (across frame boundaries) and asserts the relay does NOT forward them downstream.

**Cross-surface verification (release gate):**
- Web: open browser, web-share a shell session, run `chafa --format=sixel /tmp/test.png`. Clean prompt afterward; no leaked `10;rgb:…`, `11;rgb:…`, `62;4;9;22c` strings.
- Desktop: same `chafa` command on a Wails-attached session — confirm unchanged.
- Test on macOS using local web-share.

### Claude's Discretion

(None explicitly granted in CONTEXT.md — all major decisions are locked. Researcher / planner discretion remains for *how* the state machine is structured internally, naming, logging verbosity, and exact test layout.)

### Deferred Ideas (OUT OF SCOPE)

- xterm.js client-side changes — server-side fix only.
- Other terminal-query absorptions (e.g., OSC 4 palette query) — only the three documented in WEB-01 (OSC 10, OSC 11, DA1).
- Mirroring the same absorption into `internal/relay/server.go` (the desktop relay path) — CONTEXT scopes the fix to `internal/webserver` only. See "Open Questions" below regarding the empirical desktop-vs-web asymmetry.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WEB-01 | OSC 10, OSC 11, and CSI c responses are consumed by the requesting program and do NOT appear in shell stdin. Reproducible with `chafa --format=sixel /tmp/<png>`. | Section "Current web-bridge data flow" identifies the exact line where input bytes are written to PTY (`internal/webserver/server.go:755`). Section "Three response shapes — exact byte sequences" provides the verified byte-level patterns. Section "Streaming state machine design" gives the absorption algorithm. |
| WEB-02 | Web ↔ desktop parity holds for chafa sixel rendering. | Section "macOS web UAT setup" describes how to reproduce both surfaces on a single dev machine. Section "Open Questions" flags the empirical asymmetry that motivates the parity gate. |
| WEB-03 | Regression test (Go or e2e) covers OSC response consumption on the web bridge. | Section "Test scaffolding" identifies `testServerWithHub` in `internal/webserver/capability_test_helpers.go:131` as the existing harness — already returns an `*io.PipeReader` (`inputCaptureR`) the test can assert on. |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Detect and absorb terminal-query replies on the browser→PTY path | **API / Backend** (`internal/webserver`) | — | Locked by CONTEXT: server-side parity with how xterm.js (desktop webview) absorbs responses. Client-side would split the contract between surfaces. |
| Emit OSC 10/11/DA1 replies in response to program queries | **Browser / Client** (xterm.js) | — | xterm.js is the terminal emulator; it owns terminal-state queries by spec. Bug is downstream of this, not in this layer. |
| Forward keystrokes / control input verbatim | **API / Backend** (`internal/webserver`) | — | Existing behavior; absorption layer must NOT regress it. |
| PTY write | **OS / Process** (`internal/relay/hub.go::WriteInput`) | — | Unchanged; absorption layer sits one frame upstream. |

## Standard Stack

### Core (already in use, not added by this phase)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/coder/websocket` | v1.8.14 | WebSocket server-side, used by `internal/webserver` and `internal/relay` | [VERIFIED: go.mod] Project standard; `nhooyr/websocket` was renamed/forked to `coder/websocket`. |
| `github.com/scottkw/agenthub/internal/relay` | local | Binary framing protocol (`MsgInput`, `MsgOutput`, etc.) and Hub fan-out | [VERIFIED: codebase] Locked architecture; absorption sits between `relay.ParseFrame` and `hub.WriteInput`. |

### Supporting (no new deps required)

[VERIFIED: codebase grep] The absorption layer is pure-Go state-machine code over `[]byte` slices. No new dependencies need to be added — neither for ANSI parsing nor for buffering. Go's stdlib `bytes` package is sufficient if any helper is wanted, but the state machine is small enough to need no helpers at all.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled state machine | `github.com/leaanthony/go-ansi-parser` or similar | [ASSUMED] Adds a dep for what is ~50 lines of straightforward code; full ANSI parsers handle far more than three sequences and are overkill. Hand-roll wins on auditability and dep hygiene. |
| Filter on `WriteInput` (hub-level) | Filter at `handleWSSRelay` read pump | Filtering inside the hub would also intercept input from `cmd_attach.go` and Wails desktop path → out of scope. Filter at the webserver read pump only — scoped to the browser → web-bridge direction. |
| Regex over the whole frame | Streaming byte-level state machine | Regex breaks on frame boundaries (OSC body split across two WS frames) and on partial-match anchoring. State machine is the only correct approach. |

**Installation:** No new packages.

**Version verification:** `coder/websocket` v1.8.14 verified in `go.mod`. Project last touched it in Phase 88 / 92.

## Architecture Patterns

### System Architecture Diagram

```
                    [ Running program: chafa, vim, ... ]
                                   ▲
                          PTY stdin (writes here)
                                   │
                       hub.WriteInput(filtered) ◀──┐
                                   ▲                │
                                   │       filtered bytes (keystrokes
                                   │        & non-absorbed sequences)
                                   │                │
                              [ NEW: InputAbsorber.Filter() ]   ◀── this phase
                                   ▲
                          relay.ParseFrame -> MsgInput payload
                                   ▲
                                   │      (binary WS frame: 0x10 + UTF-8 bytes)
                                   │
                       handleWSSRelay read pump  ◀── browser sends input here
                                   ▲
                                   │      WebSocket binary frame
                                   │
                       [ Browser: term.onData(data => ws.send(makeInputFrame(data))) ]
                                   ▲
                                   │
                       xterm.js InputHandler emits:
                       - keystroke text     (e.g. "ls\r")
                       - OSC 10/11 reply    (in response to chafa probe)
                       - CSI ?1;2c reply    (in response to DA1 probe)
                                   ▲
                                   │      PTY → Hub → broadcast → client.write
                                   │
                       [ Running program writes \x1b[c or \x1b]11;?\x07 ]
```

The fix injects exactly one new node: **InputAbsorber.Filter()** between `relay.ParseFrame` and `hub.WriteInput` in `handleWSSRelay`. All other arrows are unchanged.

### Component Responsibilities

| Component | File:Line | Responsibility | Changed by this phase? |
|-----------|-----------|----------------|------------------------|
| Browser `term.onData` handler | `web/assets/terminal.js:968-973`, `frontend/src/components/TerminalPanel.tsx:283` | Emit ALL xterm.js-generated bytes (keystrokes + query replies) as `MsgInput` frames | No |
| WS read pump | `internal/webserver/server.go:739-767` | Decode `MsgInput` frame, write payload to PTY | **Yes** — inject `absorber.Filter` between `ParseFrame` and `WriteInput` |
| `relay.ParseFrame` | `internal/relay/protocol.go:53` | Strip the 1-byte message-type prefix | No |
| `hub.WriteInput` | `internal/relay/hub.go:196` | Forward bytes to the PTY's underlying writer | No |
| **NEW** `InputAbsorber` | `internal/webserver/oscabsorb.go` (proposed) | Stateful byte-level filter that consumes OSC 10/11/DA1 envelopes and passes everything else through | New file |

### Recommended Project Structure

```
internal/webserver/
├── oscabsorb.go          # NEW — InputAbsorber type, Filter([]byte) []byte method
├── oscabsorb_test.go     # NEW — table-driven tests over byte sequences
├── server.go             # MODIFIED — instantiate absorber per subscriber, call Filter before WriteInput
└── ...                   # everything else unchanged
```

### Pattern 1: Per-subscriber state, instantiated in read pump

```go
// Source: pattern observed in internal/webserver/server.go:714-767 (Subscriber lifecycle)
sub := &relay.Subscriber{ ... }
absorber := &InputAbsorber{} // zero value is the "outside" state

// In read pump:
case relay.MsgInput:
    if !sub.ReadOnly {
        filtered := absorber.Filter(payload)
        if len(filtered) > 0 {
            _ = hub.WriteInput(filtered)
        }
    }
```

The absorber is per-subscriber because each browser tab has its own xterm.js instance with its own pending query state. Multiple subscribers on the same session must not share absorber state, or one tab's mid-OSC-envelope could swallow another tab's keystroke.

### Pattern 2: State machine over bytes (no buffer for "outside" state)

```go
// Source: hand-rolled, but standard ANSI-parsing shape (see e.g. vt10x, ANSITERM in xterm.js InputHandler)
type AbsorbState int
const (
    StateOutside AbsorbState = iota
    StateGotEsc              // saw \x1b, deciding ]/[
    StateInOSC               // inside \x1b]... waiting for BEL or ESC\
    StateInOSCSeenEsc        // inside OSC, saw ESC, expecting \ for ST
    StateInCSI               // inside \x1b[... waiting for final byte 0x40-0x7E
)

type InputAbsorber struct {
    state  AbsorbState
    // For OSC: accumulate the parameter bytes so we can confirm the OSC code
    // matches 10 or 11 (avoid absorbing unrelated OSC like OSC 52 clipboard).
    oscBuf []byte
    // For CSI: same — accumulate so we can confirm the final byte is 'c'
    // and the parameter prefix is '?' (DA1 reply form).
    csiBuf []byte
    // Spillover bytes to PROduce when an envelope turns out to be non-absorbed
    // (e.g. an OSC 52 clipboard that we started buffering, then realised we
    // should pass through).
    passthrough []byte
}

func (a *InputAbsorber) Filter(in []byte) []byte { /* ... */ }
```

See Section "Streaming state machine design" for the full algorithm including the false-positive-avoidance strategy.

### Anti-Patterns to Avoid

- **Regex on `payload` per frame.** Breaks on envelope splits across frames. Don't.
- **Absorbing ALL OSC sequences.** OSC 52 (clipboard), OSC 8 (hyperlinks emitted by some user pastes), OSC 9;4 (progress) are legitimate user-input paths in some scenarios — do NOT swallow them. Filter ONLY by parsed code (10 / 11 for OSC, `?`-prefixed `c` for CSI).
- **Absorbing the whole envelope before knowing the code.** If a frame ends mid-OSC-body before we've seen the `;` that delimits the code, we must buffer; if the eventual code turns out to be e.g. OSC 52, we have to FLUSH the buffered bytes into the output. The algorithm must handle "I was buffering, now I realize this is passthrough" correctly.
- **Shared absorber state across subscribers.** State must be per-`Subscriber`. The existing `*relay.Subscriber` struct could carry it, but a separate `absorber := &InputAbsorber{}` local in `handleWSSRelay` is simpler and matches the `sub` lifetime.
- **Logging at debug level on every byte.** Log absorption events (one log per fully-absorbed envelope) for UAT auditing; don't log per-byte.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Full ANSI parser | Anything resembling a full state machine for *all* CSI/OSC/DCS/SS3 sequences | A narrow 3-sequence filter (OSC 10/11, CSI DA1) | Out of scope per CONTEXT. A full parser doubles the surface area and introduces regression risk for paste/clipboard/IIP sequences. |
| Per-byte logging | A `slog.Debug` call inside the byte loop | One log per absorbed envelope, on the OSC/CSI completion transitions | At ~30 keystrokes/sec on a single subscriber, per-byte logs flood logs and bias hot-path timing. |
| Buffered I/O wrapping | `bufio.Reader` over the WebSocket payload | Plain `[]byte` slice + indexes | Each `MsgInput` payload is already a complete `[]byte`; bufio adds nothing. |

**Key insight:** Three sequences is small enough that a hand-rolled byte-by-byte loop is more auditable and more reliable than any library. The complexity isn't in the parsing — it's in the cross-frame state continuity. A library doesn't solve that for you any cheaper than you can solve it in ~80 lines of Go.

## Runtime State Inventory

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — pure code change, no persisted state involved. | None. |
| Live service config | None — no daemon config, no service registration touched. | None. |
| OS-registered state | None — no OS-level state. | None. |
| Secrets/env vars | None — no secret rotation, no env var changes. | None. |
| Build artifacts | None — Go build is hermetic; `go build ./...` after the change is sufficient. | None. |

**Nothing found in any category — verified by:** Phase scope is code-only inside `internal/webserver`. No data migration, no service config update, no OS registration. The only "rebuild" implication is the standard `go build` after the source change (and `wails build -tags wailsassets` only if any embed.FS asset is touched — this phase doesn't touch web assets).

## Current Web-Bridge Data Flow

**Browser → PTY direction, byte-by-byte:**

1. **xterm.js emits a string via `term.onData`.** The string may be a real keystroke (e.g. `"a"` or `"\r"`) OR a synthetic terminal-query reply (e.g. `"\x1b]10;rgb:cccc/cccc/cccc\x1b\\"`).
   - **Source for keystrokes:** xterm.js core, triggered by `keydown`/`paste`/`input` events.
   - **Source for color replies:** `frontend/node_modules/.pnpm/@xterm+xterm@6.0.0/node_modules/@xterm/xterm/src/browser/CoreBrowserTerminal.ts:217` — `triggerDataEvent(\`${C0.ESC}]${ident};${toRgbString(colorRgb)}${C1_ESCAPED.ST}\`)`.
   - **Source for DA1 replies:** `frontend/node_modules/.pnpm/@xterm+xterm@6.0.0/node_modules/@xterm/xterm/src/common/InputHandler.ts:1672` — `this._coreService.triggerDataEvent(C0.ESC + '[?1;2c')` (for `xterm`/`rxvt-unicode`/`screen` TERM values; AgentHub PTY sets `TERM=xterm-256color` so this branch fires — verified in `internal/pty/native.go:46`).

2. **Browser-side handler wraps the string in a `MsgInput` frame and sends over WebSocket.**
   - **Web (served terminal):** `web/assets/terminal.js:968-973` — `term.onData(function(data) { ws.send(makeInputFrame(data)); })`.
   - **Wails desktop:** `frontend/src/components/TerminalPanel.tsx:283` — `term.onData((data) => client.sendInput(data))`, which calls `relayClient.ts:119-123 → ws.send(encodeInputFrame(text))`.
   - **Wire format:** Binary WebSocket frame; first byte is `MsgInput` (0x10), remaining bytes are the UTF-8 encoded payload. See `internal/relay/protocol.go:32-37`.

3. **Server-side WS read pump receives the frame.**
   - **Web bridge:** `internal/webserver/server.go:744` — `_, msg, err := conn.Read(ctx)`.
   - **Desktop relay:** `internal/relay/server.go:116` — same shape.

4. **`relay.ParseFrame(msg)` splits the type byte and payload.** See `internal/relay/protocol.go:53-58`.

5. **`hub.WriteInput(payload)` writes the payload to the PTY.**
   - **Web bridge call site:** `internal/webserver/server.go:753-756`.
     ```go
     case relay.MsgInput:
         if !sub.ReadOnly { // MC-03: discard input for read-only clients
             _ = hub.WriteInput(payload)   // ◀── PHASE 111 INJECTS FILTER HERE
         }
     ```
   - **Desktop relay call site:** `internal/relay/server.go:124-128` — identical shape. **OUT OF SCOPE per CONTEXT** but worth flagging in Open Questions.

6. **`hub.WriteInput` writes to the PTY's writer.** See `internal/relay/hub.go:195-199` — `_, err := h.writer.Write(data)`. The writer is the PTY stdin pipe.

**The bug:** Step 5 forwards xterm.js's *synthetic* query replies (from step 1) as if they were keystrokes. From the running program's perspective, it asked "what's your background color?" and got back its own answer typed into stdin — which then appears at the next prompt after the program exits.

## Existing Terminal Escape Sequence Handling

[VERIFIED: grep across `internal/webserver/` and `internal/relay/`]

**There is currently NO escape-sequence parsing on the input path anywhere in `internal/webserver/` or `internal/relay/`.**

Search for `OSC`, `CSI`, `\x1b`, `0x1b`, `\033`, "escape" across both packages in non-test files:

```
$ grep -rn "OSC\|CSI\|\\\\x1b\|0x1b\|escape" internal/webserver/ internal/relay/ \
    | grep -v "_test.go"
(empty)
```

The output path treats PTY bytes as opaque (chunked, frame-prepended, broadcast). The input path treats client bytes as opaque (parse frame type, forward payload). Both packages have zero awareness of terminal semantics — by design, until now.

**Frontend has some escape awareness** (OSC 8 hyperlink handling in `TerminalPanel.tsx:464+`) but that's downstream of xterm.js parsing, not at the WebSocket boundary.

**Implication:** This phase introduces the *first* escape-sequence parser on the input path in this codebase. There is no existing helper to extend — it's all greenfield within the `internal/webserver` package.

## Three Response Shapes — Exact Byte Sequences

All three confirmed against the embedded xterm.js v6.0.0 source under `frontend/node_modules/.pnpm/@xterm+xterm@6.0.0/`.

### OSC 10 Reply (Foreground Color)

```
\x1b ] 10 ; rgb : RRRR / GGGG / BBBB \x1b \\
```

[VERIFIED: `CoreBrowserTerminal.ts:217`, `XParseColor.ts:77-79`, `EscapeSequences.ts:152`]

- Introducer: `\x1b]` (ESC + `]`) — the OSC introducer per ECMA-48.
- Code: `10` (literal ASCII `"10"`).
- Separator: `;`.
- Payload: literal `rgb:` followed by three hex channels separated by `/`. xterm.js v6.0.0 defaults to 16-bit channels — `toRgbString(color, bits=16)` → `rgb:RRRR/GGGG/BBBB` (4 hex digits per channel). `pad()` produces lowercase hex.
- Terminator: `\x1b\` (ESC + backslash) — the C1_ESCAPED ST sequence. `EscapeSequences.ts:152` defines `C1_ESCAPED.ST = \`${C0.ESC}\\\``.
- **Alternate BEL terminator (`\x07`):** Per ECMA-48 and xterm spec, OSC envelopes may terminate with BEL instead of ST. xterm.js emits ST in its replies, but the spec allows both — the filter should accept both terminators to remain robust against any future xterm.js behavior change or hand-crafted clients.

**Concrete example bytes:**
```
1b 5d 31 30 3b 72 67 62 3a 63 63 63 63 2f 63 63 63 63 2f 63 63 63 63 1b 5c
ESC ]  1  0  ;  r  g  b  :  c  c  c  c  /  c  c  c  c  /  c  c  c  c  ESC \
```

### OSC 11 Reply (Background Color)

Same shape as OSC 10, only the code changes:

```
\x1b ] 11 ; rgb : RRRR / GGGG / BBBB \x1b \\
```

The bug report's `c4c4d4d4d4d4` leak is consistent with this format (the user's terminal has BG color `0xd4d4d4`, which serialized to 16-bit is `d4d4` per channel — 6 hex chars per channel pair × 3 channels = 12 chars; with the rgb: prefix and slashes that matches what they see in the prompt).

### DA1 Reply (CSI c — Primary Device Attributes)

```
\x1b [ ? <params> c
```

[VERIFIED: `InputHandler.ts:1667-1677`]

Per xterm.js v6.0.0:
- For TERM in `{xterm, rxvt-unicode, screen}` → emits `\x1b[?1;2c`. AgentHub sets `TERM=xterm-256color` (verified at `internal/pty/native.go:46`), which is recognized as xterm-family, so this is the actual emitted reply.
- For `TERM=linux` → emits `\x1b[?6c`.

The bug report's `62;4;9;22c` substring is a *different* DA1 reply shape — most likely from `chafa` running inside a different terminal entirely (vte / kitty / wezterm) when the user copied the example into the issue. xterm.js's v6.0.0 reply is always exactly `\x1b[?1;2c` for the AgentHub TERM setting.

**Filter rule:** The absorber must match `\x1b[?...c` for ANY parameter content — not just the literal `1;2`. Reasons:
- Future xterm.js versions may extend the reply.
- The filter must remain robust against the issue's documented shape (`62;4;9;22c`) even though we'd never emit it ourselves — if it appears in the input path it's a leaked reply, not a keystroke.
- A DA1 reply always starts with `\x1b[?` (note the `?` prefix is part of the response envelope, NOT the query — the query is plain `\x1b[c`). User keystrokes cannot legitimately produce `\x1b[?...c` because the leading `?` requires a DEC private-mode prefix that doesn't come from a normal keypress.

**Concrete example bytes (xterm.js v6.0.0):**
```
1b 5b 3f 31 3b 32 63
ESC [  ?  1  ;  2  c
```

### Cross-verification

[VERIFIED: WebSearch + xterm.js source] chafa specifically queries DA1 to detect sixel support (`'4'` in DA1 extensions → sixel-capable terminal). It also queries OSC 11 to determine the background color so it can choose a color palette that contrasts well. Both queries fire BEFORE the sixel/image bytes are sent — which is why the leaked responses land at the end of the chafa output (after the image, since xterm.js processes the reply after consuming the query).

## Streaming State Machine Design

### States

```
StateOutside           — default; bytes pass through verbatim until ESC
StateGotEsc            — saw \x1b, peeking the next byte to decide
StateInOSC             — inside \x1b]<code>;<body>; consuming until BEL or ESC
StateInOSCSeenEsc      — inside OSC, saw a nested ESC; next byte decides ST vs nested
StateInCSI             — inside \x1b[<params>; consuming until final byte (0x40-0x7E)
```

### Algorithm (single pass over input bytes)

For each byte `b` in `input`:

1. **`StateOutside`:**
   - If `b == 0x1b`: transition to `StateGotEsc`. **Do not yet emit** — we don't know if this is a passthrough ESC (e.g. arrow key `\x1b[A`) or an absorbed envelope. Remember the start index.
   - Else: emit `b` to output.

2. **`StateGotEsc`:**
   - If `b == ']'`: transition to `StateInOSC`. Begin collecting `oscBuf` (the body after `]`). Don't yet emit the `\x1b]`.
   - If `b == '['`: transition to `StateInCSI`. Begin collecting `csiBuf`. Don't yet emit the `\x1b[`.
   - Else: **passthrough** — emit `\x1b` followed by `b`. This handles arrow keys (`\x1b[A` after we've transitioned to StateInCSI) — wait, no, those are CSI. Handles SS3 (`\x1bO`), DEC SCS designations, and Alt+key sequences like `\x1ba` → emit both bytes verbatim. Return to `StateOutside`.

3. **`StateInOSC`:**
   - If `b == 0x07` (BEL): the OSC envelope is complete. Decide absorb-vs-passthrough by parsing `oscBuf`. See "OSC decision" below. Return to `StateOutside`. Clear `oscBuf`.
   - If `b == 0x1b`: transition to `StateInOSCSeenEsc` (might be ST or might be a nested ESC).
   - Else: append `b` to `oscBuf`. Continue.

4. **`StateInOSCSeenEsc`:**
   - If `b == '\\'` (backslash): the OSC envelope is complete (ST terminator). Decide absorb-vs-passthrough by parsing `oscBuf`. Return to `StateOutside`. Clear `oscBuf`.
   - Else: malformed — likely a nested escape. Treat conservatively: emit the buffered `\x1b]<oscBuf>\x1b` and the current `b` as passthrough; return to `StateOutside`. (This branch should be unreachable in practice from xterm.js but defends against malicious clients and protocol drift.)

5. **`StateInCSI`:**
   - If `b` is in `[0x40, 0x7e]` (the CSI final-byte range per ECMA-48): envelope complete. Decide absorb-vs-passthrough by parsing `csiBuf` + final byte `b`. See "CSI decision" below. Return to `StateOutside`. Clear `csiBuf`.
   - Else: append `b` to `csiBuf`. Continue.

### OSC Decision

Once an OSC envelope is complete (BEL or ST seen), parse `oscBuf`:

- Read characters up to the first `;` — that's the OSC code.
- If code is exactly `"10"` or `"11"`: **ABSORB.** Do not emit anything for the entire envelope. Log one debug event per absorption (with the code).
- Otherwise: **PASSTHROUGH.** Emit `\x1b]<oscBuf><terminator>` where `<terminator>` is whatever closed the envelope. This handles OSC 52 (clipboard), OSC 8 (hyperlinks), OSC 9;4 (progress), OSC 4 (palette query — explicitly out of scope), etc.

### CSI Decision

Once a CSI envelope is complete (final byte in `[0x40, 0x7e]`):

- If `csiBuf` starts with `?` AND the final byte is `c`: **ABSORB.** This is the DA1 reply shape `\x1b[?...c`. The `?` prefix is the discriminator that prevents false positives on user keystrokes — see "False-positive analysis" below.
- Otherwise: **PASSTHROUGH.** Emit `\x1b[<csiBuf><final>`. Critically, this includes:
  - Cursor movement: `\x1b[A` (up), `\x1b[B` (down), etc.
  - Function keys in app mode: `\x1b[15~`, `\x1b[1;5C` (Ctrl+Right), etc.
  - User typing `[?9c` literally: would emit `[`, `?`, `9`, `c` as separate `MsgInput` frames — each is a single character that never enters `StateInCSI` because there's no preceding `\x1b`.

### Cross-Frame Carry-Over

The state machine state IS the carry-over. Specifically:

- If a `MsgInput` payload ends mid-envelope (e.g. ends with `\x1b]11;rgb:cccc/c` — partial OSC), the state remains `StateInOSC` with `oscBuf = "11;rgb:cccc/c"`. The next `MsgInput` payload's bytes continue feeding into `oscBuf`.
- If a payload ends in `\x1b` alone, state is `StateGotEsc`. Next payload's first byte decides the transition.
- The `InputAbsorber` struct holds the state, `oscBuf`, and `csiBuf` — these persist across calls to `Filter()`.

**Bounded buffer guard:** xterm.js never emits OSC bodies longer than ~50 bytes for color replies, and DA1 replies are <20 bytes. Cap `oscBuf` and `csiBuf` at e.g. 4096 bytes each — if exceeded, flush as passthrough and reset to `StateOutside`. This bounds memory against a malicious client that opens an OSC envelope and never closes it.

### Pseudo-Code Skeleton

```go
// Source: hand-rolled, matches state machine above
func (a *InputAbsorber) Filter(in []byte) []byte {
    out := make([]byte, 0, len(in))
    for _, b := range in {
        switch a.state {
        case StateOutside:
            if b == 0x1b {
                a.state = StateGotEsc
            } else {
                out = append(out, b)
            }
        case StateGotEsc:
            switch b {
            case ']':
                a.state = StateInOSC
                a.oscBuf = a.oscBuf[:0]
            case '[':
                a.state = StateInCSI
                a.csiBuf = a.csiBuf[:0]
            default:
                // Passthrough: emit ESC + b, return to outside.
                out = append(out, 0x1b, b)
                a.state = StateOutside
            }
        case StateInOSC:
            switch b {
            case 0x07: // BEL terminator
                out = a.completeOSC(out)
            case 0x1b:
                a.state = StateInOSCSeenEsc
            default:
                a.oscBuf = append(a.oscBuf, b)
                if len(a.oscBuf) > maxEnvelopeBytes {
                    // Bail out — emit buffered + reset
                    out = append(out, 0x1b, ']')
                    out = append(out, a.oscBuf...)
                    a.oscBuf = a.oscBuf[:0]
                    a.state = StateOutside
                }
            }
        case StateInOSCSeenEsc:
            if b == '\\' {
                out = a.completeOSC(out)
            } else {
                // Malformed; flush conservatively as passthrough.
                out = append(out, 0x1b, ']')
                out = append(out, a.oscBuf...)
                out = append(out, 0x1b, b)
                a.oscBuf = a.oscBuf[:0]
                a.state = StateOutside
            }
        case StateInCSI:
            a.csiBuf = append(a.csiBuf, b)
            if b >= 0x40 && b <= 0x7e {
                out = a.completeCSI(out)
            } else if len(a.csiBuf) > maxEnvelopeBytes {
                out = append(out, 0x1b, '[')
                out = append(out, a.csiBuf...)
                a.csiBuf = a.csiBuf[:0]
                a.state = StateOutside
            }
        }
    }
    return out
}

func (a *InputAbsorber) completeOSC(out []byte) []byte {
    code, body, _ := bytes.Cut(a.oscBuf, []byte{';'})
    codeStr := string(code)
    if codeStr == "10" || codeStr == "11" {
        // Absorb — emit nothing.
        // TODO: slog.Debug("absorbed OSC", "code", codeStr, "body", string(body))
    } else {
        // Passthrough — reconstruct the envelope as ESC ] body BEL
        // (using BEL is fine — OSC handlers accept both terminators).
        out = append(out, 0x1b, ']')
        out = append(out, a.oscBuf...)
        out = append(out, 0x07)
    }
    a.oscBuf = a.oscBuf[:0]
    _ = body
    a.state = StateOutside
    return out
}

func (a *InputAbsorber) completeCSI(out []byte) []byte {
    // csiBuf includes the final byte
    if len(a.csiBuf) >= 2 && a.csiBuf[0] == '?' &&
       a.csiBuf[len(a.csiBuf)-1] == 'c' {
        // DA1 reply — absorb.
    } else {
        out = append(out, 0x1b, '[')
        out = append(out, a.csiBuf...)
    }
    a.csiBuf = a.csiBuf[:0]
    a.state = StateOutside
    return out
}
```

## False-Positive Analysis

The narrow filter (OSC code ∈ {10, 11}; CSI `?...c`) prevents most false positives by design. Specific cases:

### User Types `[?1c` Literally at the Prompt

xterm.js sends each printable keypress as a separate `MsgInput` frame OR batched as paste content. In either case:
- A literal `[` keystroke is byte `0x5b`, NOT preceded by `\x1b`. The state machine remains in `StateOutside` and emits `0x5b` verbatim.
- Same for `?`, `1`, `c` — each emitted verbatim.
- **Conclusion:** Safe. The leading `\x1b` is what distinguishes a real CSI envelope from typed text.

### User Pastes `\x1b[?1;2c` Into the Terminal

Possible via clipboard paste of binary content. xterm.js wraps paste in bracketed paste mode by default (`\x1b[200~...\x1b[201~`). The pasted bytes go through `term.onData` as a single string.

- If bracketed paste is enabled: the `\x1b[200~` prefix puts us into `StateInCSI` and gets absorbed-or-passed by the `~` final byte; the inner pasted ESC sequence is then evaluated by our absorber.
- A pasted `\x1b[?1;2c` would be absorbed as if it were a DA1 reply — **false positive**.

**Mitigation:** This is acceptable risk. Rationale:
- Pasting raw control sequences into a terminal is an unusual user action and would be expected to misbehave generally (most shells reject or warn).
- The legitimate use case (terminal capability probing) is exactly what we're absorbing — pasting it cannot be distinguished from xterm.js generating it.
- A theoretical client-side fix (the deferred option per CONTEXT) is the only way to distinguish "from xterm.js's internal reply machinery" vs "from user's clipboard" — the WS bridge has no signal for this.
- The same false-positive would affect OSC 10/11 paste — i.e., if a user pastes `\x1b]10;rgb:0000/0000/0000\x1b\\` it gets absorbed. Again, this is an edge case with no real-world impact.

### User Types Just `\x1b`

- E.g. ESC key pressed in vim/less to enter command mode.
- State machine: `StateOutside` → `StateGotEsc`. No bytes emitted YET. Next byte from the user (likely `:` or another keystroke) takes the default branch in `StateGotEsc` → emits `\x1b` followed by that byte. **Net effect:** identical to passthrough, just delayed by exactly one byte.
- **One edge case:** the ESC arrives in one `MsgInput` frame and the next byte arrives in the next frame. State is held across frames (per-subscriber). The next frame's first byte completes the resolution. **No corruption.**

### User Types Arrow Keys / Function Keys / Alt+Key

- Arrow up: `\x1b[A` — single byte sequence emitted as one `MsgInput`. `StateOutside`→`StateGotEsc`→`StateInCSI`→sees `A` (final byte 0x41, in range) → CSI complete. `csiBuf = "A"`. First byte `A` ≠ `?` AND final byte is `A` ≠ `c` → passthrough. Emits `\x1b[A`. **Safe.**
- Function key F5: `\x1b[15~` — same shape, final byte `~` (0x7e, in range). `csiBuf = "15~"`. Not `?`-prefixed → passthrough. Emits `\x1b[15~`. **Safe.**
- Ctrl+Right: `\x1b[1;5C` — final byte `C`. Not `?`-prefixed → passthrough. **Safe.**
- Alt+a: `\x1ba` — `StateGotEsc` sees `a` (not `]` not `[`) → default branch emits `\x1ba`. **Safe.**

### Programs That Emit Their Own DA1 Replies (Bash, Zsh, Vim)

No such case exists — DA1 is a terminal *capability*, not a shell capability. Only the terminal emulator (xterm.js) emits the reply. Programs *query* (`\x1b[c`) but never reply.

### Programs Pasting Color-Spec Strings Through stdin

E.g. a script does `echo -e "\x1b]10;rgb:0000/0000/0000\x1b\\\\" > /dev/tty`. This goes through PTY *output*, not input — it's xterm.js's job to PARSE that as a SET command and update the FG color. It does not flow through our input filter. **Safe.**

### Summary of False-Positive Risk

| Scenario | Outcome | Severity |
|----------|---------|----------|
| User types `[?1c` at prompt | Passthrough | None |
| User pastes raw `\x1b[?1;2c` | Absorbed (false positive) | Low — pasting control sequences is uncommon and the user would expect issues |
| User pastes raw `\x1b]10;rgb:.../...\x1b\\` | Absorbed (false positive) | Low — same as above |
| Arrow keys, function keys, Alt+key | Passthrough | None |
| ESC key alone (one byte at a time) | Delayed-passthrough (same net result) | None |
| OSC 52 clipboard, OSC 8 hyperlinks, OSC 9;4 progress | Passthrough (different OSC code) | None |
| Bracketed-paste markers `\x1b[200~`, `\x1b[201~` | Passthrough (CSI not `?`-prefixed or not `c`-terminated) | None |

## Test Scaffolding

### Existing Infrastructure (No Wave 0 Gaps)

The existing `internal/webserver/capability_test_helpers.go::testServerWithHub` (line 131) is the ideal harness:

```go
func testServerWithHub(t *testing.T, sessionID string) (
    *WebServer, *http.Client, *io.PipeWriter, *io.PipeReader,
)
```

It returns:
- An `*io.PipeWriter` (`ptyOutputW`) — write here to simulate PTY output (server-to-client direction). Not needed for this phase.
- An `*io.PipeReader` (`inputCaptureR`) — read here to observe bytes that reached the PTY's stdin pipe. **This is what we assert on.**

The pattern is already proven in `capability_test.go::TestSecurity_ReadOnlyCapabilityBlocksMsgInput` (line 130) which uses `readPipeMustTimeout(t, inputReader, 300*time.Millisecond, ...)` to assert that no bytes arrived — exactly the assertion we need for absorbed envelopes.

### Test Strategy

Two complementary layers:

**Layer 1: Pure unit tests over `InputAbsorber.Filter`** (`internal/webserver/oscabsorb_test.go`):

```go
func TestInputAbsorber_OSC10ReplyAbsorbed(t *testing.T) {
    a := &InputAbsorber{}
    in := []byte("\x1b]10;rgb:cccc/cccc/cccc\x1b\\")
    got := a.Filter(in)
    if len(got) != 0 {
        t.Errorf("expected empty output, got %q", got)
    }
}

func TestInputAbsorber_OSC10_BELTerminator(t *testing.T) { /* ... */ }
func TestInputAbsorber_OSC11ReplyAbsorbed(t *testing.T)  { /* ... */ }
func TestInputAbsorber_DA1ReplyAbsorbed(t *testing.T)    { /* ... */ }

func TestInputAbsorber_OSC52ClipboardPassthrough(t *testing.T) {
    a := &InputAbsorber{}
    in := []byte("\x1b]52;c;SGVsbG8=\x1b\\")
    got := a.Filter(in)
    if !bytes.Equal(got, in) {
        t.Errorf("expected passthrough for OSC 52, got %q", got)
    }
}

func TestInputAbsorber_ArrowKeysPassthrough(t *testing.T) { /* ... */ }
func TestInputAbsorber_KeystrokesPassthrough(t *testing.T) { /* a-z and \r \n */ }

func TestInputAbsorber_SplitAcrossFrames(t *testing.T) {
    a := &InputAbsorber{}
    // Split an OSC 11 reply across three calls:
    out1 := a.Filter([]byte("\x1b]11;rgb:"))
    out2 := a.Filter([]byte("cccc/cccc/c"))
    out3 := a.Filter([]byte("ccc\x1b\\"))
    if len(out1)+len(out2)+len(out3) != 0 {
        t.Errorf("expected zero bytes emitted across splits")
    }
}

func TestInputAbsorber_MixedInputAndReply(t *testing.T) {
    a := &InputAbsorber{}
    in := []byte("ls\r\x1b]11;rgb:cccc/cccc/cccc\x1b\\pwd\r")
    got := a.Filter(in)
    want := []byte("ls\rpwd\r")
    if !bytes.Equal(got, want) {
        t.Errorf("expected %q, got %q", want, got)
    }
}

func TestInputAbsorber_MaliciousUnclosedOSC(t *testing.T) {
    // Feed > maxEnvelopeBytes inside an OSC — verify flush-as-passthrough.
}
```

Table-driven with ~12-15 cases covers: each shape, both terminators, splits at every boundary, mixed traffic, false-positive cases.

**Layer 2: Integration test through the WebSocket** (`internal/webserver/oscabsorb_integration_test.go` or appended to an existing test file):

```go
func TestRelay_OSCRepliesAbsorbedBeforePTY(t *testing.T) {
    ws, client, _, inputReader := testServerWithHub(t, "sess-osc")
    ws.SetSigningKey(capTestKey)
    token := issueCapFor(t, ws, "sess-osc", "read,write")

    conn := dialWebServerWS(t, client, ws.BaseURL(),
        "/sessions/sess-osc/ws?cap="+token, originHeader(ws))

    // Send an OSC 11 reply as MsgInput.
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    payload := []byte("\x1b]11;rgb:cccc/cccc/cccc\x1b\\")
    if err := conn.Write(ctx, websocket.MessageBinary,
        relay.MakeInputFrame(payload)); err != nil {
        t.Fatalf("write: %v", err)
    }

    // Assert nothing reaches the PTY pipe.
    readPipeMustTimeout(t, inputReader, 300*time.Millisecond,
        "OSC 11 reply must be absorbed by the relay")
}

func TestRelay_KeystrokesStillForwarded(t *testing.T) {
    // Send "hello\r" — assert it reaches the PTY pipe.
}

func TestRelay_SplitFrameAbsorption(t *testing.T) {
    // Two writes — first ends mid-envelope, second completes it.
    // Assert no bytes reach PTY pipe across both writes.
}
```

**No Wave 0 gaps.** All harness pieces exist:
- `testServerWithHub` provides the server + PTY pipe.
- `readPipeMustTimeout` provides the "nothing arrives" assertion.
- `readPipeWithTimeout` provides the "something arrives" assertion (for the regression-doesn't-break-keystrokes case).
- `dialWebServerWS`, `issueCapFor`, `relay.MakeInputFrame` are all in place.

## macOS Web UAT Setup

### Path to Reproduce the Bug Locally

1. **Build agenthub** for the local machine (macOS):
   ```bash
   wails build -tags wailsassets
   # OR for CLI-only smoke:
   go build -o bin/agenthub .
   ```

2. **Start the daemon** (if not already running):
   ```bash
   ./bin/agenthub daemon run   # foreground, for log visibility
   # OR
   ./build/bin/agenthub.app/Contents/MacOS/agenthub  # GUI auto-starts daemon
   ```

3. **Start the local web server:**
   ```bash
   ./bin/agenthub web start
   # Prints: web server URL: https://127.0.0.1:<port>  (Tailscale or local-LAN mode)
   ```
   The `cmd_cli.go:cmdWebStart` flow (line 244) calls daemon API → `api.go:780` which creates a `webserver.NewWebServer` in either Tailscale or local mode.

4. **Create a shell session** (or use existing):
   ```bash
   ./bin/agenthub new       # creates a session, prints session ID
   ./bin/agenthub list      # confirm session ID
   ```

5. **Enable web-share for that session:**
   ```bash
   ./bin/agenthub serve <session-id>
   # Prints the shareable URL with capability token: https://...?cap=...
   ```

6. **Open the URL in Chrome** (NOT Safari first — Chrome DevTools is more useful for UAT capture). The URL points to `terminal.html` which loads `web/assets/terminal.js`.

7. **Reproduce the bug** in the browser terminal:
   ```bash
   # Inside the shared shell session:
   curl -fsSLo /tmp/test.png \
     https://upload.wikimedia.org/wikipedia/commons/thumb/0/0c/GoldenGateBridge-001.jpg/120px-GoldenGateBridge-001.jpg
   chafa --format=sixel /tmp/test.png
   # Before fix: image renders, prompt has "11;rgb:..." or "62;4;9;22c" garbage at start
   # After fix: image renders, prompt is clean
   ```

   Alternative — if chafa isn't installed:
   ```bash
   brew install chafa
   # Or fall back to manual probe verification:
   printf '\033]11;?\033\\'   # Triggers OSC 11 query; should NOT leave garbage on next prompt
   printf '\033[c'            # Triggers DA1 query; same
   ```

8. **Cross-surface parity check** — open the same session in the desktop GUI (Wails) by clicking the session in the AgentHub app window. Run the same `chafa` command. Compare prompts.

### Notes

- **No Tailscale required.** `agenthub web start` works in local-LAN mode on macOS; the web server binds to the LAN IP (see `internal/daemon/process.go:134` for the local-mode fallback path). Self-signed TLS — Chrome will show a security warning; click "Advanced" → "Proceed anyway."
- **Per `project_wails_devtools_disabled_in_prod` memory:** if you need DevTools-level inspection of what xterm.js actually sends, use the regular Chrome path (not Wails desktop), since the prod Wails build has DevTools disabled. The web-share URL is exactly that path.
- **Per `user_colorblind` memory:** verify the "leak" by reading the prompt text in a screen capture or via the test harness — don't rely on visual color difference between the leak and the clean prompt. The leak is plain ASCII text like `11;rgb:cccccccccccc` which is unambiguous in plain reading.
- **macOS-only constraint:** chafa via Homebrew works fine. ImageMagick / curl / a sample PNG suffice. No iPad or external hardware needed for Phase 111 UAT (unlike Phase 113).

## Common Pitfalls

### Pitfall 1: Forgetting cross-frame state

**What goes wrong:** A test passes the entire OSC envelope in one call but the real WS path delivers it in two frames. Bug reproduces in production but not in unit tests.
**Why it happens:** Easy to write a "happy path" test where the whole envelope arrives at once.
**How to avoid:** The unit test suite MUST include split-at-every-boundary cases (`TestInputAbsorber_SplitAcrossFrames` and variants splitting at each byte position).
**Warning signs:** Unit tests pass, but UAT still shows leaks.

### Pitfall 2: Filtering at the wrong layer

**What goes wrong:** Filter inserted inside `hub.WriteInput` — affects desktop relay (CLI/TUI `agenthub attach`) too, breaking legitimate input paths.
**Why it happens:** `hub.WriteInput` looks like a tempting choke point.
**How to avoid:** Filter ONLY in `internal/webserver/server.go::handleWSSRelay`'s read pump, right before the existing `_ = hub.WriteInput(payload)` call. Per CONTEXT, desktop relay is out of scope.
**Warning signs:** TUI / CLI shell sessions stop accepting input or behave oddly.

### Pitfall 3: Shared absorber state across subscribers

**What goes wrong:** Two browser tabs viewing the same session share an absorber → tab A's mid-OSC bytes get mixed with tab B's keystrokes → both tabs corrupt each other's input.
**Why it happens:** Putting the absorber on the hub or the `WebServer` struct instead of per-subscriber.
**How to avoid:** Instantiate `absorber := &InputAbsorber{}` as a local in `handleWSSRelay`, captured by the goroutine closure. Lives exactly as long as the WS connection. Multiple subscribers → multiple absorbers.
**Warning signs:** Multi-tab UAT shows keystroke loss or duplicate absorption events in logs.

### Pitfall 4: Absorbing legitimate OSC sequences

**What goes wrong:** A naive "absorb all OSC" implementation swallows OSC 52 (clipboard), OSC 8 (hyperlinks in paste), OSC 9;4 (progress).
**Why it happens:** Short-circuiting parse of the OSC code.
**How to avoid:** ALWAYS parse the OSC code (the digits before the first `;`) and absorb ONLY when code == `"10"` OR code == `"11"`. Default action is passthrough.
**Warning signs:** OSC 52 paste-to-clipboard fails on web after fix; vim/less progress bars break.

### Pitfall 5: Wrong CSI discriminator

**What goes wrong:** Filter absorbs all `\x1b[...c` envelopes, including any hypothetical legitimate user-typed CSI ending in `c`.
**Why it happens:** Dropping the `?`-prefix check.
**How to avoid:** Require BOTH `csiBuf[0] == '?'` AND `final == 'c'` for absorption. A bare `\x1b[c` (the QUERY shape, not the REPLY) flowing through input would mean a program is querying via xterm.js's onData — which doesn't happen in practice — but if it did, leaving it as passthrough is harmless.
**Warning signs:** Some terminal control sequence stops working after fix.

### Pitfall 6: Logging cost

**What goes wrong:** Per-byte `slog.Debug` calls in the hot loop tank performance under load (multiple subscribers, paste of 10 KB).
**Why it happens:** Easy to add log calls "just for visibility."
**How to avoid:** Log ONCE per completed absorption (inside `completeOSC` / `completeCSI`). Use `slog.Debug`, not `Info`. Include only the code or the matched envelope.
**Warning signs:** Paste latency increases noticeably; CPU profile shows logging dominating.

### Pitfall 7: Unbounded buffer growth

**What goes wrong:** A malicious or buggy client opens `\x1b]10;` and never sends a terminator → `oscBuf` grows unbounded → memory exhaustion.
**Why it happens:** Trusting the protocol.
**How to avoid:** Cap `oscBuf` and `csiBuf` at e.g. 4 KiB. On overflow, flush the buffered bytes as passthrough and reset to `StateOutside`. Pragmatic: legitimate OSC bodies are <100 bytes; the cap is far beyond any real case.
**Warning signs:** Memory growth on idle WS connections.

## Code Examples

### Pattern: Where the Filter Wires In

```go
// Source: internal/webserver/server.go:739-767 (existing code — modification below)

// Read pump — client → PTY
readDone := make(chan struct{})
absorber := &InputAbsorber{} // NEW — per-subscriber state
go func() {
    defer close(readDone)
    for {
        _, msg, err := conn.Read(ctx)
        if err != nil {
            return
        }
        msgType, payload, err := relay.ParseFrame(msg)
        if err != nil {
            continue
        }
        switch msgType {
        case relay.MsgInput:
            if !sub.ReadOnly { // MC-03: discard input for read-only clients
                filtered := absorber.Filter(payload)  // NEW
                if len(filtered) > 0 {                // NEW — skip the empty-after-absorb case
                    _ = hub.WriteInput(filtered)      // CHANGED: pass filtered instead of payload
                }
            }
        case relay.MsgResize2: /* ... */
        case relay.MsgPing:    /* ... */
        }
    }
}()
```

Total edit: ~4 lines added in `server.go`. New file `oscabsorb.go` ~80 lines.

### Pattern: Existing Test Harness in Action

```go
// Source: internal/webserver/capability_test.go:130-151 (existing test — pattern to follow)
func TestSecurity_ReadOnlyCapabilityBlocksMsgInput(t *testing.T) {
    ws, client, _, inputReader := testServerWithHub(t, "sess-block")
    ws.SetSigningKey(capTestKey)
    token := issueCapFor(t, ws, "sess-block", "read")

    headers := http.Header{}
    headers.Set("Origin", ws.BaseURL())
    conn := dialWebServerWS(t, client, ws.BaseURL(),
        "/sessions/sess-block/ws?cap="+token, headers)

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    if err := conn.Write(ctx, websocket.MessageBinary,
        relay.MakeInputFrame([]byte("should-be-blocked\n"))); err != nil {
        t.Fatalf("write input frame: %v", err)
    }

    readPipeMustTimeout(t, inputReader, 300*time.Millisecond,
        "read cap must block MsgInput at relay")
}
```

Phase 111 tests follow this exact shape — swap the payload for an OSC envelope and the cap for `"read,write"`, keep the `readPipeMustTimeout` assertion.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `nhooyr/websocket` | `coder/websocket` | Renamed in 2024 | Already migrated; `go.mod` reflects `coder/websocket v1.8.14` |
| Hand-rolled input filtering at server level | Same (this phase) | New addition | No "current best practice" library exists for ANSI escape filtering on a relay — it's bespoke per use case |

**Deprecated/outdated:** Nothing in this phase's scope changes pre-existing patterns. The fix is purely additive.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | "Desktop is unaffected by this bug" (per CONTEXT) | Open Questions | If desktop is also affected, the Wails GUI UAT will fail post-fix, exposing that `internal/relay/server.go` needs the same absorber wired in. Mitigation: when running the cross-surface UAT, capture the desktop result *before* applying the fix to verify the asymmetry exists; if it doesn't, escalate scope. |
| A2 | xterm.js v6.0.0 uses BEL or ST terminators interchangeably for OSC replies in future versions too | Three Response Shapes | xterm.js source confirms current behavior uses ST. If a future version switches to BEL, our filter still handles it (we accept both). Robust by design. |
| A3 | `TERM=xterm-256color` causes xterm.js to emit `\x1b[?1;2c` for DA1 (not `\x1b[?6c`) | Three Response Shapes / DA1 | Verified at `InputHandler.ts:1671` via the `_is('xterm')` branch. Project sets `TERM=xterm-256color` at `internal/pty/native.go:46`. xterm-256color matches the xterm-family branch. |
| A4 | OSC 52 / OSC 8 / OSC 9;4 are the only OSC codes legitimately flowing browser→PTY in current AgentHub usage | False-Positive Analysis | If a new feature in v3.4+ uses an OSC code in the absorbed set (10 or 11), it would be silently swallowed. Low risk — OSC 10/11 are spec-reserved for the color query/reply protocol; reusing them is a spec violation. |
| A5 | The `maxEnvelopeBytes = 4096` cap is safe — no legitimate OSC body or CSI param string approaches this size | Algorithm | xterm.js never emits anywhere near this. Even pathological OSC 52 clipboard pastes are typically <2 KiB. Cap is forgiving. |
| A6 | Per-subscriber absorber instance is the right scope (vs. shared) | Architecture | High confidence: multiple browser tabs of the same session have independent xterm.js instances each tracking their own pending queries. Sharing state would corrupt across tabs. |

## Open Questions

1. **Is the desktop Wails surface actually unaffected by this bug?**
   - **What we know:** CONTEXT.md says desktop is unaffected ("Wails webview goes through xterm.js directly and consumes those responses cleanly"). The bug report (Issue #54) was filed against the web surface specifically.
   - **What's unclear:** Code-level analysis shows that BOTH surfaces use `term.onData((data) => ws.send(makeInputFrame(data)))` with identical semantics, and both server-side paths (`internal/webserver/server.go:755` and `internal/relay/server.go:127`) forward to `hub.WriteInput(payload)` with no filtering. If the bug is real on web, it should be real on desktop. The "desktop unaffected" claim appears to be empirical observation rather than code-determined.
   - **Recommendation:** During cross-surface UAT (WEB-02 release gate), explicitly verify the desktop *before* applying the fix. If desktop also exhibits the leak, escalate to the planner — the fix needs to be applied symmetrically in `internal/relay/server.go:127` too, and the CONTEXT "out of scope" line about desktop relay needs to be revisited. If desktop is genuinely clean despite identical code, document the mechanism (perhaps a theme/timing difference makes it not noticeable, or perhaps `enableSizeReports: false` has a side effect we missed).

2. **Should the absorber also strip BPM (Bracketed Paste Mode) markers if they wrap an absorbed envelope?**
   - **What we know:** xterm.js wraps paste content in `\x1b[200~...\x1b[201~`. The state machine treats `\x1b[200~` as a CSI (final byte `~`, no `?` prefix) → passthrough.
   - **What's unclear:** If a user pastes content that contains an OSC 11 reply (theoretical), the wrapper passes through and the inner absorption still works correctly — the user's paste is mangled but their pasted control sequence is what's malformed, not our handling. **Conclusion:** Acceptable. No change needed.

3. **Should we add a feature flag to disable the absorber?**
   - **What we know:** If the absorber breaks something in production we have no escape valve.
   - **What's unclear:** Probably overkill for a small filter. The risk surface is tightly scoped (browser→PTY direction only, narrow filter, well-tested), and a `git revert` is the appropriate rollback. Recommend NO feature flag — keep YAGNI.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Build + test | ✓ | 1.22+ (verified via go.mod toolchain) | — |
| `coder/websocket` | WS relay code | ✓ | v1.8.14 (vendored / module-pinned) | — |
| `wails` | Desktop build for cross-surface UAT | ✓ (assumed via existing build flow) | per go.mod | Could skip desktop UAT if Wails unavailable, but per memory `project_wails_build_requires_tags`, `-tags wailsassets` build is the canonical desktop test path |
| `chafa` | UAT reproduction (sixel rendering with terminal probing) | Possibly missing on dev machine | — | `brew install chafa` on macOS; or use raw probe: `printf '\033[c'` and `printf '\033]11;?\033\\'` |
| `gh` CLI | Linking the fix back to Issue #54 | ✓ (per memory `reference_github_issues_release_planning`) | — | — |
| A sample PNG | chafa input | Trivially obtainable | — | `curl` or `imagemagick convert` to generate |
| Chrome | Web UAT | ✓ (standard macOS dev tool) | — | Safari works for UAT-pass too, but Chrome's DevTools network/protocol inspector is more useful for diagnosis |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** `chafa` — install via Homebrew if not present, or use raw printf to inject queries directly.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` stdlib (no third-party test runner) |
| Config file | None — Go convention, all test files alongside source |
| Quick run command | `go test -run InputAbsorber -count=1 ./internal/webserver/` (unit tests only, sub-second) |
| Full suite command | `go test -race -count=1 ./internal/webserver/...` (full webserver package, includes integration tests) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| WEB-01 | OSC 10 reply absorbed; nothing reaches PTY pipe | unit | `go test -run TestInputAbsorber_OSC10ReplyAbsorbed ./internal/webserver/` | ❌ Wave 0 — `internal/webserver/oscabsorb_test.go` |
| WEB-01 | OSC 11 reply absorbed | unit | `go test -run TestInputAbsorber_OSC11ReplyAbsorbed ./internal/webserver/` | ❌ Wave 0 |
| WEB-01 | DA1 reply absorbed | unit | `go test -run TestInputAbsorber_DA1ReplyAbsorbed ./internal/webserver/` | ❌ Wave 0 |
| WEB-01 | Split-across-frames absorption | unit | `go test -run TestInputAbsorber_SplitAcrossFrames ./internal/webserver/` | ❌ Wave 0 |
| WEB-01 | Keystrokes & arrow keys & OSC 52 passthrough preserved | unit | `go test -run TestInputAbsorber_Passthrough ./internal/webserver/` | ❌ Wave 0 |
| WEB-01 | End-to-end through WS: OSC 11 reply doesn't reach PTY | integration | `go test -run TestRelay_OSCRepliesAbsorbedBeforePTY ./internal/webserver/` | ❌ Wave 0 |
| WEB-01 | End-to-end through WS: keystrokes DO reach PTY (regression guard) | integration | `go test -run TestRelay_KeystrokesStillForwarded ./internal/webserver/` | ❌ Wave 0 |
| WEB-02 | Cross-surface chafa parity on macOS | manual | UAT-only: web Chrome + desktop Wails | manual UAT step (see "macOS web UAT setup") |
| WEB-03 | Future regression in absorption fails CI | unit + integration | All of the above run in `go test ./...` CI | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test -run InputAbsorber -count=1 ./internal/webserver/` (sub-second)
- **Per wave merge:** `go test -race -count=1 ./internal/webserver/...`
- **Phase gate:** `go test -race -count=1 ./...` (whole-tree green) before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/webserver/oscabsorb.go` — the `InputAbsorber` type and `Filter([]byte) []byte` method.
- [ ] `internal/webserver/oscabsorb_test.go` — unit tests (table-driven, ~15 cases covering all three shapes, both terminators, split-at-every-boundary, false-positive scenarios).
- [ ] Integration test(s) — either appended to `internal/webserver/capability_test.go` or new `internal/webserver/oscabsorb_integration_test.go`. Reuses `testServerWithHub` + `readPipeMustTimeout` from existing helpers. No new harness needed.
- No framework install needed (Go stdlib testing).

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no (this phase) | Cap-token auth already in place per Phase 87 (`requireCapability`) — unchanged |
| V3 Session Management | no | WS session already cap-bound — unchanged |
| V4 Access Control | partial | `sub.ReadOnly` check already gates input; absorber runs only when write is permitted. No new access-control surface. |
| V5 Input Validation | yes | The absorber IS an input-validation layer. Must handle malformed envelopes gracefully (covered by Pitfall 7 — unbounded growth) and not introduce parser bugs (table-driven tests cover this). |
| V6 Cryptography | no | No crypto in scope |

### Known Threat Patterns for `internal/webserver` (Go + coder/websocket)

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Resource exhaustion via unclosed OSC envelope | Denial of Service | Bounded buffer (`maxEnvelopeBytes` cap with flush-as-passthrough on overflow). See Pitfall 7. |
| State pollution across subscribers | Tampering | Per-subscriber `InputAbsorber` instance — local to read pump goroutine. See Pitfall 3. |
| Injection of unintended sequences via passthrough | Tampering | Default is passthrough — no transformation of bytes we don't recognize. Reconstruction in `completeOSC` passthrough branch uses the original `oscBuf` byte-for-byte. |
| Bypass via reconnect | Spoofing | N/A — absorber state is per-connection; a reconnect spawns a fresh absorber. No state to bypass. |
| False positive eating user keystroke | Functional (not security) | Strict discriminators (`?` prefix + `c` final byte for CSI; exact `10`/`11` code match for OSC). Verified false-positive cases all map to "user types literal control sequence at prompt" — see False-Positive Analysis. |

## Project Constraints (from CLAUDE.md)

- **Go conventions:** `go fmt`, `golangci-lint`, context-aware functions. The new `InputAbsorber` does not need a context (pure byte transformation; no I/O). The `Filter` method should be safe to call from the read-pump goroutine without additional locking — it's accessed by exactly one goroutine.
- **Testing:** Go `testing` stdlib, 80%+ coverage in critical components. Table-driven tests for the absorber. Aim for 100% coverage of `oscabsorb.go` — the file is small enough that this is trivial.
- **Make beliefs pay rent:** Section "Open Questions" item 1 (desktop-unaffected claim) is the explicit prediction-vs-verification gate for this phase.
- **Chesterton's Fence:** The existing `_ = hub.WriteInput(payload)` line at `server.go:755` is unchanged in its semantics for non-absorbed bytes; the fence remains intact. The new code adds a filtering shim, doesn't remove anything.
- **Silent fallbacks:** Per CLAUDE.md "Let it crash." The absorber's flush-on-overflow path is NOT a silent fallback — it loudly logs and passes bytes through as passthrough (which is the conservative behavior; better than dropping them).
- **Cross-surface parity is release-blocking** (per memory `feedback_cross_surface_parity`): web + desktop chafa output must match. This is encoded as WEB-02 and is the phase release gate.
- **GitHub issues drive release planning** (per memory `reference_github_issues_release_planning`): Phase 111 closes Issue #54. The fix commit message should reference `#54`.

## Sources

### Primary (HIGH confidence)

- **xterm.js v6.0.0 source** (vendored in repo) — `frontend/node_modules/.pnpm/@xterm+xterm@6.0.0/node_modules/@xterm/xterm/`:
  - `src/common/InputHandler.ts:1667-1677` — DA1 reply emission
  - `src/common/InputHandler.ts:3050-3107` — OSC 10/11 query handling
  - `src/common/input/XParseColor.ts:77-79` — `toRgbString` format
  - `src/common/data/EscapeSequences.ts:66, 152` — C0.ESC and C1_ESCAPED.ST constants
  - `src/browser/CoreBrowserTerminal.ts:213-218` — color report → triggerDataEvent
- **AgentHub codebase** — verified:
  - `internal/webserver/server.go:664-784` (handleWSSRelay, read pump)
  - `internal/relay/protocol.go` (frame format)
  - `internal/relay/hub.go:195-199` (WriteInput)
  - `internal/relay/server.go:111-139` (parallel desktop relay)
  - `internal/webserver/capability_test_helpers.go:131-170` (testServerWithHub)
  - `internal/webserver/capability_test.go:130-151` (existing input-block test pattern)
  - `internal/pty/native.go:46` (TERM=xterm-256color)
  - `web/assets/terminal.js:968-973` (web term.onData handler)
  - `frontend/src/components/TerminalPanel.tsx:283` (desktop term.onData handler)
  - `frontend/src/lib/relayClient.ts:119-123` (sendInput → encodeInputFrame)
- **Project planning docs**:
  - `.planning/REQUIREMENTS.md` (WEB-01..03 definitions)
  - `.planning/milestones/v3.3.1-ROADMAP.md:75-93` (Phase 111 charter)
  - `.planning/phases/111-web-bridge-osc-da-response-consumption/111-CONTEXT.md` (locked decisions)

### Secondary (MEDIUM confidence)

- [chafa(1) man page on Arch Manual Pages](https://man.archlinux.org/man/extra/chafa/chafa.1.en) — terminal-probing behavior, sixel detection via DA1.
- [Jexer Sixel Tests](https://jexer.sourceforge.io/sixel.html) — DA1 extension code 4 → sixel support, OSC 11 query usage by lsix.
- ECMA-48 / xterm spec on OSC and CSI envelope shapes — universally documented but not freshly fetched in this session; the embedded xterm.js source is the more authoritative source for what AgentHub's browser actually emits.

### Tertiary (LOW confidence)

- None. All claims in this research are backed by either codebase verification or vendored library source.

## Metadata

**Confidence breakdown:**
- Current web-bridge data flow: **HIGH** — verified end-to-end in code, every line:column cited.
- Three response shapes: **HIGH** — verified against xterm.js v6.0.0 vendored source, byte-for-byte.
- Streaming state machine design: **HIGH** — straightforward hand-rolled algorithm, false-positive analysis exhaustive within scope.
- False-positive analysis: **MEDIUM-HIGH** — the seven enumerated scenarios cover all realistic input shapes; pasted control sequences are the only acknowledged false-positive surface, and accepted by design.
- Test scaffolding: **HIGH** — existing harness (`testServerWithHub`, `readPipeMustTimeout`) is a direct fit, already proven in `capability_test.go`.
- macOS UAT setup: **HIGH** — CLI command flow verified in `cmd_cli.go`.
- Desktop-unaffected claim: **LOW** — the asymmetry is empirically asserted in CONTEXT but not explained by code-level analysis. Flagged as Open Question #1.

**Research date:** 2026-05-18
**Valid until:** 2026-06-17 (30 days — Go ecosystem and webserver code are stable; xterm.js v6 is current major)
