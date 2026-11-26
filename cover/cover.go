// Package cover rasterizes colored vector layers to per-pixel coverage
// buffers using two honest backends: exact analytic box-filter area and
// fixed N x N point supersampling. It also contains the deliberately
// blur-based baseline used to demonstrate what "covering errors with blur"
// does to nearby contours.
package cover

import (
	"math"
	"strconv"

	"coverlab/geom"
)

// RGB is a straight color in 0..1 range.
type RGB struct{ R, G, B float64 }

// Layer is one independently colored contour footprint (a simple polygon).
type Layer struct {
	Name  string
	Poly  []geom.Vec
	Color RGB
}

// Scene is a sized pixel canvas with ordered source-over layers.
type Scene struct {
	Name   string
	W, H   int
	Layers []Layer
	// Crop is the close-up rectangle in pixel coordinates: [X0,X1)x[Y0,Y1).
	Crop [4]int
}

// Index returns the buffer index for pixel (x,y).
func (s Scene) Index(x, y int) int { return y*s.W + x }

// AnalyticLayer computes exact polygon/pixel intersection coverage.
func AnalyticLayer(layer Layer, w, h int) []float64 {
	cov := make([]float64, w*h)
	bb := geom.PolyBBox(layer.Poly)
	x0 := clampInt(int(math.Floor(bb.MinX)), 0, w-1)
	x1 := clampInt(int(math.Ceil(bb.MaxX))-1, 0, w-1)
	y0 := clampInt(int(math.Floor(bb.MinY)), 0, h-1)
	y1 := clampInt(int(math.Ceil(bb.MaxY))-1, 0, h-1)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			cov[y*w+x] = geom.PixelCoverage(layer.Poly, x, y)
		}
	}
	return cov
}

// SampledLayer computes fixed N x N regular-grid supersampled coverage.
// N is the per-axis sample multiplier (N=2 -> 4 samples, N=3 -> 9).
func SampledLayer(layer Layer, w, h, n int) []float64 {
	cov := make([]float64, w*h)
	if n < 1 {
		n = 1
	}
	inv := 1.0 / float64(n)
	total := float64(n * n)
	bb := geom.PolyBBox(layer.Poly)
	x0 := clampInt(int(math.Floor(bb.MinX)), 0, w-1)
	x1 := clampInt(int(math.Ceil(bb.MaxX))-1, 0, w-1)
	y0 := clampInt(int(math.Floor(bb.MinY)), 0, h-1)
	y1 := clampInt(int(math.Ceil(bb.MaxY))-1, 0, h-1)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			hits := 0
			for sy := 0; sy < n; sy++ {
				py := float64(y) + (float64(sy)+0.5)*inv
				for sx := 0; sx < n; sx++ {
					px := float64(x) + (float64(sx)+0.5)*inv
					if geom.PointInPoly(geom.Vec{X: px, Y: py}, layer.Poly) {
						hits++
					}
				}
			}
			cov[y*w+x] = float64(hits) / total
		}
	}
	return cov
}

// EffectiveAlpha combines per-layer coverages into the union alpha seen by
// the background: alpha = 1 - prod(1-c_i).
func EffectiveAlpha(covs [][]float64) []float64 {
	if len(covs) == 0 {
		return nil
	}
	out := make([]float64, len(covs[0]))
	for i := range out {
		a := 1.0
		for _, c := range covs {
			a *= 1.0 - c[i]
		}
		out[i] = 1.0 - a
	}
	return out
}

// Composite source-overs the colored coverage layers over an opaque bg.
// covs aligns with scene.Layers. Returned RGB values are in 0..255.
func Composite(scene Scene, covs [][]float64, bg RGB) []RGB {
	out := make([]RGB, scene.W*scene.H)
	for i := range out {
		cur := bg
		for k, layer := range scene.Layers {
			a := covs[k][i]
			c := layer.Color
			cur = RGB{
				R: a*c.R + (1-a)*cur.R,
				G: a*c.G + (1-a)*cur.G,
				B: a*c.B + (1-a)*cur.B,
			}
		}
		out[i] = RGB{R: cur.R * 255, G: cur.G * 255, B: cur.B * 255}
	}
	return out
}

// RenderMethod runs the requested backend for every layer and returns both
// the composited 8-bit-ish RGB buffer and the effective alpha buffer.
// method is "analytic" or "ssN".
func RenderMethod(scene Scene, method string, bg RGB) (rgb []RGB, alpha []float64, covs [][]float64) {
	covs = make([][]float64, len(scene.Layers))
	for k, layer := range scene.Layers {
		if method == "analytic" {
			covs[k] = AnalyticLayer(layer, scene.W, scene.H)
		} else if n := parseSS(method); n > 0 {
			covs[k] = SampledLayer(layer, scene.W, scene.H, n)
		} else {
			panic("cover: unknown method " + method)
		}
	}
	alpha = EffectiveAlpha(covs)
	rgb = Composite(scene, covs, bg)
	return rgb, alpha, covs
}

func parseSS(method string) int {
	if len(method) < 3 || method[:2] != "ss" {
		return 0
	}
	n, err := strconv.Atoi(method[2:])
	if err != nil || n < 1 {
		return 0
	}
	return n
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }
