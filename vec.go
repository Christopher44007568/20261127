package main

import "math"

// Vec2 is a two-dimensional point or vector in font design units.
type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (p Vec2) Add(q Vec2) Vec2 { return Vec2{X: p.X + q.X, Y: p.Y + q.Y} }
func (p Vec2) Sub(q Vec2) Vec2 { return Vec2{X: p.X - q.X, Y: p.Y - q.Y} }
func (p Vec2) Scale(s float64) Vec2 {
	return Vec2{X: p.X * s, Y: p.Y * s}
}
func (p Vec2) Dot(q Vec2) float64   { return p.X*q.X + p.Y*q.Y }
func (p Vec2) Cross(q Vec2) float64 { return p.X*q.Y - p.Y*q.X }
func (p Vec2) Len() float64         { return math.Hypot(p.X, p.Y) }
func (p Vec2) Len2() float64        { return p.X*p.X + p.Y*p.Y }

func (p Vec2) Normalized() Vec2 {
	l := p.Len()
	if l < 1e-15 {
		return Vec2{}
	}
	return p.Scale(1 / l)
}

// LeftNormal returns the unit normal 90 degrees counter-clockwise.
func (p Vec2) LeftNormal() Vec2 { return Vec2{X: -p.Y, Y: p.X} }

// RightNormal returns the unit normal 90 degrees clockwise.
func (p Vec2) RightNormal() Vec2 { return Vec2{X: p.Y, Y: -p.X} }

func (p Vec2) ApproxEqual(q Vec2, eps float64) bool {
	return math.Abs(p.X-q.X) <= eps && math.Abs(p.Y-q.Y) <= eps
}

// Bounds is an axis-aligned bounding rectangle.
type Bounds struct {
	Min Vec2
	Max Vec2
}

func NewBounds() Bounds {
	inf := math.Inf(1)
	return Bounds{Min: Vec2{X: inf, Y: inf}, Max: Vec2{X: -inf, Y: -inf}}
}

func (b Bounds) Include(p Vec2) Bounds {
	b.Min.X = math.Min(b.Min.X, p.X)
	b.Min.Y = math.Min(b.Min.Y, p.Y)
	b.Max.X = math.Max(b.Max.X, p.X)
	b.Max.Y = math.Max(b.Max.Y, p.Y)
	return b
}

func (b Bounds) Empty() bool {
	return math.IsInf(b.Min.X, 1) || b.Min.X > b.Max.X || b.Min.Y > b.Max.Y
}

func (b Bounds) Width() float64  { return b.Max.X - b.Min.X }
func (b Bounds) Height() float64 { return b.Max.Y - b.Min.Y }

func (b Bounds) Expanded(d float64) Bounds {
	return Bounds{
		Min: Vec2{X: b.Min.X - d, Y: b.Min.Y - d},
		Max: Vec2{X: b.Max.X + d, Y: b.Max.Y + d},
	}
}
