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

  echo "==> Building Linux (linux/amd64) via Docker"
  docker run --rm \
    -v "$(pwd)":/app \
    -w /app \
    ghcr.io/abjrcode/cross-wails:v2.6.0 \
    wails build -platform linux/amd64 -clean
  echo "==> Linux build complete: build/bin/agenthub"
}

sign_and_notarize() {
  echo "ERROR: Code signing not yet implemented. See build.sh plan 02."
  exit 1
}

# --- Dispatch ---

case "$PLATFORM" in
  macos)   build_macos ;;
  linux)   build_linux ;;
  windows) build_windows ;;
  all)     build_macos && build_windows && build_linux ;;
  *)       usage ;;
esac
