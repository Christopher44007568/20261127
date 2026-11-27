package main

import "math"

type FilterStats struct {
	Taps          int
	Probes        int
	DesiredProbes int
	ActualProbes  int
	SampleWidth   float64
	Degraded      bool
}

func fixedFour(tex *Texture, uv Vec2) (RGB, FilterStats) {
	c, taps := tex.sampleLevel(0, uv)
	return c, FilterStats{Taps: taps, Probes: 1, DesiredProbes: 1, ActualProbes: 1}
}

func anisotropic(tex *Texture, uv Vec2, fp Footprint, budget int) (RGB, FilterStats) {
	if budget < 8 {
		return anisotropicSingleMip(tex, uv, fp, budget)
	}
	tapsPerProbe := 4
	maxProbes := budget / tapsPerProbe
	if maxProbes < 1 {
		maxProbes = 1
	}
	desired := requiredProbes(fp)
	actual := desired
	if actual > maxProbes {
		actual = maxProbes
	}
	if actual < 1 {
		actual = 1
	}
	sampleWidth := math.Max(fp.Minor, fp.Major/float64(actual))
	if actual < desired {
		// Once the budget cannot cover the preferred long-axis sampling,
		// trade short-axis sharpness for a kernel wide enough to remove
		// the uncovered phase instead of folding it into moire.
		sampleWidth = math.Max(sampleWidth, 2*fp.Major/float64(actual))
	}
	if sampleWidth < 1e-6 {
		sampleWidth = 1
	}
	lod := math.Log2(math.Max(sampleWidth, 1))

	sigma := fp.Major / math.Sqrt(12)
	if sigma < 1e-6 {
		sigma = 1e-6
	}
	var sum RGB
	var weightSum float64
	totalTaps := 0
	for i := 0; i < actual; i++ {
		u := (float64(i)+0.5)/float64(actual) - 0.5
		x := u * fp.Major
		weight := math.Exp(-0.5 * x * x / (sigma * sigma))
		p := uv.Add(fp.AxisMajor.Mul(x))
		c, taps := tex.sampleNearestMip(p, lod)
		sum = sum.Add(c.Mul(weight))
		weightSum += weight
		totalTaps += taps
	}
	color := sum.Mul(1 / weightSum)
	return color, FilterStats{
		Taps:          totalTaps,
		Probes:        actual,
		DesiredProbes: desired,
		ActualProbes:  actual,
		SampleWidth:   sampleWidth,
		Degraded:      actual < desired || (sampleWidth > 1.05 && sampleWidth > fp.Minor*1.05),
	}
}

func anisotropicSingleMip(tex *Texture, uv Vec2, fp Footprint, budget int) (RGB, FilterStats) {
	desired := requiredProbes(fp)
	sampleWidth := math.Max(fp.Minor, fp.Major)
	lod := math.Log2(math.Max(sampleWidth, 1))
	c, taps := tex.sampleNearestMip(uv, lod)
	if taps > budget {
		taps = budget
	}
	return c, FilterStats{
		Taps:          taps,
		Probes:        1,
		DesiredProbes: desired,
		ActualProbes:  1,
		SampleWidth:   sampleWidth,
		Degraded:      true,
	}
}

func requiredProbes(fp Footprint) int {
	minor := math.Max(fp.Minor, 1)
	n := int(math.Ceil(2 * fp.Major / minor))
	if n < 1 {
		n = 1
	}
	if n > 1024 {
		n = 1024
	}
	return n
}

func referenceColor(pattern *StripePattern, uv Vec2, fp Footprint) RGB {
	if !fp.Valid {
		return pattern.colorAt(uv)
	}
	return pattern.analyticFiltered(uv, fp)
}
