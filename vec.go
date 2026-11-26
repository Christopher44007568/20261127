package qem

import (
	"math"
)

// Vec3 is a three-dimensional vector or point.
type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) LenSquared() float64 { return a.Dot(a) }

func (a Vec3) Len() float64 { return math.Sqrt(a.LenSquared()) }

func (a Vec3) Distance(b Vec3) float64 { return a.Sub(b).Len() }

func (a Vec3) Normalized() Vec3 {
	l := a.Len()
	if l == 0 || math.IsNaN(l) || math.IsInf(l, 0) {
		return Vec3{}
	}
	return a.Mul(1 / l)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return Vec3{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
	}
}

func (a Vec3) finite() bool {
	return !math.IsNaN(a.X) && !math.IsNaN(a.Y) && !math.IsNaN(a.Z) &&
		!math.IsInf(a.X, 0) && !math.IsInf(a.Y, 0) && !math.IsInf(a.Z, 0)
}

func det6(a, b, c Vec3) float64 {
	return a.X*(b.Y*c.Z-b.Z*c.Y) -
		a.Y*(b.X*c.Z-b.Z*c.X) +
		a.Z*(b.X*c.Y-b.Y*c.X)
}

// Edge is an undirected canonical mesh edge.
type Edge struct {
	A int `json:"a"`
	B int `json:"b"`
}

func NewEdge(a, b int) Edge {
	if a > b {
		a, b = b, a
	}
	return Edge{A: a, B: b}
}

func (e Edge) Contains(v int) bool { return e.A == v || e.B == v }
