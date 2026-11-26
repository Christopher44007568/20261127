package main

import (
	"math"
	"sort"
)

// CurvatureField contains terrain shape measures and the normalized boundary
// attraction score. Feature is large on high-curvature and concave/depression
// cells.
type CurvatureField struct {
	Laplacian []float64
	Mean      []float64
	Magnitude []float64
	Concavity []float64
	Feature   []float64
	TangentX  []float64
	TangentY  []float64
	// HorizontalEdge/VerticalEdge score boundary edges by both feature
	// strength and alignment with the local feature-line tangent.
	HorizontalEdge []float64
	VerticalEdge   []float64
}

func ComputeCurvature(g *Grid) *CurvatureField {
	n := g.W * g.H
	f := &CurvatureField{
		Laplacian:      make([]float64, n),
		Mean:           make([]float64, n),
		Magnitude:      make([]float64, n),
		Concavity:      make([]float64, n),
		Feature:        make([]float64, n),
		TangentX:       make([]float64, n),
		TangentY:       make([]float64, n),
		HorizontalEdge: make([]float64, n),
		VerticalEdge:   make([]float64, n),
	}

	cs := g.CellSize
	if cs <= 0 {
		cs = 1
	}

	raw := make([]float64, 0, g.OpenCount())
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				continue
			}
			zc := g.Elevation[id]
			zx1 := g.ClampedElevation(x+1, y)
			zx0 := g.ClampedElevation(x-1, y)
			zy1 := g.ClampedElevation(x, y+1)
			zy0 := g.ClampedElevation(x, y-1)

			hx := (zx1 - zx0) / (2 * cs)
			hy := (zy1 - zy0) / (2 * cs)
			hxx := (zx1 - 2*zc + zx0) / (cs * cs)
			hyy := (zy1 - 2*zc + zy0) / (cs * cs)
			hxy := (g.ClampedElevation(x+1, y+1) - g.ClampedElevation(x+1, y-1) -
				g.ClampedElevation(x-1, y+1) + g.ClampedElevation(x-1, y-1)) / (4 * cs * cs)

			lap := hxx + hyy
			grad2 := hx*hx + hy*hy
			mean := ((1+hy*hy)*hxx - 2*hx*hy*hxy + (1+hx*hx)*hyy) /
				(2 * math.Pow(1+grad2, 1.5))
			magnitude := math.Sqrt(hxx*hxx+hyy*hyy+2*hxy*hxy) / (1 + grad2)
			concavity := math.Max(0, mean) + 0.35*math.Max(0, lap)

			f.Laplacian[id] = lap
			f.Mean[id] = mean
			f.Magnitude[id] = magnitude
			f.Concavity[id] = concavity

			score := magnitude + 1.35*concavity
			f.Feature[id] = score
			tx, ty := featureTangent(hxx, hyy, hxy)
			f.TangentX[id] = tx
			f.TangentY[id] = ty
			raw = append(raw, score)
		}
	}

	scale := robustScale(raw)
	for id := range f.Feature {
		if g.Blocked[id] {
			f.Feature[id] = 0
			continue
		}
		v := math.Max(0, f.Feature[id]/scale)
		if v > 1 {
			v = 1
		}
		// Gamma broadens the narrow cores of curvature lines.
		f.Feature[id] = math.Sqrt(v)
		f.HorizontalEdge[id] = f.Feature[id] * math.Abs(f.TangentX[id])
		f.VerticalEdge[id] = f.Feature[id] * math.Abs(f.TangentY[id])
	}
	return f
}

func featureTangent(hxx, hyy, hxy float64) (float64, float64) {
	lambda1 := (hxx + hyy) / 2
	delta := math.Sqrt(((hxx-hyy)/2)*(hxx-hyy)/2 + hxy*hxy)
	e1 := lambda1 + delta
	e2 := lambda1 - delta
	lambda := e1
	if math.Abs(e2) > math.Abs(e1) {
		lambda = e2
	}

	var dx, dy float64
	if math.Abs(hxy) < 1e-12 {
		if math.Abs(hxx) >= math.Abs(hyy) {
			dx, dy = 1, 0
		} else {
			dx, dy = 0, 1
		}
	} else {
		dx, dy = hxy, lambda-hxx
		norm := math.Hypot(dx, dy)
		if norm < 1e-12 {
			dx, dy = 1, 0
		} else {
			dx /= norm
			dy /= norm
		}
	}
	// Rotate the principal-curvature normal by 90 degrees to obtain the
	// tangent of a valley, ridge, or curvature contour.
	return -dy, dx
}

func robustScale(values []float64) float64 {
	if len(values) == 0 {
		return 1
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	p95 := sorted[int(0.95*float64(len(sorted)-1))]
	max := sorted[len(sorted)-1]
	if p95 > 1e-12 {
		return p95
	}
	if max > 1e-12 {
		return max
	}
	return 1
}

func edgeFeature(g *Grid, feature []float64, x, y, nx, ny int) float64 {
	if feature == nil {
		return 0
	}
	return 0.5 * (feature[g.ID(x, y)] + feature[g.ID(nx, ny)])
}

func orientedEdgeFeature(g *Grid, f *CurvatureField, x, y, nx, ny int) float64 {
	if f == nil {
		return 0
	}
	id1 := g.ID(x, y)
	id2 := g.ID(nx, ny)
	if y == ny {
		// Horizontal cell adjacency creates a vertical boundary edge.
		return 0.5 * (f.VerticalEdge[id1] + f.VerticalEdge[id2])
	}
	// Vertical cell adjacency creates a horizontal boundary edge.
	return 0.5 * (f.HorizontalEdge[id1] + f.HorizontalEdge[id2])
}
