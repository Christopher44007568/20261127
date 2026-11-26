package main

import (
	"math"
	"sort"
)

type V3 struct {
	X, Y, Z float64
}

func (a V3) Add(b V3) V3      { return V3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a V3) Sub(b V3) V3      { return V3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a V3) Mul(s float64) V3 { return V3{a.X * s, a.Y * s, a.Z * s} }
func (a V3) Div(s float64) V3 { return V3{a.X / s, a.Y / s, a.Z / s} }

func Dot(a, b V3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func Cross(a, b V3) V3 {
	return V3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func Norm2(a V3) float64 { return Dot(a, a) }
func Norm(a V3) float64  { return math.Sqrt(Norm2(a)) }

func Normalize(a V3) V3 {
	n := Norm(a)
	if n < 1e-15 {
		return V3{}
	}
	return a.Mul(1 / n)
}

func AbsDot(a, b V3) float64 { return math.Abs(Dot(a, b)) }

func LerpV3(a, b V3, t float64) V3 {
	return V3{
		a.X + (b.X-a.X)*t,
		a.Y + (b.Y-a.Y)*t,
		a.Z + (b.Z-a.Z)*t,
	}
}

type Box struct {
	Min V3
	Max V3
}

func NewBox() Box {
	inf := math.Inf(1)
	ninf := math.Inf(-1)
	return Box{Min: V3{inf, inf, inf}, Max: V3{ninf, ninf, ninf}}
}

func (b *Box) Add(p V3) {
	b.Min.X = math.Min(b.Min.X, p.X)
	b.Min.Y = math.Min(b.Min.Y, p.Y)
	b.Min.Z = math.Min(b.Min.Z, p.Z)
	b.Max.X = math.Max(b.Max.X, p.X)
	b.Max.Y = math.Max(b.Max.Y, p.Y)
	b.Max.Z = math.Max(b.Max.Z, p.Z)
}

func (b Box) Size() V3 { return b.Max.Sub(b.Min) }
func (b Box) Center() V3 {
	return V3{
		(b.Min.X + b.Max.X) * 0.5,
		(b.Min.Y + b.Max.Y) * 0.5,
		(b.Min.Z + b.Max.Z) * 0.5,
	}
}

func (b Box) Expanded(d float64) Box {
	return Box{Min: b.Min.Sub(V3{d, d, d}), Max: b.Max.Add(V3{d, d, d})}
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func lerpFloat(a, b, t float64) float64 { return a + (b-a)*t }

func medianFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return 0.5 * (cp[n/2-1] + cp[n/2])
}

func quantileSorted(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if q <= 0 {
		return sorted[0]
	}
	if q >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := q * float64(len(sorted)-1)
	i := int(math.Floor(pos))
	j := i + 1
	if j >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	return lerpFloat(sorted[i], sorted[j], pos-float64(i))
}
