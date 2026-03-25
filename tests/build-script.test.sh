#!/usr/bin/env bash
# tests/build-script.test.sh
# Behavioral tests for build.sh argument parsing, error paths, and static pattern checks.
# Does NOT require Wails, Docker, or mingw-w64.
#
# Run: bash tests/build-script.test.sh    (from project root)

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_SH="$SCRIPT_DIR/../build.sh"
PASS=0
FAIL=0

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

pass() {
  local name="$1"
  echo "  PASS: $name"
  PASS=$((PASS + 1))
}

fail() {
  local name="$1"
  local detail="${2:-}"
  echo "  FAIL: $name"
  [[ -n "$detail" ]] && echo "        $detail"
  FAIL=$((FAIL + 1))
}

assert_exit_nonzero() {
  local name="$1"
  shift
  local output
  output=$("$@" 2>&1) && { fail "$name" "expected non-zero exit, got 0. output: $output"; return; }
  pass "$name"
}

assert_exit_zero() {
  local name="$1"
  shift
  local output
  output=$("$@" 2>&1) || { fail "$name" "expected exit 0, got non-zero. output: $output"; return; }
  pass "$name"
}

# Run command, capture combined output (ignoring exit), check for substring
assert_output_contains() {
  local name="$1"
  local pattern="$2"
  shift 2
  local output
  output=$("$@" 2>&1) || true
  if echo "$output" | grep -q "$pattern"; then
    pass "$name"
  else
    fail "$name" "expected output to contain: '$pattern'. got: $output"
  fi
}

assert_file_contains_literal() {
  local name="$1"
  local pattern="$2"
  local file="$3"
  # Use -- to prevent patterns starting with -- from being parsed as grep flags
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

# ---------------------------------------------------------------------------
# Section 1: Basic file properties
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 1: File properties ==="

if [[ -f "$BUILD_SH" ]]; then
  pass "build.sh exists at project root"
else
  fail "build.sh exists at project root" "file not found: $BUILD_SH"
fi

if [[ -x "$BUILD_SH" ]]; then
  pass "build.sh is executable"
else
  fail "build.sh is executable"
fi

# ---------------------------------------------------------------------------
# Section 2: Syntax check
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 2: Syntax check ==="

assert_exit_zero "bash -n syntax check passes" bash -n "$BUILD_SH"

# ---------------------------------------------------------------------------
# Section 3: No-args exits non-zero and prints Usage:
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 3: Argument parsing — no args ==="

assert_exit_nonzero "no args exits non-zero" bash "$BUILD_SH"
assert_output_contains "no args output contains 'Usage:'" "Usage:" bash "$BUILD_SH"

# ---------------------------------------------------------------------------
# Section 4: Invalid platform exits non-zero and prints Usage: + error
# (Platform validation at lines 52-59 runs BEFORE the Wails check at 62-67,
# so no external toolchain is needed here.)
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 4: Argument parsing — invalid platform ==="

assert_exit_nonzero "--platform bogus exits non-zero" bash "$BUILD_SH" --platform bogus
assert_output_contains "--platform bogus output contains 'Usage:'" "Usage:" bash "$BUILD_SH" --platform bogus
assert_output_contains "--platform bogus output contains 'Invalid platform'" "Invalid platform" bash "$BUILD_SH" --platform bogus

# ---------------------------------------------------------------------------
# Section 5: --platform with no value prints error and exits non-zero
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 5: Argument parsing — --platform without value ==="

assert_exit_nonzero "--platform with no value exits non-zero" bash "$BUILD_SH" --platform

# Use grep without -F so the pattern is not treated as a flag
_output=$(bash "$BUILD_SH" --platform 2>&1) || true
if echo "$_output" | grep -q "requires a value"; then
  pass "--platform with no value shows 'requires a value' error"
else
  fail "--platform with no value shows 'requires a value' error" "got: $_output"
fi

# ---------------------------------------------------------------------------
# Section 6: Unknown flag exits non-zero
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 6: Argument parsing — unknown flag ==="

assert_exit_nonzero "--unknown-flag exits non-zero" bash "$BUILD_SH" --unknown-flag
assert_output_contains "--unknown-flag output contains 'Unknown flag'" "Unknown flag" bash "$BUILD_SH" --unknown-flag

# ---------------------------------------------------------------------------
# Section 7: --sign without env vars exits non-zero with clear error
#
# sign_and_notarize() is only reachable after a successful wails build.
# To test the missing-credentials error path without the actual build toolchain:
# 1. Extract just the sign_and_notarize() function body using awk
# 2. Source the extracted snippet in a subshell
# 3. Call the function directly without credentials set
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 7: --sign missing env vars error path ==="

_test_sign_missing_env_vars() {
  # Extract sign_and_notarize() function body from build.sh using awk.
  # awk collects lines from "^sign_and_notarize()" through the matching closing "^}".
  local FUNC_SNIPPET
  FUNC_SNIPPET=$(awk '/^sign_and_notarize\(\)/{found=1} found{print} found && /^\}$/{exit}' "$BUILD_SH")

  if [[ -z "$FUNC_SNIPPET" ]]; then
    fail "--sign without env vars: could not extract sign_and_notarize() from build.sh"
    return
  fi

  # Source just the function in a subshell and call it without credentials
  local output
  output=$(
    bash -c "
      $FUNC_SNIPPET
      unset MACOS_SIGNING_IDENTITY MACOS_APPLE_ID MACOS_TEAM_ID MACOS_APP_PASSWORD
      sign_and_notarize 'build/bin/agenthub.app'
    " 2>&1
  ) || true

  if echo "$output" | grep -q "Missing required environment variables"; then
    pass "--sign without env vars: output contains 'Missing required environment variables'"
  else
    fail "--sign without env vars: output contains 'Missing required environment variables'" \
         "got: $output"
  fi

  # Check all four var names appear in the error output
  for var in MACOS_SIGNING_IDENTITY MACOS_APPLE_ID MACOS_TEAM_ID MACOS_APP_PASSWORD; do
    if echo "$output" | grep -q "$var"; then
      pass "--sign without env vars: output lists missing var $var"
    else
      fail "--sign without env vars: output lists missing var $var" "got: $output"
    fi
  done
}

_test_sign_missing_env_vars

# ---------------------------------------------------------------------------
# Section 8: Static pattern checks — required function names
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 8: Static pattern checks — function names ==="

assert_file_contains_regex "contains build_macos function" \
  "^build_macos\(\)" "$BUILD_SH"

assert_file_contains_regex "contains build_windows function" \
  "^build_windows\(\)" "$BUILD_SH"

assert_file_contains_regex "contains build_linux function" \
  "^build_linux\(\)" "$BUILD_SH"

assert_file_contains_regex "contains sign_and_notarize function" \
  "^sign_and_notarize\(\)" "$BUILD_SH"

# ---------------------------------------------------------------------------
# Section 9: Static pattern checks — build tool invocations
# (Multi-line commands: check each token independently)
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 9: Static pattern checks — build tool invocations ==="

# wails build for macOS: "$WAILS" build -platform darwin/universal -clean
assert_file_contains_literal "contains 'darwin/universal'" \
  "darwin/universal" "$BUILD_SH"

assert_file_contains_literal "contains wails build -clean (macOS path)" \
  "build -platform darwin/universal -clean" "$BUILD_SH"

assert_file_contains_literal "contains x86_64-w64-mingw32-gcc reference" \
  "x86_64-w64-mingw32-gcc" "$BUILD_SH"

assert_file_contains_regex "contains docker run invocation" \
  "docker run" "$BUILD_SH"

# ---------------------------------------------------------------------------
# Section 10: Static pattern checks — notarization pipeline
# (notarytool submit and --wait are on adjacent lines — check both tokens)
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 10: Static pattern checks — notarization pipeline ==="

assert_file_contains_literal "contains notarytool submit" \
  "notarytool submit" "$BUILD_SH"

assert_file_contains_literal "contains --wait flag (critical: avoids exit-0 false success)" \
  "--wait" "$BUILD_SH"

assert_file_contains_literal "contains ditto -c -k --keepParent (not zip -r)" \
  "ditto -c -k --keepParent" "$BUILD_SH"

assert_file_contains_regex "contains spctl --assess" \
  "spctl.*--assess" "$BUILD_SH"

assert_file_contains_literal "contains stapler staple" \
  "stapler staple" "$BUILD_SH"

# ---------------------------------------------------------------------------
# Section 11: Static pattern checks — prerequisite guards
# ---------------------------------------------------------------------------
echo ""
echo "=== Section 11: Static pattern checks — prerequisite guards ==="

assert_file_contains_literal "contains docker info check" \
  "docker info" "$BUILD_SH"

# command -v check: the variable holds the compiler name; check the check pattern
assert_file_contains_literal "contains command -v prerequisite check for compiler" \
  'command -v "$MINGW_CC"' "$BUILD_SH"

assert_file_contains_regex "contains wails binary path from go env GOPATH" \
  'WAILS=.*go env GOPATH' "$BUILD_SH"

assert_file_contains_literal "contains shebang #!/usr/bin/env bash" \
  "#!/usr/bin/env bash" "$BUILD_SH"

assert_file_contains_literal "contains set -euo pipefail" \
  "set -euo pipefail" "$BUILD_SH"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "================================================"
echo "Results: $PASS passed, $FAIL failed"
echo "================================================"

if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
exit 0
