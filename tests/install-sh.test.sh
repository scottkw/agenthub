#!/usr/bin/env bash
# tests/install-sh.test.sh
# Shellcheck gate for scripts/install.sh
# Run: bash tests/install-sh.test.sh (from project root)
set -uo pipefail

PASS=0
FAIL=0
SCRIPT="scripts/install.sh"

pass() { echo "  PASS: $1"; PASS=$((PASS+1)); }
fail() { echo "  FAIL: $1"; FAIL=$((FAIL+1)); }

echo "=== install.sh checks ==="

# SC-1: file exists and is non-empty
if [ -s "$SCRIPT" ]; then pass "install.sh exists and is non-empty"
else fail "install.sh exists and is non-empty"; fi

# SC-2: shellcheck (POSIX sh mode)
if command -v shellcheck >/dev/null 2>&1; then
    if shellcheck -S warning --shell=sh "$SCRIPT"; then
        pass "shellcheck clean (--shell=sh)"
    else
        fail "shellcheck reported warnings/errors"
    fi
else
    echo "  SKIP: shellcheck not installed (skipping SC-2)"
fi

# SC-3: bash -n syntax check as fallback
if bash -n "$SCRIPT" 2>/dev/null; then pass "bash -n syntax check passes"
else fail "bash -n syntax check failed"; fi

# SC-4: contains required patterns
assert_literal() {
    local desc="$1" pattern="$2"
    if grep -qF -- "$pattern" "$SCRIPT"; then pass "$desc"
    else fail "$desc"; fi
}
assert_regex() {
    local desc="$1" pattern="$2"
    if grep -qE "$pattern" "$SCRIPT"; then pass "$desc"
    else fail "$desc"; fi
}

assert_literal "contains uname -m" "uname -m"
assert_literal "contains x86_64 arch check" "x86_64"
assert_literal "contains SHA256 verify step" "sha256sum"
assert_literal "contains /usr/local/bin install path" "/usr/local/bin"
assert_literal "contains .local/bin user-mode path" ".local/bin"
assert_literal "contains trap for cleanup" "trap"
assert_regex  "contains GitHub Releases API URL" "api.github.com/repos/scottkw/agenthub"
assert_regex  "contains sha256 mismatch error message" "[Mm]ismatch"

# WR-01: checksum-line grep must use -F (tarball dots are literal, not regex wildcards)
assert_literal "WR-01: checksum grep uses -F for exact tarball match" 'grep -F "${TARBALL}" "${TMPDIR}/checksums.txt"'

# WR-03: both the root and non-root install-dir branches must have mkdir -p
# Verify by counting occurrences — expect 2 (root + non-root)
WR03_COUNT=$(grep -cF 'mkdir -p "$INSTALL_DIR"' "$SCRIPT" || true)
if [ "$WR03_COUNT" -ge 2 ]; then pass "WR-03: both install-dir branches contain mkdir -p"
else fail "WR-03: both install-dir branches contain mkdir -p (found $WR03_COUNT, need 2)"; fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ] || exit 1
exit 0
