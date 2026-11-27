package main

import "math"

type PixelRect struct {
	X0, Y0, X1, Y1 int
}

func sampleLinear(c *Canvas, x, y float64) Vec3 {
	x = clamp(x, 0, float64(c.W)-1.001)
	y = clamp(y, 0, float64(c.H)-1.001)
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	tx, ty := x-float64(x0), y-float64(y0)
	get := func(ix, iy int) Vec3 {
		ix = int(clamp(float64(ix), 0, float64(c.W-1)))
		iy = int(clamp(float64(iy), 0, float64(c.H-1)))
		return c.Pix[c.index(ix, iy)]
	}
	a := get(x0, y0)
	b := get(x0+1, y0)
	d := get(x0, y0+1)
	e := get(x0+1, y0+1)
	top := a.Mul(1 - tx).Add(b.Mul(tx))
	bottom := d.Mul(1 - tx).Add(e.Mul(tx))
	return top.Mul(1 - ty).Add(bottom.Mul(ty))
}

// Magnify copies a world-space source rectangle into a pixel rectangle. The
// rectangles must be disjoint so the framebuffer can be sampled while writing.
func (c *Canvas) Magnify(src Rect, dst PixelRect) {
	w := dst.X1 - dst.X0
	h := dst.Y1 - dst.Y0
	for y := dst.Y0; y < dst.Y1; y++ {
		fy := (float64(y-dst.Y0) + 0.5) / float64(h)
		worldY := mix(src.Max.Y, src.Min.Y, fy)
		for x := dst.X0; x < dst.X1; x++ {
			fx := (float64(x-dst.X0) + 0.5) / float64(w)
			worldX := mix(src.Min.X, src.Max.X, fx)
			sx, sy := c.WorldToPixel(v2(worldX, worldY))
			c.Pix[c.index(x, y)] = sampleLinear(c, sx, sy)
		}
	}
}

func (c *Canvas) PixelRectBorder(r PixelRect, col Vec3) {
	a := Vec2{float64(r.X0), float64(r.Y0)}
	b := Vec2{float64(r.X1), float64(r.Y0)}
	c.LinePixels(a, b, LineStyle{Color: col, Width: 3, Alpha: 0.95})
	b = Vec2{float64(r.X1), float64(r.Y1)}
	c.LinePixels(Vec2{float64(r.X1), float64(r.Y0)}, b, LineStyle{Color: col, Width: 3, Alpha: 0.95})
	c.LinePixels(b, Vec2{float64(r.X0), float64(r.Y1)}, LineStyle{Color: col, Width: 3, Alpha: 0.95})
	c.LinePixels(Vec2{float64(r.X0), float64(r.Y1)}, a, LineStyle{Color: col, Width: 3, Alpha: 0.95})
}

func (c *Canvas) Connector(src Rect, dst PixelRect, col Vec3) {
	srcAX, srcAY := c.WorldToPixel(src.Min)
	srcBX, srcBY := c.WorldToPixel(src.Max)
	srcTL := Vec2{srcAX, srcBY} // min x, max y
	srcBR := Vec2{srcBX, srcAY} // max x, min y
	dstTL := Vec2{float64(dst.X0), float64(dst.Y0)}
	dstBR := Vec2{float64(dst.X1), float64(dst.Y1)}
	c.LinePixels(srcTL, dstTL, LineStyle{Color: col, Width: 1.6, Alpha: 0.55})
	c.LinePixels(srcBR, dstBR, LineStyle{Color: col, Width: 1.6, Alpha: 0.55})
}
