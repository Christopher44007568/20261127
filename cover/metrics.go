package cover

import "math"

// ErrStats summarizes an absolute error buffer against a reference.
type ErrStats struct {
	RMSE   float64 `json:"rmse"`
	MaxAbs float64 `json:"maxAbs"`
	Mean   float64 `json:"meanAbs"`
	SumRef float64 `json:"sumRef"`
	SumGot float64 `json:"sumGot"`
}

// DiffStats compares got against ref element-wise.
func DiffStats(ref, got []float64) ErrStats {
	st := ErrStats{}
	var sq, sumErr float64
	n := len(ref)
	for i := range ref {
		e := math.Abs(got[i] - ref[i])
		if e > st.MaxAbs {
			st.MaxAbs = e
		}
		sq += (got[i] - ref[i]) * (got[i] - ref[i])
		sumErr += e
		st.SumRef += ref[i]
		st.SumGot += got[i]
	}
	if n > 0 {
		st.RMSE = math.Sqrt(sq / float64(n))
		st.Mean = sumErr / float64(n)
	}
	return st
}

// MaxRGBLinearDiff returns the maximum per-channel difference between two
// 0..255 RGB buffers and the index where it occurs (-1 when lengths differ).
func MaxRGBLinearDiff(a, b []RGB) (max float64, index int) {
	index = -1
	if len(a) != len(b) {
		return math.Inf(1), -1
	}
	for i := range a {
		for _, d := range [3]float64{
			math.Abs(a[i].R - b[i].R),
			math.Abs(a[i].G - b[i].G),
			math.Abs(a[i].B - b[i].B),
		} {
			if d > max {
				max, index = d, i
			}
		}
	}
	return max, index
}

// StainReport measures how much removing one layer changes the render on
// pixels where that layer's reference coverage is exactly zero. A correct
// coverage pipeline must score 0; a blur pipeline will not.
type StainReport struct {
	Layer       string  `json:"layer"`
	ZeroPixels  int     `json:"zeroPixels"`
	MaxStain255 float64 `json:"maxStain255"`
	MeanStain   float64 `json:"meanStain255"`
}

// MeasureStain compares fullRGB against noLayerRGB over indices where
// refCov == 0. Differences are in 0..255 channel units.
func MeasureStain(layerName string, refCov []float64, fullRGB, noLayerRGB []RGB) StainReport {
	rep := StainReport{Layer: layerName}
	var sum float64
	for i, c := range refCov {
		if c != 0 {
			continue
		}
		rep.ZeroPixels++
		d := 0.0
		d = math.Max(d, math.Abs(fullRGB[i].R-noLayerRGB[i].R))
		d = math.Max(d, math.Abs(fullRGB[i].G-noLayerRGB[i].G))
		d = math.Max(d, math.Abs(fullRGB[i].B-noLayerRGB[i].B))
		if d > rep.MaxStain255 {
			rep.MaxStain255 = d
		}
		sum += d
	}
	if rep.ZeroPixels > 0 {
		rep.MeanStain = sum / float64(rep.ZeroPixels)
	}
	return rep
}
