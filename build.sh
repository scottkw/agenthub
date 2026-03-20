#!/usr/bin/env bash
set -euo pipefail

# build.sh — Cross-platform build script for AgentHub (Wails v2)
# Usage: build.sh --platform macos|linux|windows | --all [--sign]

PLATFORM=""
SIGN="false"

usage() {
  echo "Usage: build.sh --platform macos|linux|windows | --all [--sign]"
  echo ""
  echo "  --platform macos     Build macOS universal binary (.app bundle)"
  echo "  --platform linux     Build Linux amd64 binary via Docker"
  echo "  --platform windows   Build Windows amd64 binary via cross-compile"
  echo "  --all                Build all three platforms sequentially"
  echo "  --sign               Code-sign and notarize the macOS build (requires credentials)"
  exit 1
}

# Argument parsing
while [[ $# -gt 0 ]]; do
  case "$1" in
    --platform)
      if [[ $# -lt 2 ]]; then
        echo "ERROR: --platform requires a value (macos|linux|windows)"
        usage
      fi
      PLATFORM="$2"
      shift 2
      ;;
    --all)
      PLATFORM="all"
      shift
      ;;
    --sign)
      SIGN="true"
      shift
      ;;
    *)
      echo "ERROR: Unknown flag: $1"
      usage
      ;;
  esac
done

if [[ -z "$PLATFORM" ]]; then
  usage
fi

# Validate platform value
case "$PLATFORM" in
  macos|linux|windows|all)
    ;;
  *)
    echo "ERROR: Invalid platform: $PLATFORM"
    usage
    ;;
esac

# Wails binary check
WAILS="$(go env GOPATH)/bin/wails"
if [[ ! -x "$WAILS" ]]; then
  echo "ERROR: wails not found at $WAILS"
  echo "Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
  exit 1
fi

# --- Build functions ---

build_macos() {
  echo "==> Building macOS (darwin/universal)"
  "$WAILS" build -platform darwin/universal -clean
  echo "==> macOS build complete: build/bin/agenthub.app"

  if [[ "$SIGN" == "true" ]]; then
    sign_and_notarize "build/bin/agenthub.app"
  fi
}

build_windows() {
  local MINGW_CC="x86_64-w64-mingw32-gcc"
  if ! command -v "$MINGW_CC" &>/dev/null; then
    echo "ERROR: $MINGW_CC not found. Install: brew install mingw-w64"
    exit 1
  fi

  echo "==> Building Windows (windows/amd64)"
  CC="$MINGW_CC" CGO_ENABLED=1 "$WAILS" build -platform windows/amd64 -clean
  echo "==> Windows build complete: build/bin/agenthub.exe"
}

build_linux() {
  if ! docker info &>/dev/null; then
    echo "ERROR: Docker is not running. Linux builds require Docker."
    echo "Start Docker Desktop and try again."
    exit 1
  fi

  # Pre-build frontend assets (Docker container won't have Node/pnpm)
  echo "==> Building frontend assets for Linux"
  (cd frontend && pnpm install --frozen-lockfile && pnpm run build)

  echo "==> Building Linux (linux/amd64) via Docker"
  docker run --rm \
    --platform linux/amd64 \
    -v "$(pwd)":/app \
    -w /app \
    golang:1.26-bookworm \
    bash -c "
      apt-get update -qq && \
      apt-get install -y --no-install-recommends \
        gcc libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config && \
      CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
        go build -tags production \
        -ldflags '-w -s' \
        -o build/bin/agenthub .
    "
  echo "==> Linux build complete: build/bin/agenthub"
}

sign_and_notarize() {
  local APP="$1"
  local ENTITLEMENTS="build/entitlements.plist"
  local IDENTITY="${MACOS_SIGNING_IDENTITY:-}"
  local APPLE_ID="${MACOS_APPLE_ID:-}"
  local TEAM_ID="${MACOS_TEAM_ID:-}"
  local APP_PASSWORD="${MACOS_APP_PASSWORD:-}"

  # Validate all four required env vars
  local MISSING=()
  [[ -z "$IDENTITY" ]] && MISSING+=("MACOS_SIGNING_IDENTITY")
  [[ -z "$APPLE_ID" ]] && MISSING+=("MACOS_APPLE_ID")
  [[ -z "$TEAM_ID" ]] && MISSING+=("MACOS_TEAM_ID")
  [[ -z "$APP_PASSWORD" ]] && MISSING+=("MACOS_APP_PASSWORD")

  if [[ ${#MISSING[@]} -gt 0 ]]; then
    echo "ERROR: Missing required environment variables for code signing:"
    for var in "${MISSING[@]}"; do
      echo "  - $var"
    done
    echo ""
    echo "Setup instructions:"
    echo "  MACOS_SIGNING_IDENTITY  Run: security find-identity -v -p codesigning"
    echo "  MACOS_APPLE_ID          Your Apple ID email"
    echo "  MACOS_TEAM_ID           Apple Developer portal -> Membership -> Team ID"
    echo "  MACOS_APP_PASSWORD      appleid.apple.com -> Security -> App-Specific Passwords"
    exit 1
  fi

  # Verify entitlements file exists
  if [[ ! -f "$ENTITLEMENTS" ]]; then
    echo "ERROR: Entitlements file not found: $ENTITLEMENTS"
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
  echo "==> Verifying signature"
  codesign --verify --deep --strict --verbose=2 "$APP"

  # Step 3: Create zip for notarytool (ditto preserves macOS metadata; NOT zip -r)
  local ZIP="build/bin/agenthub-notarize.zip"
  echo "==> Creating notarization archive"
  ditto -c -k --keepParent "$APP" "$ZIP"

  # Step 4: Store credentials in keychain profile
  echo "==> Storing notarization credentials"
  xcrun notarytool store-credentials "agenthub-notarize" \
    --apple-id "$APPLE_ID" \
    --team-id "$TEAM_ID" \
    --password "$APP_PASSWORD"

  # Step 5: Submit and wait (--wait is CRITICAL: without it, exit 0 does NOT mean success)
  echo "==> Submitting for notarization (this may take several minutes)"
  xcrun notarytool submit "$ZIP" \
    --keychain-profile "agenthub-notarize" \
    --wait

  # Step 6: Staple notarization ticket to .app
  echo "==> Stapling notarization ticket"
  xcrun stapler staple "$APP"

  # Step 7: Final verification with spctl (exit 0 = Gatekeeper accepted)
  echo "==> Verifying with spctl"
  spctl --assess --verbose=2 --type install "$APP"

  # Cleanup
  rm -f "$ZIP"
  echo "==> macOS build signed and notarized: $APP"
}

# --- Dispatch ---

case "$PLATFORM" in
  macos)   build_macos ;;
  linux)   build_linux ;;
  windows) build_windows ;;
  all)     build_macos && build_windows && build_linux ;;
  *)       usage ;;
esac
