package stroke

import (
	"image/color"
	"math"
	"testing"
)

func TestOpenButtStroke(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: 0})
	p.LineTo(Point{X: 10, Y: 0})
	mesh := Stroke(*p, Style{Width: 2, Cap: CapButt})
	if got, want := meshArea(mesh), 20.0; math.Abs(got-want) > 1e-9 {
		t.Fatalf("area = %v, want %v", got, want)
	}
	b := mesh.Bounds()
	assertNear(t, b.Min.X, 0)
	assertNear(t, b.Max.X, 10)
	assertNear(t, b.Min.Y, -1)
	assertNear(t, b.Max.Y, 1)
	if pairs := DetectOverlaps(mesh, 1e-9); len(pairs) != 0 {
		t.Fatalf("unexpected overlaps: %v", pairs)
	}
}

func TestMiterLimitFallsBackToBevel(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: 0})
	p.LineTo(Point{X: 10, Y: 0})
	p.LineTo(Point{X: 5, Y: 8.660254037844387})

	unlimited := Stroke(*p, Style{
		Width: 2, Join: JoinMiter, Cap: CapButt, MiterLimit: 4,
	})
	limited := Stroke(*p, Style{
		Width: 2, Join: JoinMiter, Cap: CapButt, MiterLimit: 1.2,
	})
	if got := unlimited.Bounds().Max.X; got < 11.5 {
		t.Fatalf("expected a miter tip, max X = %v", got)
	}
	if got := limited.Bounds().Max.X; got > 11.2 {
		t.Fatalf("limited miter still extends too far, max X = %v", got)
	}
}

func TestRoundAndSquareCaps(t *testing.T) {
	round := lineMesh(CapRound, 10, 4)
	b := round.Bounds()
	assertNear(t, b.Min.X, -2)
	assertNear(t, b.Max.X, 12)
	if got, want := meshArea(round), 40+4*math.Pi; math.Abs(got-want) > 0.4 {
		t.Fatalf("round cap area = %v, want %v", got, want)
	}
	square := lineMesh(CapSquare, 10, 4)
	if got, want := meshArea(square), 56.0; math.Abs(got-want) > 1e-9 {
		t.Fatalf("square cap area = %v, want %v", got, want)
	}
}

func TestZeroLengthRoundCap(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 4, Y: -3})
	mesh := Stroke(*p, Style{Width: 4, Cap: CapRound})
	if got, want := meshArea(mesh), math.Pi*4; math.Abs(got-want) > 0.4 {
		t.Fatalf("dot area = %v, want %v", got, want)
	}
	b := mesh.Bounds()
	assertNear(t, b.Min.X, 2)
	assertNear(t, b.Max.X, 6)
	assertNear(t, b.Min.Y, -5)
	assertNear(t, b.Max.Y, -1)
}

func TestZeroLengthSubsegmentsDoNotCreateTriangles(t *testing.T) {
	base := NewPath()
	base.MoveTo(Point{X: 0, Y: 0})
	base.LineTo(Point{X: 10, Y: 0})

	dup := NewPath()
	dup.MoveTo(Point{X: 0, Y: 0})
	dup.LineTo(Point{X: 0, Y: 0})
	dup.LineTo(Point{X: 5, Y: 0})
	dup.LineTo(Point{X: 5, Y: 0})
	dup.LineTo(Point{X: 10, Y: 0})

	style := Style{Width: 2, Join: JoinMiter, Cap: CapRound}
	a := Stroke(*base, style)
	b := Stroke(*dup, style)
	if math.Abs(meshArea(a)-meshArea(b)) > 1e-9 {
		t.Fatalf("duplicate points changed area: %v vs %v", meshArea(a), meshArea(b))
	}
}

func TestZeroLengthDashedSubpath(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 4, Y: 5})
	on := Stroke(*p, Style{
		Width: 2, Cap: CapRound, Dashes: []float64{1, 1},
		DashOffset: 0,
	})
	if len(on.Triangles) == 0 {
		t.Fatal("on-phase zero-length subpath produced no dot")
	}
	off := Stroke(*p, Style{
		Width: 2, Cap: CapRound, Dashes: []float64{1, 1},
		DashOffset: 1,
	})
	if len(off.Triangles) != 0 {
		t.Fatalf("off-phase zero-length subpath produced %d triangles", len(off.Triangles))
	}
}

func TestDashIntervals(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: 0})
	p.LineTo(Point{X: 10, Y: 0})
	mesh := Stroke(*p, Style{
		Width: 2, Cap: CapButt, Dashes: []float64{2, 2},
	})
	if got, want := meshArea(mesh), 12.0; math.Abs(got-want) > 1e-9 {
		t.Fatalf("dash area = %v, want %v", got, want)
	}
	if got := len(mesh.Triangles); got != 6 {
		t.Fatalf("triangles = %d, want 6", got)
	}
}

func TestClosedDashMergesAcrossSeam(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: 0})
	p.LineTo(Point{X: 10, Y: 0})
	p.LineTo(Point{X: 10, Y: 10})
	p.LineTo(Point{X: 0, Y: 10})
	p.Close()
	mesh := Stroke(*p, Style{
		Width: 1, Join: JoinMiter, Cap: CapButt,
		Dashes: []float64{6, 2}, DashOffset: 1,
	})
	if len(mesh.Triangles) == 0 {
		t.Fatal("closed dashed path produced no triangles")
	}
}

func TestClosedDashCoveringWholeContourStaysClosed(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: 0})
	p.LineTo(Point{X: 10, Y: 0})
	p.LineTo(Point{X: 10, Y: 10})
	p.LineTo(Point{X: 0, Y: 10})
	p.Close()
	base := Style{Width: 2, Join: JoinMiter, Cap: CapRound}
	solid := Stroke(*p, base)
	dashedStyle := base
	dashedStyle.Dashes = []float64{100, 2}
	dashed := Stroke(*p, dashedStyle)
	if len(solid.Triangles) != len(dashed.Triangles) {
		t.Fatalf("solid %d triangles, full dash %d triangles",
			len(solid.Triangles), len(dashed.Triangles))
	}
	if math.Abs(meshArea(solid)-meshArea(dashed)) > 1e-9 {
		t.Fatalf("solid area %v, full dash area %v", meshArea(solid), meshArea(dashed))
	}
}

func TestShortDashesAtRoundCornersHaveNoOverlap(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 44, Y: 286})
	for i := 0; i < 8; i++ {
		x := 44.0 + float64(i)*62
		y := 218.0
		if i%2 == 1 {
			y = 286
		}
		p.LineTo(Point{X: x, Y: y})
	}
	mesh := Stroke(*p, Style{
		Width: 12, Join: JoinRound, Cap: CapRound,
		Dashes: []float64{18, 20}, DashOffset: 6,
	})
	if pairs := DetectOverlaps(mesh, 1e-9); len(pairs) != 0 {
		t.Fatalf("short dashes have %d overlapping pairs", len(pairs))
	}
}

func TestSharpTurnIsCappedInsteadOfSpiking(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: 0})
	p.LineTo(Point{X: 100, Y: 0})
	p.LineTo(Point{X: 0.1, Y: 0.2})
	mesh := Stroke(*p, Style{
		Width: 4, Join: JoinMiter, Cap: CapRound, MiterLimit: 100,
	})
	b := mesh.Bounds()
	if b.Max.X > 102.001 {
		t.Fatalf("sharp miter escaped expected bounds: %v", b.Max.X)
	}
}

func TestAcuteMiterJoinHasNoTriangleOverlap(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: 0})
	p.LineTo(Point{X: 70, Y: 146})
	p.LineTo(Point{X: 146, Y: -8})
	p.LineTo(Point{X: 222, Y: 146})
	p.LineTo(Point{X: 294, Y: -8})
	p.LineTo(Point{X: 356, Y: 114})
	mesh := Stroke(*p, Style{
		Width: 15, Join: JoinMiter, Cap: CapRound, MiterLimit: 4,
	})
	if pairs := DetectOverlaps(mesh, 1e-9); len(pairs) != 0 {
		t.Fatalf("acute miter has %d overlapping pairs", len(pairs))
	}
}

func TestTransformedClosedRoundJoinHasNoOverlap(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: -34})
	p.LineTo(Point{X: 28, Y: 0})
	p.LineTo(Point{X: 0, Y: 34})
	p.LineTo(Point{X: -28, Y: 0})
	p.Close()
	mesh := StrokeTransformed(*p, Style{
		Width: 8, Join: JoinRound, Cap: CapRound,
	}, Affine{A: 1.45, D: 0.55}, WidthFollowTransform)
	if pairs := DetectOverlaps(mesh, 1e-9); len(pairs) != 0 {
		t.Fatalf("transformed diamond has %d overlapping pairs", len(pairs))
	}
}

func TestNonUniformTransformWidthPolicies(t *testing.T) {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: 0})
	p.LineTo(Point{X: 10, Y: 0})
	a := Affine{A: 3, D: 0.5}
	style := Style{Width: 2, Cap: CapButt}

	directional := StrokeTransformed(*p, style, a, WidthFollowTransform).Bounds()
	assertNear(t, directional.Max.X, 30)
	assertNear(t, directional.Max.Y, 0.5)
	assertNear(t, directional.Min.Y, -0.5)

	byY := StrokeTransformed(*p, style, a, WidthByYScale).Bounds()
	assertNear(t, byY.Max.Y, 0.5)
	assertNear(t, byY.Min.Y, -0.5)

	byX := StrokeTransformed(*p, style, a, WidthByXScale).Bounds()
	assertNear(t, byX.Max.Y, 3)
	assertNear(t, byX.Min.Y, -3)

	geom := StrokeTransformed(*p, style, a, WidthByGeometricScale).Bounds()
	assertNear(t, geom.Max.Y, math.Sqrt(1.5))
	assertNear(t, geom.Min.Y, -math.Sqrt(1.5))
}

func TestRasterModes(t *testing.T) {
	mesh := Mesh{Triangles: []Triangle{{
		A: Point{X: 2, Y: 2},
		B: Point{X: 18, Y: 2},
		C: Point{X: 2, Y: 18},
	}}}
	opts := RasterOptions{
		Width: 20, Height: 20, Samples: 4,
		Background: colorWhite,
		Foreground: colorBlack,
		Overlap:    colorRed,
	}
	fill := RenderFill(mesh, opts)
	aa := RenderAntiAliased(mesh, opts)
	if fill == nil || aa == nil {
		t.Fatal("nil raster output")
	}
	if got := fill.NRGBAAt(5, 5); got != colorBlack {
		t.Fatalf("fill center = %v", got)
	}
	if got := aa.NRGBAAt(5, 5); got != colorBlack {
		t.Fatalf("AA center = %v", got)
	}
	edge := aa.NRGBAAt(16, 3)
	if edge.A != 255 || edge.R == colorWhite.R || edge.R == colorBlack.R {
		t.Fatalf("expected a blended AA edge, got %v", edge)
	}
}

func TestOverlapDetection(t *testing.T) {
	a := Triangle{
		A: Point{X: 0, Y: 0},
		B: Point{X: 10, Y: 0},
		C: Point{X: 0, Y: 10},
	}
	b := Triangle{
		A: Point{X: 1, Y: 1},
		B: Point{X: 9, Y: 1},
		C: Point{X: 1, Y: 9},
	}
	mesh := Mesh{Triangles: []Triangle{a, b}}
	if pairs := DetectOverlaps(mesh, 1e-9); len(pairs) != 1 {
		t.Fatalf("pairs = %v, want 1", pairs)
	}
	shared := Mesh{Triangles: []Triangle{
		{A: Point{X: 0, Y: 0}, B: Point{X: 10, Y: 0}, C: Point{X: 0, Y: 10}},
		{A: Point{X: 10, Y: 0}, B: Point{X: 10, Y: 10}, C: Point{X: 0, Y: 10}},
	}}
	if pairs := DetectOverlaps(shared, 1e-9); len(pairs) != 0 {
		t.Fatalf("shared edge reported as overlap: %v", pairs)
	}
}

func TestOverlapHighlight(t *testing.T) {
	mesh := Mesh{Triangles: []Triangle{
		{A: Point{X: 2, Y: 2}, B: Point{X: 18, Y: 2}, C: Point{X: 2, Y: 18}},
		{A: Point{X: 6, Y: 6}, B: Point{X: 6, Y: 16}, C: Point{X: 16, Y: 6}},
	}}
	opts := RasterOptions{
		Width: 20, Height: 20, Samples: 4,
		Background: colorWhite,
		Foreground: colorBlack,
		Overlap:    colorRed,
	}
	img := RenderOverlap(mesh, opts)
	if got := img.NRGBAAt(7, 7); got != colorRed {
		t.Fatalf("overlap center = %v, want red", got)
	}
}

func lineMesh(cap CapStyle, length, width float64) Mesh {
	p := NewPath()
	p.MoveTo(Point{X: 0, Y: 0})
	p.LineTo(Point{X: length, Y: 0})
	return Stroke(*p, Style{Width: width, Cap: cap})
}

func meshArea(m Mesh) float64 {
	var area float64
	for _, t := range m.Triangles {
		area += math.Abs(signedArea(t.A, t.B, t.C)) / 2
	}
	return area
}

func assertNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-7 {
		t.Fatalf("got %v, want %v", got, want)
	}
}

var (
	colorWhite = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colorBlack = color.NRGBA{A: 255}
	colorRed   = color.NRGBA{R: 255, A: 255}
)
