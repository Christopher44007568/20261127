package furnace

// EnergyCurve stores independently sampled raw and corrected E(mu) values.
type EnergyCurve struct {
	Mu        []float64
	Raw       []float64
	Corrected []float64
}

// BuildEnergyCurve samples mu from head-on toward the rim.
func BuildEnergyCurve(m Material, table *EnergyTable, nodes, samples int) EnergyCurve {
	if nodes < 8 {
		nodes = 8
	}
	if samples < 256 {
		samples = 256
	}
	if table == nil {
		table = DefaultEnergyTable(m)
	}
	wm := WhiteFurnaceMaterial(m)
	c := EnergyCurve{
		Mu:        make([]float64, nodes),
		Raw:       make([]float64, nodes),
		Corrected: make([]float64, nodes),
	}
	for i := 0; i < nodes; i++ {
		f := float64(i) / float64(nodes-1)
		mu := 1 - 0.985*f // 1.000 .. 0.015
		c.Mu[i] = mu
		c.Raw[i] = DirectionalAlbedo(wm, mu, samples).X
		c.Corrected[i] = CorrectedDirectionalAlbedo(wm, table, mu, samples).X
	}
	return c
}

func interpCurve(xs, ys []float64, x float64) float64 {
	if len(xs) == 0 {
		return 1
	}
	// Curves are stored descending in x (mu), so search in reverse.
	if x >= xs[0] {
		return ys[0]
	}
	last := len(xs) - 1
	if x <= xs[last] {
		return ys[last]
	}
	for i := 0; i < last; i++ {
		if x <= xs[i] && x >= xs[i+1] {
			span := xs[i] - xs[i+1]
			f := 0.0
			if span > 1e-12 {
				f = (xs[i] - x) / span
			}
			return lerp(ys[i], ys[i+1], f)
		}
	}
	return ys[last]
}

// RawAt interpolates raw energy at mu.
func (c EnergyCurve) RawAt(mu float64) float64 { return interpCurve(c.Mu, c.Raw, mu) }

// CorrectedAt interpolates measured corrected energy at mu.
func (c EnergyCurve) CorrectedAt(mu float64) float64 {
	return interpCurve(c.Mu, c.Corrected, mu)
}
