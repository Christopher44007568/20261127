package diag

import (
	"math"
	"sort"
)

// Summary is a robust descriptive summary of a numeric sample.
type Summary struct {
	Count  int
	Min    float64
	P5     float64
	P25    float64
	Median float64
	P75    float64
	P95    float64
	Max    float64
	Mean   float64
	StdDev float64
	CV     float64
}

func Summarize(values []float64) Summary {
	if len(values) == 0 {
		return Summary{}
	}
	v := append([]float64(nil), values...)
	sort.Float64s(v)
	s := Summary{
		Count:  len(v),
		Min:    v[0],
		P5:     percentileSorted(v, 0.05),
		P25:    percentileSorted(v, 0.25),
		Median: percentileSorted(v, 0.50),
		P75:    percentileSorted(v, 0.75),
		P95:    percentileSorted(v, 0.95),
		Max:    v[len(v)-1],
	}
	for _, x := range v {
		s.Mean += x
	}
	s.Mean /= float64(len(v))
	if len(v) > 1 {
		for _, x := range v {
			d := x - s.Mean
			s.StdDev += d * d
		}
		s.StdDev = math.Sqrt(s.StdDev / float64(len(v)-1))
	}
	if s.Mean > 1e-14 {
		s.CV = s.StdDev / s.Mean
	}
	return s
}

func percentileSorted(sortedValues []float64, q float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if q <= 0 {
		return sortedValues[0]
	}
	if q >= 1 {
		return sortedValues[len(sortedValues)-1]
	}
	pos := q * float64(len(sortedValues)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sortedValues[lo]
	}
	t := pos - float64(lo)
	return sortedValues[lo]*(1-t) + sortedValues[hi]*t
}

// Bins is a fixed-width histogram.
type Bins struct {
	Lo     float64
	Hi     float64
	Width  float64
	Counts []int
}

func Histogram(values []float64, count int) Bins {
	if count < 1 {
		count = 1
	}
	out := Bins{Counts: make([]int, count)}
	if len(values) == 0 {
		return out
	}
	out.Lo, out.Hi = values[0], values[0]
	for _, v := range values[1:] {
		out.Lo = math.Min(out.Lo, v)
		out.Hi = math.Max(out.Hi, v)
	}
	if out.Hi-out.Lo < 1e-14 {
		out.Width = 1e-14
		out.Hi = out.Lo + out.Width*float64(count)
	} else {
		out.Width = (out.Hi - out.Lo) / float64(count)
	}
	for _, v := range values {
		i := int((v - out.Lo) / out.Width)
		if i < 0 {
			i = 0
		}
		if i >= count {
			i = count - 1
		}
		out.Counts[i]++
	}
	return out
}

func Quantiles(values []float64, qs []float64) []float64 {
	v := append([]float64(nil), values...)
	sort.Float64s(v)
	out := make([]float64, len(qs))
	for i, q := range qs {
		out[i] = percentileSorted(v, q)
	}
	return out
}
