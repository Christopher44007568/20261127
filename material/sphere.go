package material

import (
	"image"
	"image/color"
	"math"
)

const (
	SphereSize   = 180
	sphereRadius = 76.35
)

// SphereBuffer is an aligned linear/display buffer for one shaded sphere.
type SphereBuffer struct {
	MaterialID string
	Tier       string
	Width      int
	Height     int
	CenterX    float64
	CenterY    float64
	Radius     float64
	Mask       []bool
	Linear     []Color
	Display    []Color
	Luma       []float64
}

func newSphereBuffer(id, tier string) *SphereBuffer {
	n := SphereSize * SphereSize
	return &SphereBuffer{
		MaterialID: id,
		Tier:       tier,
		Width:      SphereSize,
		Height:     SphereSize,
		CenterX:    SphereSize / 2.0,
		CenterY:    SphereSize/2.0 + 2,
		Radius:     sphereRadius,
		Mask:       make([]bool, n),
		Linear:     make([]Color, n),
		Display:    make([]Color, n),
		Luma:       make([]float64, n),
	}
}

// RenderSphere renders one named tier to the shared aligned buffer.
func RenderSphere(m Material, tier Tier) *SphereBuffer {
	buf := newSphereBuffer(m.ID, tier.String())
	view := Vec3{Z: 1}
	for y := 0; y < buf.Height; y++ {
		for x := 0; x < buf.Width; x++ {
			i := y*buf.Width + x
			nx, ny, nz, ok := buf.normalAt(x, y)
			if !ok {
				continue
			}
			n := Vec3{X: nx, Y: ny, Z: nz}
			linear := Shade(m, tier, n, view)
			buf.set(i, linear)
		}
	}
	return buf
}

// RenderSphereLOD renders a continuous LOD blend to the same aligned buffer.
func RenderSphereLOD(m Material, lod, crossfade float64) *SphereBuffer {
	buf := newSphereBuffer(m.ID, "lod")
	for y := 0; y < buf.Height; y++ {
		for x := 0; x < buf.Width; x++ {
			i := y*buf.Width + x
			nx, ny, nz, ok := buf.normalAt(x, y)
			if !ok {
				continue
			}
			linear, _ := ShadeLOD(m, lod, Vec3{X: nx, Y: ny, Z: nz}, Vec3{Z: 1}, crossfade)
			buf.set(i, linear)
		}
	}
	return buf
}

func (b *SphereBuffer) normalAt(x, y int) (float64, float64, float64, bool) {
	dx := float64(x) + 0.5 - b.CenterX
	dyImage := float64(y) + 0.5 - b.CenterY
	d2 := dx*dx + dyImage*dyImage
	r2 := b.Radius * b.Radius
	if d2 > r2 {
		return 0, 0, 0, false
	}
	nx := dx / b.Radius
	ny := -dyImage / b.Radius
	nz := math.Sqrt(math.Max(0, 1-nx*nx-ny*ny))
	return nx, ny, nz, true
}

func (b *SphereBuffer) set(i int, linear Color) {
	display := Display(linear)
	b.Mask[i] = true
	b.Linear[i] = linear
	b.Display[i] = display
	b.Luma[i] = display.Luminance()
}

// Pixels returns the number of covered sphere pixels.
func (b *SphereBuffer) Pixels() int {
	n := 0
	for _, ok := range b.Mask {
		if ok {
			n++
		}
	}
	return n
}

// Image converts the sphere to a transparent RGBA image.
func (b *SphereBuffer) Image() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, b.Width, b.Height))
	for i, ok := range b.Mask {
		if !ok {
			continue
		}
		c := b.Display[i]
		img.Pix[i*4+0] = byte(math.Round(clamp01(c.R) * 255))
		img.Pix[i*4+1] = byte(math.Round(clamp01(c.G) * 255))
		img.Pix[i*4+2] = byte(math.Round(clamp01(c.B) * 255))
		img.Pix[i*4+3] = 255
	}
	return img
}

// DiffImage renders an absolute luma difference using the reference heat map.
func DiffImage(a, b *SphereBuffer, scale float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, a.Width, a.Height))
	for i, ok := range a.Mask {
		if !ok || !b.Mask[i] {
			continue
		}
		d := math.Abs(a.Luma[i] - b.Luma[i])
		c := HeatColor(d / scale)
		img.SetRGBA(i%a.Width, i/a.Width, color.RGBA{
			R: byte(c.R * 255), G: byte(c.G * 255), B: byte(c.B * 255), A: 255,
		})
	}
	return img
}

// HeatColor maps 0..1 to navy, blue, purple, red, orange and pale yellow.
func HeatColor(t float64) Color {
	t = clamp01(t)
	stops := []Color{
		{R: 0.03, G: 0.05, B: 0.20},
		{R: 0.05, G: 0.16, B: 0.62},
		{R: 0.42, G: 0.08, B: 0.72},
		{R: 0.92, G: 0.16, B: 0.20},
		{R: 1.00, G: 0.62, B: 0.25},
		{R: 1.00, G: 0.93, B: 0.82},
	}
	p := t * float64(len(stops)-1)
	i := int(p)
	if i >= len(stops)-1 {
		return stops[len(stops)-1]
	}
	f := p - float64(i)
	return Color{
		R: mix(stops[i].R, stops[i+1].R, f),
		G: mix(stops[i].G, stops[i+1].G, f),
		B: mix(stops[i].B, stops[i+1].B, f),
	}
}
