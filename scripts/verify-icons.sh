#!/usr/bin/env bash
set -e

PASS=0
FAIL=0
ERRORS=""

check() {
  local desc="$1"
  shift
  if eval "$@" >/dev/null 2>&1; then
    echo "  PASS: $desc"
    PASS=$((PASS+1))
  else
    echo "  FAIL: $desc"
    ERRORS="$ERRORS\n  - $desc"
    FAIL=$((FAIL+1))
  fi
}

echo "=== AgentHub Icon Verification ==="
echo ""

# SC1: appicon.png is 1024x1024
check "appicon.png is 1024x1024" \
  "identify build/appicon.png | grep -q '1024x1024'"

# SC2: iconfile.icns exists and has content
check "iconfile.icns exists" \
  "test -s build/darwin/iconfile.icns"

# SC2: iconfile.icns has multiple entries (check file size > 100KB as proxy for 10 entries)
check "iconfile.icns has substantial content (>100KB = multi-entry)" \
  "test $(stat -f%z build/darwin/iconfile.icns 2>/dev/null || stat -c%s build/darwin/iconfile.icns 2>/dev/null) -gt 100000"

# SC3: icon.ico exists and has multiple frames
check "icon.ico exists" \
  "test -s build/windows/icon.ico"

check "icon.ico has >=4 frames" \
  "test $(identify build/windows/icon.ico 2>/dev/null | wc -l) -ge 4"

# SC4: Linux PNGs
for size in 16 32 48 128 256 512; do
  check "Linux ${size}x${size}.png exists" \
    "test -f build/linux/${size}x${size}.png"
done

# SC5: Title logo in frontend assets
check "frontend title logo exists" \
  "test -f frontend/src/assets/agenthub-title-logo.png"

# iconset folder has 10 entries
check "AppIcon.iconset has 10 entries" \
  "test $(ls build/AppIcon.iconset/*.png 2>/dev/null | wc -l) -eq 10"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
echo "All checks passed."
