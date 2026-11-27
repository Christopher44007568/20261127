package pipeline

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"materiallod/internal/chart"
	"materiallod/material"
)

func savePNG(dir, name string, img image.Image) error {
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return nil
}

func renderImages(dir string, art *Artifacts) error {
	renderers := []func(string, *Artifacts) error{
		renderSpheresPNG,
		renderViewSweepPNG,
		renderBrightnessDiffPNG,
		renderBoundaryPNG,
	}
	for _, render := range renderers {
		if err := render(dir, art); err != nil {
			return err
		}
	}
	return nil
}

func fillEllipse(cv *chart.Canvas, cx, cy, rx, ry int, col color.Color) {
	for y := -ry; y <= ry; y++ {
		for x := -rx; x <= rx; x++ {
			v := float64(x*x)/float64(rx*rx) + float64(y*y)/float64(ry*ry)
			if v <= 1 {
				cv.Img.Set(cx+x, cy+y, col)
			}
		}
	}
}

func swatchColor(c material.Color) color.RGBA {
	s := c.ToSRGB()
	return color.RGBA{
		R: byte(math.Round(s.R * 255)),
		G: byte(math.Round(s.G * 255)),
		B: byte(math.Round(s.B * 255)),
		A: 255,
	}
}
