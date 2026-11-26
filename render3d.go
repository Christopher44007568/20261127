package holefill

import (
	"math"
)

type Camera struct {
	Center Vec3
	Scale  float64
	CX, CY float64
	Yaw    float64
	Tilt   float64
	W, H   int
}

type Projected struct {
	X, Y, Z float64
}

func NewCamera(m *Mesh, w, h int) Camera {
	min, max := boundingBox(m)
	center := min.Add(max).Scale(0.5)
	cam := Camera{
		Center: center, W: w, H: h,
		Yaw: -0.16, Tilt: 0.62,
		CX: float64(w) * 0.5,
		CY: float64(h)*0.52 + 18,
	}
	corners := []Vec3{
		{min.X, min.Y, min.Z}, {max.X, min.Y, min.Z},
		{min.X, max.Y, min.Z}, {max.X, max.Y, min.Z},
		{min.X, min.Y, max.Z}, {max.X, min.Y, max.Z},
		{min.X, max.Y, max.Z}, {max.X, max.Y, max.Z},
	}
	var pxMin, pxMax, pyMin, pyMax float64 = math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range corners {
		q := cam.projectUnit(p)
		pxMin, pxMax = math.Min(pxMin, q.X), math.Max(pxMax, q.X)
		pyMin, pyMax = math.Min(pyMin, q.Y), math.Max(pyMax, q.Y)
	}
	sx := (float64(w) - 150) / math.Max(pxMax-pxMin, 1)
	sy := (float64(h) - 190) / math.Max(pyMax-pyMin, 1)
	cam.Scale = math.Min(sx, sy)
	return cam
}

func boundingBox(m *Mesh) (Vec3, Vec3) {
	min := Vec3{math.Inf(1), math.Inf(1), math.Inf(1)}
	max := Vec3{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, p := range m.V {
		min = Vec3{math.Min(min.X, p.X), math.Min(min.Y, p.Y), math.Min(min.Z, p.Z)}
		max = Vec3{math.Max(max.X, p.X), math.Max(max.Y, p.Y), math.Max(max.Z, p.Z)}
	}
	return min, max
}

func (c Camera) projectUnit(p Vec3) Projected {
	q := p.Sub(c.Center)
	cy, sy := math.Cos(c.Yaw), math.Sin(c.Yaw)
	x1 := q.X*cy - q.Y*sy
	y1 := q.X*sy + q.Y*cy
	ct, st := math.Cos(c.Tilt), math.Sin(c.Tilt)
	y2 := y1*ct - q.Z*st
	z2 := y1*st + q.Z*ct
	return Projected{X: x1, Y: y2, Z: z2}
}

func (c Camera) Project(p Vec3) Projected {
	u := c.projectUnit(p)
	return Projected{X: c.CX + u.X*c.Scale, Y: c.CY - u.Y*c.Scale, Z: u.Z}
}

func RenderMesh(m *Mesh, w, h int, background RGB, mode string) *Canvas {
	cv := NewCanvas(w, h, background)
	cam := NewCamera(m, w, h)
	RenderMeshToCanvas(cv, cam, m, background, mode)
	return cv
}

func RenderMeshToCanvas(cv *Canvas, cam Camera, m *Mesh, background RGB, mode string) {
	light := Vec3{-0.38, -0.48, 0.79}.Normalize()
	if len(m.Norm) != len(m.V) {
		m.ComputeNormals()
	}

	for ti, tri := range m.T {
		tag := FaceTag{Kind: KindScan}
		if ti < len(m.Tags) {
			tag = m.Tags[ti]
		}
		base := faceColor(tag, mode)
		p0 := cam.Project(m.V[tri[0]])
		p1 := cam.Project(m.V[tri[1]])
		p2 := cam.Project(m.V[tri[2]])
		minX := int(math.Max(0, math.Floor(math.Min(p0.X, math.Min(p1.X, p2.X)))))
		maxX := int(math.Min(float64(cv.W-1), math.Ceil(math.Max(p0.X, math.Max(p1.X, p2.X)))))
		minY := int(math.Max(0, math.Floor(math.Min(p0.Y, math.Min(p1.Y, p2.Y)))))
		maxY := int(math.Min(float64(cv.H-1), math.Ceil(math.Max(p0.Y, math.Max(p1.Y, p2.Y)))))
		den := (p1.Y-p2.Y)*(p0.X-p2.X) + (p2.X-p1.X)*(p0.Y-p2.Y)
		if math.Abs(den) < 1e-9 {
			continue
		}
		n0 := m.Norm[tri[0]]
		n1 := m.Norm[tri[1]]
		n2 := m.Norm[tri[2]]
		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				px := float64(x) + 0.5
				py := float64(y) + 0.5
				w0 := ((p1.Y-p2.Y)*(px-p2.X) + (p2.X-p1.X)*(py-p2.Y)) / den
				w1 := ((p2.Y-p0.Y)*(px-p2.X) + (p0.X-p2.X)*(py-p2.Y)) / den
				w2 := 1 - w0 - w1
				if w0 < -0.002 || w1 < -0.002 || w2 < -0.002 {
					continue
				}
				depth := w0*p0.Z + w1*p1.Z + w2*p2.Z
				norm := n0.Scale(w0).Add(n1.Scale(w1)).Add(n2.Scale(w2)).Normalize()
				if norm.Len() < 0.5 {
					norm = Vec3{0, 0, 1}
				}
				diff := math.Max(0, norm.Dot(light))
				sky := 0.18 + 0.16*math.Max(0, norm.Z)
				shade := sky + 0.78*diff
				col := ScaleRGB(base, math.Min(1.18, shade))
				if tag.Kind == KindPatch && mode == "patched" {
					edge := math.Min(w0, math.Min(w1, w2))
					if edge < 0.035 {
						col = Blend(col, RGB{15, 23, 42}, 0.28)
					}
				}
				cv.SetDepth(x, y, depth, col)
			}
		}
	}
}

func faceColor(tag FaceTag, mode string) RGB {
	switch tag.Kind {
	case KindPatch:
		col := tag.Class.Color()
		if mode == "holes" {
			return Blend(col, RGB{30, 41, 59}, 0.35)
		}
		return Blend(col, RGB{255, 255, 255}, 0.08)
	case KindBottom:
		return RGB{71, 85, 105}
	case KindWall:
		return RGB{180, 142, 92}
	default:
		if mode == "holes" {
			return RGB{103, 122, 148}
		}
		return RGB{129, 150, 176}
	}
}

func ScaleRGB(c RGB, s float64) RGB {
	return RGB{
		R: clampByte(float64(c.R) * s),
		G: clampByte(float64(c.G) * s),
		B: clampByte(float64(c.B) * s),
	}
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

func DrawLoopOverlay(cv *Canvas, cam Camera, m *Mesh, loop *BoundaryLoop, thick int) {
	col := loop.Class.Color()
	glow := RGB{15, 23, 42}
	for i := 0; i < len(loop.Verts); i++ {
		a := cam.Project(m.V[loop.Verts[i]])
		b := cam.Project(m.V[loop.Verts[(i+1)%len(loop.Verts)]])
		cv.Line(int(a.X), int(a.Y), int(b.X), int(b.Y), thick+3, glow)
		cv.Line(int(a.X), int(a.Y), int(b.X), int(b.Y), thick, col)
	}
}

func DrawAllLoopOverlays(cv *Canvas, cam Camera, m *Mesh, loops []*BoundaryLoop, includeOuter bool) {
	for _, l := range loops {
		if !includeOuter && l.Class == ClassOuter {
			continue
		}
		thick := 3
		if l.Class == ClassOuter {
			thick = 2
		}
		DrawLoopOverlay(cv, cam, m, l, thick)
	}
}
