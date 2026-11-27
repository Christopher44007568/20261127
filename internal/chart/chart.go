// Package chart provides tiny zero-dependency PNG plotting helpers.
package chart

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"materiallod/internal/fontx"
)

type Point struct{ X, Y float64 }

type Canvas struct{ Img *image.RGBA }

func New(width, height int, bg color.Color) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	return &Canvas{Img: img}
}

func Hex(hex uint32) color.RGBA {
	return color.RGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 255}
}

func (c *Canvas) Rect(x0, y0, x1, y1 int, fill color.Color) {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	r := image.Rect(x0, y0, x1, y1)
	r = r.Intersect(c.Img.Bounds())
	if r.Empty() {
		return
	}
	draw.Draw(c.Img, r, &image.Uniform{C: fill}, image.Point{}, draw.Src)
}

func (c *Canvas) Frame(x0, y0, x1, y1 int, col color.Color, thickness int) {
	c.Rect(x0, y0, x1, y0+thickness, col)
	c.Rect(x0, y1-thickness, x1, y1, col)
	c.Rect(x0, y0, x0+thickness, y1, col)
	c.Rect(x1-thickness, y0, x1, y1, col)
}

func (c *Canvas) Text(x, y int, text string, scale int, col color.Color) {
	fontx.Draw(c.Img, x, y, text, scale, col)
}

func (c *Canvas) TextCenter(cx, y int, text string, scale int, col color.Color) {
	fontx.Centered(c.Img, cx, y, text, scale, col)
}

func (c *Canvas) TextRight(x, y int, text string, scale int, col color.Color) {
	w, _ := fontx.Measure(text, scale)
	fontx.Draw(c.Img, x-w, y, text, scale, col)
}

func (c *Canvas) Line(x0, y0, x1, y1 int, col color.Color, thickness int) {
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
	r := thickness / 2
	for {
		c.Rect(x0-r, y0-r, x0+r+1, y0+r+1, col)
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

func (c *Canvas) DashedLine(x0, y0, x1, y1 int, col color.Color, thickness, dash, gap int) {
	length := math.Hypot(float64(x1-x0), float64(y1-y0))
	if length == 0 {
		return
	}
	ux := float64(x1-x0) / length
	uy := float64(y1-y0) / length
	for d := 0.0; d < length; d += float64(dash + gap) {
		end := math.Min(d+float64(dash), length)
		c.Line(
			int(math.Round(float64(x0)+ux*d)),
			int(math.Round(float64(y0)+uy*d)),
			int(math.Round(float64(x0)+ux*end)),
			int(math.Round(float64(y0)+uy*end)),
			col, thickness,
		)
	}
}

func (c *Canvas) Polyline(points []Point, col color.Color, thickness int) {
	for i := 1; i < len(points); i++ {
		c.Line(int(math.Round(points[i-1].X)), int(math.Round(points[i-1].Y)),
			int(math.Round(points[i].X)), int(math.Round(points[i].Y)), col, thickness)
	}
}

// StyledPolyline draws a solid line when dash <= 0, otherwise dash/gap strokes.
func (c *Canvas) StyledPolyline(points []Point, col color.Color, thickness int, dash, gap float64) {
	if dash <= 0 || gap <= 0 {
		c.Polyline(points, col, thickness)
		return
	}
	period := dash + gap
	traveled := 0.0
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		seg := math.Hypot(b.X-a.X, b.Y-a.Y)
		steps := int(math.Ceil(seg))
		if steps < 1 {
			continue
		}
		ux, uy := (b.X-a.X)/seg, (b.Y-a.Y)/seg
		for k := 0; k < steps; k++ {
			d0 := float64(k) / float64(steps) * seg
			d1 := float64(k+1) / float64(steps) * seg
			mid := math.Mod(traveled+(d0+d1)/2, period)
			if mid >= dash {
				continue
			}
			c.Line(
				int(math.Round(a.X+ux*d0)), int(math.Round(a.Y+uy*d0)),
				int(math.Round(a.X+ux*d1)), int(math.Round(a.Y+uy*d1)),
				col, thickness,
			)
		}
		traveled += seg
	}
}
