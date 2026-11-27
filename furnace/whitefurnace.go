package furnace

import (
	"math"
)

// FurnacePoint is one outgoing-direction white furnace measurement.
type FurnacePoint struct {
	NdotV           float64 `json:"ndot_v"`
	RawEnergy       float64 `json:"raw_energy"`
	CorrectedEnergy float64 `json:"corrected_energy"`
	AbsError        float64 `json:"abs_error"`
	RawError        float64 `json:"raw_abs_error"`
	Pass            bool    `json:"pass"`
}

// FurnaceReport contains directional and worst-case white furnace results.
type FurnaceReport struct {
	Kind        string         `json:"kind"`
	Name        string         `json:"name"`
	Tolerance   float64        `json:"tolerance"`
	Bins        int            `json:"bins"`
	LUTSamples  int            `json:"lut_samples"`
	TestSamples int            `json:"test_samples"`
	Pass        bool           `json:"pass"`
	WorstNdotV  float64        `json:"worst_ndot_v"`
	WorstError  float64        `json:"worst_error"`
	RawWorst    float64        `json:"raw_worst_error"`
	MISWorst    float64        `json:"mis_worst_error"`
	MISPass     bool           `json:"mis_pass"`
	Points      []FurnacePoint `json:"points"`
	Sphere      SphereFurnace  `json:"sphere"`
}

// SphereFurnace summarizes response over an analytic sphere, including its rim.
type SphereFurnace struct {
	Width              int     `json:"width"`
	Height             int     `json:"height"`
	DirectionChecks    int     `json:"direction_checks"`
	IntegrationSamples int     `json:"integration_samples"`
	MinNdotV           float64 `json:"min_ndot_v"`
	WorstNdotV         float64 `json:"worst_ndot_v"`
	WorstError         float64 `json:"worst_error"`
	CenterError        float64 `json:"center_error"`
	RimError           float64 `json:"rim_error"`
	Pass               bool    `json:"pass"`
}

// WhiteFurnaceDirections includes head-on, mid angles, and near-rim views.
func WhiteFurnaceDirections() []float64 {
	return []float64{1.0, 0.92, 0.78, 0.60, 0.42, 0.26, 0.14, 0.07, 0.035, 0.02}
}

// RunWhiteFurnace builds a normalization LUT and independently integrates the
// corrected directional response for m.
func RunWhiteFurnace(m Material, bins, lutSamples, testSamples int, tolerance float64) FurnaceReport {
	if bins <= 0 {
		bins = 96
	}
	if lutSamples <= 0 {
		lutSamples = 16384
	}
	if testSamples <= 0 {
		testSamples = 24576
	}
	if tolerance <= 0 {
		tolerance = 0.025
	}

	wm := WhiteFurnaceMaterial(m)
	table := BuildEnergyTable(wm, bins, lutSamples)
	report := FurnaceReport{
		Kind: m.Kind.String(), Name: m.Name, Tolerance: tolerance,
		Bins: bins, LUTSamples: lutSamples, TestSamples: testSamples,
		Pass:    true,
		MISPass: true,
	}

	for _, mu := range WhiteFurnaceDirections() {
		raw := DirectionalAlbedo(wm, mu, testSamples).X
		corrected := CorrectedDirectionalAlbedo(wm, table, mu, testSamples).X
		err := math.Abs(corrected - 1)
		p := FurnacePoint{
			NdotV: mu, RawEnergy: raw, CorrectedEnergy: corrected,
			AbsError: err, RawError: math.Abs(raw - 1),
			Pass: err <= tolerance,
		}
		report.Points = append(report.Points, p)
		if !p.Pass {
			report.Pass = false
		}
		if err > report.WorstError {
			report.WorstError = err
			report.WorstNdotV = mu
		}
		if p.RawError > report.RawWorst {
			report.RawWorst = p.RawError
		}
	}

	report.Sphere = MeasureSphereFurnace(wm, table, 256, 4096, tolerance)
	if !report.Sphere.Pass {
		report.Pass = false
	}

	for _, mu := range []float64{1, 0.5, 0.15, 0.05, 0.025} {
		e := IntegrateNormalizedBRDFMIS(wm, table, mu, testSamples).X
		if err := math.Abs(e - 1); err > report.MISWorst {
			report.MISWorst = err
		}
	}
	if report.MISWorst > 0.04 {
		report.MISPass = false
		report.Pass = false
	}
	return report
}

// RunAllWhiteFurnace tests rough metal, clearcoat, and velvet.
func RunAllWhiteFurnace() []FurnaceReport {
	presets := Presets()
	reports := make([]FurnaceReport, 0, len(presets))
	for _, m := range presets {
		reports = append(reports, RunWhiteFurnace(m, 96, 16384, 24576, 0.025))
	}
	return reports
}

// MeasureSphereFurnace walks the visible hemisphere encoded by a sphere's
// projected normal field. It deliberately checks many grazing mu values rather
// than only the center texel.
func MeasureSphereFurnace(wm Material, table *EnergyTable, size, samples int, tolerance float64) SphereFurnace {
	if size < 32 {
		size = 32
	}
	if samples <= 0 {
		samples = 4096
	}
	s := SphereFurnace{
		Width: size, Height: size, Pass: true, MinNdotV: 1,
		IntegrationSamples: samples,
	}
	radius := float64(size) * 0.46
	center := float64(size-1) / 2
	verify := BuildEnergyCurve(wm, table, 56, samples)

	// Check projected pixels plus a dense analytic radial sweep. The sweep
	// reaches the same grazing normal values used by edge pixels.
	check := func(x, y float64, pixel bool) {
		r2 := x*x + y*y
		if r2 > radius*radius {
			return
		}
		mu := math.Sqrt(math.Max(0, 1-r2/(radius*radius)))
		if mu < 0.006 {
			mu = 0.006
		}
		e := verify.CorrectedAt(mu)
		err := math.Abs(e - 1)
		s.DirectionChecks++
		if mu < s.MinNdotV {
			s.MinNdotV = mu
		}
		if err > s.WorstError {
			s.WorstError = err
			s.WorstNdotV = mu
		}
		if mu > 0.98 && err > s.CenterError {
			s.CenterError = err
		}
		if mu < 0.12 && err > s.RimError {
			s.RimError = err
		}
		if err > tolerance {
			s.Pass = false
		}
		_ = pixel
	}

	for py := 0; py < size; py += 4 {
		for px := 0; px < size; px += 4 {
			x := float64(px) - center
			y := float64(py) - center
			check(x, y, true)
		}
	}
	for i := 0; i <= 240; i++ {
		mu := 1 - 0.994*float64(i)/240
		r := radius * math.Sqrt(math.Max(0, 1-mu*mu))
		check(r, 0, false)
		check(r*0.707, r*0.707, false)
	}
	if s.WorstNdotV == 0 {
		s.WorstNdotV = 0.006
	}
	return s
}

// AllPass is a small helper for callers that only need a gate.
func AllPass(reports []FurnaceReport) bool {
	for _, r := range reports {
		if !r.Pass {
			return false
		}
	}
	return true
}
