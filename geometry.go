package main

import "math"

type Ray struct {
	Origin Vec3
	Dir    Vec3
}

type Box struct {
	Name   string
	Center Vec3
	Half   Vec3
	Color  RGB
}

func (b Box) Min() Vec3 {
	return b.Center.Sub(b.Half)
}

func (b Box) Max() Vec3 {
	return b.Center.Add(b.Half)
}

// SDF returns the exact Euclidean signed distance to the box:
// negative inside, zero on the surface, positive outside.
func (b Box) SDF(p Vec3) float64 {
	q := Vec3{
		X: math.Abs(p.X-b.Center.X) - b.Half.X,
		Y: math.Abs(p.Y-b.Center.Y) - b.Half.Y,
		Z: math.Abs(p.Z-b.Center.Z) - b.Half.Z,
	}
	out := Vec3{
		X: math.Max(q.X, 0),
		Y: math.Max(q.Y, 0),
		Z: math.Max(q.Z, 0),
	}
	outLen := out.Len()
	inside := math.Min(math.Max(q.X, math.Max(q.Y, q.Z)), 0)
	return outLen + inside
}

func (b Box) Intersect(r Ray) (float64, Vec3, bool) {
	lo := b.Min()
	hi := b.Max()
	tMin := -math.MaxFloat64
	tMax := math.MaxFloat64
	normal := Vec3{}

	axes := [3]struct {
		o, d, mn, mx float64
		n            Vec3
	}{
		{r.Origin.X, r.Dir.X, lo.X, hi.X, V3(-1, 0, 0)},
		{r.Origin.Y, r.Dir.Y, lo.Y, hi.Y, V3(0, -1, 0)},
		{r.Origin.Z, r.Dir.Z, lo.Z, hi.Z, V3(0, 0, -1)},
	}

	for _, ax := range axes {
		if math.Abs(ax.d) < 1e-12 {
			if ax.o < ax.mn || ax.o > ax.mx {
				return 0, Vec3{}, false
			}
			continue
		}
		t1 := (ax.mn - ax.o) / ax.d
		t2 := (ax.mx - ax.o) / ax.d
		axisNormal := ax.n
		if t1 > t2 {
			t1, t2 = t2, t1
			axisNormal = axisNormal.Mul(-1)
		}
		if t1 > tMin {
			tMin = t1
			normal = axisNormal
		}
		tMax = math.Min(tMax, t2)
		if tMin > tMax {
			return 0, Vec3{}, false
		}
	}

	if tMin < 1e-5 {
		if tMax < 1e-5 {
			return 0, Vec3{}, false
		}
		tMin = tMax
	}
	p := r.Origin.Add(r.Dir.Mul(tMin))
	if normal.Len() == 0 {
		normal = b.NormalAt(p)
	}
	return tMin, normal.Norm(), true
}

func (b Box) NormalAt(p Vec3) Vec3 {
	d := []float64{
		math.Abs((p.X - b.Center.X) / b.Half.X),
		math.Abs((p.Y - b.Center.Y) / b.Half.Y),
		math.Abs((p.Z - b.Center.Z) / b.Half.Z),
	}
	switch {
	case d[0] >= d[1] && d[0] >= d[2]:
		if p.X > b.Center.X {
			return V3(1, 0, 0)
		}
		return V3(-1, 0, 0)
	case d[1] >= d[2]:
		if p.Y > b.Center.Y {
			return V3(0, 1, 0)
		}
		return V3(0, -1, 0)
	default:
		if p.Z > b.Center.Z {
			return V3(0, 0, 1)
		}
		return V3(0, 0, -1)
	}
}

type Geometry []Box

func (g Geometry) SDF(p Vec3) float64 {
	best := math.MaxFloat64
	for _, b := range g {
		if d := b.SDF(p); d < best {
			best = d
		}
	}
	return best
}

func (g Geometry) Intersect(r Ray) (t float64, normal Vec3, box *Box, ok bool) {
	best := math.MaxFloat64
	for i := range g {
		hitT, n, hit := g[i].Intersect(r)
		if hit && hitT < best {
			best = hitT
			normal = n
			box = &g[i]
			ok = true
		}
	}
	return best, normal, box, ok
}
