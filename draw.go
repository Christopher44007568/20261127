package qem

import (
	"math"
)

// DrawLine draws a Bresenham line. Coordinates are clipped by Set/Blend.
func (p *PPM) DrawLine(x0, y0, x1, y1, thickness int, c RGB, alpha float64) {
	if thickness < 1 {
		thickness = 1
	}
	dx := math.Abs(float64(x1 - x0))
	dy := -math.Abs(float64(y1 - y0))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	r := thickness / 2
	for {
		for oy := -r; oy <= r; oy++ {
			for ox := -r; ox <= r; ox++ {
				if ox*ox+oy*oy <= r*r+r {
					p.Blend(x0+ox, y0+oy, c, alpha)
				}
			}
		}
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

// DrawLineDepth draws a z-tested line. Larger projected z is nearer, matching
// the renderer's depth buffer. Bias is in model units.
func (p *PPM) DrawLineDepth(x0, y0 int, z0 float64, x1, y1 int, z1 float64, thickness int, c RGB, alpha, bias float64, depth []float64) {
	if thickness < 1 {
		thickness = 1
	}
	dx := math.Abs(float64(x1 - x0))
	dy := -math.Abs(float64(y1 - y0))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	r := thickness / 2
	steps := int(math.Max(math.Abs(float64(x1-x0)), math.Abs(float64(y1-y0))))
	iteration := 0
	for {
		t := 0.0
		if steps > 0 {
			t = float64(iteration) / float64(steps)
		}
		z := z0 + (z1-z0)*t
		for oy := -r; oy <= r; oy++ {
			for ox := -r; ox <= r; ox++ {
				if ox*ox+oy*oy > r*r+r {
					continue
				}
				x, y := x0+ox, y0+oy
				if !p.InBounds(x, y) {
					continue
				}
				idx := p.index(x, y)
				if z+bias < depth[idx] {
					continue
				}
				p.Blend(x, y, c, alpha)
			}
		}
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
		iteration++
	}
}

type projectedPoint struct {
	x, y, z float64
}

// ViewFrame fixes fitting to the original mesh. Reusing it for every level is
// essential: refitting each simplified mesh would hide silhouette shrinkage.
type ViewFrame struct {
	Center        Vec3
	MaxDimension  float64
	PixelsPerUnit float64
	Width         int
	Height        int
}

func NewViewFrame(m *Mesh, width, height int) ViewFrame {
	maxDim := m.MaxDimension()
	margin := 0.84
	ppu := margin * float64(minInt(width, height)) / maxDim
	return ViewFrame{
		Center:        m.Center(),
		MaxDimension:  maxDim,
		PixelsPerUnit: ppu,
		Width:         width,
		Height:        height,
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type View struct {
	Yaw   float64
	Pitch float64
}

func (f ViewFrame) Project(p Vec3, view View) projectedPoint {
	q := p.Sub(f.Center)
	cy, sy := math.Cos(view.Yaw), math.Sin(view.Yaw)
	x := q.X*cy + q.Z*sy
	z0 := -q.X*sy + q.Z*cy
	y := q.Y
	cp, sp := math.Cos(view.Pitch), math.Sin(view.Pitch)
	y2 := y*cp - z0*sp
	z := y*sp + z0*cp
	return projectedPoint{
		x: float64(f.Width)/2 + x*f.PixelsPerUnit,
		y: float64(f.Height)/2 - y2*f.PixelsPerUnit,
		z: z,
	}
}

func (f ViewFrame) UnprojectPixelLength(pixels float64) float64 {
	return pixels / f.PixelsPerUnit
}
