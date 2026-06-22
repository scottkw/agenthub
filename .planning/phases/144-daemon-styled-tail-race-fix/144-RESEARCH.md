# Phase 144: Daemon Styled-Tail Race Fix - Research

**Researched:** 2026-06-21
**Domain:** Go concurrency / data races; charmbracelet/x/vt headless terminal emulation; ANSI escape-sequence stripping
**Confidence:** HIGH

## Summary

`SessionEngine.GetSessionStyledTailLines` (`internal/daemon/engine.go:620-700`) fails `go test -race`
on all platforms. The race is **structural**: the function spawns a drain goroutine that calls
`emu.Read()` (via `io.Copy(io.Discard, emu)`) concurrently with `emu.Close()` on the main goroutine.
Inside `charmbracelet/x/vt`, `Read()` reads an unsynchronized `closed bool` field
(`emulator.go:252`) while `Close()` writes it (`emulator.go:265`) — no mutex, no atomic. The race
detector flags every styled-tail test. [VERIFIED: reproduced via `go test -race`, goroutine stacks
captured — see Reproduction below]

The drain goroutine exists for a reason: it was added in Phase 139 to fix the #96 headless-VT pipe
deadlock. `emu.Write()` of scrollback containing terminal-query sequences (DA, DSR, DECRQM, OSC
color queries) blocks forever, because the emulator writes query *responses* into an unbuffered
`io.Pipe` (`Emulator.pw`) and `pw.Write` blocks until a reader drains it. The fix drained the pipe
in a goroutine, then closed the emulator to unblock and stop it — which is exactly what created the
concurrent Read/Close.

**Primary recommendation:** Implement **Option 2** from issue #100 — strip the query-eliciting
escape sequences from the scrollback *before* `emu.Write`, then remove the drain goroutine entirely
and run a fully synchronous `Write` -> `Close` -> read path. I verified empirically that once the
queries are stripped, `emu.Write` returns synchronously with **no goroutine at all**, so the
concurrent Read/Close — and therefore the race — becomes structurally impossible. Option 1 (bump
vt) is **dead**: I confirmed the newest available vt pseudo-version (`20260621010513`, published
today) still has an unsynchronized `closed` field, identical Read/Close code. [VERIFIED: diff of
both module versions]

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FIX-01 | `GetSessionStyledTailLines` passes `go test -race` on all platforms — no data race between the styled-tail drain goroutine and the VT emulator close (#100) | Root cause located in vt `emulator.go:252/265`; Option 2 (pre-Write query strip + remove goroutine) proven to eliminate the concurrent Read/Close empirically; exact strip byte-patterns enumerated from vt source; regression coverage identified |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Styled-tail render (scrollback -> styled grid) | API / Daemon (`internal/daemon`) | — | Pure Go server-side transform; no client involvement. The race and fix live entirely in `engine.go`. |
| Query-sequence stripping | API / Daemon | — | Must happen before bytes reach the headless emulator; daemon owns the emulator lifecycle. |

## Standard Stack

No new dependencies. This is a self-contained bug fix in existing daemon code.

### Core (already in go.mod, no change)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/charmbracelet/x/vt | v0.0.0-20260615092313-b57e5e6d29bb | Headless VT emulator for styled-cell render | Already adopted Phase 139; the bug is in *how we drive it*, not the lib choice |
| stdlib `regexp` | go1.26 | Strip query escape sequences pre-Write | Already imported in engine.go (line 12) |

### Do NOT bump vt
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Stripping queries (Option 2) | Bump `charmbracelet/x/vt` (Option 1) | **REJECTED** — newest pseudo-version `20260621010513-945fab64fd3e` still has unsynchronized `closed`; bump does not fix the race and risks unrelated behavior drift. [VERIFIED: diff] |
| Stripping queries (Option 2) | Vendor a Read/Close-serializing wrapper (Option 3) | More code, more surface to maintain; does not remove the underlying need for a concurrent drain. Option 2 removes the goroutine outright. |

**Installation:** None. No `go get`. The only go.mod-adjacent note: **do not** run `go get -u`
on vt as part of this phase.

## Package Legitimacy Audit

No external packages installed in this phase. All dependencies are already in `go.mod` and unchanged.
Audit: **N/A — no new packages.**

## Architecture Patterns

### Current (broken) flow

```
hub.ScrollbackSnapshot()  ──► strip 0x01 framing bytes ──► raw ANSI bytes
                                                                │
                              ┌─────────────────────────────────┤
                              │ (main goroutine)                 │ (drain goroutine, Phase 139)
                              ▼                                   ▼
                        emu.Write(stripped) ──blocks on──► io.Copy(io.Discard, emu)  ── emu.Read()
                        emu.Close() ◄──unblocks drain──────────────┘     ▲
                              │                                            │
                              └────── RACE: Close writes `closed` ◄───────┘  Read reads `closed`
                              ▼
                        emu.CellAt(x,y) loop ──► [][]StyledSpan
```

### Recommended (fixed) flow — Option 2

```
hub.ScrollbackSnapshot()  ──► strip 0x01 framing bytes ──► stripQuerySequences() ──► query-free ANSI bytes
                                                                                          │
                                                            (single goroutine, no drain)  ▼
                                                                            emu.Write(clean)   ◄─ returns synchronously
                                                                            emu.Close()        ◄─ no concurrent Read
                                                                                          │
                                                                                          ▼
                                                                            emu.CellAt(x,y) loop ──► [][]StyledSpan
```

### Pattern: Strip-then-synchronous-drive

**What:** Remove every input sequence that elicits a response into `Emulator.pw`, so `emu.Write`
never blocks waiting for a reader. Then drive the emulator on a single goroutine with no
concurrency at all.

**When to use:** Any headless use of `charmbracelet/x/vt` where you only want the rendered grid and
never intend to feed responses back (i.e., there is no real terminal application reading the
emulator's input pipe).

**Why it removes the race:** The `closed` field is only racy because `Read` and `Close` run on
different goroutines. With no drain goroutine, `Read` is never called concurrently with `Close`.
[VERIFIED: experiment (b) below — fully synchronous Write->Close->read with no goroutine renders
correctly and passes `-race` logic.]

### Anti-Patterns to Avoid

- **Bumping vt to "the latest" hoping it's fixed:** It is not. Confirmed identical unsynchronized
  `closed` in `20260621010513`. [VERIFIED]
- **Keeping the goroutine but adding a `sync.WaitGroup`/channel ordering trick:** Ordering does not
  help — the detector flags the *unsynchronized memory access*, not a happens-before bug you can fix
  from outside the library. The only external fix that satisfies `-race` is to never call Read
  concurrently with Close.
- **Stripping with the existing `ansiEscape` regex (engine.go:546):** That regex is used by the
  *plain-text* `GetSessionTailLines` and removes ALL CSI/OSC — it would destroy color/style. The
  styled path must keep SGR/color and strip ONLY the query-eliciting subset. Use a *separate*,
  narrow regex.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parsing every ANSI sequence to find queries | A full ANSI state machine | A narrow `regexp` matching exactly the documented query verbs | The set of response-eliciting sequences is small and fully enumerated from vt source (below) |
| Synchronizing vt's `closed` field | A vendored fork / wrapper mutex | Remove the concurrency (Option 2) | Less code, no fork to maintain, upstream-agnostic |

**Key insight:** The fix is *subtractive* — remove a goroutine and a handful of bytes — not additive.

## Runtime State Inventory

This is a code-only fix. No stored data, live service config, OS-registered state, secrets, or
build artifacts are affected.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — verified by code inspection (operates on in-memory scrollback only) | none |
| Live service config | None — no external service config references this path | none |
| OS-registered state | None | none |
| Secrets/env vars | None | none |
| Build artifacts | None — pure source change in `internal/daemon/engine.go` | none |

## Common Pitfalls

### Pitfall 1: Forgetting in-band-resize (DEC mode 2048)
**What goes wrong:** Stripping only DA/DSR/DECRQM/OSC-color leaves `ESC[?2048h` (set DEC mode 2048,
in-band resize) in the stream. Setting that mode triggers a pw write of `ansi.InBandResize(...)`
(`vt/csi_mode.go:95`), which blocks `emu.Write` again — re-introducing the #96 deadlock for any
scrollback that enables it.
**Why it happens:** It's a *mode set*, not an obvious "query," so it's easy to miss. Modern TUIs
(Claude Code et al.) may enable it.
**How to avoid:** Include `ESC[?2048h` / `ESC[?2048l` in the strip set. [VERIFIED: experiment showed
`\x1b[?2048h` hangs without stripping, no-hang with it stripped.]
**Warning signs:** A `TimeAfter` test like `QueryNoHang` hanging on a fixture that contains `?2048h`.

### Pitfall 2: Removing the goroutine but leaving the `io` import
**What goes wrong:** `io` is used in `engine.go` ONLY by the drain goroutine
(`io.Copy(io.Discard, emu)` at line 652). Removing the goroutine makes `io` an unused import → Go
compile error.
**Why it happens:** Easy to delete the goroutine and forget the import block.
**How to avoid:** Remove `"io"` from the import block (engine.go line 8) as part of the change.
[VERIFIED: `grep "io\."` shows the only live use is line 652.]
**Warning signs:** `go build ./...` fails with `"io" imported and not used`.

### Pitfall 3: Over-stripping and killing color/style
**What goes wrong:** Using a too-greedy regex (e.g., the existing `ansiEscape`) strips SGR color
codes too; `TestGetSessionStyledTailLines_ColorBold` then fails (no green, no bold).
**How to avoid:** Strip ONLY the enumerated query verbs (see Code Examples). [VERIFIED: experiment
confirmed `\x1b[1;32m` survives the narrow strip; cell renders fg=2, bold attr set.]

### Pitfall 4: Local `go test` masks the race
**What goes wrong:** `go test ./internal/daemon/` (no `-race`) passes — this is exactly how the bug
hid on the dev laptop until CI ran it. [CITED: issue #100]
**How to avoid:** Always validate with `-race`. CI uses `-race -short` (`build.yml:55,57`).

## Code Examples

### Reproduction (exact race location)
```
go test -race ./internal/daemon/ -run 'TestGetSessionStyledTailLines'
```
Captured goroutine stacks (HIGH confidence — run this session):
```
Read at ... by goroutine 18:
  vt.(*Emulator).Read()    emulator.go:252   (if e.closed)
  io.Copy()                                  <- engine.go:652 drain goroutine
  daemon.GetSessionStyledTailLines.func1()   engine.go:652
Previous write at ... by goroutine 8:
  vt.(*Emulator).Close()   emulator.go:265   (e.closed = true)
  daemon.GetSessionStyledTailLines()         engine.go:656
```

### The complete set of query-eliciting sequences (enumerated from vt source)
Every input that causes a blocking write into `Emulator.pw`. [VERIFIED: grep of all `e.pw` writes
in vt `handlers.go`, `csi.go`, `csi_mode.go`, `osc.go` + handler dispatch tables]

| Sequence | Name | vt handler | Response written |
|----------|------|-----------|------------------|
| `ESC [ c` / `ESC [ 0 c` | DA1 (Primary Device Attributes) | handlers.go:687 | PrimaryDeviceAttributes |
| `ESC [ > c` | DA2 (Secondary Device Attributes) | handlers.go:703 | SecondaryDeviceAttributes |
| `ESC [ 5 n` | DSR Operating Status | handlers.go:806 | DeviceStatusReport |
| `ESC [ 6 n` | DSR Cursor Position (CPR) | handlers.go:809 | CursorPositionReport |
| `ESC [ ? 6 n` | DECXCPR Extended Cursor Position | handlers.go:826 | ExtendedCursorPositionReport |
| `ESC [ ... $ p` | DECRQM Request Mode (ANSI) | csi.go:30 via handlers.go:834 | ReportMode |
| `ESC [ ? ... $ p` | DECRQM Request Mode (DEC) | csi.go:30 via handlers.go:840 | ReportMode |
| `ESC ] 10;? BEL/ST` | OSC query foreground color | osc.go:87 | SetForegroundColor (only if a color is set) |
| `ESC ] 11;? BEL/ST` | OSC query background color | osc.go:92 | SetBackgroundColor (only if a color is set) |
| `ESC ] 12;? BEL/ST` | OSC query cursor color | osc.go:97 | SetCursorColor (only if a color is set) |
| `ESC [ ? 2048 h` | DEC mode 2048 SET (in-band resize) | csi_mode.go:95 | InBandResize (fires on *enable*) |

Sequences that write to `pw` but are NOT triggered by `Write()` of scrollback (driven only by
explicit emulator API calls `Focus()`/`Blur()`/`Paste()`/mouse — not used by the styled-tail path):
focus.go:24/26, mouse.go:111/113, emulator.go:306/307/310/315 (Paste/InputPipe). These need no
stripping. [VERIFIED: these are method bodies, not CSI handlers reachable from `parser.Advance`.]

### Recommended strip helper (sketch — verified to compile and behave)
```go
// queryStripPattern matches the terminal-query and in-band-resize sequences that
// elicit a blocking response write into the headless emulator's response pipe
// (charmbracelet/x/vt Emulator.pw). Stripping these BEFORE emu.Write lets Write
// run synchronously with no drain goroutine. SGR/color sequences are deliberately
// preserved so the styled grid keeps its colors.
//
// Covered: DA1 (ESC[c), DA2 (ESC[>c), DSR (ESC[5n/6n), DECXCPR (ESC[?6n),
// DECRQM (ESC[...$p, ESC[?...$p), OSC color queries (ESC]10|11|12;?),
// DEC mode 2048 set/reset (ESC[?2048h/l, in-band resize).
var queryStripPattern = regexp.MustCompile(
    `\x1b\[[0-9;]*c` +              // DA1 (and any params before c)
        `|\x1b\[>[0-9;]*c` +        // DA2
        `|\x1b\[[0-9;]*n` +         // DSR (5n/6n and any CSI n)
        `|\x1b\[\?[0-9;]*n` +       // DECXCPR (ESC[?...n)
        `|\x1b\[[0-9;]*\$p` +       // DECRQM ANSI
        `|\x1b\[\?[0-9;]*\$p` +     // DECRQM DEC
        `|\x1b\[\?2048[hl]` +       // in-band resize set/reset (set triggers a pw write)
        `|\x1b\]1[012];\?(?:\x07|\x1b\\)`, // OSC 10/11/12 color query, BEL- or ST-terminated
)
```
[VERIFIED: this exact pattern, applied before a synchronous `emu.Write`/`emu.Close`, eliminated all
hangs in an 8-case experiment (DA1+DSR, DA2, DSR5, DECXCPR, DECRQM ansi+dec, OSC color, mode-2048,
kitchen-sink) while preserving `\x1b[1;32mgreen\x1b[0m` (cell rendered fg=2, bold attr set) and the
`aaaa\rbbbb` CR-overwrite collapse.]

### Recommended engine.go change (file-level sketch)
Replace the goroutine block at `engine.go:642-657` with:
```go
    // Strip terminal-query and in-band-resize sequences that would block
    // emu.Write on the emulator's unbuffered response pipe (#96). With these
    // removed, Write returns synchronously and no drain goroutine is needed —
    // which also removes the concurrent Read/Close data race (#100).
    clean := queryStripPattern.ReplaceAll(stripped, nil)
    emu.Write(clean) //nolint:errcheck // emulator Write never returns a meaningful error
    _ = emu.Close()
```
Then remove the now-unused `"io"` import (engine.go:8). The `CellAt` extraction loop (658-700) is
unchanged. **Net change:** -1 goroutine, -1 channel, -1 import, +1 package-level regex, +1
ReplaceAll call.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Drain emu.Read in goroutine + Close to unblock (Phase 139, #96 fix) | Strip queries pre-Write + synchronous drive (Phase 144, #100 fix) | This phase | Removes concurrency entirely; keeps #96 fixed without a goroutine |

**Deprecated/outdated:** Option 1 (vt version bump) — not viable; newest vt still unsynchronized.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Claude Code / supported TUIs may emit `ESC[?2048h` into scrollback, making mode-2048 stripping load-bearing in production (not just theoretically) | Pitfall 1 | If never emitted, stripping it is harmless dead-safe coverage. Low risk — stripping is conservative and verified harmless to rendering. |
| A2 | No production scrollback contains a query verb that is *also* visible content we must keep (e.g., a literal `ESC[c` we'd want to render) | Code Examples | Near-zero: these are control sequences, never user-visible glyphs. The plain-text path already discards all CSI. |

If both assumptions are wrong, the fix is still correct for the race; only the breadth of the strip
set would be debated. The race elimination does not depend on A1/A2.

## Open Questions

1. **Should mode-2048 stripping be in scope?**
   - What we know: `ESC[?2048h` provably hangs `emu.Write` without a drain (experiment).
   - What's unclear: whether current production scrollback actually contains it.
   - Recommendation: Include it. It is a real blocking write, costs one regex alternative, and is
     verified harmless to rendering. Excluding it would re-open #96 for any session that enables
     in-band resize.

2. **Add a focused regression fixture?**
   - Recommendation: Yes — extend/keep `TestGetSessionStyledTailLines_QueryNoHang` and add a
     query-heavy "kitchen-sink" fixture (all verbs + `?2048h`) so the strip set can't silently
     regress. See Validation Architecture.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build + race detector | ✓ | go1.26.4 darwin/arm64 | — |
| race detector (`-race`) | reproduce + verify fix | ✓ | bundled with Go on cgo-enabled platforms | — |
| charmbracelet/x/vt | styled render | ✓ (in module cache) | v0.0.0-20260615092313-b57e5e6d29bb | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` |
| Config file | none (go.mod-driven) |
| Quick run command | `go test -race ./internal/daemon/ -run 'StyledTail'` |
| Full suite command | `go test -race -short ./internal/...` (mirrors CI `build.yml:55`) |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FIX-01 | No data race in styled-tail render | unit (race) | `go test -race ./internal/daemon/ -run 'TestGetSessionStyledTailLines\|TestHandleGetSessionStyledTailLines'` | ✅ (4 tests already exist) |
| FIX-01 | #96 not regressed: queries don't hang | unit (timeout) | `go test -race ./internal/daemon/ -run 'TestGetSessionStyledTailLines_QueryNoHang'` | ✅ |
| FIX-01 | Color/bold preserved | unit | `go test -race ./internal/daemon/ -run 'TestGetSessionStyledTailLines_ColorBold'` | ✅ |
| FIX-01 | CR-overwrite collapse preserved (#96 rendering) | unit | `go test -race ./internal/daemon/ -run 'TestGetSessionStyledTailLines_TUI'` | ✅ |
| FIX-01 | All query verbs + mode-2048 don't hang (new coverage) | unit (timeout) | new test, e.g. `TestGetSessionStyledTailLines_AllQueriesNoHang` | ❌ Wave 0 (recommended) |

### Sampling Rate
- **Per task commit:** `go test -race ./internal/daemon/ -run 'StyledTail'`
- **Per wave merge:** `go test -race -short ./internal/daemon/`
- **Phase gate:** `go test -race -short ./internal/...` green (matches CI non-Windows path), plus
  `go build ./...` (catches the unused-`io`-import pitfall).

### Wave 0 Gaps
- [ ] (recommended) Add `TestGetSessionStyledTailLines_AllQueriesNoHang` in
      `internal/daemon/engine_test.go` — a query-heavy fixture covering DA1/DA2/DSR5/DSR6/DECXCPR/
      DECRQM(ansi+dec)/OSC-color/`?2048h`, asserting no hang (5s timeout) and that surrounding
      visible text survives. Guards the strip set against future regression.
- [ ] No framework install needed — Go `testing` is in-tree.

*Existing test infrastructure already covers the 4 issue-named cases; the only gap is the
broadened query fixture above.*

### CI consideration
CI runs `-race -short` across the four-job matrix (macOS, two Ubuntu variants, Windows;
`build.yml:49-57`). Windows runs `./internal/...` only. The fix must be green on all four. The fix
is platform-agnostic (pure Go + regex), so a local `-race` pass is strong evidence for all jobs.

## Security Domain

`security_enforcement` not present in `.planning/config.json` (treated as enabled), but this phase
is a concurrency/correctness bug fix on an in-memory render path with no auth, network, crypto,
input-validation, or access-control surface introduced or modified.

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | marginal | Scrollback bytes are already untrusted terminal output; stripping control sequences is *defensive* (reduces emulator attack surface), not a new validation requirement |
| All others (V2/V3/V4/V6) | no | No auth/session/access-control/crypto touched |

No new threat patterns. The change strictly *reduces* the set of escape sequences fed to the
emulator.

## Project Constraints (from CLAUDE.md)

- **Go conventions:** `go fmt`, `golangci-lint`, context-aware functions. Existing `//nolint`
  pragmas on the Write line should be preserved as appropriate.
- **Regression Test Convention (repo CLAUDE.md + TESTING.md Section 6 — STANDING RULE):** If a new
  test file/test is added (e.g., the recommended `AllQueriesNoHang`), TESTING.md must be updated:
  Section 2 (Suite Manifest) if a new file, Section 4 (Traceability map) for any test covering a
  v4.0 requirement (FIX-01), and run `bash tests/check-traceability-paths.sh` before committing.
  Since FIX-01 is covered by existing `internal/daemon/engine_test.go` (already row CARD-05 path),
  the traceability path likely already exists; add a FIX-01 row pointing at the same file path.
- **No global package installs / no dependency changes** — this phase adds none.

## Sources

### Primary (HIGH confidence)
- `charmbracelet/x/vt@v0.0.0-20260615092313-b57e5e6d29bb` source — `emulator.go` (Read:251-257,
  Close:260-267, pipe:102), `handlers.go` (DA1/DA2/DSR/DECXCPR/DECRQM 687-842), `csi.go`
  (handleRequestMode 18-37), `csi_mode.go` (InBandResize 95), `osc.go` (color queries 60-100) —
  direct file inspection in module cache.
- `charmbracelet/x/vt@v0.0.0-20260621010513-945fab64fd3e` (latest) — diffed `emulator.go`
  Read/Close vs old version: IDENTICAL, no sync added. Confirms Option 1 dead.
- `internal/daemon/engine.go:609-700` — the function under fix.
- `internal/daemon/engine_test.go:1770-1916` — the 4 styled-tail tests.
- `.github/workflows/build.yml:49-57` — CI race command + matrix.
- Live experiments (this session) — 3 standalone Go programs proving: (1) strip lets Write run
  synchronously with no goroutine; (2) no-strip hangs; (3) mode-2048 must be stripped; color/CR
  rendering preserved.
- `go test -race ./internal/daemon/` — reproduced the race with exact goroutine stacks.

### Secondary (MEDIUM confidence)
- GitHub issue #100 (`gh issue view 100`) — canonical spec, root-cause hypothesis (confirmed by
  primary sources above).

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Root cause: HIGH — reproduced with exact file:line goroutine stacks matching the issue.
- Fix viability (Option 2): HIGH — empirically proven that stripping enables a no-goroutine
  synchronous path; race becomes structurally impossible.
- Option 1 rejection: HIGH — diffed latest vt, unsynchronized `closed` confirmed present.
- Strip-set completeness: HIGH — enumerated every `e.pw` writer in vt and verified the
  scrollback-reachable subset; experiment exercised all of them including the easily-missed
  mode-2048 case.
- Regression safety: HIGH — color/bold and CR-overwrite rendering verified intact post-strip.

**Research date:** 2026-06-21
**Valid until:** 2026-07-21 (stable; vt is the only moving part and its relevant code is unchanged
across the two latest pseudo-versions)
