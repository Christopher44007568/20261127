package main

import (
	"image"
	"testing"
)

func TestPadFloat(t *testing.T) {
	cases := map[float64]string{
		0:      "T=00.000",
		0.125:  "T=00.125",
		3.375:  "T=03.375",
		11.875: "T=11.875",
		0.0006: "T=00.001",
	}
	for in, want := range cases {
		if got := padFloat(in); got != want {
			t.Errorf("padFloat(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestDrawTextRendersLitPixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 10))
	DrawText(img, 2, 1, 1, "T=01", colText)
	b := img.Bounds()
	empty := true
	for y := b.Min.Y; y < b.Max.Y && empty; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				empty = false
				break
			}
		}
	}
	if empty {
		t.Fatal("DrawText produced no lit pixels")
	}
}

func TestDigitZeroGlyphShape(t *testing.T) {
	// '0' must have lit top/bottom bars and an empty center pixel in the
	// actual rendered output, regardless of the internal bit convention.
	g := glyph5x7['0']
	if g[0] != 0x0E || g[6] != 0x0E {
		t.Fatalf("top/bottom rows of '0' = %#x %#x, want 0x0E", g[0], g[6])
	}
	img := image.NewRGBA(image.Rect(0, 0, 8, 9))
	DrawText(img, 0, 0, 1, "0", colText)
	if _, _, _, a := img.At(2, 3).RGBA(); a != 0 {
		t.Fatal("center pixel of '0' must be blank")
	}
	if _, _, _, a := img.At(0, 3).RGBA(); a == 0 {
		t.Fatal("side pixel of '0' must be lit")
	}
}

func TestDigitOneIsNotZero(t *testing.T) {
	if glyph5x7['1'] == glyph5x7['0'] {
		t.Fatal("glyphs for '1' and '0' must differ")
	}
}
