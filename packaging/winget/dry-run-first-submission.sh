#!/usr/bin/env bash
set -euo pipefail

# dry-run-first-submission.sh — Dry-run of distribute.yml's winget first-submission path.
#
# Resolves the latest release tag from the GitHub API, downloads checksums.txt,
# runs populate-manifests.sh to populate the manifest templates, and validates
# every generated YAML manifest for correctness.
#
# Usage: bash packaging/winget/dry-run-first-submission.sh
# Run from: project root OR any directory (SCRIPT_DIR resolves correctly)
#
# Prerequisites: curl, python3 (yaml module — stdlib since Python 3.x)
#
# This script is the local pre-flight for the live submission.
# See packaging/winget/FIRST-SUBMISSION-RUNBOOK.md for operator steps.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
POPULATE_SH="${SCRIPT_DIR}/populate-manifests.sh"

echo "=== winget first-submission dry-run ==="
echo ""

# --- Step 1: Resolve latest release tag (GitHub API, grep+sed — no jq required) ---
echo "Step 1: Resolving latest release tag from GitHub API..."
TAG=$(curl -fsSL "https://api.github.com/repos/scottkw/agenthub/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
if [ -z "$TAG" ]; then
    echo "ERROR: Could not resolve latest release tag from GitHub API." >&2
    echo "       Check network connectivity or try again." >&2
    exit 1
fi
echo "  Tag resolved: ${TAG}"

# Strip leading 'v' — populate-manifests.sh requires bare version (Pitfall 7).
# The script validates this and exits with an error if 'v' is present.
VERSION="${TAG#v}"
echo "  Version (no v prefix): ${VERSION}"
echo ""

# --- Step 2: Download checksums.txt for this release to a temp file ---
echo "Step 2: Downloading checksums.txt for ${TAG}..."
TMPDIR_DRY=$(mktemp -d)
trap 'rm -rf "$TMPDIR_DRY"' EXIT
CHECKSUMS_FILE="${TMPDIR_DRY}/checksums.txt"
CHECKSUMS_URL="https://github.com/scottkw/agenthub/releases/download/${TAG}/checksums.txt"
curl -fsSL "${CHECKSUMS_URL}" -o "${CHECKSUMS_FILE}"
echo "  Downloaded: ${CHECKSUMS_URL}"

# Show the Windows entry for operator visibility
WINDOWS_ENTRY=$(grep "windows-amd64-installer.exe" "${CHECKSUMS_FILE}" || true)
if [ -z "$WINDOWS_ENTRY" ]; then
    echo "ERROR: No windows-amd64-installer.exe entry found in checksums.txt." >&2
    echo "       The release may not have a Windows installer asset yet." >&2
    echo "       Run 'bash packaging/winget/dry-run-first-submission.sh' against a" >&2
    echo "       release that includes the Windows build." >&2
    exit 1
fi
echo "  Windows checksum entry: ${WINDOWS_ENTRY}"
echo ""

# --- Step 3: Run populate-manifests.sh with real release checksums ---
echo "Step 3: Running populate-manifests.sh ${VERSION} <checksums>..."
bash "${POPULATE_SH}" "${VERSION}" "${CHECKSUMS_FILE}"

OUTPUT_DIR="${SCRIPT_DIR}/output/${VERSION}"
echo ""

# --- Step 4: Validate every generated YAML with python3 yaml.safe_load ---
echo "Step 4: Validating generated YAML manifests..."

if ! ls "${OUTPUT_DIR}"/*.yaml >/dev/null 2>&1; then
    echo "ERROR: No YAML files found in ${OUTPUT_DIR}/" >&2
    exit 1
fi

python3 -c "
import yaml, glob, sys

output_dir = '${OUTPUT_DIR}'
files = sorted(glob.glob(output_dir + '/*.yaml'))
if not files:
    print('ERROR: No YAML files found in ' + output_dir, file=sys.stderr)
    sys.exit(1)

for path in files:
    with open(path) as f:
        try:
            data = yaml.safe_load(f)
        except yaml.YAMLError as e:
            print('ERROR: YAML parse error in ' + path + ': ' + str(e), file=sys.stderr)
            sys.exit(1)
    print('  YAML valid: ' + path)

print()
print('  All ' + str(len(files)) + ' manifest(s) parsed successfully.')
"
echo ""

# --- Step 5: Assert installer manifest contains required values ---
echo "Step 5: Asserting installer manifest values..."
INSTALLER_YAML="${OUTPUT_DIR}/scottkw.agenthub.installer.yaml"

if [ ! -f "${INSTALLER_YAML}" ]; then
    echo "ERROR: Installer manifest not found: ${INSTALLER_YAML}" >&2
    exit 1
fi

FAIL=0
pass_check() { echo "  PASS: $1"; }
fail_check() { echo "  FAIL: $1" >&2; FAIL=$((FAIL + 1)); }

# Assert PackageIdentifier
if grep -q "PackageIdentifier: scottkw.agenthub" "${INSTALLER_YAML}"; then
    pass_check "PackageIdentifier 'scottkw.agenthub' present"
else
    fail_check "PackageIdentifier 'scottkw.agenthub' NOT found in installer manifest"
fi

# Assert windows installer URL contains the required filename pattern
if grep -q "windows-amd64-installer.exe" "${INSTALLER_YAML}"; then
    pass_check "windows-amd64-installer.exe URL present"
else
    fail_check "windows-amd64-installer.exe URL NOT found in installer manifest"
fi

# Assert VERSION (without v prefix) appears in the installer manifest
if grep -q "\"${VERSION}\"" "${INSTALLER_YAML}"; then
    pass_check "version '${VERSION}' present in installer manifest"
else
    fail_check "version '${VERSION}' NOT found in installer manifest"
fi

if [ "${FAIL}" -gt 0 ]; then
    echo "" >&2
    echo "ERROR: ${FAIL} assertion(s) failed — the generated manifests do not have the expected values." >&2
    exit 1
fi

echo ""
echo "=== PASS: winget first-submission dry-run complete ==="
echo ""
echo "  Tag:     ${TAG}"
echo "  Version: ${VERSION}"
echo "  Output:  ${OUTPUT_DIR}/"
echo ""
echo "The generated manifests contain correct values. The live submission"
echo "requires operator setup. See:"
echo "  packaging/winget/FIRST-SUBMISSION-RUNBOOK.md"
