---
status: resolved
trigger: "status dot on tabs stays blue (running) and never transitions to green (idle) when a CLI session returns to its prompt"
created: 2026-03-18T00:00:00Z
updated: 2026-03-18T22:00:00Z
---

## Current Focus

hypothesis: confirmed — two independent bugs prevent the idle transition
test: code inspection + regex analysis
expecting: N/A — root cause confirmed
next_action: apply fixes

## Symptoms

expected: tab dot turns green when Claude Code returns to its prompt (❯)
actual: dot stays blue (running) indefinitely
errors: none visible — silent failure
reproduction: start a session, wait for Claude Code to complete a task and show its prompt
started: after the suffix-check fix was committed

## Eliminated

- hypothesis: Watch goroutine not starting
  evidence: app.go:148 launches `go status.Watch(hub, id, cli, ...)` correctly; hub is a real *relay.Hub which satisfies HubLike
  timestamp: 2026-03-18

- hypothesis: context guard blocking EventsEmit
  evidence: `a.ctx.Value("frontend") != nil` is the correct production guard (same pattern as beforeClose); tests pass nil-context which avoids the panic
  timestamp: 2026-03-18

- hypothesis: race condition in Watch subscription
  evidence: Watch calls hub.Subscribe before ScrollbackSnapshot, so no frames are missed; channel is buffered at 256
  timestamp: 2026-03-18

## Evidence

- timestamp: 2026-03-18
  checked: internal/relay/hub.go — broadcast() and MakeOutputFrame()
  found: every frame broadcasted to subscribers is a FRAMED byte slice: `[0x01, ...raw PTY bytes...]`
  implication: the first byte is always MsgOutput (0x01), not part of the actual terminal data

- timestamp: 2026-03-18
  checked: internal/status/detector.go — Feed() receives frames from sub.Msgs
  found: Feed() passes the raw frame directly to StripANSI then into the tail — it does NOT strip the leading 0x01 protocol byte
  implication: the tail contains binary framing noise at the start of every chunk; the trailing bytes of a chunk end with real terminal data only when the PTY output itself ends there — BUT the rolling tail sees `0x01` prefixed garbage which can corrupt suffix matching

- timestamp: 2026-03-18
  checked: internal/status/detector.go — classify(), Idle regex `❯\s*$`
  found: the regex uses `$` (end-of-string / end-of-line in Go's RE2). In Go's regexp package, `$` matches end-of-text OR immediately before a `\n` at end-of-text. Claude Code's PTY prompt output is `❯ ` (with a trailing space, no newline). The regex `❯\s*$` SHOULD match `❯ ` — the space satisfies `\s*` and `$` matches end-of-text.
  implication: the regex itself is correct for clean text — but see the frame-prefix bug above

- timestamp: 2026-03-18
  checked: internal/relay/hub.go — Run() → broadcast(frame) vs detector Feed()
  found: CONFIRMED BUG 1: Watch() receives framed messages (MsgOutput byte + payload) but Feed() treats the entire frame as raw PTY output. The 0x01 byte is fed into StripANSI which passes it through (it's not an ANSI sequence), so it accumulates in the tail.
  implication: the tail is polluted with 0x01 bytes at every chunk boundary. This is mostly harmless for Working/Waiting pattern matching but can disrupt the suffix-based Idle check if a 0x01 byte appears at the very end of the tail (e.g., an empty-payload frame or a frame boundary coinciding with the tail suffix cutoff)

- timestamp: 2026-03-18
  checked: internal/relay/hub.go ScrollbackSnapshot() vs Watch() Feed(snap)
  found: CONFIRMED BUG 2 (the primary bug): ScrollbackSnapshot() returns the raw scrollback buffer, which was populated via `s.buf = append(s.buf, frame...)` in Append(). The scrollback stores FRAMED data (0x01-prefixed). Watch() feeds this snapshot directly to detector.Feed() as if it were raw PTY bytes. The detector's tail is therefore built from framed scrollback data.
  implication: the suffix of the tail likely ends with 0x01 bytes from frame boundaries, not with the actual prompt characters. `❯\s*$` cannot match when the last byte is 0x01.

- timestamp: 2026-03-18
  checked: internal/relay/hub.go — Run() read loop
  found: frames are 32KiB read chunks, so each broadcast frame is typically large. The 0x01 prefix byte appears once per read() call — not overwhelming, but it IS in the data and grows with the tail.
  implication: over a long session, the accumulated 0x01 bytes in the tail corrupt the suffix check reliably

## Resolution

root_cause: |
  Two layered bugs prevent the idle transition:

  PRIMARY (root): The relay protocol wraps every PTY chunk in a framed message
  (MsgOutput = 0x01 prefix byte, protocol.go:MakeOutputFrame). The Hub's
  scrollback and broadcast frames both carry this prefix. status.Watch() feeds
  these framed messages directly into Detector.Feed() without stripping the
  protocol byte. As a result, the detector's rolling tail is built from
  FRAMED data, not raw PTY text.

  For the live-stream path: each `sub.Msgs` frame received in the Watch select
  loop starts with 0x01 and must have payload extracted via ParseFrame before
  being fed to the detector.

  For the scrollback path: ScrollbackSnapshot() returns framed bytes (the
  scrollback Append() stores the full frame including the 0x01 prefix). The
  snapshot must be decoded frame-by-frame before feeding.

  Because of this, the rolling tail's suffix ends with binary noise (the 0x01
  byte at the start of whatever the next frame would be, or from partial tail
  truncation), not with the `❯ ` prompt characters. The regex `❯\s*$` never
  matches the corrupted suffix.

  SECONDARY: The Idle regex `❯\s*$` is correctly written for clean text but
  Claude Code's actual prompt may include an OSC/DCS escape that the current
  reANSI regex does not strip. The existing StripANSI handles CSI sequences
  (`\x1b[...`) and a few others, but NOT OSC sequences (`\x1b]...ST` or
  `\x1b]...\x07`). If Claude Code emits `\x1b]0;window title\x07❯ `, the OSC
  bytes would survive StripANSI and appear before the `❯`, breaking the suffix
  match. This is a secondary risk to verify after fixing the primary bug.

fix: |
  In internal/status/detector.go Watch():

  1. Live-stream path: after receiving a frame from sub.Msgs, call
     relay.ParseFrame() and skip non-MsgOutput frames. Only feed the payload
     (frame[1:]) to detector.Feed().

  2. Scrollback path: the scrollback snapshot is a concatenation of raw framed
     bytes. Iterate through it with ParseFrame (scanning for 0x01-prefixed
     chunks) and feed only the payload bytes.

  Additionally, expand reANSI in detector.go to cover OSC sequences:
     `\x1b\][^\x07]*(?:\x07|\x1b\\)` — this handles both BEL-terminated and
     ST-terminated OSC sequences that terminals emit for window/tab titles.

verification: pending
files_changed:
  - internal/status/detector.go  # Watch() frame stripping + StripANSI OSC coverage
