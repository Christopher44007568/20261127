package report

import (
	"fmt"
	"io"
	"math"

	"orientation-interp/interp"
)

func times(s Series) []float64 {
	out := make([]float64, len(s.Samples))
	for i, sample := range s.Samples {
		out[i] = sample.T
	}
	return out
}

// WriteSpeedChart renders angular speed magnitude for all methods.
func WriteSpeedChart(w io.Writer, all []Series, keyTimes []float64) error {
	const width, height = 960, 420
	c := newCanvas(width, height, margin{left: 64, right: 24, top: 44, bottom: 40})
	c.title("Angular speed |ω| (rad/s)")

	yMax := 0.0
	for _, s := range all {
		yMax = math.Max(yMax, s.MaxSpeed)
	}
	yMax *= 1.08
	x0, x1 := interp.ScenarioStart, interp.ScenarioEnd
	c.axes(x0, x1, 0, yMax, 6, func(v float64) string {
		return fmt.Sprintf("%.1f", v)
	})
	c.keyframeMarkers(keyTimes, x0, x1, 0, yMax)

	items := make([]legendItem, 0, len(all))
	for _, s := range all {
		color := seriesColors[string(s.Method)]
		c.line(times(s), s.Speed, x0, x1, 0, yMax, color, 2)
		items = append(items, legendItem{Color: color, Label: s.Label})
	}
	c.legend(items)
	return c.writeTo(w)
}

// WriteOrthoChart renders the rotation matrix orthogonality error on a
// log10 scale, since well-behaved interpolation stays near 1e-15.
func WriteOrthoChart(w io.Writer, all []Series, keyTimes []float64) error {
	const width, height = 960, 420
	c := newCanvas(width, height, margin{left: 64, right: 24, top: 44, bottom: 40})
	c.title("Rotation matrix orthogonality error  max(‖RᵀR−I‖F, |det R−1|)  [log10]")

	floor, ceil := -16.0, 1.0
	logVals := make([][]float64, len(all))
	for i, s := range all {
		logVals[i] = make([]float64, len(s.Ortho))
		for j, v := range s.Ortho {
			if math.IsNaN(v) || v <= 0 {
				logVals[i][j] = math.NaN()
				continue
			}
			logVals[i][j] = math.Max(floor, math.Log10(v))
		}
	}
	x0, x1 := interp.ScenarioStart, interp.ScenarioEnd
	c.axes(x0, x1, floor, ceil, 8, func(v float64) string {
		return fmt.Sprintf("10^%.0f", v)
	})
	c.keyframeMarkers(keyTimes, x0, x1, floor, ceil)

	items := make([]legendItem, 0, len(all))
	for i, s := range all {
		color := seriesColors[string(s.Method)]
		c.line(times(s), logVals[i], x0, x1, floor, ceil, color, 2)
		items = append(items, legendItem{Color: color, Label: s.Label})
	}
	c.legend(items)
	return c.writeTo(w)
}

// WriteBlendNormChart shows the pre-normalization quaternion norm. Healthy
// interpolation stays at 1; raw nlerp dives toward zero at antipodal blends.
func WriteBlendNormChart(w io.Writer, all []Series, keyTimes []float64) error {
	const width, height = 960, 360
	c := newCanvas(width, height, margin{left: 64, right: 24, top: 44, bottom: 40})
	c.title("Pre-normalization quaternion norm (nlerp collapse indicator)")

	x0, x1 := interp.ScenarioStart, interp.ScenarioEnd
	c.axes(x0, x1, 0, 1.1, 5, func(v float64) string {
		return fmt.Sprintf("%.1f", v)
	})
	c.keyframeMarkers(keyTimes, x0, x1, 0, 1.1)

	items := make([]legendItem, 0, len(all))
	for _, s := range all {
		color := seriesColors[string(s.Method)]
		c.line(times(s), s.BlendNorm, x0, x1, 0, 1.1, color, 2)
		items = append(items, legendItem{Color: color, Label: s.Label})
	}
	c.legend(items)
	return c.writeTo(w)
}
