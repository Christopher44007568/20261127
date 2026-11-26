package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"runtime"
	"sync"
)

type View struct {
	MinX, MinY float64
	Width      float64
	Height     float64
}

// RenderView maps every destination sample back into source pixel space with
// the inverse affine matrix, then wraps and bilinearly filters it.
func RenderView(src *image.NRGBA, sourceToScene Affine, view View, width, height, samples int) *image.NRGBA {
	inverse, ok := sourceToScene.Inverse()
	if !ok {
		panic("textureaffine: singular render transform")
	}
	if samples < 1 {
		samples = 1
	}

	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	jobs := make(chan int, height)
	var wg sync.WaitGroup
	workers := runtime.GOMAXPROCS(0)
	if workers > height {
		workers = height
	}
	if workers < 1 {
		workers = 1
	}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for py := range jobs {
				renderRow(src, inverse, view, out, width, height, samples, py)
			}
		}()
	}
	for py := 0; py < height; py++ {
		jobs <- py
	}
	close(jobs)
	wg.Wait()
	return out
}

func renderRow(src *image.NRGBA, inverse Affine, view View, out *image.NRGBA, width, height, samples, py int) {
	for px := 0; px < width; px++ {
		var sumR, sumG, sumB float64
		for sy := 0; sy < samples; sy++ {
			syf := (float64(py) + (float64(sy)+0.5)/float64(samples)) / float64(height)
			wy := view.MinY + syf*view.Height
			for sx := 0; sx < samples; sx++ {
				sxf := (float64(px) + (float64(sx)+0.5)/float64(samples)) / float64(width)
				wx := view.MinX + sxf*view.Width
				tx, ty := inverse.Apply(wx, wy)
				c := SampleWrap(src, tx, ty)
				sumR += float64(c.R)
				sumG += float64(c.G)
				sumB += float64(c.B)
			}
		}
		n := float64(samples * samples)
		out.SetNRGBA(px, py, color.NRGBA{
			R: clamp8(sumR / n),
			G: clamp8(sumG / n),
			B: clamp8(sumB / n),
			A: 255,
		})
	}
}

func WritePNG(path string, img image.Image) error {
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

func degrees(v float64) float64 {
	return v * math.Pi / 180
}
