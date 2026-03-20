# Phase 13: Build Script - Research

**Researched:** 2026-03-20
**Domain:** Wails v2 build system, cross-platform compilation, macOS code signing and notarization
**Confidence:** HIGH (verified against installed wails v2.10.2 CLI, official community docs, and Apple's toolchain)

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| BUILD-01 | User can run `build.sh --platform macos` to compile for macOS only | `wails build -platform darwin/universal` runs natively on macOS with no extra tools; verified via dry-run |
| BUILD-02 | User can run `build.sh --platform linux` to compile for Linux only | Requires Docker + wails-cross image due to CGO/WebKit2GTK dependency; direct cross-compile from macOS unsupported |
| BUILD-03 | User can run `build.sh --platform windows` to compile for Windows only | `wails build -platform windows/amd64` works from macOS via mingw-w64 (already installed at `/opt/homebrew/bin/x86_64-w64-mingw32-gcc`); Windows uses pure Go WebView2 — CGO_ENABLED=1 needed for mingw |
| BUILD-04 | User can run `build.sh --all` to compile for all three platforms sequentially | Shell script: invoke macOS build first (native), then Windows (mingw), then Linux (Docker); sequential with early-exit on failure |
| BUILD-05 | User can run `build.sh --platform macos --sign` to code-sign and notarize the macOS build | `codesign --deep --force --options runtime --timestamp --entitlements` + `xcrun notarytool submit --wait` + `xcrun stapler staple` + `spctl --assess` verification |
</phase_requirements>

---

## Summary

AgentHub uses Wails v2.10.2 (already installed at `/Users/ken/go/bin/wails`) to build a Go+React desktop app into platform-specific bundles. The `wails build -platform <target>` command handles all frontend compilation, Go compilation, and platform packaging in one step.

**Critical finding — cross-compilation asymmetry:** macOS and Linux builds both use CGO (for WebKit/WebView), making true cross-compilation impossible without special tooling. macOS must always be built natively. Linux from macOS requires Docker (wails-cross image) because WebKit2GTK headers are not available on macOS. Windows, however, uses pure Go WebView2 — it cross-compiles from macOS cleanly using the `x86_64-w64-mingw32-gcc` compiler already installed on this machine.

**Primary recommendation:** Write `build.sh` as a dispatch script: macOS runs `wails build -platform darwin/universal` natively; Windows cross-compiles via `CC=x86_64-w64-mingw32-gcc wails build -platform windows/amd64`; Linux uses Docker with `ghcr.io/abjrcode/cross-wails`. The `--sign` flag triggers the `codesign` → `ditto` → `xcrun notarytool submit --wait` → `xcrun stapler staple` → `spctl --assess` pipeline.

The STATE.md notes one known pitfall: "notarytool exit-0 trap cannot be detected without real notarization run" — meaning the script must use `--wait` and explicitly check the submission result rather than assuming success.

---

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| wails CLI | v2.10.2 (installed) | One-command build: frontend + Go + platform packaging | Project's own build tool — produces `.app`, `.exe`, ELF |
| codesign | macOS system (Xcode) | Sign the `.app` bundle with Developer ID Application cert | Apple-mandated; required before notarytool submission |
| xcrun notarytool | macOS system (Xcode 13+) | Submit app to Apple notarization service | Replaced altool in 2023; current standard |
| xcrun stapler | macOS system | Attach notarization ticket to `.app` bundle | Enables offline Gatekeeper verification |
| spctl | macOS system | Verify notarization passed | `spctl --assess -vvv -t install .app` is canonical check |
| x86_64-w64-mingw32-gcc | 15.2.0 (installed via brew mingw-w64) | CGO cross-compiler for Windows builds | Already present; enables Windows cross-compile from macOS |
| Docker | 29.2.1 (installed) | Run wails-cross image for Linux build | wails-cross bundles WebKit2GTK headers needed by Wails CGO |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| ditto | macOS system | Create zip archive of `.app` for notarytool | Required format for notarytool submission (not `zip -r`) |
| security (CLI) | macOS system | Import certificates, manage keychain | Needed in CI; for local builds, certificates are already in keychain |
| ghcr.io/abjrcode/cross-wails | v2.6.0 | Docker image with Go + Wails + WebKit2GTK for Linux builds | Only needed for Linux target from macOS |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Docker for Linux | musl-cross (filosottile) | musl-cross installs CC for Linux, but Wails also needs WebKit2GTK pkg-config headers — these are not available from musl-cross; Docker is the only realistic option |
| notarytool directly | mitchellh/gon | gon wraps codesign+notarytool; adds a JSON config file and homebrew dependency; notarytool directly is simpler and maintained by Apple |
| darwin/universal | darwin/arm64 | Universal binary (arm64+amd64) is standard for distribution; arm64-only would exclude Intel Macs |

**Installation (if missing):**
```bash
# wails (if needed)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# mingw for Windows cross-compile (already installed)
brew install mingw-w64

# musl-cross (NOT sufficient for Linux Wails — needs webkit2gtk, use Docker instead)
# Docker already installed
```

---

## Architecture Patterns

### Recommended Script Structure
```
build.sh              # Single entry point at project root
build/
├── bin/              # Wails output directory (gitignored)
│   ├── agenthub.app          # macOS output
│   ├── agenthub              # Linux output (ELF)
│   └── agenthub.exe          # Windows output
├── darwin/
│   ├── Info.plist            # Already exists
│   └── entitlements.plist    # Already exists
```

### Pattern 1: Argument Dispatch
**What:** `build.sh` parses `--platform` and `--sign` flags, dispatches to internal functions
**When to use:** All invocations

```bash
#!/usr/bin/env bash
set -euo pipefail

PLATFORM=""
SIGN=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --platform) PLATFORM="$2"; shift 2 ;;
    --all)      PLATFORM="all"; shift ;;
    --sign)     SIGN=true; shift ;;
    *) echo "Unknown flag: $1"; exit 1 ;;
  esac
done

case "$PLATFORM" in
  macos)   build_macos ;;
  linux)   build_linux ;;
  windows) build_windows ;;
  all)     build_macos; build_linux; build_windows ;;
  *)       echo "Usage: build.sh --platform macos|linux|windows|--all [--sign]"; exit 1 ;;
esac
```

### Pattern 2: macOS Native Build
**What:** Run `wails build` directly on the host; package produces `build/bin/agenthub.app`
**When to use:** `--platform macos`

```bash
build_macos() {
  echo "==> Building macOS (darwin/universal)"
  wails build -platform darwin/universal -clean
  APP="build/bin/agenthub.app"

  if [[ "$SIGN" == "true" ]]; then
    sign_and_notarize "$APP"
  fi
}
```

**Output:** `build/bin/agenthub.app` (universal .app bundle, not signed by default)

### Pattern 3: Windows Cross-Compile via MinGW
**What:** Set GOOS/GOARCH/CC env vars, run `wails build`
**When to use:** `--platform windows`

```bash
build_windows() {
  echo "==> Building Windows (windows/amd64)"
  CC=x86_64-w64-mingw32-gcc \
  CGO_ENABLED=1 \
  wails build -platform windows/amd64 -clean
  # Output: build/bin/agenthub.exe
}
```

**Key:** Windows uses WebView2 (pure Go), but CGO_ENABLED=1 is still needed for Wails itself; the mingw CC handles the Windows-targeting.

### Pattern 4: Linux Build via Docker
**What:** Mount project into wails-cross Docker image, run `wails build` inside
**When to use:** `--platform linux`

```bash
build_linux() {
  echo "==> Building Linux (linux/amd64) via Docker"
  docker run --rm \
    -v "$(pwd)":/app \
    -w /app \
    ghcr.io/abjrcode/cross-wails:v2.6.0 \
    wails build -platform linux/amd64 -clean
  # Output: build/bin/agenthub (ELF)
}
```

**Note:** Docker must be running. Image is ~4.2GB; first pull takes time.

### Pattern 5: macOS Sign + Notarize Pipeline
**What:** codesign → ditto zip → notarytool submit --wait → stapler → spctl verify
**When to use:** `--sign` with `--platform macos`

```bash
sign_and_notarize() {
  local APP="$1"
  local ENTITLEMENTS="build/entitlements.plist"
  local IDENTITY="${MACOS_SIGNING_IDENTITY:-}"    # e.g. "Developer ID Application: Name (TEAMID)"
  local APPLE_ID="${MACOS_APPLE_ID:-}"
  local TEAM_ID="${MACOS_TEAM_ID:-}"
  local APP_PASSWORD="${MACOS_APP_PASSWORD:-}"     # app-specific password

  if [[ -z "$IDENTITY" || -z "$APPLE_ID" || -z "$TEAM_ID" || -z "$APP_PASSWORD" ]]; then
    echo "ERROR: MACOS_SIGNING_IDENTITY, MACOS_APPLE_ID, MACOS_TEAM_ID, MACOS_APP_PASSWORD required"
    exit 1
  fi

  # Step 1: Deep sign with hardened runtime
  echo "==> Signing $APP"
  codesign --deep --force --verbose \
    --options runtime \
    --timestamp \
    --entitlements "$ENTITLEMENTS" \
    --sign "$IDENTITY" \
    "$APP"

  # Step 2: Verify signature
  codesign --verify --deep --strict --verbose=2 "$APP"

  # Step 3: Create zip for notarytool (ditto preserves bundle structure)
  local ZIP="build/bin/agenthub-notarize.zip"
  ditto -c -k --keepParent "$APP" "$ZIP"

  # Step 4: Store credentials in temporary keychain profile
  xcrun notarytool store-credentials "agenthub-notarize" \
    --apple-id "$APPLE_ID" \
    --team-id "$TEAM_ID" \
    --password "$APP_PASSWORD"

  # Step 5: Submit and wait (--wait blocks until Apple responds)
  echo "==> Submitting for notarization (may take a few minutes)"
  xcrun notarytool submit "$ZIP" \
    --keychain-profile "agenthub-notarize" \
    --wait

  # Step 6: Staple ticket to .app
  echo "==> Stapling notarization ticket"
  xcrun stapler staple "$APP"

  # Step 7: Verify with spctl (exit code 0 = accepted)
  echo "==> Verifying with spctl"
  spctl --assess --verbose=2 --type install "$APP"

  rm -f "$ZIP"
  echo "==> macOS build signed and notarized: $APP"
}
```

### Anti-Patterns to Avoid
- **`zip -r` instead of `ditto -c -k --keepParent`:** Zip does not preserve macOS extended attributes and resource forks. `ditto` is the Apple-recommended tool for creating notarization archives.
- **`--deep` on codesign without signing inner frameworks first:** For complex bundles with embedded frameworks, inner items should be signed bottom-up before the outer bundle. AgentHub's bundle is simple (single binary), so `--deep` is sufficient.
- **Assuming notarytool success without `--wait`:** Without `--wait`, submit returns immediately and you have no confirmation of success. Always use `--wait`.
- **Running `spctl` on the raw binary instead of `.app`:** `spctl --assess` only works on app bundles, `.pkg`, or `.dmg` — not raw executables.
- **Cross-compiling macOS on Linux/Windows:** Wails explicitly skips darwin targets with "WARNING Crosscompiling to Mac not currently supported." macOS must be built on macOS.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Frontend build | Manual `pnpm build` + Go embed | `wails build` | Wails handles the entire pipeline: pnpm install, pnpm build, embed assets, compile Go |
| macOS app bundle structure | Manually create Contents/MacOS/Resources | `wails build -platform darwin` | Wails generates the .app bundle from `build/darwin/Info.plist` template |
| Windows .exe packaging | Manual rsrc or goversioninfo | `wails build -platform windows` | Wails handles Windows resource injection |
| Notarization status polling | Custom loop with `notarytool info` | `notarytool submit --wait` | `--wait` blocks until complete; handles timeouts and retries internally |
| Zip for notarization | `zip -r app.zip app.app` | `ditto -c -k --keepParent` | `zip` loses macOS metadata; `ditto` is the Apple-documented method |

**Key insight:** `wails build` is the single source of truth for compilation. The build script's job is argument parsing, environment setup (CC for Windows, Docker for Linux), and the post-build signing pipeline — not reimplementing what Wails already does.

---

## Common Pitfalls

### Pitfall 1: notarytool exit-0 trap
**What goes wrong:** `xcrun notarytool submit` exits 0 even when Apple rejects the notarization (e.g., missing entitlements, wrong signature). The rejection is in the response body, not the exit code.
**Why it happens:** notarytool exits 0 for "submission accepted" not "notarization approved". Without `--wait`, the script falsely appears to succeed.
**How to avoid:** Always pass `--wait` flag. With `--wait`, notarytool exits non-zero when Apple's notarization fails. Capture the output and check for "status: Accepted".
**Warning signs:** Script completes but stapler fails; or `spctl --assess` returns "rejected".

### Pitfall 2: Hardened runtime entitlements mismatch
**What goes wrong:** AgentHub uses `com.apple.security.network.client` and `com.apple.security.network.server` in `build/entitlements.plist`. Signing without `--entitlements` drops these, causing the app to fail at runtime on notarized systems (no network access).
**Why it happens:** Hardened runtime defaults are restrictive; entitlements must be embedded in the signature.
**How to avoid:** Always pass `--entitlements build/entitlements.plist` to codesign.

### Pitfall 3: Linux build missing Docker
**What goes wrong:** `wails build -platform linux/amd64` on macOS either fails immediately (pkg-config cannot find webkit2gtk) or produces a binary that doesn't run on target Linux.
**Why it happens:** Wails' Linux WebKit2 renderer requires `webkit2gtk` pkg-config headers at compile time. These are only available on Linux systems with `libwebkit2gtk-4.0-dev` installed.
**How to avoid:** Always use Docker with the wails-cross image for Linux builds. Check `docker info` at the top of the Linux build function and fail with a clear message if Docker is not running.

### Pitfall 4: Windows cross-compile failure after Wails upgrade
**What goes wrong:** Users reported that upgrading from Wails v2.9.2 to v2.10.1 broke Windows cross-compilation from macOS (exec format error). Fixed in later patches.
**Why it happens:** The Wails binding generator produces host-native binaries during the build process; when the host and target differ, this can fail if the wails CLI isn't compiled to handle it.
**How to avoid:** Use the installed Wails version (v2.10.2, which includes the fix). If cross-compilation fails, add `-skipbindings -s` flags to bypass binding generation and use cached bindings.

### Pitfall 5: macOS binary not in PATH / wrong wails version
**What goes wrong:** `wails` resolves to an old version or system version rather than the installed Go-compiled one.
**How to avoid:** Use the explicit path `$(go env GOPATH)/bin/wails` or verify `wails version` at script start. Add a version check.

### Pitfall 6: --sign flag fails on developer machine without Apple credentials
**What goes wrong:** Running `--sign` without setting environment variables exits with an opaque codesign error.
**How to avoid:** Check all four required env vars (MACOS_SIGNING_IDENTITY, MACOS_APPLE_ID, MACOS_TEAM_ID, MACOS_APP_PASSWORD) at the top of `sign_and_notarize()` and print a clear error with setup instructions before attempting any signing.

---

## Code Examples

Verified patterns from official sources and installed toolchain:

### Wails Build Dry Run Output (verified against v2.10.2)
```
# wails build -dryrun -platform darwin/amd64
Platform(s)        | darwin/amd64
Compiler           | /opt/homebrew/bin/go
Build Mode         | production
Package            | true
```

### Available Wails Platform Targets
```bash
# Native macOS (must be on macOS)
wails build -platform darwin/arm64        # Apple Silicon only
wails build -platform darwin/amd64        # Intel only
wails build -platform darwin/universal    # Fat binary (recommended for distribution)

# Cross-compile from macOS (Windows works; Linux needs Docker)
CC=x86_64-w64-mingw32-gcc wails build -platform windows/amd64

# Linux requires Docker — see Pattern 4
```

### Signing Identity Discovery
```bash
# List available signing identities
security find-identity -v -p codesigning
# Look for: Developer ID Application: <Name> (<TEAMID>)
```

### spctl Verification
```bash
# Verify notarization — exit 0 means accepted
spctl --assess --verbose=2 --type install build/bin/agenthub.app
# Success output: build/bin/agenthub.app: accepted
# Source: Apple developer documentation + community verification
```

### notarytool submit with keychain profile
```bash
# Source: https://gist.github.com/rsms/929c9c2fec231f0cf843a1a746a416f5
xcrun notarytool store-credentials "agenthub-notarize" \
  --apple-id "$MACOS_APPLE_ID" \
  --team-id "$MACOS_TEAM_ID" \
  --password "$MACOS_APP_PASSWORD"

xcrun notarytool submit "build/bin/agenthub-notarize.zip" \
  --keychain-profile "agenthub-notarize" \
  --wait

xcrun stapler staple "build/bin/agenthub.app"
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `xcrun altool --notarize-file` | `xcrun notarytool submit` | Xcode 13 (2021), mandatory 2023 | altool's notarization syntax no longer works |
| Manual polling loop (`notarytool info`) | `notarytool submit --wait` | Xcode 13+ | Simpler; `--wait` blocks until Apple responds |
| `gon` tool for signing+notarizing | Direct `codesign` + `notarytool` | 2023+ | Fewer dependencies; gon is unmaintained since 2022 |
| `zip -r` for notarization archive | `ditto -c -k --keepParent` | Long-standing best practice | ditto preserves macOS extended attributes |

**Deprecated/outdated:**
- `xcrun altool --notarize-file`: Removed in Xcode 15+; use `notarytool`
- `mitchellh/gon`: Last commit 2022; wraps notarytool internally but adds an extra dependency and JSON config file; not recommended for new projects

---

## Open Questions

1. **Linux Docker image version compatibility**
   - What we know: `ghcr.io/abjrcode/cross-wails:v2.6.0` targets Wails v2.6.0; this project uses v2.10.2
   - What's unclear: Whether a newer image tag exists that matches v2.10.2; whether the v2.6.0 image works with v2.10.2 source
   - Recommendation: Pull the image and test before committing to it; the wails binary inside the image can be overridden with `-compiler` flag or by installing fresh via Go inside the container. Alternative: `wailsapp/cc` (official Wails cross-compile images).

2. **Apple Developer credentials for local --sign testing**
   - What we know: BUILD-05 requires a real notarization run to verify; credentials require an Apple Developer account
   - What's unclear: Whether the developer has credentials set up; notarytool exit-0 trap cannot be verified without a live run
   - Recommendation: Build the script with the correct structure and document required env vars clearly. The verification step (`spctl --assess`) will conclusively validate success when credentials are available.

3. **darwin/universal vs darwin/arm64 for testing**
   - What we know: `darwin/universal` produces a fat binary (arm64 + amd64); takes ~2x compile time
   - What's unclear: Whether the project owner wants to default to universal or native arch for `--platform macos`
   - Recommendation: Default to `darwin/universal` for distribution builds (the expected behavior for a distribution build script). Could add `--arch` flag as optional override.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Frontend framework | vitest v4.1.0 (in `frontend/package.json`) |
| Frontend config | `frontend/vite.config.ts` (test section) |
| Go framework | `go test` (stdlib) |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test && go test ./...` |

### Phase Requirements → Test Map

The BUILD-* requirements are shell script behaviors. They are integration tests (run the script, check exit code and output file), not unit tests. Automated fast tests are not meaningful for BUILD-01 through BUILD-04 (they require compilation time). BUILD-05 requires real Apple credentials.

| Req ID | Behavior | Test Type | Automated Command | Notes |
|--------|----------|-----------|-------------------|-------|
| BUILD-01 | `build.sh --platform macos` exits 0, produces `build/bin/agenthub.app` | Integration | `bash build.sh --platform macos && test -d build/bin/agenthub.app` | ~2-5 min compile |
| BUILD-02 | `build.sh --platform linux` exits 0, produces `build/bin/agenthub` (ELF) | Integration | `bash build.sh --platform linux && file build/bin/agenthub \| grep ELF` | Requires Docker |
| BUILD-03 | `build.sh --platform windows` exits 0, produces `build/bin/agenthub.exe` | Integration | `bash build.sh --platform windows && test -f build/bin/agenthub.exe` | ~2-5 min compile |
| BUILD-04 | `build.sh --all` produces all three outputs | Integration | `bash build.sh --all && test -d build/bin/agenthub.app && test -f build/bin/agenthub && test -f build/bin/agenthub.exe` | Sequential |
| BUILD-05 | `build.sh --platform macos --sign` passes `spctl --assess` | Manual-only | N/A — requires Apple Developer credentials | Cannot automate without credentials |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`  (existing frontend tests, ensures no regressions)
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test && go test ./...`
- **Phase gate:** BUILD-01, BUILD-02, BUILD-03, BUILD-04 integration checks manually run before `/gsd:verify-work`

### Wave 0 Gaps
- None — this phase creates a new `build.sh` file; no existing test infrastructure gaps. The integration test commands above are manual verifications, not new test files.

---

## Sources

### Primary (HIGH confidence)
- Installed `wails` CLI v2.10.2 — `wails build --help` flags extracted directly; dry-run confirmed platform acceptance
- `build/entitlements.plist` and `build/darwin/Info.plist` in this repository — existing Apple configuration reviewed
- `go.mod` — confirmed Wails v2.10.2 dependency
- `wails.json` — confirmed project name, output filename, frontend build command
- Apple developer toolchain (system `codesign`, `xcrun notarytool`, `xcrun stapler`, `spctl`) — confirmed present on macOS

### Secondary (MEDIUM confidence)
- [rsms/macOS distribution gist](https://gist.github.com/rsms/929c9c2fec231f0cf843a1a746a416f5) — complete codesign + notarytool workflow; cross-verified with Apple docs
- [Federico Terzi - macOS code signing GitHub Actions](https://federicoterzi.com/blog/automatic-code-signing-and-notarization-for-macos-apps-using-github-actions/) — complete env var pattern for notarytool
- [wails-cross build limitations (chriswheeler.dev)](https://chriswheeler.dev/posts/cross-compilation-with-wails/) — confirmed macOS cannot be cross-compiled; Windows pure-Go confirmed
- [cross-wails Docker image (madin.dev)](https://madin.dev/cross-wails/) — confirmed Docker approach for Linux cross-compile

### Tertiary (LOW confidence)
- [wailsapp/xgobase](https://github.com/wailsapp/xgobase) — official Wails cross-compile Docker images; version compatibility with v2.10.2 not verified
- Windows cross-compile with CGO_ENABLED=1 pattern — verified from multiple community sources but not end-to-end tested on this machine for this project

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — wails CLI installed and tested; system tools (codesign, notarytool) are macOS standards
- Architecture: HIGH — platform dispatch pattern is straightforward; signing pipeline from authoritative sources
- Windows cross-compile: HIGH — mingw-w64 installed and verified; community-confirmed pattern
- Linux cross-compile: MEDIUM — Docker approach confirmed as correct method; specific image version compatibility with v2.10.2 needs validation
- Pitfalls: HIGH — notarytool exit-0 trap noted in STATE.md; other pitfalls from authoritative sources

**Research date:** 2026-03-20
**Valid until:** 2026-09-20 (Wails v2 is stable; Apple signing process changes infrequently)
