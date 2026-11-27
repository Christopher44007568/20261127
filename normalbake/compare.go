package main

import (
	"image"
	"image/color"
	"math"
)

var lightDir = Vec3{0.5, 0.7, 1}.Norm()

// renderMesh rasterizes a mesh with an orthographic front camera. If baked is
// non-nil, the normal map is applied in tangent space; otherwise interpolated
// vertex normals are used directly.
func renderMesh(m *Mesh, baked *BakeResult, res int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, res, res))
	zbuf := make([]float64, res*res)
	for i := range zbuf {
		zbuf[i] = -1e30
	}
	bg := color.RGBA{30, 30, 34, 255}
	for i := 0; i < res*res; i++ {
		img.SetRGBA(i%res, i/res, bg)
	}
	const fit = 0.94 / 2 / 1.85 // torus radius ~1.85
	project := func(p Vec3) (float64, float64) {
		return (p.X*fit + 0.5) * float64(res), (0.5 - p.Y*fit) * float64(res)
	}
	for _, tri := range m.Tris {
		p0, p1, p2 := m.Pos[tri.I[0]], m.Pos[tri.I[1]], m.Pos[tri.I[2]]
		n0, n1, n2 := m.Nrm[tri.I[0]], m.Nrm[tri.I[1]], m.Nrm[tri.I[2]]
		x0, y0 := project(p0)
		x1, y1 := project(p1)
		x2, y2 := project(p2)
		det := (x1-x0)*(y2-y0) - (x2-x0)*(y1-y0)
		if math.Abs(det) < 1e-12 {
			continue
		}
		minX := int(math.Max(0, math.Floor(min3(x0, x1, x2))))
		maxX := int(math.Min(float64(res-1), math.Ceil(max3(x0, x1, x2))))
		minY := int(math.Max(0, math.Floor(min3(y0, y1, y2))))
		maxY := int(math.Min(float64(res-1), math.Ceil(max3(y0, y1, y2))))

		var Tt, Bt Vec3
		if baked != nil {
			uv0, uv1, uv2 := m.UV[tri.I[0]], m.UV[tri.I[1]], m.UV[tri.I[2]]
			e1, e2 := p1.Sub(p0), p2.Sub(p0)
			udet := (uv1.X-uv0.X)*(uv2.Y-uv0.Y) - (uv2.X-uv0.X)*(uv1.Y-uv0.Y)
			if math.Abs(udet) > 1e-14 {
				inv := 1 / udet
				Tt = e1.Mul((uv2.Y - uv0.Y) * inv).Sub(e2.Mul((uv1.Y - uv0.Y) * inv))
				Bt = e2.Mul((uv1.X - uv0.X) * inv).Sub(e1.Mul((uv2.X - uv0.X) * inv))
			}
		}
		for py := minY; py <= maxY; py++ {
			for px := minX; px <= maxX; px++ {
				cx, cy := float64(px)+0.5, float64(py)+0.5
				w0 := ((y1-y2)*(cx-x2) + (x2-x1)*(cy-y2)) / det
				w1 := ((y2-y0)*(cx-x2) + (x0-x2)*(cy-y2)) / det
				w2 := 1 - w0 - w1
				if w0 < 0 || w1 < 0 || w2 < 0 {
					continue
				}
				z := w0*p0.Z + w1*p1.Z + w2*p2.Z
				zi := py*res + px
				if z <= zbuf[zi] {
					continue
				}
				zbuf[zi] = z
				N := n0.Mul(w0).Add(n1.Mul(w1)).Add(n2.Mul(w2)).Norm()
				world := N
				if baked != nil {
					uv0, uv1, uv2 := m.UV[tri.I[0]], m.UV[tri.I[1]], m.UV[tri.I[2]]
					u := w0*uv0.X + w1*uv1.X + w2*uv2.X
					v := w0*uv0.Y + w1*uv1.Y + w2*uv2.Y
					ts := sampleMap(baked, u, v)
					T := Tt.Sub(N.Mul(Tt.Dot(N)))
					if T.Len() < 1e-8 {
						T = Vec3{1, 0, 0}
					}
					T = T.Norm()
					hand := 1.0
					if Tt.Cross(Bt).Dot(N) < 0 {
						hand = -1
					}
					B := N.Cross(T).Mul(hand)
					world = T.Mul(ts.X).Add(B.Mul(ts.Y)).Add(N.Mul(ts.Z)).Norm()
				}
				diff := math.Max(0, world.Dot(lightDir))
				h := lightDir.Add(Vec3{0, 0, 1}).Norm()
				spec := math.Pow(math.Max(0, world.Dot(h)), 32)
				lum := 0.12 + 0.88*diff
				img.SetRGBA(px, py, color.RGBA{
					uint8(clamp01(0.72*lum+0.3*spec) * 255),
					uint8(clamp01(0.74*lum+0.3*spec) * 255),
					uint8(clamp01(0.78*lum+0.3*spec) * 255),
					255,
				})
			}
		}
	}
	return img
}

// SaveCompare renders low-poly+baked map (left) next to the high-poly
// reference (right).
func SaveCompare(low, high *Mesh, baked *BakeResult, path string, res int) error {
	left := renderMesh(low, baked, res)
	right := renderMesh(high, nil, res)
	img := image.NewRGBA(image.Rect(0, 0, 2*res+1, res))
	for y := 0; y < res; y++ {
		for x := 0; x < res; x++ {
			img.SetRGBA(x, y, left.RGBAAt(x, y))
			img.SetRGBA(res+1+x, y, right.RGBAAt(x, y))
		}
		img.SetRGBA(res, y, color.RGBA{90, 90, 90, 255})
	}
	return writePNG(img, path)
}
