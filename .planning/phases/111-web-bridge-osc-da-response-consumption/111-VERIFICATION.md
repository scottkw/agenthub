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

**Web surface UAT — PASS (2026-05-18)**

Driven via headless Chromium (Playwright 1.59.1, `ignoreHTTPSErrors`,
basic-auth wrapped). Two probes captured under `uat-evidence/`:

- **`web-chafa.png`** — `chafa --format=sixel /tmp/web02-test.png && echo
  MARKER_DONE_<ns>` against the Golden Gate test image. Sixel image
  renders top-left; `MARKER_DONE_663347000` shows cleanly on the line
  below; the prompt that follows is the shell glyphs only — no
  `10;rgb:...`, `11;rgb:...`, `?1;2c`, or `62;4;9;22c` text injected.
- **`web-osc-probe.png`** — sensitive probe:
  `printf '\x1b]11;?\x1b\\'; printf '\x1b[c'; echo ZZZ_MARKER`. Result:
  only `ZZZ_MARKER` on the next line, then a clean prompt. The OSC 11
  BG-color and DA1 replies that xterm.js emits in response to those
  queries were absorbed by `InputAbsorber` before reaching the PTY
  stdin — exactly the WEB-01 contract.

WEB-01 and WEB-02 web-side **PASS** at commit `7441475` (HEAD of `main`
at sign-off time). Webserver was running in `local` mode with
`Password=web02test` for the cap-URL handshake; the absorber path is
mode-agnostic (lives in `handleWSSRelay`, both local and tailscale
modes share it).

**Desktop surface UAT — DEFERRED**

The Wails GUI requires a desktop session this headless macOS thread
cannot drive. RESEARCH Open Question 1 (does desktop also leak?) is
left **unresolved by automation**. Path forward:

- Manual desktop UAT on this same macOS box: launch the Wails app
  (`wails dev` or a `-tags wailsassets` build), attach to a shell
  session, repeat the two probes, save `desktop-chafa.png` +
  `desktop-osc-probe.png`.
- If desktop is clean → resolution: "desktop empirically unaffected,
  CONTEXT D-LOCKED holds, no follow-up needed."
- If desktop ALSO leaks → **do NOT expand Phase 111 scope**. File
  follow-up issue "Desktop relay also leaks OSC/DA1 replies
  (follow-up to #54)" against `internal/relay/server.go` per the
  patch-release boundary.

**Resume signal:** `approved-web-only` (web absorption verified;
desktop UAT pending operator).

---

**Date:** 2026-05-18
**Commit SHA:** 7441475
**macOS operator:** ken@kscott (headless Playwright UAT)
**Confirmation:** Web surface PASS — both `web-chafa.png` and
`web-osc-probe.png` show clean prompts after probes that would have
leaked `10;rgb:`, `11;rgb:`, `?1;2c`, `62;4;9;22c` on `main`-before-fix.
Desktop surface UAT deferred to manual operator run.
