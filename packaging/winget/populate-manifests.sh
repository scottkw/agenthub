#!/usr/bin/env bash
set -euo pipefail

# populate-manifests.sh — Populate WinGet manifest templates for manual first submission.
#
# Usage: ./packaging/winget/populate-manifests.sh <VERSION> <CHECKSUMS_FILE>
#
# Arguments:
#   VERSION        Version string without v prefix (e.g., "1.8.0")
#   CHECKSUMS_FILE Path to checksums.txt downloaded from the GitHub release
#
# Output:
#   Populated YAML manifests in output/<VERSION>/ ready for manual submission.
#   Files go to manifests/s/scottkw/agenthub/<VERSION>/ in the winget-pkgs fork.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ $# -ne 2 ]; then
  echo "Usage: $0 <VERSION> <CHECKSUMS_FILE>"
  echo ""
  echo "  VERSION        Version without v prefix (e.g., 1.8.0)"
  echo "  CHECKSUMS_FILE Path to checksums.txt from the GitHub release"
  exit 1
fi

VERSION="$1"
CHECKSUMS_FILE="$2"

# Validate VERSION does not start with 'v' (WinGet PackageVersion must not have v prefix)
if [[ "$VERSION" == v* ]]; then
  echo "ERROR: VERSION must not start with 'v' (WinGet format requires bare version)"
  echo "  Got: ${VERSION}"
  echo "  Expected: ${VERSION#v}"
  exit 1
fi

# Extract WINDOWS_SHA256 from checksums file
WINDOWS_SHA256=$(grep "agenthub-v${VERSION}-windows-amd64-installer.exe" "$CHECKSUMS_FILE" | awk '{print $1}')

if [ -z "$WINDOWS_SHA256" ]; then
  echo "ERROR: Could not find SHA256 for agenthub-v${VERSION}-windows-amd64-installer.exe in ${CHECKSUMS_FILE}"
  echo ""
  echo "Available entries in checksums file:"
  cat "$CHECKSUMS_FILE"
  exit 1
fi

# Create output directory
OUTPUT_DIR="${SCRIPT_DIR}/output/${VERSION}"
mkdir -p "$OUTPUT_DIR"

# Populate each manifest template
MANIFESTS_DIR="${SCRIPT_DIR}/manifests"
for f in "${MANIFESTS_DIR}"/*.yaml; do
  BASENAME="$(basename "$f")"
  sed \
    -e "s/{{VERSION}}/${VERSION}/g" \
    -e "s/{{WINDOWS_SHA256}}/${WINDOWS_SHA256}/g" \
    "$f" > "${OUTPUT_DIR}/${BASENAME}"
done

echo ""
echo "Manifests populated successfully:"
echo "  Version:   ${VERSION}"
echo "  SHA256:    ${WINDOWS_SHA256}"
echo "  Output:    ${OUTPUT_DIR}/"
echo ""
echo "Next steps for manual submission:"
echo "  1. Run 'winget validate' on the populated manifests (Windows required)"
echo "  2. Place files at manifests/s/scottkw/agenthub/${VERSION}/ in your winget-pkgs fork"
echo "  3. Open a PR to microsoft/winget-pkgs master branch"
echo ""
echo "  Sparse clone command (PowerShell on Windows):"
echo "    git clone --filter=blob:none --no-checkout https://github.com/scottkw/winget-pkgs"
echo "    cd winget-pkgs"
echo "    git sparse-checkout set manifests\\s\\scottkw"
echo "    git checkout"
echo "    git checkout -b scottkw-agenthub-${VERSION}"
echo "    # Copy files from ${OUTPUT_DIR}/ to manifests\\s\\scottkw\\agenthub\\${VERSION}\\"
echo "    git add ."
echo "    git commit -m \"Add scottkw.agenthub version ${VERSION}\""
echo "    git push"
