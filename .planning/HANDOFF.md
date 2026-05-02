---
created: 2026-05-02
session_topic: v3.1 milestone UAT
status: paused_mid_phase_90_preflight
last_commit: 6dd5c5b
---

# Handoff — v3.1 UAT Continuation

## What was completed this session

**Phase 89 (Vendored Terminal Assets + CSP) — DONE.** All 3 human UAT items PASS, committed as `6dd5c5b`. VERIFICATION.md status flipped `human_needed` → `resolved`. Verification debt for Phase 89: 0.

- UAT-1 Safari (Tailscale): PASS — only noise was 3 source-map 404s
- UAT-2 LAN-fallback: PASS in Chrome — 0 CSP violations across 1779 requests, READ ONLY badge enforced
- UAT-3 Network audit: PASS — 0 third-party CDN requests over 5.5-min session. Bonus: the 12× style-src-elem violations called out in the doc preamble are GONE under the D-09 amendment

## Phase 89 follow-up findings (filed in 89-HUMAN-UAT.md, none blocking)

1. Sessions panel doesn't show LAN Basic Auth password (only Settings does)
2. Safari rejects self-signed cert exceptions for subresources in LAN-fallback mode
3. AgentHub native session pane stays blank when web-enabled in LAN mode (Phase 87 native render bug, not Phase 89 scope)
4. UAT-2 doc references `agenthub start` — not a real command. Correct: `open -a AgentHub` or `agenthub` no-arg

Plus: launching AgentHub via Finder/`open` causes share URLs not to appear. Launching via terminal works. Likely env-var inheritance issue. Phase 87 follow-up.

## Phase 90 pre-flight — HALTED at step 4

Reached step 4 of `90-06-HUMAN-UAT.md`. Found three blockers before any RC tag can be pushed:

### Blocker 1 — 121 unpushed commits on local main

`git status -sb` shows `## main...origin/main [ahead 121]`. All Phase 87 + 89 + 90 work is local-only since 2026-04-19. Last successful build.yml run on origin: `cde1cec2` (4-19). Until HEAD is pushed and build.yml goes green on origin, the RC tag would fire release.yml against code GitHub has never matrix-built.

### Blocker 2 — Working tree drift

```
M frontend/src/wailsjs/wailsjs/runtime/package.json
M frontend/src/wailsjs/wailsjs/runtime/runtime.d.ts
M frontend/src/wailsjs/wailsjs/runtime/runtime.js
M go.mod
M go.sum
```

Likely artifacts from the local wails build (we ran `bash build.sh --platform macos` to get a fresh AgentHub.app for UAT). Need to investigate before pushing — either commit (if real) or revert (if reproducible from upstream).

### Blocker 3 — `release` env has zero protection rules

`gh api /repos/scottkw/agenthub/environments/release` returned `protection_rules: []`, `deployment_branch_policy: None`. The moment v3.1.0-rc1 tag pushes, `sign-macos` will run with full notarization creds and there's no human-approval gate. Out of Phase 90 scope, but worth knowing before firing a live signed release. User may want to add a required reviewer or wait timer first.

### Pre-flight steps that DID pass

- Step 1 (static gate) — PASS. grep-gate clean, build-script.test.sh 38/38 PASS, go test -race -short 12/12 packages PASS, YAML validate OK. *Note: had to use `go build -tags tools . ./internal/...` instead of `./...` to exclude gitignored `security-review/` dir which has mixed-package files. UAT command should reflect this.*
- Step 2 (tap branch) — DONE. `release-90-test` branch created on `scottkw/homebrew-agenthub` and pushed. Verified via `gh api`.
- Step 5 (no conflicting Dependabot PRs) — clean.

## Where to resume

User chose option C (pause and review the 121-commit divergence and unprotected release env personally) before continuing. Decision points awaiting user:

1. Investigate `go.mod`, `go.sum`, and wailsjs runtime drift — commit or revert?
2. Push 121 commits to `origin/main` — first public push of v3.1 work
3. Wait for build.yml to go green on origin
4. Optionally add `release` env protection rule (required reviewer or wait timer) before live RC test
5. Then resume `90-06-HUMAN-UAT.md` at step 4

After all four are decided/done, the next session can pick up at step 6 (push v3.1.0-rc1 tag) and run through to step 15 (sign-off).

## Files to read first on next session

- This handoff
- `.planning/phases/90-release-pipeline-hardening/90-06-HUMAN-UAT.md` — the runbook
- `.planning/phases/89-vendored-terminal-assets-csp/89-HUMAN-UAT.md` — sign-off and follow-up findings
- `.planning/STATE.md` — milestone progress
