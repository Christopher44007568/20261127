package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := encoder.Encode(file, img); err != nil {
		return err
	}
	return file.Close()
}

func CheckerPreview(source image.Image, cell int) *image.NRGBA {
	bounds := source.Bounds()
	preview := image.NewNRGBA(bounds)
	light := color.NRGBA{R: 248, G: 250, B: 252, A: 255}
	dark := color.NRGBA{R: 203, G: 213, B: 225, A: 255}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			checker := ((x-bounds.Min.X)/cell + (y-bounds.Min.Y)/cell) % 2
			i := preview.PixOffset(x, y)
			c := light
			if checker == 1 {
				c = dark
			}
			preview.Pix[i+0] = c.R
			preview.Pix[i+1] = c.G
			preview.Pix[i+2] = c.B
			preview.Pix[i+3] = 255
		}
	}

	draw.Draw(preview, bounds, source, bounds.Min, draw.Over)
	return preview
}
