package main

import "math"

// Vec2 is a 2D world-space vector used for both camera and character paths.
type Vec2 struct {
	X float64
	Y float64
}

func (v Vec2) Add(o Vec2) Vec2      { return Vec2{v.X + o.X, v.Y + o.Y} }
func (v Vec2) Sub(o Vec2) Vec2      { return Vec2{v.X - o.X, v.Y - o.Y} }
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }
func (v Vec2) Dot(o Vec2) float64   { return v.X*o.X + v.Y*o.Y }
func (v Vec2) Len() float64         { return math.Hypot(v.X, v.Y) }

func Dist(a, b Vec2) float64 { return math.Hypot(a.X-b.X, a.Y-b.Y) }

// Lerp interpolates linearly between a and b.
func Lerp(a, b, u float64) float64 { return a + (b-a)*u }

// LerpVec2 interpolates linearly between two points.
func LerpVec2(a, b Vec2, u float64) Vec2 {
	return Vec2{Lerp(a.X, b.X, u), Lerp(a.Y, b.Y, u)}
}

// AngleLerp interpolates between two angles along the shortest arc so that
// heading never spins the long way around.
func AngleLerp(a, b, u float64) float64 {
	diff := normalizeAngle(b - a)
	if diff > math.Pi {
		diff -= 2 * math.Pi
	}
	return normalizeAngle(a + diff*u)
}

func normalizeAngle(a float64) float64 {
	const twoPi = 2 * math.Pi
	return math.Mod(a+twoPi*64, twoPi)
}

// Clamp keeps v inside [lo, hi].
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Rect is an axis-aligned world rectangle (the playable bounds).
type Rect struct {
	Min Vec2
	Max Vec2
}

func (r Rect) Contains(p Vec2) bool {
	const eps = 1e-9
	return p.X >= r.Min.X-eps && p.X <= r.Max.X+eps &&
		p.Y >= r.Min.Y-eps && p.Y <= r.Max.Y+eps
}

func (r Rect) Width() float64  { return r.Max.X - r.Min.X }
func (r Rect) Height() float64 { return r.Max.Y - r.Min.Y }
