//go:build ignore

// gen_progress_icons.go generates four 18×18 PNG tray quartile glyphs for Phase 98 PRG-03.
//
// Usage:
//
//	cd /path/to/agenthub && go run build/gen_progress_icons.go
//
// Output: assets/tray_icon_progress_{25,50,75,100}.png
//
// Each icon is a copy of assets/tray_icon.png with a horizontal accent bar
// overlaid at the bottom. The bar fills a fraction of the icon width proportional
// to the quartile (25/50/75/100%). Color: TokyoNight accent #7aa2f7 (RGB 122, 162, 247).
//
// The //go:build ignore tag prevents this script from being compiled into the binary.
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Determine repo root from current working directory.
	// Script is invoked from repo root: go run build/gen_progress_icons.go
	repoRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("os.Getwd: %v", err)
	}

	// Read base tray icon (18×18 PNG).
	baseFile, err := os.Open(filepath.Join(repoRoot, "assets", "tray_icon.png"))
	if err != nil {
		log.Fatalf("open tray_icon.png: %v", err)
	}
	baseImg, err := png.Decode(baseFile)
	baseFile.Close()
	if err != nil {
		log.Fatalf("decode tray_icon.png: %v", err)
	}

	// TokyoNight accent color #7aa2f7 (RGB 122, 162, 247, fully opaque).
	accentColor := color.RGBA{R: 122, G: 162, B: 247, A: 255}

	// Quartile definitions: suffix → bar fill width in pixels (out of 18).
	quartiles := []struct {
		suffix string
		width  int
	}{
		{"25", 5},  // round(18 * 0.25) = 4.5 → 5
		{"50", 9},  // round(18 * 0.50) = 9
		{"75", 13}, // round(18 * 0.75) = 13.5 → 13 (floor to avoid clipping)
		{"100", 18}, // full width
	}

	for _, q := range quartiles {
		// Convert base to RGBA for pixel manipulation.
		rgba := image.NewRGBA(baseImg.Bounds())
		draw.Draw(rgba, rgba.Bounds(), baseImg, image.Point{}, draw.Src)

		// Overlay 2px accent bar at the bottom: rows [16, 17], columns [0, q.width).
		bounds := rgba.Bounds()
		imgH := bounds.Max.Y
		barH := 2 // bar height in pixels
		for y := imgH - barH; y < imgH; y++ {
			for x := 0; x < q.width; x++ {
				rgba.Set(x, y, accentColor)
			}
		}

		// Write to assets/tray_icon_progress_<suffix>.png
		outPath := filepath.Join(repoRoot, "assets", "tray_icon_progress_"+q.suffix+".png")
		outFile, err := os.Create(outPath)
		if err != nil {
			log.Fatalf("create %s: %v", outPath, err)
		}
		if err := png.Encode(outFile, rgba); err != nil {
			outFile.Close()
			log.Fatalf("encode %s: %v", outPath, err)
		}
		outFile.Close()
		log.Printf("wrote %s", outPath)
	}
	log.Println("Done.")
}
