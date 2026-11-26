package gfx

import (
	"image/color"
	"math"

	"featurelines/internal/mesh"
)

// DrawMesh renders m with smooth Gouraud shading and a z-buffer.
func (s *Scene) DrawMesh(m *mesh.Mesh, base color.RGBA) {
	light := Vec3{X: 0.45, Y: -0.35, Z: 0.82}.Normalized()
	ambient := 0.42

	type pv struct {
		sx, sy, z float64
		r, g, b   float64
	}
	verts := make([]pv, len(m.V))
	for i, p := range m.V {
		sx, sy, z := s.Project(p)
		n := m.VN[i]
		diff := math.Abs(n.Dot(light))
		inten := ambient + 0.62*diff
		if inten > 1.15 {
			inten = 1.15
		}
		verts[i] = pv{sx, sy, z,
			float64(base.R) / 255 * inten,
			float64(base.G) / 255 * inten,
			float64(base.B) / 255 * inten}
	}

	for _, t := range m.T {
		p0, p1, p2 := verts[t[0]], verts[t[1]], verts[t[2]]
		minX := int(math.Floor(math.Min(p0.sx, math.Min(p1.sx, p2.sx))))
		maxX := int(math.Ceil(math.Max(p0.sx, math.Max(p1.sx, p2.sx))))
		minY := int(math.Floor(math.Min(p0.sy, math.Min(p1.sy, p2.sy))))
		maxY := int(math.Ceil(math.Max(p0.sy, math.Max(p1.sy, p2.sy))))
		minX = max(minX, s.view.X)
		minY = max(minY, s.view.Y)
		maxX = min(maxX, s.view.X+s.view.W-1)
		maxY = min(maxY, s.view.Y+s.view.H-1)
		if minX > maxX || minY > maxY {
			continue
		}
		area := edge(p0.sx, p0.sy, p1.sx, p1.sy, p2.sx, p2.sy)
		if math.Abs(area) < 1e-9 {
			continue
		}
		invArea := 1 / area

		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				px := float64(x) + 0.5
				py := float64(y) + 0.5
				w0 := edge(p1.sx, p1.sy, p2.sx, p2.sy, px, py) * invArea
				w1 := edge(p2.sx, p2.sy, p0.sx, p0.sy, px, py) * invArea
				w2 := 1 - w0 - w1
				if w0 < 0 || w1 < 0 || w2 < 0 {
					if !(w0 > -1e-6 && w1 > -1e-6 && w2 > -1e-6) {
						continue
					}
				}
				z := w0*p0.z + w1*p1.z + w2*p2.z
				idx := y*s.W + x
				if z >= s.Z[idx] {
					continue
				}
				s.Z[idx] = z
				rr := clampByte((w0*p0.r + w1*p1.r + w2*p2.r) * 255)
				gg := clampByte((w0*p0.g + w1*p1.g + w2*p2.g) * 255)
				bb := clampByte((w0*p0.b + w1*p1.b + w2*p2.b) * 255)
				off := s.Img.PixOffset(x, y)
				s.Img.Pix[off] = rr
				s.Img.Pix[off+1] = gg
				s.Img.Pix[off+2] = bb
				s.Img.Pix[off+3] = 255
			}
		}
	}
}

// edge is the signed edge function for barycentric coordinates.
func edge(ax, ay, bx, by, px, py float64) float64 {
	return (px-ax)*(by-ay) - (py-ay)*(bx-ax)
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
