package main

import (
	"image"
	"math"
)

type PointF struct {
	X, Y float64
}

type StrokePath struct {
	Points []PointF
	Width  float64
	Closed bool
}

// rasterizeArea evaluates samples*samples subpixels per pixel. Averaging the
// binary results produces genuine anti-aliased coverage in the alpha channel.
func rasterizeArea(bounds, area image.Rectangle, samples int, inside func(PointF) bool) *Mask {
	mask := NewMask(bounds)
	if samples < 1 {
		samples = 1
	}
	area = area.Intersect(bounds)
	if area.Empty() {
		return mask
	}

	total := float64(samples * samples)
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			hits := 0
			for sy := 0; sy < samples; sy++ {
				fy := (float64(sy) + 0.5) / float64(samples)
				py := float64(y) + fy
				for sx := 0; sx < samples; sx++ {
					fx := (float64(sx) + 0.5) / float64(samples)
					px := float64(x) + fx
					if inside(PointF{X: px, Y: py}) {
						hits++
					}
				}
			}
			coverage := uint8(math.Round(255 * float64(hits) / total))
			mask.setAlpha(x, y, coverage)
		}
	}
	return mask
}

func CircleMask(bounds image.Rectangle, center PointF, radius float64, samples int) *Mask {
	area := image.Rect(
		int(math.Floor(center.X-radius))-1,
		int(math.Floor(center.Y-radius))-1,
		int(math.Ceil(center.X+radius))+2,
		int(math.Ceil(center.Y+radius))+2,
	)
	radiusSquared := radius * radius
	return rasterizeArea(bounds, area, samples, func(p PointF) bool {
		dx := p.X - center.X
		dy := p.Y - center.Y
		return dx*dx+dy*dy <= radiusSquared
	})
}

func StarVertices(center PointF, outerRadius, innerRadius float64, points int, rotation float64) []PointF {
	vertices := make([]PointF, 0, points*2)
	for i := 0; i < points*2; i++ {
		radius := innerRadius
		if i%2 == 0 {
			radius = outerRadius
		}
		angle := rotation + float64(i)*math.Pi/float64(points)
		vertices = append(vertices, PointF{
			X: center.X + math.Cos(angle)*radius,
			Y: center.Y + math.Sin(angle)*radius,
		})
	}
	return vertices
}

func PolygonMask(bounds image.Rectangle, polygon []PointF, samples int) *Mask {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, p := range polygon {
		minX = math.Min(minX, p.X)
		maxX = math.Max(maxX, p.X)
		minY = math.Min(minY, p.Y)
		maxY = math.Max(maxY, p.Y)
	}
	area := image.Rect(int(math.Floor(minX))-1, int(math.Floor(minY))-1,
		int(math.Ceil(maxX))+2, int(math.Ceil(maxY))+2)
	return rasterizeArea(bounds, area, samples, func(p PointF) bool {
		return pointInPolygon(p, polygon)
	})
}

func pointInPolygon(p PointF, polygon []PointF) bool {
	inside := false
	n := len(polygon)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		a := polygon[j]
		b := polygon[i]
		if (a.Y > p.Y) != (b.Y > p.Y) {
			crossX := (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y) + a.X
			if p.X < crossX {
				inside = !inside
			}
		}
	}
	return inside
}

func StrokeMask(bounds image.Rectangle, paths []StrokePath, samples int) *Mask {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, path := range paths {
		half := path.Width / 2
		for _, p := range path.Points {
			minX = math.Min(minX, p.X-half)
			maxX = math.Max(maxX, p.X+half)
			minY = math.Min(minY, p.Y-half)
			maxY = math.Max(maxY, p.Y+half)
		}
	}
	area := image.Rect(int(math.Floor(minX))-1, int(math.Floor(minY))-1,
		int(math.Ceil(maxX))+2, int(math.Ceil(maxY))+2)
	return rasterizeArea(bounds, area, samples, func(p PointF) bool {
		for _, path := range paths {
			if strokePathContains(path, p) {
				return true
			}
		}
		return false
	})
}

func strokePathContains(path StrokePath, p PointF) bool {
	if len(path.Points) == 1 {
		dx := p.X - path.Points[0].X
		dy := p.Y - path.Points[0].Y
		half := path.Width / 2
		return dx*dx+dy*dy <= half*half
	}
	limitSquared := (path.Width / 2) * (path.Width / 2)
	segments := len(path.Points) - 1
	if path.Closed {
		segments = len(path.Points)
	}
	for i := 0; i < segments; i++ {
		a := path.Points[i]
		b := path.Points[(i+1)%len(path.Points)]
		if distanceToSegmentSquared(p, a, b) <= limitSquared {
			return true
		}
	}
	return false
}

func distanceToSegmentSquared(p, a, b PointF) float64 {
	vx := b.X - a.X
	vy := b.Y - a.Y
	wx := p.X - a.X
	wy := p.Y - a.Y
	lengthSquared := vx*vx + vy*vy
	if lengthSquared == 0 {
		return wx*wx + wy*wy
	}
	t := (wx*vx + wy*vy) / lengthSquared
	t = math.Max(0, math.Min(1, t))
	qx := a.X + t*vx
	qy := a.Y + t*vy
	dx := p.X - qx
	dy := p.Y - qy
	return dx*dx + dy*dy
}
