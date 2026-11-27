package main

import "math"

type Ray struct {
	Orig, Dir Vec3
}

func (r Ray) at(t float64) Vec3 { return r.Orig.Add(r.Dir.Scale(t)) }

type Material struct {
	Params
	// Brush amplitude/frequency; zero means a smooth metal.
	BrushAmp  float64
	BrushFreq float64
	BrushAxis int // 0 = vary along tangent, 1 = vary along bitangent
	AngleRad  float64
}

type Hit struct {
	T      float64
	Point  Vec3
	Normal Vec3
	Mat    *Material
	Self   Object
}

type Object interface {
	intersect(r Ray) (Hit, bool)
	material() *Material
}

// Sphere is analytically intersected.
type Sphere struct {
	Center Vec3
	Radius float64
	Mat    *Material
}

func (s *Sphere) material() *Material { return s.Mat }

func (s *Sphere) intersect(r Ray) (Hit, bool) {
	oc := r.Orig.Sub(s.Center)
	b := oc.Dot(r.Dir)
	c := oc.Len2() - s.Radius*s.Radius
	disc := b*b - c
	if disc < 0 {
		return Hit{}, false
	}
	sqrtDisc := math.Sqrt(disc)
	t := -b - sqrtDisc
	if t < 1e-4 {
		t = -b + sqrtDisc
	}
	if t < 1e-4 {
		return Hit{}, false
	}
	p := r.at(t)
	n := p.Sub(s.Center).Scale(1 / s.Radius)
	return Hit{T: t, Point: p, Normal: n, Mat: s.Mat, Self: s}, true
}

// Plate is a rounded box evaluated by sphere tracing its SDF.
type Plate struct {
	Center Vec3
	Half   Vec3
	Radius float64
	Mat    *Material
}

func (p *Plate) material() *Material { return p.Mat }

func sdRoundBox(x, h Vec3, radius float64) float64 {
	q := Vec3{math.Abs(x.X) - h.X, math.Abs(x.Y) - h.Y, math.Abs(x.Z) - h.Z}
	qc := Vec3{math.Max(q.X, 0), math.Max(q.Y, 0), math.Max(q.Z, 0)}
	return qc.Len() + math.Min(math.Max(q.X, math.Max(q.Y, q.Z)), 0) - radius
}

func (p *Plate) intersect(r Ray) (Hit, bool) {
	t := 0.02
	const maxT = 60
	const steps = 96
	for i := 0; i < steps; i++ {
		pos := r.at(t)
		local := pos.Sub(p.Center)
		d := sdRoundBox(local, p.Half, p.Radius)
		if d < 1e-4 {
			n := p.normalAt(pos)
			return Hit{T: t, Point: pos, Normal: n, Mat: p.Mat, Self: p}, true
		}
		t += math.Max(d*0.9, 1e-4)
		if t > maxT {
			break
		}
	}
	return Hit{}, false
}

func (p *Plate) normalAt(pos Vec3) Vec3 {
	const eps = 1e-4
	d := func(dx, dy, dz float64) float64 {
		q := pos.Add(Vec3{dx, dy, dz}).Sub(p.Center)
		return sdRoundBox(q, p.Half, p.Radius)
	}
	n := Vec3{
		d(eps, 0, 0) - d(-eps, 0, 0),
		d(0, eps, 0) - d(0, -eps, 0),
		d(0, 0, eps) - d(0, 0, -eps),
	}
	return n.Normalize()
}

// surfaceFrame builds the tangent frame at a hit, applying the material's
// fiber rotation. T follows the fiber direction.
func surfaceFrame(h Hit) Frame {
	n := h.Normal.Normalize()
	return makeFrame(n, h.Mat.AngleRad)
}

// effectiveRoughness adds deterministic brushed-metal micro-roughness.
func effectiveRoughness(h Hit, f Frame, base float64) float64 {
	mat := h.Mat
	if mat.BrushAmp <= 0 {
		return base
	}
	var coord float64
	if mat.BrushAxis == 0 {
		coord = h.Point.Dot(f.T)
	} else {
		coord = h.Point.Dot(f.B)
	}
	// Two octaves of value noise in world units give long fine streaks.
	n := fbm(coord*mat.BrushFreq)*0.7 + fbm(coord*mat.BrushFreq*3.7)*0.3
	return clampf(base*(1+mat.BrushAmp*(n-0.5)*2), 0.045, 1)
}

type Scene struct {
	Objects []Object
}

func (s *Scene) trace(r Ray) (Hit, bool) {
	best := Hit{T: math.Inf(1)}
	found := false
	for _, obj := range s.Objects {
		if h, ok := obj.intersect(r); ok && h.T < best.T {
			best = h
			found = true
		}
	}
	return best, found
}
