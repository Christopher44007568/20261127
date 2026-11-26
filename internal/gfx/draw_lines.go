package gfx

import (
	"image/color"
	"math"
)

// Polyline3D draws a thick line through world-space points. When xray is true
// a translucent through-surface pass is drawn first; the opaque core still
// respects the z-buffer with a small bias.
func (s *Scene) Polyline3D(pts []Vec3, closed bool, col color.RGBA, width float64, xray bool) {
	if len(pts) < 2 {
		return
	}
	if xray {
		s.polylinePass(pts, closed, col, width+3.5, 46, false, s.depthBias*4)
	}
	halo := color.RGBA{255, 255, 255, 255}
	s.polylinePass(pts, closed, halo, width+3.2, 215, true, s.depthBias+0.02)
	s.polylinePass(pts, closed, col, width, 255, true, s.depthBias)
}

// Dashed3D draws a dashed world-space line (used for bridge highlights).
func (s *Scene) Dashed3D(pts []Vec3, col color.RGBA, width float64, dashPx, gapPx float64) {
	if len(pts) < 2 {
		return
	}
	for i := 0; i < len(pts)-1; i++ {
		ax, ay, az := s.Project(pts[i])
		bx, by, bz := s.Project(pts[i+1])
		dx, dy := bx-ax, by-ay
		length := math.Hypot(dx, dy)
		if length < 1 {
			continue
		}
		for d := 0.0; d < length; d += dashPx + gapPx {
			d2 := math.Min(d+dashPx, length)
			t1, t2 := d/length, d2/length
			x1, y1 := ax+dx*t1, ay+dy*t1
			x2, y2 := ax+dx*t2, ay+dy*t2
			z1, z2 := az+(bz-az)*t1, az+(bz-az)*t2
			steps := int(math.Ceil(math.Max(math.Abs(x2-x1), math.Abs(y2-y1))))
			r := width / 2
			for k := 0; k <= steps; k++ {
				t := float64(k) / float64(max(steps, 1))
				s.stamp(x1+(x2-x1)*t, y1+(y2-y1)*t, z1+(z2-z1)*t, r, col, 255, true, s.depthBias)
			}
		}
	}
}

func (s *Scene) polylinePass(pts []Vec3, closed bool, col color.RGBA, width float64, alpha uint8, depth bool, bias float64) {
	r := width / 2
	segments := len(pts) - 1
	if closed {
		segments++
	}
	for i := 0; i < segments; i++ {
		a := pts[i%len(pts)]
		b := pts[(i+1)%len(pts)]
		ax, ay, az := s.Project(a)
		bx, by, bz := s.Project(b)
		dx, dy := bx-ax, by-ay
		steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
		if steps < 1 {
			steps = 1
		}
		for k := 0; k <= steps; k++ {
			t := float64(k) / float64(steps)
			x := ax + dx*t
			y := ay + dy*t
			z := az + (bz-az)*t
			s.stamp(x, y, z, r, col, alpha, depth, bias)
		}
	}
}

func (s *Scene) stamp(cx, cy, cz, r float64, col color.RGBA, alpha uint8, depth bool, bias float64) {
	ir := int(math.Ceil(r))
	for oy := -ir; oy <= ir; oy++ {
		for ox := -ir; ox <= ir; ox++ {
			if float64(ox*ox+oy*oy) > (r+0.3)*(r+0.3) {
				continue
			}
			x := int(math.Round(cx)) + ox
			y := int(math.Round(cy)) + oy
			if x < 0 || y < 0 || x >= s.W || y >= s.H {
				continue
			}
			idx := y*s.W + x
			if depth && cz > s.Z[idx]+bias {
				continue
			}
			if alpha >= 255 {
				off := s.Img.PixOffset(x, y)
				s.Img.Pix[off] = col.R
				s.Img.Pix[off+1] = col.G
				s.Img.Pix[off+2] = col.B
				s.Img.Pix[off+3] = 255
			} else {
				s.blendPixel(x, y, col, alpha)
			}
		}
	}
}

// MarkerKind selects a 2D marker shape.
type MarkerKind int

const (
	MarkerDot MarkerKind = iota
	MarkerRing
	MarkerCross
	MarkerStar
	MarkerSquare
	MarkerDiamond
)

// Marker3D projects p and draws a high-contrast marker with a white halo.
func (s *Scene) Marker3D(p Vec3, kind MarkerKind, r float64, col color.RGBA) {
	x, y, _ := s.Project(p)
	s.Marker2D(int(math.Round(x)), int(math.Round(y)), kind, r, col)
}

// Marker2D draws a marker in pixel coordinates.
func (s *Scene) Marker2D(cx, cy int, kind MarkerKind, r float64, col color.RGBA) {
	white := color.RGBA{255, 255, 255, 255}
	switch kind {
	case MarkerDot:
		s.disc(cx, cy, r+2.2, white)
		s.disc(cx, cy, r, col)
	case MarkerRing:
		s.ring(cx, cy, r+1.6, 3.4, white)
		s.ring(cx, cy, r, 2.0, col)
	case MarkerSquare:
		s.box(cx, cy, r+2, white)
		s.box(cx, cy, r, col)
	case MarkerDiamond:
		s.diamond(cx, cy, r+2, white)
		s.diamond(cx, cy, r, col)
	case MarkerCross:
		s.cross(cx, cy, r+1.5, 4.2, white)
		s.cross(cx, cy, r, 2.4, col)
	case MarkerStar:
		s.star(cx, cy, r+1.5, 4.2, white)
		s.star(cx, cy, r, 2.2, col)
	}
}

func (s *Scene) disc(cx, cy int, r float64, c color.RGBA) {
	ir := int(math.Ceil(r))
	for y := -ir; y <= ir; y++ {
		for x := -ir; x <= ir; x++ {
			if float64(x*x+y*y) <= r*r {
				s.blendPixel(cx+x, cy+y, c, 255)
			}
		}
	}
}

func (s *Scene) ring(cx, cy int, r, thick float64, c color.RGBA) {
	ir := int(math.Ceil(r + thick))
	for y := -ir; y <= ir; y++ {
		for x := -ir; x <= ir; x++ {
			d := math.Hypot(float64(x), float64(y))
			if math.Abs(d-r) <= thick*0.5 {
				s.blendPixel(cx+x, cy+y, c, 255)
			}
		}
	}
}

func (s *Scene) box(cx, cy int, r float64, c color.RGBA) {
	ir := int(math.Ceil(r))
	for y := -ir; y <= ir; y++ {
		for x := -ir; x <= ir; x++ {
			if math.Max(math.Abs(float64(x)), math.Abs(float64(y))) <= r {
				s.blendPixel(cx+x, cy+y, c, 255)
			}
		}
	}
}

func (s *Scene) diamond(cx, cy int, r float64, c color.RGBA) {
	ir := int(math.Ceil(r))
	for y := -ir; y <= ir; y++ {
		for x := -ir; x <= ir; x++ {
			if math.Abs(float64(x))+math.Abs(float64(y)) <= r {
				s.blendPixel(cx+x, cy+y, c, 255)
			}
		}
	}
}

func (s *Scene) cross(cx, cy int, r, thick float64, c color.RGBA) {
	ir := int(math.Ceil(r))
	it := int(math.Ceil(thick / 2))
	for d := -ir; d <= ir; d++ {
		for t := -it; t <= it; t++ {
			s.blendPixel(cx+d, cy+t, c, 255)
			s.blendPixel(cx+t, cy+d, c, 255)
		}
	}
}

func (s *Scene) star(cx, cy int, r, thick float64, c color.RGBA) {
	s.cross(cx, cy, r, thick, c)
	// Diagonal strokes via small line stamps.
	for k := -int(r); k <= int(r); k++ {
		for t := -1; t <= 1; t++ {
			s.blendPixel(cx+k, cy+k+t, c, 255)
			s.blendPixel(cx+k, cy-k+t, c, 255)
		}
	}
}

// Line2D draws a simple 2D line with thickness.
func (s *Scene) Line2D(x0, y0, x1, y1 int, c color.RGBA, thick int) {
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
	if steps == 0 {
		return
	}
	r := float64(thick) / 2
	for k := 0; k <= steps; k++ {
		t := float64(k) / float64(steps)
		s.disc(int(float64(x0)+dx*t), int(float64(y0)+dy*t), r, c)
	}
}
