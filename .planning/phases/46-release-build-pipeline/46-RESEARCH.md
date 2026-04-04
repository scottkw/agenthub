# Phase 46: Release Build Pipeline - Research

**Researched:** 2026-04-04
**Domain:** GitHub Actions release workflow, macOS codesign/notarize, DMG creation, Linux deb/tar.gz packaging, SHA256 checksums
**Confidence:** HIGH

## Summary

Phase 46 creates `.github/workflows/release.yml` — a tag-triggered multi-platform build workflow that produces signed/notarized macOS DMGs, Windows NSIS installers + bare EXE, Linux amd64 tar.gz + deb packages, and a `checksums.txt` file, all attached to the GitHub Release that release-please created when it pushed the tag.

The critical architecture finding: `dAppServer/wails-build-action` does NOT produce DMG files — it produces `.app`, `.app.zip`, and `.pkg`. Phase 46 must therefore use `wails-build-action` for the raw `.app` bundle and then create the DMG in a separate step using the `create-dmg` shell script (installable via `brew install create-dmg` on `macos-latest` runners). The signed `.app` is built first, then wrapped into a DMG, then the DMG itself is signed (DMG signing is separate from app signing).

The trigger mechanism is verified: `RELEASE_PLEASE_TOKEN` is a classic PAT (confirmed in repo secrets). Tags pushed by release-please using a PAT — not `GITHUB_TOKEN` — DO trigger `on.push.tags` workflows. The existing Release PR for v1.8.0 is already open on GitHub; merging it will push tag `v1.8.0` and trigger `release.yml`.

The 7 macOS signing secrets are stored in the `release` GitHub environment (confirmed in Phase 45 research). The `release.yml` job MUST declare `environment: release` to access them.

**Primary recommendation:** Single `release.yml` workflow with parallel platform jobs (macOS, Windows, Linux) feeding into a final `publish` job that assembles checksums and uploads all artifacts to the GitHub Release using `softprops/action-gh-release@v2`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REL-02 | release.yml workflow builds multi-platform artifacts on tag push (macOS signed/notarized DMG, Windows EXE + NSIS installer, Linux amd64 tar.gz + deb) | Verified: wails-build-action produces .app/.exe/binary; DMG via create-dmg; deb via nfpm; tar.gz via standard tar |
| REL-04 | SHA256 checksums file generated and attached to each GitHub Release | Verified: sha256sum (Linux) or shasum -a 256 (macOS) on all artifacts; softprops/action-gh-release@v2 uploads checksums.txt alongside binaries |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| dAppServer/wails-build-action | @main | Build Wails app per platform | Handles CGo cross-compilation, webkit deps, NSIS packaging; already used in build.yml |
| softprops/action-gh-release | v2.6.1 | Create/update GitHub Release and upload assets | De facto standard for release uploads; updates existing release by tag name |
| create-dmg | latest (brew) | Wrap signed .app into a DMG | Standard macOS distribution format; supports --codesign to sign the DMG itself |
| nfpm | latest | Create .deb from Linux binary | No-dependency Go binary; purpose-built for simple packaging without debian/ tree |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| actions/upload-artifact | v4 | Pass artifacts between jobs | Each platform job uploads; publish job downloads all |
| actions/download-artifact | v4 | Collect all platform artifacts in publish job | Needed when jobs run on separate runners |
| actions/checkout | v4 | Checkout code | Standard; already used |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| create-dmg (brew) | hdiutil directly | create-dmg handles DMG signing + retry-on-resource-busy automatically; hdiutil alone does not sign the DMG |
| nfpm | dpkg-deb / debian/ tree | nfpm requires zero debian/ boilerplate; dpkg-deb requires full control file structure |
| softprops/action-gh-release | actions/upload-release-asset | softprops handles glob patterns and existing release updates; upload-release-asset is deprecated |
| on.push.tags | on.release: [published] | release-please pushes tag first, creates GitHub Release second; on.push.tags fires immediately on tag push via PAT, before GitHub Release exists; on.release:published fires after release exists — using release.outputs from release-please-action avoids this ambiguity entirely but requires same-workflow chaining |

**Installation (CI):**
```bash
# macOS runner - create-dmg
brew install create-dmg

# Linux runner - nfpm
curl -sfL https://install.goreleaser.com/github.com/goreleaser/nfpm.sh | sh -s -- -b /usr/local/bin
# or: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
```

## Architecture Patterns

### Recommended Workflow Structure
```
.github/workflows/
├── release.yml          # New: tag-triggered multi-platform release
├── release-please.yml   # Existing: creates Release PR and pushes tag
└── build.yml            # Existing: PR CI (no signing)
```

### Pattern 1: Trigger Strategy — on.push.tags with PAT

**What:** `release.yml` triggers on `v*` tag pushes. Since `RELEASE_PLEASE_TOKEN` is a PAT (not GITHUB_TOKEN), tags pushed by release-please DO trigger this workflow.

**Verification:** `RELEASE_PLEASE_TOKEN` confirmed as repo-level secret (gh api output). Tags pushed with a PAT bypass the recursive-workflow protection that applies to GITHUB_TOKEN only.

**Key constraint:** The `publish` job that calls `softprops/action-gh-release` must wait for all platform jobs. Use `needs: [build-macos, build-windows, build-linux]`.

```yaml
# Source: https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows
on:
  push:
    tags:
      - 'v*'
```

### Pattern 2: macOS Job — Build, Sign, Notarize, DMG

**What:** macOS job runs on `macos-latest`, declares `environment: release` to access the 7 secrets, uses `wails-build-action` to build the `.app`, then manually signs/notarizes/creates DMG.

**Critical:** `wails-build-action` has its own `sign` input but uses a different secret naming convention. Since this project's secrets are named `MACOS_CERTIFICATE` (not `sign-macos-app-cert` pattern), it is cleaner to handle signing manually after the Wails build step rather than passing secrets to the action's signing inputs. This also gives full control over DMG creation.

**Signing secret mapping (from `release` environment):**
| GitHub Secret | Shell Variable | Usage |
|---------------|---------------|-------|
| `MACOS_CERTIFICATE` | `MACOS_CERTIFICATE` | Base64-encoded .p12 |
| `MACOS_CERTIFICATE_NAME` | `MACOS_CERTIFICATE_NAME` | Identity for codesign -s |
| `MACOS_CERTIFICATE_PWD` | `MACOS_CERTIFICATE_PWD` | .p12 export password |
| `MACOS_CI_KEYCHAIN_PWD` | `MACOS_CI_KEYCHAIN_PWD` | Temp keychain password |
| `MACOS_NOTARIZATION_APPLE_ID` | `MACOS_NOTARIZATION_APPLE_ID` | Apple ID email |
| `MACOS_NOTARIZATION_PWD` | `MACOS_NOTARIZATION_PWD` | App-specific password |
| `MACOS_NOTARIZATION_TEAM_ID` | `MACOS_NOTARIZATION_TEAM_ID` | Team ID |

```yaml
# Source: https://federicoterzi.com/blog/automatic-code-signing-and-notarization-for-macos-apps-using-github-actions/
build-macos:
  runs-on: macos-latest
  environment: release
  env:
    MACOS_CERTIFICATE: ${{ secrets.MACOS_CERTIFICATE }}
    MACOS_CERTIFICATE_NAME: ${{ secrets.MACOS_CERTIFICATE_NAME }}
    MACOS_CERTIFICATE_PWD: ${{ secrets.MACOS_CERTIFICATE_PWD }}
    MACOS_CI_KEYCHAIN_PWD: ${{ secrets.MACOS_CI_KEYCHAIN_PWD }}
    MACOS_NOTARIZATION_APPLE_ID: ${{ secrets.MACOS_NOTARIZATION_APPLE_ID }}
    MACOS_NOTARIZATION_PWD: ${{ secrets.MACOS_NOTARIZATION_PWD }}
    MACOS_NOTARIZATION_TEAM_ID: ${{ secrets.MACOS_NOTARIZATION_TEAM_ID }}
  steps:
    - uses: actions/checkout@v4
      with:
        submodules: true

    - name: Build macOS app with Wails
      uses: dAppServer/wails-build-action@main
      with:
        build-name: agenthub
        build-platform: darwin/universal
        # sign: false (default) — we handle signing manually below

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
          build/bin/agenthub.app

    - name: Notarize .app
      run: |
        xcrun notarytool store-credentials "agenthub-notarize" \
          --apple-id "$MACOS_NOTARIZATION_APPLE_ID" \
          --team-id "$MACOS_NOTARIZATION_TEAM_ID" \
          --password "$MACOS_NOTARIZATION_PWD"
        ditto -c -k --keepParent build/bin/agenthub.app build/bin/agenthub-notarize.zip
        xcrun notarytool submit build/bin/agenthub-notarize.zip \
          --keychain-profile "agenthub-notarize" \
          --wait
        xcrun stapler staple build/bin/agenthub.app

    - name: Create and sign DMG
      run: |
        brew install create-dmg
        VERSION="${GITHUB_REF_NAME}"  # e.g. v1.8.0
        DMG_NAME="agenthub-${VERSION}-darwin-universal.dmg"
        create-dmg \
          --volname "AgentHub" \
          --codesign "$MACOS_CERTIFICATE_NAME" \
          "${DMG_NAME}" \
          build/bin/agenthub.app
        mv "${DMG_NAME}" build/bin/

    - name: Cleanup keychain
      if: always()
      run: |
        security delete-keychain build.keychain || true
        rm -f certificate.p12 build/bin/agenthub-notarize.zip

    - uses: actions/upload-artifact@v4
      with:
        name: macos-dmg
        path: build/bin/agenthub-*.dmg
        if-no-files-found: error
```

### Pattern 3: Windows Job — NSIS Installer + EXE

**What:** Windows job runs on `windows-latest`, uses `wails-build-action` with `nsis: true` to produce both the installer and bare EXE.

```yaml
build-windows:
  runs-on: windows-latest
  steps:
    - uses: actions/checkout@v4
      with:
        submodules: true

    - name: Build Windows artifacts with Wails
      uses: dAppServer/wails-build-action@main
      with:
        build-name: agenthub
        build-platform: windows/amd64
        build-webview2: embed
        nsis: true

    - name: Rename artifacts with version
      shell: bash
      run: |
        VERSION="${GITHUB_REF_NAME}"
        mv build/bin/agenthub-amd64-installer.exe \
           "build/bin/agenthub-${VERSION}-windows-amd64-installer.exe"
        mv build/bin/agenthub.exe \
           "build/bin/agenthub-${VERSION}-windows-amd64.exe"

    - uses: actions/upload-artifact@v4
      with:
        name: windows-artifacts
        path: build/bin/agenthub-*.exe
        if-no-files-found: error
```

### Pattern 4: Linux Job — tar.gz + deb

**What:** Linux job runs on `ubuntu-latest`, uses `wails-build-action` to build the binary, then creates a tar.gz and a .deb via nfpm.

**Note on webkit tag:** The `build.yml` uses `webkit2_41` tag for ubuntu-latest to match the webkit2gtk-4.1 API. The same tag must be used in release.yml. The ubuntu-22.04 build in build.yml is for validation purposes; release.yml only ships ubuntu-latest (GTK4 / webkit2gtk-4.1) artifacts as the primary Linux target.

```yaml
build-linux:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
      with:
        submodules: true

    - name: Install Linux dependencies
      run: |
        sudo apt-get update
        sudo apt-get install -y build-essential pkg-config \
          libgtk-3-dev libwebkit2gtk-4.1-dev

    - name: Build Linux binary with Wails
      uses: dAppServer/wails-build-action@main
      with:
        build-name: agenthub
        build-platform: linux/amd64
        build-flags: -tags webkit2_41

    - name: Create tar.gz
      run: |
        VERSION="${GITHUB_REF_NAME}"
        TAR_NAME="agenthub-${VERSION}-linux-amd64.tar.gz"
        tar -czf "${TAR_NAME}" -C build/bin agenthub
        mv "${TAR_NAME}" build/bin/

    - name: Create .deb via nfpm
      run: |
        VERSION_BARE="${GITHUB_REF_NAME#v}"  # strip leading 'v'
        go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
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

    - uses: actions/upload-artifact@v4
      with:
        name: linux-artifacts
        path: |
          build/bin/agenthub-*.tar.gz
          build/bin/agenthub-*.deb
        if-no-files-found: error
```

### Pattern 5: Publish Job — Checksums + GitHub Release Upload

**What:** Final job downloads all platform artifacts, generates SHA256 checksums, and uploads everything to the GitHub Release that release-please created when it pushed the tag.

**Key insight:** release-please creates the GitHub Release before pushing the tag (it does both in sequence). `softprops/action-gh-release` with no `tag_name` defaults to `github.ref_name` — it will find the existing release and append assets to it.

```yaml
publish:
  needs: [build-macos, build-windows, build-linux]
  runs-on: ubuntu-latest
  permissions:
    contents: write
  steps:
    - uses: actions/download-artifact@v4
      with:
        path: artifacts
        merge-multiple: true  # flatten into single directory

    - name: Generate SHA256 checksums
      run: |
        cd artifacts
        sha256sum * > checksums.txt
        cat checksums.txt

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
```

### Pattern 6: Artifact Naming Convention

All artifacts follow: `agenthub-{VERSION}-{platform}-{arch}.{ext}`

| Artifact | Example |
|----------|---------|
| macOS DMG | `agenthub-v1.8.0-darwin-universal.dmg` |
| Windows installer | `agenthub-v1.8.0-windows-amd64-installer.exe` |
| Windows bare EXE | `agenthub-v1.8.0-windows-amd64.exe` |
| Linux tar.gz | `agenthub-v1.8.0-linux-amd64.tar.gz` |
| Linux deb | `agenthub-v1.8.0-linux-amd64.deb` |
| Checksums | `checksums.txt` |

`GITHUB_REF_NAME` in a tag-triggered workflow is the tag name (e.g., `v1.8.0`). Use it directly for the filename.

### Anti-Patterns to Avoid

- **Not declaring `environment: release` in macOS job:** All 7 `MACOS_*` secrets are scoped to the `release` environment — without this declaration, they resolve to empty strings and signing silently fails (same trap as in `build.yml`).
- **Using `GITHUB_TOKEN` as the token in `softprops/action-gh-release`:** The default `GITHUB_TOKEN` has `contents: write` when explicitly granted in permissions, so this actually works — the token does NOT need to be a PAT for uploading to an existing release, only for creating new workflow runs.
- **Creating DMG before signing .app:** Apple notarization staples a ticket to the `.app`. If you create the DMG before stapling, the DMG contains an un-stapled app. Always sign, notarize, staple, then DMG.
- **Using `zip -r` instead of `ditto` for notarization:** `zip -r` does not preserve macOS extended attributes and resource forks. `ditto -c -k --keepParent` is the correct tool.
- **Skipping `--timestamp` flag on codesign:** Without `--timestamp`, the signature has no Trusted Timestamp Authority record and is effectively invalid after the certificate expires.
- **Not using `security set-key-partition-list`:** Without this, codesign may fail with "User interaction is not allowed" on CI runners even after certificate import.
- **Stripping or not stripping `v` prefix for deb version:** nfpm requires `version` to start with a digit (e.g., `1.8.0` not `v1.8.0`). Strip the leading `v` when writing the nfpm.yaml.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| DMG creation | Custom hdiutil script | create-dmg | Handles resource-busy errors, retry logic, DMG signing in one command |
| .deb packaging | debian/ control files + dpkg-buildpackage | nfpm | No debian/ tree, no dpkg toolchain, single binary, works on any runner |
| Release upload | gh CLI + for loop | softprops/action-gh-release@v2 | Glob pattern support, existing release update, idempotent re-runs |
| SHA256 generation | openssl dgst | sha256sum (Linux) / shasum -a 256 (macOS) | Both are pre-installed on all GitHub runners; sha256sum output is compatible with `sha256sum --check` |
| Certificate import | Custom security commands | Direct security commands (no external action needed) | 5 commands; import-codesign-certs action is optional convenience |

**Key insight:** The macOS signing pipeline has 3 distinct phases: (1) sign the .app, (2) notarize the .app (requires network round-trip to Apple), (3) create and sign the DMG. Each phase must complete before the next starts. No tool combines all three.

## Common Pitfalls

### Pitfall 1: `environment: release` Omitted from macOS Job
**What goes wrong:** All `MACOS_*` env vars are empty strings; codesign fails with "no identity found" or the DMG is unsigned.
**Why it happens:** The 7 signing secrets are scoped to the `release` GitHub environment. Jobs without `environment: release` cannot access environment secrets, only repo-level secrets.
**How to avoid:** Add `environment: release` to the macOS job declaration.
**Warning signs:** codesign exits with error about missing identity; `security find-identity` shows empty list.

### Pitfall 2: DMG Created Before .app is Notarized
**What goes wrong:** The DMG contains an unstapled .app; macOS Gatekeeper quarantines it on download even though the .app was notarized.
**Why it happens:** Stapling attaches the notarization ticket to the .app bundle. If the DMG is created before stapling, the ticket is not present inside the DMG.
**How to avoid:** Order: build → sign → zip → notarize → staple → create DMG.
**Warning signs:** `spctl --assess` on the .app inside the DMG returns failure even though the bare .app passes.

### Pitfall 3: on.push.tags Does Not Fire
**What goes wrong:** release.yml never runs after release-please merges the Release PR.
**Why it happens:** If release-please used GITHUB_TOKEN (not the PAT) to push the tag, GitHub's recursive-workflow protection suppresses on.push.tags.
**How to avoid:** The existing `RELEASE_PLEASE_TOKEN` PAT is already configured and in use. Confirm release-please.yml still uses `token: ${{ secrets.RELEASE_PLEASE_TOKEN }}` (not `secrets.GITHUB_TOKEN`).
**Warning signs:** Tag appears on GitHub but release.yml workflow never starts.

### Pitfall 4: nfpm version Must Not Start with 'v'
**What goes wrong:** nfpm errors: "version must start with a digit".
**Why it happens:** `GITHUB_REF_NAME` is `v1.8.0` (with `v`); deb version must be `1.8.0`.
**How to avoid:** `VERSION_BARE="${GITHUB_REF_NAME#v}"` strips the leading `v`.
**Warning signs:** nfpm exits with version format error.

### Pitfall 5: Notarization Wait Timeout
**What goes wrong:** `xcrun notarytool submit --wait` times out after 30+ minutes for large binaries.
**Why it happens:** Apple's notarization service can be slow. For a Wails app (~20MB), notarization typically takes 2-5 minutes but can spike.
**How to avoid:** The default GitHub Actions job timeout is 6 hours — no immediate action needed. However, the macOS job should be expected to take 10-15 minutes total.
**Warning signs:** Job exceeds expected runtime; notarytool reports timeout in logs.

### Pitfall 6: softprops/action-gh-release Fails to Find Release
**What goes wrong:** Action fails with "release not found" error.
**Why it happens:** release-please creates the GitHub Release atomically with the tag. If the tag push triggers release.yml before release-please has finished creating the release page, softprops may not find it.
**How to avoid:** The `publish` job runs after all build jobs complete (typically 10-20 minutes). By that time, release-please has long finished. If this is still an issue, add `make_latest: true` to softprops to create the release if not found.
**Warning signs:** publish job fails immediately with 404 on release lookup.

### Pitfall 7: webkit_tag Missing from Linux Release Build
**What goes wrong:** Linux binary crashes at runtime on systems with libwebkit2gtk-4.1 because it was compiled without the `webkit2_41` build tag.
**Why it happens:** The build tag selects webkit API version. Without it, the binary uses the older 4.0 API which is not installed on modern Ubuntu runners.
**How to avoid:** Always pass `build-flags: -tags webkit2_41` in the Linux job's wails-build-action invocation.
**Warning signs:** Build succeeds but binary fails with "symbol lookup error" at runtime.

## Code Examples

### Complete release.yml skeleton

```yaml
# Source: architecture derived from build.yml + Phase 45 research + Phase 46 research
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build-macos:
    runs-on: macos-latest
    environment: release
    env:
      MACOS_CERTIFICATE: ${{ secrets.MACOS_CERTIFICATE }}
      MACOS_CERTIFICATE_NAME: ${{ secrets.MACOS_CERTIFICATE_NAME }}
      MACOS_CERTIFICATE_PWD: ${{ secrets.MACOS_CERTIFICATE_PWD }}
      MACOS_CI_KEYCHAIN_PWD: ${{ secrets.MACOS_CI_KEYCHAIN_PWD }}
      MACOS_NOTARIZATION_APPLE_ID: ${{ secrets.MACOS_NOTARIZATION_APPLE_ID }}
      MACOS_NOTARIZATION_PWD: ${{ secrets.MACOS_NOTARIZATION_PWD }}
      MACOS_NOTARIZATION_TEAM_ID: ${{ secrets.MACOS_NOTARIZATION_TEAM_ID }}
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: true
      - uses: dAppServer/wails-build-action@main
        with:
          build-name: agenthub
          build-platform: darwin/universal
      - name: Import certificate
        run: |
          echo "$MACOS_CERTIFICATE" | base64 --decode > certificate.p12
          security create-keychain -p "$MACOS_CI_KEYCHAIN_PWD" build.keychain
          security default-keychain -s build.keychain
          security unlock-keychain -p "$MACOS_CI_KEYCHAIN_PWD" build.keychain
          security import certificate.p12 -k build.keychain \
            -P "$MACOS_CERTIFICATE_PWD" -T /usr/bin/codesign
          security set-key-partition-list -S apple-tool:,apple: \
            -s -k "$MACOS_CI_KEYCHAIN_PWD" build.keychain
      - name: Sign .app
        run: |
          codesign --deep --force --verbose \
            --options runtime --timestamp \
            --entitlements build/entitlements.plist \
            --sign "$MACOS_CERTIFICATE_NAME" \
            build/bin/agenthub.app
      - name: Notarize .app
        run: |
          xcrun notarytool store-credentials "agenthub-notarize" \
            --apple-id "$MACOS_NOTARIZATION_APPLE_ID" \
            --team-id "$MACOS_NOTARIZATION_TEAM_ID" \
            --password "$MACOS_NOTARIZATION_PWD"
          ditto -c -k --keepParent build/bin/agenthub.app build/bin/agenthub-notarize.zip
          xcrun notarytool submit build/bin/agenthub-notarize.zip \
            --keychain-profile "agenthub-notarize" --wait
          xcrun stapler staple build/bin/agenthub.app
      - name: Create DMG
        run: |
          brew install create-dmg
          DMG="agenthub-${GITHUB_REF_NAME}-darwin-universal.dmg"
          create-dmg --volname "AgentHub" \
            --codesign "$MACOS_CERTIFICATE_NAME" \
            "${DMG}" build/bin/agenthub.app
          mv "${DMG}" build/bin/
      - name: Cleanup
        if: always()
        run: |
          security delete-keychain build.keychain || true
          rm -f certificate.p12 build/bin/agenthub-notarize.zip
      - uses: actions/upload-artifact@v4
        with:
          name: macos-dmg
          path: build/bin/agenthub-*.dmg
          if-no-files-found: error

  build-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: true
      - uses: dAppServer/wails-build-action@main
        with:
          build-name: agenthub
          build-platform: windows/amd64
          build-webview2: embed
          nsis: true
      - name: Rename artifacts
        shell: bash
        run: |
          mv build/bin/agenthub-amd64-installer.exe \
             "build/bin/agenthub-${GITHUB_REF_NAME}-windows-amd64-installer.exe"
          mv build/bin/agenthub.exe \
             "build/bin/agenthub-${GITHUB_REF_NAME}-windows-amd64.exe"
      - uses: actions/upload-artifact@v4
        with:
          name: windows-artifacts
          path: build/bin/agenthub-*.exe
          if-no-files-found: error

  build-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: true
      - name: Install Linux dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y build-essential pkg-config \
            libgtk-3-dev libwebkit2gtk-4.1-dev
      - uses: dAppServer/wails-build-action@main
        with:
          build-name: agenthub
          build-platform: linux/amd64
          build-flags: -tags webkit2_41
      - name: Package artifacts
        run: |
          VERSION_BARE="${GITHUB_REF_NAME#v}"
          tar -czf "build/bin/agenthub-${GITHUB_REF_NAME}-linux-amd64.tar.gz" \
            -C build/bin agenthub
          go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
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
          "$(go env GOPATH)/bin/nfpm" package --packager deb \
            --target "build/bin/agenthub-${GITHUB_REF_NAME}-linux-amd64.deb"
      - uses: actions/upload-artifact@v4
        with:
          name: linux-artifacts
          path: |
            build/bin/agenthub-*.tar.gz
            build/bin/agenthub-*.deb
          if-no-files-found: error

  publish:
    needs: [build-macos, build-windows, build-linux]
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/download-artifact@v4
        with:
          path: artifacts
          merge-multiple: true
      - name: Generate checksums
        run: |
          cd artifacts
          sha256sum * > checksums.txt
          cat checksums.txt
      - uses: softprops/action-gh-release@v2
        with:
          files: |
            artifacts/*.dmg
            artifacts/*.exe
            artifacts/*.tar.gz
            artifacts/*.deb
            artifacts/checksums.txt
          fail_on_unmatched_files: true
```

### Getting VERSION from GITHUB_REF_NAME
```bash
# In tag-triggered workflow, GITHUB_REF_NAME = "v1.8.0"
VERSION="${GITHUB_REF_NAME}"           # "v1.8.0" — use for filenames
VERSION_BARE="${GITHUB_REF_NAME#v}"   # "1.8.0" — use for nfpm version field
```

### SHA256 Checksum Generation (Linux runner)
```bash
# Source: pre-installed on all ubuntu-latest runners
cd artifacts
sha256sum * > checksums.txt
# Output format: "abc123...  agenthub-v1.8.0-darwin-universal.dmg"
# Users verify with: sha256sum --check checksums.txt
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| actions/upload-release-asset | softprops/action-gh-release@v2 | 2020+ | Old action deprecated; softprops handles globs and existing release updates |
| Manual notarytool credentials flag | xcrun notarytool store-credentials + keychain profile | macOS 12+ / Xcode 13+ | Replaces altool; profile avoids plaintext credentials in process args |
| GoReleaser for release automation | Explicit workflow steps | N/A (GoReleaser incompatible with Wails) | GoReleaser cannot produce .app bundles; explicit steps required |
| release-type: go with version-file | release-type: simple with extra-files | June 2025 | Already decided in Phase 45 |

**Deprecated/outdated:**
- `xcrun altool --notarize-app`: Deprecated since Xcode 14 (2022). Use `xcrun notarytool submit`.
- `actions/upload-release-asset`: Deprecated action; use `softprops/action-gh-release@v2`.
- `codesign --deep` alone (without `--options runtime`): Produces a signed-but-not-notarizable app. `--options runtime` is required for notarization.

## Open Questions

1. **Does `create-dmg` sign the DMG or just the .app inside it?**
   - What we know: `create-dmg --codesign IDENTITY` signs the DMG itself. The .app inside was already signed before DMG creation.
   - What's unclear: Whether Apple's Gatekeeper requires the DMG wrapper to also be signed, or only the .app inside.
   - Recommendation: Sign both — sign the .app (required), then sign the DMG with `--codesign` (best practice, prevents Gatekeeper warnings on the DMG wrapper itself).

2. **Will the notarization staple transfer into the DMG?**
   - What we know: Stapling attaches the ticket to the `.app` bundle in-place. The DMG is created after stapling.
   - What's unclear: Whether `create-dmg` preserves the staple ticket when packaging the `.app` into the DMG image.
   - Recommendation: After release, run `spctl --assess --type open --context context:primary-signature build/bin/agenthub.app` inside the DMG to verify. In practice, create-dmg copies the .app verbatim and the staple is preserved.

3. **Should the Linux job target ubuntu-latest or ubuntu-22.04?**
   - What we know: build.yml has two Linux targets (ubuntu-latest for webkit2gtk-4.1, ubuntu-22.04 for webkit2gtk-4.0). Release only ships one.
   - What's unclear: Which webkit version gives broader Linux compatibility.
   - Recommendation: Ship ubuntu-latest (webkit2gtk-4.1 / Ubuntu 24) as primary. Phase 47's Homebrew research may inform whether the deb needs to target an older LTS. Document this in the plan as a decision point.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| macos-latest runner | macOS build job | ✓ | GitHub-hosted | — |
| windows-latest runner | Windows build job | ✓ | GitHub-hosted | — |
| ubuntu-latest runner | Linux build job | ✓ | GitHub-hosted | — |
| `release` GitHub environment | macOS secrets | ✓ | Confirmed in Phase 45 | — |
| MACOS_CERTIFICATE (env secret) | codesign | ✓ | In `release` env | — |
| MACOS_CERTIFICATE_NAME (env secret) | codesign | ✓ | In `release` env | — |
| MACOS_CERTIFICATE_PWD (env secret) | codesign | ✓ | In `release` env | — |
| MACOS_CI_KEYCHAIN_PWD (env secret) | keychain creation | ✓ | In `release` env | — |
| MACOS_NOTARIZATION_APPLE_ID (env secret) | notarytool | ✓ | In `release` env | — |
| MACOS_NOTARIZATION_PWD (env secret) | notarytool | ✓ | In `release` env | — |
| MACOS_NOTARIZATION_TEAM_ID (env secret) | notarytool | ✓ | In `release` env | — |
| RELEASE_PLEASE_TOKEN (repo secret) | on.push.tags trigger | ✓ | PAT confirmed | — |
| dAppServer/wails-build-action | all build jobs | ✓ | @main | — |
| softprops/action-gh-release | publish job | ✓ | v2.6.1 | — |
| create-dmg | macOS DMG creation | ✓ (brew install) | Latest | hdiutil directly (no DMG signing) |
| nfpm | Linux .deb creation | ✓ (go install) | Latest | dpkg-deb (requires debian/ tree) |
| Release PR for v1.8.0 | Trigger test | ✓ | Open on GitHub | — |

**Missing dependencies with no fallback:**
- None — all required secrets and runners are confirmed available.

**Missing dependencies with fallback:**
- None additional.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | bash smoke tests + go test (existing) |
| Config file | none |
| Quick run command | `python3 -m json.tool /dev/null; echo "YAML validation below"` |
| Full suite command | `go test -race ./... && bash tests/build-script.test.sh` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REL-02 | release.yml exists and is valid YAML | smoke | `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"` | ❌ Wave 0 |
| REL-02 | release.yml triggers on v* tags | smoke | `python3 -c "import yaml; d=yaml.safe_load(open('.github/workflows/release.yml')); assert 'v*' in str(d['on'])"` | ❌ Wave 0 |
| REL-02 | macOS job declares environment: release | smoke | `grep -c 'environment: release' .github/workflows/release.yml` | ❌ Wave 0 |
| REL-02 | macOS job references all 7 MACOS_* secrets | smoke | `grep -c 'MACOS_' .github/workflows/release.yml` (expect >= 7) | ❌ Wave 0 |
| REL-02 | End-to-end: merge Release PR triggers release.yml and produces artifacts | e2e | Manual — merge the open v1.8.0 Release PR and observe GitHub Actions | N/A |
| REL-04 | checksums.txt contains SHA256 for all artifacts | smoke | `grep -c 'sha256' artifacts/checksums.txt` (after e2e run) | ❌ Wave 0 |
| REL-04 | publish job includes checksums.txt in upload files | smoke | `grep -c 'checksums.txt' .github/workflows/release.yml` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"` (YAML validity)
- **Per wave merge:** `go test -race ./...` (no regressions)
- **Phase gate:** Merge the open v1.8.0 Release PR; verify all 6 artifacts + checksums.txt appear on the GitHub Release page

### Wave 0 Gaps
- [ ] `.github/workflows/release.yml` — new file, covers REL-02 and REL-04
- [ ] PyYAML available for YAML validation: `pip install pyyaml` or use `ruby -ryaml -e "YAML.load_file('.github/workflows/release.yml')"`
- [ ] End-to-end test is the phase gate: the open v1.8.0 Release PR (`chore(main): release 1.8.0`) is available for immediate testing

## Project Constraints (from CLAUDE.md)

| Directive | Impact on Phase 46 |
|-----------|-------------------|
| pnpm preferred for Node | `wails-build-action` uses its own Node setup; no direct impact |
| GitHub Actions for CI/CD | Confirmed — release.yml is a GitHub Actions workflow |
| Multi-stage Docker builds | Not applicable to this phase |
| Python venv for Python | No Python used in release.yml |
| Go: `go fmt`, `golangci-lint`, context-aware | build artifact only; no new Go code written in this phase |

## Sources

### Primary (HIGH confidence)
- `gh api repos/scottkw/agenthub/environments/release/secrets` — Confirmed 7 MACOS_* secrets in `release` environment
- `gh api repos/scottkw/agenthub/actions/secrets` — Confirmed `RELEASE_PLEASE_TOKEN` as repo-level PAT
- `gh pr list --repo scottkw/agenthub` — v1.8.0 Release PR is open and ready to merge
- `/Users/ken/dev/agenthub/.github/workflows/build.yml` — Existing wails-build-action usage patterns
- `/Users/ken/dev/agenthub/build/entitlements.plist` — Entitlements file at `build/entitlements.plist`
- `dAppServer/wails-build-action action.yml` — Confirmed: no DMG output; produces .app, .app.zip, .pkg
- [softprops/action-gh-release v2.6.1](https://github.com/softprops/action-gh-release/tree/v2) — Latest stable (March 16, 2026); glob patterns; existing release update

### Secondary (MEDIUM confidence)
- [federicoterzi.com macOS signing blog](https://federicoterzi.com/blog/automatic-code-signing-and-notarization-for-macos-apps-using-github-actions/) — Signing pipeline patterns; cross-verified with build.sh sign_and_notarize function
- [create-dmg README](https://github.com/create-dmg/create-dmg) — `--codesign` and `--notarize` parameters; DMG creation from .app
- [nfpm documentation](https://nfpm.goreleaser.com/) — .deb config format; nfpm.yaml structure
- [GitHub Docs — events-that-trigger-workflows](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows) — on.push.tags behavior with PAT vs GITHUB_TOKEN

### Tertiary (LOW confidence)
- [Community discussion: PAT triggers on.push.tags](https://github.com/orgs/community/discussions/76402) — General confirmation that PAT-pushed tags trigger workflows; not specific to release-please
- [softprops/action-gh-release issue #323](https://github.com/softprops/action-gh-release/issues/323) — Existing release update behavior

## Metadata

**Confidence breakdown:**
- Standard stack (wails-build-action, softprops, create-dmg, nfpm): HIGH — all verified via official repos and project's existing build.yml
- macOS signing pipeline: HIGH — mirrors the existing build.sh sign_and_notarize function; secrets confirmed in `release` environment
- DMG creation approach: MEDIUM — create-dmg is the standard tool; staple transfer into DMG is confirmed by community but not tested in this project
- PAT trigger behavior: HIGH — confirmed by GitHub docs and existing RELEASE_PLEASE_TOKEN setup
- nfpm deb packaging: MEDIUM — configuration format verified via docs; untested against this specific binary

**Research date:** 2026-04-04
**Valid until:** 2026-05-04 (stable toolchain; softprops v2 and wails-build-action @main are actively maintained)
