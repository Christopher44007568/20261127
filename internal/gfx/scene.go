// Package gfx is a dependency-free software renderer: an orthographic
// z-buffer mesh view, thick depth-tested polylines, markers and 2D diagrams.
package gfx

import (
	"image"
	"image/color"
	"math"

	"featurelines/internal/linalg"
)

// Vec3 alias.
type Vec3 = linalg.Vec3

// Rect is an integer pixel rectangle.
type Rect struct{ X, Y, W, H int }

// Scene is an RGBA canvas with a depth buffer and orbit camera.
type Scene struct {
	Img *image.RGBA
	Z   []float64
	W   int
	H   int

	view       Rect
	center     Vec3
	scale      float64
	yaw, pitch float64
	right, up  Vec3
	fwd        Vec3
	depthBias  float64
	fitRadius  float64
	cxPx, cyPx float64
}

// NewScene allocates a WxH scene cleared to bg.
func NewScene(w, h int, bg color.RGBA) *Scene {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	s := &Scene{
		Img: img, W: w, H: h,
		Z:    make([]float64, w*h),
		view: Rect{0, 0, w, h},
	}
	s.Clear(bg)
	return s
}

// SetView reserves a sub-rectangle for 3D projection (leaves header space).
func (s *Scene) SetView(r Rect) { s.view = r }

// Clear fills color and resets the depth buffer.
func (s *Scene) Clear(bg color.RGBA) {
	for i := range s.Img.Pix {
		s.Img.Pix[i] = 0
	}
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			s.Img.SetRGBA(x, y, bg)
		}
	}
	for i := range s.Z {
		s.Z[i] = math.Inf(1)
	}
}

// Fit positions the orbit camera and computes a scale that frames the bounding
// box projection inside the viewport. topPad/bottomPad reserve UI bands.
func (s *Scene) Fit(lo, hi Vec3, yaw, pitch, topPad, bottomPad float64) {
	center := lo.Add(hi).Scale(0.5)
	s.center = center
	s.yaw, s.pitch = yaw, pitch
	cp := math.Cos(pitch)
	edir := Vec3{X: cp * math.Cos(yaw), Y: cp * math.Sin(yaw), Z: math.Sin(pitch)}
	s.fwd = edir.Scale(-1)
	worldUp := Vec3{X: 0, Y: 0, Z: 1}
	s.right = s.fwd.Cross(worldUp).Normalized()
	if s.right.Len2() < 1e-9 {
		s.right = Vec3{X: 1, Y: 0, Z: 0}
	}
	s.up = s.right.Cross(s.fwd).Normalized()

	vh := float64(s.view.H) - topPad - bottomPad
	if vh < 40 {
		vh = float64(s.view.H)
	}
	vw := float64(s.view.W)

	// Project the eight bounding-box corners in world units to find the
	// actual on-screen extents for this camera angle.
	var ex, ey float64
	for _, cx := range []float64{lo.X, hi.X} {
		for _, cy := range []float64{lo.Y, hi.Y} {
			for _, cz := range []float64{lo.Z, hi.Z} {
				rel := Vec3{X: cx - center.X, Y: cy - center.Y, Z: cz - center.Z}
				ex = math.Max(ex, math.Abs(rel.Dot(s.right)))
				ey = math.Max(ey, math.Abs(rel.Dot(s.up)))
			}
		}
	}
	ex = math.Max(ex, 1e-6)
	ey = math.Max(ey, 1e-6)
	s.scale = math.Min(vw*0.9/(2*ex), vh*0.9/(2*ey))
	diag := lo.Dist(hi)
	s.depthBias = diag * 0.004
	s.fitRadius = diag * 0.5
	s.cxPx = float64(s.view.X) + float64(s.view.W)*0.5
	s.cyPx = float64(s.view.Y) + topPad + vh*0.5
}

// Project maps a world point to pixel coordinates and view depth.
func (s *Scene) Project(p Vec3) (sx, sy, depth float64) {
	rel := p.Sub(s.center)
	vx := rel.Dot(s.right)
	vy := rel.Dot(s.up)
	vz := rel.Dot(s.fwd)
	sx = s.cxPx + vx*s.scale
	sy = s.cyPx - vy*s.scale
	return sx, sy, vz
}

func (s *Scene) setPixel(x, y int, z float64, c color.RGBA) {
	if x < s.view.X || y < s.view.Y || x >= s.view.X+s.view.W || y >= s.view.Y+s.view.H {
		return
	}
	idx := y*s.W + x
	if z < s.Z[idx] {
		s.Z[idx] = z
		s.Img.SetRGBA(x, y, c)
	}
}

// blendPixel alpha-blends without touching the depth buffer.
func (s *Scene) blendPixel(x, y int, c color.RGBA, alpha uint8) {
	if x < 0 || y < 0 || x >= s.W || y >= s.H {
		return
	}
	ai := uint32(alpha)
	i := s.Img.PixOffset(x, y)
	dr, dg, db := uint32(s.Img.Pix[i]), uint32(s.Img.Pix[i+1]), uint32(s.Img.Pix[i+2])
	r := (uint32(c.R)*ai + dr*(255-ai)) / 255
	g := (uint32(c.G)*ai + dg*(255-ai)) / 255
	b := (uint32(c.B)*ai + db*(255-ai)) / 255
	s.Img.Pix[i] = uint8(r)
	s.Img.Pix[i+1] = uint8(g)
	s.Img.Pix[i+2] = uint8(b)
	s.Img.Pix[i+3] = 255
}
