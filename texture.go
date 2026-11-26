package main

import (
	"image"
	"image/color"
	"math"
)

var (
	textureA = color.NRGBA{R: 18, G: 52, B: 72, A: 255}
	textureB = color.NRGBA{R: 21, G: 78, B: 84, A: 255}
	deepInk  = color.NRGBA{R: 7, G: 27, B: 39, A: 255}
	gold     = color.NRGBA{R: 224, G: 171, B: 77, A: 255}
	cream    = color.NRGBA{R: 241, G: 222, B: 181, A: 255}
	coral    = color.NRGBA{R: 224, G: 83, B: 55, A: 255}
	aqua     = color.NRGBA{R: 63, G: 194, B: 180, A: 255}
)

// MakeTexture draws a periodic source tile. The final 2x downsample gives the
// fine concentric rings and arrow edges stable antialiasing without a font or
// image dependency.
func MakeTexture(size int) *image.NRGBA {
	if size < 64 {
		size = 64
	}
	hi := size * 2
	img := image.NewNRGBA(image.Rect(0, 0, hi, hi))
	drawWovenGround(img, hi)
	drawCornerRivets(img, hi)
	drawFlowMarks(img, hi)
	drawCompass(img, hi)
	return downsample(img, size)
}

func drawWovenGround(img *image.NRGBA, size int) {
	cells := 8
	cell := float64(size) / float64(cells)
	for y := 0; y < size; y++ {
		fy := float64(y) + 0.5
		cy := int(fy / cell)
		for x := 0; x < size; x++ {
			fx := float64(x) + 0.5
			cx := int(fx / cell)

			c := textureA
			if (cx+cy)%2 == 0 {
				c = textureB
			}
			tone := 0.86 + 0.14*math.Cos(2*math.Pi*(fx+fy)/float64(size))
			c.R = clamp8(float64(c.R) * tone)
			c.G = clamp8(float64(c.G) * tone)
			c.B = clamp8(float64(c.B) * tone)
			img.SetNRGBA(x, y, c)

			weave := math.Abs(math.Sin(2 * math.Pi * (5*fx - 3*fy) / float64(size)))
			if weave < 0.035 {
				blendPixel(img, x, y, gold, 0.055)
			}
		}
	}
}

func drawCornerRivets(img *image.NRGBA, size int) {
	hi := float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := math.Min(float64(x)+0.5, hi-(float64(x)+0.5))
			dy := math.Min(float64(y)+0.5, hi-(float64(y)+0.5))
			r := math.Hypot(dx, dy)
			if (r >= 37 && r <= 44) || r <= 15 {
				blendPixel(img, x, y, gold, 0.92)
			}
			if r >= 20 && r <= 25 {
				blendPixel(img, x, y, deepInk, 0.62)
			}
		}
	}
}

func drawFlowMarks(img *image.NRGBA, size int) {
	hi := float64(size)
	for i := 0; i < 3; i++ {
		x := hi * (0.18 + float64(i)*0.13)
		y := hi * (0.18 + float64(i)*0.05)
		drawDisc(img, x, y, hi*0.013, gold)
		drawDisc(img, hi-x, hi-y, hi*0.020, aqua)
	}
}

func drawCompass(img *image.NRGBA, size int) {
	hi := float64(size)
	cx, cy := hi*0.5, hi*0.5

	drawDisc(img, cx+hi*0.018, cy+hi*0.024, hi*0.277, deepInk)
	drawDisc(img, cx, cy, hi*0.262, cream)
	drawRing(img, cx, cy, hi*0.273, hi*0.010, deepInk)
	drawRing(img, cx, cy, hi*0.258, hi*0.007, gold)

	start := point{hi * 0.315, hi * 0.655}
	tip := point{hi * 0.685, hi * 0.345}
	drawSegment(img, start, tip, hi*0.055, deepInk)
	drawSegment(img, start, tip, hi*0.038, coral)

	dx, dy := tip.x-start.x, tip.y-start.y
	length := math.Hypot(dx, dy)
	ux, uy := dx/length, dy/length
	nx, ny := -uy, ux
	base := point{tip.x - ux*hi*0.115, tip.y - uy*hi*0.115}
	head := []point{
		tip,
		{base.x + nx*hi*0.052, base.y + ny*hi*0.052},
		{base.x - nx*hi*0.052, base.y - ny*hi*0.052},
	}
	drawPolygon(img, head, coral)

	drawDisc(img, start.x, start.y, hi*0.046, deepInk)
	drawDisc(img, start.x, start.y, hi*0.031, aqua)
	drawDisc(img, start.x+hi*0.010, start.y-hi*0.012, hi*0.010, cream)

	for i := 0; i < 3; i++ {
		x := cx + hi*(0.64+float64(i)*0.025)
		y := cy - hi*(0.50-float64(i)*0.035)
		drawSegment(img,
			point{x, y}, point{x + hi*0.025, y - hi*0.010},
			hi*0.008, gold)
	}
}

type point struct{ x, y float64 }

func drawDisc(img *image.NRGBA, cx, cy, radius float64, c color.NRGBA) {
	for y := int(math.Floor(cy - radius)); y <= int(math.Ceil(cy+radius)); y++ {
		for x := int(math.Floor(cx - radius)); x <= int(math.Ceil(cx+radius)); x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			coverage := math.Max(0, math.Min(1, radius-d+0.5))
			if coverage > 0 {
				blendPixel(img, x, y, c, coverage)
			}
		}
	}
}

func drawRing(img *image.NRGBA, cx, cy, radius, thickness float64, c color.NRGBA) {
	outer := radius + thickness*0.5
	inner := radius - thickness*0.5
	for y := int(math.Floor(cy - outer)); y <= int(math.Ceil(cy+outer)); y++ {
		for x := int(math.Floor(cx - outer)); x <= int(math.Ceil(cx+outer)); x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			coverage := math.Max(0, math.Min(1, d-inner+0.5))
			coverage = math.Min(coverage, math.Max(0, outer-d+0.5))
			if coverage > 0 {
				blendPixel(img, x, y, c, coverage)
			}
		}
	}
}

func drawSegment(img *image.NRGBA, a, b point, width float64, c color.NRGBA) {
	minX := math.Min(a.x, b.x) - width
	maxX := math.Max(a.x, b.x) + width
	minY := math.Min(a.y, b.y) - width
	maxY := math.Max(a.y, b.y) + width
	for y := int(math.Floor(minY)); y <= int(math.Ceil(maxY)); y++ {
		for x := int(math.Floor(minX)); x <= int(math.Ceil(maxX)); x++ {
			d := distanceToSegment(point{float64(x) + 0.5, float64(y) + 0.5}, a, b)
			coverage := math.Max(0, math.Min(1, width*0.5-d+0.5))
			if coverage > 0 {
				blendPixel(img, x, y, c, coverage)
			}
		}
	}
}

func distanceToSegment(p, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	if dx == 0 && dy == 0 {
		return math.Hypot(p.x-a.x, p.y-a.y)
	}
	t := ((p.x-a.x)*dx + (p.y-a.y)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(p.x-(a.x+t*dx), p.y-(a.y+t*dy))
}

func drawPolygon(img *image.NRGBA, vertices []point, c color.NRGBA) {
	if len(vertices) < 3 {
		return
	}
	minX, maxX := vertices[0].x, vertices[0].x
	minY, maxY := vertices[0].y, vertices[0].y
	for _, p := range vertices[1:] {
		minX, maxX = math.Min(minX, p.x), math.Max(maxX, p.x)
		minY, maxY = math.Min(minY, p.y), math.Max(maxY, p.y)
	}
	for y := int(math.Floor(minY)); y <= int(math.Ceil(maxY)); y++ {
		for x := int(math.Floor(minX)); x <= int(math.Ceil(maxX)); x++ {
			if pointInPolygon(float64(x)+0.5, float64(y)+0.5, vertices) {
				blendPixel(img, x, y, c, 1)
			}
		}
	}
}

func pointInPolygon(x, y float64, vertices []point) bool {
	inside := false
	j := len(vertices) - 1
	for i := range vertices {
		yi, yj := vertices[i].y, vertices[j].y
		if (yi > y) != (yj > y) &&
			x < (vertices[j].x-vertices[i].x)*(y-yi)/(yj-yi)+vertices[i].x {
			inside = !inside
		}
		j = i
	}
	return inside
}

func blendPixel(img *image.NRGBA, x, y int, c color.NRGBA, coverage float64) {
	if !image.Pt(x, y).In(img.Bounds()) || coverage <= 0 {
		return
	}
	alpha := coverage * float64(c.A) / 255
	if alpha > 1 {
		alpha = 1
	}
	dst := img.NRGBAAt(x, y)
	img.SetNRGBA(x, y, color.NRGBA{
		R: clamp8(float64(dst.R)*(1-alpha) + float64(c.R)*alpha),
		G: clamp8(float64(dst.G)*(1-alpha) + float64(c.G)*alpha),
		B: clamp8(float64(dst.B)*(1-alpha) + float64(c.B)*alpha),
		A: clamp8(float64(dst.A)*(1-alpha) + 255*alpha),
	})
}

func downsample(src *image.NRGBA, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := src.Bounds().Dx() / size
	if scale < 1 {
		scale = 1
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a uint64
			var count uint64
			for sy := y * scale; sy < (y+1)*scale; sy++ {
				for sx := x * scale; sx < (x+1)*scale; sx++ {
					c := src.NRGBAAt(sx, sy)
					r += uint64(c.R)
					g += uint64(c.G)
					b += uint64(c.B)
					a += uint64(c.A)
					count++
				}
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r / count), G: uint8(g / count), B: uint8(b / count),
				A: uint8(a / count),
			})
		}
	}
	return dst
}
