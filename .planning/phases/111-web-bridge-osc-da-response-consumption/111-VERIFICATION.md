# Phase 111 Verification

**Closes:** GitHub Issue #54. Requirements WEB-01..03.

**Executor host:** macOS — Go-level tests run locally; cross-surface
chafa UAT (web vs. desktop) is `human_needed` because it requires
launching the GUI Wails app, a Chrome browser session against a
self-signed-cert dev share URL, and visual evaluation of a sixel
rendering's trailing prompt (per project memory `user_colorblind`
the assessment is text-based, not color-based).

## Cross-surface UAT (WEB-02 release gate)

| Item | Requirement | Surface | Status | Reproduction | Owner |
|------|-------------|---------|--------|--------------|-------|
| Web chafa sixel clean prompt | WEB-01, WEB-02 | Web (Chrome on localhost) | human_needed | `./bin/agenthub daemon run &`; `./bin/agenthub web start`; `./bin/agenthub new bash $HOME` (note SID); `./bin/agenthub serve <SID>` (note share URL); open share URL in Chrome (accept self-signed cert); run `curl -fsSLo /tmp/test.png https://upload.wikimedia.org/wikipedia/commons/thumb/0/0c/GoldenGateBridge-001.jpg/120px-GoldenGateBridge-001.jpg && chafa --format=sixel /tmp/test.png`. **PASS criterion:** the prompt that follows the image is clean — no `10;rgb:...`, `11;rgb:...`, `?1;2c`, or `62;4;9;22c` text injected before the prompt char. Screenshot to `.planning/phases/111-.../uat-evidence/web-chafa.png`. | macOS dev box |
| Web OSC/DA probe (sensitive) | WEB-01 | Web (Chrome) | human_needed | In the same web terminal: `printf '\033]11;?\033\\'; printf '\033[c'; echo ZZZ`. **PASS:** next line shows only `ZZZ` with nothing before it. **FAIL:** junk before `ZZZ`. | macOS dev box |
| Desktop chafa sixel clean prompt | WEB-02 | macOS desktop (Wails GUI) | human_needed | Build/launch the Wails app (`./build/bin/agenthub.app/Contents/MacOS/agenthub`, or `wails build -tags wailsassets` first per project memory `project_wails_build_requires_tags`). Attach to the same session SID. Repeat the chafa command + the OSC/DA probe. Save `.planning/phases/111-.../uat-evidence/desktop-chafa.png`. | macOS dev box |
| Parity decision | WEB-02 | both | human_needed | Compare the two screenshots. Both must be clean. If desktop ALSO leaks, **do NOT expand scope** — file a follow-up GitHub issue ("Desktop relay also leaks OSC/DA1 replies (follow-up to #54)") and record `approved with desktop follow-up: #<n>` in the resume signal. | macOS dev box |
| Regression smoke (web) | WEB-01 | Web (Chrome) | human_needed | In the web terminal: type `ls`, Enter (output renders). Press Up / Down arrows (history works). Type a word, Backspace mid-word (edit works). Confirm no regression. | macOS dev box |
| Regression smoke (desktop) | WEB-01 | macOS desktop | human_needed | Same in the desktop terminal tab. | macOS dev box |

## Automated tests

| Item | Requirement | Command | Status | Notes |
|------|-------------|---------|--------|-------|
| TestInputAbsorber_* unit suite | WEB-03 | `go test ./internal/webserver -race -count=1 -run 'TestInputAbsorber'` | auto PASS | 26 named subtests across 9 top-level `TestInputAbsorber_*` functions (≥15 required). Covers absorb (OSC 10/11 with both ST + BEL terminators, DA1 single + multi-param), passthrough (keystrokes, arrow keys, F-keys, Alt-key, OSC 52, OSC 8, bracketed-paste), cross-frame splits at every byte boundary (S1–S5), mixed traffic (M1), bare-ESC delayed passthrough (P9), and robustness (R1 overflow, R2 malformed). Executor 2026-05-18 — PASS, 1.04 s. |
| TestRelay_* integration suite | WEB-01, WEB-03 | `go test ./internal/webserver -race -count=3 -run 'TestRelay_OSC\|TestRelay_KeystrokesStillForwarded\|TestRelay_MixedReplyAndKeystrokes'` | auto PASS | 6 integration tests through `testServerWithHub` + `dialWebServerWS` + `readPipeMustTimeout`/`readPipeWithTimeout`: OSC 10 / OSC 11 / DA1 absorbed before PTY; keystrokes still forwarded; OSC 11 split across two WS frames absorbed in full (proves per-subscriber state survives MsgInput frame boundaries); mixed reply+keystrokes yields exactly `"ls\rpwd\r"` at the pipe. Executor 2026-05-18 — PASS under -race -count=3, 4.18 s. |
| Full webserver suite under race | WEB-03 | `go test ./internal/webserver/... -race -count=1` | auto PASS | Executor 2026-05-18 — PASS, 4.18 s. No new flakes introduced. |
| go vet (webserver) | WEB-03 | `go vet ./internal/webserver/...` | auto PASS | Clean. |
| gofmt (new files) | WEB-03 | `gofmt -l internal/webserver/oscabsorb.go internal/webserver/oscabsorb_test.go internal/webserver/oscabsorb_relay_test.go internal/webserver/server.go` | auto PASS | All four files gofmt-clean. |

## Static checks

| Check | Status | Notes |
|-------|--------|-------|
| `internal/webserver/oscabsorb.go` exists with `type InputAbsorber struct` and `func (a *InputAbsorber) Filter(in []byte) []byte` | auto PASS | Confirmed. |
| `internal/webserver/oscabsorb_test.go` defines `TestInputAbsorber*` with ≥15 named subtests | auto PASS | 26 subtests in 9 test functions. |
| `internal/webserver/oscabsorb_relay_test.go` defines ≥6 `TestRelay_*` integration tests | auto PASS | 6 functions. |
| `grep -nE 'absorber\s*:=\s*&InputAbsorber\{\}\|filtered\s*:=\s*absorber\.Filter' internal/webserver/server.go` shows exactly two matches | auto PASS | Lines 741 and 756. |
| `git diff main -- internal/webserver/server.go \| grep -c '^+'` shows ≤6 net added lines | auto PASS | 5 added, 1 removed (net +4). |
| `git diff main -- internal/relay/` is empty | auto PASS | Patch-release scope honored. |
| `git diff main -- go.mod go.sum` is empty | auto PASS | No new dependencies. |
| Source-line count of `oscabsorb.go` (non-blank, non-comment) ≤120 | auto PASS | 117 source lines. |
| `slog.Debug` used for absorption logging (≤1 per absorbed envelope; never per byte) | auto PASS | Two call sites — `completeOSC` (when code is 10 or 11) and `completeCSI` (when DA1). |

## Sign-off gate

All `human_needed` items in the cross-surface UAT table must be recorded
as PASS (with the macOS operator's signature line below) before this phase
is marked complete in ROADMAP.md. Resolution of Open Question 1 (desktop
empirical state) must be recorded in `111-01-SUMMARY.md`.

## macOS operator sign-off

_Pending — to be filled in by the macOS dev-box operator after the
cross-surface UAT runs. Required:_

- _Date and commit SHA (latest on `main` at time of sign-off)_
- _Output of the chafa command in the web terminal (text capture of the
  prompt that follows the image)_
- _Output of the chafa command in the desktop terminal (same)_
- _Output of the OSC/DA probe in both terminals (the `ZZZ` line)_
- _Two screenshots committed under `uat-evidence/`: `web-chafa.png` and
  `desktop-chafa.png`_
- _Resume signal — one of:_
  - _`approved` (both clean; Open Question 1 resolved: desktop empirically clean)_
  - _`approved with desktop follow-up: #<issue-number>` (web clean, desktop also leaks → follow-up issue filed against `internal/relay/server.go`)_
  - _`failed: <description>` (revert Task 2 wiring and re-investigate per RESEARCH §Pitfalls 1–7)_

---

**Date:** _____________________
**Commit SHA:** _____________________
**macOS operator:** _____________________
**Confirmation:** _____________________
