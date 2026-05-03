# Phase 90: Release Pipeline Hardening - Pattern Map

**Mapped:** 2026-04-23
**Files analyzed:** 10 (5 new, 5 modified)
**Analogs found:** 8 / 10 (2 net-new files have no in-repo analog — sourced from RESEARCH.md recipes)

## Scope Reminder

Phase 90 is **CI/CD surface only** — workflow YAMLs, shell scripts, Go build tooling, Dependabot config. No runtime code. No frontend. No application-level tests. The only code excerpts below come from `.github/workflows/*.yml`, `build.sh`, `tests/build-script.test.sh`, and `go.mod` — all already SHA-pinned-style file types.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `.github/dependabot.yml` (NEW) | config | event-driven (weekly schedule) | *(none in repo — RESEARCH Example 4)* | no-analog |
| `tools.go` (NEW, repo root) | config (Go build-tool manifest) | n/a (compile-time build-tag gate) | *(none in repo — RESEARCH Example 1)* | no-analog |
| `.github/workflows/hardening-check.yml` (NEW — Claude's discretion per D-09) OR inline step in `build.yml` | config (CI workflow) | event-driven (push/PR) | `.github/workflows/build.yml` (simple single-job shell-gate shape) | role-match |
| `.github/workflows/build.yml` (MODIFIED) | config (CI workflow) | event-driven (push, PR) | *(self — SHA-pin + wails-install swap in place)* | exact |
| `.github/workflows/release.yml` (MODIFIED — biggest change: split) | config (CI workflow) | event-driven (tag push → multi-job pipeline) | *(self — existing `build-macos`/`build-windows`/`build-linux`/`publish` skeleton is ~80% of the final shape)* | exact |
| `.github/workflows/distribute.yml` (MODIFIED) | config (CI workflow) | event-driven (release:published) | *(self — swap one action, add rc-guard, move winget job to windows-latest)* | exact |
| `.github/workflows/release-please.yml` (MODIFIED) | config (CI workflow) | event-driven (push to main) | *(self — single-line SHA-pin)* | exact |
| `build.sh` (MODIFIED) | utility (shell script) | batch (CLI-invoked) | *(self — swap install hint on line 65; add pinned-version sanity gate)* | exact |
| `tests/build-script.test.sh` (MODIFIED) | test (Bash) | batch | *(self — already has `assert_file_contains_regex` / `assert_file_contains_literal` harness; add new assertions)* | exact |
| `go.mod` / `go.sum` (MODIFIED) | config (Go module manifest) | n/a (declarative) | *(self — add `nfpm` require block alongside existing `wails`)* | exact |

---

## Pattern Assignments

### `.github/workflows/build.yml` (MODIFIED — SHA-pin + wails-install swap)

**Analog:** `.github/workflows/build.yml` itself (file is mostly right, two patterns change).

**Pattern to REPLACE — `wails@latest` install** (line 80-81, SEC-10 violation):
```yaml
# CURRENT (line 80-81)
- name: Install Wails CLI
  run: go install github.com/wailsapp/wails/v2/cmd/wails@latest
```
Replace with the **D-11 go-list-derived pattern** (see Shared Patterns §"Wails CLI install from go.mod" below).

**Pattern to REPLACE — unpinned `@v*` Actions**: every `uses:` line at:
- Line 34 `actions/checkout@v4`
- Line 45 `actions/setup-go@v5`
- Lines 65 `pnpm/action-setup@v4`
- Line 70 `actions/setup-node@v4`
- Lines 106, 114, 122, 130 `actions/upload-artifact@v4`

Replace each per the SHA-pin table in Shared Patterns §"SHA-pinned Actions uses: block" below.

**Pattern to PRESERVE — matrix + conditional steps** (lines 6-29, 38-42, 49-58, 77, 104-136):
Existing shape is fine; no structural change needed. The `if: runner.os == 'Linux'` and platform-conditional step guards already compose cleanly with a SHA-pinned `uses:`.

---

### `.github/workflows/release.yml` (MODIFIED — MAJOR: three-stage split)

**Analog:** `.github/workflows/release.yml` itself (the existing file contains ~80% of the final shape; the work is **extraction and secret rescoping**, not rewriting from scratch).

**EXTRACT pattern — signing steps → new `sign-macos` job** (lines 93-141):
```yaml
# release.yml:93-141 (verbatim) — moves to sign-macos job
- name: Import certificate to keychain
  run: |
    echo "$MACOS_CERTIFICATE" | base64 --decode > certificate.p12
    security create-keychain -p "$MACOS_CI_KEYCHAIN_PWD" build.keychain
    security default-keychain -s build.keychain
    security unlock-keychain -p "$MACOS_CI_KEYCHAIN_PWD" build.keychain
    security import certificate.p12 -k build.keychain \
      -P "$MACOS_CERTIFICATE_PWD" -T /usr/bin/codesign
    security set-key-partition-list -S apple-tool:,apple: \
      -s -k "$MACOS_CI_KEYCHAIN_PWD" build.keychain

- name: Sign .app with hardened runtime
  run: |
    codesign --deep --force --verbose \
      --options runtime \
      --timestamp \
      --entitlements build/entitlements.plist \
      --sign "$MACOS_CERTIFICATE_NAME" \
      build/bin/AgentHub.app

- name: Notarize .app
  run: |
    xcrun notarytool store-credentials "agenthub-notarize" \
      --apple-id "$MACOS_NOTARIZATION_APPLE_ID" \
      --team-id "$MACOS_NOTARIZATION_TEAM_ID" \
      --password "$MACOS_NOTARIZATION_PWD"
    ditto -c -k --keepParent build/bin/AgentHub.app build/bin/agenthub-notarize.zip
    xcrun notarytool submit build/bin/agenthub-notarize.zip \
      --keychain-profile "agenthub-notarize" \
      --wait
    xcrun stapler staple build/bin/AgentHub.app

- name: Create and sign DMG
  run: |
    brew install create-dmg
    VERSION="${GITHUB_REF_NAME}"
    DMG_NAME="agenthub-${VERSION}-darwin-universal.dmg"
    create-dmg \
      --volname "AgentHub" \
      --codesign "$MACOS_CERTIFICATE_NAME" \
      "${DMG_NAME}" \
      build/bin/AgentHub.app
    mv "${DMG_NAME}" build/bin/

- name: Cleanup keychain
  if: always()
  run: |
    security delete-keychain build.keychain || true
    rm -f certificate.p12 build/bin/agenthub-notarize.zip
```
These steps move verbatim to `sign-macos`. **The codesign/notarize logic is not what's changing — only which job holds the secrets.**

**REMOVE pattern — MACOS_* env block from `build-macos`** (lines 46-54):
```yaml
# CURRENT — build-macos has MACOS_* secrets (SEC-11 violation)
environment: release
env:
  MACOS_CERTIFICATE: ${{ secrets.MACOS_CERTIFICATE }}
  MACOS_CERTIFICATE_NAME: ${{ secrets.MACOS_CERTIFICATE_NAME }}
  MACOS_CERTIFICATE_PWD: ${{ secrets.MACOS_CERTIFICATE_PWD }}
  MACOS_CI_KEYCHAIN_PWD: ${{ secrets.MACOS_CI_KEYCHAIN_PWD }}
  MACOS_NOTARIZATION_APPLE_ID: ${{ secrets.MACOS_NOTARIZATION_APPLE_ID }}
  MACOS_NOTARIZATION_PWD: ${{ secrets.MACOS_NOTARIZATION_PWD }}
  MACOS_NOTARIZATION_TEAM_ID: ${{ secrets.MACOS_NOTARIZATION_TEAM_ID }}
```
After split: `build-macos` has **no `env:` block and no `environment:` declaration**. The `environment: release` declaration moves to `sign-macos` (preserves secret binding — Pitfall 4 in RESEARCH). The whole env block above moves into `sign-macos`.

**REPLACE pattern — publish job token misuse** (line 316, D-02 fix):
```yaml
# CURRENT (line 316) — TAP_DEPLOY_TOKEN misused
          token: ${{ secrets.TAP_DEPLOY_TOKEN }}
```
Replace with:
```yaml
          token: ${{ secrets.GITHUB_TOKEN }}   # D-02: TAP_DEPLOY_TOKEN belongs to distribute.yml
```

**ADD pattern — tar-before-upload for macOS .app** (between line 91 and 143 of current file, in new `build-macos` job):
```yaml
# CRITICAL per RESEARCH "surprise #2": upload-artifact@v4+ strips symlinks + x-bits
- name: Tar .app bundle (preserve symlinks and +x bits)
  run: tar -czf build/bin/AgentHub.app.tar.gz -C build/bin AgentHub.app

# ...upload the .tar.gz, not the raw .app...
- name: Upload unsigned .app bundle (tarred)
  uses: actions/upload-artifact@<sha> # v7.0.1 (see Shared Patterns)
  with:
    name: agenthub-darwin-universal-unsigned
    path: build/bin/AgentHub.app.tar.gz
    if-no-files-found: error
```
And the mirrored **untar in sign-macos** (from RESEARCH Pattern 1):
```yaml
- name: Untar .app bundle
  run: tar -xzf artifacts/AgentHub.app.tar.gz -C build/bin/
```

**ADD pattern — internal build-provenance attestation** (new step in each `build-*` job, D-04):
See Shared Patterns §"Internal attestation + verify across trust boundary" below.

**ADD pattern — release attestation in publish job** (new step before `softprops/action-gh-release`, D-05):
```yaml
- name: Generate SHA256 checksums
  run: |
    cd artifacts
    sha256sum * > checksums.txt
    cat checksums.txt

- name: Release build-provenance attestation
  uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
  with:
    subject-checksums: artifacts/checksums.txt
```
The `subject-checksums` input attests every file listed in `checksums.txt` with one action call — RESEARCH confirms this is the documented pattern for multi-artifact releases.

**ADD pattern — draft release on rc tags** (modify `softprops/action-gh-release` step, D-15):
```yaml
- name: Upload to GitHub Release
  uses: softprops/action-gh-release@b4309332981a82ec1c5618f44dd2e27cc8bfbfda # v3.0.0
  with:
    files: |
      artifacts/*.dmg
      artifacts/*.exe
      artifacts/*.tar.gz
      artifacts/*.deb
      artifacts/checksums.txt
    fail_on_unmatched_files: true
    draft: ${{ contains(github.ref, '-rc') }}   # D-15
    token: ${{ secrets.GITHUB_TOKEN }}          # D-02
```
**Hyphen-anchored** (`'-rc'` not `'rc'`) per RESEARCH Anti-Pattern note — `v3.1.0-archive` would false-match `'rc'`.

**PRESERVE pattern — existing publish structure** (lines 288-314):
```yaml
publish:
  needs: [build-macos, build-windows, build-linux]
  runs-on: ubuntu-latest
  permissions:
    contents: write
  steps:
    - name: Download all artifacts
      uses: actions/download-artifact@v4
      with:
        path: artifacts
        merge-multiple: true

    - name: Generate SHA256 checksums
      run: |
        cd artifacts
        sha256sum * > checksums.txt
        cat checksums.txt

    - name: Upload to GitHub Release
      uses: softprops/action-gh-release@v2
```
~80% of the shape survives. Changes: `needs:` adds `sign-macos`; `permissions:` adds `id-token: write` and `attestations: write` for D-05; the checksum + attest + release-upload steps get the three modifications above; download-artifact gets SHA-pinned.

---

### `.github/workflows/distribute.yml` (MODIFIED)

**Analog:** `.github/workflows/distribute.yml` itself.

**Pattern to PRESERVE — `update-homebrew-tap` job** (lines 16-70):
The entire tap-update job is structurally fine. Changes are surgical:
- SHA-pin `actions/checkout@v4` (line 49) and `nick-fields/retry@v3` (line 27).
- Add **D-16 rc-branch guard** to the `Checkout tap repo` step:
```yaml
- name: Checkout tap repo
  uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
  with:
    repository: scottkw/homebrew-agenthub
    ref: ${{ contains(github.ref, '-rc') && 'release-90-test' || 'main' }}  # D-16
    token: ${{ secrets.TAP_DEPLOY_TOKEN }}
    path: tap
```
And the git-push step needs the same branch awareness:
```yaml
- name: Commit and push formula update
  run: |
    cd tap
    BRANCH="${{ contains(github.ref, '-rc') && 'release-90-test' || 'main' }}"
    # ...
    git push origin HEAD:"$BRANCH"
```

**Pattern to REPLACE — `submit-winget` job** (lines 72-81):
Current:
```yaml
submit-winget:
  runs-on: ubuntu-latest
  continue-on-error: true  # WinGet first submission pending — remove after accepted
  steps:
    - uses: vedantmgoyal9/winget-releaser@main
      with:
        identifier: scottkw.agenthub
        installers-regex: 'agenthub-v[\d.]+-windows-amd64-installer\.exe$'
        token: ${{ secrets.WINGET_TOKEN }}
        release-tag: ${{ env.RELEASE_TAG }}
```
Three things wrong with this: runner must become `windows-latest` (wingetcreate is Windows-only — RESEARCH surprise #1), the third-party `@main` ref is the only floating branch in the repo (SEC-09 violation), and the WinGet queue shouldn't see rc tags (D-17).

Replace per **RESEARCH Pattern 4** (copy verbatim from RESEARCH lines 393-422):
```yaml
submit-winget:
  runs-on: windows-latest           # D-08: wingetcreate is .NET+Windows only
  continue-on-error: true           # preserve — WinGet first submission is known-flaky
  if: ${{ !contains(github.ref, '-rc') }}   # D-17
  env:
    WINGET_CREATE_GITHUB_TOKEN: ${{ secrets.WINGET_TOKEN }}
    WINGETCREATE_VERSION: v1.12.8.0
    WINGETCREATE_SHA256: 8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D
  steps:
    - name: Download and verify wingetcreate.exe
      shell: pwsh
      run: |
        $url = "https://github.com/microsoft/winget-create/releases/download/$env:WINGETCREATE_VERSION/wingetcreate.exe"
        Invoke-WebRequest -Uri $url -OutFile wingetcreate.exe
        $actual = (Get-FileHash wingetcreate.exe -Algorithm SHA256).Hash
        if ($actual -ne $env:WINGETCREATE_SHA256) {
          Write-Error "SHA256 mismatch: expected $env:WINGETCREATE_SHA256 got $actual"
          exit 1
        }
    - name: Submit to WinGet
      shell: pwsh
      run: |
        $version = "${{ github.event.release.tag_name }}".Trim('v')
        $installerUrl = "https://github.com/scottkw/agenthub/releases/download/${{ github.event.release.tag_name }}/agenthub-${{ github.event.release.tag_name }}-windows-amd64-installer.exe"
        .\wingetcreate.exe update scottkw.agenthub `
          --version $version `
          --urls $installerUrl `
          --submit
```
**`continue-on-error: true` is preserved** — RESEARCH flags this is intentional ("WinGet first submission pending" comment at distribute.yml:74 is load-bearing). Keep the comment.

**Pattern to PRESERVE — env-scoped `RELEASE_TAG`** (lines 12-13):
```yaml
env:
  RELEASE_TAG: ${{ inputs.tag || github.ref_name }}
```
Unchanged — this is a clean pattern.

---

### `.github/workflows/release-please.yml` (MODIFIED — single SHA-pin)

**Analog:** the file itself (two lines of change).

**Pattern to REPLACE** (line 16):
```yaml
# CURRENT
- uses: googleapis/release-please-action@v4
```
Replace with SHA-pinned form per Shared Patterns table.

**PRESERVE everything else** — the `permissions:`, `secrets.RELEASE_PLEASE_TOKEN`, and config-file references are fine.

---

### `.github/workflows/hardening-check.yml` (NEW — OR inline step in build.yml; Claude's discretion per D-09)

**Analog (closest existing shape):** `.github/workflows/build.yml` — a simple single-job workflow that runs shell assertions.

**Source content:** RESEARCH Example 5 (lines 684-719) is the gate logic. Wrap it in a minimal workflow:

```yaml
name: Hardening Check

on:
  push:
    branches: [main]
  pull_request:
    paths:
      - '.github/workflows/**'
      - 'build.sh'
      - 'tests/**'

jobs:
  grep-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
      - name: Assert no floating refs in workflows
        run: |
          set -euo pipefail
          BAD=$(grep -rEn 'uses:\s*[^#]*@(main|master|v[0-9]+|[a-z]+)$' .github/workflows/ || true)
          if [[ -n "$BAD" ]]; then
            echo "FAIL: unpinned action refs found:"; echo "$BAD"; exit 1
          fi
          LATEST=$(grep -rEn '@latest' .github/workflows/ build.sh tests/ || true)
          if [[ -n "$LATEST" ]]; then
            echo "FAIL: @latest references found:"; echo "$LATEST"; exit 1
          fi
          NON_SHA=$(grep -rE 'uses:\s*[^ ]+@' .github/workflows/ \
            | grep -Ev 'uses:\s*[^ ]+@[a-f0-9]{40}(\s|$)' || true)
          if [[ -n "$NON_SHA" ]]; then
            echo "FAIL: non-SHA action refs:"; echo "$NON_SHA"; exit 1
          fi
          echo "PASS: all action refs are SHA-pinned"
```

**Claude's discretion note:** RESEARCH lines 720-721 **recommend the inline-step-in-build.yml form** ("cheaper than a separate workflow"). Planner picks. Either way, the step **body** above is reused verbatim.

---

### `tools.go` (NEW — repo root)

**Analog:** None in repo. Content is RESEARCH Example 1 (lines 554-573) verbatim:

```go
//go:build tools
// +build tools

// Package tools documents build-tool dependencies. It is excluded from normal
// builds by the `tools` build tag. The blank imports cause `go mod tidy` to
// keep these modules in go.mod alongside runtime dependencies, making Dependabot
// aware of them via the gomod ecosystem.
//
// CI and build.sh install these tools using:
//   go install <path>@$(go list -m -f '{{.Version}}' <module>)
//
// See .planning/phases/90-release-pipeline-hardening/ for rationale.
package tools

import (
	_ "github.com/goreleaser/nfpm/v2/cmd/nfpm"
	_ "github.com/wailsapp/wails/v2/cmd/wails"
)
```

**Location per D-10 + Claude's discretion (RESEARCH line 575):** repo root. Not `internal/tools/`. Well-known Go projects (grpc-go, kubernetes, cockroachdb) all use root-level `tools.go`.

---

### `.github/dependabot.yml` (NEW)

**Analog:** None in repo. Content is RESEARCH Example 4 (lines 651-678) verbatim:

```yaml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "09:00"
      timezone: "America/Los_Angeles"
    open-pull-requests-limit: 5
    commit-message:
      prefix: "ci(actions)"
    labels:
      - "dependencies"
      - "github-actions"
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
    open-pull-requests-limit: 5
    commit-message:
      prefix: "deps"
    labels:
      - "dependencies"
      - "go"
```

**D-07 "no auto-merge":** No schema field needed; auto-merge is a separate repo setting. Per RESEARCH Pitfall 6, **the planner should NOT add an auto-merge workflow**, and should document manual-merge-only expectation.

---

### `build.sh` (MODIFIED)

**Analog:** `build.sh` itself (lines 61-67 change; rest unchanged).

**Pattern to REPLACE** (lines 61-67, per D-12 / D-13):
```bash
# CURRENT (build.sh:61-67)
# Wails binary check
WAILS="$(go env GOPATH)/bin/wails"
if [[ ! -x "$WAILS" ]]; then
  echo "ERROR: wails not found at $WAILS"
  echo "Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
  exit 1
fi
```

Replace with RESEARCH Example 3 content (lines 619-636):
```bash
# NEW
WAILS="$(go env GOPATH)/bin/wails"
WAILS_PINNED_VER="$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2 2>/dev/null)"
if [[ -z "$WAILS_PINNED_VER" ]]; then
  echo "ERROR: wails not pinned in go.mod — Phase 90 tools.go setup missing"
  exit 1
fi
if [[ ! -x "$WAILS" ]]; then
  echo "ERROR: wails not found at $WAILS"
  echo "Install: go install github.com/wailsapp/wails/v2/cmd/wails@$WAILS_PINNED_VER"
  exit 1
fi
# Optional: verify installed version matches pinned version
if ! "$WAILS" version 2>/dev/null | grep -qF "$WAILS_PINNED_VER"; then
  echo "WARN: installed wails does not match pinned version ($WAILS_PINNED_VER)"
  echo "Reinstall: go install github.com/wailsapp/wails/v2/cmd/wails@$WAILS_PINNED_VER"
fi
```

**PRESERVE patterns** — the `set -euo pipefail`, argument parsing (lines 22-49), `build_macos`/`build_windows`/`build_linux` dispatch functions (lines 71-127), and the full `sign_and_notarize()` function (lines 130-207) are all unchanged. SEC-11's trust-boundary concern is about CI secret scope, not this local dev script.

---

### `tests/build-script.test.sh` (MODIFIED)

**Analog:** The file itself — it already has a clean helper harness. Just add assertions.

**Established test helpers available** (from test file, lines 63-84):
```bash
assert_file_contains_literal() {
  local name="$1"
  local pattern="$2"
  local file="$3"
  if grep -qF -- "$pattern" "$file"; then
    pass "$name"
  else
    fail "$name" "literal pattern not found in $file: $pattern"
  fi
}

assert_file_contains_regex() {
  local name="$1"
  local pattern="$2"
  local file="$3"
  if grep -qE "$pattern" "$file"; then
    pass "$name"
  else
    fail "$name" "regex pattern not found in $file: $pattern"
  fi
}
```

**Existing section to extend** — "Section 9: Static pattern checks — build tool invocations" (lines 228-246) is the right home. Add from RESEARCH Example 3 (lines 638-647):
```bash
# New Section 12 (or extend Section 11): SEC-10 compliance
echo ""
echo "=== Section 12: SEC-10 compliance — no @latest refs ==="

# Negation: @latest must be ABSENT
output=$(grep -c '@latest' "$BUILD_SH" || true)
if [[ "$output" -eq 0 ]]; then
  pass "build.sh contains no @latest references (SEC-10)"
else
  fail "build.sh contains @latest — SEC-10 violation" "matches: $output"
fi

# Positive assertion: new pinned-install pattern present
assert_file_contains_literal "build.sh contains go list -m pin pattern" \
  "go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2" "$BUILD_SH"

assert_file_contains_literal "build.sh gates on WAILS_PINNED_VER" \
  "WAILS_PINNED_VER" "$BUILD_SH"
```

**PRESERVE — Section numbering / PASS/FAIL accounting** (lines 12-13, 295-303). The `PASS=0; FAIL=0` counters and `echo "Results: $PASS passed, $FAIL failed"` footer must remain intact.

---

### `go.mod` / `go.sum` (MODIFIED)

**Analog:** `go.mod` itself.

**Pattern — add nfpm to `require` block** (insert alphabetically between `github.com/godbus/dbus/v5` at line 16 and `github.com/kardianos/service` at line 17):
```go
require (
    // ... existing entries ...
    github.com/godbus/dbus/v5 v5.2.0
    github.com/goreleaser/nfpm/v2 v2.46.3       // NEW (Phase 90 — via tools.go)
    github.com/kardianos/service v1.2.4
    // ... rest unchanged ...
    github.com/wailsapp/wails/v2 v2.10.2        // existing — optionally bump to v2.12.0
    // ... rest unchanged ...
)
```

**Auto-update mechanism:** `go mod tidy` (run after creating `tools.go`) will populate the entry and `go.sum` automatically. Planner should NOT hand-author go.sum.

**Wails version bump (RESEARCH §Standard Stack, "Build verification" — recommends option (a)):** Bump `github.com/wailsapp/wails/v2` from `v2.10.2` to `v2.12.0` in the same commit that lands `tools.go`, so runtime and CLI share one source of truth. This is one coordinated bump, covered by a later Dependabot PR otherwise.

---

## Shared Patterns

These apply across multiple workflow files.

### SHA-pinned Actions `uses:` block

**Source:** RESEARCH §"Core third-party GitHub Actions (pinned SHAs)" (lines 101-119) — all SHAs live-verified on 2026-04-23.

**Apply to:** Every `uses:` line in `.github/workflows/*.yml`.

**Pinning format** (D-06):
```yaml
uses: <owner>/<repo>@<40-char-sha> # vX.Y.Z
```

| Action | SHA-pinned form |
|--------|----------------|
| `actions/checkout` | `actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2` |
| `actions/setup-go` | `actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0` |
| `actions/setup-node` | `actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6.4.0` |
| `actions/upload-artifact` | `actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1` |
| `actions/download-artifact` | `actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1` |
| `actions/attest-build-provenance` | `actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0` |
| `pnpm/action-setup` | `pnpm/action-setup@903f9c1a6ebcba6cf41d87230be49611ac97822e # v6.0.3` |
| `softprops/action-gh-release` | `softprops/action-gh-release@b4309332981a82ec1c5618f44dd2e27cc8bfbfda # v3.0.0` |
| `nick-fields/retry` | `nick-fields/retry@ad984534de44a9489a53aefd81eb77f87c70dc60 # v4.0.0` |
| `googleapis/release-please-action` | `googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0` |

**To be REMOVED:** `vedantmgoyal9/winget-releaser@main` (replaced by inline `wingetcreate.exe` per D-08).

**RESEARCH Assumption A1:** upload-artifact@v4→v7 is backwards-compatible for plain files. Planner may elect to stay on v4 SHA-pinned if risk-averse; both forms satisfy SEC-09.

---

### Wails CLI install from go.mod (D-11)

**Source:** RESEARCH Example 2 (lines 597-602) + Pitfall 5 wrap (lines 519-524).

**Apply to:** Every CI step currently reading `go install github.com/wailsapp/wails/v2/cmd/wails@latest`:
- `.github/workflows/build.yml` line 80-81
- `.github/workflows/release.yml` lines 78-79, 176-177, 242-243 (three build-* jobs — each needs its own install step after the split)

**Pattern:**
```yaml
- name: Install Wails CLI (version from go.mod)
  run: |
    WAILS_VER=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
    [ -n "$WAILS_VER" ] || { echo "wails not pinned in go.mod"; exit 1; }
    go install github.com/wailsapp/wails/v2/cmd/wails@"$WAILS_VER"
```

**Mirror for nfpm** (replaces `release.yml:264`):
```yaml
- name: Install nfpm (version from go.mod)
  run: |
    NFPM_VER=$(go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2)
    [ -n "$NFPM_VER" ] || { echo "nfpm not pinned in go.mod"; exit 1; }
    go install github.com/goreleaser/nfpm/v2/cmd/nfpm@"$NFPM_VER"
```

---

### Internal attestation + verify across trust boundary (D-04)

**Source:** RESEARCH Pattern 1 (lines 245-319) + Pattern 3 (lines 369-385).

**Apply to:** Each `build-<platform>` job (generate); `sign-macos` job (verify).

**Build-side — generate attestation + upload bundle:**
```yaml
permissions:
  id-token: write          # for attest-build-provenance OIDC
  attestations: write      # for persist-to-GH-attestations-API
  contents: read

steps:
  # ... build wails ...
  # ... tar .app for macOS only ...
  - name: Attest unsigned artifact (internal)
    id: attest
    uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
    with:
      subject-path: build/bin/AgentHub.app.tar.gz   # or .exe / .tar.gz / .deb per platform
  - name: Upload unsigned artifact + attestation bundle
    uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
    with:
      name: agenthub-<platform>-unsigned
      path: |
        build/bin/AgentHub.app.tar.gz
        ${{ steps.attest.outputs.bundle-path }}
      if-no-files-found: error
```

**Sign-side — verify before signing:**
```yaml
- name: Download unsigned artifact + attestation bundle
  uses: actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1
  with:
    name: agenthub-darwin-universal-unsigned
    path: artifacts/
- name: Verify internal attestation
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: |
    BUNDLE=$(ls artifacts/attestation.json artifacts/*.sigstore.json 2>/dev/null | head -1)
    gh attestation verify artifacts/AgentHub.app.tar.gz \
      --repo ${{ github.repository }} \
      --bundle "$BUNDLE"
```
Per RESEARCH Open Question #2: `sign-macos` likely only needs `permissions: contents: read` for offline `--bundle` verify. Start minimal; escalate only on failure.

---

### Per-job secret scoping (D-01 / Pitfall 4)

**Source:** RESEARCH §"Key properties" (lines 352-355) + Pitfall 4 (lines 504-511).

**Apply to:** `release.yml` post-split.

**Rules:**
- `build-macos`, `build-windows`, `build-linux`: no `environment:` declaration, no `env:` block (or env only for non-secret build tunables).
- `sign-macos`: `environment: release` (preserves repo-environment protection rules); `env:` contains ONLY `MACOS_*` secrets (exact list from release.yml:48-54 above).
- `publish`: no `environment:` declaration; only `secrets.GITHUB_TOKEN` in-step, no repo-wide env block.

---

### rc-tag guards (D-15 / D-16 / D-17)

**Apply to:** `release.yml` publish step (draft), `distribute.yml` tap checkout (branch), `distribute.yml` submit-winget (skip).

**Always use hyphen-anchored form** per RESEARCH Anti-Pattern (lines 436):
```yaml
${{ contains(github.ref, '-rc') }}
```
Not `contains(github.ref, 'rc')` — would false-match `v3.1.0-archive` etc.

Three uses:
1. `release.yml` upload step: `draft: ${{ contains(github.ref, '-rc') }}`
2. `distribute.yml` tap checkout: `ref: ${{ contains(github.ref, '-rc') && 'release-90-test' || 'main' }}`
3. `distribute.yml` submit-winget job: `if: ${{ !contains(github.ref, '-rc') }}`

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `.github/dependabot.yml` | config | event-driven (weekly) | Repo has never had Dependabot config before. Content sourced from RESEARCH Example 4. |
| `tools.go` | config (Go build manifest) | n/a | Repo has no tools.go pattern anywhere. Content sourced from RESEARCH Example 1. |

Both files are tiny, schema-driven, and RESEARCH provides verbatim copy-ready content. No analog needed.

---

## Metadata

**Analog search scope:**
- `/Users/ken/dev/agenthub/.github/workflows/` — all 4 existing workflow YAMLs read in full.
- `/Users/ken/dev/agenthub/build.sh` — read in full (218 lines).
- `/Users/ken/dev/agenthub/tests/build-script.test.sh` — read in full (304 lines).
- `/Users/ken/dev/agenthub/go.mod` — read in full (101 lines).
- Repo-wide grep for `tools.go` / `//go:build tools` — zero matches (confirms tools.go is net-new).
- `/Users/ken/dev/agenthub/.github/dependabot.yml` — not present (confirms Dependabot is net-new).

**Files scanned:** 8 source files + 1 repo-wide grep. Total analogs used: 6 (4 workflows + build.sh + tests). Content pulled verbatim from RESEARCH.md for 4 items (tools.go, dependabot.yml, wingetcreate recipe, grep-gate). All 40-char SHAs in this doc come from RESEARCH's live-API-verified table.

**Pattern extraction date:** 2026-04-23

**Output file:** `.planning/phases/90-release-pipeline-hardening/90-PATTERNS.md`
