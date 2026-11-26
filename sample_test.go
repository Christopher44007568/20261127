package main

import (
	"image"
	"image/color"
	"testing"
)

func TestWrapSampleIsPeriodicInBothAxes(t *testing.T) {
	img := testTexture(16)
	for y := 0; y < 16; y++ {
		for _, x := range []float64{-9.75, 0, 0.25, 7.5, 15.999} {
			a := SampleWrap(img, x, float64(y))
			b := SampleWrap(img, x+16, float64(y))
			if a != b {
				t.Fatalf("x periodicity failed at (%v,%d): %#v != %#v", x, y, a, b)
			}
			c := SampleWrap(img, x, float64(y+16))
			if a != c {
				t.Fatalf("y periodicity failed at (%v,%d): %#v != %#v", x, y, a, c)
			}
		}
	}
}

func TestWrapSampleNeverFallsToTransparentBlack(t *testing.T) {
	img := testTexture(12)
	for _, p := range []point{
		{-1000, -1000}, {-0.5, 0}, {-1, 60}, {12, -12},
		{12000, -300}, {0.25, 0.25}, {11.99, 11.99},
	} {
		c := SampleWrap(img, p.x, p.y)
		if c.A != 255 {
			t.Fatalf("sample %v has alpha %d; wrap leaked outside texture", p, c.A)
		}
		if c.R == 0 && c.G == 0 && c.B == 0 {
			t.Fatalf("sample %v became black", p)
		}
	}
}

func TestBilinearTapsCrossTheWrapEdge(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{B: 255, A: 255})
	img.SetNRGBA(3, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	// At x=0 the left tap must wrap to pixel 3 and blend it with pixel 0.
	c := SampleWrap(img, 0, 0)
	if c.G == 0 || c.B == 0 {
		t.Fatalf("left edge did not blend wrapped tap: %#v", c)
	}
}

func testTexture(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(20 + x*7),
				G: uint8(30 + y*5),
				B: uint8(40 + (x+y)%11),
				A: 255,
			})
		}
	}
	return img
}
