package skinsss

// Ellipsoid is an axis-aligned ellipsoid implicit surface.
type Ellipsoid struct {
	Center Vec3
	Radii  Vec3
}

// SDF evaluates an approximate signed distance field (exact on the surface,
// bounded-error in space, which is sufficient for sphere tracing).
func (e Ellipsoid) SDF(p Vec3) float64 {
	q := p.Sub(e.Center)
	k0 := Vec3{q.X / e.Radii.X, q.Y / e.Radii.Y, q.Z / e.Radii.Z}.Length() - 1
	k1 := Vec3{
		q.X / (e.Radii.X * e.Radii.X),
		q.Y / (e.Radii.Y * e.Radii.Y),
		q.Z / (e.Radii.Z * e.Radii.Z),
	}.Length()
	return k0 / k1
}

// Scene is the union of a thick head and two thin flattened ears.
type Scene struct {
	Head Ellipsoid
	Ears [2]Ellipsoid
}

// DefaultScene builds the head/ears used by the renderer.
func DefaultScene() Scene {
	head := Ellipsoid{
		Center: Vec3{0, 0, 0},
		Radii:  Vec3{0.85, 1.05, 0.75},
	}
	earR := Ellipsoid{
		Center: Vec3{0.84, 0.02, 0.02},
		Radii:  Vec3{0.12, 0.42, 0.26},
	}
	earL := earR
	earL.Center.X = -0.84
	return Scene{Head: head, Ears: [2]Ellipsoid{earR, earL}}
}

// SDF returns the union (minimum) distance of all primitives.
func (s Scene) SDF(p Vec3) float64 {
	d := s.Head.SDF(p)
	for _, e := range s.Ears {
		if ed := e.SDF(p); ed < d {
			d = ed
		}
	}
	return d
}

// Normal computes the unit surface normal via tetrahedron gradients.
func (s Scene) Normal(p Vec3) Vec3 {
	const eps = 1e-3
	e1 := Vec3{1, -1, -1}
	e2 := Vec3{-1, -1, 1}
	e3 := Vec3{-1, 1, -1}
	e4 := Vec3{1, 1, 1}
	n := e1.Scale(s.SDF(p.Add(e1.Scale(eps)))).
		Add(e2.Scale(s.SDF(p.Add(e2.Scale(eps))))).
		Add(e3.Scale(s.SDF(p.Add(e3.Scale(eps))))).
		Add(e4.Scale(s.SDF(p.Add(e4.Scale(eps)))))
	return n.Normalize()
}

// Curvature returns the mean curvature at p, estimated from the SDF Laplacian
// divided by the gradient magnitude. Convex outward regions are positive.
func (s Scene) Curvature(p Vec3) float64 {
	const eps = 0.02
	lap := (s.SDF(p.Add(Vec3{eps, 0, 0})) + s.SDF(p.Add(Vec3{-eps, 0, 0})) +
		s.SDF(p.Add(Vec3{0, eps, 0})) + s.SDF(p.Add(Vec3{0, -eps, 0})) +
		s.SDF(p.Add(Vec3{0, 0, eps})) + s.SDF(p.Add(Vec3{0, 0, -eps})) -
		6*s.SDF(p)) / (eps * eps)
	return -lap // Laplacian is negative on convex outward surfaces
}

// ChordThickness marches from a surface point along the photon travel
// direction d (d points away from the light). It first crosses into the
// volume then accumulates distance until the surface is exited, returning the
// interior chord length. A zero result means the ray never enters the volume.
func (s Scene) ChordThickness(p, d Vec3) float64 {
	const step = 0.008
	const maxSteps = 480

	q := p
	t := 0.0
	entered := false
	for i := 0; i < maxSteps; i++ {
		q = q.Add(d.Scale(step))
		t += step
		inside := s.SDF(q) < 0
		if !entered {
			if inside {
				entered = true
			}
			continue
		}
		if !inside {
			return t
		}
	}
	if !entered {
		return 0
	}
	return t
}
