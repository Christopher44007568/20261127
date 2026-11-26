package holefill

import (
	"bufio"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
)

type RGB struct {
	R, G, B uint8
}

func (c RGB) RGBA() color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
}

func Blend(a, b RGB, t float64) RGB {
	t = math.Max(0, math.Min(1, t))
	return RGB{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
	}
}

type Canvas struct {
	W, H int
	Pix  []RGB
	Z    []float64
}

func NewCanvas(w, h int, bg RGB) *Canvas {
	pix := make([]RGB, w*h)
	for i := range pix {
		pix[i] = bg
	}
	z := make([]float64, w*h)
	for i := range z {
		z[i] = math.Inf(-1)
	}
	return &Canvas{W: w, H: h, Pix: pix, Z: z}
}

func (c *Canvas) ClearDepth() {
	for i := range c.Z {
		c.Z[i] = math.Inf(-1)
	}
}

func (c *Canvas) Set(x, y int, col RGB) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Pix[y*c.W+x] = col
}

func (c *Canvas) SetDepth(x, y int, depth float64, col RGB) bool {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return false
	}
	i := y*c.W + x
	if depth <= c.Z[i] {
		return false
	}
	c.Z[i] = depth
	c.Pix[i] = col
	return true
}

func (c *Canvas) Get(x, y int) RGB {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return RGB{}
	}
	return c.Pix[y*c.W+x]
}

func (c *Canvas) FillRect(x0, y0, x1, y1 int, col RGB) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			c.Set(x, y, col)
		}
	}
}

func (c *Canvas) FrameRect(x0, y0, x1, y1, thick int, col RGB) {
	c.FillRect(x0, y0, x1, y0+thick-1, col)
	c.FillRect(x0, y1-thick+1, x1, y1, col)
	c.FillRect(x0, y0, x0+thick-1, y1, col)
	c.FillRect(x1-thick+1, y0, x1, y1, col)
}

func (c *Canvas) Line(x0, y0, x1, y1, thick int, col RGB) {
	dx := math.Abs(float64(x1 - x0))
	dy := -math.Abs(float64(y1 - y0))
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	r := thick / 2
	for {
		c.FillRect(x0-r, y0-r, x0+r, y0+r, col)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (c *Canvas) WritePPM(w io.Writer) error {
	bw := bufio.NewWriter(w)
	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n255\n", c.W, c.H); err != nil {
		return err
	}
	for _, p := range c.Pix {
		if _, err := bw.Write([]byte{p.R, p.G, p.B}); err != nil {
			return err
		}
	}
	return bw.Flush()
}

func WritePPMFile(path string, c *Canvas) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := c.WritePPM(f); err != nil {
		return err
	}
	return f.Close()
}
