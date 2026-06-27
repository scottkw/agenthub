#!/usr/bin/env sh
# scripts/install.sh
# Linux installer for agenthub — POSIX sh (works with dash)
# Usage: curl -fsSL https://raw.githubusercontent.com/scottkw/agenthub/main/scripts/install.sh | sh
set -eu

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || {
        printf 'Error: required command not found: %s\n' "$1" >&2
        exit 1
    }
}

# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------

need_cmd curl
need_cmd tar

# Pick SHA256 command (sha256sum on GNU/Linux; shasum -a 256 as fallback)
if command -v sha256sum >/dev/null 2>&1; then
    SHA_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
    SHA_CMD="shasum -a 256"
else
    printf 'Error: neither sha256sum nor shasum found\n' >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Architecture detection
# ---------------------------------------------------------------------------

ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    *)
        printf 'Error: unsupported architecture: %s\n' "$ARCH" >&2
        printf 'Only linux/amd64 is available. See https://github.com/scottkw/agenthub/releases\n' >&2
        exit 1 ;;
esac

# ---------------------------------------------------------------------------
# Resolve latest release version
# ---------------------------------------------------------------------------

VERSION=$(curl -fsSL "https://api.github.com/repos/scottkw/agenthub/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

[ -n "$VERSION" ] || {
    printf 'Error: could not resolve latest release version\n' >&2
    exit 1
}

printf 'Installing agenthub %s for linux/%s...\n' "$VERSION" "$ARCH"

# ---------------------------------------------------------------------------
# Download + verify + extract
# ---------------------------------------------------------------------------

TARBALL="agenthub-${VERSION}-linux-${ARCH}.tar.gz"
BASE_URL="https://github.com/scottkw/agenthub/releases/download/${VERSION}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "${BASE_URL}/${TARBALL}"    -o "${TMPDIR}/${TARBALL}"
curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMPDIR}/checksums.txt"

# Require a non-empty checksum entry for this tarball
EXPECTED=$(grep "${TARBALL}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
[ -n "$EXPECTED" ] || {
    printf 'Error: %s not found in checksums.txt\n' "$TARBALL" >&2
    exit 1
}

# Compute actual hash (full path avoids cwd-relative pitfall)
ACTUAL=$($SHA_CMD "${TMPDIR}/${TARBALL}" | awk '{print $1}')

[ "$ACTUAL" = "$EXPECTED" ] || {
    printf 'Error: SHA256 mismatch — download may be corrupt\n' >&2
    printf '  Expected: %s\n' "$EXPECTED" >&2
    printf '  Actual:   %s\n' "$ACTUAL"   >&2
    exit 1
}

printf 'SHA256 verified.\n'

# Extract single binary at archive root (no subdirectory)
tar xzf "${TMPDIR}/${TARBALL}" -C "$TMPDIR" agenthub

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------

if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

cp "${TMPDIR}/agenthub" "${INSTALL_DIR}/agenthub"
chmod 755 "${INSTALL_DIR}/agenthub"

printf 'Installed agenthub to %s/agenthub\n' "$INSTALL_DIR"

# ---------------------------------------------------------------------------
# PATH warning for non-root installs
# ---------------------------------------------------------------------------

case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        printf '\nNote: %s is not in your PATH.\n' "$INSTALL_DIR"
        printf 'Add this line to ~/.bashrc or ~/.zshrc:\n'
        printf '  export PATH="$HOME/.local/bin:$PATH"\n'
        ;;
esac
