package main

import (
	"fmt"
	"image/color"
	"strings"

	"example.com/bezierstroke"
)

var (
	colBackground = color.RGBA{R: 7, G: 12, B: 20, A: 255}
	colGrid       = color.RGBA{R: 18, G: 31, B: 48, A: 255}
	colAxis       = color.RGBA{R: 31, G: 60, B: 84, A: 255}
	colBorder     = color.RGBA{R: 44, G: 76, B: 104, A: 255}
	colExact      = color.RGBA{R: 85, G: 235, B: 160, A: 255}
	colExactFaint = color.RGBA{R: 36, G: 86, B: 76, A: 255}
	colCenter     = color.RGBA{R: 255, G: 178, B: 82, A: 255}
	colWire       = color.RGBA{R: 45, G: 136, B: 170, A: 255}
	colVertex     = color.RGBA{R: 255, G: 226, B: 147, A: 255}
	colGlow       = color.RGBA{R: 13, G: 66, B: 88, A: 255}
	colFinal      = color.RGBA{R: 105, G: 229, B: 255, A: 255}
	colPanel      = color.RGBA{R: 5, G: 10, B: 17, A: 255}
	colText       = color.RGBA{R: 221, G: 235, B: 247, A: 255}
	colMuted      = color.RGBA{R: 129, G: 157, B: 179, A: 255}
	colAccent     = color.RGBA{R: 255, G: 178, B: 82, A: 255}
)

type CanvasRenderer struct {
	PreparedScene
	Canvas  *bezierstroke.Canvas
	Samples int
}

func newCanvasRenderer(p PreparedScene, samples int) *CanvasRenderer {
	return &CanvasRenderer{
		PreparedScene: p,
		Canvas:        bezierstroke.NewCanvas(p.W, p.H, colBackground),
		Samples:       samples,
	}
}

func (r *CanvasRenderer) drawOriginal() {
	r.drawGrid()
	r.drawLine(adaptivePoints(r.Exact), 2.0, colExact, true, 0.02)
	r.drawEndpoints(colExact)
	r.drawHUD("ORIGINAL CURVE",
		fmt.Sprintf("EXACT CUBIC   FLATNESS 0.02 PX   SCALE %.0f PX/UNIT", r.Scale),
		"REFERENCE GEOMETRY / NO POLYGONAL APPROXIMATION IN THE SOURCE")
}

func (r *CanvasRenderer) drawMesh() {
	r.drawGrid()
	r.drawLine(adaptivePoints(r.Exact), 1.4, colExactFaint, true, 0.04)

	var wireLines [][]bezierstroke.Point
	for _, poly := range r.Mesh.Polygons {
		wireLines = append(wireLines, closedOutline(poly))
	}
	r.drawLines(wireLines, 0.65, colWire, false, 8, 0.05)
	r.drawLine(adaptivePoints(r.Adaptive), 1.35, colCenter, false, 0.05)
	r.drawVertices(r.Adaptive, 2.6, colVertex)
	r.drawHUD("ADAPTIVE MESH",
		fmt.Sprintf("TOLERANCE %.2f PX   VERTICES %d   MAX CHORD ERROR %.3f PX",
			r.Tolerance, len(r.Adaptive), r.Metrics.MaxChordErrorPx),
		fmt.Sprintf("FIXED 8 ERROR %.3f PX   ORANGE CENTERLINE   CYAN STROKE MESH",
			r.Metrics.Fixed8MaxErrorPx))
}

func (r *CanvasRenderer) drawFinal() {
	r.drawGrid()
	width := r.StrokeWidth
	glow := bezierstroke.StrokePolyline(adaptivePoints(r.Adaptive), bezierstroke.StrokeStyle{
		Width:     width + 10,
		Join:      bezierstroke.JoinRound,
		Cap:       bezierstroke.CapRound,
		Tolerance: 0.08,
	})
	r.Canvas.FillPolygons(glow.Polygons, r.Samples, colGlow)
	r.Canvas.FillPolygons(r.Mesh.Polygons, r.Samples, colFinal)
	r.drawHUD("COVERAGE FINAL",
		fmt.Sprintf("WIDTH %.0f PX   TOLERANCE %.2f PX   %d X %d SAMPLES/PX",
			r.StrokeWidth, r.Tolerance, r.Samples, r.Samples),
		fmt.Sprintf("FIXED 8 ERROR %.3f PX   ADAPTIVE ERROR %.3f PX   CONSTANT WIDTH",
			r.Metrics.Fixed8MaxErrorPx, r.Metrics.MaxChordErrorPx))
}

func (r *CanvasRenderer) drawHUD(stage, detail, footer string) {
	r.fillRects([][4]float64{
		{0, 0, float64(r.W), 92},
		{0, float64(r.H - 46), float64(r.W), float64(r.H)},
		{16, 20, 20, 72},
	}, colPanel, r.Samples)
	r.fillRects([][4]float64{{18, 18, 22, 74}}, colAccent, r.Samples)

	sceneName := strings.ReplaceAll(strings.ToUpper(r.Key), "_", " ")
	title := sceneName + "  /  " + strings.ToUpper(stage)
	drawText(r.Canvas, 36, 16, 3, title, colText)
	drawText(r.Canvas, 37, 51, 2, detail, colMuted)
	drawText(r.Canvas, 22, r.H-32, 2, footer, colMuted)
}

func (r *CanvasRenderer) drawGrid() {
	w, h := r.W, r.H
	lines := make([][]bezierstroke.Point, 0, 64)
	for x := 80; x < w; x += 80 {
		lines = append(lines, []bezierstroke.Point{{X: float64(x), Y: 0}, {X: float64(x), Y: float64(h)}})
	}
	for y := 80; y < h; y += 80 {
		lines = append(lines, []bezierstroke.Point{{X: 0, Y: float64(y)}, {X: float64(w), Y: float64(y)}})
	}

	// Grid is intentionally one color; the axes are redrawn below.
	r.drawLines(lines, 1.0, colGrid, false, 4, 0.05)
	cx, cy := float64(w)/2, float64(h)/2
	r.drawLines([][]bezierstroke.Point{
		{{X: cx, Y: 0}, {X: cx, Y: float64(h)}},
		{{X: 0, Y: cy}, {X: float64(w), Y: cy}},
	}, 1.1, colAxis, false, 4, 0.05)
	r.fillRects([][4]float64{
		{0, 0, float64(w), 2},
		{0, float64(h - 2), float64(w), float64(h)},
		{0, 0, 2, float64(h)},
		{float64(w - 2), 0, float64(w), float64(h)},
	}, colBorder, r.Samples)
}

func (r *CanvasRenderer) drawLine(points []bezierstroke.Point, width float64,
	fg color.RGBA, round bool, tol float64) {
	if len(points) < 2 {
		return
	}
	style := bezierstroke.StrokeStyle{
		Width:     width,
		Join:      bezierstroke.JoinBevel,
		Cap:       bezierstroke.CapButt,
		Tolerance: tol,
	}
	if round {
		style.Join = bezierstroke.JoinRound
		style.Cap = bezierstroke.CapRound
	}
	mesh := bezierstroke.StrokePolyline(points, style)
	r.Canvas.FillPolygons(mesh.Polygons, r.Samples, fg)
}

func (r *CanvasRenderer) drawLines(lines [][]bezierstroke.Point, width float64,
	fg color.RGBA, round bool, samples int, tol float64) {
	if len(lines) == 0 {
		return
	}
	style := bezierstroke.StrokeStyle{
		Width:     width,
		Join:      bezierstroke.JoinBevel,
		Cap:       bezierstroke.CapButt,
		Tolerance: tol,
	}
	if round {
		style.Join = bezierstroke.JoinRound
		style.Cap = bezierstroke.CapRound
	}
	var polys []bezierstroke.Polygon
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		mesh := bezierstroke.StrokePolyline(line, style)
		polys = append(polys, mesh.Polygons...)
	}
	r.Canvas.FillPolygons(polys, samples, fg)
}

func (r *CanvasRenderer) drawVertices(vertices []bezierstroke.FlatVertex,
	radius float64, fg color.RGBA) {
	var polys []bezierstroke.Polygon
	for _, v := range vertices {
		p := v.Point
		polys = append(polys, bezierstroke.Polygon{
			{X: p.X, Y: p.Y - radius},
			{X: p.X + radius, Y: p.Y},
			{X: p.X, Y: p.Y + radius},
			{X: p.X - radius, Y: p.Y},
		})
	}
	r.Canvas.FillPolygons(polys, r.Samples, fg)
}

func (r *CanvasRenderer) drawEndpoints(fg color.RGBA) {
	for _, v := range []bezierstroke.FlatVertex{r.Exact[0], r.Exact[len(r.Exact)-1]} {
		r.drawVertices([]bezierstroke.FlatVertex{v}, 5, fg)
	}
}

func (r *CanvasRenderer) fillRects(rects [][4]float64, fg color.RGBA, samples int) {
	var polys []bezierstroke.Polygon
	for _, q := range rects {
		polys = append(polys, bezierstroke.Polygon{
			{X: q[0], Y: q[1]},
			{X: q[2], Y: q[1]},
			{X: q[2], Y: q[3]},
			{X: q[0], Y: q[3]},
		})
	}
	r.Canvas.FillPolygons(polys, samples, fg)
}

func closedOutline(p bezierstroke.Polygon) []bezierstroke.Point {
	if len(p) == 0 {
		return nil
	}
	out := make([]bezierstroke.Point, len(p)+1)
	copy(out, p)
	out[len(p)] = p[0]
	return out
}
