package main

import "math"

type Ray2 struct {
	O, D Vec2
}

type Hit2 struct {
	T      float64
	P      Vec2
	Normal Vec2
}

type Shape2 interface {
	IntersectRay(r Ray2) (Hit2, bool)
	Contains(p Vec2) bool
	Bounds() Rect
}

type Circle2 struct {
	C Vec2
	R float64
}

func (c Circle2) IntersectRay(r Ray2) (Hit2, bool) {
	oc := r.O.Sub(c.C)
	b := oc.Dot(r.D)
	cc := oc.Dot(oc) - c.R*c.R
	disc := b*b - cc
	if disc < 0 {
		return Hit2{}, false
	}
	s := math.Sqrt(disc)
	t := -b - s
	if t < 1e-5 {
		t = -b + s
	}
	if t < 1e-5 {
		return Hit2{}, false
	}
	p := r.O.Add(r.D.Mul(t))
	n := p.Sub(c.C).Norm()
	return Hit2{T: t, P: p, Normal: n}, true
}

func (c Circle2) Contains(p Vec2) bool {
	return p.Sub(c.C).Len() <= c.R+1e-7
}

func (c Circle2) Bounds() Rect {
	return Rect{Min: v2(c.C.X-c.R, c.C.Y-c.R), Max: v2(c.C.X+c.R, c.C.Y+c.R)}
}

type Polygon2 struct {
	Verts   []Vec2
	Normals []Vec2
}

func NewPolygon(verts []Vec2) Polygon2 {
	p := Polygon2{Verts: verts}
	n := len(verts)
	p.Normals = make([]Vec2, n)
	for i := 0; i < n; i++ {
		a := verts[i]
		b := verts[(i+1)%n]
		e := b.Sub(a)
		// Vertices are counter-clockwise; (e.Y, -e.X) is the outward normal.
		p.Normals[i] = Vec2{e.Y, -e.X}.Norm()
	}
	return p
}

func regularPolygon(c Vec2, radius float64, sides int, rotation float64) Polygon2 {
	verts := make([]Vec2, sides)
	for i := 0; i < sides; i++ {
		a := rotation + 2*math.Pi*float64(i)/float64(sides)
		verts[i] = c.Add(v2(math.Cos(a)*radius, math.Sin(a)*radius))
	}
	return NewPolygon(verts)
}

func rotatedRect(c Vec2, width, height, angle float64) Polygon2 {
	ax := v2(math.Cos(angle), math.Sin(angle)).Mul(width / 2)
	ay := v2(-math.Sin(angle), math.Cos(angle)).Mul(height / 2)
	return NewPolygon([]Vec2{
		c.Sub(ax).Sub(ay),
		c.Add(ax).Sub(ay),
		c.Add(ax).Add(ay),
		c.Sub(ax).Add(ay),
	})
}

func (p Polygon2) IntersectRay(r Ray2) (Hit2, bool) {
	best := Hit2{T: math.Inf(1)}
	found := false
	for i := 0; i < len(p.Verts); i++ {
		a := p.Verts[i]
		b := p.Verts[(i+1)%len(p.Verts)]
		e := b.Sub(a)
		den := r.D.Cross(e)
		if math.Abs(den) < 1e-12 {
			continue
		}
		ao := a.Sub(r.O)
		t := ao.Cross(e) / den
		u := ao.Cross(r.D) / den
		if t > 1e-5 && u >= -1e-7 && u <= 1+1e-7 && t < best.T {
			best = Hit2{T: t, P: r.O.Add(r.D.Mul(t)), Normal: p.Normals[i]}
			found = true
		}
	}
	return best, found
}

func (p Polygon2) Contains(q Vec2) bool {
	for i := 0; i < len(p.Verts); i++ {
		a := p.Verts[i]
		b := p.Verts[(i+1)%len(p.Verts)]
		e := b.Sub(a)
		if e.Cross(q.Sub(a)) < -1e-6 {
			return false
		}
	}
	return true
}

func (p Polygon2) Bounds() Rect {
	r := Rect{Min: p.Verts[0], Max: p.Verts[0]}
	for _, q := range p.Verts[1:] {
		r.Min.X = math.Min(r.Min.X, q.X)
		r.Min.Y = math.Min(r.Min.Y, q.Y)
		r.Max.X = math.Max(r.Max.X, q.X)
		r.Max.Y = math.Max(r.Max.Y, q.Y)
	}
	return r
}

func nearestShape(r Ray2, shapes []Shape2) (Hit2, Shape2, bool) {
	var best Hit2
	var shape Shape2
	found := false
	for _, s := range shapes {
		h, ok := s.IntersectRay(r)
		if ok && (!found || h.T < best.T) {
			best, shape, found = h, s, true
		}
	}
	return best, shape, found
}

func orientNormal(d, n Vec2) Vec2 {
	if d.Dot(n) > 0 {
		return n.Mul(-1)
	}
	return n
}

// Refract2 applies Snell's law. normal is automatically oriented toward the
// incident side. ok is false for total internal reflection.
func Refract2(d, normal Vec2, n1, n2 float64) (out Vec2, ok bool) {
	n := orientNormal(d, normal)
	cosI := clamp(-d.Dot(n), 0, 1)
	eta := n1 / n2
	sin2I := 1 - cosI*cosI
	k := 1 - eta*eta*sin2I
	if k < 0 {
		return Vec2{}, false
	}
	out = d.Mul(eta).Add(n.Mul(eta*cosI - math.Sqrt(k))).Norm()
	return out, true
}

func Reflect2(d, normal Vec2) Vec2 {
	n := orientNormal(d, normal)
	return d.Sub(n.Mul(2 * d.Dot(n))).Norm()
}

func chordLength(s Shape2, p, dir Vec2) float64 {
	d := dir.Norm()
	forward, fok := s.IntersectRay(Ray2{O: p, D: d})
	backward, bok := s.IntersectRay(Ray2{O: p, D: d.Mul(-1)})
	if !fok || !bok {
		return 0
	}
	return forward.T + backward.T
}

type BeamPath struct {
	Lambda float64
	Points []Vec2
}

func TraceBeam(start, dir Vec2, lambda float64, shapes []Shape2, bounds Rect) BeamPath {
	path := BeamPath{Lambda: lambda, Points: []Vec2{start}}
	r := Ray2{O: start, D: dir.Norm()}
	inside := false

	for bounce := 0; bounce < 10; bounce++ {
		h, _, ok := nearestShape(r, shapes)
		exitT := rayRectExit(r, bounds)
		if !ok || exitT < h.T {
			path.Points = append(path.Points, r.O.Add(r.D.Mul(exitT)))
			return path
		}

		path.Points = append(path.Points, h.P)
		nGlass := RefractiveIndex(lambda)
		n1, n2 := 1.0, nGlass
		if inside {
			n1, n2 = nGlass, 1.0
		}
		next, ok := Refract2(r.D, h.Normal, n1, n2)
		if !ok {
			next = Reflect2(r.D, h.Normal)
		} else {
			inside = !inside
		}
		r = Ray2{O: h.P.Add(next.Mul(1e-4)), D: next}
	}
	t := rayRectExit(r, bounds)
	path.Points = append(path.Points, r.O.Add(r.D.Mul(t)))
	return path
}

type Rect struct {
	Min, Max Vec2
}

func rayRectExit(r Ray2, b Rect) float64 {
	best := math.Inf(1)
	if math.Abs(r.D.X) > 1e-12 {
		for _, x := range []float64{b.Min.X, b.Max.X} {
			t := (x - r.O.X) / r.D.X
			if t > 1e-6 {
				py := r.O.Y + t*r.D.Y
				if py >= b.Min.Y-1e-6 && py <= b.Max.Y+1e-6 && t < best {
					best = t
				}
			}
		}
	}
	if math.Abs(r.D.Y) > 1e-12 {
		for _, y := range []float64{b.Min.Y, b.Max.Y} {
			t := (y - r.O.Y) / r.D.Y
			if t > 1e-6 {
				px := r.O.X + t*r.D.X
				if px >= b.Min.X-1e-6 && px <= b.Max.X+1e-6 && t < best {
					best = t
				}
			}
		}
	}
	if math.IsInf(best, 1) {
		return 20
	}
	return best
}
