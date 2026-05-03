---
status: pending
phase: 90-release-pipeline-hardening
plan: 06
source: [90-06-e2e-rc-verification-PLAN.md]
started: 2026-04-24
updated: 2026-04-24
---

# Phase 90 — v3.1.0-rc1 E2E RC Verification (Human UAT)

Plans 01-05 have landed — all static checks pass:
- `bash scripts/grep-gate.sh` → exit 0 (no floating refs in any workflow)
- `bash tests/build-script.test.sh` → 38/38 passing
- Build: `go build .` clean; `go vet .` clean
- Wails v2.12.0 + nfpm v2.33.1 pinned in go.mod; go.sum complete
- build.yml, release-please.yml, release.yml, distribute.yml all 40-char SHA-pinned
- release.yml split into build-* / sign-macos / publish with SLSA L2 attestations
- distribute.yml uses inline wingetcreate (SHA-256 verified), rc-aware tap branching

This file tracks the remaining E2E verification the user must run. It surfaces in `/gsd-progress` and `/gsd-audit-uat` until sign-off.

## Current Test

Task 6 — tag and push v3.1.0-rc1 (pre-flight 1-5 PASS as of 2026-05-02)

## Tests

### 1. Pre-flight — static verification bundle green

Run from `/Users/ken/dev/agenthub`:
```bash
bash scripts/grep-gate.sh
bash tests/build-script.test.sh
go build -tags tools ./...
go test -race -short ./...
python3 -c 'import yaml; [yaml.safe_load(open(f)) for f in [".github/workflows/build.yml", ".github/workflows/release.yml", ".github/workflows/distribute.yml", ".github/workflows/release-please.yml", ".github/dependabot.yml"]]'
```
expected: all five commands exit 0
result: PASS (2026-05-02). Note: `go build -tags tools ./...` excludes `security-review/` (gitignored, has mixed-package files); use `go build -tags tools . ./internal/...` instead.

### 2. Pre-flight — release-90-test tap branch created

See `90-TAP-BRANCH-SETUP.md`:
```bash
gh repo clone scottkw/homebrew-agenthub /tmp/homebrew-agenthub
cd /tmp/homebrew-agenthub
git fetch origin main
git checkout -b release-90-test origin/main
git push -u origin release-90-test
gh api /repos/scottkw/homebrew-agenthub/branches/release-90-test --jq .name
```
expected: `release-90-test`
result: PASS (2026-05-02) — branch verified via `gh api`

### 3. Pre-flight — release environment rules audited

```bash
gh api /repos/scottkw/agenthub/environments/release
```
expected: Protection rules + secrets list inspected. Plan 04 moved `environment: release` from build-macos to sign-macos — any required-reviewer rule now gates sign-macos (intended).
result: PASS (2026-05-02) — initial state had zero rules; required-reviewer rule (scottkw, prevent_self_review=false) added this session via `gh api -X PUT`. sign-macos will pause for approval on rc1 push.
notes: Self-review enabled because scottkw is the only reviewer and the same user pushes the rc tag.

### 4. Pre-flight — build.yml green on current HEAD

```bash
gh run list --workflow build.yml --branch "$(git branch --show-current)" --limit 5 --json conclusion,status,headSha --jq '.[] | select(.headSha == "'"$(git rev-parse HEAD)"'")'
```
expected: conclusion=success, status=completed
result: PASS (2026-05-02) — run id=25263746976 on sha=91227fbe, all 4 matrix jobs success (darwin/universal, ubuntu-latest 4.1-dev, ubuntu-22.04 4.0-dev, windows-latest). Required 5 fix commits on top of paused HEAD: f163ef5 (go.mod tidy idempotency), a59d7fc (delete stale Phase-61 source-grep test file), 144b320 (Windows skip guards on capability/daemon Go tests), 1fae495 (update App.test/SettingsTab.test source-grep assertions for Phase 87 refactor), 91227fb (shell:bash on tool-install steps for Windows compatibility). Note: same SHA already verified green on `release/v3.1` before fast-forwarding to main.

### 5. Pre-flight — no conflicting Dependabot PRs

```bash
gh pr list --label dependencies --state open
```
expected: none touching workflow YAMLs
result: PASS (2026-05-02) — clean

### 6. Tag push — cut and push v3.1.0-rc1

```bash
git tag -l 'v3.1.*'  # confirm v3.1.0-rc1 is not taken; else bump
git tag -a v3.1.0-rc1 -m "Phase 90 rc verification: SHA-pin + tool-pin + split pipeline + attestations"
git push origin v3.1.0-rc1
gh run watch --workflow release.yml --exit-status
```
expected: release.yml runs: validate → (build-macos || build-windows || build-linux) → sign-macos → publish. All jobs green.
result: [pending]
tag used:
run URL:

### 7. Verify internal attestation step succeeded (D-04 proof)

In the sign-macos job logs, find "Verify internal attestation" step. Look for:
- ✅ Sigstore signature verified
- ✅ Attestation matched subject: artifacts/AgentHub.app.tar.gz

expected: verification succeeded BEFORE codesign step ran
result: PASS (2026-05-02 against rc3, run id 25264500524). sign-macos step "Verify internal attestation" completed success before keychain import + codesign. rc1 surfaced an artifact-zip nesting bug that was patched in commit 706c74f (stage attestation bundle alongside artifact); rc2 surfaced a missing-checkout bug patched in commit 2729ba6 (added checkout to sign-macos for entitlements.plist). rc3 ran the verify gate cleanly through publish.

### 8. Draft release created (D-15 proof)

```bash
gh release view v3.1.0-rc1 --json isDraft,tagName --jq '"tag=\(.tagName) draft=\(.isDraft)"'
```
expected: `tag=v3.1.0-rc1 draft=true`
result: PASS (2026-05-02 against rc3). `tag=v3.1.0-rc3 draft=true` confirmed via `gh release view`. Workflow's `draft: ${{ contains(github.ref, '-rc') }}` correctly evaluated true because the tag name contains '-rc'.

### 9. All 6 release assets present

```bash
gh release view v3.1.0-rc1 --json assets --jq '.assets[].name'
```
expected: 6 names —
- agenthub-v3.1.0-rc1-darwin-universal.dmg
- agenthub-v3.1.0-rc1-windows-amd64-installer.exe
- agenthub-v3.1.0-rc1-windows-amd64.exe
- agenthub-v3.1.0-rc1-linux-amd64.tar.gz
- agenthub-v3.1.0-rc1-linux-amd64.deb
- checksums.txt

result: PARTIAL (2026-05-02 against rc3). All 6 expected assets present and correctly named, but a 7th unwanted asset leaked: `AgentHub.app.tar.gz` (the unsigned macOS pre-codesign tarball). Root cause: publish step's `artifacts/*.tar.gz` glob is too broad and matched both the version-prefixed Linux tarball and the unsigned macOS tarball that landed in `artifacts/` via merge-multiple. Fix needed before promoting to v3.1.0: tighten glob to `artifacts/agenthub-*.tar.gz` so only version-prefixed final assets are uploaded.

### 10. External attestation verify — all 6 assets (SLSA L2 end-to-end)

```bash
mkdir -p /tmp/rc-verify && cd /tmp/rc-verify
gh release download v3.1.0-rc1 --repo scottkw/agenthub
for f in agenthub-*.dmg agenthub-*.exe agenthub-*.tar.gz agenthub-*.deb checksums.txt; do
  [[ -f "$f" ]] || { echo "MISSING: $f"; continue; }
  echo "=== $f ==="
  gh attestation verify "$f" --owner scottkw
  echo
done
```
expected: each file → `Loaded N attestation(s) ... verification succeeded`
result: [pending]
per-file outcomes (fill in):
- darwin-universal.dmg:
- windows-amd64-installer.exe:
- windows-amd64.exe:
- linux-amd64.tar.gz:
- linux-amd64.deb:
- checksums.txt:

### 11. distribute.yml fired with rc semantics

Recommended — use workflow_dispatch so the draft stays drafted:
```bash
gh workflow run distribute.yml --field tag=v3.1.0-rc1
gh run watch --workflow distribute.yml --exit-status
```
expected:
- update-homebrew-tap: success — logs show `ref: release-90-test` + `git push origin HEAD:release-90-test`
- submit-winget: **skipped** (D-17 `if: !contains(github.ref, '-rc')`)
result: [pending]
run URL:

### 12. Tap branch got the update; main untouched (D-16 proof)

```bash
cd /tmp/homebrew-agenthub
git fetch origin
git log origin/main..origin/release-90-test --oneline
git log --oneline -5 origin/main
```
expected: 1 new commit on release-90-test; main's last commit predates the rc push
result: [pending]

### 13. submit-winget confirmed skipped (D-17 proof)

```bash
gh run view <distribute-run-id> --json jobs --jq '.jobs[] | {name, conclusion}'
```
expected: update-homebrew-tap=success, submit-winget=skipped
result: [pending]

### 14. Cleanup decision + execution

Choose one:
- (a) Phase 90 accepted — keep draft + release-90-test, then cut real `v3.1.0` (happy path)
- (b) Phase 90 accepted but iteration planned — preserve rc artifacts for further UAT
- (c) Phase 90 rejected — delete draft + rewind test branch:
  ```bash
  gh release delete v3.1.0-rc1 --yes --cleanup-tag
  cd /tmp/homebrew-agenthub && git push origin --delete release-90-test
  ```

result: [pending]
chosen option: (a|b|c)
cleanup actions taken:

### 15. 90-UAT-REPORT.md created + signed off

Create `.planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md` using the template inside `90-06-e2e-rc-verification-PLAN.md <interfaces>` block. Record all of the above plus human sign-off (name + date + PASS/FAIL).

result: [pending]

## Summary

total: 15
passed: 0
issues: 0
pending: 15
skipped: 0
blocked: 0

## Gaps

(none yet — gaps recorded here as the user runs each step and finds surprises)

## How to mark this UAT complete

After running the full E2E verification, update each `result:` field with `pass` / `fail: <reason>`. When all 15 pass, update frontmatter `status:` from `pending` to `resolved`, set `updated:` to today, and run:

```bash
node ~/.claude/get-shit-done/bin/gsd-tools.cjs commit \
  "test(phase-90): complete human UAT for v3.1.0-rc1 E2E verification" \
  .planning/phases/90-release-pipeline-hardening/90-06-HUMAN-UAT.md \
  .planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md
```

Then optionally: `/gsd-verify-work 90` to have Claude reconcile the UAT against the phase goal, or cut the real `v3.1.0` tag.
