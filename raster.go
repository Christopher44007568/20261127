package main

import (
	"image"
	"image/color"
	"math"
)

func viewport(p Rect) Rect {
	return Rect{X: p.X + 1, Y: p.Y + 52, W: p.W - 2, H: p.H - 178}
}

func renderMesh(img *image.RGBA, r Rect, cam Camera, mesh *Mesh, depth []float64, baseline bool) {
	base := mesh.Scene.Color
	if baseline {
		base = mixColor(base, [3]float64{0.95, 0.34, 0.28}, 0.22)
	}
	light1 := V(-0.42, 0.82, 0.48).Normalize()
	light2 := V(0.65, 0.35, -0.55).Normalize()

	for _, tri := range mesh.Tris {
		id0, id1, id2 := tri.V[0], tri.V[1], tri.V[2]
		p0w, p1w, p2w := mesh.Pos[id0], mesh.Pos[id1], mesh.Pos[id2]
		p0, ok0 := cam.Project(p0w)
		p1, ok1 := cam.Project(p1w)
		p2, ok2 := cam.Project(p2w)
		if !ok0 || !ok1 || !ok2 {
			continue
		}
		minX := int(math.Floor(math.Min(p0.X, math.Min(p1.X, p2.X))))
		maxX := int(math.Ceil(math.Max(p0.X, math.Max(p1.X, p2.X))))
		minY := int(math.Floor(math.Min(p0.Y, math.Min(p1.Y, p2.Y))))
		maxY := int(math.Ceil(math.Max(p0.Y, math.Max(p1.Y, p2.Y))))
		minX = clampInt(minX, r.X, r.X+r.W-1)
		maxX = clampInt(maxX, r.X, r.X+r.W-1)
		minY = clampInt(minY, r.Y, r.Y+r.H-1)
		maxY = clampInt(maxY, r.Y, r.Y+r.H-1)
		if minX > maxX || minY > maxY {
			continue
		}
		den := (p1.Y-p2.Y)*(p0.X-p2.X) + (p2.X-p1.X)*(p0.Y-p2.Y)
		if math.Abs(den) < 1e-9 {
			continue
		}

		faceN := p1w.Sub(p0w).Cross(p2w.Sub(p0w)).Normalize()
		for py := minY; py <= maxY; py++ {
			for px := minX; px <= maxX; px++ {
				sx := float64(px) + 0.5
				sy := float64(py) + 0.5
				w0 := ((p1.Y-p2.Y)*(sx-p2.X) + (p2.X-p1.X)*(sy-p2.Y)) / den
				w1 := ((p2.Y-p0.Y)*(sx-p2.X) + (p0.X-p2.X)*(sy-p2.Y)) / den
				w2 := 1 - w0 - w1
				if w0 < 0 || w1 < 0 || w2 < 0 {
					continue
				}
				invZ := w0*p0.W + w1*p1.W + w2*p2.W
				di := (py-r.Y)*r.W + (px - r.X)
				if invZ <= depth[di] {
					continue
				}
				world := perspectivePoint(w0, w1, w2, p0.W, p1.W, p2.W, p0w, p1w, p2w, invZ)
				if mesh.Scene.Clip && world.X+world.Z > 0.72 {
					continue
				}

				n := faceN
				if baseline || !mesh.Scene.Flat {
					n = mesh.Scene.Normal(world)
				}
				eyeDir := cam.Eye.Sub(world).Normalize()
				if n.Dot(eyeDir) < 0 {
					n = n.Mul(-1)
				}
				key := math.Max(n.Dot(light1), 0)
				fill := math.Max(n.Dot(light2), 0)
				head := math.Max(n.Dot(eyeDir), 0)
				shade := 0.30 + 0.56*key + 0.16*fill + 0.10*head
				col := [3]float64{
					clampFloat(base[0]*shade, 0, 1),
					clampFloat(base[1]*shade, 0, 1),
					clampFloat(base[2]*shade, 0, 1),
				}
				if !mesh.Scene.Flat {
					half := light1.Add(eyeDir).Normalize()
					spec := math.Pow(math.Max(n.Dot(half), 0), 34) * 0.28
					col[0] = clampFloat(col[0]+spec, 0, 1)
					col[1] = clampFloat(col[1]+spec, 0, 1)
					col[2] = clampFloat(col[2]+spec, 0, 1)
				}
				img.SetRGBA(px, py, rgbaFrom(col, 255))
				depth[di] = invZ
			}
		}
	}
}

func perspectivePoint(w0, w1, w2, iz0, iz1, iz2 float64, p0, p1, p2 Vec3, invZ float64) Vec3 {
	return p0.Mul(w0 * iz0).Add(p1.Mul(w1 * iz1)).Add(p2.Mul(w2 * iz2)).Div(invZ)
}

func mixColor(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{
		a[0]*(1-t) + b[0]*t,
		a[1]*(1-t) + b[1]*t,
		a[2]*(1-t) + b[2]*t,
	}
}

func rgbaFrom(c [3]float64, alpha uint8) color.RGBA {
	return color.RGBA{
		R: uint8(clampFloat(c[0]*255, 0, 255)),
		G: uint8(clampFloat(c[1]*255, 0, 255)),
		B: uint8(clampFloat(c[2]*255, 0, 255)),
		A: alpha,
	}
}
