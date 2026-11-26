package stroke

import (
	"image"
	"image/color"
	"math"
)

// RasterOptions maps user-space triangles to an image. A zero Transform means
// Identity. Samples is the per-axis AA sample count (4 by default, clamped to
// 1..8).
type RasterOptions struct {
	Width, Height int
	Transform     Affine
	Samples       int
	Background    color.NRGBA
	Foreground    color.NRGBA
	Overlap       color.NRGBA
}

// TrianglePair identifies two triangles whose interiors overlap with positive
// area.
type TrianglePair struct {
	A, B int
}

// Diagnostics contains the three requested renderings and the geometric
// overlap report used to produce the error highlight.
type Diagnostics struct {
	Fill         *image.NRGBA
	AntiAliased  *image.NRGBA
	Overlaps     *image.NRGBA
	OverlapPairs []TrianglePair
}

// RenderDiagnostics renders the fill, AA, and overlap-highlight images.
func RenderDiagnostics(mesh Mesh, opts RasterOptions) Diagnostics {
	pairs := DetectOverlaps(mesh, 1e-9)
	return Diagnostics{
		Fill:         renderMesh(mesh, opts, renderFillMode),
		AntiAliased:  renderMesh(mesh, opts, renderAAMode),
		Overlaps:     renderOverlap(mesh, opts, pairs),
		OverlapPairs: pairs,
	}
}

// RenderFill returns a non-antialiased solid fill, testing pixel centers.
func RenderFill(mesh Mesh, opts RasterOptions) *image.NRGBA {
	return renderMesh(mesh, opts, renderFillMode)
}

// RenderAntiAliased returns a coverage-antialiased solid fill.
func RenderAntiAliased(mesh Mesh, opts RasterOptions) *image.NRGBA {
	return renderMesh(mesh, opts, renderAAMode)
}

// RenderOverlap returns the fill in Foreground with positive-area triangle
// intersections in Overlap color.
func RenderOverlap(mesh Mesh, opts RasterOptions) *image.NRGBA {
	return renderOverlap(mesh, opts, DetectOverlaps(mesh, 1e-9))
}

type renderMode uint8

const (
	renderFillMode renderMode = iota
	renderAAMode
)

func renderMesh(mesh Mesh, opts RasterOptions, mode renderMode) *image.NRGBA {
	if opts.Width <= 0 || opts.Height <= 0 {
		return nil
	}
	opts = normalizeRasterOptions(opts)
	img := image.NewNRGBA(image.Rect(0, 0, opts.Width, opts.Height))
	for y := 0; y < opts.Height; y++ {
		for x := 0; x < opts.Width; x++ {
			setPixel(img, x, y, opts.Background)
		}
	}
	tris := transformedTriangles(mesh, opts.Transform)
	if mode == renderFillMode {
		for _, t := range tris {
			rasterFillTriangle(img, t, opts.Foreground)
		}
		return img
	}
	counts, rate := coverageCounts(tris, opts.Width, opts.Height, opts.Samples)
	n := rate * rate
	for y := 0; y < opts.Height; y++ {
		for x := 0; x < opts.Width; x++ {
			covered := 0
			base := (y*opts.Width + x) * n
			for i := 0; i < n; i++ {
				if counts[base+i] != 0 {
					covered++
				}
			}
			if covered == 0 {
				continue
			}
			setPixel(img, x, y, blend(opts.Background, opts.Foreground,
				float64(covered)/float64(n)))
		}
	}
	return img
}

func renderOverlap(mesh Mesh, opts RasterOptions, pairs []TrianglePair) *image.NRGBA {
	if opts.Width <= 0 || opts.Height <= 0 {
		return nil
	}
	opts = normalizeRasterOptions(opts)
	img := image.NewNRGBA(image.Rect(0, 0, opts.Width, opts.Height))
	for y := 0; y < opts.Height; y++ {
		for x := 0; x < opts.Width; x++ {
			setPixel(img, x, y, opts.Background)
		}
	}
	tris := transformedTriangles(mesh, opts.Transform)
	counts, rate := coverageCounts(tris, opts.Width, opts.Height, opts.Samples)
	n := rate * rate
	var overlapTris []Triangle
	if len(pairs) > 0 {
		overlapTris = transformedTriangles(
			Mesh{Triangles: overlapIntersectionTriangles(mesh, pairs)},
			opts.Transform,
		)
	}
	var overlapCounts []uint8
	if len(overlapTris) > 0 {
		overlapCounts, _ = coverageCounts(overlapTris, opts.Width, opts.Height, opts.Samples)
	}
	for y := 0; y < opts.Height; y++ {
		for x := 0; x < opts.Width; x++ {
			covered := 0
			base := (y*opts.Width + x) * n
			for i := 0; i < n; i++ {
				if counts[base+i] != 0 {
					covered++
				}
			}
			if covered == 0 {
				continue
			}
			c := blend(opts.Background, opts.Foreground,
				float64(covered)/float64(n))
			if len(overlapCounts) > 0 {
				overlapping := 0
				for i := 0; i < n; i++ {
					if overlapCounts[base+i] != 0 {
						overlapping++
					}
				}
				if overlapping > 0 {
					c = blend(c, opts.Overlap, float64(overlapping)/float64(n))
				}
			}
			setPixel(img, x, y, c)
		}
	}
	return img
}

func normalizeRasterOptions(opts RasterOptions) RasterOptions {
	if opts.Transform == (Affine{}) {
		opts.Transform = Identity()
	}
	if opts.Samples <= 0 {
		opts.Samples = 4
	}
	if opts.Samples > 8 {
		opts.Samples = 8
	}
	if opts.Overlap == (color.NRGBA{}) {
		opts.Overlap = color.NRGBA{R: 255, A: 255}
	}
	return opts
}

func transformedTriangles(mesh Mesh, a Affine) []Triangle {
	out := make([]Triangle, len(mesh.Triangles))
	for i, t := range mesh.Triangles {
		out[i] = Triangle{
			A: a.Apply(t.A),
			B: a.Apply(t.B),
			C: a.Apply(t.C),
		}
	}
	return out
}

func rasterFillTriangle(img *image.NRGBA, t Triangle, c color.NRGBA) {
	minX, minY, maxX, maxY := triangleBounds(t, img.Bounds().Dx(), img.Bounds().Dy())
	for y := minY; y < maxY; y++ {
		py := float64(y) + 0.5
		for x := minX; x < maxX; x++ {
			px := float64(x) + 0.5
			if pointInTriangle(Point{X: px, Y: py}, t.A, t.B, t.C) {
				setPixel(img, x, y, c)
			}
		}
	}
}

func coverageCounts(tris []Triangle, width, height, rate int) ([]uint8, int) {
	if rate <= 0 {
		rate = 4
	}
	if rate > 8 {
		rate = 8
	}
	n := rate * rate
	counts := make([]uint8, width*height*n)
	for _, t := range tris {
		minX, minY, maxX, maxY := triangleBounds(t, width, height)
		for y := minY; y < maxY; y++ {
			for x := minX; x < maxX; x++ {
				base := (y*width + x) * n
				for sy := 0; sy < rate; sy++ {
					py := float64(y) + (float64(sy)+0.5)/float64(rate)
					for sx := 0; sx < rate; sx++ {
						px := float64(x) + (float64(sx)+0.5)/float64(rate)
						if !pointInTriangle(Point{X: px, Y: py}, t.A, t.B, t.C) {
							continue
						}
						i := base + sy*rate + sx
						if counts[i] < 2 {
							counts[i]++
						}
					}
				}
			}
		}
	}
	return counts, rate
}

func triangleBounds(t Triangle, width, height int) (int, int, int, int) {
	minX := floorInt(math.Min(t.A.X, math.Min(t.B.X, t.C.X)))
	minY := floorInt(math.Min(t.A.Y, math.Min(t.B.Y, t.C.Y)))
	maxX := ceilInt(math.Max(t.A.X, math.Max(t.B.X, t.C.X))) + 1
	maxY := ceilInt(math.Max(t.A.Y, math.Max(t.B.Y, t.C.Y))) + 1
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX > width {
		maxX = width
	}
	if maxY > height {
		maxY = height
	}
	return minX, minY, maxX, maxY
}

func pointInTriangle(p, a, b, c Point) bool {
	d1 := cross(sub(b, a), sub(p, a))
	d2 := cross(sub(c, b), sub(p, b))
	d3 := cross(sub(a, c), sub(p, c))
	hasNeg := d1 < 0 || d2 < 0 || d3 < 0
	hasPos := d1 > 0 || d2 > 0 || d3 > 0
	return !(hasNeg && hasPos)
}

func setPixel(img *image.NRGBA, x, y int, c color.NRGBA) {
	i := img.PixOffset(x, y)
	img.Pix[i] = c.R
	img.Pix[i+1] = c.G
	img.Pix[i+2] = c.B
	img.Pix[i+3] = c.A
}

func blend(bg, fg color.NRGBA, alpha float64) color.NRGBA {
	if alpha <= 0 {
		return bg
	}
	if alpha >= 1 {
		return fg
	}
	fa := alpha * float64(fg.A) / 255
	ba := float64(bg.A) / 255
	oa := fa + ba*(1-fa)
	if oa <= 0 {
		return color.NRGBA{}
	}
	mix := func(f, b uint8) uint8 {
		v := (float64(f)*fa + float64(b)*ba*(1-fa)) / oa
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint8(math.Round(v))
	}
	return color.NRGBA{
		R: mix(fg.R, bg.R),
		G: mix(fg.G, bg.G),
		B: mix(fg.B, bg.B),
		A: uint8(math.Round(oa * 255)),
	}
}
