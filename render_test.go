package main

import (
	"image"
	"testing"
)

func TestRenderWrapHasNoTransparentOrBlackSeam(t *testing.T) {
	texture := MakeTexture(96)
	mapping := Translate(13, -9).
		Mul(Rotate(degrees(-17))).
		Mul(Scale(1, 1))
	img := RenderView(
		texture, mapping,
		View{MinX: -128, MinY: -128, Width: 256, Height: 256},
		256, 256, 2,
	)
	assertOpaqueAndNotBlack(t, img)
}

func TestEdgeZoomHasNoTransparentOrBlackLine(t *testing.T) {
	texture := MakeTexture(96)
	mapping := Translate(13, -9).
		Mul(Rotate(degrees(-17))).
		Mul(Scale(1, 1))
	seamX, seamY := mapping.Apply(0, 0)
	img := RenderView(
		texture, mapping,
		View{MinX: seamX - 36, MinY: seamY - 36, Width: 72, Height: 72},
		288, 288, 2,
	)
	assertOpaqueAndNotBlack(t, img)
}

func TestMirroredRenderMovesDirectionMarker(t *testing.T) {
	const size = 128
	texture := MakeTexture(size)
	normal := RenderView(
		texture, Identity(),
		View{MinX: 0, MinY: 0, Width: size, Height: size},
		size, size, 2,
	)
	mirrored := RenderView(
		texture, Translate(size, 0).Mul(MirrorX()),
		View{MinX: 0, MinY: 0, Width: size, Height: size},
		size, size, 2,
	)

	normalX := coralCentroidX(t, normal)
	mirroredX := coralCentroidX(t, mirrored)
	if normalX <= size/2 {
		t.Fatalf("unmirrored direction marker is not on the right: x=%v", normalX)
	}
	if mirroredX >= size/2 {
		t.Fatalf("mirrored direction marker did not cross the mirror axis: x=%v", mirroredX)
	}
	if normalX-mirroredX < float64(size)/25 {
		t.Fatalf("marker moved too little when mirrored: %v -> %v", normalX, mirroredX)
	}
}

func assertOpaqueAndNotBlack(t *testing.T, img *image.NRGBA) {
	t.Helper()
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.A != 255 {
				t.Fatalf("transparent pixel at (%d,%d): %#v", x, y, c)
			}
			if int(c.R)+int(c.G)+int(c.B) < 40 {
				t.Fatalf("near-black seam pixel at (%d,%d): %#v", x, y, c)
			}
		}
	}
}

func coralCentroidX(t *testing.T, img *image.NRGBA) float64 {
	t.Helper()
	var sumX, count float64
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.R > 170 && c.G < 140 && c.B < 120 {
				sumX += float64(x)
				count++
			}
		}
	}
	if count == 0 {
		t.Fatal("direction marker color not found")
	}
	return sumX / count
}
