package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// Transform maps font units (Y up) to raster pixels (Y down).
type Transform struct {
	Scale float64
	X0    float64
	Y0    float64
	B     Bounds
}

func FitTransform(b Bounds, width, height int, padding float64) Transform {
	if b.Empty() {
		b = Bounds{Min: Vec2{}, Max: Vec2{X: 1, Y: 1}}
	}
	bw := math.Max(b.Width(), 1e-9)
	bh := math.Max(b.Height(), 1e-9)
	scale := math.Min((float64(width)-2*padding)/bw, (float64(height)-2*padding)/bh)
	drawW := bw * scale
	drawH := bh * scale
	return Transform{
		Scale: scale,
		X0:    (float64(width) - drawW) * 0.5,
		Y0:    (float64(height) - drawH) * 0.5,
		B:     b,
	}
}

func (t Transform) Map(p Vec2) Vec2 {
	return Vec2{
		X: t.X0 + (p.X-t.B.Min.X)*t.Scale,
		Y: t.Y0 + (t.B.Max.Y-p.Y)*t.Scale,
	}
}

func (t Transform) Unmap(p Vec2) Vec2 {
	return Vec2{
		X: t.B.Min.X + (p.X-t.X0)/t.Scale,
		Y: t.B.Max.Y - (p.Y-t.Y0)/t.Scale,
	}
}

type Canvas struct {
	Image  *image.NRGBA
	Width  int
	Height int
}

func NewCanvas(width, height int, background color.NRGBA) *Canvas {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, background)
		}
	}
	return &Canvas{Image: img, Width: width, Height: height}
}

func clamp01(x float64) float64 { return math.Max(0, math.Min(1, x)) }

func blendPixel(dst color.NRGBA, src color.NRGBA, coverage float64) color.NRGBA {
	a := clamp01(coverage) * float64(src.A) / 255.0
	out := dst
	out.R = uint8(clamp01((float64(dst.R)*(1-a)+float64(src.R)*a)/255.0) * 255)
	out.G = uint8(clamp01((float64(dst.G)*(1-a)+float64(src.G)*a)/255.0) * 255)
	out.B = uint8(clamp01((float64(dst.B)*(1-a)+float64(src.B)*a)/255.0) * 255)
	out.A = 255
	return out
}

// FillOutline uses rotated-grid-free NxN supersampling and even-odd containment.
func (cv *Canvas) FillOutline(o Outline, tr Transform, fill color.NRGBA, samples int) {
	if samples < 1 {
		samples = 4
	}
	for y := 0; y < cv.Height; y++ {
		for x := 0; x < cv.Width; x++ {
			inside := 0
			total := samples * samples
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					px := float64(x) + (float64(sx)+0.5)/float64(samples)
					py := float64(y) + (float64(sy)+0.5)/float64(samples)
					if o.Contains(tr.Unmap(Vec2{X: px, Y: py})) {
						inside++
					}
				}
			}
			if inside == 0 {
				continue
			}
			dst := cv.Image.NRGBAAt(x, y)
			cv.Image.SetNRGBA(x, y, blendPixel(dst, fill, float64(inside)/float64(total)))
		}
	}
}

func (cv *Canvas) strokePixelSegment(a, b Vec2, col color.NRGBA, width float64) {
	radius := width * 0.5
	minX := int(math.Floor(math.Min(a.X, b.X) - radius - 1))
	maxX := int(math.Ceil(math.Max(a.X, b.X) + radius + 1))
	minY := int(math.Floor(math.Min(a.Y, b.Y) - radius - 1))
	maxY := int(math.Ceil(math.Max(a.Y, b.Y) + radius + 1))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX > cv.Width-1 {
		maxX = cv.Width - 1
	}
	if maxY > cv.Height-1 {
		maxY = cv.Height - 1
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := Vec2{X: float64(x) + 0.5, Y: float64(y) + 0.5}
			d, _ := PointSegmentDistance(p, a, b)
			coverage := clamp01(radius + 0.6 - d)
			if coverage <= 0 {
				continue
			}
			dst := cv.Image.NRGBAAt(x, y)
			cv.Image.SetNRGBA(x, y, blendPixel(dst, col, coverage))
		}
	}
}

func (cv *Canvas) StrokeContour(c Contour, tr Transform, col color.NRGBA, width float64) {
	if len(c) < 2 || width <= 0 {
		return
	}
	pixels := make([]Vec2, len(c))
	for i, p := range c {
		pixels[i] = tr.Map(p)
	}
	for i := range pixels {
		cv.strokePixelSegment(pixels[i], pixels[(i+1)%len(pixels)], col, width)
	}
}

func (cv *Canvas) StrokeOutline(o Outline, tr Transform, col color.NRGBA, width float64) {
	cv.StrokeContour(o.Outer, tr, col, width)
	for _, h := range o.Holes {
		cv.StrokeContour(h, tr, col, width)
	}
}

func (cv *Canvas) FillCircle(center Vec2, radius float64, col color.NRGBA) {
	minX := int(math.Floor(center.X - radius - 1))
	maxX := int(math.Ceil(center.X + radius + 1))
	minY := int(math.Floor(center.Y - radius - 1))
	maxY := int(math.Ceil(center.Y + radius + 1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= cv.Width || y >= cv.Height {
				continue
			}
			p := Vec2{X: float64(x) + 0.5, Y: float64(y) + 0.5}
			d := p.Sub(center).Len()
			coverage := clamp01(radius + 0.6 - d)
			if coverage <= 0 {
				continue
			}
			dst := cv.Image.NRGBAAt(x, y)
			cv.Image.SetNRGBA(x, y, blendPixel(dst, col, coverage))
		}
	}
}

// DrawMarker draws a high-contrast target dot at a design coordinate.
func (cv *Canvas) DrawMarker(p Vec2, tr Transform) {
	center := tr.Map(p)
	cv.FillCircle(center, 12, color.NRGBA{R: 185, G: 28, B: 28, A: 230})
	cv.FillCircle(center, 7, color.NRGBA{R: 253, G: 224, B: 71, A: 255})
	cv.FillCircle(center, 3, color.NRGBA{R: 127, G: 29, B: 29, A: 255})
}

func SavePNG(img image.Image, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	return encoder.Encode(file, img)
}
