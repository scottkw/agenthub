#!/usr/bin/env bash
# tests/check-traceability-paths.sh
# Verify every test path in TESTING.md traceability table still exists on disk.
# Fails loudly if any path is missing — prevents renamed/deleted tests from
# silently making the traceability map lie.
#
# Parsing convention: traceability table rows have format:
#   | REQ-ID | path/to/test_file.go | group |
# Lines starting with '|' and containing a file extension are path rows.
#
# Run: bash tests/check-traceability-paths.sh    (from project root)

set -euo pipefail

if [[ ! -f "TESTING.md" ]]; then
  echo "TESTING.md not present yet — skipping path check"
  exit 0
fi

# Extract repo-relative test paths from the traceability table.
# Matches the second column of '| ... | path.ext | ... |' rows.
# grep -oP is used on ubuntu-latest (Linux CI); macOS grep does not support -P.
FAIL=0
while IFS= read -r path; do
  if [[ ! -e "$path" ]]; then
    echo "MISSING traceability path: $path"
    FAIL=$((FAIL + 1))
  fi
done < <(grep -oP '(?<=\| )[^\|]+\.(?:go|ts|tsx|sh)(?= \|)' TESTING.md | tr -d ' ')

if [[ $FAIL -gt 0 ]]; then
  echo "ERROR: $FAIL traceability path(s) missing — update TESTING.md or restore the test file"
  exit 1
fi
echo "OK: all traceability paths exist"
