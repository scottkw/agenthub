#!/usr/bin/env bash
# grep-gate.sh — Phase 90 SEC-09 + SEC-10 regression guard
#
# Asserts zero floating-ref, @latest, or non-SHA action references in
# .github/workflows/, build.sh, and tests/.
#
# EXPECTED to FAIL during Phase 90 Waves 1-4; passes once all SHA-pin
# work lands (end of Wave 4). Becomes part of CI after the hardening-check
# workflow step is added (Plan 03 or Plan 04).

set -euo pipefail

echo "==> Checking for unpinned @main / @master / @latest refs..."

# Check 1: Floating-ref check
# Any uses: line whose ref ends with @main, @master, @vMajor, or plain-word ref — reject
BAD=$(grep -rEn 'uses:\s*[^#]*@(main|master|v[0-9]+|[a-z]+)$' .github/workflows/ || true)
if [[ -n "$BAD" ]]; then
  echo "FAIL: unpinned action refs found:"
  echo "$BAD"
  exit 1
fi

# Check 2: @latest check
# Catches bare @latest refs in tool installs (e.g. go install tool@latest) and
# workflow steps. Excludes lines where @latest appears only as a quoted string
# literal (e.g. grep pattern strings in test assertions, comments, echo messages).
# Pattern: require a word character immediately before @latest — bare refs like
# tool@latest or action@latest match; quoted-string mentions like '@latest' or
# "contains @latest" do not.
LATEST=$(grep -rEn '\w@latest' .github/workflows/ build.sh tests/ || true)
if [[ -n "$LATEST" ]]; then
  echo "FAIL: @latest references found:"
  echo "$LATEST"
  exit 1
fi

# Check 3: Non-SHA action ref check
# Negation check — every uses: line must match 40-char SHA
NON_SHA=$(grep -rE 'uses:\s*[^ ]+@' .github/workflows/ \
  | grep -Ev 'uses:\s*[^ ]+@[a-f0-9]{40}(\s|$)' || true)
if [[ -n "$NON_SHA" ]]; then
  echo "FAIL: non-SHA action refs (likely @v4 tags):"
  echo "$NON_SHA"
  exit 1
fi

echo "PASS: all action refs are SHA-pinned"
