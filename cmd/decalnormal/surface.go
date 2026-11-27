package main

import "math"

// Frame is an orthonormal tangent frame at a surface point.
// T1 and T2 span the tangent plane; N is the outward normal.
type Frame struct {
	Origin Vec3
	T1, T2 Vec3
	N      Vec3
}

// gaussian is a smooth bump and its partial derivatives.
// g = exp(-r2 / (2 s2)); d/dx g = g * (cx-x) / s2.
func gaussian(x, y, cx, cy, s float64) (v, dx, dy float64) {
	s2 := s * s
	r2 := (x-cx)*(x-cx) + (y-cy)*(y-cy)
	v = math.Exp(-r2 / (2 * s2))
	dx = v * (cx - x) / s2
	dy = v * (cy - y) / s2
	return
}

// sampleHeight evaluates the heightfield z = h(x, y) together with its
// gradient. The surface mixes broad and sharp features so decals can be
// moved across regions of clearly different curvature.
func sampleHeight(x, y float64) (h, fx, fy float64) {
	g1, g1x, g1y := gaussian(x, y, 0, 0, 1.7)
	g2, g2x, g2y := gaussian(x, y, -2.7, 1.3, 0.65)
	g3, g3x, g3y := gaussian(x, y, 2.8, -1.4, 0.95)
	h = 0.6*g1 + 0.38*g2 - 0.42*g3 + 0.07*math.Sin(1.3*x)*math.Cos(1.1*y)
	fx = 0.6*g1x + 0.38*g2x - 0.42*g3x + 0.07*1.3*math.Cos(1.3*x)*math.Cos(1.1*y)
	fy = 0.6*g1y + 0.38*g2y - 0.42*g3y - 0.07*1.1*math.Sin(1.3*x)*math.Sin(1.1*y)
	return
}

// frameAt builds an orthonormal tangent frame at (x, y, h(x, y)).
func frameAt(x, y float64) Frame {
	h, fx, fy := sampleHeight(x, y)
	n := v3(-fx, -fy, 1).Normalized()
	t1 := v3(1, 0, fx).Normalized()
	t2 := n.Cross(t1).Normalized()
	return Frame{Origin: v3(x, y, h), T1: t1, T2: t2, N: n}
}

// castRay intersects the heightfield with a ray. It brackets the crossing
// along the ray and refines it with bisection, which is robust for the
// steep slopes used here and needs no external solver.
func castRay(origin, dir Vec3) (Vec3, bool) {
	const (
		zMin   = -1.3
		zMax   = 1.4
		steps  = 96
		refine = 44
	)
	if dir.Z >= 0 {
		return Vec3{}, false
	}
	tForZ := func(z float64) float64 { return (z - origin.Z) / dir.Z }
	tLo, tHi := tForZ(zMax), tForZ(zMin)
	if tHi < tLo {
		tLo, tHi = tHi, tLo
	}
	gap := func(t float64) float64 {
		p := origin.Add(dir.Scale(t))
		h, _, _ := sampleHeight(p.X, p.Y)
		return p.Z - h
	}
	prevT := tLo
	prevG := gap(prevT)
	hit := false
	for i := 1; i <= steps; i++ {
		t := tLo + (tHi-tLo)*float64(i)/steps
		g := gap(t)
		if prevG == 0 || (prevG > 0) != (g > 0) {
			tLo, tHi = prevT, t
			hit = true
			break
		}
		prevT, prevG = t, g
	}
	if !hit {
		return Vec3{}, false
	}
	for i := 0; i < refine; i++ {
		mid := (tLo + tHi) * 0.5
		if gap(mid) > 0 {
			tLo = mid
		} else {
			tHi = mid
		}
	}
	return origin.Add(dir.Scale((tLo + tHi) * 0.5)), true
}
