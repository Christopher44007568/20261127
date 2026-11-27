package material

import "math"

// BrightnessStats summarizes aligned display-luma differences.
type BrightnessStats struct {
	MaterialID string  `json:"material_id"`
	Compare    string  `json:"compare"`
	Pixels     int     `json:"pixels"`
	MAE        float64 `json:"mae"`
	RMSE       float64 `json:"rmse"`
	MaxAbs     float64 `json:"max_abs"`
	MeanSigned float64 `json:"mean_signed"`
}

// CompareBuffers computes differences over shared sphere mask pixels.
func CompareBuffers(materialID, compare string, a, b *SphereBuffer) BrightnessStats {
	s := BrightnessStats{MaterialID: materialID, Compare: compare}
	var sumAbs, sumSq, sumSigned float64
	for i, ok := range a.Mask {
		if !ok || !b.Mask[i] {
			continue
		}
		s.Pixels++
		d := a.Luma[i] - b.Luma[i]
		ad := math.Abs(d)
		sumAbs += ad
		sumSq += d * d
		sumSigned += d
		if ad > s.MaxAbs {
			s.MaxAbs = ad
		}
	}
	if s.Pixels > 0 {
		n := float64(s.Pixels)
		s.MAE = sumAbs / n
		s.RMSE = math.Sqrt(sumSq / n)
		s.MeanSigned = sumSigned / n
	}
	return s
}

// MaxStepJump returns the largest per-pixel luma change between two LOD frames.
func MaxStepJump(a, b *SphereBuffer) float64 {
	var maxJump float64
	for i, ok := range a.Mask {
		if !ok || !b.Mask[i] {
			continue
		}
		if d := math.Abs(a.Luma[i] - b.Luma[i]); d > maxJump {
			maxJump = d
		}
	}
	return maxJump
}

// ViewSample is one fixed-normal view-angle luminance sample.
type ViewSample struct {
	MaterialID      string  `json:"material_id"`
	MaterialKind    string  `json:"material_kind"`
	Tier            string  `json:"tier"`
	AngleDeg        float64 `json:"angle_deg"`
	LinearLuminance float64 `json:"linear_luminance"`
	DisplayLuma     float64 `json:"display_luma"`
	DisplayR        float64 `json:"display_r"`
	DisplayG        float64 `json:"display_g"`
	DisplayB        float64 `json:"display_b"`
}

// SweepView evaluates N=(0,0,1), V at angle degrees from N.
func SweepView(m Material, tier Tier, angleDeg float64) ViewSample {
	rad := angleDeg * math.Pi / 180
	view := Vec3{X: math.Sin(rad), Z: math.Cos(rad)}.Normalize()
	normal := Vec3{Z: 1}
	linear := Shade(m, tier, normal, view)
	display := Display(linear)
	return ViewSample{
		MaterialID:      m.ID,
		MaterialKind:    m.Kind,
		Tier:            tier.String(),
		AngleDeg:        angleDeg,
		LinearLuminance: linear.Luminance(),
		DisplayLuma:     display.Luminance(),
		DisplayR:        display.R,
		DisplayG:        display.G,
		DisplayB:        display.B,
	}
}

// ViewSweep returns inclusive 0..90 degree samples at one degree resolution.
func ViewSweep(m Material, tier Tier) []ViewSample {
	samples := make([]ViewSample, 0, 91)
	for angle := 0.0; angle <= 90.0001; angle++ {
		samples = append(samples, SweepView(m, tier, angle))
	}
	return samples
}

// BoundaryRow is one point in the continuous LOD error scan.
type BoundaryRow struct {
	LOD          float64    `json:"lod"`
	MaterialID   string     `json:"material_id"`
	BoundaryZone string     `json:"boundary_zone"`
	HardTier     string     `json:"hard_tier"`
	Weights      LODWeights `json:"weights"`
	MAEvsHigh    float64    `json:"mae_vs_high"`
	RMSEvsHigh   float64    `json:"rmse_vs_high"`
	MaxvsHigh    float64    `json:"max_vs_high"`
	MaxStepJump  float64    `json:"max_step_jump"`
}

// BoundarySummary reports one hard boundary and its continuous path.
type BoundarySummary struct {
	BoundaryLOD       float64 `json:"boundary_lod"`
	Transition        string  `json:"transition"`
	PairMAE           float64 `json:"pair_mae"`
	PairRMSE          float64 `json:"pair_rmse"`
	PairMax           float64 `json:"pair_max"`
	ContinuousMaxStep float64 `json:"continuous_max_step"`
	Step              float64 `json:"step"`
	JumpThreshold     float64 `json:"jump_threshold"`
	Passed            bool    `json:"passed"`
}

// PreservationReport proves identity parameters remain fixed across tiers.
type PreservationReport struct {
	MaterialID        string  `json:"material_id"`
	BaseColorDelta    float64 `json:"base_color_delta"`
	HueDeltaDeg       float64 `json:"hue_delta_deg"`
	RoughnessMinDelta float64 `json:"roughness_min_delta"`
	RoughnessMaxDelta float64 `json:"roughness_max_delta"`
	RangePreserved    bool    `json:"range_preserved"`
	ColorPreserved    bool    `json:"color_preserved"`
}

// ContinuityReport is the complete structured boundary report.
type ContinuityReport struct {
	TransitionWidth float64              `json:"transition_width"`
	LODStep         float64              `json:"lod_step"`
	JumpThreshold   float64              `json:"jump_threshold"`
	Rows            []BoundaryRow        `json:"rows"`
	Boundaries      []BoundarySummary    `json:"boundaries"`
	Preservation    []PreservationReport `json:"preservation"`
}

// HardTierAt returns the tier selected by a conventional nearest LOD switch.
func HardTierAt(lod float64) Tier {
	switch {
	case lod < 0.5:
		return TierLow
	case lod < 1.5:
		return TierMedium
	default:
		return TierHigh
	}
}

func boundaryZone(lod, width float64) string {
	switch {
	case lod >= 0.5-width-1e-9 && lod <= 0.5+width+1e-9:
		return "low_medium"
	case lod >= 1.5-width-1e-9 && lod <= 1.5+width+1e-9:
		return "medium_high"
	default:
		return "none"
	}
}

// ScanContinuity renders 201 LOD frames per material and measures jumps.
func ScanContinuity(materials []Material, step, width, threshold float64) ContinuityReport {
	report := ContinuityReport{
		TransitionWidth: width,
		LODStep:         step,
		JumpThreshold:   threshold,
		Rows:            []BoundaryRow{},
		Boundaries:      []BoundarySummary{},
		Preservation:    Preservation(materials),
	}

	pairs := map[string]seriesMetrics{}
	var globalBoundaryJumps = map[float64]float64{0.5: 0, 1.5: 0}

	for _, m := range materials {
		low := sweepLuma(m, TierLow)
		medium := sweepLuma(m, TierMedium)
		high := sweepLuma(m, TierHigh)
		pairs[m.ID+":0.5"] = seriesDifference(low, medium)
		pairs[m.ID+":1.5"] = seriesDifference(medium, high)

		var previous []float64
		steps := int(math.Round(2 / step))
		for idx := 0; idx <= steps; idx++ {
			lod := clamp(float64(idx)*step, 0, 2)
			current := sweepLODLuma(m, lod, width)
			vsHigh := seriesDifference(current, high)
			jump := 0.0
			if previous != nil {
				jump = maxSeriesStep(previous, current)
			}
			w := WeightsAt(lod, width)
			report.Rows = append(report.Rows, BoundaryRow{
				LOD:          math.Round(lod*1e6) / 1e6,
				MaterialID:   m.ID,
				BoundaryZone: boundaryZone(lod, width),
				HardTier:     HardTierAt(lod).String(),
				Weights:      w,
				MAEvsHigh:    vsHigh.mae,
				RMSEvsHigh:   vsHigh.rmse,
				MaxvsHigh:    vsHigh.max,
				MaxStepJump:  jump,
			})
			if boundaryZone(lod, width) != "none" {
				boundary := 0.5
				if lod > 1 {
					boundary = 1.5
				}
				if jump > globalBoundaryJumps[boundary] {
					globalBoundaryJumps[boundary] = jump
				}
			}
			previous = current
		}
	}

	transitions := []struct {
		lod  float64
		name string
		key  string
	}{
		{0.5, "low->medium", ":0.5"},
		{1.5, "medium->high", ":1.5"},
	}
	for _, tr := range transitions {
		var sumMAE, sumMSE, pairMax, maxStep float64
		for _, m := range materials {
			p := pairs[m.ID+tr.key]
			sumMAE += p.mae
			sumMSE += p.rmse * p.rmse
			if p.max > pairMax {
				pairMax = p.max
			}
			if globalBoundaryJumps[tr.lod] > maxStep {
				maxStep = globalBoundaryJumps[tr.lod]
			}
		}
		n := float64(len(materials))
		// Pair metrics are reported per material in the Markdown table.
		// BoundarySummary in the JSON report is an aggregate across materials.
		report.Boundaries = append(report.Boundaries, BoundarySummary{
			BoundaryLOD:       tr.lod,
			Transition:        tr.name,
			PairMAE:           sumMAE / n,
			PairRMSE:          math.Sqrt(sumMSE / n),
			PairMax:           pairMax,
			ContinuousMaxStep: maxStep,
			Step:              step,
			JumpThreshold:     threshold,
			Passed:            maxStep <= threshold,
		})
	}
	return report
}

type seriesMetrics struct {
	mae  float64
	rmse float64
	max  float64
}

func sweepLuma(m Material, tier Tier) []float64 {
	samples := ViewSweep(m, tier)
	out := make([]float64, len(samples))
	for i, sample := range samples {
		out[i] = sample.DisplayLuma
	}
	return out
}

func sweepLODLuma(m Material, lod, width float64) []float64 {
	out := make([]float64, 0, 91)
	for angle := 0.0; angle <= 90.0001; angle++ {
		rad := angle * math.Pi / 180
		view := Vec3{X: math.Sin(rad), Z: math.Cos(rad)}.Normalize()
		linear, _ := ShadeLOD(m, lod, Vec3{Z: 1}, view, width)
		out = append(out, Display(linear).Luminance())
	}
	return out
}

func seriesDifference(a, b []float64) seriesMetrics {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var sumAbs, sumSq float64
	var maxAbs float64
	for i := 0; i < n; i++ {
		d := a[i] - b[i]
		ad := math.Abs(d)
		sumAbs += ad
		sumSq += d * d
		if ad > maxAbs {
			maxAbs = ad
		}
	}
	if n == 0 {
		return seriesMetrics{}
	}
	fn := float64(n)
	return seriesMetrics{mae: sumAbs / fn, rmse: math.Sqrt(sumSq / fn), max: maxAbs}
}

func maxSeriesStep(previous, current []float64) float64 {
	n := len(previous)
	if len(current) < n {
		n = len(current)
	}
	var maxJump float64
	for i := 0; i < n; i++ {
		if jump := math.Abs(current[i] - previous[i]); jump > maxJump {
			maxJump = jump
		}
	}
	return maxJump
}

// Preservation compares the low uniform identity block to the full material.
func Preservation(materials []Material) []PreservationReport {
	out := make([]PreservationReport, 0, len(materials))
	for _, m := range materials {
		u := m.LowUniforms()
		colorDelta := colorDistance(m.BaseColor, u.BaseColor)
		hueDelta := HueDelta(m.BaseColor.ToSRGB(), u.BaseColor.ToSRGB())
		minDelta := math.Abs(m.RoughnessMin - u.RoughnessMin)
		maxDelta := math.Abs(m.RoughnessMax - u.RoughnessMax)
		out = append(out, PreservationReport{
			MaterialID:        m.ID,
			BaseColorDelta:    colorDelta,
			HueDeltaDeg:       hueDelta,
			RoughnessMinDelta: minDelta,
			RoughnessMaxDelta: maxDelta,
			RangePreserved:    minDelta == 0 && maxDelta == 0,
			ColorPreserved:    colorDelta == 0 && hueDelta == 0,
		})
	}
	return out
}

func colorDistance(a, b Color) float64 {
	return math.Sqrt((a.R-b.R)*(a.R-b.R) + (a.G-b.G)*(a.G-b.G) + (a.B-b.B)*(a.B-b.B))
}
