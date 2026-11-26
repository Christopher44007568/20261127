package render

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"
)

// Canvas is a small software rasterizer with a depth buffer.
type Canvas struct {
	W, H int
	Pix  []color.RGBA
	Z    []float64
}

func NewCanvas(w, h int, bg color.RGBA) *Canvas {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	c := &Canvas{W: w, H: h, Pix: make([]color.RGBA, w*h), Z: make([]float64, w*h)}
	for i := range c.Pix {
		c.Pix[i] = bg
		c.Z[i] = math.Inf(1)
	}
	return c
}

func (c *Canvas) Index(x, y int) int { return y*c.W + x }

func (c *Canvas) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < c.W && y < c.H
}

func (c *Canvas) Blend(x, y int, col color.RGBA) {
	if !c.InBounds(x, y) || col.A == 0 {
		return
	}
	i := c.Index(x, y)
	if col.A == 255 {
		c.Pix[i] = col
		return
	}
	dst := c.Pix[i]
	a := float64(col.A) / 255
	c.Pix[i] = color.RGBA{
		R: uint8(float64(col.R)*a + float64(dst.R)*(1-a)),
		G: uint8(float64(col.G)*a + float64(dst.G)*(1-a)),
		B: uint8(float64(col.B)*a + float64(dst.B)*(1-a)),
		A: 255,
	}
}

// FillTriangle rasterizes a depth-tested screen-space triangle.
func (c *Canvas) FillTriangle(x0, y0, z0, x1, y1, z1, x2, y2, z2 float64, col color.RGBA) {
	den := (y1-y2)*(x0-x2) + (x2-x1)*(y0-y2)
	if math.Abs(den) < 1e-12 {
		return
	}
	minX := int(math.Floor(math.Min(x0, math.Min(x1, x2))))
	maxX := int(math.Ceil(math.Max(x0, math.Max(x1, x2))))
	minY := int(math.Floor(math.Min(y0, math.Min(y1, y2))))
	maxY := int(math.Ceil(math.Max(y0, math.Max(y1, y2))))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= c.W {
		maxX = c.W - 1
	}
	if maxY >= c.H {
		maxY = c.H - 1
	}
	for py := minY; py <= maxY; py++ {
		cy := float64(py) + 0.5
		for px := minX; px <= maxX; px++ {
			cx := float64(px) + 0.5
			w0 := ((y1-y2)*(cx-x2) + (x2-x1)*(cy-y2)) / den
			w1 := ((y2-y0)*(cx-x2) + (x0-x2)*(cy-y2)) / den
			w2 := 1 - w0 - w1
			if w0 < -1e-5 || w1 < -1e-5 || w2 < -1e-5 {
				continue
			}
			z := w0*z0 + w1*z1 + w2*z2
			i := c.Index(px, py)
			if z <= c.Z[i] {
				c.Z[i] = z
				c.Pix[i] = col
			}
		}
	}
}

// DrawLineZ draws a depth-tested line. z is the camera-space depth.
func (c *Canvas) DrawLineZ(x0, y0, z0, x1, y1, z1 float64, width int, col color.RGBA) {
	dx, dy := x1-x0, y1-y0
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	if steps < 1 {
		steps = 1
	}
	if width < 1 {
		width = 1
	}
	rad := width / 2
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(x0 + dx*t))
		y := int(math.Round(y0 + dy*t))
		z := z0 + (z1-z0)*t
		for oy := -rad; oy <= rad; oy++ {
			for ox := -rad; ox <= rad; ox++ {
				px, py := x+ox, y+oy
				if !c.InBounds(px, py) {
					continue
				}
				idx := c.Index(px, py)
				tol := math.Max(0.002, 0.012*math.Abs(z))
				if z <= c.Z[idx]+tol {
					c.Blend(px, py, col)
				}
			}
		}
	}
}

func (c *Canvas) DrawLine(x0, y0, x1, y1 float64, width int, col color.RGBA) {
	dx, dy := x1-x0, y1-y0
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	if steps < 1 {
		steps = 1
	}
	if width < 1 {
		width = 1
	}
	rad := width / 2
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(x0 + dx*t))
		y := int(math.Round(y0 + dy*t))
		for oy := -rad; oy <= rad; oy++ {
			for ox := -rad; ox <= rad; ox++ {
				c.Blend(x+ox, y+oy, col)
			}
		}
	}
}

func (c *Canvas) FillRect(x, y, w, h int, col color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			c.Blend(px, py, col)
		}
	}
}

func (c *Canvas) StrokeRect(x, y, w, h, lineWidth int, col color.RGBA) {
	c.FillRect(x, y, w, lineWidth, col)
	c.FillRect(x, y+h-lineWidth, w, lineWidth, col)
	c.FillRect(x, y, lineWidth, h, col)
	c.FillRect(x+w-lineWidth, y, lineWidth, h, col)
}

func (c *Canvas) FillCircle(cx, cy, radius int, col color.RGBA) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				c.Blend(x, y, col)
			}
		}
	}
}

// DrawText uses a compact built-in 5x7 font. Non-ASCII runs become filled
// boxes so missing glyphs are obvious rather than silently invisible.
func (c *Canvas) DrawText(x, y int, text string, scale int, col color.RGBA) {
	if scale < 1 {
		scale = 1
	}
	cursor := x
	for _, r := range strings.ToUpper(text) {
		if r == '\n' {
			continue
		}
		if r == ' ' {
			cursor += 6 * scale
			continue
		}
		g, ok := glyph5x7[r]
		if !ok {
			c.FillRect(cursor, y, 4*scale, 7*scale, col)
			cursor += 6 * scale
			continue
		}
		for row := 0; row < 7; row++ {
			bits := g[row]
			for k := 0; k < 5; k++ {
				if bits&(1<<(4-k)) != 0 {
					c.FillRect(cursor+k*scale, y+row*scale, scale, scale, col)
				}
			}
		}
		cursor += 6 * scale
	}
}

func TextWidth(text string, scale int) int {
	n := 0
	for _, r := range strings.ToUpper(text) {
		if r != '\n' {
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return n*6*scale - scale
}

// Downsample box-filters a supersampled canvas and writes an opaque PNG.
func (c *Canvas) Downsample(factor int) *image.RGBA {
	if factor < 1 {
		factor = 1
	}
	w, h := c.W/factor, c.H/factor
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b float64
			for oy := 0; oy < factor; oy++ {
				for ox := 0; ox < factor; ox++ {
					p := c.Pix[c.Index(x*factor+ox, y*factor+oy)]
					r += float64(p.R)
					g += float64(p.G)
					b += float64(p.B)
				}
			}
			n := float64(factor * factor)
			out.SetRGBA(x, y, color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), 255})
		}
	}
	return out
}

func SavePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
