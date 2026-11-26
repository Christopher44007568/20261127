package diag

import (
	"math"
	"testing"
)

func TestSummarize(t *testing.T) {
	s := Summarize([]float64{1, 2, 3, 4, 5})
	if s.Count != 5 || s.Min != 1 || s.Max != 5 {
		t.Fatalf("unexpected count/range: %+v", s)
	}
	if math.Abs(s.Mean-3) > 1e-12 || math.Abs(s.Median-3) > 1e-12 {
		t.Fatalf("unexpected center: mean=%v median=%v", s.Mean, s.Median)
	}
	if s.P5 < 1 || s.P95 > 5 {
		t.Fatalf("percentiles out of range: %+v", s)
	}
}

func TestHistogramCountsAllValues(t *testing.T) {
	values := []float64{1, 1, 2, 3, 5, 8}
	b := Histogram(values, 4)
	total := 0
	for _, count := range b.Counts {
		total += count
	}
	if total != len(values) {
		t.Fatalf("histogram counted %d values, want %d", total, len(values))
	}
	if b.Lo != 1 || b.Hi != 8 {
		t.Fatalf("range = %v..%v", b.Lo, b.Hi)
	}
}

func TestQuantiles(t *testing.T) {
	q := Quantiles([]float64{0, 10, 20, 30, 40}, []float64{0, 0.5, 1})
	want := []float64{0, 20, 40}
	for i := range want {
		if math.Abs(q[i]-want[i]) > 1e-12 {
			t.Fatalf("quantile %d = %v, want %v", i, q[i], want[i])
		}
	}
}
