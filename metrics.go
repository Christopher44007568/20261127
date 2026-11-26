package main

import (
	"math"
	"sort"
)

type ErrorStats struct {
	Count  int     `json:"count"`
	Mean   float64 `json:"mean_m"`
	RMSE   float64 `json:"rmse_m"`
	Median float64 `json:"median_m"`
	P95    float64 `json:"p95_m"`
	Max    float64 `json:"max_m"`
}

func StatsFromValues(values []float64) ErrorStats {
	if len(values) == 0 {
		return ErrorStats{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	sum := 0.0
	sq := 0.0
	for _, v := range values {
		sum += v
		sq += v * v
	}
	idx := func(q float64) float64 {
		pos := q * float64(len(sorted)-1)
		i := int(pos)
		if i >= len(sorted)-1 {
			return sorted[len(sorted)-1]
		}
		t := pos - float64(i)
		return sorted[i]*(1-t) + sorted[i+1]*t
	}
	return ErrorStats{
		Count:  len(values),
		Mean:   sum / float64(len(values)),
		RMSE:   math.Sqrt(sq / float64(len(values))),
		Median: idx(0.5),
		P95:    idx(0.95),
		Max:    sorted[len(sorted)-1],
	}
}

func DistanceStats(points []Vec3, geom Geometry) ErrorStats {
	values := make([]float64, 0, len(points))
	for _, p := range points {
		values = append(values, math.Abs(geom.SDF(p)))
	}
	return StatsFromValues(values)
}

func BuildRawCloud(frames []*DepthFrame, pixelStep int) []Vec3 {
	points := make([]Vec3, 0, 20000)
	if pixelStep <= 1 {
		pixelStep = 2
	}
	for _, f := range frames {
		for y := 0; y < f.Height; y += pixelStep {
			for x := 0; x < f.Width; x += pixelStep {
				z := f.Depth[y*f.Width+x]
				if z <= 0 {
					continue
				}
				points = append(points, f.Fusion.UnprojectDepth(x, y, z))
			}
		}
	}
	return points
}

func SegmentPoints(segments []Segment3) []Vec3 {
	points := make([]Vec3, 0, len(segments)*3)
	for _, s := range segments {
		points = append(points, s.A, s.B, s.A.Lerp(s.B, 0.5))
	}
	return points
}

func ScanYZeroCrossings(g *VoxelGrid, ix, iz int) []float64 {
	var ys []float64
	for iy := 0; iy < g.NY-1; iy++ {
		id0 := g.Index(ix, iy, iz)
		id1 := g.Index(ix, iy+1, iz)
		a := g.Value[id0]
		b := g.Value[id1]
		if g.Weight[id0] < 0.12 || g.Weight[id1] < 0.12 {
			continue
		}
		if (a < 0) == (b < 0) {
			continue
		}
		p0 := g.Center(ix, iy, iz)
		p1 := g.Center(ix, iy+1, iz)
		ys = append(ys, interpolateZero(p0, p1, a, b).Y)
	}
	return ys
}

type ThinCheck struct {
	SampleX           []float64 `json:"sample_x"`
	ThicknessValues   []float64 `json:"thickness_values_m"`
	MeanThickness     float64   `json:"mean_thickness_m"`
	ThicknessError    float64   `json:"thickness_error_m"`
	ExpectedThickness float64   `json:"expected_thickness_m"`
}

func MeasureThinSheet(g *VoxelGrid, iz int, expected float64) ThinCheck {
	check := ThinCheck{ExpectedThickness: expected}
	for _, x := range []float64{-0.50, -0.25, 0, 0.25, 0.50} {
		ix := int((x-g.Min.X)/g.Voxel - 0.5)
		ix = clampInt(ix, 1, g.NX-2)
		crossings := ScanYZeroCrossings(g, ix, iz)
		if len(crossings) < 2 {
			continue
		}
		sort.Float64s(crossings)
		check.SampleX = append(check.SampleX, g.Center(ix, 0, iz).X)
		thickness := crossings[len(crossings)-1] - crossings[0]
		check.ThicknessValues = append(check.ThicknessValues, thickness)
	}
	if len(check.ThicknessValues) > 0 {
		sum := 0.0
		for _, v := range check.ThicknessValues {
			sum += v
		}
		check.MeanThickness = sum / float64(len(check.ThicknessValues))
		check.ThicknessError = math.Abs(check.MeanThickness - expected)
	}
	return check
}

type CornerCheck struct {
	Expected       Vec3    `json:"expected"`
	Estimated      Vec3    `json:"estimated"`
	Error          float64 `json:"error_m"`
	EmptyProbe     Vec3    `json:"empty_probe"`
	ProbeMetricSDF float64 `json:"probe_metric_sdf_m"`
	ProbeTSDF      float64 `json:"probe_tsdf"`
}

func fitIntercept(points []Vec3, horizontal bool) (slope, intercept float64, ok bool) {
	if len(points) < 8 {
		return 0, 0, false
	}
	sx, sy, sxx, sxy := 0.0, 0.0, 0.0, 0.0
	n := float64(len(points))
	for _, p := range points {
		var x, y float64
		if horizontal {
			x, y = p.X, p.Y
		} else {
			x, y = p.Y, p.X
		}
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
	}
	den := n*sxx - sx*sx
	if math.Abs(den) < 1e-9 {
		return 0, 0, false
	}
	slope = (n*sxy - sx*sy) / den
	intercept = (sy - slope*sx) / n
	return slope, intercept, true
}

func MeasureCorner(g *VoxelGrid, segments []Segment3, iz int, expected, probe Vec3) CornerCheck {
	check := CornerCheck{Expected: expected, EmptyProbe: probe}
	var horizontalPts, verticalPts []Vec3
	for _, s := range segments {
		m := s.A.Lerp(s.B, 0.5)
		if m.X > -0.34 && m.X < 0.58 && m.Y > 0.37 && m.Y < 0.47 {
			horizontalPts = append(horizontalPts, m)
		}
		if m.X > -0.445 && m.X < -0.345 && m.Y > -0.43 && m.Y < 0.30 {
			verticalPts = append(verticalPts, m)
		}
	}
	mH, bH, okH := fitIntercept(horizontalPts, true)
	mV, bV, okV := fitIntercept(verticalPts, false)
	if okH && okV {
		// y = mH*x + bH; x = mV*y + bV
		y := (mH*bV + bH) / (1 - mH*mV)
		x := mV*y + bV
		check.Estimated = V3(x, y, expected.Z)
		check.Error = math.Hypot(x-expected.X, y-expected.Y)
	}

	fx, fy, fz := g.WorldToIndex(probe)
	ix := clampInt(int(math.Round(fx+0.5)), 0, g.NX-1)
	iy := clampInt(int(math.Round(fy+0.5)), 0, g.NY-1)
	iz = clampInt(int(math.Round(fz+0.5)), 0, g.NZ-1)
	id := g.Index(ix, iy, iz)
	if g.Weight[id] >= 0.12 {
		check.ProbeTSDF = float64(g.Value[id])
		check.ProbeMetricSDF = g.MetricSDF(id)
	}
	return check
}

type CoverageCheck struct {
	Samples       int     `json:"samples"`
	Covered       int     `json:"covered"`
	CoverageRatio float64 `json:"coverage_ratio"`
	SearchRadius  float64 `json:"search_radius_m"`
}

func MeasureMeshCoverage(mesh *Mesh, samples []Vec3, radius float64) CoverageCheck {
	type cell = [3]int
	occupied := make(map[cell]struct{}, len(mesh.Vertices))
	cellSize := radius * 0.7
	for _, v := range mesh.Vertices {
		key := cell{
			int(math.Floor(v.Pos.X / cellSize)),
			int(math.Floor(v.Pos.Y / cellSize)),
			int(math.Floor(v.Pos.Z / cellSize)),
		}
		occupied[key] = struct{}{}
	}
	check := CoverageCheck{Samples: len(samples), SearchRadius: radius}
	offsets := []int{-1, 0, 1}
	for _, p := range samples {
		base := cell{
			int(math.Floor(p.X / cellSize)),
			int(math.Floor(p.Y / cellSize)),
			int(math.Floor(p.Z / cellSize)),
		}
		found := false
		for _, ox := range offsets {
			for _, oy := range offsets {
				for _, oz := range offsets {
					key := cell{base[0] + ox, base[1] + oy, base[2] + oz}
					if _, ok := occupied[key]; ok {
						found = true
					}
				}
			}
		}
		if found {
			check.Covered++
		}
	}
	if check.Samples > 0 {
		check.CoverageRatio = float64(check.Covered) / float64(check.Samples)
	}
	return check
}

func DepthTruthSamples(frames []*DepthFrame, step int) []Vec3 {
	var pts []Vec3
	for _, f := range frames {
		for y := 0; y < f.Height; y += step {
			for x := 0; x < f.Width; x += step {
				z := f.Depth[y*f.Width+x]
				if z <= 0 {
					continue
				}
				pts = append(pts, f.True.UnprojectDepth(x, y, z))
			}
		}
	}
	return pts
}
