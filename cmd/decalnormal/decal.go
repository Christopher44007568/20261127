package main

import "math"

const (
	patRings = iota
	patRib
	patBump
)

// Decal is a world-space projector placed on the surface. The projector
// frame is built from the surface tangent at its center, so its footprint
// keeps a constant world size regardless of the underlying curvature.
type Decal struct {
	Name     string
	Frame    Frame
	Radius   float64
	Kind     int
	Strength float64
	Tint     Vec3
}

func newDecal(name string, cx, cy, radius float64, kind int, strength float64, tint Vec3) Decal {
	return Decal{
		Name:     name,
		Frame:    frameAt(cx, cy),
		Radius:   radius,
		Kind:     kind,
		Strength: strength,
		Tint:     tint,
	}
}

// smoothstep is the canonical Hermite ramp.
func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
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

// patternHeight is a procedural tangent-space relief height in unit UV space.
// It is generated analytically, so the build has no external assets.
func patternHeight(kind int, u, v float64) float64 {
	r := math.Hypot(u, v)
	switch kind {
	case patRings:
		return 0.10 * math.Sin(7*2*math.Pi*r) * (1 - smoothstep(0.6, 1.0, r))
	case patRib:
		return 0.09 * math.Sin(10*math.Pi*u) * (1 - smoothstep(0.55, 1.0, r))
	case patBump:
		return 0.075 * math.Sin(6*math.Pi*u) * math.Cos(6*math.Pi*v) *
			(1 - smoothstep(0.5, 1.0, r))
	default:
		return 0
	}
}

// detailNormal computes a unit tangent-space normal from the relief height
// by finite differences. Z stays the unperturbed direction, which is what
// makes the detail reusable on any surface through reorientation.
func (d Decal) detailNormal(u, v float64) Vec3 {
	const eps = 1e-3
	h0 := patternHeight(d.Kind, u-eps, v)
	h1 := patternHeight(d.Kind, u+eps, v)
	du := (h1 - h0) / (2 * eps)
	h0 = patternHeight(d.Kind, u, v-eps)
	h1 = patternHeight(d.Kind, u, v+eps)
	dv := (h1 - h0) / (2 * eps)
	return v3(-du*d.Strength, -dv*d.Strength, 1).Normalized()
}

// Projection places a hit point inside the projector.
type Projection struct {
	U, V float64
	Mask float64
}

// project maps a world point onto the projector plane and returns unit UV
// coordinates plus a soft edge mask (full detail at 0.75 radius, zero at 1).
func (d Decal) project(p Vec3) Projection {
	rel := p.Sub(d.Frame.Origin)
	u := rel.Dot(d.Frame.T1) / d.Radius
	v := rel.Dot(d.Frame.T2) / d.Radius
	r := math.Hypot(u, v)
	mask := 1 - smoothstep(0.75, 1.0, r)
	return Projection{U: u, V: v, Mask: mask}
}

// reorient rotates a tangent-space detail normal so its "up" direction (Z)
// follows the base surface normal. It uses the shortest-arc quaternion
// equivalent of Reoriented Normal Mapping: unlike a linear average it
// preserves the detail's highlight direction, and for unit inputs the
// output is unit length before the final safety normalization.
func reorient(base, detail Vec3) Vec3 {
	axis := v3(-base.Y, base.X, 0)
	q := Vec3{axis.X, axis.Y, axis.Z}
	qw := 1 + base.Z
	// Rotate detail by quaternion (q, qw).
	t := q.Scale(2).Cross(detail)
	rotated := detail.Add(t.Scale(qw)).Add(q.Cross(t))
	return rotated.Normalized()
}

// blendDetail applies one decal to the running world-space normal.
// Masked regions keep the incoming normal untouched; inside the decal the
// detail is reoriented to the current surface normal then slerped in by the
// edge mask, with a final normalization guaranteeing |n| == 1 even where
// many decals overlap.
func blendDetail(current Vec3, d Decal, pr Projection) Vec3 {
	if pr.Mask <= 0 {
		return current
	}
	tn := d.detailNormal(pr.U, pr.V)
	oriented := reorient(current, tn)
	return Lerp(current, oriented, pr.Mask).Normalized()
}
