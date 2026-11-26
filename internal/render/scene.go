package render

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"

	"meshdiagnostics/internal/diag"
	"meshdiagnostics/internal/mesh"
	"meshdiagnostics/internal/qmesh"
	"meshdiagnostics/internal/remesh"
)

// Camera is a perspective camera fitted to a sphere around Target.
type Camera struct {
	Eye, Target, Right, Up, Forward mesh.Vec3
	W, H, F                         float64
}

func NewCamera(target mesh.Vec3, radius float64, width, height int, direction mesh.Vec3) *Camera {
	if radius < 1e-10 {
		radius = 1
	}
	if direction.Len2() < 1e-12 {
		direction = mesh.V(0.55, 0.30, -0.78)
	}
	direction = direction.Normalize()
	eye := target.Add(direction.Scale(radius * 3.9))
	worldUp := mesh.V(0, 1, 0)
	forward := target.Sub(eye).Normalize()
	if math.Abs(forward.Dot(worldUp)) > 0.98 {
		worldUp = mesh.V(0, 0, 1)
	}
	right := worldUp.Cross(forward).Normalize()
	up := forward.Cross(right).Normalize()
	fov := 34.0 * math.Pi / 180
	focal := float64(height) * 0.5 / math.Tan(fov*0.5)
	return &Camera{
		Eye: eye, Target: target, Right: right, Up: up, Forward: forward,
		W: float64(width), H: float64(height), F: focal,
	}
}

func (c *Camera) Project(p mesh.Vec3) (float64, float64, float64, bool) {
	d := p.Sub(c.Eye)
	z := d.Dot(c.Forward)
	if z <= 0.02 {
		return 0, 0, 0, false
	}
	x := d.Dot(c.Right)
	y := d.Dot(c.Up)
	return c.W*0.5 + x*c.F/z, c.H*0.5 - y*c.F/z, z, true
}

type sceneEdge struct {
	A, B   mesh.Vec3
	Color  color.RGBA
	Width  int
	Hidden bool
}

func drawFaces(c *Canvas, cam *Camera, vertices []mesh.Vec3, faces [][]int, faceColor func(int) color.RGBA) {
	light := mesh.V(0.35, 0.72, -0.60).Normalize()
	for fi, f := range faces {
		if len(f) < 3 {
			continue
		}
		pts := make([]mesh.Vec3, len(f))
		var center, normal mesh.Vec3
		for i, vi := range f {
			pts[i] = vertices[vi]
			center = center.Add(pts[i])
		}
		center = center.Scale(1 / float64(len(f)))
		for i := 0; i < len(f); i++ {
			a, b := pts[i], pts[(i+1)%len(f)]
			normal.X += (a.Y - b.Y) * (a.Z + b.Z)
			normal.Y += (a.Z - b.Z) * (a.X + b.X)
			normal.Z += (a.X - b.X) * (a.Y + b.Y)
		}
		normal = normal.Normalize()
		if normal.Dot(cam.Eye.Sub(center)) < 0 {
			normal = normal.Scale(-1)
		}
		viewDir := center.Sub(cam.Eye).Normalize()
		diffuse := math.Abs(normal.Dot(light))
		rim := math.Pow(math.Max(0, 1-math.Abs(normal.Dot(viewDir))), 2)
		intensity := 0.52 + 0.40*diffuse + 0.18*rim
		base := faceColor(fi)
		col := color.RGBA{
			R: clampByte(float64(base.R) * intensity),
			G: clampByte(float64(base.G) * intensity),
			B: clampByte(float64(base.B) * intensity),
			A: 255,
		}
		sx := make([]float64, len(f))
		sy := make([]float64, len(f))
		sz := make([]float64, len(f))
		ok := true
		for i, p := range pts {
			sx[i], sy[i], sz[i], ok = cam.Project(p)
			if !ok {
				break
			}
		}
		if !ok {
			continue
		}
		for i := 1; i+1 < len(f); i++ {
			c.FillTriangle(sx[0], sy[0], sz[0], sx[i], sy[i], sz[i], sx[i+1], sy[i+1], sz[i+1], col)
		}
	}
}

func drawEdges(c *Canvas, cam *Camera, edges []sceneEdge) {
	for _, e := range edges {
		x0, y0, z0, ok0 := cam.Project(e.A)
		x1, y1, z1, ok1 := cam.Project(e.B)
		if !ok0 || !ok1 {
			continue
		}
		if e.Hidden {
			c.DrawLineZ(x0, y0, z0, x1, y1, z1, e.Width, e.Color)
		} else {
			c.DrawLine(x0, y0, x1, y1, e.Width, e.Color)
		}
	}
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func backgroundGradient(c *Canvas, top, bottom color.RGBA) {
	for y := 0; y < c.H; y++ {
		t := float64(y) / float64(c.H-1)
		col := color.RGBA{
			R: uint8(float64(top.R)*(1-t) + float64(bottom.R)*t),
			G: uint8(float64(top.G)*(1-t) + float64(bottom.G)*t),
			B: uint8(float64(top.B)*(1-t) + float64(bottom.B)*t),
			A: 255,
		}
		for x := 0; x < c.W; x++ {
			c.Pix[c.Index(x, y)] = col
		}
	}
}

func fitSphere(vertices []mesh.Vec3) (mesh.Vec3, float64) {
	lo, hi := mesh.Bounds(vertices)
	center := lo.Add(hi).Scale(0.5)
	radius := hi.Sub(lo).Len() * 0.5
	if radius < 1e-12 {
		radius = 1
	}
	return center, radius
}

func triFaces(m *mesh.TriangleMesh) [][]int {
	out := make([][]int, len(m.Faces))
	for i, f := range m.Faces {
		out[i] = []int{f[0], f[1], f[2]}
	}
	return out
}

func quadFaces(q *qmesh.Mesh) [][]int {
	out := make([][]int, len(q.Faces))
	for i, f := range q.Faces {
		out[i] = []int{f[0], f[1], f[2], f[3]}
	}
	return out
}

func edgeList(q *qmesh.Mesh, dark color.RGBA, width int) []sceneEdge {
	edges := q.Edges()
	out := make([]sceneEdge, 0, len(edges))
	for key := range edges {
		col := dark
		w := width
		if q.Features[key] {
			col = color.RGBA{232, 78, 52, 255}
			w = width*2 + 1
		}
		out = append(out, sceneEdge{A: q.Vertices[key.A], B: q.Vertices[key.B], Color: col, Width: w, Hidden: true})
	}
	return out
}

// RenderOptions controls output resolution and styling.
type RenderOptions struct {
	Width, Height int
	CollapseDeg   float64
	SuperSample   int
}

func (o RenderOptions) canvas(bg color.RGBA) *Canvas {
	f := o.SuperSample
	if f < 1 {
		f = 1
	}
	return NewCanvas(o.Width*f, o.Height*f, bg)
}

func (o RenderOptions) save(c *Canvas, path string) error {
	img := c.Downsample(o.SuperSample)
	return SavePNG(path, img)
}

// RenderOriginal draws the source triangle mesh with detected feature edges.
func RenderOriginal(path string, src *mesh.TriangleMesh, features mesh.FeatureInfo, opt RenderOptions) error {
	bg := color.RGBA{18, 28, 36, 255}
	c := opt.canvas(bg)
	backgroundGradient(c, color.RGBA{27, 44, 54, 255}, color.RGBA{10, 17, 24, 255})
	center, radius := fitSphere(src.Vertices)
	cam := NewCamera(center, radius, c.W, c.H, mesh.V(0.55, 0.28, -0.78))
	drawFaces(c, cam, src.Vertices, triFaces(src), func(int) color.RGBA {
		return color.RGBA{83, 173, 167, 255}
	})
	edges := src.EdgeMap()
	wires := make([]sceneEdge, 0, len(edges))
	for key := range edges {
		col := color.RGBA{13, 32, 42, 210}
		w := 1 * opt.SuperSample
		if features.Edges[key] {
			col = color.RGBA{255, 112, 60, 255}
			w = 2 * opt.SuperSample
		}
		wires = append(wires, sceneEdge{
			A: src.Vertices[key.A], B: src.Vertices[key.B], Color: col, Width: w, Hidden: true,
		})
	}
	drawEdges(c, cam, wires)
	drawHeader(c, opt, "ORIGINAL TRIANGLE MESH", []string{
		fmt.Sprintf("VERTICES %d   TRIANGLES %d", len(src.Vertices), len(src.Faces)),
		fmt.Sprintf("SHARP FEATURE EDGES %d", len(features.Edges)),
	}, color.RGBA{210, 242, 240, 255}, color.RGBA{255, 112, 60, 255}, "SHARP FEATURE")
	return opt.save(c, path)
}

// RenderQuad draws the all-quad result, highlighting collapsed faces.
func RenderQuad(path string, q *qmesh.Mesh, a *diag.Analysis, rs remesh.Stats, opt RenderOptions) error {
	bg := color.RGBA{236, 238, 231, 255}
	c := opt.canvas(bg)
	backgroundGradient(c, color.RGBA{247, 245, 236, 255}, color.RGBA{209, 219, 214, 255})
	center, radius := fitSphere(q.Vertices)
	cam := NewCamera(center, radius, c.W, c.H, mesh.V(0.55, 0.28, -0.78))
	drawFaces(c, cam, q.Vertices, quadFaces(q), func(i int) color.RGBA {
		m := a.FaceMetrics[i]
		switch {
		case contains(a.Collapsed, i) || !m.Convex:
			return color.RGBA{226, 66, 55, 255}
		case m.MinAngleDeg < opt.CollapseDeg*2:
			return color.RGBA{239, 177, 72, 255}
		default:
			return color.RGBA{218, 231, 224, 255}
		}
	})
	drawEdges(c, cam, edgeList(q, color.RGBA{21, 50, 59, 135}, 1*opt.SuperSample))
	drawHeader(c, opt, "FEATURE-PRESERVING QUAD MESH", []string{
		fmt.Sprintf("QUADS %d   EDGES %d", len(q.Faces), rs.QuadEdges),
		fmt.Sprintf("FEATURE LENGTH KEPT %.2f%%", rs.FeatureLengthRatio*100),
		fmt.Sprintf("COLLAPSED QUADS %d / %d", len(a.Collapsed), len(q.Faces)),
	}, color.RGBA{24, 45, 51, 255}, color.RGBA{226, 66, 55, 255}, "ANGLE COLLAPSE")
	drawSwatchLegend(c, opt, 2, "WARNING 2X THRESHOLD", color.RGBA{239, 177, 72, 255})
	return opt.save(c, path)
}

func contains(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func drawHeader(c *Canvas, opt RenderOptions, title string, lines []string, text, accent color.RGBA, legend string) {
	s := opt.SuperSample
	x, y := 24*s, 22*s
	scale := 4 * s
	c.DrawText(x+2*s, y+2*s, title, scale, color.RGBA{0, 0, 0, 100})
	c.DrawText(x, y, title, scale, text)
	y += 44 * s
	scale = 3 * s
	for _, line := range lines {
		c.DrawText(x, y, line, scale, text)
		y += 24 * s
	}
	c.FillRect(x, y+4*s, 28*s, 3*s, accent)
	c.DrawText(x+35*s, y, legend, scale, text)
}

func drawSwatchLegend(c *Canvas, opt RenderOptions, index int, label string, col color.RGBA) {
	s := opt.SuperSample
	x := 24 * s
	y := (opt.Height - 34 - index*24) * s
	c.FillRect(x, y, 18*s, 12*s, col)
	c.DrawText(x+24*s, y, label, 3*s, color.RGBA{24, 45, 51, 255})
}

var curvaturePalette = []color.RGBA{
	{42, 82, 150, 255},
	{42, 158, 199, 255},
	{112, 197, 144, 255},
	{242, 202, 80, 255},
	{224, 72, 51, 255},
}

// RenderCurvature shows quantile curvature bands over the quad mesh.
func RenderCurvature(path string, q *qmesh.Mesh, a *diag.Analysis, opt RenderOptions) error {
	bg := color.RGBA{246, 244, 236, 255}
	c := opt.canvas(bg)
	backgroundGradient(c, color.RGBA{251, 249, 242, 255}, color.RGBA{222, 227, 220, 255})
	center, radius := fitSphere(q.Vertices)
	cam := NewCamera(center, radius, c.W, c.H, mesh.V(0.55, 0.28, -0.78))
	drawFaces(c, cam, q.Vertices, quadFaces(q), func(i int) color.RGBA {
		band := a.CurvBands[i]
		if band < 0 || band >= len(curvaturePalette) {
			band = 0
		}
		return curvaturePalette[band]
	})
	drawEdges(c, cam, edgeList(q, color.RGBA{18, 37, 45, 105}, 1*opt.SuperSample))
	drawHeader(c, opt, "QUAD MESH CURVATURE BANDS", []string{
		fmt.Sprintf("MEAN |K| %.3f   MAX %.3f", a.CurvStats.Mean, a.CurvStats.Max),
		fmt.Sprintf("QUANTILE BANDS   FEATURES %d", len(q.Features)),
	}, color.RGBA{21, 42, 48, 255}, color.RGBA{21, 42, 48, 255}, "BEND / LENGTH")
	drawCurvatureLegend(c, opt, a.CurvThresholds)
	return opt.save(c, path)
}

func drawCurvatureLegend(c *Canvas, opt RenderOptions, thresholds []float64) {
	s := opt.SuperSample
	x := 24 * s
	y := (opt.Height - 58) * s
	w, h := 50*s, 13*s
	for i, col := range curvaturePalette {
		c.FillRect(x+i*w, y, w, h, col)
	}
	for i := 0; i < len(thresholds) && i+1 < len(curvaturePalette); i++ {
		c.DrawLine(float64(x+(i+1)*w), float64(y), float64(x+(i+1)*w), float64(y+h), s, color.RGBA{20, 25, 26, 255})
	}
	labels := []string{"0-20%", "20-40%", "40-60%", "60-80%", "80-100%"}
	for i, label := range labels {
		c.DrawText(x+i*w, y+h+4*s, label, s, color.RGBA{25, 42, 46, 255})
	}
}

// RenderZoom focuses on the quad with the smallest interior angle.
func RenderZoom(path string, q *qmesh.Mesh, a *diag.Analysis, rs remesh.Stats, opt RenderOptions) error {
	bg := color.RGBA{242, 239, 230, 255}
	c := opt.canvas(bg)
	backgroundGradient(c, color.RGBA{249, 246, 237, 255}, color.RGBA{211, 216, 206, 255})
	globalCenter, globalRadius := fitSphere(q.Vertices)
	target := globalCenter
	radius := globalRadius
	var worst qmesh.Face
	if a.WorstFace >= 0 && a.WorstFace < len(q.Faces) {
		worst = q.Faces[a.WorstFace]
		target = q.FaceCentroid(worst)
		radius = math.Max(1.9*a.WorstMetrics.MaxEdge, globalRadius*0.014)
		if radius < globalRadius*0.006 {
			radius = globalRadius * 0.006
		}
	}
	direction := q.FaceNormal(worst).Normalize()
	if direction.Dot(target.Sub(globalCenter)) < 0 {
		direction = direction.Scale(-1)
	}
	if direction.Len2() < 1e-12 {
		direction = mesh.V(0.55, 0.28, -0.78)
	}
	direction = direction.Add(mesh.V(0.18, 0.15, 0)).Normalize()
	cam := NewCamera(target, radius, c.W, c.H, direction)
	drawFaces(c, cam, q.Vertices, quadFaces(q), func(i int) color.RGBA {
		if i == a.WorstFace {
			return color.RGBA{236, 91, 48, 255}
		}
		if contains(a.Collapsed, i) {
			return color.RGBA{246, 194, 91, 255}
		}
		return color.RGBA{201, 216, 208, 255}
	})
	drawEdges(c, cam, edgeList(q, color.RGBA{22, 48, 54, 220}, 1*opt.SuperSample))
	drawWorstQuadOverlay(c, cam, q, worst, a.WorstMetrics, opt)
	drawHeader(c, opt, "LOCAL ZOOM - WORST QUAD", nil, color.RGBA{30, 44, 46, 255}, color.RGBA{236, 91, 48, 255}, "")
	drawZoomPanel(c, opt, a, rs)
	return opt.save(c, path)
}

func drawWorstQuadOverlay(c *Canvas, cam *Camera, q *qmesh.Mesh, f qmesh.Face, m qmesh.Metrics, opt RenderOptions) {
	if len(f) != 4 {
		return
	}
	s := opt.SuperSample
	var pts [4][2]float64
	for i := 0; i < 4; i++ {
		x, y, _, ok := cam.Project(q.Vertices[f[i]])
		if !ok {
			return
		}
		pts[i] = [2]float64{x, y}
	}
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		c.DrawLine(pts[i][0], pts[i][1], pts[j][0], pts[j][1], 3*s, color.RGBA{255, 220, 82, 255})
		c.FillCircle(int(pts[i][0]), int(pts[i][1]), 3*s, color.RGBA{255, 246, 190, 255})
	}
	minIndex := 0
	for i := 1; i < 4; i++ {
		if m.AnglesDeg[i] < m.AnglesDeg[minIndex] {
			minIndex = i
		}
	}
	c.DrawText(int(pts[minIndex][0])+5*s, int(pts[minIndex][1])-12*s,
		fmt.Sprintf("MIN %.1f DEG", m.MinAngleDeg), 3*s, color.RGBA{255, 248, 218, 255})
}

func drawZoomPanel(c *Canvas, opt RenderOptions, a *diag.Analysis, rs remesh.Stats) {
	s := opt.SuperSample
	width := 410 * s
	x := (opt.Width - 410 - 18) * s
	y := (opt.Height - 226) * s
	panel := color.RGBA{15, 30, 33, 225}
	c.FillRect(x, y, width, 204*s, panel)
	c.FillRect(x, y, 5*s, 204*s, color.RGBA{236, 91, 48, 255})
	x += 16 * s
	y += 16 * s
	text := color.RGBA{230, 242, 236, 255}
	t := fmt.Sprintf("QUAD #%d", a.WorstFace)
	c.DrawText(x, y, t, 4*s, color.RGBA{255, 220, 82, 255})
	y += 32 * s
	lines := []string{
		fmt.Sprintf("MIN ANGLE %.2f DEG", a.WorstMetrics.MinAngleDeg),
		fmt.Sprintf("MAX ANGLE %.2f DEG", a.WorstMetrics.MaxAngleDeg),
		fmt.Sprintf("CONVEX %s", yesNo(a.WorstMetrics.Convex)),
		fmt.Sprintf("COLLAPSED %d / %d  %.2f%%", len(a.Collapsed), a.FaceCount,
			float64(len(a.Collapsed))*100/float64(a.FaceCount)),
		fmt.Sprintf("THRESHOLD %.1f DEG", opt.CollapseDeg),
		fmt.Sprintf("FEATURE EDGES %d  KEPT %.1f%%", rs.QuadFeatures, rs.FeatureLengthRatio*100),
	}
	lineScale := 2 * s
	for _, line := range lines {
		c.DrawText(x, y, line, lineScale, text)
		y += 18 * s
	}
}

func yesNo(v bool) string {
	if v {
		return "YES"
	}
	return "NO"
}

func OutputName(dir, name string) string {
	return filepath.Join(dir, name)
}
