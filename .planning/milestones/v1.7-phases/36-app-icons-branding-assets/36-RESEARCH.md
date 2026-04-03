# Phase 36: App Icons & Branding Assets - Research

**Researched:** 2026-03-31
**Domain:** macOS/Windows/Linux icon formats, Wails v2 build pipeline, ImageMagick, iconutil
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BRND-01 | App icon set generated from logomark: .icns (macOS), .ico (Windows), multi-size PNGs (Linux/Wails) | Wails v2.10.2 packager source confirmed; icon pipeline fully documented below |
</phase_requirements>

---

## Summary

Phase 36 produces the branded icon assets that replace the placeholder blue "A" box currently in `build/appicon.png`. The deliverables are: a 1024x1024 square logomark PNG (source of truth), a properly-sized `AppIcon.icns` built via `iconutil` with all 10 standard macOS sizes, a `build/windows/icon.ico` containing at minimum 16/32/48/256px sizes, multi-size PNGs in `build/linux/`, and the full title logo copied to `frontend/src/assets/`.

**Key architectural finding:** Wails v2.10.2 auto-generates the ICNS at build time from `build/appicon.png` using `jackmordaunt/icns@v1.0.0` which only encodes 3 sizes (1024/512/256). The success criterion requires 10 sizes including 1024x1024@2x Retina-ready. This means the ICNS must be pre-built via `iconutil` from a hand-crafted `.iconset/` folder and placed at `build/darwin/AppIcon.icns` — the Wails auto-generation path must be bypassed. The `Info.plist` references `CFBundleIconFile = "iconfile"`, so the pre-built file must be named `iconfile.icns` or the plist value updated.

**Critical asset gap:** The only brand asset is `docs/agenthub-title-logo.png` (805x208, light background, no transparency). It shows the "A" logomark on the left side and "AgentHub" wordmark on the right. A **square 1024x1024 logomark PNG** does not exist and must be created before icon generation can proceed. This is the phase's primary creative dependency.

**Primary recommendation:** Create a 1024x1024 RGBA square logomark PNG (transparent or dark-navy background with the "A" mark), generate the `.iconset/` folder with all 10 sizes using ImageMagick + sips, compile via `iconutil`, and pre-place the ICNS in `build/darwin/` so Wails embeds it directly rather than auto-generating.

## Project Constraints (from CLAUDE.md)

- Node package manager: `pnpm` preferred
- Python: use virtual environment, never install globally
- JS/TS: TypeScript types, ESLint + Prettier
- No unrelated changes to active code paths
- Chesterton's Fence: don't touch `Info.plist` `CFBundleIconFile` value without understanding — it is currently `"iconfile"` and Wails hardcodes output to `iconfile.icns`; changing this would break Wails auto-generation for other icons (file associations)

---

## Standard Stack

### Core

| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| `iconutil` | macOS built-in `/usr/bin/iconutil` | Compile `.iconset/` folder to `.icns` | Apple's official tool; produces Apple-spec compliant ICNS with all 10 sizes |
| `sips` | macOS built-in `/usr/bin/sips` | Resize PNGs for iconset | Built into macOS; no install required; handles HiDPI correctly |
| `convert` (ImageMagick 7) | 7.1.2-18 (verified installed at `/opt/homebrew/bin/convert`) | Multi-size PNG generation; ICO assembly | Industry-standard image processing; supports -resize, ICO multi-frame output |
| Pillow (Python) | system-installed (verified) | Optional scripting for letterboxing/padding | Available in dev environment |

### Supporting

| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| `wails build` | v2.10.2 (go.mod) | Final assembly — embeds `iconfile.icns` into .app bundle | Production build verification |
| `build/gen_icon.go` | existing | Reference only — shows placeholder geometry | Do NOT run; this generated the placeholder |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `iconutil` + sips | `jackmordaunt/icns` (Wails auto-gen) | Auto-gen only produces 3 sizes (1024/512/256); fails success criterion for 10 sizes |
| `iconutil` + sips | rsvg-convert + inkscape | Overkill; source is PNG not SVG |
| Pre-built ICO | Wails auto-gen ICO | Wails only generates ICO if `build/windows/icon.ico` does NOT exist (line 203 packager.go); pre-placing it is safe and gives control over sizes |

**Installation:** All required tools are already present on the dev machine. No installs needed.

---

## Architecture Patterns

### How Wails v2.10.2 Icon Pipeline Works (VERIFIED from source)

**Source:** `/Users/ken/go/pkg/mod/github.com/wailsapp/wails/v2@v2.10.2/pkg/commands/build/packager.go`

**macOS path:**
```
build/appicon.png
  → processDarwinIcon() reads via buildassets.ReadFile(projectData, "appicon.png")
    = filepath.Join(projectData.GetBuildDir(), "appicon.png")
    = build/appicon.png
  → jackmordaunt/icns.Encode() → generates ONLY 3 sizes (1024, 512, 256) based on largest side
  → writes to: {bundle}/Contents/Resources/iconfile.icns
```

**Key insight:** Wails does NOT look for a pre-existing `.icns` file in `build/darwin/`. It always re-encodes from `build/appicon.png`. To bypass this and use a full 10-size ICNS, the approach is:

**Option A (recommended):** Place the full `iconfile.icns` at `build/darwin/iconfile.icns` and add a `wails build` post-hook or use a build script that copies it into the bundle after Wails generates the 3-size version. **OR**

**Option B (simpler, verified):** Since Wails reads `build/appicon.png` and writes into the bundle, and the success criterion says "AppIcon.icns contains all 10 required size/density entries" — the target is the bundled ICNS inside `build/bin/agenthub.app/Contents/Resources/iconfile.icns`. The workaround is: run `wails build`, then **replace** `build/bin/agenthub.app/Contents/Resources/iconfile.icns` with our full 10-size ICNS as a post-build step. But this breaks notarization/re-signing.

**Option C (cleanest):** Provide a 1024x1024 `build/appicon.png` as source. The `jackmordaunt/icns` library generates sizes `sizesFrom(1024)` = [1024, 512, 256]. It maps: `ic10`=1024, `ic14`=512, `ic13`=256. This still only gives 3 ICNS entries. NOT sufficient for all 10.

**Option D (production-correct, recommended):** Use the `wails build -tags wailsassets` production build (required per project MEMORY.md), then overwrite the ICNS in-place inside the `.app` bundle. The ICNS is inside the bundle Resources, not signed separately on macOS before notarization — it can be overwritten pre-notarization.

**Simplest production approach:** Pre-generate `AppIcon.icns` using `iconutil` and overwrite after `wails build`. The SUCCESS criterion verifies the built `.app` bundle, not what's in `build/darwin/`.

**Windows path:**
```
build/appicon.png
  → generateIcoFile() checks if build/windows/icon.ico EXISTS
  → If EXISTS: SKIPS generation (uses pre-placed file as-is)
  → If NOT EXISTS: generates from appicon.png with sizes [256, 128, 64, 48, 32, 16]
```
Current `build/windows/icon.ico` exists with 6 sizes (256/128/64/48/32/16). It passes the success criterion (needs 16/32/48/256). However, it was generated from the placeholder. Must be **deleted** before replacing `appicon.png`, or replaced directly.

**Linux path:**
```
packageApplicationForLinux() → returns nil (no-op)
```
Linux icons are NOT embedded by Wails packaging. They must either:
1. Be placed as files in `build/linux/` (used by distribution packaging scripts, `.desktop` files)
2. Be passed via `options.Linux{Icon: []byte{...}}` in `main.go` for the GTK window icon at runtime

### Recommended File Layout After Phase 36

```
build/
├── appicon.png              # 1024x1024 square logomark (NEW - replaces 256x256 placeholder)
├── AppIcon.iconset/         # Intermediate folder for iconutil (can be deleted after)
│   ├── icon_16x16.png
│   ├── icon_16x16@2x.png
│   ├── icon_32x32.png
│   ├── icon_32x32@2x.png
│   ├── icon_128x128.png
│   ├── icon_128x128@2x.png
│   ├── icon_256x256.png
│   ├── icon_256x256@2x.png
│   ├── icon_512x512.png
│   └── icon_512x512@2x.png
├── darwin/
│   ├── Info.plist           # unchanged
│   └── Info.dev.plist       # unchanged
├── windows/
│   └── icon.ico             # REGENERATED from new logomark (delete old, let Wails rebuild OR replace directly)
└── linux/
    ├── 16x16.png
    ├── 32x32.png
    ├── 48x48.png
    ├── 128x128.png
    ├── 256x256.png
    └── 512x512.png

frontend/
└── src/
    └── assets/
        └── agenthub-title-logo.png    # copied from docs/
```

### Pattern 1: Generate Full iconset with sips + iconutil

```bash
# Source: Apple Developer Documentation + macOS iconutil man page (verified)
# Assumes: 1024x1024 source at build/appicon.png

mkdir -p build/AppIcon.iconset

# Standard 10 entries (5 logical sizes × 2 pixel densities)
sips -z 16 16     build/appicon.png --out build/AppIcon.iconset/icon_16x16.png
sips -z 32 32     build/appicon.png --out build/AppIcon.iconset/icon_16x16@2x.png
sips -z 32 32     build/appicon.png --out build/AppIcon.iconset/icon_32x32.png
sips -z 64 64     build/appicon.png --out build/AppIcon.iconset/icon_32x32@2x.png
sips -z 128 128   build/appicon.png --out build/AppIcon.iconset/icon_128x128.png
sips -z 256 256   build/appicon.png --out build/AppIcon.iconset/icon_128x128@2x.png
sips -z 256 256   build/appicon.png --out build/AppIcon.iconset/icon_256x256.png
sips -z 512 512   build/appicon.png --out build/AppIcon.iconset/icon_256x256@2x.png
sips -z 512 512   build/appicon.png --out build/AppIcon.iconset/icon_512x512.png
sips -z 1024 1024 build/appicon.png --out build/AppIcon.iconset/icon_512x512@2x.png

iconutil -c icns build/AppIcon.iconset -o build/darwin/AppIcon.icns
```

### Pattern 2: Generate ICO with ImageMagick (multi-frame)

```bash
# Source: ImageMagick documentation — ICO multi-frame output
# Required sizes per success criterion: 16, 32, 48, 256
# Generate all 6 sizes matching Wails default for consistency

convert build/appicon.png \
  \( -clone 0 -resize 16x16 \) \
  \( -clone 0 -resize 32x32 \) \
  \( -clone 0 -resize 48x48 \) \
  \( -clone 0 -resize 64x64 \) \
  \( -clone 0 -resize 128x128 \) \
  \( -clone 0 -resize 256x256 \) \
  -delete 0 build/windows/icon.ico
```

Note: Delete the pre-existing `build/windows/icon.ico` first, OR replace in-place. Since Wails skips ICO generation when file exists, pre-placing is safe.

### Pattern 3: Generate Linux PNGs

```bash
# Source: freedesktop.org icon spec (48px standard sizes)
mkdir -p build/linux
for size in 16 32 48 128 256 512; do
  sips -z $size $size build/appicon.png --out build/linux/${size}x${size}.png
done
```

### Pattern 4: Create 1024x1024 Square Logomark from Title Logo

The title logo (805x208 RGBA, light grey background) contains the "A" logomark on the left side (approximately x=0-208 region based on pixel scanning). The approach for the square logomark:

**Option A — Extract and center on dark background:**
```bash
# Crop the logomark region (approx 208x208 from left) then place on 1024x1024 dark navy canvas
convert docs/agenthub-title-logo.png \
  -crop 208x208+0+0 \               # extract left square
  -background '#1a237e' \            # dark navy (matches app background 0x1a,0x1b,0x26)
  -gravity center \
  -extent 1024x1024 \
  build/appicon.png
```

**Option B — Letterbox entire title logo on dark square (simpler, less ideal for small sizes):**
```bash
convert docs/agenthub-title-logo.png \
  -background '#1a1b26' \
  -gravity center \
  -extent 1024x1024 \
  build/appicon.png
```

Option B is NOT recommended — at 16x16 the wordmark becomes unreadable and the icon will look like a rectangle.

**Recommended approach:** Option A with crop/center, but the exact crop coordinates for the "A" mark must be verified visually. The pixel scan shows the logomark "A" blue pixels appear around x=80-160, suggesting it may be more centered than a pure left-crop would capture. The executor should **visually inspect** and adjust crop rectangle.

**Alternative if logomark cannot be cleanly extracted:** Use the placeholder geometry (`build/gen_icon.go` describes a blue "A" on dark navy at 256x256) and scale it up. But the real brand asset is strongly preferred.

### Anti-Patterns to Avoid

- **Do not rely on Wails auto-generation for ICNS:** `jackmordaunt/icns@v1.0.0` only produces 3 sizes; success criterion requires 10
- **Do not delete `build/windows/icon.ico` and rely on Wails rebuild:** Wails generates ICO only if file does not exist at build time; if we want our own sizes, either pre-place or delete before `wails build`
- **Do not place ICNS at `build/darwin/icons.icns`:** Wails does NOT look here for darwin icons; it generates from `build/appicon.png` into the bundle
- **Do not letterbox the wordmark (805x208) without padding:** At 16x16, "AgentHub" text becomes invisible; use only the logomark
- **Do not create `frontend/src/assets/` without verifying Vite config:** The asset path must be importable as a static asset; since no `publicDir` override is configured in `vite.config.ts`, Vite's default `public/` dir handles truly static files, but `src/assets/` works for import-referenced files

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ICNS assembly | Custom Go/Python ICNS writer | `iconutil` (macOS built-in) | Apple's own format; iconutil handles all chunk types, compression, and retina metadata correctly |
| PNG resizing | Manual pixel scaling | `sips` or `convert -resize` | Lanczos resampling, ICC profile preservation, correct alpha handling |
| ICO multi-frame | Custom ICO binary packer | `convert` (ImageMagick) | ICO format edge cases: Windows 10 requires PNG-encoded frames for 256px; ImageMagick handles this automatically |
| Logo extraction | Manual pixel editing | ImageMagick `-crop` + `-gravity center` | Repeatable, scriptable, deterministic |

---

## Common Pitfalls

### Pitfall 1: Source Image Too Small for iconutil
**What goes wrong:** `iconutil` fails or produces blurry icons if source PNG is smaller than required output (e.g., 256x256 source → 512x512 entry = upscaling artifacts)
**Why it happens:** `sips` upscales with bilinear; small sources produce noticeably blurry large icons
**How to avoid:** Source `appicon.png` MUST be 1024x1024 before generating the iconset
**Warning signs:** iconset entries at 512x512 look blurry compared to 256x256

### Pitfall 2: Wails Overwrites Pre-Placed ICNS
**What goes wrong:** You place a hand-crafted ICNS in `build/darwin/` — but Wails doesn't look there; it re-generates from `build/appicon.png` into the bundle's Resources directory every build
**Why it happens:** Wails `processDarwinIcon()` hardcodes output path to `{bundle}/Contents/Resources/iconfile.icns` and always regenerates
**How to avoid:** Either (a) post-process the bundle after `wails build` to overwrite `iconfile.icns`, or (b) accept the 3-size ICNS from Wails and verify success criterion is met with what `jackmordaunt` produces from a 1024x1024 source (it will include 1024/512/256 — but NOT the small sizes 16/32/128 required by Apple HIG)
**Warning signs:** Running `wails build` and then checking the bundle ICNS shows only 3 chunk types (ic10, ic14, ic13)

### Pitfall 3: Windows icon.ico Not Regenerated
**What goes wrong:** Wails skips ICO generation if `build/windows/icon.ico` already exists (line 203 packager.go: `if !fs.FileExists(icoFile)`)
**Why it happens:** Wails treats pre-existing ICO as intentional; won't overwrite
**How to avoid:** Delete `build/windows/icon.ico` before running `wails build`, OR replace it manually with the new branded ICO before building
**Warning signs:** After replacing `appicon.png` and running `wails build`, the app EXE still shows old placeholder icon

### Pitfall 4: Grey Background in Logomark
**What goes wrong:** The title logo has a solid light grey background (RGBA ~223,224,226,255 — fully opaque). If used directly, icons will show a grey box at all sizes.
**Why it happens:** The title logo PNG is designed for light-background display, not as an icon asset
**How to avoid:** Either (a) create a new transparent-background logomark, (b) replace grey background with transparency using `convert -fuzz 10% -transparent '#dfe0e2' ...`, or (c) use a dark navy background that matches the app's theme
**Warning signs:** Generated icon appears with grey/white square background visible in Finder/Dock

### Pitfall 5: Vite Asset Import vs Public Dir
**What goes wrong:** Image copied to `frontend/src/assets/` but referenced as `/agenthub-title-logo.png` in HTML fails at runtime
**Why it happens:** Vite `src/assets/` files must be imported via ES module syntax; only files in `public/` are served as static paths
**How to avoid:** Copy title logo to `frontend/src/assets/`; it will be used as an ES import in the splash screen component (Phase 37), not as a direct URL. The copy in Phase 36 is prep work for Phase 37. No code in Phase 36 imports it.
**Warning signs:** 404 errors in dev mode when accessing `/agenthub-title-logo.png` directly

### Pitfall 6: ICNS 1024x1024@2x Entry
**What goes wrong:** The `icon_512x512@2x.png` entry in the iconset IS the 1024x1024@2x entry — the @2x files are physically larger (512@2x = 1024px, 256@2x = 512px). Confusion leads to missing the Retina entry.
**Why it happens:** Apple's iconset naming is `{logical_size}@{scale}` but physical pixels = logical × scale
**How to avoid:** The iconset table above correctly maps: `icon_512x512@2x.png` = 1024px actual = the "1024x1024@2x Retina" entry

---

## Code Examples

### Full iconset generation script (production-quality)
```bash
# Source: Apple iconutil man page + macOS developer documentation
# Run from project root: bash build/gen_icns.sh

set -e
SRC="build/appicon.png"
ICONSET="build/AppIcon.iconset"
OUT="build/darwin/iconfile.icns"  # matches CFBundleIconFile="iconfile" in Info.plist

mkdir -p "$ICONSET"

sips -z 16   16   "$SRC" --out "$ICONSET/icon_16x16.png"
sips -z 32   32   "$SRC" --out "$ICONSET/icon_16x16@2x.png"
sips -z 32   32   "$SRC" --out "$ICONSET/icon_32x32.png"
sips -z 64   64   "$SRC" --out "$ICONSET/icon_32x32@2x.png"
sips -z 128  128  "$SRC" --out "$ICONSET/icon_128x128.png"
sips -z 256  256  "$SRC" --out "$ICONSET/icon_128x128@2x.png"
sips -z 256  256  "$SRC" --out "$ICONSET/icon_256x256.png"
sips -z 512  512  "$SRC" --out "$ICONSET/icon_256x256@2x.png"
sips -z 512  512  "$SRC" --out "$ICONSET/icon_512x512.png"
sips -z 1024 1024 "$SRC" --out "$ICONSET/icon_512x512@2x.png"

iconutil -c icns "$ICONSET" -o "$OUT"
echo "Generated: $OUT"
```

### Post-build ICNS injection (to bypass Wails 3-size limit)
```bash
# After wails build: overwrite the bundle's auto-generated ICNS with our full 10-size version
BUNDLE="build/bin/agenthub.app"
cp build/darwin/iconfile.icns "$BUNDLE/Contents/Resources/iconfile.icns"
# Force macOS to reload icon cache
touch "$BUNDLE"
```

### Verify ICNS contents (Python)
```python
import struct

def list_icns_chunks(path):
    with open(path, 'rb') as f:
        data = f.read()
    assert data[:4] == b'icns', "Not an ICNS file"
    pos = 8
    chunks = []
    while pos < len(data):
        ostype = data[pos:pos+4].decode('ascii')
        size = struct.unpack('>I', data[pos+4:pos+8])[0]
        chunks.append((ostype, size))
        pos += size
    return chunks

# Expected for full 10-size ICNS:
# icp4=16px, icp5=32px, icp6=64px, ic07=128px, ic08=256px,
# ic09=512px, ic10=1024px (also ic11=32@2x, ic12=64@2x, ic13=256@2x, ic14=512@2x)
# iconutil's actual output OSTypes depend on macOS version
```

### Verify ICO contents
```python
import struct

def list_ico_frames(path):
    with open(path, 'rb') as f:
        data = f.read()
    count = struct.unpack('<H', data[4:6])[0]
    sizes = []
    for i in range(count):
        off = 6 + i*16
        w = data[off] or 256
        h = data[off+1] or 256
        sizes.append((w, h))
    return sizes
```

---

## Runtime State Inventory

This phase is purely asset generation (PNG/ICNS/ICO files + one file copy). No runtime state categories apply.

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | None — verified | N/A |
| Live service config | None — verified | N/A |
| OS-registered state | None — verified | N/A |
| Secrets/env vars | None — verified | N/A |
| Build artifacts | `build/appicon.png` (256x256 placeholder), `build/windows/icon.ico` (6-size placeholder) | Replace both; delete ICO before wails build or pre-place new one |

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `iconutil` | macOS ICNS generation | Yes | macOS built-in | None (macOS-only; must run on macOS dev machine) |
| `sips` | PNG resizing for iconset | Yes | macOS built-in | `convert -resize` (ImageMagick) |
| `convert` (ImageMagick) | ICO generation, PNG processing | Yes | 7.1.2-18 at `/opt/homebrew/bin/convert` | None needed — installed |
| `python3` + Pillow | Optional logo analysis/manipulation | Yes | Pillow verified available | Not required |
| `wails build -tags wailsassets` | Production build verification | Yes | v2.10.2 | N/A |

**Missing dependencies with no fallback:** None — all tools available.

**Missing dependencies with fallback:** None applicable.

**Platform note:** `iconutil` and `sips` are macOS-only. The `.icns` generation script must run on macOS. The repo targets macOS (dev machine is macOS Darwin 25.3.0 arm64). Linux cross-compilation would require a different approach, but is out of scope.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Generate ICNS programmatically in Go | Use `iconutil` (native macOS tool) | Always best practice | iconutil produces Apple-valid ICNS with all standard chunk types |
| Single-size appicon.png (256px) | 1024x1024 source for all generations | macOS HiDPI since 2012 | Small source = blurry Retina icons |
| Inline icon in binary | Pre-placed asset in `.app` bundle Resources | N/A for Wails | Wails handles bundle assembly |

**Deprecated/outdated:**
- `build/gen_icon.go`: Generated placeholder. After Phase 36, this file is superseded by the real logomark. Can be kept as documentation artifact or deleted.
- The existing 256x256 `build/appicon.png`: Will be replaced with 1024x1024 branded version.

---

## Validation Architecture

nyquist_validation key is absent from `.planning/config.json` — treating as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest (frontend) + manual shell verification (icon assets) |
| Config file | `frontend/vite.config.ts` |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test:coverage` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BRND-01 SC1 | macOS .app bundle shows branded icon | manual | `open build/bin/agenthub.app` → inspect Dock | N/A |
| BRND-01 SC2 | AppIcon.icns has 10 size/density entries | automated shell | `python3 -c "..."` (see Code Examples) | Wave 0 |
| BRND-01 SC3 | icon.ico has ≥4 sizes (16,32,48,256) | automated shell | `python3 -c "..."` (see Code Examples) | Wave 0 |
| BRND-01 SC4 | Linux PNGs present in build/linux/ | automated shell | `ls build/linux/*.png \| wc -l` ≥ 6 | Wave 0 |
| BRND-01 SC5 | Title logo in frontend/src/assets/ | automated shell | `test -f frontend/src/assets/agenthub-title-logo.png` | Wave 0 |

### Sampling Rate
- **Per task commit:** Shell file-existence checks (instant)
- **Per wave merge:** Full verification script + visual Dock/Finder inspection
- **Phase gate:** All 5 success criteria verified before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `build/verify_icons.sh` — runs all automated checks for SC2-SC5
- [ ] Manual verification checklist in PLAN.md for SC1 (Dock icon visual)

---

## Open Questions

1. **Exact logomark crop coordinates**
   - What we know: Title logo is 805x208 with grey background; blue "A" mark pixels visible around x=80-200 range in scan
   - What's unclear: The exact bounding box of the logomark without padding; whether the mark looks correct at 16x16 after crop
   - Recommendation: Executor should do a test crop, generate a 16x16 preview, and inspect visually before committing. May need minor padding adjustments.

2. **Background color for icon square**
   - What we know: App background is `#1a1b26` (dark navy); the "A" mark uses blue `#1751ca` approximately
   - What's unclear: Whether a transparent background or dark navy background is better for macOS Finder/Dock display (macOS adds rounded corners and drop shadow to icons with transparency)
   - Recommendation: Use transparent RGBA background — macOS applies standardized icon treatment (rounded corners, shadow) to transparent PNG icons, producing the most native appearance

3. **ICNS injection vs accepting Wails 3-size output**
   - What we know: Success criterion requires 10 sizes including 1024x1024@2x; Wails auto-gen produces 3
   - What's unclear: Whether the success criterion strictly requires the ICNS file to have 10 chunks, or if "10 required size/density entries" means the iconset folder is the artifact
   - Recommendation: The success criterion says "AppIcon.icns contains all 10 required size/density entries." The most straightforward interpretation is the ICNS file itself. Use the post-build injection pattern (Pattern 2 above) to overwrite the bundle's auto-generated ICNS.

---

## Sources

### Primary (HIGH confidence)
- Wails v2.10.2 packager source at `/Users/ken/go/pkg/mod/github.com/wailsapp/wails/v2@v2.10.2/pkg/commands/build/packager.go` — confirmed icon pipeline behavior, file paths, ICO skip-if-exists logic
- `jackmordaunt/icns@v1.0.0` source at `/Users/ken/go/pkg/mod/github.com/jackmordaunt/icns@v1.0.0/icns.go` — confirmed only 3 sizes generated (1024/512/256)
- macOS `iconutil` man page — confirmed `.iconset/` format and compile command
- `build/darwin/Info.plist` — confirmed `CFBundleIconFile = "iconfile"` value
- `build/windows/icon.ico` — confirmed existing 6-size ICO (256/128/64/48/32/16) from placeholder
- `docs/agenthub-title-logo.png` — confirmed 805x208 RGBA, grey background, not transparent

### Secondary (MEDIUM confidence)
- Wails GitHub issues #1431 and #3860 — confirmed ICO skip-if-exists behavior and ICNS naming
- WebSearch: Wails v2 icon build directory conventions — consistent with source code findings

### Tertiary (LOW confidence)
- None — all critical claims verified from source code

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified installed tools, Wails source confirmed
- Architecture: HIGH — read actual Wails packager.go source
- Pitfalls: HIGH — derived from source code behavior, not hearsay
- Logomark extraction: MEDIUM — pixel-level analysis done, exact crop needs visual validation

**Research date:** 2026-03-31
**Valid until:** 2026-06-30 (stable macOS tooling; Wails v2.10.2 pinned in go.mod)
