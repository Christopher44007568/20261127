package main

import (
	"image"
	"image/png"
	"math"
	"os"
)

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Sync()
}

func errorHeatmap(scene *Scene, img, ref *image.RGBA, fps *FootprintCache, scale float64) *image.RGBA {
	out := backgroundImage(scene)
	for _, p := range fps.Pixels {
		d := getRGB(img, p.X, p.Y).Luma() - getRGB(ref, p.X, p.Y).Luma()
		t := clamp(math.Abs(d)*scale, 0, 1)
		t = math.Pow(t, 0.58)
		setRGB(out, p.X, p.Y, heatPalette(t))
	}
	return out
}

func heatPalette(t float64) RGB {
	type stop struct {
		t float64
		c RGB
	}
	stops := [...]stop{
		{0.00, RGB{0.00, 0.00, 0.055}},
		{0.12, RGB{0.01, 0.02, 0.22}},
		{0.32, RGB{0.00, 0.20, 0.78}},
		{0.53, RGB{0.00, 0.76, 0.94}},
		{0.72, RGB{0.42, 0.98, 0.52}},
		{0.88, RGB{1.00, 0.94, 0.16}},
		{1.00, RGB{1.00, 0.48, 0.08}},
	}
	t = clamp(t, 0, 1)
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].t {
			f := (t - stops[i-1].t) / (stops[i].t - stops[i-1].t)
			return stops[i-1].c.Mul(1 - f).Add(stops[i].c.Mul(f))
		}
	}
	return stops[len(stops)-1].c
}

func cropImage(src *image.RGBA, r image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			sx := r.Min.X + x
			sy := r.Min.Y + y
			if image.Pt(sx, sy).In(src.Rect) {
				dst.SetRGBA(x, y, src.RGBAAt(sx, sy))
			}
		}
	}
	return dst
}

func resizeBilinear(src *image.RGBA, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	sw := src.Rect.Dx()
	sh := src.Rect.Dy()
	for y := 0; y < height; y++ {
		fy := (float64(y)+0.5)*float64(sh)/float64(height) - 0.5
		y0 := int(math.Floor(fy))
		ty := fy - float64(y0)
		y0 = int(clamp(float64(y0), 0, float64(sh-1)))
		y1 := int(clamp(float64(y0+1), 0, float64(sh-1)))
		for x := 0; x < width; x++ {
			fx := (float64(x)+0.5)*float64(sw)/float64(width) - 0.5
			x0 := int(math.Floor(fx))
			tx := fx - float64(x0)
			x0 = int(clamp(float64(x0), 0, float64(sw-1)))
			x1 := int(clamp(float64(x0+1), 0, float64(sw-1)))
			c00 := getRGB(src, x0, y0)
			c10 := getRGB(src, x1, y0)
			c01 := getRGB(src, x0, y1)
			c11 := getRGB(src, x1, y1)
			top := c00.Mul(1 - tx).Add(c10.Mul(tx))
			bottom := c01.Mul(1 - tx).Add(c11.Mul(tx))
			setRGB(dst, x, y, top.Mul(1-ty).Add(bottom.Mul(ty)))
		}
	}
	return dst
}

func resizeArea(src *image.RGBA, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	sw := src.Rect.Dx()
	sh := src.Rect.Dy()
	for y := 0; y < height; y++ {
		sy0 := y * sh / height
		sy1 := (y + 1) * sh / height
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < width; x++ {
			sx0 := x * sw / width
			sx1 := (x + 1) * sw / width
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var sum RGB
			var count int
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					sum = sum.Add(getRGB(src, sx, sy))
					count++
				}
			}
			setRGB(dst, x, y, sum.Mul(1/float64(count)))
		}
	}
	return dst
}

func resizeImage(src *image.RGBA, width, height int) *image.RGBA {
	if width < src.Rect.Dx() || height < src.Rect.Dy() {
		return resizeArea(src, width, height)
	}
	return resizeBilinear(src, width, height)
}
