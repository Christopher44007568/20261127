package main

import (
	"math"

	"example.com/bezierstroke"
)

const detailTolerance = 0.02

type PreparedScene struct {
	Scene
	ScreenCurve bezierstroke.Cubic
	Exact       []bezierstroke.FlatVertex
	Adaptive    []bezierstroke.FlatVertex
	Mesh        bezierstroke.StrokeMesh
	Metrics     SceneMetrics
}

func prepareScene(scene Scene) PreparedScene {
	screen := bezierstroke.TransformCubic(scene.Curve, func(p bezierstroke.Point) bezierstroke.Point {
		return scene.toScreen(p)
	})
	exact := screen.AdaptiveFlattenDetailed(detailTolerance)
	adaptive := screen.AdaptiveFlattenDetailed(scene.Tolerance)
	mesh := bezierstroke.StrokePolyline(adaptivePoints(adaptive), bezierstroke.StrokeStyle{
		Width:      scene.StrokeWidth,
		Join:       bezierstroke.JoinRound,
		Cap:        bezierstroke.CapRound,
		Tolerance:  0.06,
		MiterLimit: 4,
	})

	maxError := maxParamError(screen, adaptive)
	fixed8 := maxFixedError(screen, 8)
	return PreparedScene{
		Scene:       scene,
		ScreenCurve: screen,
		Exact:       exact,
		Adaptive:    adaptive,
		Mesh:        mesh,
		Metrics: SceneMetrics{
			Key:              scene.Key,
			Title:            scene.Title,
			Curve:            [4]bezierstroke.Point{scene.Curve.P0, scene.Curve.P1, scene.Curve.P2, scene.Curve.P3},
			Viewport:         map[string]interface{}{"width": scene.W, "height": scene.H, "scale_px_per_unit": scene.Scale, "center": scene.Center},
			TolerancePx:      scene.Tolerance,
			StrokeWidthPx:    scene.StrokeWidth,
			CurveLengthPx:    curveLength(screen),
			AdaptiveVertices: len(adaptive),
			AdaptiveSegments: maxInt(0, len(adaptive)-1),
			MaxChordErrorPx:  maxError,
			Fixed8MaxErrorPx: fixed8,
			MeshPolygons:     len(mesh.Polygons),
		},
	}
}

func (s Scene) toScreen(p bezierstroke.Point) bezierstroke.Point {
	return bezierstroke.Point{
		X: float64(s.W)/2 + (p.X-s.Center.X)*s.Scale,
		Y: float64(s.H)/2 - (p.Y-s.Center.Y)*s.Scale,
	}
}

func adaptivePoints(v []bezierstroke.FlatVertex) []bezierstroke.Point {
	out := make([]bezierstroke.Point, len(v))
	for i := range v {
		out[i] = v[i].Point
	}
	return out
}

func maxParamError(c bezierstroke.Cubic, vertices []bezierstroke.FlatVertex) float64 {
	maxErr := 0.0
	for i := 0; i+1 < len(vertices); i++ {
		seg := c.Segment(vertices[i].T, vertices[i+1].T)
		maxErr = math.Max(maxErr, finite(seg.ChordHeight()))
	}
	return maxErr
}

func maxFixedError(c bezierstroke.Cubic, n int) float64 {
	maxErr := 0.0
	for i := 0; i < n; i++ {
		t0 := float64(i) / float64(n)
		t1 := float64(i+1) / float64(n)
		seg := c.Segment(t0, t1)
		maxErr = math.Max(maxErr, finite(seg.ChordHeight()))
	}
	return maxErr
}

func curveLength(c bezierstroke.Cubic) float64 {
	const n = 4096
	length := 0.0
	prev := c.Eval(0)
	for i := 1; i <= n; i++ {
		next := c.Eval(float64(i) / float64(n))
		length += prev.Distance(next)
		prev = next
	}
	return length
}

func (p PreparedScene) metrics(paths map[string]string) SceneMetrics {
	m := p.Metrics
	m.OutputOriginal, _ = paths["original"]
	m.OutputMesh, _ = paths["mesh"]
	m.OutputFinal, _ = paths["final"]
	return m
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
