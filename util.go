package main

import "strconv"

// formatTick renders small axis numbers without third-party dependencies.
func formatTick(v float64) string {
	av := v
	if av < 0 {
		av = -av
	}
	switch {
	case av >= 100:
		return strconv.FormatFloat(v, 'f', 0, 64)
	case av >= 10:
		return strconv.FormatFloat(v, 'f', 1, 64)
	default:
		return strconv.FormatFloat(v, 'f', 2, 64)
	}
}
