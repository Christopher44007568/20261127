package main

import "math"

// Vec is a 2D vector in the sagittal plane: +X points right/forward, +Y is up.
type Vec struct {
	X, Y float64
}

func vec(x, y float64) Vec { return Vec{X: x, Y: y} }

func (a Vec) add(b Vec) Vec { return Vec{X: a.X + b.X, Y: a.Y + b.Y} }

func (a Vec) sub(b Vec) Vec { return Vec{X: a.X - b.X, Y: a.Y - b.Y} }

func (a Vec) mul(s float64) Vec { return Vec{X: a.X * s, Y: a.Y * s} }

func (a Vec) dot(b Vec) float64 { return a.X*b.X + a.Y*b.Y }

// cross returns the z component of a x b. Positive means b is CCW from a.
func (a Vec) cross(b Vec) float64 { return a.X*b.Y - a.Y*b.X }

func (a Vec) len() float64 { return math.Hypot(a.X, a.Y) }

func (a Vec) norm() Vec {
	l := a.len()
	if l < 1e-15 {
		return Vec{}
	}
	return a.mul(1 / l)
}

func (a Vec) angle() float64 { return math.Atan2(a.Y, a.X) }

func (a Vec) rotate(ang float64) Vec {
	c, s := math.Cos(ang), math.Sin(ang)
	return Vec{X: a.X*c - a.Y*s, Y: a.X*s + a.Y*c}
}

func lerpVec(a, b Vec, t float64) Vec {
	return Vec{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t}
}

func clampF(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func clamp01(x float64) float64 { return clampF(x, 0, 1) }

// wrapAngle maps an angle to (-pi, pi].
func wrapAngle(a float64) float64 {
	return math.Atan2(math.Sin(a), math.Cos(a))
}

func smoothStep(t float64) float64 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}

func deg(r float64) float64 { return r * 180 / math.Pi }

func rad(d float64) float64 { return d * math.Pi / 180 }
