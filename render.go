package qem

import "math"

var materialPalette = []RGB{
	{96, 165, 250},
	{255, 183, 77},
	{110, 222, 168},
	{226, 118, 255},
	{255, 107, 107},
	{125, 211, 252},
}

func materialColor(id int) RGB {
	if id < 0 {
		id = 0
	}
	return materialPalette[id%len(materialPalette)]
}

func rotateViewVector(n Vec3, view View) Vec3 {
	cy, sy := math.Cos(view.Yaw), math.Sin(view.Yaw)
	x := n.X*cy + n.Z*sy
	z0 := -n.X*sy + n.Z*cy
	y := n.Y
	cp, sp := math.Cos(view.Pitch), math.Sin(view.Pitch)
	y2 := y*cp - z0*sp
	z := y*sp + z0*cp
	return V(x, y2, z)
}

type rasterVertex struct {
	p projectedPoint
	w [3]float64
}

func edgeFunction(ax, ay, bx, by, px, py float64) float64 {
	return (px-ax)*(by-ay) - (py-ay)*(bx-ax)
}

// RenderShaded renders active triangles with a z-buffer and two-sided Lambert
// lighting. Flat normals make normal-detail loss visually direct.
func RenderShaded(m *Mesh, frame ViewFrame, view View) *PPM {
	img, _ := renderShadedDepth(m, frame, view)
	return img
}

func renderShadedDepth(m *Mesh, frame ViewFrame, view View) (*PPM, []float64) {
	img := NewPPM(frame.Width, frame.Height, ColorBackground)
	depth := make([]float64, frame.Width*frame.Height)
	for i := range depth {
		depth[i] = math.Inf(-1)
	}
	light := V(-0.38, 0.62, 0.69).Normalized()

	for fid, tri := range m.Triangles {
		if !tri.Active {
			continue
		}
		ids := tri.Vertices()
		pp := [3]projectedPoint{
			frame.Project(m.Vertices[ids[0]].Pos, view),
			frame.Project(m.Vertices[ids[1]].Pos, view),
			frame.Project(m.Vertices[ids[2]].Pos, view),
		}
		minX := maxInt(0, int(math.Floor(math.Min(pp[0].x, math.Min(pp[1].x, pp[2].x)))))
		maxX := minInt(frame.Width-1, int(math.Ceil(math.Max(pp[0].x, math.Max(pp[1].x, pp[2].x)))))
		minY := maxInt(0, int(math.Floor(math.Min(pp[0].y, math.Min(pp[1].y, pp[2].y)))))
		maxY := minInt(frame.Height-1, int(math.Ceil(math.Max(pp[0].y, math.Max(pp[1].y, pp[2].y)))))
		area := edgeFunction(pp[0].x, pp[0].y, pp[1].x, pp[1].y, pp[2].x, pp[2].y)
		if math.Abs(area) < 1e-9 {
			continue
		}
		normal := rotateViewVector(m.FaceNormal(fid), view)
		diffuse := math.Abs(normal.Dot(light))
		shade := 0.34 + 0.72*diffuse
		base := materialColor(tri.Material)
		color := rgbf(float64(base.R)*shade, float64(base.G)*shade, float64(base.B)*shade)

		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				px := float64(x) + 0.5
				py := float64(y) + 0.5
				w0 := edgeFunction(pp[1].x, pp[1].y, pp[2].x, pp[2].y, px, py) / area
				w1 := edgeFunction(pp[2].x, pp[2].y, pp[0].x, pp[0].y, px, py) / area
				w2 := edgeFunction(pp[0].x, pp[0].y, pp[1].x, pp[1].y, px, py) / area
				if w0 < -1e-5 || w1 < -1e-5 || w2 < -1e-5 {
					continue
				}
				z := w0*pp[0].z + w1*pp[1].z + w2*pp[2].z
				idx := y*frame.Width + x
				if z <= depth[idx] {
					continue
				}
				depth[idx] = z
				img.Pix[idx] = color
			}
		}
	}
	return img, depth
}

// RenderCollapseHighlights renders the current mesh and all recorded collapsed
// edges. Old endpoint positions are used, so highlights show the exact locked
// candidate edge that was removed.
func RenderCollapseHighlights(m *Mesh, frame ViewFrame, view View, records []*CollapseRecord, start int) *PPM {
	img, depth := renderShadedDepth(m, frame, view)
	if start < 0 {
		start = 0
	}
	if start > len(records) {
		start = len(records)
	}
	bias := 0.006 * frame.MaxDimension
	for i := start; i < len(records); i++ {
		rec := records[i]
		a := frame.Project(rec.RemovedOldPos, view)
		b := frame.Project(rec.KeptOldPos, view)
		latest := i == len(records)-1
		thickness := 1
		main := ColorHighlight
		if latest {
			thickness = 2
			main = ColorHighlight2
		}
		x0, y0 := int(a.x), int(a.y)
		x1, y1 := int(b.x), int(b.y)
		img.DrawLineDepth(x0, y0, a.z, x1, y1, b.z, thickness+2, RGB{5, 6, 10}, 0.70, bias*1.8, depth)
		img.DrawLineDepth(x0, y0, a.z, x1, y1, b.z, thickness, main, 0.98, bias, depth)
	}
	return img
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TurntableViews returns evenly spaced yaw views with a slight fixed pitch.
func TurntableViews(count int, pitch float64) []View {
	views := make([]View, count)
	for i := range views {
		views[i] = View{Yaw: 2 * math.Pi * float64(i) / float64(count), Pitch: pitch}
	}
	return views
}
