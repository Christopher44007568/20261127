package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
)

// WeightToDelta maps a CSS-like wght axis (100..900) to design-unit expansion.
// 400 is the supplied regular outline; heavier values expand the strokes.
func WeightToDelta(weight int) float64 {
	w := float64(weight)
	w = math.Max(100, math.Min(900, w))
	return math.Max(0, (w-400)*0.06)
}

// SizeMetric records continuous vector counter area for one font size.
type SizeMetric struct {
	FontSize              float64 `json:"font_size"`
	Weight                int     `json:"weight"`
	Scale                 float64 `json:"scale"`
	RequestedDeltaUnits   float64 `json:"requested_delta_units"`
	AppliedDeltaUnits     float64 `json:"applied_delta_units"`
	DeltaPixels           float64 `json:"delta_pixels"`
	CounterAreaUnits      float64 `json:"counter_area_units"`
	CounterAreaPixels     float64 `json:"counter_area_pixels"`
	NormalizedCounterArea float64 `json:"normalized_counter_area"`
	IntersectionCount     int     `json:"intersection_count"`
	TopologyValid         bool    `json:"topology_valid"`
}

// MeasureFontSize scales one design-space expansion to a physical font size.
// Area is computed from continuous vector contours, not integer pixel counts,
// so font-size animation does not suffer raster grid jumps.
func MeasureFontSize(base Outline, weight int, fontSize float64, opts ExpandOptions) SizeMetric {
	requested := WeightToDelta(weight)
	expansion := ExpandOutline(base, requested, opts)
	scale := fontSize / UnitsPerEm
	areaUnits := expansion.Outline.CounterArea()
	return SizeMetric{
		FontSize:              fontSize,
		Weight:                weight,
		Scale:                 scale,
		RequestedDeltaUnits:   requested,
		AppliedDeltaUnits:     expansion.Delta,
		DeltaPixels:           expansion.Delta * scale,
		CounterAreaUnits:      areaUnits,
		CounterAreaPixels:     areaUnits * scale * scale,
		NormalizedCounterArea: areaUnits / (UnitsPerEm * UnitsPerEm),
		IntersectionCount:     len(expansion.Intersections),
		TopologyValid:         expansion.Valid && len(expansion.Intersections) == 0,
	}
}

func MeasureSizeSeries(base Outline, weight int, sizes []float64, opts ExpandOptions) []SizeMetric {
	result := make([]SizeMetric, 0, len(sizes))
	for _, size := range sizes {
		result = append(result, MeasureFontSize(base, weight, size, opts))
	}
	return result
}

func WriteSizeCSV(w io.Writer, metrics []SizeMetric) error {
	cw := csv.NewWriter(w)
	header := []string{
		"font_size", "weight", "scale", "requested_delta_units",
		"applied_delta_units", "delta_pixels", "counter_area_units",
		"counter_area_pixels", "normalized_counter_area",
		"intersection_count", "topology_valid",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, m := range metrics {
		row := []string{
			fmt.Sprintf("%.3f", m.FontSize),
			fmt.Sprintf("%d", m.Weight),
			fmt.Sprintf("%.8f", m.Scale),
			fmt.Sprintf("%.6f", m.RequestedDeltaUnits),
			fmt.Sprintf("%.6f", m.AppliedDeltaUnits),
			fmt.Sprintf("%.6f", m.DeltaPixels),
			fmt.Sprintf("%.6f", m.CounterAreaUnits),
			fmt.Sprintf("%.6f", m.CounterAreaPixels),
			fmt.Sprintf("%.10f", m.NormalizedCounterArea),
			fmt.Sprintf("%d", m.IntersectionCount),
			fmt.Sprintf("%t", m.TopologyValid),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// TopologyReport is a compact invariant summary.
type TopologyReport struct {
	ContourCount      int       `json:"contour_count"`
	OuterSignedArea   float64   `json:"outer_signed_area"`
	HoleSignedAreas   []float64 `json:"hole_signed_areas"`
	NestingSignature  []int     `json:"nesting_signature"`
	IntersectionCount int       `json:"intersection_count"`
	Valid             bool      `json:"valid"`
}

func AnalyzeTopology(base, candidate Outline) TopologyReport {
	intersections := FindOutlineIntersections(candidate)
	holes := make([]float64, len(candidate.Holes))
	for i, h := range candidate.Holes {
		holes[i] = h.SignedArea()
	}
	signature := candidate.NestingSignature()
	sameNest := true
	want := base.NestingSignature()
	if len(want) != len(signature) {
		sameNest = false
	} else {
		for i := range want {
			if want[i] != signature[i] {
				sameNest = false
			}
		}
	}
	orientationOK := candidate.Outer.SignedArea() > 0
	for _, h := range candidate.Holes {
		if h.SignedArea() >= 0 {
			orientationOK = false
		}
	}
	return TopologyReport{
		ContourCount:      len(candidate.Contours()),
		OuterSignedArea:   candidate.Outer.SignedArea(),
		HoleSignedAreas:   holes,
		NestingSignature:  signature,
		IntersectionCount: len(intersections),
		Valid:             sameNest && orientationOK && len(intersections) == 0,
	}
}
