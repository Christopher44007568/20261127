package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type Canvas struct {
	W, H       int
	World      Rect
	Pix        []Vec3
	BeamSum    []Vec3
	BeamWeight []float64
}

func NewCanvas(w, h int, world Rect) *Canvas {
	n := w * h
	return &Canvas{
		W: w, H: h, World: world,
		Pix:        make([]Vec3, n),
		BeamSum:    make([]Vec3, n),
		BeamWeight: make([]float64, n),
	}
}

func (c *Canvas) index(x, y int) int { return y*c.W + x }

func (c *Canvas) Set(x, y int, col Vec3) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Pix[c.index(x, y)] = col
}

func (c *Canvas) Add(x, y int, col Vec3) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := c.index(x, y)
	c.Pix[i] = c.Pix[i].Add(col)
}

func (c *Canvas) Blend(x, y int, col Vec3, alpha float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || alpha <= 0 {
		return
	}
	i := c.index(x, y)
	a := clamp01(alpha)
	c.Pix[i] = c.Pix[i].Mul(1 - a).Add(col.Mul(a))
}

func (c *Canvas) WorldToPixel(p Vec2) (float64, float64) {
	x := (p.X - c.World.Min.X) / (c.World.Max.X - c.World.Min.X) * float64(c.W)
	y := (c.World.Max.Y - p.Y) / (c.World.Max.Y - c.World.Min.Y) * float64(c.H)
	return x, y
}

func (c *Canvas) PixelToWorld(x, y float64) Vec2 {
	wx := c.World.Min.X + x/float64(c.W)*(c.World.Max.X-c.World.Min.X)
	wy := c.World.Max.Y - y/float64(c.H)*(c.World.Max.Y-c.World.Min.Y)
	return v2(wx, wy)
}

func distToSegment(px, py float64, a, b Vec2) float64 {
	abx, aby := b.X-a.X, b.Y-a.Y
	apx, apy := px-a.X, py-a.Y
	l2 := abx*abx + aby*aby
	t := 0.0
	if l2 > 1e-12 {
		t = clamp01((apx*abx + apy*aby) / l2)
	}
	cx, cy := a.X+abx*t, a.Y+aby*t
	return math.Hypot(px-cx, py-cy)
}

type LineStyle struct {
	Color Vec3
	Width float64
	Alpha float64
	Glow  bool
}

func (c *Canvas) LineWorld(a, b Vec2, style LineStyle) {
	ax, ay := c.WorldToPixel(a)
	bx, by := c.WorldToPixel(b)
	c.LinePixels(Vec2{ax, ay}, Vec2{bx, by}, style)
}

func (c *Canvas) LinePixels(a, b Vec2, style LineStyle) {
	radius := style.Width / 2
	pad := int(math.Ceil(radius + 2))
	x0 := int(math.Floor(math.Min(a.X, b.X))) - pad
	x1 := int(math.Ceil(math.Max(a.X, b.X))) + pad
	y0 := int(math.Floor(math.Min(a.Y, b.Y))) - pad
	y1 := int(math.Ceil(math.Max(a.Y, b.Y))) + pad
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x < 0 || y < 0 || x >= c.W || y >= c.H {
				continue
			}
			d := distToSegment(float64(x)+0.5, float64(y)+0.5, a, b)
			coverage := clamp(radius+0.65-d, 0, 1)
			if coverage <= 0 {
				continue
			}
			alpha := coverage * style.Alpha
			if style.Glow {
				c.Add(x, y, style.Color.Mul(alpha))
			} else {
				c.Blend(x, y, style.Color, alpha)
			}
		}
	}
}

func (c *Canvas) CircleWorld(center Vec2, radiusWorld float64, col Vec3, alpha float64, additive bool) {
	cx, cy := c.WorldToPixel(center)
	scale := float64(c.W) / (c.World.Max.X - c.World.Min.X)
	r := radiusWorld * scale
	c.CirclePixels(Vec2{cx, cy}, r, col, alpha, additive)
}

func (c *Canvas) CirclePixels(center Vec2, radius float64, col Vec3, alpha float64, additive bool) {
	pad := int(math.Ceil(radius + 2))
	cx, cy := int(math.Round(center.X)), int(math.Round(center.Y))
	for y := cy - pad; y <= cy+pad; y++ {
		for x := cx - pad; x <= cx+pad; x++ {
			if x < 0 || y < 0 || x >= c.W || y >= c.H {
				continue
			}
			d := math.Hypot(float64(x)+0.5-center.X, float64(y)+0.5-center.Y)
			coverage := clamp(radius+0.65-d, 0, 1)
			a := coverage * alpha
			if additive {
				c.Add(x, y, col.Mul(a))
			} else {
				c.Blend(x, y, col, a)
			}
		}
	}
}

func (c *Canvas) SoftDotWorld(center Vec2, radiusWorld float64, col Vec3, intensity float64) {
	cx, cy := c.WorldToPixel(center)
	scale := float64(c.W) / (c.World.Max.X - c.World.Min.X)
	r := radiusWorld * scale
	pad := int(math.Ceil(r + 1))
	ipx, ipy := int(math.Round(cx)), int(math.Round(cy))
	for y := ipy - pad; y <= ipy+pad; y++ {
		for x := ipx - pad; x <= ipx+pad; x++ {
			if x < 0 || y < 0 || x >= c.W || y >= c.H {
				continue
			}
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) / r
			if d >= 1 {
				continue
			}
			f := math.Pow(1-d*d, 2)
			c.Add(x, y, col.Mul(f*intensity))
		}
	}
}

// BeamLineWorld accumulates coloured coverage. Overlapping wavelengths are
// averaged instead of letting the last drawing operation overwrite the rest.
func (c *Canvas) BeamLineWorld(a, b Vec2, width float64, col Vec3, strength float64) {
	ax, ay := c.WorldToPixel(a)
	bx, by := c.WorldToPixel(b)
	radius := width / 2
	pad := int(math.Ceil(radius + 2))
	x0 := int(math.Floor(math.Min(ax, bx))) - pad
	x1 := int(math.Ceil(math.Max(ax, bx))) + pad
	y0 := int(math.Floor(math.Min(ay, by))) - pad
	y1 := int(math.Ceil(math.Max(ay, by))) + pad
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x < 0 || y < 0 || x >= c.W || y >= c.H {
				continue
			}
			d := distToSegment(float64(x)+0.5, float64(y)+0.5, Vec2{ax, ay}, Vec2{bx, by})
			coverage := clamp(radius+0.65-d, 0, 1)
			w := coverage * strength
			if w <= 0 {
				continue
			}
			i := c.index(x, y)
			c.BeamSum[i] = c.BeamSum[i].Add(col.Mul(w))
			c.BeamWeight[i] += w
		}
	}
}

func (c *Canvas) ResolveBeams() {
	for i, w := range c.BeamWeight {
		if w <= 1e-6 {
			continue
		}
		avg := c.BeamSum[i].Mul(1 / w)
		alpha := math.Min(0.97, w*0.96)
		c.Pix[i] = c.Pix[i].Mul(1 - alpha).Add(avg.Mul(alpha))
	}
}

func (c *Canvas) SavePNG(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			col := c.Pix[c.index(x, y)]
			r := linearToSRGB(tonemap(col.X))
			g := linearToSRGB(tonemap(col.Y))
			b := linearToSRGB(tonemap(col.Z))
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round(r * 255)),
				G: uint8(math.Round(g * 255)),
				B: uint8(math.Round(b * 255)),
				A: 255,
			})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
