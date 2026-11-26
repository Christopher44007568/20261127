package main

import "math"

func scaleRGB(c RGB, s float64) RGB {
	return RGB{R: c.R * s, G: c.G * s, B: c.B * s}
}

func renderAzimuth(sceneID string) float64 {
	switch sceneID {
	case "thin_sheet":
		return -34
	case "concave_corner":
		return 32
	default:
		return -28
	}
}

func WriteMeshRenderPNG(path string, mesh *Mesh, scene Scene, errorMode bool) error {
	W, H := 820, 600
	label := "reconstructed TSDF surface"
	if errorMode {
		label = "surface vertex error heatmap"
	}
	cv := NewCanvas(W, H, RGB{0.93, 0.95, 0.98})
	cv.DrawText(18, 14, scene.ID+" / "+label, 2, RGB{0.04, 0.06, 0.09})

	vpX, vpY, vpW, vpH := 18, 48, W-36, H-92
	cv.FillRect(vpX, vpY, vpW, vpH, RGB{0.07, 0.09, 0.13})
	renderCam := OrbitCamera("render", V3(0, 0, 0.55), 2.35, renderAzimuth(scene.ID), 0.82, 38, vpW, vpH)
	depth := make([]float64, vpW*vpH)
	light := V3(-0.42, -0.58, 0.76).Norm()

	type rv struct {
		sx, sy, z, invZ float64
		n               Vec3
		err             float64
	}
	projectVertex := func(v MeshVertex) rv {
		pu, pv, pz, _ := renderCam.Project(v.Pos)
		return rv{
			sx:   float64(vpX) + pu,
			sy:   float64(vpY) + pv,
			z:    pz,
			invZ: 1 / pz,
			n:    v.Normal,
			err:  v.Error,
		}
	}

	for _, tri := range mesh.Triangles {
		r0 := projectVertex(mesh.Vertices[tri[0]])
		r1 := projectVertex(mesh.Vertices[tri[1]])
		r2 := projectVertex(mesh.Vertices[tri[2]])
		if r0.z <= 0.05 || r1.z <= 0.05 || r2.z <= 0.05 {
			continue
		}
		minX := int(math.Floor(math.Min(r0.sx, math.Min(r1.sx, r2.sx))))
		maxX := int(math.Ceil(math.Max(r0.sx, math.Max(r1.sx, r2.sx))))
		minY := int(math.Floor(math.Min(r0.sy, math.Min(r1.sy, r2.sy))))
		maxY := int(math.Ceil(math.Max(r0.sy, math.Max(r1.sy, r2.sy))))
		minX = clampInt(minX, vpX, vpX+vpW-1)
		maxX = clampInt(maxX, vpX, vpX+vpW-1)
		minY = clampInt(minY, vpY, vpY+vpH-1)
		maxY = clampInt(maxY, vpY, vpY+vpH-1)
		den := (r1.sy-r2.sy)*(r0.sx-r2.sx) + (r2.sx-r1.sx)*(r0.sy-r2.sy)
		if math.Abs(den) < 1e-9 {
			continue
		}

		for py := minY; py <= maxY; py++ {
			for px := minX; px <= maxX; px++ {
				pcx := float64(px) + 0.5
				pcy := float64(py) + 0.5
				w0 := ((r1.sy-r2.sy)*(pcx-r2.sx) + (r2.sx-r1.sx)*(pcy-r2.sy)) / den
				w1 := ((r2.sy-r0.sy)*(pcx-r2.sx) + (r0.sx-r2.sx)*(pcy-r2.sy)) / den
				w2 := 1 - w0 - w1
				if w0 < -1e-5 || w1 < -1e-5 || w2 < -1e-5 {
					continue
				}
				invZ := w0*r0.invZ + w1*r1.invZ + w2*r2.invZ
				id := (py-vpY)*vpW + (px - vpX)
				if invZ <= depth[id] {
					continue
				}
				depth[id] = invZ
				nx := (w0*r0.n.X/r0.z + w1*r1.n.X/r1.z + w2*r2.n.X/r2.z) / invZ
				ny := (w0*r0.n.Y/r0.z + w1*r1.n.Y/r1.z + w2*r2.n.Y/r2.z) / invZ
				nz := (w0*r0.n.Z/r0.z + w1*r1.n.Z/r1.z + w2*r2.n.Z/r2.z) / invZ
				n := V3(nx, ny, nz).Norm()
				diffuse := math.Max(0, n.Dot(light))
				intensity := 0.34 + 0.72*diffuse
				var col RGB
				if errorMode {
					err := (w0*r0.err/r0.z + w1*r1.err/r1.z + w2*r2.err/r2.z) / invZ
					col = scaleRGB(HeatColor(err/0.03), 0.62+0.45*diffuse)
				} else {
					base := RGB{0.70, 0.84, 0.93}
					rim := math.Pow(1-math.Abs(n.Z), 2) * 0.12
					col = scaleRGB(base, intensity+rim)
				}
				cv.Set(px, py, col)
			}
		}
	}

	cv.FrameRect(vpX, vpY, vpW, vpH, 2, RGB{0.18, 0.22, 0.28})
	if errorMode {
		barW := 260
		barX := vpX + 20
		barY := H - 28
		for x := 0; x < barW; x++ {
			cv.FillRect(barX+x, barY, 1, 10, HeatColor(float64(x)/float64(barW)))
		}
		cv.DrawText(barX, barY+13, "0 MM", 1, RGB{0.08, 0.09, 0.12})
		label30 := "30 MM"
		cv.DrawText(barX+barW-TextWidth(label30, 1), barY+13, label30, 1, RGB{0.08, 0.09, 0.12})
	} else {
		cv.DrawText(vpX+12, H-27, "MARCHING TETRAHEDRA ISO=0 / WEIGHT-GATED VOXELS", 1, RGB{0.10, 0.13, 0.18})
	}
	return SaveCanvasPNG(path, cv)
}
