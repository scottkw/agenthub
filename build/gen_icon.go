//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

func main() {
	// Create a 256x256 RGBA image with a dark blue background
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))

	// Fill background with dark blue (#1a237e)
	bg := color.RGBA{R: 26, G: 35, B: 126, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	// Draw a simple "A" shape using rectangles (stylized)
	// Left diagonal of A
	accent := color.RGBA{R: 100, G: 181, B: 246, A: 255}
	drawRect(img, 68, 200, 90, 56, accent)   // left leg
	drawRect(img, 166, 200, 90, 56, accent)  // right leg
	drawRect(img, 90, 120, 76, 24, accent)   // crossbar
	drawRect(img, 90, 56, 76, 64, accent)    // top connector

	f, err := os.Create("appicon.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func drawRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			img.SetRGBA(x+dx, y+dy, c)
		}
	}
}
