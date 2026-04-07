//go:build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// Colors
var (
	white   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	navy    = color.RGBA{R: 26, G: 35, B: 126, A: 255}
	accent  = color.RGBA{R: 66, G: 133, B: 244, A: 255} // Google blue
	black   = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	transp = color.RGBA{R: 0, G: 0, B: 0, A: 0}
)

func main() {
	// Generate 1024x1024 app icon
	appIcon := generateAppIcon(1024)
	writeImage("appicon.png", appIcon)

	// Generate iconset sizes for macOS
	sizes := []int{16, 32, 64, 128, 256, 512, 1024}
	for _, sz := range sizes {
		icon := generateAppIcon(sz)
		writeImage(fmt.Sprintf("icon_%dx%d.png", sz, sz), icon)
	}

	// Generate tray icons (18x18 monochrome template icons for macOS)
	trayNormal := generateTrayIcon(18, false)
	writeImage("tray_icon.png", trayNormal)

	trayError := generateTrayIcon(18, true)
	writeImage("tray_icon_error.png", trayError)

	// Generate larger tray icon for reference/testing
	trayLarge := generateTrayIcon(72, false)
	writeImage("tray_icon_preview.png", trayLarge)

	fmt.Println("All icons generated.")
}

// generateAppIcon creates the main AgentHub app icon at the given size.
// Design: A bold geometric "A" with integrated "hub" node dots, on white background.
// The "A" is constructed from thick strokes with rounded appearance via anti-aliasing.
func generateAppIcon(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	s := float64(size)

	// Fill white background
	fillRect(img, 0, 0, size, size, white)

	// Add rounded rectangle navy background with margin
	margin := s * 0.08
	radius := s * 0.18
	fillRoundedRect(img, margin, margin, s-margin, s-margin, radius, navy)

	// Draw the "A" letterform in white on the navy rounded rect
	// The A is geometric: two diagonal legs, a crossbar, and a pointed top
	cx := s / 2 // center x
	strokeW := s * 0.105

	// A shape parameters
	topY := s * 0.18      // apex of the A
	baseY := s * 0.82     // bottom of the legs
	crossY := s * 0.56    // crossbar center Y
	crossH := strokeW * 0.8
	legSpreadBot := s * 0.30 // half-width at base
	legSpreadTop := s * 0.01 // half-width at top (nearly pointed)

	// Draw left leg of A (thick diagonal)
	drawThickLine(img, cx-legSpreadTop, topY, cx-legSpreadBot, baseY, strokeW, white)
	// Draw right leg of A
	drawThickLine(img, cx+legSpreadTop, topY, cx+legSpreadBot, baseY, strokeW, white)

	// Draw crossbar
	// Calculate crossbar endpoints at crossbar Y
	frac := (crossY - topY) / (baseY - topY)
	crossLeftX := cx - (legSpreadTop + frac*(legSpreadBot-legSpreadTop))
	crossRightX := cx + (legSpreadTop + frac*(legSpreadBot-legSpreadTop))
	fillRectF(img, crossLeftX-strokeW*0.3, crossY-crossH/2, crossRightX+strokeW*0.3, crossY+crossH/2, white)

	// Draw three small "hub" dots (connection nodes) in accent blue
	// Positioned: one on each leg bottom and one at the apex
	dotR := s * 0.038
	drawCircle(img, cx, topY+dotR*0.5, dotR, accent)                // apex dot
	drawCircle(img, cx-legSpreadBot*0.85, baseY-dotR*0.5, dotR, accent) // left dot
	drawCircle(img, cx+legSpreadBot*0.85, baseY-dotR*0.5, dotR, accent) // right dot

	return img
}

// generateTrayIcon creates an 18x18 monochrome template icon for macOS menu bar.
// Template icons should be black with alpha channel — macOS inverts automatically.
// Design: simplified "A" with hub dots, using alpha for anti-aliasing.
func generateTrayIcon(size int, isError bool) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	s := float64(size)

	// Fill transparent background
	fillRect(img, 0, 0, size, size, transp)

	fg := black
	if isError {
		// Error icon: use a circle with exclamation mark
		// Draw circle outline
		cx, cy := s/2, s/2
		outerR := s*0.45
		innerR := s*0.35
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx := float64(x) + 0.5 - cx
				dy := float64(y) + 0.5 - cy
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist <= outerR && dist >= innerR {
					// Anti-alias the edges
					outerAA := clamp01(outerR - dist + 0.5)
					innerAA := clamp01(dist - innerR + 0.5)
					a := uint8(255 * outerAA * innerAA)
					img.SetRGBA(x, y, color.RGBA{R: 0, G: 0, B: 0, A: a})
				}
			}
		}
		// Draw exclamation mark: vertical bar + dot
		bangW := s * 0.12
		bangTop := s * 0.22
		bangBot := s * 0.55
		bangDotY := s * 0.72
		bangDotR := s * 0.08
		fillRectAA(img, cx-bangW/2, bangTop, cx+bangW/2, bangBot, fg)
		drawCircle(img, cx, bangDotY, bangDotR, fg)
		return img
	}

	// Normal tray icon: simplified "A" with hub dots
	cx := s / 2
	strokeW := s * 0.14

	topY := s * 0.10
	baseY := s * 0.88
	legSpreadBot := s * 0.36
	legSpreadTop := s * 0.02

	// Draw A legs
	drawThickLine(img, cx-legSpreadTop, topY, cx-legSpreadBot, baseY, strokeW, fg)
	drawThickLine(img, cx+legSpreadTop, topY, cx+legSpreadBot, baseY, strokeW, fg)

	// Draw crossbar
	crossY := s * 0.58
	crossH := strokeW * 0.75
	frac := (crossY - topY) / (baseY - topY)
	crossLeftX := cx - (legSpreadTop + frac*(legSpreadBot-legSpreadTop))
	crossRightX := cx + (legSpreadTop + frac*(legSpreadBot-legSpreadTop))
	fillRectAA(img, crossLeftX-strokeW*0.2, crossY-crossH/2, crossRightX+strokeW*0.2, crossY+crossH/2, fg)

	// Hub dots
	dotR := s * 0.06
	drawCircle(img, cx, topY+dotR, dotR, fg)
	drawCircle(img, cx-legSpreadBot*0.8, baseY-dotR, dotR, fg)
	drawCircle(img, cx+legSpreadBot*0.8, baseY-dotR, dotR, fg)

	return img
}

// --- Drawing primitives ---

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			img.SetRGBA(x+dx, y+dy, c)
		}
	}
}

func fillRectF(img *image.RGBA, x1, y1, x2, y2 float64, c color.RGBA) {
	bounds := img.Bounds()
	ix1 := int(math.Floor(x1))
	iy1 := int(math.Floor(y1))
	ix2 := int(math.Ceil(x2))
	iy2 := int(math.Ceil(y2))
	for y := iy1; y < iy2; y++ {
		for x := ix1; x < ix2; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
				continue
			}
			// Anti-alias edges
			ax := clamp01(math.Min(float64(x)+1-x1, x2-float64(x)))
			ay := clamp01(math.Min(float64(y)+1-y1, y2-float64(y)))
			a := ax * ay
			blendPixel(img, x, y, c, a)
		}
	}
}

func fillRectAA(img *image.RGBA, x1, y1, x2, y2 float64, c color.RGBA) {
	fillRectF(img, x1, y1, x2, y2, c)
}

func fillRoundedRect(img *image.RGBA, x1, y1, x2, y2, radius float64, c color.RGBA) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			fx := float64(x) + 0.5
			fy := float64(y) + 0.5
			if fx < x1 || fx > x2 || fy < y1 || fy > y2 {
				continue
			}
			// Check if inside rounded corners
			dist := roundedRectDist(fx, fy, x1, y1, x2, y2, radius)
			if dist <= 0.5 {
				a := clamp01(0.5 - dist + 0.5)
				blendPixel(img, x, y, c, a)
			}
		}
	}
}

func roundedRectDist(px, py, x1, y1, x2, y2, r float64) float64 {
	// Signed distance from point to rounded rectangle (negative = inside)
	cx := math.Max(x1+r, math.Min(px, x2-r))
	cy := math.Max(y1+r, math.Min(py, y2-r))
	dx := px - cx
	dy := py - cy
	if px >= x1+r && px <= x2-r {
		return math.Max(y1-py, py-y2)
	}
	if py >= y1+r && py <= y2-r {
		return math.Max(x1-px, px-x2)
	}
	return math.Sqrt(dx*dx+dy*dy) - r
}

func drawThickLine(img *image.RGBA, x1, y1, x2, y2, width float64, c color.RGBA) {
	bounds := img.Bounds()
	halfW := width / 2

	// Bounding box
	minX := int(math.Floor(math.Min(x1, x2) - halfW - 1))
	maxX := int(math.Ceil(math.Max(x1, x2) + halfW + 1))
	minY := int(math.Floor(math.Min(y1, y2) - halfW - 1))
	maxY := int(math.Ceil(math.Max(y1, y2) + halfW + 1))

	// Line direction
	dx := x2 - x1
	dy := y2 - y1
	lineLen := math.Sqrt(dx*dx + dy*dy)
	if lineLen < 0.001 {
		return
	}

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
				continue
			}
			px := float64(x) + 0.5
			py := float64(y) + 0.5

			// Distance from point to line segment
			dist := pointToSegmentDist(px, py, x1, y1, x2, y2)

			if dist < halfW+0.5 {
				a := clamp01(halfW - dist + 0.5)
				blendPixel(img, x, y, c, a)
			}
		}
	}
}

func pointToSegmentDist(px, py, x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	lenSq := dx*dx + dy*dy
	if lenSq < 0.001 {
		return math.Sqrt((px-x1)*(px-x1) + (py-y1)*(py-y1))
	}
	t := ((px-x1)*dx + (py-y1)*dy) / lenSq
	t = math.Max(0, math.Min(1, t))
	closestX := x1 + t*dx
	closestY := y1 + t*dy
	ddx := px - closestX
	ddy := py - closestY
	return math.Sqrt(ddx*ddx + ddy*ddy)
}

func drawCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	bounds := img.Bounds()
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
				continue
			}
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < r+0.5 {
				a := clamp01(r - dist + 0.5)
				blendPixel(img, x, y, c, a)
			}
		}
	}
}

func blendPixel(img *image.RGBA, x, y int, c color.RGBA, alpha float64) {
	if alpha <= 0 {
		return
	}
	existing := img.RGBAAt(x, y)
	srcA := float64(c.A) / 255 * alpha
	dstA := float64(existing.A) / 255
	outA := srcA + dstA*(1-srcA)
	if outA <= 0 {
		return
	}
	outR := (float64(c.R)*srcA + float64(existing.R)*dstA*(1-srcA)) / outA
	outG := (float64(c.G)*srcA + float64(existing.G)*dstA*(1-srcA)) / outA
	outB := (float64(c.B)*srcA + float64(existing.B)*dstA*(1-srcA)) / outA
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(math.Round(outR)),
		G: uint8(math.Round(outG)),
		B: uint8(math.Round(outB)),
		A: uint8(math.Round(outA * 255)),
	})
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func writeImage(name string, img *image.RGBA) {
	f, err := os.Create(name)
	if err != nil {
		panic(fmt.Sprintf("create %s: %v", name, err))
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(fmt.Sprintf("encode %s: %v", name, err))
	}
	fmt.Printf("  wrote %s (%dx%d)\n", name, img.Bounds().Dx(), img.Bounds().Dy())
}

