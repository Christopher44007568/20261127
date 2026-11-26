package bloom

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func WriteComparisonPNG(path string, reference, actual *Image) error {
	if err := reference.Check(); err != nil {
		return err
	}
	if err := actual.Check(); err != nil {
		return err
	}
	if err := sameDimensions(reference, actual); err != nil {
		return err
	}
	gap := 4
	out := image.NewRGBA(image.Rect(0, 0, reference.Width*2+gap, reference.Height))
	for y := 0; y < reference.Height; y++ {
		for x := 0; x < reference.Width; x++ {
			out.SetRGBA(x, y, displayColor(reference.At(x, y)))
			out.SetRGBA(x+reference.Width+gap, y, displayColor(actual.At(x, y)))
		}
	}
	for y := 0; y < reference.Height; y++ {
		for x := reference.Width; x < reference.Width+gap; x++ {
			out.SetRGBA(x, y, color.RGBA{R: 255, G: 210, B: 76, A: 255})
		}
	}
	return writePNG(path, out)
}

func WriteQuantizationHeatmap(path string, errorImage *Image, normalization float64) error {
	if err := errorImage.Check(); err != nil {
		return err
	}
	if normalization <= 0 || math.IsNaN(normalization) || math.IsInf(normalization, 0) {
		normalization = 1
	}
	const floor = 1e-9
	logNorm := math.Log1p(normalization / floor)
	height := errorImage.Height + 12
	out := image.NewRGBA(image.Rect(0, 0, errorImage.Width, height))
	for y := 0; y < errorImage.Height; y++ {
		for x := 0; x < errorImage.Width; x++ {
			e := errorImage.At(x, y).MaxComponent()
			t := math.Log1p(e/floor) / logNorm
			c := HeatColor(t)
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round(c.R * 255)),
				G: uint8(math.Round(c.G * 255)),
				B: uint8(math.Round(c.B * 255)),
				A: 255,
			})
		}
	}
	for x := 0; x < errorImage.Width; x++ {
		t := float64(x) / float64(maxInt(1, errorImage.Width-1))
		c := HeatColor(t)
		for y := errorImage.Height; y < height; y++ {
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round(c.R * 255)),
				G: uint8(math.Round(c.G * 255)),
				B: uint8(math.Round(c.B * 255)),
				A: 255,
			})
		}
	}
	return writePNG(path, out)
}

func displayColor(c RGB) color.RGBA {
	mapped := ToneMapReinhard(c)
	toByte := func(v float64) uint8 {
		v = math.Max(0, math.Min(1, v))
		encoded := linearToSRGB(v)
		return uint8(math.Round(encoded * 255))
	}
	return color.RGBA{R: toByte(mapped.R), G: toByte(mapped.G), B: toByte(mapped.B), A: 255}
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return fmt.Errorf("encode PNG: %w", err)
	}
	return f.Close()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
