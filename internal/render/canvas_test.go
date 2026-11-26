package render

import (
	"bytes"
	"image/png"
	"testing"
)

func TestCanvasSetAtAndBounds(t *testing.T) {
	c := NewCanvas(12, 8, RGB{10, 20, 30})
	if got := c.At(0, 0); got != (RGB{10, 20, 30}) {
		t.Fatalf("fill mismatch: %v", got)
	}
	// Out-of-bounds access must be safe.
	c.Set(-1, -1, RGB{1, 2, 3})
	c.Set(12, 8, RGB{1, 2, 3})
	c.Set(5, 4, RGB{200, 100, 50})
	if got := c.At(5, 4); got != (RGB{200, 100, 50}) {
		t.Fatalf("set/at mismatch: %v", got)
	}
}

func TestLineEndpoints(t *testing.T) {
	c := NewCanvas(20, 20, RGB{0, 0, 0})
	col := RGB{255, 255, 255}
	c.Line(2, 3, 17, 15, col)
	if c.At(2, 3) != col || c.At(17, 15) != col {
		t.Fatal("line endpoints not painted")
	}
}

func TestPNGEncoding(t *testing.T) {
	c := NewCanvas(16, 9, RGB{120, 60, 200})
	var buf bytes.Buffer
	if err := c.EncodePNG(&buf); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if img.Bounds().Dx() != 16 || img.Bounds().Dy() != 9 {
		t.Fatalf("decoded size = %v", img.Bounds())
	}
}

func TestFigureDimensions(t *testing.T) {
	ramp := DrawRamp(RampConfig{Width: 320, Height: 200, FilmN: 1.5,
		ThicknessMax: 1000})
	if ramp.W != 320 || ramp.H != 200 {
		t.Fatalf("ramp size = %dx%d", ramp.W, ramp.H)
	}
	spec := DrawSpectrum(SpectrumConfig{Width: 320, Height: 220, FilmN: 1.5,
		Thicknesses: []float64{0, 400, 800}})
	if spec.W != 320 || spec.H != 220 {
		t.Fatalf("spectrum size = %dx%d", spec.W, spec.H)
	}
	spheres := DrawSpheres(SphereConfig{Width: 360, Height: 240, FilmN: 1.5,
		FilmThickness: 500})
	if spheres.W != 360 || spheres.H != 240 {
		t.Fatalf("spheres size = %dx%d", spheres.W, spheres.H)
	}
}

func TestSpectralColorRange(t *testing.T) {
	for lambda := 380.0; lambda <= 780; lambda += 10 {
		col := spectralColor(lambda)
		for _, v := range []uint8{col.R, col.G, col.B} {
			if v > 255 {
				t.Fatalf("spectral color out of range at %.0f", lambda)
			}
		}
	}
}
