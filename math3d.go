package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3    { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }
func (a Vec3) Sub(b Vec3) Vec3    { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }
func (a Vec3) Mul(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }
func (a Vec3) Div(s float64) Vec3 { return V(a.X/s, a.Y/s, a.Z/s) }
func (a Vec3) Neg() Vec3          { return V(-a.X, -a.Y, -a.Z) }
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) LengthSquared() float64 { return a.Dot(a) }
func (a Vec3) Length() float64        { return math.Sqrt(a.LengthSquared()) }

func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l < 1e-12 {
		return V(0, 0, 0)
	}
	return a.Div(l)
}

func (a Vec3) MulVec(b Vec3) Vec3 {
	return V(a.X*b.X, a.Y*b.Y, a.Z*b.Z)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Mul(t))
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func clampInt(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func smoothstep(a, b, x float64) float64 {
	if a == b {
		return 0
	}
	t := clamp((x-a)/(b-a), 0, 1)
	return t * t * (3 - 2*t)
}

func mix(a, b, t float64) float64 { return a + (b-a)*t }

type Ray struct {
	Origin    Vec3
	Direction Vec3
}

func (r Ray) At(t float64) Vec3 { return r.Origin.Add(r.Direction.Mul(t)) }

type AABB struct {
	Min Vec3
	Max Vec3
}

func (b AABB) Center() Vec3 {
	return b.Min.Add(b.Max).Mul(0.5)
}

func (b AABB) Extent() Vec3 { return b.Max.Sub(b.Min) }

func (b AABB) Union(c AABB) AABB {
	return AABB{
		Min: V(math.Min(b.Min.X, c.Min.X), math.Min(b.Min.Y, c.Min.Y), math.Min(b.Min.Z, c.Min.Z)),
		Max: V(math.Max(b.Max.X, c.Max.X), math.Max(b.Max.Y, c.Max.Y), math.Max(b.Max.Z, c.Max.Z)),
	}
}

func (b AABB) Empty() bool {
	return b.Min.X > b.Max.X || b.Min.Y > b.Max.Y || b.Min.Z > b.Max.Z
}

func emptyAABB() AABB {
	return AABB{Min: V(math.Inf(1), math.Inf(1), math.Inf(1)), Max: V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
}

func (b AABB) Contains(p Vec3) bool {
	const eps = 1e-9
	return p.X >= b.Min.X-eps && p.X <= b.Max.X+eps &&
		p.Y >= b.Min.Y-eps && p.Y <= b.Max.Y+eps &&
		p.Z >= b.Min.Z-eps && p.Z <= b.Max.Z+eps
}

func (b AABB) DistanceSquared(p Vec3) float64 {
	dx := math.Max(math.Max(b.Min.X-p.X, 0), p.X-b.Max.X)
	dy := math.Max(math.Max(b.Min.Y-p.Y, 0), p.Y-b.Max.Y)
	dz := math.Max(math.Max(b.Min.Z-p.Z, 0), p.Z-b.Max.Z)
	return dx*dx + dy*dy + dz*dz
}

func sphereIntersectsAABB(center Vec3, radius float64, box AABB) bool {
	return box.DistanceSquared(center) <= radius*radius
}

// rayBox returns the overlapping ray parameter interval, if any.
func rayBox(r Ray, box AABB) (float64, float64, bool) {
	tmin, tmax := 0.0, math.Inf(1)
	orig := [3]float64{r.Origin.X, r.Origin.Y, r.Origin.Z}
	dir := [3]float64{r.Direction.X, r.Direction.Y, r.Direction.Z}
	mn := [3]float64{box.Min.X, box.Min.Y, box.Min.Z}
	mx := [3]float64{box.Max.X, box.Max.Y, box.Max.Z}
	for i := 0; i < 3; i++ {
		if math.Abs(dir[i]) < 1e-12 {
			if orig[i] < mn[i] || orig[i] > mx[i] {
				return 0, 0, false
			}
			continue
		}
		inv := 1 / dir[i]
		t0 := (mn[i] - orig[i]) * inv
		t1 := (mx[i] - orig[i]) * inv
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		tmin = math.Max(tmin, t0)
		tmax = math.Min(tmax, t1)
		if tmin > tmax {
			return 0, 0, false
		}
	}
	return tmin, tmax, true
}

func reflect(d, n Vec3) Vec3 {
	return d.Sub(n.Mul(2 * d.Dot(n)))
}
