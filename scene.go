package main

import "math"

const (
	imageWidth  = 960
	imageHeight = 720
	textureSize = 1024
)

type Camera struct {
	Pos        Vec3
	Forward    Vec3
	Right      Vec3
	Up         Vec3
	TanHalfFov float64
	Aspect     float64
}

func NewCamera(pos, target Vec3, fovYDeg float64, aspect float64) Camera {
	forward := target.Sub(pos).Unit()
	worldUp := Vec3{0, 1, 0}
	right := cross3(worldUp, forward).Unit()
	up := cross3(forward, right).Unit()
	return Camera{
		Pos:        pos,
		Forward:    forward,
		Right:      right,
		Up:         up,
		TanHalfFov: math.Tan(fovYDeg * math.Pi / 360),
		Aspect:     aspect,
	}
}

type Scene struct {
	Width, Height int
	Camera        Camera
	MinX, MaxX    float64
	MinZ, MaxZ    float64
	TexelsPerUnit float64
}

func NewScene() *Scene {
	return &Scene{
		Width:         imageWidth,
		Height:        imageHeight,
		Camera:        NewCamera(Vec3{0, 3.0, -7.0}, Vec3{0, 0, 4.2}, 43, float64(imageWidth)/imageHeight),
		MinX:          -9,
		MaxX:          9,
		MinZ:          -4,
		MaxZ:          14,
		TexelsPerUnit: 56,
	}
}

func (s *Scene) primaryRay(x, y float64) Vec3 {
	ndcX := 2*(x+0.5)/float64(s.Width) - 1
	ndcY := 1 - 2*(y+0.5)/float64(s.Height)
	return s.Camera.Forward.
		Add(s.Camera.Right.Mul(ndcX * s.Camera.TanHalfFov * s.Camera.Aspect)).
		Add(s.Camera.Up.Mul(ndcY * s.Camera.TanHalfFov)).
		Unit()
}

type Hit struct {
	OK    bool
	World Vec3
	UV    Vec2
	T     float64
}

func (s *Scene) hit(x, y float64) Hit {
	dir := s.primaryRay(x, y)
	if dir.Y >= -1e-8 {
		return Hit{}
	}
	t := -s.Camera.Pos.Y / dir.Y
	if t <= 0 {
		return Hit{}
	}
	p := s.Camera.Pos.Add(dir.Mul(t))
	if p.X < s.MinX || p.X > s.MaxX || p.Z < s.MinZ || p.Z > s.MaxZ {
		return Hit{}
	}
	uv := Vec2{
		X: (p.X - s.MinX) * s.TexelsPerUnit,
		Y: (p.Z - s.MinZ) * s.TexelsPerUnit,
	}
	return Hit{OK: true, World: p, UV: uv, T: t}
}

func (s *Scene) project(p Vec3) (Vec2, bool) {
	q := p.Sub(s.Camera.Pos)
	z := q.Dot(s.Camera.Forward)
	if z <= 1e-6 {
		return Vec2{}, false
	}
	nx := q.Dot(s.Camera.Right) / (z * s.Camera.TanHalfFov * s.Camera.Aspect)
	ny := q.Dot(s.Camera.Up) / (z * s.Camera.TanHalfFov)
	return Vec2{
		X: (nx + 1) * 0.5 * float64(s.Width),
		Y: (1 - ny) * 0.5 * float64(s.Height),
	}, true
}

func (s *Scene) worldFromUV(uv Vec2) Vec3 {
	return Vec3{
		X: s.MinX + uv.X/s.TexelsPerUnit,
		Y: 0,
		Z: s.MinZ + uv.Y/s.TexelsPerUnit,
	}
}

type Footprint struct {
	Valid     bool
	Major     float64
	Minor     float64
	AxisMajor Vec2
	AxisMinor Vec2
	Dx        Vec2
	Dy        Vec2
}

func (s *Scene) footprint(x, y int, center Hit) Footprint {
	if !center.OK {
		return Footprint{}
	}
	uvAt := func(px, py int) (Vec2, bool) {
		h := s.hit(float64(px), float64(py))
		return h.UV, h.OK
	}

	var dx, dy Vec2
	l, lok := uvAt(x-1, y)
	r, rok := uvAt(x+1, y)
	switch {
	case lok && rok:
		dx = r.Sub(l).Mul(0.5)
	case rok:
		dx = r.Sub(center.UV)
	case lok:
		dx = center.UV.Sub(l)
	}

	u, uok := uvAt(x, y-1)
	d, dok := uvAt(x, y+1)
	switch {
	case uok && dok:
		dy = d.Sub(u).Mul(0.5)
	case dok:
		dy = d.Sub(center.UV)
	case uok:
		dy = center.UV.Sub(u)
	}

	if math.Abs(dx.X)+math.Abs(dx.Y) < 1e-10 {
		dx = Vec2{1, 0}
	}
	if math.Abs(dy.X)+math.Abs(dy.Y) < 1e-10 {
		dy = Vec2{0, 1}
	}
	dx = clampVec2(dx, 4096)
	dy = clampVec2(dy, 4096)
	return makeFootprint(dx, dy)
}

func makeFootprint(dx, dy Vec2) Footprint {
	xx := dx.X*dx.X + dy.X*dy.X
	yy := dx.Y*dx.Y + dy.Y*dy.Y
	xy := dx.X*dx.Y + dy.X*dy.Y
	disc := math.Hypot(xx-yy, 2*xy)
	l1 := 0.5 * (xx + yy + disc)
	l0 := math.Max(0, 0.5*(xx+yy-disc))
	theta := 0.5 * math.Atan2(2*xy, xx-yy)
	majorAxis := Vec2{math.Cos(theta), math.Sin(theta)}
	minorAxis := Vec2{-majorAxis.Y, majorAxis.X}
	return Footprint{
		Valid:     true,
		Major:     math.Sqrt(math.Max(l1, 0)),
		Minor:     math.Sqrt(math.Max(l0, 1e-10)),
		AxisMajor: majorAxis,
		AxisMinor: minorAxis,
		Dx:        dx,
		Dy:        dy,
	}
}

func clampVec2(v Vec2, maxLen float64) Vec2 {
	n := v.Len()
	if n > maxLen {
		return v.Mul(maxLen / n)
	}
	return v
}
