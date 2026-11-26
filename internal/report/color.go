package report

import (
	"fmt"
	"math"
)

type rgb struct {
	r, g, b float64
}

var (
	heatCool    = rgb{31, 79, 128}
	heatNeutral = rgb{230, 231, 226}
	heatWarm    = rgb{174, 49, 40}
)

// heatColor maps a local area ratio to a blue-neutral-red scale. The scale is
// intentionally common to LBS and DQ so the two panels can be compared.
func heatColor(ratio float64) string {
	const visibleSpan = 0.75
	delta := (ratio - 1) / visibleSpan
	delta = math.Max(-1, math.Min(1, delta))

	from, to, amount := heatNeutral, heatWarm, delta
	if delta < 0 {
		from, to, amount = heatNeutral, heatCool, -delta
	}
	return fmt.Sprintf(
		"#%02x%02x%02x",
		int(math.Round(from.r+(to.r-from.r)*amount)),
		int(math.Round(from.g+(to.g-from.g)*amount)),
		int(math.Round(from.b+(to.b-from.b)*amount)),
	)
}
