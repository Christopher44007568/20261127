package main

import "math"

func (c *Canvas) DrawPoint3D(cam Camera, p V3, radius int, col RGB) {
	pr := cam.Project(p)
	x := int(math.Round(pr.X))
	y := int(math.Round(pr.Y))
	r2 := radius * radius
	for oy := -radius; oy <= radius; oy++ {
		for ox := -radius; ox <= radius; ox++ {
			if ox*ox+oy*oy > r2 {
				continue
			}
			c.SetDepthPixel(x+ox, y+oy, pr.Depth, 0, col)
		}
	}
}

func (c *Canvas) DrawCloud(cam Camera, points []V3, colors []RGB, radius int) {
	for i, p := range points {
		col := RGB{220, 230, 238}
		if i < len(colors) {
			col = colors[i]
		}
		c.DrawPoint3D(cam, p, radius, col)
	}
}

func (c *Canvas) DrawLine3D(cam Camera, a, b V3, col RGB, thickness int, bias float64) {
	pa := cam.Project(a)
	pb := cam.Project(b)
	steps := int(math.Max(math.Abs(pb.X-pa.X), math.Abs(pb.Y-pa.Y))) + 1
	if steps > 4000 {
		steps = 4000
	}
	if thickness < 1 {
		thickness = 1
	}
	r := thickness / 2
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		x := lerpFloat(pa.X, pb.X, t)
		y := lerpFloat(pa.Y, pb.Y, t)
		d := lerpFloat(pa.Depth, pb.Depth, t)
		ix := int(math.Round(x))
		iy := int(math.Round(y))
		for oy := -r; oy <= r; oy++ {
			for ox := -r; ox <= r; ox++ {
				if thickness <= 2 || ox*ox+oy*oy <= r*r+r {
					c.SetDepthPixel(ix+ox, iy+oy, d, bias, col)
				}
			}
		}
	}
}

type projectedVertex struct {
	x, y, d float64
	p       V3
	n       V3
	c       RGB
}

func interpolateRGB(a, b, cc RGB, wa, wb, wc float64) RGB {
	return RGB{
		R: clampUint8((float64(a.R)*wa + float64(b.R)*wb + float64(cc.R)*wc)),
		G: clampUint8((float64(a.G)*wa + float64(b.G)*wb + float64(cc.G)*wc)),
		B: clampUint8((float64(a.B)*wa + float64(b.B)*wb + float64(cc.B)*wc)),
	}
}

func (c *Canvas) DrawMesh(cam Camera, m *Mesh, vertexColors bool, base RGB, lightDir V3) {
	if len(m.N) != len(m.P) {
		m.ComputeNormals()
	}
	lightDir = Normalize(lightDir)
	if Norm(lightDir) < 0.5 {
		lightDir = V3{-0.35, -0.45, 0.82}
	}

	for _, tri := range m.Tri {
		var pv [3]projectedVertex
		for k, id := range tri {
			pr := cam.Project(m.P[id])
			pv[k] = projectedVertex{x: pr.X, y: pr.Y, d: pr.Depth, p: m.P[id], n: m.N[id]}
			if vertexColors && id < len(m.C) {
				pv[k].c = m.C[id]
			} else {
				pv[k].c = base
			}
		}
		minX := int(math.Floor(math.Min(pv[0].x, math.Min(pv[1].x, pv[2].x))))
		maxX := int(math.Ceil(math.Max(pv[0].x, math.Max(pv[1].x, pv[2].x))))
		minY := int(math.Floor(math.Min(pv[0].y, math.Min(pv[1].y, pv[2].y))))
		maxY := int(math.Ceil(math.Max(pv[0].y, math.Max(pv[1].y, pv[2].y))))
		minX = maxInt(minX, 0)
		maxX = minInt(maxX, c.W-1)
		minY = maxInt(minY, 0)
		maxY = minInt(maxY, c.H-1)
		den := (pv[1].y-pv[2].y)*(pv[0].x-pv[2].x) +
			(pv[2].x-pv[1].x)*(pv[0].y-pv[2].y)
		if math.Abs(den) < 1e-9 {
			continue
		}

		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				px := float64(x) + 0.5
				py := float64(y) + 0.5
				w0 := ((pv[1].y-pv[2].y)*(px-pv[2].x) +
					(pv[2].x-pv[1].x)*(py-pv[2].y)) / den
				w1 := ((pv[2].y-pv[0].y)*(px-pv[2].x) +
					(pv[0].x-pv[2].x)*(py-pv[2].y)) / den
				w2 := 1 - w0 - w1
				if w0 < -0.002 || w1 < -0.002 || w2 < -0.002 {
					continue
				}
				depth := w0*pv[0].d + w1*pv[1].d + w2*pv[2].d
				n := Normalize(V3{
					w0*pv[0].n.X + w1*pv[1].n.X + w2*pv[2].n.X,
					w0*pv[0].n.Y + w1*pv[1].n.Y + w2*pv[2].n.Y,
					w0*pv[0].n.Z + w1*pv[1].n.Z + w2*pv[2].n.Z,
				})
				if Norm(n) < 0.5 {
					n = V3{0, 0, 1}
				}
				diff := math.Abs(Dot(n, lightDir))
				shade := 0.40 + 0.60*diff
				var col RGB
				if vertexColors {
					col = interpolateRGB(pv[0].c, pv[1].c, pv[2].c, w0, w1, w2)
					col = rgbf(float64(col.R)/255*(0.76+0.24*diff),
						float64(col.G)/255*(0.76+0.24*diff),
						float64(col.B)/255*(0.76+0.24*diff))
				} else {
					col = rgbf(float64(base.R)/255*shade,
						float64(base.G)/255*shade,
						float64(base.B)/255*shade)
				}
				c.SetDepthPixel(x, y, depth, 0, col)
			}
		}
	}
}

func (c *Canvas) DrawBoundaryEdges(cam Camera, m *Mesh, col RGB) {
	for _, e := range m.BoundaryEdges() {
		if e.A >= len(m.P) || e.B >= len(m.P) {
			continue
		}
		c.DrawLine3D(cam, m.P[e.A], m.P[e.B], col, 2, 0.006)
	}
}
