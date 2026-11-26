package main

import (
	"image"
	"image/color"
	"math"
)

// RGB is an opaque 24-bit color.
type RGB struct {
	R, G, B uint8
}

func hexRGB(h uint32) RGB {
	return RGB{R: uint8(h >> 16), G: uint8(h >> 8), B: uint8(h)}
}

var (
	colInk       = hexRGB(0x111827)
	colPaper     = hexRGB(0xFFFFFF)
	colPanel     = hexRGB(0xF7F8FA)
	colGrid      = hexRGB(0xE3E7EC)
	colSlope     = hexRGB(0x6B4F2A)
	colGround    = hexRGB(0xE9DFCB)
	colGroundDot = hexRGB(0xC9B58C)
	colHipL      = hexRGB(0x2563EB)
	colHipR      = hexRGB(0x0D9488)
	colFar       = hexRGB(0x94A3B8)
	colTorso     = hexRGB(0x374151)
	colSkin      = hexRGB(0xE8B08A)
	colFoot      = hexRGB(0x1F2937)
	colTarget    = hexRGB(0x111827)
	colOK        = hexRGB(0x16A34A)
	colWarn      = hexRGB(0xDC2626)
	colAmber     = hexRGB(0xD97706)
	colBlue      = hexRGB(0x2563EB)
	colTeal      = hexRGB(0x0D9488)
	colPurple    = hexRGB(0x7C3AED)
	colGray      = hexRGB(0x6B7280)
	colLightGray = hexRGB(0xD1D5DB)
)

// Canvas is an RGBA image with a clipping rectangle and simple raster ops.
type Canvas struct {
	img  *image.RGBA
	clip image.Rectangle
}

func NewCanvas(w, h int, bg RGB) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := &Canvas{img: img, clip: img.Bounds()}
	c.clear(bg)
	return c
}

func (c *Canvas) clear(col RGB) {
	for y := c.clip.Min.Y; y < c.clip.Max.Y; y++ {
		for x := c.clip.Min.X; x < c.clip.Max.X; x++ {
			c.img.SetRGBA(x, y, color.RGBA{R: col.R, G: col.G, B: col.B, A: 255})
		}
	}
}

// panel returns a canvas sharing the same image, clipped to r.
func (c *Canvas) panel(r image.Rectangle) *Canvas {
	return &Canvas{img: c.img, clip: c.clip.Intersect(r)}
}

func (c *Canvas) rect() image.Rectangle { return c.clip }

func (c *Canvas) setPixel(x, y int, col RGB, alpha float64) {
	if alpha <= 0 {
		return
	}
	if x < c.clip.Min.X || x >= c.clip.Max.X || y < c.clip.Min.Y || y >= c.clip.Max.Y {
		return
	}
	if alpha >= 1 {
		c.img.SetRGBA(x, y, color.RGBA{R: col.R, G: col.G, B: col.B, A: 255})
		return
	}
	old := c.img.RGBAAt(x, y)
	a := alpha
	nr := float64(col.R)*a + float64(old.R)*(1-a)
	ng := float64(col.G)*a + float64(old.G)*(1-a)
	nb := float64(col.B)*a + float64(old.B)*(1-a)
	c.img.SetRGBA(x, y, color.RGBA{R: uint8(nr), G: uint8(ng), B: uint8(nb), A: 255})
}

func (c *Canvas) fillRect(x, y, w, h int, col RGB) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.setPixel(xx, yy, col, 1)
		}
	}
}

func (c *Canvas) frameRect(r image.Rectangle, thick int, col RGB) {
	c.fillRect(r.Min.X, r.Min.Y, r.Dx(), thick, col)
	c.fillRect(r.Min.X, r.Max.Y-thick, r.Dx(), thick, col)
	c.fillRect(r.Min.X, r.Min.Y, thick, r.Dy(), col)
	c.fillRect(r.Max.X-thick, r.Min.Y, thick, r.Dy(), col)
}

// line draws a round-capped thick segment in pixel coordinates.
func (c *Canvas) line(x0, y0, x1, y1 int, thick float64, col RGB) {
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	dist := math.Hypot(dx, dy)
	r := thick / 2
	steps := int(math.Max(1, math.Ceil(dist/math.Max(1.0, r*0.6))))
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		c.fillCircle(int(math.Round(float64(x0)+dx*t)), int(math.Round(float64(y0)+dy*t)), r, col)
	}
}

func (c *Canvas) dashedLine(x0, y0, x1, y1 int, thick float64, dashPx, gapPx float64, col RGB) {
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		return
	}
	ux, uy := dx/dist, dy/dist
	pos := 0.0
	on := true
	for pos < dist {
		seg := dashPx
		if !on {
			seg = gapPx
		}
		end := math.Min(dist, pos+seg)
		if on {
			c.line(int(math.Round(float64(x0)+ux*pos)), int(math.Round(float64(y0)+uy*pos)),
				int(math.Round(float64(x0)+ux*end)), int(math.Round(float64(y0)+uy*end)), thick, col)
		}
		pos = end
		on = !on
	}
}

func (c *Canvas) fillCircle(cx, cy int, radius float64, col RGB) {
	r := int(math.Ceil(radius))
	rr := radius * radius
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := float64(x-cx) + 0.0
			dy := float64(y-cy) + 0.0
			if dx*dx+dy*dy <= rr {
				c.setPixel(x, y, col, 1)
			}
		}
	}
}

// disc is an alpha-blended filled circle (for ghost poses).
func (c *Canvas) disc(cx, cy int, radius float64, col RGB, alpha float64) {
	r := int(math.Ceil(radius))
	rr := radius * radius
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			if dx*dx+dy*dy <= rr {
				c.setPixel(x, y, col, alpha)
			}
		}
	}
}

func (c *Canvas) ring(cx, cy int, radius, thick float64, col RGB) {
	r := int(math.Ceil(radius + thick))
	half := thick / 2
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			d := math.Hypot(float64(x-cx), float64(y-cy))
			if math.Abs(d-radius) <= half {
				c.setPixel(x, y, col, 1)
			}
		}
	}
}

// fillPolygon rasterizes a convex-ish polygon with a scanline test.
func (c *Canvas) fillPolygon(pts []image.Point, col RGB) {
	if len(pts) < 3 {
		return
	}
	minY, maxY := pts[0].Y, pts[0].Y
	minX, maxX := pts[0].X, pts[0].X
	for _, p := range pts[1:] {
		minY = minI(minY, p.Y)
		maxY = maxI(maxY, p.Y)
		minX = minI(minX, p.X)
		maxX = maxI(maxX, p.X)
	}
	for y := minY; y <= maxY; y++ {
		var xs []int
		for i := 0; i < len(pts); i++ {
			a := pts[i]
			b := pts[(i+1)%len(pts)]
			if (a.Y <= y && b.Y > y) || (b.Y <= y && a.Y > y) {
				xf := float64(a.X) + float64(y-a.Y)*float64(b.X-a.X)/float64(b.Y-a.Y)
				xs = append(xs, int(math.Round(xf)))
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			x0, x1 := xs[i], xs[i+1]
			if x0 > x1 {
				x0, x1 = x1, x0
			}
			for x := x0; x <= x1; x++ {
				c.setPixel(x, y, col, 1)
			}
		}
	}
	_ = minX
	_ = maxX
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// View maps world meters to absolute sheet pixels inside a panel.
type View struct {
	CX, CY float64 // absolute pixel position of world Center
	Center Vec
	Scale  float64 // pixels per meter
}

func (v View) px(p Vec) (int, int) {
	return int(math.Round(v.CX + (p.X-v.Center.X)*v.Scale)),
		int(math.Round(v.CY - (p.Y-v.Center.Y)*v.Scale))
}

func (v View) lenPx(meters float64) float64 { return meters * v.Scale }

type wcanvas struct {
	cv *Canvas
	v  View
}

func (w wcanvas) line(a, b Vec, thickM float64, col RGB) {
	x0, y0 := w.v.px(a)
	x1, y1 := w.v.px(b)
	w.cv.line(x0, y0, x1, y1, thickM*w.v.Scale, col)
}

func (w wcanvas) dashed(a, b Vec, thickM, dashM, gapM float64, col RGB) {
	x0, y0 := w.v.px(a)
	x1, y1 := w.v.px(b)
	w.cv.dashedLine(x0, y0, x1, y1, thickM*w.v.Scale, dashM*w.v.Scale, gapM*w.v.Scale, col)
}

func (w wcanvas) circle(p Vec, rM float64, col RGB) {
	x, y := w.v.px(p)
	w.cv.fillCircle(x, y, rM*w.v.Scale, col)
}

func (w wcanvas) ring(p Vec, rM, thickM float64, col RGB) {
	x, y := w.v.px(p)
	w.cv.ring(x, y, rM*w.v.Scale, thickM*w.v.Scale, col)
}
