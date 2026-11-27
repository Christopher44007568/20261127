package main

import "math"

type SceneKind int

const (
	KindSphere SceneKind = iota
	KindBox
	KindShell
)

type Scene struct {
	ID        string
	Title     string
	Subtitle  string
	Kind      SceneKind
	Radius    float64
	Half      Vec3
	Thickness float64
	MaxDepth  int
	MinDepth  int
	Color     [3]float64
	Flat      bool
	Clip      bool
}

func (s Scene) SDF(p Vec3) float64 {
	switch s.Kind {
	case KindSphere:
		return p.Norm() - s.Radius
	case KindBox:
		return sdBox(p, s.Half)
	case KindShell:
		inner := s.Half.Sub(V(s.Thickness, s.Thickness, s.Thickness))
		outer := sdBox(p, s.Half)
		cavity := sdBox(p, inner)
		return math.Max(outer, -cavity)
	default:
		return 0
	}
}

func (s Scene) Normal(p Vec3) Vec3 {
	const eps = 1.5e-4
	ex := V(eps, 0, 0)
	ey := V(0, eps, 0)
	ez := V(0, 0, eps)
	dx := s.SDF(p.Add(ex)) - s.SDF(p.Sub(ex))
	dy := s.SDF(p.Add(ey)) - s.SDF(p.Sub(ey))
	dz := s.SDF(p.Add(ez)) - s.SDF(p.Sub(ez))
	return V(dx, dy, dz).Normalize()
}

// FeatureDistance returns distance to the ground-truth crease network.
// Sphere has no creases and returns +Inf.
func (s Scene) FeatureDistance(p Vec3) float64 {
	switch s.Kind {
	case KindBox:
		return distanceToBoxEdges(p, s.Half)
	case KindShell:
		inner := s.Half.Sub(V(s.Thickness, s.Thickness, s.Thickness))
		return math.Min(distanceToBoxEdges(p, s.Half), distanceToBoxEdges(p, inner))
	default:
		return math.Inf(1)
	}
}

func sdBox(p, b Vec3) float64 {
	q := p.Abs().Sub(b)
	out := V(math.Max(q.X, 0), math.Max(q.Y, 0), math.Max(q.Z, 0))
	outside := out.Norm()
	inside := math.Min(math.Max(q.X, math.Max(q.Y, q.Z)), 0)
	return outside + inside
}

func distanceToBoxEdges(p, b Vec3) float64 {
	best := math.Inf(1)
	// Edges parallel to each axis: the other two coordinates are at ±b.
	for axis := 0; axis < 3; axis++ {
		a1 := (axis + 1) % 3
		a2 := (axis + 2) % 3
		for _, s1 := range []float64{-1, 1} {
			for _, s2 := range []float64{-1, 1} {
				q := p
				setCoord := func(i int, v float64) {
					switch i {
					case 0:
						q.X = v
					case 1:
						q.Y = v
					case 2:
						q.Z = v
					}
				}
				getCoord := func(i int) float64 {
					switch i {
					case 0:
						return q.X
					case 1:
						return q.Y
					default:
						return q.Z
					}
				}
				setCoord(a1, s1*coord(b, a1))
				setCoord(a2, s2*coord(b, a2))
				lo := -coord(b, axis)
				hi := coord(b, axis)
				av := clampFloat(getCoord(axis), lo, hi)
				setCoord(axis, av)
				best = math.Min(best, p.Sub(q).Norm())
			}
		}
	}
	return best
}

func coord(v Vec3, axis int) float64 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		return v.Z
	}
}

func setCoordPtr(v *Vec3, axis int, x float64) {
	switch axis {
	case 0:
		v.X = x
	case 1:
		v.Y = x
	default:
		v.Z = x
	}
}
