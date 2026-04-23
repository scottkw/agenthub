---
phase: 90-release-pipeline-hardening
plan: 04
type: execute
wave: 3
depends_on: [02]
files_modified:
  - .github/workflows/release.yml
autonomous: true
requirements: [SEC-09, SEC-10, SEC-11]
tags: [ci, hardening, release-yml, job-split, attestation, wave-3]

must_haves:
  truths:
    - "release.yml's build-macos/build-windows/build-linux jobs run without any MACOS_*, WINGET_TOKEN, TAP_DEPLOY_TOKEN, RELEASE_PLEASE_TOKEN, or environment: release declaration — i.e., the untrusted build step has zero access to signing/publish secrets"
    - "release.yml has a new sign-macos job with environment: release that holds ONLY the MACOS_* secrets, consumes build-macos output, verifies the internal attestation cryptographically before signing, and uploads only the signed DMG"
    - "release.yml's publish job depends on [build-macos, build-windows, build-linux, sign-macos], uses secrets.GITHUB_TOKEN (NOT TAP_DEPLOY_TOKEN) as the release-upload token, attests every asset in the published release via actions/attest-build-provenance, and marks the release as draft when the tag matches v*-rc*"
    - "macOS .app bundle is tar -czf'd before upload and untar'd before sign — symlinks and +x bits survive the cross-job artifact handoff"
    - "Every uses: in release.yml is SHA-pinned with a trailing # vX.Y.Z comment; zero @latest references (wails and nfpm use the go list -m pattern)"
  artifacts:
    - path: ".github/workflows/release.yml"
      provides: "Split three-stage release pipeline with internal + release SLSA L2 attestations"
      contains: "attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32"
  key_links:
    - from: "build-macos job (tar step)"
      to: "sign-macos job (untar step)"
      via: "actions/upload-artifact@v7 + actions/download-artifact@v8 with tarball"
      pattern: "tar -czf build/bin/AgentHub.app.tar.gz -C build/bin AgentHub.app"
    - from: "build-<platform> attest step"
      to: "sign-macos gh attestation verify step"
      via: "bundle-path output uploaded + gh attestation verify --bundle"
      pattern: "gh attestation verify.*--bundle"
    - from: "publish job"
      to: "softprops/action-gh-release@v3.0.0"
      via: "token: secrets.GITHUB_TOKEN (NOT TAP_DEPLOY_TOKEN) + draft: contains(github.ref, '-rc')"
      pattern: "token: \\$\\{\\{ secrets.GITHUB_TOKEN \\}\\}"
---

<objective>
Execute the core SEC-11 restructure: split release.yml's tag-triggered pipeline into three secret-isolated stages (build → sign → publish), add SLSA L2 provenance attestations at both the internal trust boundary (D-04) and for public consumption (D-05), fix the TAP_DEPLOY_TOKEN misuse (D-02), SHA-pin every action (SEC-09), swap wails/nfpm installs to the go-list pattern (SEC-10), and anchor the rc-draft behavior (D-15) with the hyphen-safe form.

Purpose: This is the load-bearing phase plan. The current `build-macos` job runs `wails build` (untrusted code invoking `npm install` transitively) with `MACOS_CERTIFICATE`, `MACOS_NOTARIZATION_*`, etc. in its env — a single compromised downstream dep can exfiltrate signing material. After this plan, `build-macos` has literally empty `env:` and no `environment:` declaration; signing material is bound only to the minimal `sign-macos` job whose code path does not execute untrusted third-party Go/Node deps.

Output: `.github/workflows/release.yml` restructured into validate → {build-macos, build-windows, build-linux} → sign-macos → publish. Each job's secret scope is the minimum necessary. The existing codesign/notarize/DMG logic is re-homed verbatim — the cryptographic material is unchanged; only the container holding it moves.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md

@.planning/phases/90-release-pipeline-hardening/90-CONTEXT.md
@.planning/phases/90-release-pipeline-hardening/90-RESEARCH.md
@.planning/phases/90-release-pipeline-hardening/90-PATTERNS.md

@.planning/phases/90-release-pipeline-hardening/90-02-SUMMARY.md
@.planning/phases/90-release-pipeline-hardening/90-03-SUMMARY.md

@./CLAUDE.md

<interfaces>
<!-- SHA-pin reference table (verbatim from 90-PATTERNS.md) -->

| Action | Pinned form |
|--------|-------------|
| `actions/checkout` | `actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2` |
| `actions/setup-go` | `actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0` |
| `actions/setup-node` | `actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6.4.0` |
| `actions/upload-artifact` | `actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1` |
| `actions/download-artifact` | `actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1` |
| `actions/attest-build-provenance` | `actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0` |
| `pnpm/action-setup` | `pnpm/action-setup@903f9c1a6ebcba6cf41d87230be49611ac97822e # v6.0.3` |
| `softprops/action-gh-release` | `softprops/action-gh-release@b4309332981a82ec1c5618f44dd2e27cc8bfbfda # v3.0.0` |

<!-- Wails + nfpm install pattern (D-11, from 90-PATTERNS.md Shared Patterns) -->

```yaml
- name: Install Wails CLI (version from go.mod)
  run: |
    WAILS_VER=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
    [ -n "$WAILS_VER" ] || { echo "wails not pinned in go.mod"; exit 1; }
    go install github.com/wailsapp/wails/v2/cmd/wails@"$WAILS_VER"

- name: Install nfpm (version from go.mod)
  run: |
    NFPM_VER=$(go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2)
    [ -n "$NFPM_VER" ] || { echo "nfpm not pinned in go.mod"; exit 1; }
    go install github.com/goreleaser/nfpm/v2/cmd/nfpm@"$NFPM_VER"
```

<!-- Internal attestation (build side) per D-04 and Pattern 1 -->

```yaml
# In each build-<platform> job:
permissions:
  id-token: write          # attest-build-provenance OIDC signing
  attestations: write      # persist to GH attestations API
  contents: read

steps:
  # ... wails build ...
  # ... (macOS ONLY) tar .app before attest + upload ...
  - name: Attest unsigned artifact (internal)
    id: attest
    uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
    with:
      subject-path: <exact path to unsigned artifact>
  - name: Upload unsigned artifact + attestation bundle
    uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
    with:
      name: <platform-specific name>-unsigned
      path: |
        <artifact path(s)>
        ${{ steps.attest.outputs.bundle-path }}
      if-no-files-found: error
```

<!-- sign-macos verify + sign template -->

```yaml
sign-macos:
  needs: [build-macos]
  runs-on: macos-latest
  environment: release
  permissions:
    contents: read         # gh attestation verify --bundle is offline per RESEARCH Open Q2
  env:
    MACOS_CERTIFICATE: ${{ secrets.MACOS_CERTIFICATE }}
    MACOS_CERTIFICATE_NAME: ${{ secrets.MACOS_CERTIFICATE_NAME }}
    MACOS_CERTIFICATE_PWD: ${{ secrets.MACOS_CERTIFICATE_PWD }}
    MACOS_CI_KEYCHAIN_PWD: ${{ secrets.MACOS_CI_KEYCHAIN_PWD }}
    MACOS_NOTARIZATION_APPLE_ID: ${{ secrets.MACOS_NOTARIZATION_APPLE_ID }}
    MACOS_NOTARIZATION_PWD: ${{ secrets.MACOS_NOTARIZATION_PWD }}
    MACOS_NOTARIZATION_TEAM_ID: ${{ secrets.MACOS_NOTARIZATION_TEAM_ID }}
  steps:
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
        if [[ -z "$BUNDLE" ]]; then echo "No bundle found in artifacts/"; exit 1; fi
        gh attestation verify artifacts/AgentHub.app.tar.gz \
          --repo ${{ github.repository }} \
          --bundle "$BUNDLE"
    - name: Untar .app bundle
      run: |
        mkdir -p build/bin
        tar -xzf artifacts/AgentHub.app.tar.gz -C build/bin/
    # ... existing codesign + notarize + DMG steps (moved from build-macos) ...
    - name: Upload signed DMG
      uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
      with:
        name: macos-dmg
        path: build/bin/agenthub-*.dmg
        if-no-files-found: error
```

<!-- publish job with release attestation + D-02 fix + D-15 draft-on-rc -->

```yaml
publish:
  needs: [build-macos, build-windows, build-linux, sign-macos]
  runs-on: ubuntu-latest
  permissions:
    contents: write          # create/update release
    id-token: write          # release attestation OIDC
    attestations: write      # persist release attestation
  steps:
    - uses: actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1
      with:
        path: artifacts
        merge-multiple: true
    - name: Generate SHA256 checksums
      run: |
        cd artifacts
        sha256sum *.dmg *.exe *.tar.gz *.deb > checksums.txt
        cat checksums.txt
    - name: Release build-provenance attestation
      uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
      with:
        subject-checksums: artifacts/checksums.txt
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
        draft: ${{ contains(github.ref, '-rc') }}   # D-15 hyphen-anchored
        token: ${{ secrets.GITHUB_TOKEN }}          # D-02 — was TAP_DEPLOY_TOKEN
```

<!-- Existing sign/notarize steps to PRESERVE VERBATIM (current release.yml:93-141) -->

```yaml
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
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Restructure build-* jobs — strip secrets, SHA-pin, tar-before-upload for macOS, internal attestation</name>
  <read_first>
    - .github/workflows/release.yml (FULL FILE — 316 lines; this task touches the validate + three build jobs, lines 1-286)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 56-219 — full release.yml pattern list; specifically lines 115-128 for secret removal, 141-162 for tar + internal attest)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Pattern 1 lines 237-349 — three-stage template; Pitfall 1 lines 471-477 — .app tar requirement; Pitfall 4 lines 504-511 — environment-scoping)
  </read_first>
  <files>.github/workflows/release.yml</files>
  <action>
    **Goal:** Restructure the three `build-<platform>` jobs + `validate` job. The `sign-macos` extraction happens in Task 2; `publish` in Task 3. This task focuses on the secret-scope-zero build stage.

    **Edit 1 — `validate` job (lines 9-41):** SHA-pin only. Preserve all logic.
    - Line 13: `actions/checkout@v4` → `actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2`
    - Line 21: `actions/setup-go@v5` → `actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0`
    - Line 29: `pnpm/action-setup@v4` → `pnpm/action-setup@903f9c1a6ebcba6cf41d87230be49611ac97822e # v6.0.3`
    - Line 34: `actions/setup-node@v4` → `actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6.4.0`

    **Edit 2 — `build-macos` job (lines 43-148) — BIGGEST changes. After this edit it becomes the unsigned-build-only job:**

    a. **Remove** `environment: release` (line 46) and the entire `env:` block (lines 47-54 — the seven MACOS_* secrets). Replace with:
    ```yaml
    build-macos:
      needs: [validate]
      runs-on: macos-latest
      permissions:
        id-token: write          # attest-build-provenance OIDC
        attestations: write      # persist to GH attestations API
        contents: read
    ```
    The `env:` block is gone entirely. `environment: release` declaration is gone. This is the SEC-11 core: zero secret access for the build job.

    b. **SHA-pin** the setup actions (lines 57, 62, 67, 72):
    - `actions/checkout@v4` → checkout v6.0.2 SHA (as above)
    - `actions/setup-go@v5` → setup-go v6.4.0 SHA
    - `pnpm/action-setup@v4` → pnpm v6.0.3 SHA
    - `actions/setup-node@v4` → setup-node v6.4.0 SHA

    c. **Replace wails install** (lines 78-79):
    ```yaml
    - name: Install Wails CLI (version from go.mod)
      run: |
        WAILS_VER=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
        [ -n "$WAILS_VER" ] || { echo "wails not pinned in go.mod"; exit 1; }
        go install github.com/wailsapp/wails/v2/cmd/wails@"$WAILS_VER"
    ```

    d. **PRESERVE** the `Ensure frontend/dist exists for embed` step (lines 81-82) and `Build macOS app` step (lines 84-87) and `Copy branded ICNS into bundle` step (lines 89-91). These are the build steps; they stay.

    e. **REMOVE** all signing/notarize/DMG-create/cleanup steps (lines 93-141). These move to `sign-macos` in Task 2. At this task's completion, `build-macos` ends immediately after `Copy branded ICNS into bundle`.

    f. **INSERT after the build + ICNS steps** (before the old upload step):
    ```yaml
    # CRITICAL: tar .app BEFORE upload — actions/upload-artifact zips and strips symlinks/+x bits
    # (90-RESEARCH.md Pitfall 1 lines 471-477)
    - name: Tar .app bundle (preserve symlinks and +x bits)
      run: tar -czf build/bin/AgentHub.app.tar.gz -C build/bin AgentHub.app

    - name: Attest unsigned artifact (internal)
      id: attest
      uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
      with:
        subject-path: build/bin/AgentHub.app.tar.gz
    ```

    g. **REPLACE** the upload-DMG step (lines 143-148) with an upload-tarball-plus-bundle step:
    ```yaml
    - name: Upload unsigned .app bundle + attestation bundle
      uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
      with:
        name: agenthub-darwin-universal-unsigned
        path: |
          build/bin/AgentHub.app.tar.gz
          ${{ steps.attest.outputs.bundle-path }}
        if-no-files-found: error
    ```

    **Edit 3 — `build-windows` job (lines 150-208):**

    a. SHA-pin lines 155, 160, 165, 170, 204:
    - checkout v6.0.2, setup-go v6.4.0, pnpm v6.0.3, setup-node v6.4.0, upload-artifact v7.0.1.

    b. Replace wails install (lines 176-177) with the go-list pattern.

    c. Add `permissions:` block at the job level:
    ```yaml
    build-windows:
      needs: [validate]
      runs-on: windows-latest
      permissions:
        id-token: write
        attestations: write
        contents: read
    ```

    d. **INSERT BEFORE the upload step** (after "Rename artifacts with version"):
    ```yaml
    - name: Attest unsigned Windows artifacts (internal)
      id: attest
      uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
      with:
        subject-path: 'build/bin/agenthub-*-windows-amd64*.exe'
    ```

    e. **MODIFY upload step** (line 204) to also include the attestation bundle:
    ```yaml
    - name: Upload Windows artifacts + attestation bundle
      uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
      with:
        name: windows-artifacts
        path: |
          build/bin/agenthub-*.exe
          ${{ steps.attest.outputs.bundle-path }}
        if-no-files-found: error
    ```

    Note: Windows doesn't need tar — the two `.exe` files are real files, not directories with symlinks.

    **Edit 4 — `build-linux` job (lines 210-286):**

    a. SHA-pin lines 215, 226, 231, 236, 280.

    b. Replace wails install (lines 242-243) with go-list pattern.

    c. Add `permissions:` block at job level (same as windows/macos).

    d. **REPLACE** `nfpm@latest` install inside the "Create .deb via nfpm" step (line 264). Current block (lines 261-277):
    ```yaml
    - name: Create .deb via nfpm
      run: |
        VERSION_BARE="${GITHUB_REF_NAME#v}"
        go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
        cat > nfpm.yaml <<EOF
        ...
    ```

    **Split** the nfpm install into its own step (before the package-create step), using the go-list pattern:
    ```yaml
    - name: Install nfpm (version from go.mod)
      run: |
        NFPM_VER=$(go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2)
        [ -n "$NFPM_VER" ] || { echo "nfpm not pinned in go.mod"; exit 1; }
        go install github.com/goreleaser/nfpm/v2/cmd/nfpm@"$NFPM_VER"

    - name: Create .deb via nfpm
      run: |
        VERSION_BARE="${GITHUB_REF_NAME#v}"
        cat > nfpm.yaml <<EOF
        name: agenthub
        arch: amd64
        platform: linux
        version: "${VERSION_BARE}"
        maintainer: AgentHub <noreply@github.com>
        description: AI coding session manager
        contents:
          - src: build/bin/agenthub
            dst: /usr/bin/agenthub
        EOF
        $(go env GOPATH)/bin/nfpm package --packager deb \
          --target "build/bin/agenthub-${GITHUB_REF_NAME}-linux-amd64.deb"
    ```

    e. **INSERT BEFORE upload step:**
    ```yaml
    - name: Attest unsigned Linux artifacts (internal)
      id: attest
      uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
      with:
        subject-path: |
          build/bin/agenthub-*.tar.gz
          build/bin/agenthub-*.deb
    ```

    f. **MODIFY upload step** (line 280) to include the attestation bundle path (same pattern as windows).

    **At end of this task:** The three build-* jobs are secret-free, SHA-pinned, attest-their-output, and (for macOS only) tar before upload. Task 2 adds `sign-macos`; Task 3 modifies `publish`.
  </action>
  <verify>
    <automated>! grep -E '^\s*environment:\s*release\s*$' .github/workflows/release.yml | head -1 | grep -q build-macos && grep -F 'id-token: write' .github/workflows/release.yml && grep -c 'attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32' .github/workflows/release.yml | (read n; [ "$n" -ge 3 ] && echo "PASS: attestation pinned in >=3 build jobs" || (echo "FAIL: attestation count = $n (expected >= 3)"; exit 1)) && grep -F 'tar -czf build/bin/AgentHub.app.tar.gz' .github/workflows/release.yml && grep -c '@latest' .github/workflows/release.yml | (read n; [ "$n" -eq 0 ] && echo "PASS: no @latest" || (echo "FAIL: @latest still present"; exit 1))</automated>
  </verify>
  <acceptance_criteria>
    - `grep -F 'AgentHub.app.tar.gz' .github/workflows/release.yml` matches (tar pattern present — Pitfall 1 defense)
    - `grep -c '@latest' .github/workflows/release.yml` returns 0 (SEC-10 for build-* jobs; publish is Task 3 but the only @latest cluster was in build-* jobs)
    - `grep -c 'attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0' .github/workflows/release.yml` returns >= 3 (one per build-* job; release attestation added in Task 3 will bring count to >= 4)
    - `grep -c "go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2" .github/workflows/release.yml` returns >= 3 (one per build-* job)
    - `grep -c "go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2" .github/workflows/release.yml` returns 1 (build-linux only)
    - `grep -c 'MACOS_CERTIFICATE:' .github/workflows/release.yml` returns 0 after this task, OR returns 1 if sign-macos already added (but sign-macos is Task 2; this task should complete with 0). **Acceptance for Task 1 is 0.** Task 2 acceptance raises it to 1.
    - `grep -c 'permissions:' .github/workflows/release.yml` returns >= 3 (one per build-* job; publish already had one, sign-macos adds another)
    - `grep -c 'id-token: write' .github/workflows/release.yml` returns >= 3 (attestation OIDC permission on each build job)
    - `grep -c 'attestations: write' .github/workflows/release.yml` returns >= 3
    - YAML valid: `python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/release.yml"))'` exits 0
    - Every `uses:` line in release.yml is a 40-char hex SHA (for lines that exist after Task 1 — validate + three build jobs + publish download-artifact). Negative grep: `grep -E 'uses:\s*[^ ]+@' .github/workflows/release.yml | grep -Ev '@[a-f0-9]{40}(\s|$)'` returns no matches (exit 1).
    - `grep -F 'softprops/action-gh-release@v2' .github/workflows/release.yml` returns 0 (old unpinned ref gone; Task 3 will add the v3.0.0 SHA-pinned version)
  </acceptance_criteria>
  <done>
    build-macos, build-windows, build-linux are all (a) SHA-pinned, (b) secret-free (no MACOS_*/WINGET/TAP/RELEASE_PLEASE env blocks, no environment: release), (c) produce internal attestations via attest-build-provenance, (d) for macOS: tar .app before upload. The sign-macos job does not yet exist; the publish job still has old softprops v2 and TAP_DEPLOY_TOKEN references — those are Task 3 targets. Commit: `ci(90): strip secrets from build-* jobs + SHA-pin + internal attestation + tar .app (SEC-09, SEC-10, SEC-11)`.
  </done>
</task>

<task type="auto">
  <name>Task 2: Add sign-macos job — verify attestation, untar, re-home codesign/notarize steps</name>
  <read_first>
    - .github/workflows/release.yml (CURRENT STATE after Task 1 — should have build-macos ending with upload of tarball+bundle)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 60-113 — the verbatim codesign/notarize blocks that move into sign-macos; lines 285-319 in RESEARCH for the full sign-macos template)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Pattern 1 lines 286-319 — sign-macos skeleton; Pattern 3 lines 369-385 — gh attestation verify; Open Question 2 line 758 — contents: read is sufficient for offline --bundle verify)
  </read_first>
  <files>.github/workflows/release.yml</files>
  <action>
    **Insert a new `sign-macos:` job** between `build-linux` (ends around line 286 before this task) and `publish` (starts around line 288 before this task — though Task 3 will restructure publish, `publish` currently exists with `needs: [build-macos, build-windows, build-linux]`).

    Structure (verbatim from the `<interfaces>` sign-macos block):

    ```yaml
    sign-macos:
      needs: [build-macos]
      runs-on: macos-latest
      environment: release
      permissions:
        contents: read
      env:
        MACOS_CERTIFICATE: ${{ secrets.MACOS_CERTIFICATE }}
        MACOS_CERTIFICATE_NAME: ${{ secrets.MACOS_CERTIFICATE_NAME }}
        MACOS_CERTIFICATE_PWD: ${{ secrets.MACOS_CERTIFICATE_PWD }}
        MACOS_CI_KEYCHAIN_PWD: ${{ secrets.MACOS_CI_KEYCHAIN_PWD }}
        MACOS_NOTARIZATION_APPLE_ID: ${{ secrets.MACOS_NOTARIZATION_APPLE_ID }}
        MACOS_NOTARIZATION_PWD: ${{ secrets.MACOS_NOTARIZATION_PWD }}
        MACOS_NOTARIZATION_TEAM_ID: ${{ secrets.MACOS_NOTARIZATION_TEAM_ID }}
      steps:
        - name: Download unsigned .app bundle + attestation bundle
          uses: actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1
          with:
            name: agenthub-darwin-universal-unsigned
            path: artifacts/

        - name: Verify internal attestation
          env:
            GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          run: |
            BUNDLE=$(ls artifacts/attestation.json artifacts/*.sigstore.json 2>/dev/null | head -1 || true)
            if [[ -z "$BUNDLE" ]]; then
              echo "FAIL: no attestation bundle found in artifacts/"
              ls -la artifacts/
              exit 1
            fi
            echo "Verifying attestation bundle: $BUNDLE"
            gh attestation verify artifacts/AgentHub.app.tar.gz \
              --repo ${{ github.repository }} \
              --bundle "$BUNDLE"

        - name: Untar .app bundle
          run: |
            mkdir -p build/bin
            tar -xzf artifacts/AgentHub.app.tar.gz -C build/bin/

        # ======================================================================
        # Below: codesign/notarize/DMG steps MOVED VERBATIM from old build-macos
        # (previous release.yml lines 93-141). Cryptographic logic unchanged.
        # ======================================================================

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

        - name: Upload signed DMG
          uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
          with:
            name: macos-dmg
            path: build/bin/agenthub-*.dmg
            if-no-files-found: error
    ```

    **Key properties established:**
    - `environment: release` is ONLY on sign-macos (preserves the GH environment protection rules that apply to the MACOS_* secrets — see Pitfall 4 line 504-511 and RESEARCH Open Q4 line 768).
    - Secrets in `env:` block: ONLY the 7 MACOS_* secrets. No WINGET/TAP/RELEASE_PLEASE/GITHUB_TOKEN (the last is scoped per-step via `env: GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` on the verify step, which is the minimum needed for gh attestation verify even in --bundle mode).
    - `permissions: contents: read` is sufficient for offline `--bundle` verify (RESEARCH Open Q2 line 757-761). If the verify step fails in practice with a permissions error, escalate to `attestations: read`.
    - **Bundle filename robustness:** The `ls artifacts/attestation.json artifacts/*.sigstore.json 2>/dev/null | head -1` pattern handles both the legacy `attestation.json` name and the newer `*.sigstore.json` conventions that `actions/attest-build-provenance` emits. The `|| true` + explicit empty-string check prevents `set -e` from masking a missing-bundle condition.

    **Do not remove** the existing codesign/notarize step comments or step names. The step names are what surface in the GitHub Actions UI; preserve them for continuity with historical runs.
  </action>
  <verify>
    <automated>grep -F 'sign-macos:' .github/workflows/release.yml && grep -c 'needs: \[build-macos\]' .github/workflows/release.yml | (read n; [ "$n" -ge 1 ] && echo "PASS: sign-macos has correct needs" || (echo "FAIL: sign-macos needs missing"; exit 1)) && grep -F 'gh attestation verify artifacts/AgentHub.app.tar.gz' .github/workflows/release.yml && grep -F 'MACOS_CERTIFICATE:' .github/workflows/release.yml && grep -c 'environment: release' .github/workflows/release.yml | (read n; [ "$n" -eq 1 ] && echo "PASS: environment: release appears exactly once (sign-macos only)" || (echo "FAIL: environment: release appears $n times"; exit 1))</automated>
  </verify>
  <acceptance_criteria>
    - `grep -c '^\s*sign-macos:' .github/workflows/release.yml` returns 1 (new job exists)
    - `grep -F 'needs: [build-macos]' .github/workflows/release.yml` matches (exactly — sign-macos depends only on build-macos, NOT the other build jobs)
    - `grep -c 'environment: release' .github/workflows/release.yml` returns 1 (appears ONLY in sign-macos, not in build-macos or publish — Pitfall 4 resolution)
    - `grep -F 'gh attestation verify artifacts/AgentHub.app.tar.gz' .github/workflows/release.yml` matches at least once
    - `grep -F 'tar -xzf artifacts/AgentHub.app.tar.gz' .github/workflows/release.yml` matches (the sign-side untar)
    - `grep -c 'MACOS_CERTIFICATE:' .github/workflows/release.yml` returns 1 (only in sign-macos env block — NOT in any build job)
    - `grep -c 'MACOS_NOTARIZATION_TEAM_ID:' .github/workflows/release.yml` returns 1 (same)
    - `grep -F 'codesign --deep --force --verbose' .github/workflows/release.yml` matches exactly 1 (moved from build-macos to sign-macos — not duplicated)
    - `grep -F 'xcrun notarytool submit' .github/workflows/release.yml` matches exactly 1
    - `grep -F 'xcrun stapler staple' .github/workflows/release.yml` matches exactly 1
    - `grep -F 'create-dmg' .github/workflows/release.yml` matches at least 1 (exact count may vary — brew install line + invocation line — but not 0)
    - `grep -F 'name: macos-dmg' .github/workflows/release.yml` matches (upload step naming for the signed DMG artifact)
    - YAML valid: `python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/release.yml"))'` exits 0
    - All `uses:` lines in sign-macos are 40-char SHA: `grep -E 'uses:\s*[^ ]+@' .github/workflows/release.yml | grep -Ev '@[a-f0-9]{40}(\s|$)'` returns 0 matches
  </acceptance_criteria>
  <done>
    sign-macos job exists with environment: release, MACOS_* env block, attestation verify step, untar step, all codesign/notarize/DMG steps moved verbatim from the old build-macos location, and a signed-DMG upload. build-macos is now purely an unsigned-build producer with attestation. Commit: `ci(90): add sign-macos job — verify attestation + sign + notarize (SEC-11 D-01 + D-04)`.
  </done>
</task>

<task type="auto">
  <name>Task 3: Restructure publish job — add sign-macos to needs, fix TAP_DEPLOY_TOKEN, add release attestation, add rc draft guard, SHA-pin</name>
  <read_first>
    - .github/workflows/release.yml (CURRENT STATE after Tasks 1 + 2 — publish job still has old softprops@v2 and TAP_DEPLOY_TOKEN)
    - .planning/phases/90-release-pipeline-hardening/90-PATTERNS.md (lines 160-218 — publish modifications; lines 680-693 — rc-tag guards)
    - .planning/phases/90-release-pipeline-hardening/90-RESEARCH.md (Pattern 2 lines 355-367 — release attestation + subject-checksums; D-15 line 41)
    - .planning/phases/90-release-pipeline-hardening/90-CONTEXT.md (D-02 — TAP_DEPLOY_TOKEN misuse fix; D-05 — release attestation; D-15 — draft on rc)
  </read_first>
  <files>.github/workflows/release.yml</files>
  <action>
    **Modify the `publish:` job** (currently the last job, around the original lines 288-316).

    **Change 1 — `needs:`** Add `sign-macos` to the dependency list:
    ```yaml
    # BEFORE
    needs: [build-macos, build-windows, build-linux]

    # AFTER
    needs: [build-macos, build-windows, build-linux, sign-macos]
    ```
    This ensures publish runs AFTER sign-macos produces the signed DMG.

    **Change 2 — `permissions:`** Extend for release attestation:
    ```yaml
    # BEFORE
    permissions:
      contents: write

    # AFTER
    permissions:
      contents: write          # create/update release
      id-token: write          # release attestation OIDC
      attestations: write      # persist release attestation to GH API
    ```

    **Change 3 — SHA-pin download-artifact:**
    ```yaml
    # BEFORE (line 295)
    uses: actions/download-artifact@v4

    # AFTER
    uses: actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1
    ```

    **Change 4 — Tighten checksums step** (line 300-304):
    Current:
    ```yaml
    - name: Generate SHA256 checksums
      run: |
        cd artifacts
        sha256sum * > checksums.txt
        cat checksums.txt
    ```

    Replace with a more targeted glob so the attestation bundle JSONs (uploaded alongside the artifacts by each build job) are NOT included in `checksums.txt`:
    ```yaml
    - name: Generate SHA256 checksums
      run: |
        cd artifacts
        # Explicit globs so attestation bundle .json files are excluded
        sha256sum *.dmg *.exe *.tar.gz *.deb > checksums.txt
        cat checksums.txt
    ```
    Note: `merge-multiple: true` on the download step flattens all artifacts into a single `artifacts/` directory, so attestation bundles (named `attestation.json` or `*.sigstore.json`) and the actual release artifacts (`.dmg`, `.exe`, `.tar.gz`, `.deb`) share a directory. The shell glob must be specific.

    **Change 5 — Add release attestation step (D-05)** BEFORE the upload-to-release step:
    ```yaml
    - name: Release build-provenance attestation
      uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
      with:
        subject-checksums: artifacts/checksums.txt
    ```
    The `subject-checksums` input attests every file listed in checksums.txt in one action call (90-RESEARCH.md Pattern 2 lines 365-367). Downstream consumers run `gh attestation verify <file> --owner scottkw` to verify.

    **Change 6 — Replace upload-to-release step entirely** (lines 306-316 before edit):

    Before:
    ```yaml
    - name: Upload to GitHub Release
      uses: softprops/action-gh-release@v2
      with:
        files: |
          artifacts/*.dmg
          artifacts/*.exe
          artifacts/*.tar.gz
          artifacts/*.deb
          artifacts/checksums.txt
        fail_on_unmatched_files: true
        token: ${{ secrets.TAP_DEPLOY_TOKEN }}
    ```

    After (three changes: SHA-pin, add `draft:` with hyphen-anchored rc check, swap token D-02):
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
        draft: ${{ contains(github.ref, '-rc') }}
        token: ${{ secrets.GITHUB_TOKEN }}
    ```

    **The three critical properties established in this step:**
    1. **D-15** — `draft: ${{ contains(github.ref, '-rc') }}`: rc tags (e.g., `v3.1.0-rc1`) produce draft releases. Real tags (e.g., `v3.1.0`) produce public releases. Hyphen-anchored per RESEARCH anti-pattern line 436 — NOT `'rc'`.
    2. **D-02** — token swapped from `TAP_DEPLOY_TOKEN` to `GITHUB_TOKEN`. TAP_DEPLOY_TOKEN is a PAT scoped for the Homebrew tap repo push in `distribute.yml`; it has no business being the release-upload token here. `GITHUB_TOKEN` is the ephemeral per-run token with `contents: write` from the permissions block above — minimum necessary.
    3. **SEC-09** — softprops/action-gh-release@v3.0.0 SHA-pinned.

    **At end of Task 3:** release.yml has validate → build-macos/build-windows/build-linux → sign-macos → publish, with every secret scoped per job, internal + release attestations, rc-draft guard, SHA-pinned everywhere, `@latest`-free, and the TAP_DEPLOY_TOKEN misuse fixed.

    **Final self-check against grep-gate:** run `bash scripts/grep-gate.sh`. It should now pass for `.github/workflows/release.yml` — the remaining floating refs live only in `distribute.yml` (owned by Plan 05).
  </action>
  <verify>
    <automated>grep -F 'needs: [build-macos, build-windows, build-linux, sign-macos]' .github/workflows/release.yml && grep -F 'token: ${{ secrets.GITHUB_TOKEN }}' .github/workflows/release.yml && grep -F 'draft: ${{ contains(github.ref, ''-rc'') }}' .github/workflows/release.yml && grep -F 'softprops/action-gh-release@b4309332981a82ec1c5618f44dd2e27cc8bfbfda # v3.0.0' .github/workflows/release.yml && grep -F 'subject-checksums: artifacts/checksums.txt' .github/workflows/release.yml && (grep -c 'TAP_DEPLOY_TOKEN' .github/workflows/release.yml | (read n; [ "$n" -eq 0 ] && echo "PASS: TAP_DEPLOY_TOKEN gone" || (echo "FAIL: TAP_DEPLOY_TOKEN still present"; exit 1)))</automated>
  </verify>
  <acceptance_criteria>
    - `grep -F 'needs: [build-macos, build-windows, build-linux, sign-macos]' .github/workflows/release.yml` matches (publish depends on sign-macos)
    - `grep -c 'TAP_DEPLOY_TOKEN' .github/workflows/release.yml` returns 0 (D-02 fix complete)
    - `grep -F 'token: ${{ secrets.GITHUB_TOKEN }}' .github/workflows/release.yml` matches at least once (new release-upload token)
    - `grep -F "draft: \${{ contains(github.ref, '-rc') }}" .github/workflows/release.yml` matches (hyphen-anchored — NOT `'rc'`)
    - `grep -c "contains(github.ref, 'rc')" .github/workflows/release.yml` returns 0 (no non-hyphen-anchored variant slipped in — anti-pattern defense)
    - `grep -F 'softprops/action-gh-release@b4309332981a82ec1c5618f44dd2e27cc8bfbfda # v3.0.0' .github/workflows/release.yml` matches
    - `grep -F 'subject-checksums: artifacts/checksums.txt' .github/workflows/release.yml` matches (release attestation input)
    - `grep -c 'attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0' .github/workflows/release.yml` returns 4 (one per build-* job, plus release attestation in publish)
    - `grep -F 'actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1' .github/workflows/release.yml` matches (download-artifact SHA-pinned)
    - `grep -c 'id-token: write' .github/workflows/release.yml` returns 4 (3 build jobs + publish)
    - `grep -c 'attestations: write' .github/workflows/release.yml` returns 4
    - `grep -c '@latest' .github/workflows/release.yml` returns 0
    - `grep -cE 'uses:\s*[^ ]+@[a-f0-9]{40}(\s|$)' .github/workflows/release.yml` returns >= 14 (every use must be a 40-char SHA; count is the "at-least" floor given validate job has 4, each build job has 4-5, sign-macos has 2, publish has 3 = roughly 14-18 total)
    - `grep -E 'uses:\s*[^ ]+@' .github/workflows/release.yml | grep -Ev '@[a-f0-9]{40}(\s|$)'` returns 0 matches (negative SHA-pin check — every `uses:` is a 40-char SHA)
    - `bash scripts/grep-gate.sh` exits 0 when run ONLY against .github/workflows/release.yml (but WILL STILL FAIL overall because distribute.yml is Plan 05's territory — this is expected until Wave 4 completes)
    - YAML valid end-to-end: `python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/release.yml"))'` exits 0
  </acceptance_criteria>
  <done>
    publish job: depends on sign-macos, attests final artifacts via subject-checksums, uses hyphen-anchored rc-draft toggle, uses GITHUB_TOKEN (not TAP_DEPLOY_TOKEN), and SHA-pins softprops + download-artifact. release.yml end-to-end is SEC-09/10/11 compliant for its scope. Commit: `ci(90): restructure publish — attestation + rc-draft + GITHUB_TOKEN fix (SEC-09 + SEC-11 D-02 + D-05 + D-15)`.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Untrusted build code → signing secrets | The SEC-11 boundary. `wails build` invokes `npm install` which runs arbitrary postinstall scripts from transitively-trusted packages. Before this plan: that code ran in a job with MACOS signing material in `env:`. After: that code runs in a job with literally no secret access (`env:` empty, no `environment:` declaration). |
| build-macos → sign-macos artifact handoff | Cross-job upload/download is transport; internal attestation is the integrity proof. Sigstore ephemeral cert via OIDC proves the attestation was issued by the correct GitHub repo + workflow + job. |
| publish → public release artifacts | Release attestation (D-05) is the public proof for downstream consumers (Homebrew tap users, corporate scanners, `gh attestation verify` CLI users). |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-90-17 | Information Disclosure | Compromised transitive dep in `wails build` exfiltrates MACOS_CERTIFICATE | mitigate | SEC-11 core: `build-macos` `env:` block is empty and job has no `environment:` declaration. Attacker code has no secret to exfiltrate. |
| T-90-18 | Spoofing | Malicious agent between build-macos and sign-macos substitutes the `.app.tar.gz` | mitigate | D-04 internal attestation: build-macos signs the tar hash via Sigstore OIDC; sign-macos calls `gh attestation verify --bundle` before untar. Substitution is detected because the SHA of the new tar does not match the attested hash. The attestation is cryptographically bound to the exact GitHub repo, workflow file path, job name, and run number — cross-repo substitution also fails. |
| T-90-19 | Tampering | Attacker adds `upload-artifact` permissions to a different job and uploads a forged bundle with the same artifact name | mitigate | Artifact naming is scoped to the job's `name:` — sign-macos downloads specifically `agenthub-darwin-universal-unsigned`. A forger would also need the Sigstore cert issued to the matching job/run; the OIDC token binding prevents cross-job forgery. |
| T-90-20 | Repudiation | Disputed release — was it built from the claimed commit? | mitigate | D-05 release attestation binds every file in `checksums.txt` to the exact commit SHA via Sigstore. Any verifier (including third parties) can run `gh attestation verify agenthub-v3.1.0.dmg --owner scottkw` and get a signed predicate that says "this file was built from scottkw/agenthub@<sha> by job <name> on <date>". |
| T-90-21 | Elevation of Privilege | `sign-macos` granted more permissions than needed | mitigate | sign-macos has `permissions: contents: read` only. `env:` contains only MACOS_* secrets. No attestations: write (not producing attestation — only verifying). RESEARCH Open Q2 confirms offline `--bundle` verify works with contents:read. |
| T-90-22 | Elevation of Privilege | `publish` reuses `TAP_DEPLOY_TOKEN` as release-upload PAT (pre-D-02) | mitigate | D-02 fix: publish now uses `secrets.GITHUB_TOKEN` (ephemeral per-run, `contents: write` via permissions block). TAP_DEPLOY_TOKEN stays in `distribute.yml` where it's legitimately needed for tap repo push. |
| T-90-23 | Information Disclosure | `wingetcreate` or other tool in publish leaks secrets via logs | accept | Publish job only has `GITHUB_TOKEN` (ephemeral) + OIDC token for attestation (also ephemeral, bound to run). Even complete log exfiltration is time-boxed to the run duration. |
| T-90-24 | Tampering | Composite action transitive dep drift (Pitfall 3) | accept | `softprops/action-gh-release@v3.0.0` is `runs: using: "node24"` — no nested third-party actions (RESEARCH line 501). `actions/attest-build-provenance@v4.1.0` internally uses `actions/attest@v4.1.0` which IS SHA-pinned in its action.yml (RESEARCH line 501 verified). Future drift monitored via Dependabot. |
| T-90-25 | Elevation of Privilege | `draft: contains(github.ref, 'rc')` (missing hyphen) false-positives non-rc tags like `v3.1.0-archive` | mitigate | Hyphen-anchored form `'-rc'` enforced in Task 3 acceptance criteria. Grep-gate for the non-hyphen form included in acceptance. |

**Residual risk:**
- **`continue-on-error` on `submit-winget`:** Not in release.yml; that job lives in distribute.yml (Plan 05). Carried forward risk because the `continue-on-error: true` there is load-bearing for first-submission flakiness per RESEARCH line 437.
- **Upload-artifact v4 → v7 bump (A1):** New wire format; tested only against `.exe`/`.tar.gz`/`.deb` (which are simple files); the `.app.tar.gz` is also a simple file so should be unaffected. If v7 breaks, fallback is the pinned v4 SHA `11bd71901bbe5b1630ceea73d27597364c9af683` (90-RESEARCH.md line 107 footnote).
- **Environment protection rules (A6):** If repo's `release` environment has required-reviewer or wait-timer rules, moving the environment declaration from build-macos to sign-macos changes when those rules fire. RESEARCH Open Q4 recommends `gh api /repos/scottkw/agenthub/environments/release` to audit before running the rc tag. Documented in Plan 06's pre-flight section.
</threat_model>

<verification>
After all three tasks land:
1. **YAML validity:** `python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/release.yml"))'` exits 0
2. **Secret scoping:**
   - `grep -B1 'MACOS_CERTIFICATE' .github/workflows/release.yml | grep 'build-macos:'` returns nothing (MACOS_* not in build-macos)
   - `grep -c 'environment: release' .github/workflows/release.yml` returns 1 (sign-macos only)
   - `grep -c 'TAP_DEPLOY_TOKEN' .github/workflows/release.yml` returns 0 (D-02)
3. **SHA pinning:** `grep -E 'uses:\s*[^ ]+@' .github/workflows/release.yml | grep -Ev '@[a-f0-9]{40}(\s|$)'` returns 0 matches
4. **@latest removal:** `grep -c '@latest' .github/workflows/release.yml` returns 0
5. **Attestation scaffolding:**
   - `grep -c 'attest-build-provenance' .github/workflows/release.yml` returns 4 (3 build + 1 release)
   - `grep -c 'gh attestation verify' .github/workflows/release.yml` returns 1 (sign-macos)
6. **rc-draft anchoring:** `grep -F "contains(github.ref, '-rc')" .github/workflows/release.yml` matches
7. **Tar-before-upload:** `grep -F 'tar -czf build/bin/AgentHub.app.tar.gz' .github/workflows/release.yml` matches
</verification>

<success_criteria>
- release.yml has four jobs post-validate: build-macos (secret-free), build-windows (secret-free), build-linux (secret-free), sign-macos (MACOS_* only, environment: release), publish (GITHUB_TOKEN + attestations scopes)
- Zero floating refs, zero @latest
- Internal attestation on every build-* job; release attestation on publish via subject-checksums
- TAP_DEPLOY_TOKEN removed from release.yml entirely (D-02 fixed)
- rc-draft uses hyphen-anchored guard (D-15)
- .app is tarred before upload and untarred before signing (Pitfall 1 resolved)
- Handoff to Plan 05: distribute.yml untouched by this plan — Plan 05 can run after (Wave 4)
</success_criteria>

<output>
After completion, create `.planning/phases/90-release-pipeline-hardening/90-04-SUMMARY.md` documenting:
- File modified: `.github/workflows/release.yml` — 5 jobs (validate + 3 build + sign-macos + publish), every uses: SHA-pinned, attestations end-to-end, D-02/D-04/D-05/D-15 resolved
- New job added: sign-macos
- Old job modified: build-macos (stripped secrets, added attestation, added tar); build-windows and build-linux (SHA-pin + attestation); publish (SHA-pin + attestation + draft + token fix)
- Grep-gate state: fails ONLY on distribute.yml (Plan 05's territory)
- Handoff to Plan 05: distribute.yml is untouched; can proceed immediately (Wave 4)
</output>
