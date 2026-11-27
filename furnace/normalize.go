package furnace

import (
	"math"
	"sync"
)

// EnergyTable is a directional albedo LUT E(mu). Mu is N dot V.
type EnergyTable struct {
	Kind    Kind
	Bins    int
	Samples int
	Mu      []float64

	// Raw is the colored directional energy of the supplied material.
	Raw []Vec3

	// White is the scalar white-furnace directional energy of the lobe shape.
	White []float64
}

var defaultEnergyCache sync.Map

type energyCacheKey struct {
	kind           Kind
	albedo         Vec3
	sheenTint      Vec3
	roughness      float64
	metallic       float64
	coat           float64
	coatRoughness  float64
	coatF0         float64
	sheen          float64
	sheenRoughness float64
	retro          float64
	grazing        float64
	bins           int
	samples        int
}

// DefaultEnergyTable returns a cached high-resolution LUT for m.
func DefaultEnergyTable(m Material) *EnergyTable {
	key := energyCacheKey{
		kind: m.Kind, albedo: m.Albedo, sheenTint: m.SheenTint,
		roughness: roundCache(m.Roughness), metallic: roundCache(m.Metallic),
		coat: roundCache(m.Coat), coatRoughness: roundCache(m.CoatRoughness), coatF0: roundCache(m.CoatF0),
		sheen: roundCache(m.Sheen), sheenRoughness: roundCache(m.SheenRoughness),
		retro: roundCache(m.SheenRetro), grazing: roundCache(m.SheenGrazingGain),
		bins: 72, samples: 8192,
	}
	if cached, ok := defaultEnergyCache.Load(key); ok {
		return cached.(*EnergyTable)
	}
	t := BuildEnergyTable(m, key.bins, key.samples)
	defaultEnergyCache.Store(key, t)
	return t
}

func roundCache(x float64) float64 { return math.Round(x*10000) / 10000 }

// BuildEnergyTable numerically integrates RawBRDF over the hemisphere for a
// cosine-sparse but grazing-aware set of outgoing directions.
func BuildEnergyTable(m Material, bins, samples int) *EnergyTable {
	if bins < 12 {
		bins = 12
	}
	if samples < 64 {
		samples = 64
	}
	t := &EnergyTable{
		Kind:    m.Kind,
		Bins:    bins,
		Samples: samples,
		Mu:      make([]float64, bins),
		Raw:     make([]Vec3, bins),
		White:   make([]float64, bins),
	}
	white := WhiteFurnaceMaterial(m)
	for i := 0; i < bins; i++ {
		f := float64(i) / float64(bins-1)
		// First node is close to, but not exactly at, the measure-zero rim.
		mu := 0.002 + 0.998*f
		t.Mu[i] = mu
		t.Raw[i] = DirectionalAlbedo(m, mu, samples)
		t.White[i] = saturateScalar(DirectionalAlbedo(white, mu, samples).X, 8)
	}
	return t
}

func saturateScalar(x, hi float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) || x < 0 {
		return 0
	}
	if x > hi {
		return hi
	}
	return x
}

// WhiteEnergy interpolates the raw white-furnace energy E(mu).
func (t *EnergyTable) WhiteEnergy(nDotV float64) float64 {
	if t == nil || len(t.Mu) == 0 {
		return 1
	}
	mu := saturate(nDotV)
	if mu <= t.Mu[0] {
		return t.White[0]
	}
	last := len(t.Mu) - 1
	if mu >= t.Mu[last] {
		return t.White[last]
	}
	i := 0
	for i < last && t.Mu[i+1] < mu {
		i++
	}
	span := t.Mu[i+1] - t.Mu[i]
	f := 0.0
	if span > 1e-12 {
		f = (mu - t.Mu[i]) / span
	}
	return lerp(t.White[i], t.White[i+1], f)
}

// RawEnergy interpolates the colored directional albedo.
func (t *EnergyTable) RawEnergy(nDotV float64) Vec3 {
	if t == nil || len(t.Mu) == 0 {
		return V(0, 0, 0)
	}
	mu := saturate(nDotV)
	if mu <= t.Mu[0] {
		return t.Raw[0]
	}
	last := len(t.Mu) - 1
	if mu >= t.Mu[last] {
		return t.Raw[last]
	}
	i := 0
	for i < last && t.Mu[i+1] < mu {
		i++
	}
	span := t.Mu[i+1] - t.Mu[i]
	f := 0.0
	if span > 1e-12 {
		f = (mu - t.Mu[i]) / span
	}
	return t.Raw[i].Mul(1 - f).Add(t.Raw[i+1].Mul(f))
}

// Correction returns the per-view scale and Lambert deficit coefficient.
// For raw E>1 the raw lobe is divided by E. For E<1 a reciprocal-friendly
// multiple-scattering-style term supplies exactly 1-E in white furnace.
func (t *EnergyTable) Correction(nDotV float64) (scale, deficit float64) {
	e := t.WhiteEnergy(nDotV)
	const eps = 1e-6
	if e > 1+eps {
		return 1 / e, 0
	}
	if e < 1-eps {
		return 1, 1 - e
	}
	return 1, 0
}

// ScaleAt is the directional normalization multiplier.
func (t *EnergyTable) ScaleAt(nDotV float64) float64 {
	s, _ := t.Correction(nDotV)
	return s
}

// DeficitAt is the added multiple-scattering/diffuse energy.
func (t *EnergyTable) DeficitAt(nDotV float64) float64 {
	_, d := t.Correction(nDotV)
	return d
}

// NormalizeEnergyScalar maps a raw directional energy to its scale and deficit.
func NormalizeEnergyScalar(e float64) (scale, deficit float64) {
	if e > 1 {
		return 1 / e, 0
	}
	return 1, 1 - e
}

// NormalizeAlbedo keeps an albedo vector in the legal RGB cube. Values supplied
// to this program are already legal; this is useful for externally generated
// parameter sweeps.
func NormalizeAlbedo(a Vec3) Vec3 {
	m := a.MaxChannel()
	if m > 1 {
		a = a.Mul(1 / m)
	}
	return a.Clamp(0, 1)
}

// CompensationColor is the colored multiple-scattering tint. It becomes white
// in the white-furnace variant, making the missing energy exactly one.
func CompensationColor(m Material) Vec3 {
	return m.Albedo.Clamp(0, 1)
}

// NormalizedBRDF applies directional energy normalization to RawBRDF.
func NormalizedBRDF(m Material, table *EnergyTable, v, l Vec3) Vec3 {
	v = v.Normalize()
	l = l.Normalize()
	if table == nil {
		table = DefaultEnergyTable(m)
	}
	if v.Z <= 1e-7 || l.Z <= 1e-7 {
		return V(0, 0, 0)
	}
	scale, deficit := table.Correction(v.Z)
	raw := RawBRDF(m, v, l).Mul(scale)
	comp := CompensationColor(m).Mul(deficit / math.Pi)
	return raw.Add(comp)
}

// CorrectedDirectionalAlbedo integrates the analytically decomposed normalized
// response. The correction depends only on V, so its Lambert deficit integrates
// to deficit exactly.
func CorrectedDirectionalAlbedo(m Material, table *EnergyTable, nDotV float64, samples int) Vec3 {
	if table == nil {
		table = DefaultEnergyTable(m)
	}
	mu := clamp(nDotV, 1e-4, 1)
	raw := DirectionalAlbedo(m, mu, samples)
	scale, deficit := table.Correction(mu)
	return raw.Mul(scale).Add(CompensationColor(m).Mul(deficit))
}
